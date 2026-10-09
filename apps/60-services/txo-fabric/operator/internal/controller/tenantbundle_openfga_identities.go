package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// These fields live on the retained, platform-private tenant/store/model
	// binding, not on the TenantBundle spec/status or a tenant-owned Secret.
	identityIssuerKey = "identityIssuer"
	identityBindingsKey = "identityBindings"
	identitySchemaKey = "identitySchema"
	maxIdentityBindingJSONBytes = 1024 * 1024
)

// reconcileOpenFGAHumanIdentityRegistry bootstraps the EMPTY identity ledger
// exactly once after the tenant's trusted store and model have been persisted.
// A human identity is NOT enrolled by the presence of a workspace user,
// login, OIDC group claim, or Authentik group membership.
//
// This does not write FGA grants, validate OIDC tokens or infer a sub/UUID
// mapping. A future trusted Fabric enrollment service must verify the actual
// OIDC subject and the Authentik UUID before CAS-updating the retained ledger.
func (r *TenantBundleReconciler) reconcileOpenFGAHumanIdentityRegistry(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) error {
	if bundle == nil || !bundle.DeletionTimestamp.IsZero() {
		return errors.New("missing or deleting Fabric tenant identity registry")
	}
	if bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil {
		return nil // no human OIDC surface; NEVER derive users from workspaces
	}
	if err := validateTenantIAM(bundle); err != nil {
		return fmt.Errorf("invalid Fabric IAM tenant identity scope: %w", err)
	}
	if _, err := r.readOpenFGAAuthorizationBinding(ctx, bundle); err != nil {
		return fmt.Errorf("untrusted Fabric OpenFGA binding for human identity ledger: %w", err)
	}
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil { return err }
	var binding corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{Namespace:openFGAPlatformNamespace, Name:name}, &binding); err != nil {
		return err
	}
	issuer := bundle.Spec.HumanAccess.Web.OIDC.Issuer
	// Either the complete pair is present or neither was ever initialized.
	// A partially written/damaged ledger is never accepted or repaired.
	pinnedIssuer, issuerExists := binding.Data[identityIssuerKey]
	_, bindingsExist := binding.Data[identityBindingsKey]
	schema, schemaExists := binding.Data[identitySchemaKey]
	if issuerExists || bindingsExist || schemaExists {
		if !issuerExists || !bindingsExist || !schemaExists || schema != "v1" {
			return errors.New("partial or unapproved retained Fabric human identity ledger")
		}
		if pinnedIssuer != issuer {
			return errors.New("retained Fabric OIDC issuer drift: explicit identity migration required")
		}
		_, err := r.readOpenFGAHumanIdentityRegistry(ctx, bundle)
		return err
	}
	// Insertion into the *existing* tenant-bound ConfigMap is a
	// resourceVersion-checked update. Model/store/UID remain unmodified.
	// [] is a valid EMPTY enrolled roster but NEVER a complete Authentik
	// membership snapshot, and NEVER proof that tuple sync is complete.
	if _, err := authentik.NewIdentityRegistry(bundle.Name, bundle.Spec.TenantID,
		issuer, bundle.Spec.HumanAccess.Web.IAMGroups, nil); err != nil {
		return errors.New("invalid Fabric human identity scope")
	}
	if binding.Data == nil { return errors.New("untrusted empty Fabric authorization binding") }
	binding.Data[identityIssuerKey] = issuer
	binding.Data[identityBindingsKey] = "[]"
	binding.Data[identitySchemaKey] = "v1"
	if err := r.Update(ctx, &binding); err != nil {
		return fmt.Errorf("cannot initialize retained Fabric identity ledger: %w", err)
	}
	_, err = r.readOpenFGAHumanIdentityRegistry(ctx, bundle)
	return err
}

// readOpenFGAHumanIdentityRegistry is platform-only. The retained TenantBundle
// UID + tenantID + store/model fingerprint must be trusted on EVERY read.
// Unknown/duplicated subject, Authentik UUID, Fabric ID, malformed JSON,
// mismatched issuer and missing binding all fail closed.
//
// Never use the presence of a nonempty registry as authorization readiness:
// freshness, trusted enrollment, IAM memberships, tuple sync and revocation
// are independently required and still pending under #3996.
func (r *TenantBundleReconciler) readOpenFGAHumanIdentityRegistry(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (*authentik.IdentityRegistry, error) {
	if bundle == nil || !bundle.DeletionTimestamp.IsZero() ||
		bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil {
		return nil, errors.New("Fabric human identity registry is not available")
	}
	if err := validateTenantIAM(bundle); err != nil { return nil, err }
	approved, err := r.readOpenFGAAuthorizationBinding(ctx, bundle)
	if err != nil {
		return nil, fmt.Errorf("invalid Fabric identity registry tenant/store/model ownership: %w", err)
	}
	// A tampered local ConfigMap store/model association must not be enough
	// to authorize a human mapping. Re-check the model against the service:
	// a foreign tenant store or OpenFGA outage denies the entire read.
	service, err := r.modelProvisioner(ctx)
	if err != nil { return nil, errors.New("Fabric human identity model verifier unavailable") }
	if err := service.ValidateModelBinding(ctx, approved.Store.ID, approved.ModelID); err != nil {
		return nil, fmt.Errorf("Fabric human identity ledger model binding not verified: %w", err)
	}
	name, err := openFGABindingName(bundle.Spec.TenantID)
	if err != nil { return nil, err }
	var binding corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{Namespace:openFGAPlatformNamespace, Name:name}, &binding); err != nil {
		return nil, err
	}
	issuer := bundle.Spec.HumanAccess.Web.OIDC.Issuer
	if binding.Data[identitySchemaKey] != "v1" || binding.Data[identityIssuerKey] != issuer {
		return nil, errors.New("missing, partial or drifting Fabric identity issuer/schema")
	}
	raw, exists := binding.Data[identityBindingsKey]
	if !exists || len(raw) == 0 || len(raw) > maxIdentityBindingJSONBytes {
		return nil, errors.New("missing or oversized Fabric identity binding ledger")
	}
	var identities []authentik.IdentityBinding
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&identities); err != nil || identities == nil {
		return nil, errors.New("malformed Fabric identity ledger")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, errors.New("Fabric identity ledger contains trailing data")
	}
	return authentik.NewIdentityRegistry(bundle.Name,bundle.Spec.TenantID,issuer,
		bundle.Spec.HumanAccess.Web.IAMGroups,identities)
}
