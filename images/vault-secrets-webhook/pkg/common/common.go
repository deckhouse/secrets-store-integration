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
	MutateProbesAnnotation                = "secrets-store.deckhouse.io/mutate-probes"

	EnableJSONLogAnnotation = "secrets-store.deckhouse.io/enable-json-log"
	// Vault annotations
	VaultAddrAnnotation                     = "secrets-store.deckhouse.io/addr"
	VaultRoleAnnotation                     = "secrets-store.deckhouse.io/role"
	VaultPathAnnotation                     = "secrets-store.deckhouse.io/auth-path"
	VaultSkipVerifyAnnotation               = "secrets-store.deckhouse.io/tls-skip-verify"
	VaultTLSSecretAnnotation                = "secrets-store.deckhouse.io/tls-secret"
	VaultIgnoreMissingSecretsAnnotation     = "secrets-store.deckhouse.io/ignore-missing-secrets"
	VaultClientTimeoutAnnotation            = "secrets-store.deckhouse.io/client-timeout"
	VaultNamespaceAnnotation                = "secrets-store.deckhouse.io/namespace"
	ServiceAccountTokenVolumeNameAnnotation = "secrets-store.deckhouse.io/service-account-token-volume-name"
	LogLevelAnnotation                      = "secrets-store.deckhouse.io/log-level"
	VaultEnvFromPathAnnotation              = "secrets-store.deckhouse.io/env-from-path"
	SkipMutateContainersAnnotation          = "secrets-store.deckhouse.io/skip-mutate-containers"
)

func HasVaultPrefix(value string) bool {
	return strings.HasPrefix(value, "secrets-store:") || strings.HasPrefix(value, ">>secrets-store:")
}
