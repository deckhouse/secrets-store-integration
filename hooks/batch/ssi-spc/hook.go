// Copyright (c) Flant JSC
// SPDX-License-Identifier: Apache-2.0

package ssi_spc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	ssi "secrets-store-integration-hook/api"
	"secrets-store-integration-hook/consts"

	"github.com/deckhouse/module-sdk/pkg"
	objectpatch "github.com/deckhouse/module-sdk/pkg/object-patch"
	"github.com/deckhouse/module-sdk/pkg/registry"
	"github.com/deckhouse/module-sdk/pkg/utils/ptr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	_ "github.com/deckhouse/module-sdk/common-hooks/ensure_crds"
)

var _ = registry.RegisterFunc(configUpstreamCRDCreateUpdate, HookHandler)

var JQFilter = `{
	metadata: .metadata,
	spec: .spec
}`

type id struct {
	name      string
	namespace string
}

// vaultObject - объект для параметра "objects" в SecretProviderClass.
type vaultObject struct {
	ObjectName string `yaml:"objectName"`
	SecretPath string `yaml:"secretPath"`
	SecretKey  string `yaml:"secretKey"`
}

// requiredLabels — Labels которые должны быть на всех SPC созданных этим модулем
var requiredLabels = map[string]string{
	"heritage": "deckhouse",
	"module":   "secrets-store-integration",
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
			JqFilter:                     JQFilter,
			ExecuteHookOnEvents:          ptr.Bool(false),
			ExecuteHookOnSynchronization: ptr.Bool(false),
		},
	},
	Queue: "/modules/secrets-store-integration/secrets-store",
}

// newSecretProviderClass конструктор SecretProviderClass
func newSecretProviderClass() SecretProviderClass {
	return SecretProviderClass{
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
			Parameters: make(map[string]string),
			SecretObjects: []*SecretObject{
				{
					Type: "Opaque",
				},
			},
		},
	}
}

// specHash вычисляет хэш от Spec для сравнения объектов
func specHash(spec SecretProviderClassSpec) (string, error) {
	data, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshal spec: %w", err)
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash[:16]), nil
}

// hasRequiredLabels проверяет наличие обязательных Labels на SPC
func hasRequiredLabels(spc SecretProviderClass) bool {
	for k, v := range requiredLabels {
		if spc.Labels[k] != v {
			return false
		}
	}
	return true
}

// needsUpdate проверяет, нужно ли обновлять существующий SPC
func needsUpdate(existing, new SecretProviderClass) (bool, error) {
	// Labels изменились
	if !hasRequiredLabels(existing) {
		return true, nil
	}

	// Spec изменился
	existingHash, err := specHash(existing.Spec)
	if err != nil {
		return false, fmt.Errorf("hash existing spec: %w", err)
	}

	newHash, err := specHash(new.Spec)
	if err != nil {
		return false, fmt.Errorf("hash new spec: %w", err)
	}

	return existingHash != newHash, nil
}

func HookHandler(_ context.Context, input *pkg.HookInput) error {
	ssiList, err := objectpatch.UnmarshalToStruct[ssi.SecretStoreImport](input.Snapshots, "secrets-store-import")
	if err != nil {
		return fmt.Errorf("unmarshal SSI to struct: %w", err)
	}

	spcList, err := objectpatch.UnmarshalToStruct[SecretProviderClass](input.Snapshots, "secret-provider-class")
	if err != nil {
		return fmt.Errorf("unmarshal SPC to struct: %w", err)
	}

	ssiExistenceMap := make(map[id]struct{}, len(ssiList))
	for i := range ssiList {
		ssiExistenceMap[id{ssiList[i].Name, ssiList[i].Namespace}] = struct{}{}
	}

	spcFullMap := make(map[id]SecretProviderClass, len(spcList))

	// Обрабатываем существующие SPC
	for i := range spcList {
		spcID := id{spcList[i].Name, spcList[i].Namespace}

		// Проверяем что JQFilter вернул spec
		if spcList[i].Spec.Provider == "" {
			input.Logger.Warn("SPC has empty spec, skipping",
				"name", spcList[i].Name,
				"namespace", spcList[i].Namespace)
			continue
		}

		// SPC имеет соответствующий SSI — сохраняем для сравнения
		if _, exists := ssiExistenceMap[spcID]; exists {
			input.Logger.Debug("SPC has matching SSI, keeping",
				"name", spcList[i].Name,
				"namespace", spcList[i].Namespace)
			spcFullMap[spcID] = spcList[i]
			continue
		}

		// SPC без наших Labels — не трогаем, но и не сохраняем
		if !hasRequiredLabels(spcList[i]) {
			input.Logger.Debug("SPC missing required labels, skipping",
				"name", spcList[i].Name,
				"namespace", spcList[i].Namespace)
			continue
		}

		// Осиротевший SPC с нашими Labels — удаляем
		input.Logger.Debug("deleting orphaned SPC",
			"name", spcList[i].Name,
			"namespace", spcList[i].Namespace)
		input.PatchCollector.Delete(consts.SPCapiVersion, consts.SPCKind, spcList[i].Namespace, spcList[i].Name)
	}

	// Обрабатываем каждый SSI
	for i := range ssiList {
		spc := newSecretProviderClass()
		if err := deepCopy(&ssiList[i], &spc); err != nil {
			return fmt.Errorf("deepCopy SSI %s/%s: %w", ssiList[i].Namespace, ssiList[i].Name, err)
		}

		spcID := id{ssiList[i].Name, ssiList[i].Namespace}
		existingSPC, exists := spcFullMap[spcID]

		// SPC не существует — создаём
		if !exists {
			input.Logger.Debug("creating new SPC",
				"name", ssiList[i].Name,
				"namespace", ssiList[i].Namespace)
			input.PatchCollector.CreateIfNotExists(&spc)
			continue
		}

		// SPC существует — проверяем нужно ли обновлять
		updated, err := needsUpdate(existingSPC, spc)
		if err != nil {
			return fmt.Errorf("check needsUpdate for %s/%s: %w", ssiList[i].Namespace, ssiList[i].Name, err)
		}

		if !updated {
			input.Logger.Debug("SPC unchanged, skipping",
				"name", ssiList[i].Name,
				"namespace", ssiList[i].Namespace)
			continue
		}

		input.Logger.Debug("updating SPC",
			"name", ssiList[i].Name,
			"namespace", ssiList[i].Namespace)
		input.PatchCollector.CreateOrUpdate(&spc)
	}

	return nil
}

// deepCopy копирует данные из SecretsStoreImport в SecretProviderClass
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

	vaultObjects := make([]vaultObject, 0, len(ssi.Spec.Files))
	for _, file := range ssi.Spec.Files {
		vaultObjects = append(vaultObjects, vaultObject{
			ObjectName: file.Name,
			SecretPath: file.Source.Path,
			SecretKey:  file.Source.Key,
		})
		spc.Spec.SecretObjects[0].Data = append(spc.Spec.SecretObjects[0].Data, &SecretObjectData{
			Key:          file.Source.Key,
			ObjectName:   file.Name,
			DecodeBase64: file.DecodeBase64,
		})
	}

	objectsYAML, err := yaml.Marshal(vaultObjects)
	if err != nil {
		return fmt.Errorf("marshal vault objects: %w", err)
	}
	spc.Spec.Parameters["objects"] = string(objectsYAML)

	return nil
}
