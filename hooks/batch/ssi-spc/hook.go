// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package ssi_spc

import (
	"context"
	"fmt"
	ssi "secrets-store-integration-hook/api"
	"secrets-store-integration-hook/consts"
	"strings"

	_ "github.com/deckhouse/module-sdk/common-hooks/ensure_crds"
	"github.com/deckhouse/module-sdk/pkg"
	objectpatch "github.com/deckhouse/module-sdk/pkg/object-patch"
	"github.com/deckhouse/module-sdk/pkg/registry"
	"github.com/deckhouse/module-sdk/pkg/utils/ptr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = registry.RegisterFunc(configUpstreamCRDCreateUpdate, HookHandler)

var JQFilter = `{
	metadata: .metadata,
	spec:.spec
}`

type id struct {
	name      string
	namespace string
}

var secretProviderClassTemplate = SecretProviderClass{
	TypeMeta: metav1.TypeMeta{
		Kind:       consts.SPCKind,
		APIVersion: consts.SPCapiVersion,
	},
	ObjectMeta: metav1.ObjectMeta{
		Labels: map[string]string{
			"heritage": "deckhouse",
			"module":   "secrets-store-integration",
		},
	},
	Spec: SecretProviderClassSpec{
		Provider:   "vault",
		Parameters: map[string]string{},
		SecretObjects: []*SecretObject{
			{
				Type: "Opaque",
			},
		},
	},
}

var configUpstreamCRDCreateUpdate = &pkg.HookConfig{
	OnBeforeHelm: &pkg.OrderedConfig{Order: 10},
	Kubernetes: []pkg.KubernetesConfig{
		{
			Name:       "secrets-store-import",
			APIVersion: consts.SSIapiVersion,
			Kind:       consts.SSIKind,
			JqFilter:   JQFilter,
		},
		{
			Name:                         "secret-provider-class",
			APIVersion:                   consts.SPCapiVersion,
			Kind:                         consts.SPCKind,
			JqFilter:                     `{ metadata: .metadata }`,
			ExecuteHookOnEvents:          ptr.Bool(false),
			ExecuteHookOnSynchronization: ptr.Bool(false),
		},
	},
	Queue: "/modules/secrets-store-integration/secrets-store",
}

func HookHandler(ctx context.Context, input *pkg.HookInput) error {
	ssiList, err := objectpatch.UnmarshalToStruct[ssi.SecretStoreImport](input.Snapshots, "secrets-store-import")
	if err != nil {
		return fmt.Errorf("unmarshal to struct: %w", err)
	}
	spcList, err := objectpatch.UnmarshalToStruct[SecretProviderClass](input.Snapshots, "secret-provider-class")
	if err != nil {
		return fmt.Errorf("unmarshal to struct: %w", err)
	}
	var spcExistanceMap = make(map[id]struct{}, len(spcList))
	var ssiExistanceMap = make(map[id]struct{}, len(ssiList))

	for i := range ssiList {
		ssiExistanceMap[id{ssiList[i].Name, ssiList[i].Namespace}] = struct{}{}
	}
	for i := range spcList {
		spcExistanceMap[id{spcList[i].Name, spcList[i].Namespace}] = struct{}{}
		if _, ok := ssiExistanceMap[id{spcList[i].Name, spcList[i].Namespace}]; !ok {
			if spcList[i].Labels["heritage"] == "deckhouse" && spcList[i].Labels["module"] == "secrets-store-integration" {
				input.PatchCollector.Delete(consts.SPCapiVersion, consts.SPCKind, spcList[i].Namespace, spcList[i].Name)
			}
		}
	}
	for i := range ssiList {
		spc := secretProviderClassTemplate
		err := deepCopy(&ssiList[i], &spc)
		if err != nil {
			return fmt.Errorf("deepcopy: %w", err)
		}
		if _, ok := spcExistanceMap[id{ssiList[i].Name, ssiList[i].Namespace}]; !ok {
			input.PatchCollector.CreateIfNotExists(&spc)
		} else {
			input.PatchCollector.CreateOrUpdate(&spc)
		}
	}

	return nil
}

func deepCopy(ssi *ssi.SecretStoreImport, spc *SecretProviderClass) error {
	spc.Name = ssi.Name
	spc.Namespace = ssi.Namespace
	spc.Spec.Parameters["roleName"] = ssi.Spec.Role
	spc.Spec.Parameters["vaultAuthMountPath"] = ssi.Spec.AuthPath
	spc.Spec.Parameters["vaultNamespace"] = ssi.Spec.Namespace
	spc.Spec.Parameters["vaultAddress"] = ssi.Spec.Address
	spc.Spec.Parameters["vaultCACert"] = ssi.Spec.CACert
	spc.Spec.Parameters["audience"] = ssi.Spec.Audience
	spc.Spec.Parameters["vaultSkipTLSVerify"] = "false"
	if ssi.Spec.SkipTLSVerify {
		spc.Spec.Parameters["vaultSkipTLSVerify"] = "true"
	}
	spc.Spec.SecretObjects[0].SecretName = ssi.Name
	spc.Spec.SecretObjects[0].Data = make([]*SecretObjectData, 0, len(ssi.Spec.Files))
	var sb strings.Builder
	for _, object := range ssi.Spec.Files {
		_, err := sb.WriteString(fmt.Sprintf("- objectName: \"%s\"\n  secretPath: \"%s\"\n  secretKey: \"%s\"\n", object.Name, object.Source.Path, object.Source.Key))
		if err != nil {
			return fmt.Errorf("error populating objects: %w", err)
		}
		spc.Spec.SecretObjects[0].Data = append(spc.Spec.SecretObjects[0].Data, &SecretObjectData{
			Key:          object.Source.Key,
			ObjectName:   object.Name,
			DecodeBase64: object.DecodeBase64,
		})
	}
	spc.Spec.Parameters["objects"] = sb.String()
	return nil
}
