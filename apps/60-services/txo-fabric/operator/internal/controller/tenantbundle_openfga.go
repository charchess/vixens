package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	openFGAPlatformNamespace = "txo-fabric-system"
	openFGACredentialSecret = "txo-openfga-runtime"
	openFGABindingPrefix = "txo-openfga-tenant-"
)

// openFGAStoreProvisioner lets the operator be tested without a Kubernetes
// cluster or access to the OpenBao-managed service token.
type openFGAStoreProvisioner interface {
	EnsureTenantStore(context.Context, string) (openfga.Store, error)
}

func openFGABindingName(tenantID string) (string, error) {
	name, err := openfga.TenantStoreName(tenantID)
	if err != nil { return "", err }
	return strings.Replace(name, "txo-fabric-tenant-", openFGABindingPrefix, 1), nil
}

// reconcileOpenFGAStore is the next unit of TenantBundle reconciliation.
// The public API, browser and tenant pods must NEVER control the store ID.
//
// This only ensures a *store and its durable association*. It never grants a
// permission or claims the model/tuple graph is ready for end-user use.
func (r *TenantBundleReconciler) reconcileOpenFGAStore(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (openfga.Store, error) {
	if bundle == nil || !bundle.DeletionTimestamp.IsZero() {
		return openfga.Store{}, errors.New("cannot provision authorization for missing or deleting TenantBundle")
	}
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil { return openfga.Store{}, err }

	// An immutable business ID must never be shared between two live objects.
	// Namespaces and group names alone cannot provide this identity guarantee.
	var bundles fabricv1alpha1.TenantBundleList
	if err := r.List(ctx, &bundles); err != nil { return openfga.Store{}, err }
	for i := range bundles.Items {
		candidate := &bundles.Items[i]
		if candidate.Name != bundle.Name && candidate.Spec.TenantID == bundle.Spec.TenantID {
			return openfga.Store{}, errors.New("multiple TenantBundles claim an immutable tenant ID")
		}
	}

	key := types.NamespacedName{Namespace: openFGAPlatformNamespace, Name: name}
	var binding corev1.ConfigMap
	found := true
	if err := r.Get(ctx, key, &binding); err != nil {
		if !apierrors.IsNotFound(err) { return openfga.Store{}, err }
		found = false
	}
	if found {
		// Kept deliberately across tenant deletion. Re-creation with a new UID
		// must require an explicit reviewed adoption rather than inheriting
		// previous customer grants or durable authorization state.
		if binding.Data["tenantID"] != bundle.Spec.TenantID ||
			binding.Data["tenantName"] != bundle.Name ||
			binding.Data["tenantUID"] != string(bundle.UID) ||
			binding.Data["storeID"] == "" {
			return openfga.Store{}, errors.New("retained Fabric authorization store binding has conflicting identity")
		}
	}

	service := r.OpenFGAStoreClient
	if service == nil {
		var credential corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: openFGAPlatformNamespace, Name: openFGACredentialSecret,
		}, &credential); err != nil {
			return openfga.Store{}, fmt.Errorf("Fabric OpenFGA service credential is not ready: %w", err)
		}
		service, err = openfga.NewClient(string(credential.Data["presharedKeys"]), nil)
		if err != nil {
			return openfga.Store{}, errors.New("Fabric OpenFGA service credential is invalid")
		}
	}

	store, err := service.EnsureTenantStore(ctx, bundle.Spec.TenantID)
	if err != nil { return openfga.Store{}, fmt.Errorf("Fabric OpenFGA store reconciliation failed: %w", err) }
	expected, _ := openfga.TenantStoreName(bundle.Spec.TenantID)
	if store.Name != expected || store.ID == "" {
		return openfga.Store{}, errors.New("Fabric OpenFGA store result does not match tenant identity")
	}
	if found {
		if binding.Data["storeID"] != store.ID {
			return openfga.Store{}, errors.New("Fabric OpenFGA store binding drift: refusing rebind")
		}
		return store, nil
	}

	newBinding := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: openFGAPlatformNamespace,
			Name: name,
			Labels: map[string]string{
				LabelPartOf: "txo-fabric",
				LabelManaged: "true",
				LabelTenantID: bundle.Spec.TenantID,
				LabelTenantName: bundle.Name,
				"app.kubernetes.io/component": "tenant-authorization-binding",
			},
		},
		Data: map[string]string{
			"tenantID": bundle.Spec.TenantID,
			"tenantName": bundle.Name,
			"tenantUID": string(bundle.UID),
			"storeID": store.ID,
		},
	}
	// Atomic create protects the binding against racing operators. A
	// conflicting AlreadyExists must not overwrite the other identity.
	if err := r.Create(ctx, newBinding); err != nil {
		if !apierrors.IsAlreadyExists(err) { return openfga.Store{}, err }
		var existing corev1.ConfigMap
		if err := r.Get(ctx, key, &existing); err != nil { return openfga.Store{}, err }
		if existing.Data["tenantID"] != bundle.Spec.TenantID ||
			existing.Data["tenantName"] != bundle.Name ||
			existing.Data["tenantUID"] != string(bundle.UID) ||
			existing.Data["storeID"] != store.ID {
			return openfga.Store{}, errors.New("conflicting concurrent Fabric authorization store binding")
		}
	}
	return store, nil
}

// OpenFGAStoreBinding is backend-owned durable state; exposing this ConfigMap
// to tenant workloads would bypass store isolation. No FGA secret lives here.
func (r *TenantBundleReconciler) readOpenFGAStoreBinding(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (openfga.Store, error) {
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil { return openfga.Store{}, err }
	var binding corev1.ConfigMap
	if err := r.Get(ctx, client.ObjectKey{Namespace: openFGAPlatformNamespace, Name: name}, &binding); err != nil {
		return openfga.Store{}, err
	}
	if binding.Data["tenantID"] != bundle.Spec.TenantID ||
		binding.Data["tenantName"] != bundle.Name ||
		binding.Data["tenantUID"] != string(bundle.UID) ||
		binding.Data["storeID"] == "" {
		return openfga.Store{}, errors.New("untrusted or stale Fabric authorization store binding")
	}
	storeName, _ := openfga.TenantStoreName(bundle.Spec.TenantID)
	return openfga.Store{ID: binding.Data["storeID"], Name: storeName}, nil
}
