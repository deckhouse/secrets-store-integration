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

package webhook

import (
	"strconv"
	"time"

	"github.com/slok/kubewebhook/v2/pkg/model"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"vault-secrets-webhook/pkg/common"
)

// VaultConfig represents vault options
type VaultConfig struct {
	Addr                          string
	AuthMethod                    string
	Role                          string
	Path                          string
	SkipVerify                    bool
	TLSSecret                     string
	ClientTimeout                 time.Duration
	TransitKeyID                  string
	TransitPath                   string
	TransitBatchSize              int
	PspAllowPrivilegeEscalation   bool
	RunAsNonRoot                  bool
	RunAsUser                     int64
	RunAsGroup                    int64
	ReadOnlyRootFilesystem        bool
	RegistrySkipVerify            bool
	IgnoreMissingSecrets          string
	VaultEnvPassThrough           string
	EnableJSONLog                 string
	LogLevel                      string
	ServiceAccountTokenVolumeName string
	EnvImage                      string
	EnvImagePullPolicy            corev1.PullPolicy
	EnvLogServer                  string
	Skip                          bool
	VaultEnvFromPath              string
	EnvCPURequest                 resource.Quantity
	EnvMemoryRequest              resource.Quantity
	EnvCPULimit                   resource.Quantity
	EnvMemoryLimit                resource.Quantity
	VaultNamespace                string
	VaultServiceAccount           string
	ObjectNamespace               string
	MutateProbes                  bool
	Token                         string
}

func parseVaultConfig(obj metav1.Object, ar *model.AdmissionReview) VaultConfig {
	vaultConfig := VaultConfig{
		ObjectNamespace: ar.Namespace,
	}

	annotations := obj.GetAnnotations()

	if val := annotations[common.MutateAnnotation]; val == "skip" {
		vaultConfig.Skip = true

		return vaultConfig
	}

	if val, ok := annotations[common.VaultAddrAnnotation]; ok {
		vaultConfig.Addr = val
	} else {
		vaultConfig.Addr = viper.GetString("addr")
	}

	if val, ok := annotations[common.VaultRoleAnnotation]; ok {
		vaultConfig.Role = val
	} else {
		if val := viper.GetString("role"); val != "" {
			vaultConfig.Role = val
		} else {
			switch p := obj.(type) {
			case *corev1.Pod:
				vaultConfig.Role = p.Spec.ServiceAccountName
			default:
				vaultConfig.Role = "default"
			}
		}
	}

	if val, ok := annotations[common.VaultAuthMethodAnnotation]; ok {
		vaultConfig.AuthMethod = val
	} else {
		vaultConfig.AuthMethod = viper.GetString("auth_method")
	}

	if val, ok := annotations[common.VaultPathAnnotation]; ok {
		vaultConfig.Path = val
	} else {
		vaultConfig.Path = viper.GetString("auth_path")
	}

	// TODO: Check for flag to verify we want to use namespace-local SAs instead of the vault webhook namespaces SA
	if val, ok := annotations[common.VaultServiceaccountAnnotation]; ok {
		vaultConfig.VaultServiceAccount = val
	} else {
		vaultConfig.VaultServiceAccount = viper.GetString("serviceaccount")
	}

	if val, ok := annotations[common.VaultSkipVerifyAnnotation]; ok {
		vaultConfig.SkipVerify, _ = strconv.ParseBool(val)
	} else {
		vaultConfig.SkipVerify = viper.GetBool("tls_skip_verify")
	}

	if val, ok := annotations[common.VaultTLSSecretAnnotation]; ok {
		vaultConfig.TLSSecret = val
	} else {
		vaultConfig.TLSSecret = viper.GetString("tls_secret")
	}

	if val, ok := annotations[common.VaultClientTimeoutAnnotation]; ok {
		vaultConfig.ClientTimeout, _ = time.ParseDuration(val)
	} else {
		vaultConfig.ClientTimeout, _ = time.ParseDuration(viper.GetString("client_timeout"))
	}

	if val, ok := annotations[common.ServiceAccountTokenVolumeNameAnnotation]; ok {
		vaultConfig.ServiceAccountTokenVolumeName = val
	} else if viper.GetString("SERVICE_ACCOUNT_TOKEN_VOLUME_NAME") != "" {
		vaultConfig.ServiceAccountTokenVolumeName = viper.GetString("SERVICE_ACCOUNT_TOKEN_VOLUME_NAME")
	} else {
		vaultConfig.ServiceAccountTokenVolumeName = "/var/run/secrets/kubernetes.io/serviceaccount"
	}

	if val, ok := annotations[common.VaultIgnoreMissingSecretsAnnotation]; ok {
		vaultConfig.IgnoreMissingSecrets = val
	} else {
		vaultConfig.IgnoreMissingSecrets = viper.GetString("ignore_missing_secrets")
	}
	if val, ok := annotations[common.VaultEnvPassthroughAnnotation]; ok {
		vaultConfig.VaultEnvPassThrough = val
	} else {
		vaultConfig.VaultEnvPassThrough = viper.GetString("vault_env_passthrough")
	}

	if val, ok := annotations[common.PSPAllowPrivilegeEscalationAnnotation]; ok {
		vaultConfig.PspAllowPrivilegeEscalation, _ = strconv.ParseBool(val)
	} else {
		vaultConfig.PspAllowPrivilegeEscalation, _ = strconv.ParseBool(viper.GetString("psp_allow_privilege_escalation"))
	}

	if val, ok := annotations[common.RunAsNonRootAnnotation]; ok {
		vaultConfig.RunAsNonRoot, _ = strconv.ParseBool(val)
	} else {
		vaultConfig.RunAsNonRoot, _ = strconv.ParseBool(viper.GetString("run_as_non_root"))
	}

	if val, ok := annotations[common.RunAsUserAnnotation]; ok {
		vaultConfig.RunAsUser, _ = strconv.ParseInt(val, 10, 64)
	} else {
		vaultConfig.RunAsUser, _ = strconv.ParseInt(viper.GetString("run_as_user"), 0, 64)
	}

	if val, ok := annotations[common.RunAsGroupAnnotation]; ok {
		vaultConfig.RunAsGroup, _ = strconv.ParseInt(val, 10, 64)
	} else {
		vaultConfig.RunAsGroup, _ = strconv.ParseInt(viper.GetString("run_as_group"), 0, 64)
	}

	if val, ok := annotations[common.ReadOnlyRootFsAnnotation]; ok {
		vaultConfig.ReadOnlyRootFilesystem, _ = strconv.ParseBool(val)
	} else {
		vaultConfig.ReadOnlyRootFilesystem, _ = strconv.ParseBool(viper.GetString("readonly_root_fs"))
	}

	if val, ok := annotations[common.RegistrySkipVerifyAnnotation]; ok {
		vaultConfig.RegistrySkipVerify, _ = strconv.ParseBool(val)
	} else {
		vaultConfig.RegistrySkipVerify, _ = strconv.ParseBool(viper.GetString("registry_skip_verify"))
	}

	if val, ok := annotations[common.LogLevelAnnotation]; ok {
		vaultConfig.LogLevel = val
	} else {
		vaultConfig.LogLevel = viper.GetString("log_level")
	}

	if val, ok := annotations[common.EnableJSONLogAnnotation]; ok {
		vaultConfig.EnableJSONLog = val
	} else {
		vaultConfig.EnableJSONLog = viper.GetString("enable_json_log")
	}

	if val, ok := annotations[common.TransitKeyIDAnnotation]; ok {
		vaultConfig.TransitKeyID = val
	} else {
		vaultConfig.TransitKeyID = viper.GetString("transit_key_id")
	}

	if val, ok := annotations[common.TransitPathAnnotation]; ok {
		vaultConfig.TransitPath = val
	} else {
		vaultConfig.TransitPath = viper.GetString("transit_path")
	}

	if val, ok := annotations[common.VaultEnvFromPathAnnotation]; ok {
		vaultConfig.VaultEnvFromPath = val
	}

	if val, ok := annotations[common.VaultEnvImageAnnotation]; ok {
		vaultConfig.EnvImage = val
	} else {
		vaultConfig.EnvImage = viper.GetString("env_injector_image")
	}

	vaultConfig.EnvLogServer = viper.GetString("VAULT_ENV_LOG_SERVER")

	if val, ok := annotations[common.VaultEnvImagePullPolicyAnnotation]; ok {
		vaultConfig.EnvImagePullPolicy = getPullPolicy(val)
	} else {
		vaultConfig.EnvImagePullPolicy = getPullPolicy(viper.GetString("env_injector_pull_policy"))
	}

	if val, ok := annotations[common.VaultNamespaceAnnotation]; ok {
		vaultConfig.VaultNamespace = val
	} else {
		vaultConfig.VaultNamespace = viper.GetString("VAULT_NAMESPACE")
	}

	if val, err := resource.ParseQuantity(viper.GetString("VAULT_ENV_CPU_REQUEST")); err == nil {
		vaultConfig.EnvCPURequest = val
	} else {
		vaultConfig.EnvCPURequest = resource.MustParse("50m")
	}

	if val, err := resource.ParseQuantity(viper.GetString("VAULT_ENV_MEMORY_REQUEST")); err == nil {
		vaultConfig.EnvMemoryRequest = val
	} else {
		vaultConfig.EnvMemoryRequest = resource.MustParse("64Mi")
	}

	if val, err := resource.ParseQuantity(viper.GetString("VAULT_ENV_CPU_LIMIT")); err == nil {
		vaultConfig.EnvCPULimit = val
	} else {
		vaultConfig.EnvCPULimit = resource.MustParse("250m")
	}

	if val, err := resource.ParseQuantity(viper.GetString("VAULT_ENV_MEMORY_LIMIT")); err == nil {
		vaultConfig.EnvMemoryLimit = val
	} else {
		vaultConfig.EnvMemoryLimit = resource.MustParse("64Mi")
	}

	if val, ok := annotations[common.MutateProbesAnnotation]; ok {
		vaultConfig.MutateProbes, _ = strconv.ParseBool(val)
	} else {
		vaultConfig.MutateProbes = false
	}

	if val, ok := annotations[common.TransitBatchSizeAnnotation]; ok {
		batchSize, _ := strconv.ParseInt(val, 10, 32)
		vaultConfig.TransitBatchSize = int(batchSize)
	} else {
		vaultConfig.TransitBatchSize = viper.GetInt("transit_batch_size")
	}

	vaultConfig.Token = viper.GetString("vault_token")

	return vaultConfig
}

func getPullPolicy(pullPolicyStr string) corev1.PullPolicy {
	switch pullPolicyStr {
	case "Never", "never":
		return corev1.PullNever
	case "Always", "always":
		return corev1.PullAlways
	case "IfNotPresent", "ifnotpresent":
		return corev1.PullIfNotPresent
	}

	return corev1.PullIfNotPresent
}

func SetConfigDefaults() {
	viper.SetDefault("env_injector_image", "trublast/env-injector:v0.0.1")
	viper.SetDefault("env_injector_pull_policy", string(corev1.PullIfNotPresent))
	viper.SetDefault("addr", "https://stronghold.d8-stronghold:8200")
	viper.SetDefault("tls_skip_verify", "false")
	viper.SetDefault("auth_path", "kubernetes_local")
	viper.SetDefault("auth_method", "jwt")
	viper.SetDefault("role", "")
	viper.SetDefault("tls_secret", "")
	viper.SetDefault("client_timeout", "10s")
	viper.SetDefault("psp_allow_privilege_escalation", "false")
	viper.SetDefault("run_as_non_root", "false")
	viper.SetDefault("run_as_user", "0")
	viper.SetDefault("run_as_group", "0")
	viper.SetDefault("readonly_root_fs", "true")
	viper.SetDefault("ignore_missing_secrets", "false")
	viper.SetDefault("vault_env_passthrough", "")
	viper.SetDefault("tls_cert_file", "")
	viper.SetDefault("tls_private_key_file", "")
	viper.SetDefault("listen_address", ":8443")
	viper.SetDefault("telemetry_listen_address", "")
	viper.SetDefault("transit_key_id", "")
	viper.SetDefault("transit_path", "")
	viper.SetDefault("transit_batch_size", 25)
	viper.SetDefault("default_image_pull_secret", "")
	viper.SetDefault("default_image_pull_secret_service_account", "")
	viper.SetDefault("default_image_pull_secret_namespace", "")
	viper.SetDefault("registry_skip_verify", "false")
	viper.SetDefault("enable_json_log", "false")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("VAULT_ENV_CPU_REQUEST", "")
	viper.SetDefault("VAULT_ENV_MEMORY_REQUEST", "")
	viper.SetDefault("VAULT_ENV_CPU_LIMIT", "")
	viper.SetDefault("VAULT_ENV_MEMORY_LIMIT", "")
	viper.SetDefault("VAULT_ENV_LOG_SERVER", "")
	viper.SetDefault("VAULT_NAMESPACE", "")

	viper.AutomaticEnv()
}
