// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package ssi_spc

import (
	"context"
	"fmt"

	_ "github.com/deckhouse/module-sdk/common-hooks/ensure_crds"
	"github.com/deckhouse/module-sdk/pkg"
	objectpatch "github.com/deckhouse/module-sdk/pkg/object-patch"
	"github.com/deckhouse/module-sdk/pkg/registry"
	"github.com/deckhouse/module-sdk/pkg/utils/ptr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = registry.RegisterFunc(configUpstreamCRDCreateUpdate, HookHandler)

const (
	SPCKind       string = "SecretProviderClass"
	SPCapiVersion string = "secrets-store.csi.x-k8s.io/v1"
)

// SecretProviderClass is the Schema for the secretproviderclasses API
type SecretProviderClass struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`
}

var configUpstreamCRDCreateUpdate = &pkg.HookConfig{
	OnBeforeHelm: &pkg.OrderedConfig{Order: 10},
	Kubernetes: []pkg.KubernetesConfig{
		{
			Name:       "secret-provider-class",
			APIVersion: SPCapiVersion,
			Kind:       SPCKind,
			LabelSelector: &v1.LabelSelector{
				MatchLabels: map[string]string{"heritage": "deckhouse", "module": "secrets-store-integration"},
			},
			JqFilter:                     `{metadata: .metadata}`,
			ExecuteHookOnEvents:          ptr.Bool(false),
			ExecuteHookOnSynchronization: ptr.Bool(true),
		},
	},
	Queue: "/modules/secrets-store-integration/secrets-store",
}

func HookHandler(ctx context.Context, input *pkg.HookInput) error {
	spcList, err := objectpatch.UnmarshalToStruct[SecretProviderClass](input.Snapshots, "secret-provider-class")
	if err != nil {
		return fmt.Errorf("unmarshal to struct: %w", err)
	}

	for i := range spcList {
		spc := &spcList[i]
		if spc.Labels != nil {
			if _, exists := spc.Labels["heritage"]; exists {
				mergePatch := map[string]any{
					"metadata": map[string]any{
						"labels": map[string]any{
							"heritage": nil,
						},
					},
				}
				input.PatchCollector.PatchWithMerge(
					mergePatch,
					SPCapiVersion,
					SPCKind,
					spc.Namespace,
					spc.Name,
				)
			}
		}
	}
	return nil
}
