package controller

import (
	"context"
	"fmt"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// observeWorkspaceMigrationAttestations persists only a completed Job with exact live PVC UID bindings.
func (r *TenantBundleReconciler) observeWorkspaceMigrationAttestations(ctx context.Context, bundle *fabricv1alpha1.TenantBundle) error {
	w := bundle.Spec.Workspace
	if w == nil || w.Migration == nil || w.Migration.CopyJobPrefix == "" {
		return nil
	}
	ns := tenantNamespace(bundle.Name)
	var jobs batchv1.JobList
	if err := r.List(ctx, &jobs, client.InNamespace(ns)); err != nil {
		return fmt.Errorf("list workspace copy Jobs: %w", err)
	}
	prefix := w.Migration.CopyJobPrefix + "-"
	seen := map[string]int{}
	for i := range bundle.Status.WorkspaceMigration {
		seen[bundle.Status.WorkspaceMigration[i].SourceClaimName] = i
	}
	for i := range jobs.Items {
		job := &jobs.Items[i]
		if !strings.HasPrefix(job.Name, prefix) || !workspaceMigrationJobComplete(job) {
			continue
		}
		sourceName := strings.TrimPrefix(job.Name, prefix)
		targetName := sourceName + "-" + w.Migration.TargetClaimSuffix
		var source, target corev1.PersistentVolumeClaim
		if r.Get(ctx, client.ObjectKey{Namespace: ns, Name: sourceName}, &source) != nil || r.Get(ctx, client.ObjectKey{Namespace: ns, Name: targetName}, &target) != nil {
			continue
		}
		if source.UID == "" || target.UID == "" || target.Status.Phase != corev1.ClaimBound || job.UID == "" || job.Labels["fabric.truxonline.io/storage-migration-source-uid"] != string(source.UID) || job.Labels["fabric.truxonline.io/storage-migration-target-uid"] != string(target.UID) {
			continue
		}
		e := fabricv1alpha1.WorkspaceMigrationAttestationStatus{SourceClaimName: sourceName, SourcePVCUID: string(source.UID), TargetClaimName: targetName, TargetPVCUID: string(target.UID), JobName: job.Name, JobUID: string(job.UID), Phase: fabricv1alpha1.WorkspaceMigrationSucceeded, VerificationResult: "Succeeded"}
		if n, ok := seen[sourceName]; ok {
			bundle.Status.WorkspaceMigration[n] = e
		} else {
			bundle.Status.WorkspaceMigration = append(bundle.Status.WorkspaceMigration, e)
			seen[sourceName] = len(bundle.Status.WorkspaceMigration) - 1
		}
	}
	return nil
}
func workspaceMigrationJobComplete(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
