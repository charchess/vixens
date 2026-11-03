package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// A stored tenant with no agentStorage policy must not follow a mutable Hermes
// runtime-profile class after its already-Bound Retain PVC was created.
// Exact PV claimRef, ownership and immutable class remain independently checked.
func TestBoundRetainPVCWithoutTenantPolicySurvivesProfileClassChange(t *testing.T) {
	for _, tenantName := range []string{"generic-tenant-alpha", "generic-tenant-beta"} {
		t.Run(tenantName, func(t *testing.T) {
			ctx := context.Background()
			scheme := testScheme(t)
			tenant := testTenant()
			tenant.Name = tenantName
			tenant.Spec.TenantID = "TEN-" + tenantName
			tenant.Spec.AgentStorage = nil
			agent := testStorageIdentity(tenant.Name, "agent", StorageRetentionRetain)
			profile := testRuntimeProfile()
			profile.Spec.Storage.StorageClassName = "runtime-profile-new-class"
			ns := tenantNamespace(tenant.Name)
			name := runtimePVCName(agent.Spec.AgentKey)
			uid := types.UID("established-" + tenantName)
			pvName := "pv-" + tenantName
			originalClass := "old-immutable-retain-class"
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: name, Namespace: ns, UID: uid,
					Labels: agentLabels(agent, tenant),
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: stringPtr(originalClass),
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					VolumeName: pvName,
				},
				Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
			}
			pvc.Labels[LabelStorageRetention] = StorageRetentionRetain
			pv := &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{Name: pvName},
				Spec: corev1.PersistentVolumeSpec{
					StorageClassName: originalClass,
					ClaimRef: &corev1.ObjectReference{Name: name, Namespace: ns, UID: uid},
					PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
				},
			}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc, pv).Build()
			r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: scheme}
			for attempt := 0; attempt < 2; attempt++ {
				if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err != nil {
					t.Fatalf("reconcile %d rejected verified existing claim: %v", attempt, err)
				}
				var current corev1.PersistentVolumeClaim
				if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, &current); err != nil {
					t.Fatal(err)
				}
				if current.UID != uid || current.Spec.VolumeName != pvName ||
					current.Spec.StorageClassName == nil || *current.Spec.StorageClassName != originalClass {
					t.Fatalf("reconcile %d changed retained PVC binding: %#v", attempt, current.Spec)
				}
				var claims corev1.PersistentVolumeClaimList
				if err := c.List(ctx, &claims); err != nil {
					t.Fatal(err)
				}
				if len(claims.Items) != 1 {
					t.Fatalf("reconcile %d created additional agent claims: %#v", attempt, claims.Items)
				}
			}
		})
	}
}

func TestBoundRetainPVCProfileClassChangeRejectsUnverifiedBinding(t *testing.T) {
	for _, tc := range []struct {
		name string
		mutate func(*corev1.PersistentVolumeClaim, *corev1.PersistentVolume)
		omitPV bool
	}{
		{name: "foreign tenant", mutate: func(_ *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) {
			pv.Spec.ClaimRef.Namespace = "tenant-other"
		}},
		{name: "foreign UID", mutate: func(_ *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) {
			pv.Spec.ClaimRef.UID = types.UID("foreign")
		}},
		{name: "mismatched PV class", mutate: func(_ *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) {
			pv.Spec.StorageClassName = "different-class"
		}},
		{name: "missing identity label", mutate: func(pvc *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) {
			delete(pvc.Labels, LabelTenantName)
		}},
		{name: "pending claim", mutate: func(pvc *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) {
			pvc.Status.Phase = corev1.ClaimPending
		}},
		{name: "no PV", omitPV: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			scheme := testScheme(t)
			tenant := testTenant()
			tenant.Spec.AgentStorage = nil
			agent := testStorageIdentity(tenant.Name, "agent", StorageRetentionRetain)
			profile := testRuntimeProfile()
			profile.Spec.Storage.StorageClassName = "new-class"
			ns := tenantNamespace(tenant.Name)
			name := runtimePVCName(agent.Spec.AgentKey)
			uid := types.UID("old-claim-uid")
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, UID: uid,
					Labels: agentLabels(agent, tenant)},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: stringPtr("old-class"),
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					VolumeName: "bound-pv",
				},
				Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
			}
			pvc.Labels[LabelStorageRetention] = StorageRetentionRetain
			pv := &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{Name: "bound-pv"},
				Spec: corev1.PersistentVolumeSpec{
					StorageClassName: "old-class",
					ClaimRef: &corev1.ObjectReference{Namespace: ns, Name: name, UID: uid},
					PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
				},
			}
			if tc.mutate != nil {
				tc.mutate(pvc, pv)
			}
			builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc)
			if !tc.omitPV {
				builder = builder.WithObjects(pv)
			}
			c := builder.Build()
			r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: scheme}
			for attempt := 0; attempt < 2; attempt++ {
				if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err == nil {
					t.Fatalf("reconcile %d accepted an unverified Retain claim", attempt)
				}
			}
			var current corev1.PersistentVolumeClaim
			if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, &current); err != nil {
				t.Fatal(err)
			}
			if current.UID != pvc.UID || current.Spec.VolumeName != pvc.Spec.VolumeName ||
				*current.Spec.StorageClassName != *pvc.Spec.StorageClassName {
				t.Fatal("rejected retained claim identity was mutated")
			}
		})
	}
}
