package controller

import (
	"context"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/privatebridge"
	"k8s.io/apimachinery/pkg/types"
)

// ResolvePrivateResponsesAuthority resolves a single server-requested tenant
// into immutable lifecycle facts for the private Responses authority path.
// It deliberately does not accept a human identity, endpoint, credential,
// OpenFGA store/model, namespace, or runtime address.
func (r *TenantBundleReconciler) ResolvePrivateResponsesAuthority(ctx context.Context, q privatebridge.ResponsesAuthorityRequest, now time.Time) (privatebridge.ResponsesAuthorityResponse, error) {
	if r == nil || r.Client == nil || now.IsZero() || q.Operation != privatebridge.ResponsesAuthorityResolveTenant {
		return privatebridge.ResponsesAuthorityResponse{}, errBrowserAuthorityUnavailable
	}
	reader := r.APIReader
	if reader == nil {
		reader = r.Client
	}
	var bundle fabricv1alpha1.TenantBundle
	if err := reader.Get(ctx, types.NamespacedName{Name: q.TenantKey}, &bundle); err != nil || !browserTenantCurrent(&bundle, now) || bundle.Spec.TenantID != q.FabricTenantID {
		return privatebridge.ResponsesAuthorityResponse{}, errBrowserAuthorityUnavailable
	}
	return privatebridge.ResponsesAuthorityResponse{Tenant: &privatebridge.ResponsesTenantAuthority{
		TenantKey:       bundle.Name,
		FabricTenantID:  bundle.Spec.TenantID,
		TenantBundleUID: string(bundle.UID),
		Status:          "Ready",
	}}, nil
}
