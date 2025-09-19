// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecretStoreImportSpec struct {
	AuthPath      string `json:"authPath,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	Address       string `json:"address,omitempty"`
	CACert        string `json:"caCert,omitempty"`
	Audience      string `json:"audience,omitempty"`
	SkipTLSVerify bool   `json:"skipTLSVerify,omitempty"`
	Files         []*struct {
		Name         string `json:"name,omitempty"`
		DecodeBase64 bool   `json:"decodeBase64,omitempty"`
		Source       struct {
			Key  string `json:"key,omitempty"`
			Path string `json:"path,omitempty"`
		} `json:"source,omitempty"`
	} `json:"files,omitempty"`
	Role string `json:"role,omitempty"`
	Type string `json:"type,omitempty"`
}

type SecretStoreImport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SecretStoreImportSpec `json:"spec,omitempty"`
}
