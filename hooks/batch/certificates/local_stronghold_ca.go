// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package certificates

import (
	"context"
	"encoding/base64"
	"fmt"
	"secrets-store-integration-hook/consts"

	"github.com/deckhouse/module-sdk/pkg"
	objectpatch "github.com/deckhouse/module-sdk/pkg/object-patch"
	"github.com/deckhouse/module-sdk/pkg/registry"
)

const (
	snapshotKey                    = "localStrongholdCACerts"
	openAPIPath                    = ".internal.localStrongholdCaCert"
	localStrongholdCaCertsFullPath = consts.DotValuesModuleName + openAPIPath
)

var _ = registry.RegisterFunc(configSaveLocalStrongholdCACert, saveLocalStrongholdCACertHandler)

var localStrongholdCACertJQFilter = `{
    "ca": .data."ca.crt"
}`

var configSaveLocalStrongholdCACert = &pkg.HookConfig{
	OnBeforeHelm: &pkg.OrderedConfig{Order: 10},
	Kubernetes: []pkg.KubernetesConfig{
		{
			Name:       snapshotKey,
			APIVersion: "v1",
			Kind:       "Secret",
			NameSelector: &pkg.NameSelector{
				MatchNames: []string{"stronghold-tls"},
			},
			NamespaceSelector: &pkg.NamespaceSelector{
				NameSelector: &pkg.NameSelector{
					MatchNames: []string{"d8-stronghold"},
				},
			},
			JqFilter: localStrongholdCACertJQFilter,
		},
	},
}

func saveLocalStrongholdCACertHandler(_ context.Context, input *pkg.HookInput) error {
	caCerts, err := objectpatch.UnmarshalToStruct[struct {
		CA string `json:"ca"`
	}](input.Snapshots, snapshotKey)

	if err != nil {
		return fmt.Errorf("failed to unmarshal to struct: %w", err)
	}

	if len(caCerts) == 0 {
		// input.Values.Remove(localStrongholdCaCertsFullPath)
		return nil
	}

	decodedCA, err := base64.StdEncoding.DecodeString(caCerts[0].CA)
	if err != nil {
		return fmt.Errorf("failed to decode CA certificate: %w", err)
	}

	input.Values.Set(localStrongholdCaCertsFullPath, string(decodedCA))

	return nil
}
