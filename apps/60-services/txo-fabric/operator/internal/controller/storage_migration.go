package controller

import (
	"context"
	"fmt"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// runtimeMigrationTargetPVCName derives a distinct and deterministic target.
// The source identity is never renamed or reused during prepare/copy.
func runtimeMigrationTargetPVCName(agentKey, suffix string) string {
	return runtimePVCName(agentKey) + "-" + suffix
}

func expectedMigrationRWOClass(agent *fabricv1alpha1.AgentIdentity) string {
	if storageRetentionPolicy(agent) == StorageRetentionDelete {
		return "storage-rwo-delete"
	}
	return "storage-rwo-retain"
}

func runtimeActivePVCName(agent *fabricv1alpha1.AgentIdentity) string {
	if migration := agent.Spec.Runtime.Storage.Migration; migration != nil && migration.Cutover {
		return runtimeMigrationTargetPVCName(agent.Spec.AgentKey, migration.TargetClaimSuffix)
	}
	return runtimePVCName(agent.Spec.AgentKey)
}

// ensureRuntimeMigrationTargetPVC implements the non-destructive prepare gate.
// A cutover additionally requires a Bound target plus a GitOps-owned completed
// copy Job whose labels bind the exact source and target PVC UIDs.
func (r *AgentIdentityReconciler) ensureRuntimeMigrationTargetPVC(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle, profile *fabricv1alpha1.AgentRuntimeProfile, namespace string) error {
	migration := agent.Spec.Runtime.Storage.Migration
	if migration == nil {
		return nil
	}
	if migration.TargetStorageClassName != expectedMigrationRWOClass(agent) {
		return fmt.Errorf("AgentIdentity %q migration target storageClass %q does not match required %q", agent.Name, migration.TargetStorageClassName, expectedMigrationRWOClass(agent))
	}

	sourceName := runtimePVCName(agent.Spec.AgentKey)
	targetName := runtimeMigrationTargetPVCName(agent.Spec.AgentKey, migration.TargetClaimSuffix)
	if targetName == sourceName {
		return fmt.Errorf("AgentIdentity %q migration target PVC must differ from source PVC", agent.Name)
	}

	var source corev1.PersistentVolumeClaim
	sourceErr := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: sourceName}, &source)
	if sourceErr != nil && !apierrors.IsNotFound(sourceErr) {
		return fmt.Errorf("read migration source PVC %s/%s: %w", namespace, sourceName, sourceErr)
	}

	key := types.NamespacedName{Namespace: namespace, Name: targetName}
	var target corev1.PersistentVolumeClaim
	err := r.Get(ctx, key, &target)
	if apierrors.IsNotFound(err) {
		if sourceErr != nil {
			return fmt.Errorf("migration target PVC %s/%s is absent and source PVC %s/%s is retired", namespace, targetName, namespace, sourceName)
		}
		labels := agentLabels(agent, tenant)
		labels[LabelStorageRetention] = storageRetentionPolicy(agent)
		labels["fabric.truxonline.io/storage-migration-source-uid"] = string(source.UID)
		target = corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: targetName, Namespace: namespace, Labels: labels},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				StorageClassName: stringPtr(migration.TargetStorageClassName),
				Resources:        corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: profile.Spec.Storage.Size}},
			},
		}
		if err := r.setRuntimePVCOwnership(agent, &target); err != nil {
			return err
		}
		if err := r.Create(ctx, &target); err != nil {
			return err
		}
		if migration.Cutover {
			return fmt.Errorf("migration target PVC %s/%s was created; wait for Bound copy evidence before cutover", namespace, targetName)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if valueOrEmpty(target.Spec.StorageClassName) != migration.TargetStorageClassName {
		return fmt.Errorf("migration target PVC %s/%s uses immutable storageClass %q, expected %q", namespace, targetName, valueOrEmpty(target.Spec.StorageClassName), migration.TargetStorageClassName)
	}
	if migration.Cutover && apierrors.IsNotFound(sourceErr) {
		if target.Status.Phase != corev1.ClaimBound {
			return fmt.Errorf("finalized migration target PVC %s/%s is not Bound", namespace, targetName)
		}
		for key, expected := range agentLabels(agent, tenant) {
			if target.Labels[key] != expected {
				return fmt.Errorf("finalized migration target PVC %s/%s has mismatched %s label", namespace, targetName, key)
			}
		}
		legacyUID := target.Labels["fabric.truxonline.io/storage-migration-source-uid"]
		if legacyUID == "" {
			return fmt.Errorf("finalized migration target PVC %s/%s has no historical source UID", namespace, targetName)
		}
		return r.recordRuntimeMigrationAttestation(ctx, agent, fabricv1alpha1.RuntimeStorageMigrationAttestationStatus{SourceClaimName: sourceName, SourcePVCUID: legacyUID, TargetClaimName: targetName, TargetPVCUID: string(target.UID), CopyJobName: migration.CopyJobName, Phase: "LegacyRecovered", EvidenceSource: "LegacyTargetBinding", VerificationResult: "Succeeded"})
	}
	if sourceErr != nil {
		return fmt.Errorf("read migration source PVC %s/%s: %w", namespace, sourceName, sourceErr)
	}
	if target.Labels["fabric.truxonline.io/storage-migration-source-uid"] != string(source.UID) {
		return fmt.Errorf("migration target PVC %s/%s is not bound to source PVC UID %q", namespace, targetName, source.UID)
	}
	if !migration.Cutover {
		return nil
	}
	if target.Status.Phase != corev1.ClaimBound {
		return fmt.Errorf("migration target PVC %s/%s is not Bound", namespace, targetName)
	}
	if migration.CopyJobName == "" {
		return fmt.Errorf("AgentIdentity %q requests storage migration cutover without copyJobName", agent.Name)
	}
	var job batchv1.Job
	err = r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: migration.CopyJobName}, &job)
	if apierrors.IsNotFound(err) {
		// TTL cleanup is permitted for copy Jobs, but cannot erase proof required
		// by a later reconcile. GitOps retains one UID-bound ConfigMap attestation.
		proofName := migration.CopyJobName + "-proof"
		if len(proofName) > 63 {
			return fmt.Errorf("storage migration copy Job name %q is too long for its durable proof", migration.CopyJobName)
		}
		var proof corev1.ConfigMap
		if proofErr := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: proofName}, &proof); proofErr != nil {
			return fmt.Errorf("read storage migration copy Job %s/%s and durable proof %s/%s: %w", namespace, migration.CopyJobName, namespace, proofName, proofErr)
		}
		labels := proof.GetLabels()
		if labels["fabric.truxonline.io/storage-migration-source-uid"] != string(source.UID) || labels["fabric.truxonline.io/storage-migration-target-uid"] != string(target.UID) || proof.Data["verified"] != "true" {
			return fmt.Errorf("storage migration durable proof %s/%s is not an exact verified source/target UID attestation", namespace, proofName)
		}
		return r.recordRuntimeMigrationAttestation(ctx, agent, fabricv1alpha1.RuntimeStorageMigrationAttestationStatus{SourceClaimName: sourceName, SourcePVCUID: string(source.UID), TargetClaimName: targetName, TargetPVCUID: string(target.UID), CopyJobName: migration.CopyJobName, Phase: "Succeeded", EvidenceSource: "Job", VerificationResult: "Succeeded"})
	}
	if err != nil {
		return fmt.Errorf("read storage migration copy Job %s/%s: %w", namespace, migration.CopyJobName, err)
	}
	labels := job.GetLabels()
	if labels["fabric.truxonline.io/storage-migration-source-uid"] != string(source.UID) || labels["fabric.truxonline.io/storage-migration-target-uid"] != string(target.UID) {
		return fmt.Errorf("storage migration copy Job %s/%s does not bind exact source/target PVC UIDs", namespace, job.Name)
	}
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return r.recordRuntimeMigrationAttestation(ctx, agent, fabricv1alpha1.RuntimeStorageMigrationAttestationStatus{SourceClaimName: sourceName, SourcePVCUID: string(source.UID), TargetClaimName: targetName, TargetPVCUID: string(target.UID), CopyJobName: migration.CopyJobName, CopyJobUID: string(job.UID), Phase: "Succeeded", EvidenceSource: "Job", VerificationResult: "Succeeded"})
		}
	}
	return fmt.Errorf("storage migration copy Job %s/%s has not completed successfully", namespace, job.Name)
}

func (r *AgentIdentityReconciler) recordRuntimeMigrationAttestation(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, want fabricv1alpha1.RuntimeStorageMigrationAttestationStatus) error {
	got := agent.Status.RuntimeStorageMigration
	if got != nil {
		if got.TargetClaimName != want.TargetClaimName || got.TargetPVCUID != want.TargetPVCUID || got.SourceClaimName != want.SourceClaimName || got.SourcePVCUID != want.SourcePVCUID {
			return fmt.Errorf("runtime storage migration attestation conflicts with immutable active volume identity")
		}
		return nil
	}
	agent.Status.RuntimeStorageMigration = &want
	if err := r.Status().Update(ctx, agent); err != nil {
		// Direct unit helpers intentionally use detached objects; a real reconcile
		// always fetched AgentIdentity first, so other errors remain fatal.
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}
