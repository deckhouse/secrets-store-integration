// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package upstream_crd

import (
	"context"
	"fmt"
	ssi "secrets-store-integration-hook/api"

	_ "github.com/deckhouse/module-sdk/common-hooks/ensure_crds"
	"github.com/deckhouse/module-sdk/pkg"
	objectpatch "github.com/deckhouse/module-sdk/pkg/object-patch"
	"github.com/deckhouse/module-sdk/pkg/registry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	sscv1 "sigs.k8s.io/secrets-store-csi-driver/apis/v1"
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

var secretProviderClassTemplate = sscv1.SecretProviderClass{
	TypeMeta: metav1.TypeMeta{
		Kind:       "SecretProviderClass",
		APIVersion: "secrets-store.csi.x-k8s.io/v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Labels: map[string]string{
			"heritage": "deckhouse",
			"module":   "secrets-store-integration",
		},
	},
}

var configUpstreamCRDCreateUpdate = &pkg.HookConfig{
	OnBeforeHelm: &pkg.OrderedConfig{Order: 10},
	Kubernetes: []pkg.KubernetesConfig{
		{
			Name:       "secrets-store-import",
			APIVersion: "deckhouse.io/v1alpha1",
			Kind:       "SecretsStoreImport",
			JqFilter:   JQFilter,
		},
		{
			Name:                         "secret-provider-class",
			APIVersion:                   "secrets-store.csi.x-k8s.io/v1",
			Kind:                         "SecretProviderClass",
			JqFilter:                     JQFilter,
			ExecuteHookOnEvents:          pointer.Bool(false),
			ExecuteHookOnSynchronization: pointer.Bool(false),
		},
	},
	Queue: "/modules/secrets-store-integration/secrets-store",
}

func HookHandler(ctx context.Context, input *pkg.HookInput) error {

	secretsStoreImportList, err := objectpatch.UnmarshalToStruct[ssi.SecretStoreImport](input.Snapshots, "secrets-store-import")
	if err != nil {
		return fmt.Errorf("unmarshal to struct: %w", err)
	}
	secretProviderClassList, err := objectpatch.UnmarshalToStruct[sscv1.SecretProviderClass](input.Snapshots, "secret-provider-class")
	if err != nil {
		return fmt.Errorf("unmarshal to struct: %w", err)
	}
	var spcExistanceMap = make(map[id]struct{}, len(secretProviderClassList))
	var ssiExistanceMap = make(map[id]struct{}, len(secretsStoreImportList))

	for _, ssi := range secretsStoreImportList {
		ssiExistanceMap[id{ssi.Metadata.Name, ssi.Metadata.Namespace}] = struct{}{}
	}
	for _, spc := range secretProviderClassList {
		spcExistanceMap[id{spc.Name, spc.Namespace}] = struct{}{}
		if _, ok := ssiExistanceMap[id{spc.Name, spc.Namespace}]; !ok {
			input.PatchCollector.Delete(spc.APIVersion, spc.Kind, spc.Namespace, spc.Name)
		}
	}
	for _, ssi := range secretsStoreImportList {
		if _, ok := spcExistanceMap[id{ssi.Metadata.Name, ssi.Metadata.Namespace}]; !ok { // maybe check for field changes
			continue
		}
		spc := secretProviderClassTemplate
		deepCopy(&ssi, &spc)
		input.PatchCollector.CreateIfNotExists(spc)
	}

	return nil
}

func deepCopy(ssi *ssi.SecretStoreImport, spc *sscv1.SecretProviderClass) {
	spc.ObjectMeta.Name = ssi.Metadata.Name
	spc.ObjectMeta.Namespace = ssi.Metadata.Namespace
	spc.Spec.Parameters["roleName"] = ssi.Spec.Role
	spc.Spec.Parameters["vaultAuthMountPath"] = ssi.Spec.AuthPath
	spc.Spec.Parameters["vaultNamespace"] = ssi.Spec.Namespace
	spc.Spec.Parameters["vaultAddress"] = ssi.Spec.Address
	spc.Spec.Parameters["vaultCACert"] = ssi.Spec.CACert
	spc.Spec.Parameters["audience"] = ssi.Spec.Audience
	if ssi.Spec.SkipTLSVerify {
		spc.Spec.Parameters["vaultSkipTLSVerify"] = "true"
	} else {
		spc.Spec.Parameters["vaultSkipTLSVerify"] = "false"
	}
	spc.Spec.SecretObjects = make([]*sscv1.SecretObject, 1)
	spc.Spec.SecretObjects[0].SecretName = ssi.Metadata.Name
	spc.Spec.SecretObjects[0].Data = make([]*sscv1.SecretObjectData, 0, len(ssi.Spec.Files))
	for _, object := range ssi.Spec.Files {
		spc.Spec.Parameters["objects"] = fmt.Sprintf("- objectName: %s\n  secretPath: %s\n  secretKey: %s\n", object.Name, object.Source.Path, object.Source.Key)
		spc.Spec.SecretObjects[0].Data = append(spc.Spec.SecretObjects[0].Data, &sscv1.SecretObjectData{
			Key:        object.Source.Key,
			ObjectName: object.Name,
		})
	}
}
