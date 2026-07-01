/*
Copyright 2026 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"emperror.dev/errors"
	"github.com/deckhouse/deckhouse/pkg/log"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/spf13/cast"
)

// The special value for VAULT_ENV which marks that the login token needs to be passed through to the application
// which was acquired during the new Vault client creation
const vaultLogin = "vault:login"

type sanitizedEnviron struct {
	env   []string
	login bool
}

type envType struct {
	login bool
}

var sanitizeEnvmap = map[string]envType{
	"VAULT_TOKEN":                        {login: true},
	"VAULT_ADDR":                         {login: true},
	"VAULT_AGENT_ADDR":                   {login: true},
	"VAULT_CACERT":                       {login: true},
	"VAULT_CAPATH":                       {login: true},
	"VAULT_CLIENT_CERT":                  {login: true},
	"VAULT_CLIENT_KEY":                   {login: true},
	"VAULT_CLIENT_TIMEOUT":               {login: true},
	"VAULT_SRV_LOOKUP":                   {login: true},
	"VAULT_SKIP_VERIFY":                  {login: true},
	"VAULT_NAMESPACE":                    {login: true},
	"VAULT_TLS_SERVER_NAME":              {login: true},
	"VAULT_WRAP_TTL":                     {login: true},
	"VAULT_MFA":                          {login: true},
	"VAULT_MAX_RETRIES":                  {login: true},
	"VAULT_CLUSTER_ADDR":                 {login: false},
	"VAULT_REDIRECT_ADDR":                {login: false},
	"VAULT_CLI_NO_COLOR":                 {login: false},
	"VAULT_RATE_LIMIT":                   {login: false},
	"VAULT_ROLE":                         {login: false},
	"VAULT_PATH":                         {login: false},
	"VAULT_AUTH_METHOD":                  {login: false},
	"VAULT_JWT_FILE":                     {login: false},
	"VAULT_IGNORE_MISSING_SECRETS":       {login: false},
	"VAULT_ENV_PASSTHROUGH":              {login: false},
	"VAULT_JSON_LOG":                     {login: false},
	"VAULT_LOG_LEVEL":                    {login: false},
	"VAULT_REVOKE_TOKEN":                 {login: false},
	"VAULT_ENV_DAEMON":                   {login: false},
	"VAULT_ENV_FROM_PATH":                {login: false},
	"VAULT_ENV_DELAY":                    {login: false},
	"VAULT_ENV_RESTART_ON_SECRET_CHANGE": {login: false},
	"VAULT_ENV_SECRET_POLL_INTERVAL":     {login: false},
}

// Appends variable an entry (name=value) into the environ list.
// VAULT_* variables are not populated into this list if this is not a login scenario.
func (e *sanitizedEnviron) append(name string, value string) {
	if envType, ok := sanitizeEnvmap[name]; !ok || (e.login && envType.login) {
		e.env = append(e.env, fmt.Sprintf("%s=%s", name, value))
	}
}

type daemonSecretRenewer struct {
	client *Client
	sigs   chan os.Signal
	logger Logger
}

func (r daemonSecretRenewer) Renew(path string, secret *vaultapi.Secret) error {
	watcherInput := vaultapi.LifetimeWatcherInput{Secret: secret}
	watcher, err := r.client.RawClient().NewLifetimeWatcher(&watcherInput)
	if err != nil {
		return errors.Wrap(err, "failed to create secret watcher")
	}

	go watcher.Start()

	go func() {
		defer watcher.Stop()
		for {
			select {
			case renewOutput := <-watcher.RenewCh():
				r.logger.Info("secret renewed", "path", path, "lease-duration", time.Duration(renewOutput.Secret.LeaseDuration)*time.Second)
			case doneError := <-watcher.DoneCh():
				if !secret.Renewable {
					leaseDuration := time.Duration(secret.LeaseDuration) * time.Second
					time.Sleep(leaseDuration)

					r.logger.Info("secret lease has expired", "path", path, "lease-duration", leaseDuration)
				}

				r.logger.Info("secret renewal has stopped, sending SIGTERM to process", "path", path, "done-error", doneError)

				r.sigs <- syscall.SIGTERM

				timeout := <-time.After(10 * time.Second)
				r.logger.Info("killing process due to SIGTERM timeout", "timeout", timeout)
				r.sigs <- syscall.SIGKILL

				return
			}
		}
	}()

	return nil
}

// kubernetesSignals are delivered to env-injector (PID 1) by the container
// runtime / kubelet and must be forwarded to the application process.
// Internal SIGTERM/SIGKILL requests (secret change, lease expiry) are sent
// directly to the sigs channel and are not registered via signal.Notify.
var kubernetesSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGHUP,
	syscall.SIGQUIT,
}

// jitterFraction is the maximum relative skew applied to the poll interval,
// i.e. the effective interval is uniformly distributed in [0.7*i, 1.3*i].
const jitterFraction = 0.3

// jitteredInterval randomizes the poll interval by ±jitterFraction. Each pod
// draws its own value at startup, so replicas poll with different periods,
// their phases drift apart, and a secret change doesn't restart all of them
// at the same moment.
func jitteredInterval(interval time.Duration) time.Duration {
	maxSkew := time.Duration(float64(interval) * jitterFraction)
	if maxSkew <= 0 {
		return interval
	}

	return interval - maxSkew + rand.N(2*maxSkew)
}

func watchSecretsForChanges(
	ctx context.Context,
	client *Client,
	config Config,
	templateEnviron map[string]string,
	envFromPath string,
	initialHash string,
	pollInterval time.Duration,
	sigs chan os.Signal,
	restartDueToSecretChange *atomic.Bool,
	logger Logger,
) {
	// The poll injector has no renewer, so it must never try to start
	// lease renewal (that would dereference a nil renewer in daemon mode).
	pollConfig := config
	pollConfig.DaemonMode = false

	effectiveInterval := jitteredInterval(pollInterval)
	logger.Info("watching secrets for changes", "effective-poll-interval", effectiveInterval)

	ticker := time.NewTicker(effectiveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			secretsSet := make(map[string]string)
			inject := func(key, value string) {
				secretsSet[key] = value
			}

			// Re-resolve from a fresh injector and a fresh copy of the templates each
			// poll: the injector caches resolved paths, and we must always start from
			// templates (secrets-store:...), not from previously resolved Vault values.
			references := make(map[string]string, len(templateEnviron))
			for name, value := range templateEnviron {
				references[name] = value
			}

			pollInjector := NewSecretInjector(pollConfig, client, nil, logger)
			err := pollInjector.InjectSecretsFromVault(references, inject)
			if err == nil && envFromPath != "" {
				err = pollInjector.InjectSecretsFromVaultPath(envFromPath, inject)
			}
			if err != nil {
				logger.Warn("failed to poll secrets from vault", log.Err(err))

				// "permission denied" means the token has expired or been revoked
				// (e.g. renewal reached the token max TTL); try to re-authenticate
				// so that the next poll can succeed.
				if isPermissionDeniedError(err) {
					if reloginErr := client.Relogin(); reloginErr != nil {
						logger.Warn("failed to re-login to vault", log.Err(reloginErr))
					} else {
						logger.Info("re-logged in to vault after permission denied")
					}
				}

				continue
			}

			newHash := computeSecretsHash(secretsSet)
			// Drop the resolved secret values as soon as the hash is computed.
			clear(secretsSet)
			clear(references)

			if newHash == initialHash {
				continue
			}

			logger.Info("secret values changed, restarting process",
				"old-hash", initialHash,
				"new-hash", newHash,
			)

			restartDueToSecretChange.Store(true)

			select {
			case sigs <- syscall.SIGTERM:
			case <-ctx.Done():
				return
			}

			select {
			case <-time.After(10 * time.Second):
			case <-ctx.Done():
				return
			}

			logger.Info("killing process due to SIGTERM timeout after secret change")

			select {
			case sigs <- syscall.SIGKILL:
			case <-ctx.Done():
			}

			return
		}
	}
}

func main() {
	logger := newLogger()

	if len(os.Args) == 1 {
		logger.Error("no command is given, env-injector can't determine the entrypoint (command), please specify it explicitly or let the webhook query it (see documentation)")

		os.Exit(1)
	}

	if len(os.Args) == 2 && os.Args[1] == "--dummy-run" { //check binary can run on node
		os.Exit(0)
	}

	if len(os.Args) == 2 && os.Args[1] == "--self-copy" {
		source, err := os.Open("/bin/env-injector") //open the source file
		if err != nil {
			logger.Error("failed to open env-injector source", log.Err(err))
			os.Exit(1)
		}
		defer source.Close()

		destination, err := os.Create("/vault/env-injector") //create the destination file
		if err != nil {
			logger.Error("failed to create env-injector destination", log.Err(err))
			os.Exit(1)
		}
		defer destination.Close()
		_, err = io.Copy(destination, source) //copy the contents of source to destination file
		if err != nil {
			logger.Error("failed to copy env-injector binary", log.Err(err))
			os.Exit(1)
		}
		err = os.Chmod("/vault/env-injector", 0555)
		if err != nil {
			logger.Error("failed to chmod env-injector destination", log.Err(err))
			os.Exit(1)
		}

		os.Exit(0)
	}

	daemonMode := cast.ToBool(os.Getenv("VAULT_ENV_DAEMON"))
	restartMode := os.Getenv("VAULT_ENV_RESTART_ON_SECRET_CHANGE")
	if restartMode == "watch-for-lease" {
		daemonMode = true
	}
	watchData := restartMode == "watch-for-data"
	delayExec := cast.ToDuration(os.Getenv("VAULT_ENV_DELAY"))
	secretPollInterval := cast.ToDuration(os.Getenv("VAULT_ENV_SECRET_POLL_INTERVAL"))
	if watchData && secretPollInterval <= 0 {
		secretPollInterval = 120 * time.Second
	}
	sigs := make(chan os.Signal, 1)

	entrypointCmd := os.Args[1:]

	binary, err := exec.LookPath(entrypointCmd[0])
	if err != nil {
		logger.Error("binary not found", "binary", entrypointCmd[0])

		os.Exit(1)
	}

	ignoreMissingSecrets := cast.ToBool(os.Getenv("VAULT_IGNORE_MISSING_SECRETS"))

	// The login procedure takes the token from a file (if using Vault Agent)
	// or requests one for itself via Kubernetes/JWT auth,
	// so if we got a VAULT_TOKEN for the special value with "vault:login"
	isLogin := os.Getenv("VAULT_TOKEN") == vaultLogin

	client, err := newVaultClient(logger)
	if err != nil {
		logger.Error("failed to create vault client", log.Err(err))

		os.Exit(1)
	}

	passthroughEnvVars := strings.Split(os.Getenv("VAULT_ENV_PASSTHROUGH"), ",")

	if isLogin {
		_ = os.Setenv("VAULT_TOKEN", vaultLogin)
		passthroughEnvVars = append(passthroughEnvVars, "VAULT_TOKEN")
	}

	// do not sanitize env vars specified in VAULT_ENV_PASSTHROUGH
	for _, envVar := range passthroughEnvVars {
		if trimmed := strings.TrimSpace(envVar); trimmed != "" {
			delete(sanitizeEnvmap, trimmed)
		}
	}

	// initial and sanitized environs
	environ := make(map[string]string, len(os.Environ()))
	sanitized := sanitizedEnviron{login: isLogin}

	config := Config{
		DaemonMode:           daemonMode,
		IgnoreMissingSecrets: ignoreMissingSecrets,
	}

	var secretRenewer SecretRenewer

	if daemonMode {
		secretRenewer = daemonSecretRenewer{client: client, sigs: sigs, logger: logger}
	}

	secretInjector := NewSecretInjector(config, client, secretRenewer, logger)

	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)
		name := split[0]
		value := split[1]
		environ[name] = value
	}

	// Snapshot the templated environment (e.g. MY_APP=secrets-store:path#key) before
	// injection mutates it. Periodic polling must re-resolve from these templates,
	// never from the values already fetched from Vault.
	var templateEnviron map[string]string
	if watchData {
		templateEnviron = make(map[string]string, len(environ))
		for name, value := range environ {
			templateEnviron[name] = value
		}
	}

	// collect all secrets into a map, to avoid duplicate entries, then append to an environ
	secretsSet := make(map[string]string)
	inject := func(key, value string) {
		secretsSet[key] = value
	}

	err = secretInjector.InjectSecretsFromVault(environ, inject)
	if err != nil {
		logger.Error("failed to inject secrets from vault", log.Err(err))

		os.Exit(1)
	}

	if paths := os.Getenv("VAULT_ENV_FROM_PATH"); paths != "" {
		err = secretInjector.InjectSecretsFromVaultPath(paths, inject)
	}
	if err != nil {
		logger.Error("failed to inject secrets from vault path", log.Err(err))

		os.Exit(1)
	}

	for key, value := range secretsSet {
		sanitized.append(key, value)
	}

	envFromPath := os.Getenv("VAULT_ENV_FROM_PATH")

	// Prepare the change watching baseline. Update-style references
	// (">>secrets-store:...") perform a Vault write on every resolution, so they
	// can't be polled: they are excluded both from the watched templates and from
	// the baseline hash so that both sides of the comparison cover the same set.
	var initialSecretsHash string
	var watchedTemplates map[string]string
	if watchData {
		watchedTemplates = make(map[string]string, len(templateEnviron))
		baseline := make(map[string]string, len(secretsSet))
		for key, value := range secretsSet {
			baseline[key] = value
		}

		for name, value := range templateEnviron {
			if isUpdateReference(value) {
				logger.Info("excluding update-style secret reference from change watching", "name", name)
				delete(baseline, name)

				continue
			}
			watchedTemplates[name] = value
		}

		initialSecretsHash = computeSecretsHash(baseline)
		clear(baseline)

		hasWatchedSecrets := envFromPath != ""
		for _, value := range watchedTemplates {
			if IsValidPrefix(value) || HasInlineVaultDelimiters(value) {
				hasWatchedSecrets = true

				break
			}
		}
		if !hasWatchedSecrets {
			logger.Info("no pollable secret references found, secret change watching is disabled")
			watchData = false
		}
	}

	// Drop resolved secret values which are not needed anymore, the child
	// environment has already been built.
	clear(secretsSet)

	if cast.ToBool(os.Getenv("VAULT_REVOKE_TOKEN")) {
		if daemonMode || watchData {
			// The client must stay usable to renew leases or poll secret values.
			logger.Warn("VAULT_REVOKE_TOKEN is ignored, the vault client must stay active in watch mode")
		} else {
			// ref: https://www.vaultproject.io/api/auth/token/index.html#revoke-a-token-self-
			err = client.RawClient().Auth().Token().RevokeSelf(client.RawClient().Token())
			if err != nil {
				// Do not exit on error, token revoking can be denied by policy
				logger.Warn("failed to revoke token")
			}

			client.Close()
		}
	}

	if delayExec > 0 {
		logger.Info("sleeping before process start", "delay", delayExec)
		time.Sleep(delayExec)
	}

	logger.Info("spawning process", "entrypoint", fmt.Sprint(entrypointCmd))

	if daemonMode || watchData {
		if daemonMode {
			logger.Info("running in watch-for-lease mode")
		}
		if watchData {
			logger.Info("running in watch-for-data mode", "poll-interval", secretPollInterval)
		}

		cmd := exec.Command(binary, entrypointCmd[1:]...)
		// Pass only the sanitized environment, same as the exec path below:
		// the original environment contains secret templates and VAULT_* variables
		// which must not leak into the child process.
		cmd.Env = sanitized.env
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		signal.Notify(sigs, kubernetesSignals...)

		err = cmd.Start()
		if err != nil {
			logger.Error("failed to start process", log.Err(err), "entrypoint", fmt.Sprint(entrypointCmd))

			os.Exit(1)
		}

		watchCtx, watchCancel := context.WithCancel(context.Background())
		defer watchCancel()

		var restartDueToSecretChange atomic.Bool

		if watchData {
			go watchSecretsForChanges(
				watchCtx,
				client,
				config,
				watchedTemplates,
				envFromPath,
				initialSecretsHash,
				secretPollInterval,
				sigs,
				&restartDueToSecretChange,
				logger,
			)
		}

		go func() {
			for sig := range sigs {
				if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
					break
				}

				err := cmd.Process.Signal(sig)
				if err != nil {
					if errors.Is(err, os.ErrProcessDone) {
						break
					}

					logger.Warn("failed to signal process", log.Err(err), "signal", sig.String())

					continue
				}

				logger.Info("forwarded signal to process", "signal", sig.String())
			}
		}()

		err = cmd.Wait()

		watchCancel()

		signal.Stop(sigs)
		// The sigs channel is deliberately not closed: the watch and renewal
		// goroutines may still attempt to send to it, and a send to a closed
		// channel would panic. The process exits shortly anyway.

		if restartDueToSecretChange.Load() {
			logger.Info("exiting to allow container restart after secret change")
			_ = os.WriteFile("/dev/termination-log", []byte("secret values changed, restarting container"), 0o666)
			os.Exit(2)
		}

		if err != nil {
			exitCode := -1
			// try to get the original exit code if possible
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				exitCode = exitError.ExitCode()
			}

			logger.Error("failed to exec process", log.Err(err), "entrypoint", fmt.Sprint(entrypointCmd))

			os.Exit(exitCode)
		}

		os.Exit(cmd.ProcessState.ExitCode())
	} else {
		err = syscall.Exec(binary, entrypointCmd, sanitized.env)
		if err != nil {
			logger.Error("failed to exec process", log.Err(err), "entrypoint", fmt.Sprint(entrypointCmd))

			os.Exit(1)
		}
	}
}
