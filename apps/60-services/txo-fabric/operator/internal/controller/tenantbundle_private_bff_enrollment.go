package controller

import (
	"context"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

// EnrollPrivateBFFIdentity is the narrow, authenticated BFF-to-Fabric entry
// point for first human enrollment. Its caller MUST already have authenticated
// the BFF ServiceAccount, verified the signed OIDC proof server-side, bound all
// proof fields in a one-shot assertion, and consumed that assertion nonce.
//
// This method never accepts a browser identity, creates an Authentik account,
// changes group membership, or grants chat. It only invokes the CAS-protected
// ledger primitive after reloading the canonical TenantBundle and authoritative
// Authentik account reader. Group/OpenFGA chat authorization remains the normal
// reconciliation path after enrollment.
func (r *TenantBundleReconciler) EnrollPrivateBFFIdentity(
	ctx context.Context, tenantKey, issuer, subject, uuid string, now time.Time,
) (string, error) {
	if r == nil || r.Client == nil || now.IsZero() || len(tenantKey) > 63 ||
		issuer == "" || subject == "" || uuid == "" {
		return "", errPrivateIdentityUnavailable
	}
	reader := r.APIReader
	if reader == nil {
		reader = r.Client
	}
	var bundle fabricv1alpha1.TenantBundle
	if err := reader.Get(ctx, types.NamespacedName{Name: tenantKey}, &bundle); err != nil ||
		bundle.Name != tenantKey || !bundle.DeletionTimestamp.IsZero() ||
		bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil {
		return "", errPrivateIdentityUnavailable
	}
	source, err := r.authentikMembershipSource(ctx)
	if err != nil {
		return "", errPrivateIdentityUnavailable
	}
	accounts, ok := source.(authoritativeHumanAccountReader)
	if !ok {
		return "", errPrivateIdentityUnavailable
	}
	id, err := r.enrollHumanFromVerifiedIdentity(ctx, &bundle, trustedHumanOIDCProof{
		globalIssuer: issuer, globalSubject: subject, authentikUUID: uuid,
	}, accounts)
	if err != nil || id == "" {
		return "", errPrivateIdentityUnavailable
	}
	return id, nil
}

// VerifyPrivateBFFActiveHuman is an authenticated BFF preflight only. It
// confirms the immutable Authentik account is active at two reads. It does not
// infer tenant membership, access a ledger, create a binding, or grant chat.
// The caller has verified the signed OIDC identity and bound those fields in a
// one-shot BFF assertion; issuer and subject are deliberately not converted
// into a Fabric identity here.
func (r *TenantBundleReconciler) VerifyPrivateBFFActiveHuman(
	ctx context.Context, issuer, subject, uuid string, now time.Time,
) error {
	if r == nil || r.Client == nil || now.IsZero() || issuer == "" || subject == "" || uuid == "" {
		return errPrivateIdentityUnavailable
	}
	source, err := r.authentikMembershipSource(ctx)
	if err != nil {
		return errPrivateIdentityUnavailable
	}
	accounts, ok := source.(authoritativeHumanAccountReader)
	if !ok {
		return errPrivateIdentityUnavailable
	}
	before, err := accounts.SnapshotUserAccount(ctx, uuid)
	if err != nil || before.UUID != uuid || !before.Active {
		return errPrivateIdentityUnavailable
	}
	after, err := accounts.SnapshotUserAccount(ctx, uuid)
	if err != nil || after.UUID != uuid || !after.Active {
		return errPrivateIdentityUnavailable
	}
	return nil
}
