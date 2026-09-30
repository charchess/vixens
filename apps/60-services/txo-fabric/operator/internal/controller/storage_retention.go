package controller

import (
	"context"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *TenantBundleReconciler) retainedFabricPVCCount(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (int, error) {
	var pvcs corev1.PersistentVolumeClaimList
	if err := r.List(
		ctx,
		&pvcs,
		client.InNamespace(tenantNamespace(bundle.Name)),
		client.MatchingLabels{
			LabelPartOf:     "txo-fabric",
			LabelTenantName: bundle.Name,
		},
	); err != nil {
		return 0, err
	}

	retained := 0
	for i := range pvcs.Items {
		labels := pvcs.Items[i].Labels
		if labels[LabelWorkspaceScope] != "" {
			if labels[LabelWorkspaceRetention] == fabricv1alpha1.WorkspaceRetentionRetain {
				retained++
			}
			continue
		}
		if labels[LabelStorageRetention] == StorageRetentionRetain {
			retained++
		}
	}
	return retained, nil
}

// retainedAgentPVCCount is kept as the tenant teardown call-site contract while
// retained Fabric storage now includes both agent-private and shared workspace PVCs.
func (r *TenantBundleReconciler) retainedAgentPVCCount(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (int, error) {
	return r.retainedFabricPVCCount(ctx, bundle)
}
