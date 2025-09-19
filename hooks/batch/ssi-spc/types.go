// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package ssi_spc

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Provider enum for all the provider names
type Provider string

// SecretObjectData defines the desired state of synced K8s secret object data
type SecretObjectData struct {
	// name of the object to sync
	ObjectName string `json:"objectName,omitempty"`
	// data field to populate
	Key          string `json:"key,omitempty"`
	DecodeBase64 bool   `json:"decodeBase64,omitempty"`
}

// SecretObject defines the desired state of synced K8s secret objects
type SecretObject struct {
	// name of the K8s secret object
	SecretName string `json:"secretName,omitempty"`
	// type of K8s secret object
	Type string `json:"type,omitempty"`
	// labels of K8s secret object
	Labels map[string]string `json:"labels,omitempty"`
	// annotations of k8s secret object
	Annotations map[string]string   `json:"annotations,omitempty"`
	Data        []*SecretObjectData `json:"data,omitempty"`
}

// SecretProviderClassSpec defines the desired state of SecretProviderClass
type SecretProviderClassSpec struct {
	// Configuration for provider name
	Provider Provider `json:"provider,omitempty"`
	// Configuration for specific provider
	Parameters    map[string]string `json:"parameters,omitempty"`
	SecretObjects []*SecretObject   `json:"secretObjects,omitempty"`
}

// SecretProviderClassStatus defines the observed state of SecretProviderClass
type SecretProviderClassStatus struct {
}

// SecretProviderClass is the Schema for the secretproviderclasses API
type SecretProviderClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretProviderClassSpec   `json:"spec,omitempty"`
	Status SecretProviderClassStatus `json:"status,omitempty"`
}

// SecretProviderClassList contains a list of SecretProviderClass
type SecretProviderClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretProviderClass `json:"items"`
}
