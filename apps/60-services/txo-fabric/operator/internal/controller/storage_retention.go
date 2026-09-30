package controller

import (
	"context"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *TenantBundleReconciler) retainedAgentPVCCount(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) (int, error) {
	var pvcs corev1.PersistentVolumeClaimList
	if err := r.List(
		ctx,
		&pvcs,
		client.InNamespace(tenantNamespace(bundle.Name)),
		client.MatchingLabels{
			LabelPartOf:           "txo-fabric",
			LabelTenantName:       bundle.Name,
			LabelStorageRetention: StorageRetentionRetain,
		},
	); err != nil {
		return 0, err
	}
	return len(pvcs.Items), nil
}
