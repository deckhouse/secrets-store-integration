/*
Copyright 2024 Flant JSC

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
	"net/http"
	"os"
	"strings"
	"sync"

	"emperror.dev/errors"
	"github.com/deckhouse/deckhouse/pkg/log"
	vaultapi "github.com/hashicorp/vault/api"
	vaultk8s "github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/spf13/cast"
)

// Client wraps a HashiCorp Vault API client.
type Client struct {
	client       *vaultapi.Client
	tokenWatcher *vaultapi.Renewer
	logger       Logger
	mu           sync.Mutex
	closed       bool
	longRunning  bool
	// How the client was authenticated, needed for re-login:
	// a token file which can be re-read, or the Kubernetes login flow.
	tokenFile string
	usedLogin bool
}

// RawClient returns the underlying HashiCorp Vault API client.
func (c *Client) RawClient() *vaultapi.Client {
	return c.client
}

// Close stops background token renewal.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true

	if c.tokenWatcher != nil {
		c.tokenWatcher.Stop()
	}
}

func newVaultClient(logger Logger) (*Client, error) {
	cfg := vaultapi.DefaultConfig()
	if cfg.Error != nil {
		return nil, cfg.Error
	}

	raw, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	client := &Client{
		client: raw,
		logger: logger,
	}

	daemonMode := cast.ToBool(os.Getenv("VAULT_ENV_DAEMON"))
	restartMode := os.Getenv("VAULT_ENV_RESTART_ON_SECRET_CHANGE")
	// watch-for-lease implies daemon mode even when VAULT_ENV_DAEMON is not set
	// explicitly, see main().
	client.longRunning = daemonMode || restartMode == "watch-for-data" || restartMode == "watch-for-lease"

	token := os.Getenv("VAULT_TOKEN")
	isLogin := token == vaultLogin

	if tokenFile := os.Getenv("VAULT_TOKEN_FILE"); tokenFile != "" {
		b, err := os.ReadFile(tokenFile)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read vault token file: %s", tokenFile)
		}
		raw.SetToken(strings.TrimSpace(string(b)))
		client.tokenFile = tokenFile
	} else if token != "" && !isLogin {
		raw.SetToken(token)
	} else {
		if isLogin {
			_ = os.Unsetenv("VAULT_TOKEN")
		}

		loginSecret, err := kubernetesLogin(raw)
		if err != nil {
			return nil, err
		}
		client.usedLogin = true

		if client.longRunning && loginSecret != nil {
			if err := client.startTokenRenewal(loginSecret); err != nil {
				return nil, err
			}
		}
	}

	return client, nil
}

// Relogin re-authenticates the client after its token has expired or been
// revoked. Depending on how the client was configured it either re-reads the
// token file (Vault Agent rotates it) or repeats the Kubernetes login. Clients
// configured with a static VAULT_TOKEN cannot be re-authenticated.
func (c *Client) Relogin() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("vault client is closed")
	}

	switch {
	case c.tokenFile != "":
		b, err := os.ReadFile(c.tokenFile)
		if err != nil {
			return errors.Wrapf(err, "could not read vault token file: %s", c.tokenFile)
		}
		c.client.SetToken(strings.TrimSpace(string(b)))
	case c.usedLogin:
		// The old watcher renews a token which is no longer valid, stop it
		// before acquiring a new one.
		if c.tokenWatcher != nil {
			c.tokenWatcher.Stop()
			c.tokenWatcher = nil
		}

		loginSecret, err := kubernetesLogin(c.client)
		if err != nil {
			return errors.Wrap(err, "failed to re-login to vault")
		}

		if c.longRunning && loginSecret != nil {
			if err := c.startTokenRenewal(loginSecret); err != nil {
				return err
			}
		}
	default:
		return errors.New("cannot re-login: the client was configured with a static VAULT_TOKEN")
	}

	return nil
}

// isPermissionDeniedError reports whether the error is a Vault "permission denied"
// response, which is what requests with an expired or revoked token produce.
func isPermissionDeniedError(err error) bool {
	var respErr *vaultapi.ResponseError

	return errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden
}

func kubernetesLogin(raw *vaultapi.Client) (*vaultapi.Secret, error) {
	role := os.Getenv("VAULT_ROLE")
	if role == "" {
		role = "default"
	}

	var opts []vaultk8s.LoginOption
	if authPath := os.Getenv("VAULT_PATH"); authPath != "" {
		opts = append(opts, vaultk8s.WithMountPath(authPath))
	}
	if jwtPath := os.Getenv("VAULT_JWT_FILE"); jwtPath != "" {
		opts = append(opts, vaultk8s.WithServiceAccountTokenPath(jwtPath))
	}

	auth, err := vaultk8s.NewKubernetesAuth(role, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to configure kubernetes auth")
	}

	secret, err := raw.Auth().Login(context.Background(), auth)
	if err != nil {
		return nil, errors.Wrap(err, "failed to login to vault")
	}

	return secret, nil
}

func (c *Client) startTokenRenewal(secret *vaultapi.Secret) error {
	watcher, err := c.client.NewLifetimeWatcher(&vaultapi.LifetimeWatcherInput{Secret: secret})
	if err != nil {
		return errors.Wrap(err, "failed to create token watcher")
	}

	c.tokenWatcher = watcher

	go watcher.Start()
	go c.runTokenRenewChecker(watcher)

	return nil
}

func (c *Client) runTokenRenewChecker(watcher *vaultapi.Renewer) {
	for {
		select {
		case err := <-watcher.DoneCh():
			if err != nil {
				c.logger.Error("error in vault token renewal", log.Err(err))
			}

			return
		case output := <-watcher.RenewCh():
			if output == nil {
				continue
			}

			ttl, _ := output.Secret.TokenTTL()
			c.logger.Info("renewed vault token", "ttl", ttl)
		}
	}
}
