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

package controller

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/deckhouse/deckhouse/pkg/log"
	deckhouseiov1alpha1 "github.com/deckhouse/ssi-controller/api/v1alpha1"
)

// SecretsStoreImportReconciler reconciles a SecretsStoreImport object
type SecretsStoreImportReconciler struct {
	client.Client
	Log    log.Logger
	Scheme *runtime.Scheme
}

type vaultObject struct {
	ObjectName string `yaml:"objectName"`
	SecretPath string `yaml:"secretPath"`
	SecretKey  string `yaml:"secretKey"`
}

// +kubebuilder:rbac:groups=deckhouse.io,resources=secretsstoreimports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=deckhouse.io,resources=secretsstoreimports/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=deckhouse.io,resources=secretsstoreimports/finalizers,verbs=update
// +kubebuilder:rbac:groups=secrets-store.csi.x-k8s.io,resources=secretproviderclasses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SecretsStoreImport object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *SecretsStoreImportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	ssi := &deckhouseiov1alpha1.SecretsStoreImport{}
	err := r.Get(ctx, req.NamespacedName, ssi)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Resource not found. Ignoring since object must be deleted", "name", ssi.Name, "namespace", ssi.Namespace)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error("Failed to get resource", "error", err, "name", ssi.Name, "namespace", ssi.Namespace)
		return ctrl.Result{}, err
	}

	// Create SecretProviderClass as unstructured from SecretsStoreImport
	spcUnstructured, err := createSecretProviderClassFromSSI(ssi)
	if err != nil {
		log.Error("Failed to create SecretProviderClass from SecretsStoreImport", "error", err, "name", ssi.Name, "namespace", ssi.Namespace)
		return ctrl.Result{}, err
	}

	// Try to get existing SecretProviderClass
	existingSPC := &unstructured.Unstructured{}
	existingSPC.SetGroupVersionKind(spcUnstructured.GroupVersionKind())
	err = r.Get(ctx, client.ObjectKey{Name: ssi.Name, Namespace: ssi.Namespace}, existingSPC)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object doesn't exist, create it
			log.Info("Creating SecretProviderClass", "name", ssi.Name, "namespace", ssi.Namespace)
			if err := r.Create(ctx, spcUnstructured); err != nil {
				log.Error("Failed to create SecretProviderClass", "error", err, "name", ssi.Name, "namespace", ssi.Namespace)
				return ctrl.Result{}, err
			}
			log.Info("Successfully created SecretProviderClass", "name", ssi.Name, "namespace", ssi.Namespace)
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get SecretProviderClass", "error", err)
		return ctrl.Result{}, err
	}
	// Object exists, update it
	log.Info("Updating SecretProviderClass", "name", ssi.Name, "namespace", ssi.Namespace)
	spcUnstructured.SetResourceVersion(existingSPC.GetResourceVersion())
	if err := r.Update(ctx, spcUnstructured); err != nil {
		log.Error("Failed to update SecretProviderClass", "error", err)
		return ctrl.Result{}, err
	}
	log.Info("Successfully updated SecretProviderClass", "name", ssi.Name, "namespace", ssi.Namespace)

	return ctrl.Result{}, nil
}

// createSecretProviderClassFromSSI creates an unstructured SecretProviderClass from SecretsStoreImport
func createSecretProviderClassFromSSI(ssi *deckhouseiov1alpha1.SecretsStoreImport) (*unstructured.Unstructured, error) {
	// Build parameters map
	parameters := map[string]interface{}{
		"roleName": ssi.Spec.Role,
	}
	if ssi.Spec.AuthPath != "" {
		parameters["vaultAuthMountPath"] = ssi.Spec.AuthPath
	}
	if ssi.Spec.Namespace != "" {
		parameters["vaultNamespace"] = ssi.Spec.Namespace
	}
	if ssi.Spec.Address != "" {
		parameters["vaultAddress"] = ssi.Spec.Address
	}
	if ssi.Spec.CACert != "" {
		parameters["vaultCACert"] = ssi.Spec.CACert
	}
	if ssi.Spec.Audience != "" {
		parameters["audience"] = ssi.Spec.Audience
	}
	vaultSkipTLSVerify := "false"
	if ssi.Spec.SkipTLSVerify {
		vaultSkipTLSVerify = "true"
	}
	parameters["vaultSkipTLSVerify"] = vaultSkipTLSVerify

	// Build objects YAML string and secret object data
	vaultObjects := make([]vaultObject, 0, len(ssi.Spec.Files))
	secretObjectData := make([]interface{}, 0, len(ssi.Spec.Files))
	for _, file := range ssi.Spec.Files {
		vaultObjects = append(vaultObjects, vaultObject{
			ObjectName: file.Name,
			SecretPath: file.Source.Path,
			SecretKey:  file.Source.Key,
		})
		secretObjectData = append(secretObjectData, map[string]interface{}{
			"key":          file.Source.Key,
			"objectName":   file.Name,
			"decodeBase64": file.DecodeBase64,
		})
	}
	objectsYAML, err := yaml.Marshal(vaultObjects)
	if err != nil {
		return nil, fmt.Errorf("marshal vault objects: %w", err)
	}
	parameters["objects"] = string(objectsYAML)

	// Build secret objects array
	secretObjects := []interface{}{
		map[string]interface{}{
			"secretName": ssi.Name,
			"type":       "Opaque",
			"data":       secretObjectData,
		},
	}

	// Build the unstructured object
	spc := &unstructured.Unstructured{}
	spc.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "secrets-store.csi.x-k8s.io",
		Version: "v1",
		Kind:    "SecretProviderClass",
	})
	spc.SetName(ssi.Name)
	spc.SetNamespace(ssi.Namespace)
	spc.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by": "secrets-store-import",
	})

	// Set owner reference to SecretsStoreImport for automatic cleanup
	ownerRef := map[string]interface{}{
		"apiVersion": "deckhouse.io/v1alpha1",
		"kind":       "SecretsStoreImport",
		"name":       ssi.Name,
		"uid":        string(ssi.UID),
		"controller": true,
	}
	ownerRefs := []interface{}{ownerRef}
	if err := unstructured.SetNestedField(spc.Object, ownerRefs, "metadata", "ownerReferences"); err != nil {
		return nil, fmt.Errorf("failed to set ownerReferences: %w", err)
	}

	// Set spec
	if err := unstructured.SetNestedField(spc.Object, "vault", "spec", "provider"); err != nil {
		return nil, fmt.Errorf("failed to set provider: %w", err)
	}
	if err := unstructured.SetNestedField(spc.Object, parameters, "spec", "parameters"); err != nil {
		return nil, fmt.Errorf("failed to set parameters: %w", err)
	}
	if err := unstructured.SetNestedField(spc.Object, secretObjects, "spec", "secretObjects"); err != nil {
		return nil, fmt.Errorf("failed to set secretObjects: %w", err)
	}

	return spc, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretsStoreImportReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&deckhouseiov1alpha1.SecretsStoreImport{}).
		Named("secretsstoreimport").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		}).
		Complete(r)
}
