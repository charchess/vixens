package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestObserveWorkspaceMigrationAttestationsPersistsOnlyExactCompletedJob(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := workspaceTestTenant()
	tenant.Spec.Workspace.Migration = &fabricv1alpha1.TenantWorkspaceMigrationBinding{TargetClaimSuffix: "synelia", CopyJobPrefix: "copy"}
	ns := tenantNamespace(tenant.Name)
	source := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "ws-group-sales-rw", Namespace: ns, UID: types.UID("source-uid")}}
	target := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "ws-group-sales-rw-synelia", Namespace: ns, UID: types.UID("target-uid")}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound}}
	done := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "copy-ws-group-sales-rw", Namespace: ns, UID: types.UID("job-uid"), Labels: map[string]string{"fabric.truxonline.io/storage-migration-source-uid": "source-uid", "fabric.truxonline.io/storage-migration-target-uid": "target-uid"}}, Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}}}
	bad := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "copy-ws-group-tech-ref", Namespace: ns, UID: types.UID("bad-job"), Labels: map[string]string{"fabric.truxonline.io/storage-migration-source-uid": "wrong"}}, Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}}}
	r := &TenantBundleReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(source, target, done, bad).Build(), Scheme: scheme}
	if err := r.observeWorkspaceMigrationAttestations(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	if len(tenant.Status.WorkspaceMigration) != 1 {
		t.Fatalf("attestations=%#v", tenant.Status.WorkspaceMigration)
	}
	got := tenant.Status.WorkspaceMigration[0]
	if got.SourcePVCUID != "source-uid" || got.TargetPVCUID != "target-uid" || got.JobUID != "job-uid" || got.Phase != fabricv1alpha1.WorkspaceMigrationSucceeded {
		t.Fatalf("attestation=%#v", got)
	}
}
