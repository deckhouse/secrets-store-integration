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

	if val, ok := annotations[common.VaultRoleAnnotation]; ok {
		vaultConfig.Role = val
	} else {
		vaultConfig.Skip = true
		return vaultConfig
	}

	vaultConfig.Addr = viper.GetString("addr")
	vaultConfig.AuthMethod = "jwt"
	vaultConfig.Path = viper.GetString("auth_path")

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

	vaultConfig.ServiceAccountTokenVolumeName = "/var/run/secrets/kubernetes.io/serviceaccount"

	if val, ok := annotations[common.VaultIgnoreMissingSecretsAnnotation]; ok {
		vaultConfig.IgnoreMissingSecrets = val
	} else {
		vaultConfig.IgnoreMissingSecrets = viper.GetString("ignore_missing_secrets")
	}

	vaultConfig.PspAllowPrivilegeEscalation = false
	vaultConfig.RunAsNonRoot = true

	vaultConfig.RunAsUser = 64535
	vaultConfig.RunAsGroup = 64535

	vaultConfig.ReadOnlyRootFilesystem = true

	vaultConfig.RegistrySkipVerify = true

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

	if val, ok := annotations[common.VaultEnvFromPathAnnotation]; ok {
		vaultConfig.VaultEnvFromPath = val
	}

	vaultConfig.EnvImage = viper.GetString("env_injector_image")

	vaultConfig.EnvLogServer = viper.GetString("VAULT_ENV_LOG_SERVER")

	if val, ok := annotations[common.VaultNamespaceAnnotation]; ok {
		vaultConfig.VaultNamespace = val
	} else {
		vaultConfig.VaultNamespace = viper.GetString("VAULT_NAMESPACE")
	}

	vaultConfig.EnvCPURequest = resource.MustParse("50m")
	vaultConfig.EnvMemoryRequest = resource.MustParse("64Mi")
	vaultConfig.EnvCPULimit = resource.MustParse("250m")
	vaultConfig.EnvMemoryLimit = resource.MustParse("64Mi")

	if val, ok := annotations[common.MutateProbesAnnotation]; ok {
		vaultConfig.MutateProbes, _ = strconv.ParseBool(val)
	} else {
		vaultConfig.MutateProbes = false
	}

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
	// viper.SetDefault("addr", "https://stronghold.d8-stronghold:8200")
	viper.SetDefault("tls_skip_verify", "false")
	// viper.SetDefault("auth_path", "kubernetes_local")
	viper.SetDefault("auth_method", "jwt")
	viper.SetDefault("role", "")
	viper.SetDefault("tls_secret", "")
	viper.SetDefault("client_timeout", "10s")
	viper.SetDefault("ignore_missing_secrets", "false")
	viper.SetDefault("tls_cert_file", "")
	viper.SetDefault("tls_private_key_file", "")
	viper.SetDefault("listen_address", ":8443")
	viper.SetDefault("telemetry_listen_address", "")
	viper.SetDefault("registry_skip_verify", "false")
	viper.SetDefault("enable_json_log", "false")
	viper.SetDefault("log_level", "info")

	viper.AutomaticEnv()
}
