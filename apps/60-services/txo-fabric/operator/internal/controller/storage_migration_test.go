package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
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
		{"cutover-without-copy-evidence", &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{TargetStorageClassName: "storage-rwo-retain", TargetClaimSuffix: "synelia", Cutover: true}, "before verified copy-job"},
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
