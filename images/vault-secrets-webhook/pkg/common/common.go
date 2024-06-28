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

package common

import (
	"strings"
)

const (
	// Webhook annotations
	// ref: https://bank-vaults.dev/docs/mutating-webhook/annotations/
	MutateProbesAnnotation                = "secret-store.deckhouse.io/mutate-probes"

	EnableJSONLogAnnotation = "secret-store.deckhouse.io/enable-json-log"
	// Vault annotations
	VaultAddrAnnotation                     = "secret-store.deckhouse.io/addr"
	VaultRoleAnnotation                     = "secret-store.deckhouse.io/role"
	VaultPathAnnotation                     = "secret-store.deckhouse.io/auth-path"
	VaultSkipVerifyAnnotation               = "secret-store.deckhouse.io/tls-skip-verify"
	VaultTLSSecretAnnotation                = "secret-store.deckhouse.io/tls-secret"
	VaultIgnoreMissingSecretsAnnotation     = "secret-store.deckhouse.io/ignore-missing-secrets"
	VaultClientTimeoutAnnotation            = "secret-store.deckhouse.io/client-timeout"
	VaultNamespaceAnnotation                = "secret-store.deckhouse.io/namespace"
	ServiceAccountTokenVolumeNameAnnotation = "secret-store.deckhouse.io/service-account-token-volume-name"
	LogLevelAnnotation                      = "secret-store.deckhouse.io/log-level"
	VaultEnvFromPathAnnotation = "secret-store.deckhouse.io/env-from-path"

)

func HasVaultPrefix(value string) bool {
	return strings.HasPrefix(value, "secret-store:") || strings.HasPrefix(value, ">>secret-store:")
}
