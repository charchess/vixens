package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRuntimeMigrationPrepareCreatesDistinctCanonicalTarget(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "canary", StorageRetentionRetain)
	agent.Spec.Runtime.Storage.Migration = &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{
		TargetStorageClassName: "storage-rwo-retain",
		TargetClaimSuffix:      "synelia",
	}
	profile := testRuntimeProfile()
	ns := tenantNamespace(tenant.Name)
	source := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns, UID: types.UID("source-uid")}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(source).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	if err := r.ensureRuntimeMigrationTargetPVC(ctx, agent, tenant, profile, ns); err != nil {
		t.Fatal(err)
	}
	var target corev1.PersistentVolumeClaim
	key := types.NamespacedName{Namespace: ns, Name: runtimeMigrationTargetPVCName(agent.Spec.AgentKey, "synelia")}
	if err := c.Get(ctx, key, &target); err != nil {
		t.Fatal(err)
	}
	if target.Name == source.Name || target.Spec.StorageClassName == nil || *target.Spec.StorageClassName != "storage-rwo-retain" {
		t.Fatalf("unexpected target: %#v", target)
	}
	if target.Labels["fabric.truxonline.io/storage-migration-source-uid"] != "source-uid" {
		t.Fatalf("target does not bind source UID: %#v", target.Labels)
	}
	if owner := metav1.GetControllerOf(&target); owner != nil {
		t.Fatalf("retained target must not carry GC owner: %#v", owner)
	}
}

func TestRuntimeMigrationCutoverRequiresVerifiedJobAndSelectsTarget(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "canary", StorageRetentionRetain)
	agent.Spec.Runtime.Storage.Migration = &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{TargetStorageClassName: "storage-rwo-retain", TargetClaimSuffix: "synelia", CopyJobName: "copy-canary", Cutover: true}
	profile := testRuntimeProfile()
	ns := tenantNamespace(tenant.Name)
	source := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns, UID: types.UID("source-uid")}}
	target := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimeMigrationTargetPVCName(agent.Spec.AgentKey, "synelia"), Namespace: ns, UID: types.UID("target-uid"), Labels: map[string]string{"fabric.truxonline.io/storage-migration-source-uid": "source-uid"}}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound}, Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: stringPtr("storage-rwo-retain")}}
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "copy-canary", Namespace: ns, Labels: map[string]string{"fabric.truxonline.io/storage-migration-source-uid": "source-uid", "fabric.truxonline.io/storage-migration-target-uid": "target-uid"}}, Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(source, target, job).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := r.ensureRuntimeMigrationTargetPVC(ctx, agent, tenant, profile, ns); err != nil {
		t.Fatal(err)
	}
	if got := runtimeActivePVCName(agent); got != target.Name {
		t.Fatalf("cutover claim=%q, want %q", got, target.Name)
	}
	if err := r.ensureDeploymentRuntime(ctx, agent, tenant, profile, ns, "", "", effectiveToolsetPolicy{}, integrationResolution{}); err != nil {
		t.Fatalf("reconcile cutover deployment: %v", err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: runtimeName(agent.Spec.AgentKey)}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := deployment.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName; got != target.Name {
		t.Fatalf("deployment data claim=%q, want migrated %q", got, target.Name)
	}

	job.Labels["fabric.truxonline.io/storage-migration-target-uid"] = "wrong"
	if err := c.Update(ctx, job); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureRuntimeMigrationTargetPVC(ctx, agent, tenant, profile, ns); err == nil || !strings.Contains(err.Error(), "exact source/target PVC UIDs") {
		t.Fatalf("UID mismatch accepted: %v", err)
	}
}

func TestRuntimeMigrationRejectsCutoverWithoutCopyJobName(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "canary", StorageRetentionRetain)
	agent.Spec.Runtime.Storage.Migration = &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{
		TargetStorageClassName: "storage-rwo-retain", TargetClaimSuffix: "synelia", Cutover: true,
	}
	profile := testRuntimeProfile()
	ns := tenantNamespace(tenant.Name)
	source := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns, UID: types.UID("source-uid")}}
	target := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimeMigrationTargetPVCName(agent.Spec.AgentKey, "synelia"), Namespace: ns, UID: types.UID("target-uid"), Labels: map[string]string{"fabric.truxonline.io/storage-migration-source-uid": "source-uid"}}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound}, Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: stringPtr("storage-rwo-retain")}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(source, target).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	err := r.ensureRuntimeMigrationTargetPVC(ctx, agent, tenant, profile, ns)
	if err == nil || !strings.Contains(err.Error(), "without copyJobName") {
		t.Fatalf("cutover without copy job was accepted: %v", err)
	}
}

func TestRuntimeMigrationPrepareFailsClosed(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "canary", StorageRetentionRetain)
	profile := testRuntimeProfile()
	ns := tenantNamespace(tenant.Name)
	source := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns, UID: types.UID("source-uid")}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(source).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	cases := []struct {
		name      string
		migration *fabricv1alpha1.AgentRuntimeStorageMigrationBinding
		contains  string
	}{
		{"wrong-class", &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{TargetStorageClassName: "storage-rwo-delete", TargetClaimSuffix: "synelia"}, "does not match required"},
		{"cutover-before-target-bound", &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{TargetStorageClassName: "storage-rwo-retain", TargetClaimSuffix: "synelia", Cutover: true}, "was created; wait for Bound"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agent.Spec.Runtime.Storage.Migration = tc.migration
			err := r.ensureRuntimeMigrationTargetPVC(ctx, agent, tenant, profile, ns)
			if err == nil || !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("got %v, expected %q", err, tc.contains)
			}
		})
	}
}

func TestRuntimeMigrationCutoverAcceptsExactDurableProofAfterJobTTL(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "canary", StorageRetentionRetain)
	agent.Spec.Runtime.Storage.Migration = &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{TargetStorageClassName: "storage-rwo-retain", TargetClaimSuffix: "synelia", CopyJobName: "copy-canary", Cutover: true}
	profile := testRuntimeProfile()
	ns := tenantNamespace(tenant.Name)
	source := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns, UID: types.UID("source-uid")}}
	target := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimeMigrationTargetPVCName(agent.Spec.AgentKey, "synelia"), Namespace: ns, UID: types.UID("target-uid"), Labels: map[string]string{"fabric.truxonline.io/storage-migration-source-uid": "source-uid"}}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound}, Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: stringPtr("storage-rwo-retain")}}
	proof := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "copy-canary-proof", Namespace: ns, Labels: map[string]string{"fabric.truxonline.io/storage-migration-source-uid": "source-uid", "fabric.truxonline.io/storage-migration-target-uid": "target-uid"}}, Data: map[string]string{"verified": "true"}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(source, target, proof).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := r.ensureRuntimeMigrationTargetPVC(ctx, agent, tenant, profile, ns); err != nil {
		t.Fatalf("durable exact proof should permit cutover after Job TTL: %v", err)
	}
	proof.Data["verified"] = "false"
	if err := c.Update(ctx, proof); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureRuntimeMigrationTargetPVC(ctx, agent, tenant, profile, ns); err == nil || !strings.Contains(err.Error(), "durable proof") {
		t.Fatalf("unverified durable proof was accepted: %v", err)
	}
}

func TestRuntimeMigrationLegacyRecoveryPersistsTargetIdentity(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "legacy-recovery", StorageRetentionRetain)
	agent.Spec.Runtime.Storage.Migration = &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{TargetStorageClassName: "storage-rwo-retain", TargetClaimSuffix: "synelia", CopyJobName: "expired-copy", Cutover: true}
	profile := testRuntimeProfile()
	ns := tenantNamespace(tenant.Name)
	labels := agentLabels(agent, tenant)
	labels["fabric.truxonline.io/storage-migration-source-uid"] = "retired-source-uid"
	target := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimeMigrationTargetPVCName(agent.Spec.AgentKey, "synelia"), Namespace: ns, UID: types.UID("canonical-target-uid"), Labels: labels}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound}, Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: stringPtr("storage-rwo-retain")}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(agent).WithObjects(agent, target).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := r.ensureRuntimeMigrationTargetPVC(ctx, agent, tenant, profile, ns); err != nil {
		t.Fatalf("legacy recovery rejected: %v", err)
	}
	var stored fabricv1alpha1.AgentIdentity
	if err := c.Get(ctx, types.NamespacedName{Name: agent.Name}, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Status.RuntimeStorageMigration == nil || stored.Status.RuntimeStorageMigration.TargetPVCUID != string(target.UID) || stored.Status.RuntimeStorageMigration.Phase != "LegacyRecovered" {
		t.Fatalf("missing exact legacy attestation: %#v", stored.Status.RuntimeStorageMigration)
	}
	if err := r.ensureRuntimeMigrationTargetPVC(ctx, &stored, tenant, profile, ns); err != nil {
		t.Fatalf("repeated recovery was not idempotent: %v", err)
	}
}

func TestRuntimeMigrationPendingHistoricalSourceRecoversCanonicalTarget(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "pending-source", StorageRetentionRetain)
	agent.Spec.Runtime.Storage.Migration = &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{TargetStorageClassName: "storage-rwo-retain", TargetClaimSuffix: "synelia", CopyJobName: "expired-copy", Cutover: true}
	profile := testRuntimeProfile()
	ns := tenantNamespace(tenant.Name)
	labels := agentLabels(agent, tenant)
	labels["fabric.truxonline.io/storage-migration-source-uid"] = "historical-source-uid"
	target := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimeMigrationTargetPVCName(agent.Spec.AgentKey, "synelia"), Namespace: ns, UID: types.UID("canonical-target-uid"), Labels: labels}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound}, Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: stringPtr("storage-rwo-retain")}}
	pending := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns, UID: types.UID("new-empty-source-uid")}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(agent).WithObjects(agent, target, pending).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := r.ensureRuntimeMigrationTargetPVC(ctx, agent, tenant, profile, ns); err != nil {
		t.Fatalf("Pending historical artifact rejected canonical target: %v", err)
	}
	var live corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: target.Name}, &live); err != nil || live.UID != target.UID {
		t.Fatalf("canonical target changed: %#v %v", live, err)
	}
	var source corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: pending.Name}, &source); err != nil || source.UID != pending.UID {
		t.Fatalf("Pending artifact was mutated: %#v %v", source, err)
	}
}
