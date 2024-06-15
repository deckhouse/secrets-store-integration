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
	PSPAllowPrivilegeEscalationAnnotation = "stronghold.deckhouse.io/psp-allow-privilege-escalation"
	RunAsNonRootAnnotation                = "stronghold.deckhouse.io/run-as-non-root"
	RunAsUserAnnotation                   = "stronghold.deckhouse.io/run-as-user"
	RunAsGroupAnnotation                  = "stronghold.deckhouse.io/run-as-group"
	ReadOnlyRootFsAnnotation              = "stronghold.deckhouse.io/readonly-root-fs"
	RegistrySkipVerifyAnnotation          = "stronghold.deckhouse.io/registry-skip-verify"
	MutateAnnotation                      = "stronghold.deckhouse.io/mutate"
	MutateProbesAnnotation                = "stronghold.deckhouse.io/mutate-probes"

	EnableJSONLogAnnotation = "stronghold.deckhouse.io/enable-json-log"
	// SecretInitJSONLogAnnotation = "stronghold.deckhouse.io/secret-init-json-log"
	VaultEnvImageAnnotation = "stronghold.deckhouse.io/env-injector-image"
	// SecretInitImageAnnotation = "stronghold.deckhouse.io/secret-init-image"
	VaultEnvImagePullPolicyAnnotation = "stronghold.deckhouse.io/env-injector-image-pull-policy"
	// SecretInitImagePullPolicyAnnotation = "stronghold.deckhouse.io/secret-init-image-pull-policy"

	// Vault annotations
	VaultAddrAnnotation                     = "stronghold.deckhouse.io/addr"
	VaultRoleAnnotation                     = "stronghold.deckhouse.io/role"
	VaultPathAnnotation                     = "stronghold.deckhouse.io/auth-path"
	VaultSkipVerifyAnnotation               = "stronghold.deckhouse.io/tls-skip-verify"
	VaultTLSSecretAnnotation                = "stronghold.deckhouse.io/tls-secret"
	VaultIgnoreMissingSecretsAnnotation     = "stronghold.deckhouse.io/ignore-missing-secrets"
	VaultClientTimeoutAnnotation            = "stronghold.deckhouse.io/client-timeout"
	TransitKeyIDAnnotation                  = "stronghold.deckhouse.io/transit-key-id"
	TransitPathAnnotation                   = "stronghold.deckhouse.io/transit-path"
	VaultAuthMethodAnnotation               = "stronghold.deckhouse.io/auth-method"
	TransitBatchSizeAnnotation              = "stronghold.deckhouse.io/transit-batch-size"
	VaultServiceaccountAnnotation           = "stronghold.deckhouse.io/serviceaccount"
	VaultNamespaceAnnotation                = "stronghold.deckhouse.io/namespace"
	ServiceAccountTokenVolumeNameAnnotation = "stronghold.deckhouse.io/service-account-token-volume-name"
	LogLevelAnnotation                      = "stronghold.deckhouse.io/log-level"
	VaultEnvPassthroughAnnotation = "stronghold.deckhouse.io/vault-env-passthrough"
	VaultEnvFromPathAnnotation = "stronghold.deckhouse.io/env-from-path"

)

func HasVaultPrefix(value string) bool {
	return strings.HasPrefix(value, "stronghold:") || strings.HasPrefix(value, ">>stronghold:")
}
