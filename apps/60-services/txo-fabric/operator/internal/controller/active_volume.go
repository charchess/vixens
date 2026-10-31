package controller

import (
	"context"
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// validateActiveVolumeBinding is deliberately read-only. A missing or replaced
// finalized claim is an error, never an invitation to create/adopt another PVC.
// The claimRef UID check prevents a recreated PVC from acquiring the same PV.
func validateActiveVolumeBinding(ctx context.Context, reader client.Reader, namespace string, binding *fabricv1alpha1.ActiveVolumeBinding, labels map[string]string, access corev1.PersistentVolumeAccessMode, retention, retentionLabel string) error {
	if binding == nil || binding.ClaimName == "" || binding.PVCUID == "" || binding.VolumeName == "" || binding.StorageClassName == "" {
		return fmt.Errorf("active volume requires exact claim name, PVC UID, PV name and storage class")
	}
	if binding.AccessMode != string(access) || binding.RetentionPolicy != retention {
		return fmt.Errorf("active volume %s has access/retention policy different from its owning workload", binding.ClaimName)
	}
	var pvc corev1.PersistentVolumeClaim
	if err := reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: binding.ClaimName}, &pvc); err != nil {
		return fmt.Errorf("read pinned active PVC %s/%s: %w", namespace, binding.ClaimName, err)
	}
	if pvc.UID == "" || string(pvc.UID) != binding.PVCUID || !pvc.DeletionTimestamp.IsZero() || pvc.Status.Phase != corev1.ClaimBound {
		return fmt.Errorf("active PVC %s/%s UID/Bound/lifecycle no longer matches the pinned identity", namespace, pvc.Name)
	}
	if pvc.Spec.VolumeName != binding.VolumeName || valueOrEmpty(pvc.Spec.StorageClassName) != binding.StorageClassName ||
		len(pvc.Spec.AccessModes) != 1 || pvc.Spec.AccessModes[0] != access {
		return fmt.Errorf("active PVC %s/%s has mismatched PV, storage class or access mode", namespace, pvc.Name)
	}
	for k, v := range labels {
		if pvc.Labels[k] != v {
			return fmt.Errorf("active PVC %s/%s has mismatched owner/scope label %q", namespace, pvc.Name, k)
		}
	}
	if pvc.Labels[retentionLabel] != retention {
		return fmt.Errorf("active PVC %s/%s has mismatched retention label", namespace, pvc.Name)
	}
	var pv corev1.PersistentVolume
	if err := reader.Get(ctx, types.NamespacedName{Name: binding.VolumeName}, &pv); err != nil {
		return fmt.Errorf("read pinned active PV %q: %w", binding.VolumeName, err)
	}
	if pv.Spec.ClaimRef == nil || pv.Spec.ClaimRef.Namespace != namespace ||
		pv.Spec.ClaimRef.Name != binding.ClaimName || string(pv.Spec.ClaimRef.UID) != binding.PVCUID ||
		pv.Status.Phase != corev1.VolumeBound {
		return fmt.Errorf("active PV %q is not Bound to the exact pinned PVC UID", pv.Name)
	}
	wantReclaim := corev1.PersistentVolumeReclaimRetain
	if retention == StorageRetentionDelete {
		wantReclaim = corev1.PersistentVolumeReclaimDelete
	}
	if pv.Spec.PersistentVolumeReclaimPolicy != wantReclaim {
		return fmt.Errorf("active PV %q reclaim policy %q differs from pinned retention %q", pv.Name, pv.Spec.PersistentVolumeReclaimPolicy, retention)
	}
	return nil
}

func (r *AgentIdentityReconciler) ensureRuntimeActiveVolume(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, namespace string) error {
	binding := agent.Spec.Runtime.Storage.ActiveVolume
	if binding == nil {
		return nil
	}
	if m := agent.Spec.Runtime.Storage.Migration; m != nil && !m.Cutover {
		return fmt.Errorf("AgentIdentity %q cannot finalize active volume while migration is still in preparation", agent.Name)
	}
	labels := agentLabels(agent, tenant)
	labels[LabelStorageRetention] = storageRetentionPolicy(agent)
	reader := client.Reader(r.Client)
	if r.APIReader != nil {
		reader = r.APIReader
	}
	if err := validateActiveVolumeBinding(ctx, reader, namespace, binding, labels, corev1.ReadWriteOnce, storageRetentionPolicy(agent), LabelStorageRetention); err != nil {
		return err
	}
	var pvc corev1.PersistentVolumeClaim
	if err := reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: binding.ClaimName}, &pvc); err != nil {
		return err
	}
	owner := pvc.GetOwnerReferences()
	for _, ref := range owner {
		if ref.Controller != nil && *ref.Controller {
			if storageRetentionPolicy(agent) != StorageRetentionDelete || ref.Kind != "AgentIdentity" || ref.Name != agent.Name || ref.UID != agent.UID {
				return fmt.Errorf("active PVC %s/%s has a foreign or unexpected controller owner", namespace, pvc.Name)
			}
		}
	}
	return nil
}
