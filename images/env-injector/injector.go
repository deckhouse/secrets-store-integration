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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"emperror.dev/errors"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/spf13/cast"
)

type SecretInjectorFunc func(key, value string)

type SecretRenewer interface {
	Renew(path string, secret *vaultapi.Secret) error
}

type Config struct {
	IgnoreMissingSecrets bool
	DaemonMode           bool
}

type SecretInjector struct {
	mu          sync.RWMutex
	config      Config
	client      *Client
	renewer     SecretRenewer
	logger      Logger
	secretCache map[string]map[string]interface{}
}

func NewSecretInjector(config Config, client *Client, renewer SecretRenewer, logger Logger) SecretInjector {
	return SecretInjector{
		config:      config,
		client:      client,
		renewer:     renewer,
		logger:      logger,
		secretCache: map[string]map[string]interface{}{},
	}
}

var inlineMutationRegex = regexp.MustCompile(`\${([>]{0,2}secrets-store:.*?#*}?)}`)

func (i *SecretInjector) InjectSecretsFromVault(references map[string]string, inject SecretInjectorFunc) error {
	return i.InjectSecretsFromVaultWithContext(context.Background(), references, inject)
}

func (i *SecretInjector) InjectSecretsFromVaultWithContext(ctx context.Context, references map[string]string, inject SecretInjectorFunc) error {
	for name, value := range references {
		if HasInlineVaultDelimiters(value) {
			for _, vaultSecretReference := range FindInlineVaultDelimiters(value) {
				mapData, err := i.GetDataFromVaultWithContext(ctx, map[string]string{name: vaultSecretReference[1]})
				if err != nil {
					return err
				}
				for _, v := range mapData {
					value = strings.ReplaceAll(value, vaultSecretReference[0], v)
				}
			}
			inject(name, value)

			continue
		}

		var update bool
		if strings.HasPrefix(value, ">>secrets-store:") {
			value = strings.TrimPrefix(value, ">>")
			update = true
		} else {
			update = false
		}

		if !strings.HasPrefix(value, "secrets-store:") {
			inject(name, value)

			continue
		}

		valuePath := strings.TrimPrefix(value, "secrets-store:")

		// handle special case for vault:login env value
		// namely pass through the VAULT_TOKEN received from the Vault login procedure
		if name == "VAULT_TOKEN" && valuePath == "login" {
			value = i.client.RawClient().Token()
			inject(name, value)

			continue
		}

		split := strings.SplitN(valuePath, "#", 3)
		valuePath = split[0]

		if len(split) < 2 {
			return errors.New("secret data key or template not defined")
		}

		key := split[1]

		versionOrData := "-1"
		if update {
			versionOrData = "{}"
		}
		if len(split) == 3 {
			versionOrData = split[2]
		}

		secretCacheKey := valuePath + "#" + versionOrData
		var data map[string]interface{}
		var err error

		i.mu.RLock()
		if data = i.secretCache[secretCacheKey]; data == nil {
			data, err = i.readVaultPath(ctx, valuePath, versionOrData, update)
		}
		i.mu.RUnlock()

		if err != nil {
			return err
		}

		if data == nil {
			if !i.config.IgnoreMissingSecrets {
				return errors.Errorf("path not found: %s", valuePath)
			}
			i.logger.Warn(fmt.Sprintf("path not found %s", valuePath))

			continue
		}

		i.mu.Lock()
		i.secretCache[secretCacheKey] = data
		i.mu.Unlock()

		tmpl := NewTemplater(DefaultLeftDelimiter, DefaultRightDelimiter)

		if tmpl.IsGoTemplate(key) {
			value, err := tmpl.Template(key, data)
			if err != nil {
				return errors.Wrapf(err, "failed to interpolate template key with vault data: %s", key)
			}
			inject(name, value.String())
		} else if value, ok := data[key]; ok {
			value, err := cast.ToStringE(value)
			if err != nil {
				return errors.Wrap(err, "value can't be cast to a string")
			}
			inject(name, value)
		} else {
			return errors.Errorf("key '%s' not found under path: %s", key, valuePath)
		}
	}

	return nil
}

func (i *SecretInjector) InjectSecretsFromVaultPath(paths string, inject SecretInjectorFunc) error {
	return i.InjectSecretsFromVaultPathWithContext(context.Background(), paths, inject)
}

func (i *SecretInjector) InjectSecretsFromVaultPathWithContext(ctx context.Context, paths string, inject SecretInjectorFunc) error {
	vaultPaths := strings.Split(paths, ",")

	for _, path := range vaultPaths {
		split := strings.SplitN(path, "#", 2)
		valuePath := split[0]

		version := "-1"

		if len(split) == 2 {
			version = split[1]
		}

		data, err := i.readVaultPath(ctx, valuePath, version, false)
		if err != nil {
			return err
		}

		if data == nil {
			if !i.config.IgnoreMissingSecrets {
				return errors.Errorf("path not found: %s", valuePath)
			}
			i.logger.Warn(fmt.Sprintf("path not found %s", valuePath))

			continue
		}

		for key, value := range data {
			value, err := cast.ToStringE(value)
			if err != nil {
				return errors.Wrap(err, "value can't be cast to a string for key: "+key)
			}
			inject(key, value)
		}
	}

	return nil
}

func (i *SecretInjector) readVaultPath(ctx context.Context, path, versionOrData string, update bool) (map[string]interface{}, error) {
	var secretData map[string]interface{}

	var secret *vaultapi.Secret
	var err error

	if update {
		var data map[string]interface{}
		err = json.Unmarshal([]byte(versionOrData), &data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal data for writing")
		}

		secret, err = i.client.RawClient().Logical().WriteWithContext(ctx, path, data)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write secret to path: %s", path)
		}
	} else {
		secret, err = i.client.RawClient().Logical().ReadWithDataWithContext(ctx, path, map[string][]string{"version": {versionOrData}})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read secret from path: %s", path)
		}
	}

	if i.config.DaemonMode && secret != nil && secret.LeaseDuration > 0 {
		i.logger.Info("secret has a lease duration, starting renewal", "path", path, "lease-duration", secret.LeaseDuration)

		err = i.renewer.Renew(path, secret)
		if err != nil {
			return nil, errors.Wrap(err, "secret renewal can't be established")
		}
	}

	if secret == nil {
		return nil, nil
	}

	for _, warning := range secret.Warnings {
		i.logger.Warn(warning, "path", path)
	}

	v2Data, ok := secret.Data["data"]
	if ok {
		secretData = cast.ToStringMap(v2Data)

		// Handle the case where "metadata" key is not present or is nil.
		metadataRaw, ok := secret.Data["metadata"]
		if metadataRaw == nil || !ok {
			return nil, errors.New("metadata key not found or is nil in secret")
		}

		// Handle the case where the type assertion fails.
		metadata, ok := metadataRaw.(map[string]interface{})
		if !ok {
			return nil, errors.New("metadata has an unexpected type")
		}

		// Check if a given version of a path is destroyed
		// Handle the case where "destroyed" key is not present or has an unexpected type.
		destroyed, _ := metadata["destroyed"].(bool)
		if destroyed {
			i.logger.Warn("version of secret has been permanently destroyed", "path", path, "version", versionOrData)
		}

		// Check if a given version of a path still exists
		if deletionTime, ok := metadata["deletion_time"].(string); ok && deletionTime != "" {
			i.logger.Warn(
				"cannot find data for path, given version has been deleted",
				"path", path,
				"version", versionOrData,
				"deletion-time", deletionTime,
			)
		}
	} else {
		secretData = cast.ToStringMap(secret.Data)
	}

	return secretData, nil
}

func IsValidPrefix(value string) bool {
	return strings.HasPrefix(value, "secrets-store:") || strings.HasPrefix(value, ">>secrets-store:")
}

// isUpdateReference reports whether resolving the value performs a Vault write
// (">>secrets-store:..." either as a plain value or inside inline delimiters).
// Such references must not be re-resolved by the secret change watcher.
func isUpdateReference(value string) bool {
	if strings.HasPrefix(value, ">>secrets-store:") {
		return true
	}

	for _, match := range FindInlineVaultDelimiters(value) {
		if strings.HasPrefix(match[1], ">>") {
			return true
		}
	}

	return false
}

func HasInlineVaultDelimiters(value string) bool {
	return len(FindInlineVaultDelimiters(value)) > 0
}

func FindInlineVaultDelimiters(value string) [][]string {
	return inlineMutationRegex.FindAllStringSubmatch(value, -1)
}

func (i *SecretInjector) GetDataFromVault(data map[string]string) (map[string]string, error) {
	return i.GetDataFromVaultWithContext(context.Background(), data)
}

func (i *SecretInjector) GetDataFromVaultWithContext(ctx context.Context, data map[string]string) (map[string]string, error) {
	vaultData := make(map[string]string, len(data))

	inject := func(key, value string) {
		vaultData[key] = value
	}

	return vaultData, i.InjectSecretsFromVaultWithContext(ctx, data, inject)
}
