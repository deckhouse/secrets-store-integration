/*
Copyright 2025 Flant JSC

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type VaultSource struct {
	// Vault secretKey
	// +kubebuilder:validation:Pattern:=^[-a-zA-Z0-9_.]+$
	Key string `json:"key"`
	// Vault secretPath
	// +kubebuilder:validation:Pattern:=^[-a-zA-Z0-9_.\/]+$
	Path string `json:"path"`
}

type Files struct {
	// +required
	// Filename to create
	// +kubebuilder:validation:Pattern:=^[-_a-zA-Z0-9.]+$
	Name string `json:"name"`
	// Decode base64-encoded secret value to raw data
	// +kubebuilder:validation:default:=false
	DecodeBase64 bool `json:"decodeBase64,omitempty"`
	// Vault secret source
	// +required
	Source VaultSource `json:"source"`
}

// SecretsStoreImportSpec defines the desired state of SecretsStoreImport.
type SecretsStoreImportSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Only mode CSI supported
	// +kubebuilder:validation:Enum=CSI
	// +kubebuilder:default:=CSI
	Type string `json:"type"`
	// Role in Vault-compatible storage
	// +kubebuilder:validation:Pattern:=^[-_.a-zA-Z0-9]+$
	Role string `json:"role"`
	// Auth mount path in Vault-compatible storage. If omitted value from ModuleConfig will be used.
	// +kubebuilder:validation:Pattern:=^[A-Za-z0-9-_]+$
	AuthPath string `json:"authPath,omitempty"`
	// Namespace where secret is stored. If omitted value from ModuleConfig will be used.
	// +kubebuilder:validation:Pattern:="^[-_./a-zA-Z0-9]+$"
	Namespace string `json:"namespace,omitempty"`
	// Address of Vault-compatible storage. If omitted value from ModuleConfig will be used.
	// +kubebuilder:validation:Pattern:="^https?://[.:0-9a-zA-Z-]+$"
	Address string `json:"address,omitempty"`
	// Stronghold or Vault CA in PEM format. If omitted value from ModuleConfig will be used.
	// +kubebuilder:validation:Pattern:="^-----BEGIN CERTIFICATE-----\n(.+\n){5}"
	CACert string `json:"caCert,omitempty"`
	// JWT audience
	// +kubebuilder:validation:Pattern:=^[-_\.a-zA-Z0-9]+$
	Audience string `json:"audience,omitempty"`
	// Skip TLS verification
	// +kubebuilder:validation:default:=false
	SkipTLSVerify bool `json:"skipTLSVerify,omitempty"`

	Files []Files `json:"files,omitempty"`
}

// SecretsStoreImportStatus defines the observed state of SecretsStoreImport.
// type SecretsStoreImportStatus struct {
// 	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
// 	// Important: Run "make" to regenerate code after modifying this file
// }

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Namespaced Kubernetes Custom resource for mapping between Vault-compatible storage and filename in container
type SecretsStoreImport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SecretsStoreImportSpec `json:"spec"`
	// Status SecretsStoreImportStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecretsStoreImportList contains a list of SecretsStoreImport.
type SecretsStoreImportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretsStoreImport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretsStoreImport{}, &SecretsStoreImportList{})
}
