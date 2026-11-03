package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// A missing tenant-owned storage policy is never permission to provision a new
// claim from a mutable Hermes runtime profile, even for an old stored TenantBundle.
func TestMissingTenantStoragePolicyCannotProvisionNewAgentPVC(t *testing.T) {
	for _, retention := range []string{StorageRetentionRetain, StorageRetentionDelete} {
		t.Run(retention, func(t *testing.T) {
			ctx := context.Background()
			tenant := testTenant()
			tenant.Spec.AgentStorage = nil
			agent := testStorageIdentity(tenant.Name, "fresh-agent", retention)
			profile := testRuntimeProfile()
			profile.Spec.Storage.StorageClassName = "profile-only-class"
			namespace := tenantNamespace(tenant.Name)
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()
			r := &AgentIdentityReconciler{Client: c, Scheme: testScheme(t)}

			for attempt := 0; attempt < 2; attempt++ {
				err := r.ensurePVC(ctx, agent, tenant, profile, namespace)
				if err == nil || !strings.Contains(err.Error(), "requires spec.agentStorage") {
					t.Fatalf("attempt %d: missing policy must fail closed, got %v", attempt, err)
				}
				var claim corev1.PersistentVolumeClaim
				key := types.NamespacedName{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: namespace}
				if err := c.Get(ctx, key, &claim); !apierrors.IsNotFound(err) {
					t.Fatalf("attempt %d: missing policy created a PVC or caused a read error: %v", attempt, err)
				}
			}
		})
	}
}

// Preserve stored-CR compatibility without allowing it to create replacement
// volumes. A verified existing claim remains the one and only adopted identity.
func TestMissingTenantStoragePolicyKeepsExistingAgentPVC(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	tenant.Spec.AgentStorage = nil
	agent := testStorageIdentity(tenant.Name, "existing-agent", StorageRetentionRetain)
	profile := testRuntimeProfile()
	namespace := tenantNamespace(tenant.Name)
	name := runtimePVCName(agent.Spec.AgentKey)
	uid := types.UID("preexisting-pvc")
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: uid},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr(profile.Spec.Storage.StorageClassName),
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}
	pvc.Labels = agentLabels(agent, tenant)
	pvc.Labels[LabelStorageRetention] = StorageRetentionRetain
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	key := types.NamespacedName{Namespace: namespace, Name: name}

	for attempt := 0; attempt < 2; attempt++ {
		if err := r.ensurePVC(ctx, agent, tenant, profile, namespace); err != nil {
			t.Fatalf("attempt %d: existing verified claim should remain valid: %v", attempt, err)
		}
		var current corev1.PersistentVolumeClaim
		if err := c.Get(ctx, key, &current); err != nil {
			t.Fatal(err)
		}
		if current.UID != uid || current.Spec.StorageClassName == nil ||
			*current.Spec.StorageClassName != profile.Spec.Storage.StorageClassName {
			t.Fatalf("attempt %d: retained claim identity/class mutated: %#v", attempt, current)
		}
	}
}

// Compile-time fixture shape: this compatibility contract applies to arbitrary
// tenants; it is not gated by the hAIrem or Indiba identity.
var _ *fabricv1alpha1.TenantBundle
