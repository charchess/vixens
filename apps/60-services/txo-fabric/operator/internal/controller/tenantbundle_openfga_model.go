package controller

import (
	"context"
	"errors"
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// openFGAModelProvisioner is a test seam for the private, platform-owned
// OpenFGA model API. No tenant workload or browser can supply a model ID.
type openFGAModelProvisioner interface {
	EnsureAuthorizationModel(context.Context, string) (string, error)
	ValidateModelBinding(context.Context, string, string) error
}

// openFGAAuthorizationBinding is resolved solely from durable Fabric-owned
// state. An incomplete binding never denotes an operational authorization
// graph: tuple reconciliation and authorization readiness remain separate.
type openFGAAuthorizationBinding struct {
	Store            openfga.Store
	ModelID          string
	ModelFingerprint string
}

func (r *TenantBundleReconciler) modelProvisioner(ctx context.Context) (openFGAModelProvisioner, error) {
	if r.OpenFGAModelClient != nil {
		return r.OpenFGAModelClient, nil
	}
	var credential corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: openFGAPlatformNamespace, Name: openFGACredentialSecret,
	}, &credential); err != nil {
		return nil, fmt.Errorf("Fabric OpenFGA model credential is not ready: %w", err)
	}
	service, err := openfga.NewClient(string(credential.Data["presharedKeys"]), nil)
	if err != nil {
		return nil, errors.New("Fabric OpenFGA model credential is invalid")
	}
	return service, nil
}

// reconcileOpenFGAModel pins the exact approved model to the existing,
// previously reconciled tenant store. The model ID and content fingerprint are
// committed together in one resource-version-checked ConfigMap update.
// Partial state, concurrent conflict or unknown model revision always denies;
// it is safe to adopt a verified publication after an interrupted reconcile.
func (r *TenantBundleReconciler) reconcileOpenFGAModel(ctx context.Context, bundle *fabricv1alpha1.TenantBundle, store openfga.Store) (string, error) {
	if bundle == nil || !bundle.DeletionTimestamp.IsZero() {
		return "", errors.New("cannot publish a model for missing or deleting TenantBundle")
	}
	boundStore, err := r.readOpenFGAStoreBinding(ctx, bundle)
	if err != nil {
		return "", fmt.Errorf("missing trusted Fabric OpenFGA tenant store: %w", err)
	}
	if store != boundStore {
		return "", errors.New("tenant OpenFGA store differs from retained Fabric binding")
	}
	fingerprint, err := openfga.ModelFingerprint()
	if err != nil {
		return "", err
	}
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil {
		return "", err
	}
	var binding corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{Namespace: openFGAPlatformNamespace, Name: name}, &binding); err != nil {
		return "", err
	}
	storedModelID := binding.Data["modelID"]
	storedFingerprint := binding.Data["modelFingerprint"]
	if (storedModelID == "") != (storedFingerprint == "") {
		return "", errors.New("partial Fabric OpenFGA authorization model binding")
	}
	if storedFingerprint != "" && storedFingerprint != fingerprint {
		return "", errors.New("Fabric OpenFGA model fingerprint drift: explicit migration required")
	}
	service, err := r.modelProvisioner(ctx)
	if err != nil {
		return "", err
	}
	if storedModelID != "" {
		if err := service.ValidateModelBinding(ctx, store.ID, storedModelID); err != nil {
			return "", fmt.Errorf("Fabric OpenFGA authorization model binding validation failed: %w", err)
		}
		return storedModelID, nil
	}

	modelID, err := service.EnsureAuthorizationModel(ctx, store.ID)
	if err != nil {
		return "", fmt.Errorf("Fabric OpenFGA authorization model publication failed: %w", err)
	}
	if modelID == "" {
		return "", errors.New("Fabric OpenFGA authorization model has no confirmed ID")
	}
	if err := service.ValidateModelBinding(ctx, store.ID, modelID); err != nil {
		return "", fmt.Errorf("Fabric OpenFGA new authorization model validation failed: %w", err)
	}

	// Never overwrite the retained store identity or another writer's model.
	// A concurrent ConfigMap update returns Conflict and retries on next
	// reconcile, where the model is revalidated without republishing.
	binding.Data["modelID"] = modelID
	binding.Data["modelFingerprint"] = fingerprint
	if err := r.Update(ctx, &binding); err != nil {
		return "", fmt.Errorf("unable to pin Fabric OpenFGA authorization model: %w", err)
	}
	confirmed, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil || confirmed.Store != store || confirmed.ModelID != modelID {
		return "", errors.New("Fabric OpenFGA model binding not confirmed after update")
	}
	return modelID, nil
}

// readOpenFGAAuthorizationBinding rejects incomplete, forged, or superseded
// local bindings. Runtime checks must additionally confirm model freshness
// against the authoritative OpenFGA service before allowing access.
func (r *TenantBundleReconciler) readOpenFGAAuthorizationBinding(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (openFGAAuthorizationBinding, error) {
	if bundle == nil || !bundle.DeletionTimestamp.IsZero() {
		return openFGAAuthorizationBinding{}, errors.New("missing or deleting Fabric tenant")
	}
	store, err := r.readOpenFGAStoreBinding(ctx, bundle)
	if err != nil {
		return openFGAAuthorizationBinding{}, err
	}
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil {
		return openFGAAuthorizationBinding{}, err
	}
	var binding corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{Namespace: openFGAPlatformNamespace, Name: name}, &binding); err != nil {
		return openFGAAuthorizationBinding{}, err
	}
	modelID := binding.Data["modelID"]
	fingerprint := binding.Data["modelFingerprint"]
	approvedFingerprint, err := openfga.ModelFingerprint()
	if err != nil {
		return openFGAAuthorizationBinding{}, err
	}
	if modelID == "" || fingerprint != approvedFingerprint {
		return openFGAAuthorizationBinding{}, errors.New("Fabric OpenFGA authorization model is missing or unapproved")
	}
	return openFGAAuthorizationBinding{
		Store: store, ModelID: modelID, ModelFingerprint: fingerprint,
	}, nil
}
