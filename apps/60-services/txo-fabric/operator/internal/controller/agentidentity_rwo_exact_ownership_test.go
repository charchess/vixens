package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestVerifiedRWOPVCExistingClaimRequiresExactOwnership(t *testing.T) {
	cases := []struct {
		name string
		change func(*corev1.PersistentVolumeClaim, *corev1.PersistentVolume)
		valid bool
	}{
		{name: "valid-native-retain", valid: true},
		{name: "missing-all-labels", change: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Labels = nil }},
		{name: "missing-tenant-id", change: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { delete(p.Labels, LabelTenantID) }},
		{name: "foreign-tenant", change: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Labels[LabelTenantName] = "other" }},
		{name: "foreign-agent", change: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Labels[LabelAgent] = "other" }},
		{name: "retention-absent", change: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { delete(p.Labels, LabelStorageRetention) }},
		{name: "RWX-instead-of-RWO", change: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany} }},
		{name: "foreign-owner", change: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) {
			controller := true
			p.OwnerReferences = []metav1.OwnerReference{{APIVersion: "fabric.truxonline.io/v1alpha1", Kind: "AgentIdentity", Name: "other", UID: "other-uid", Controller: &controller}}
		}},
		{name: "foreign-pv-uid", change: func(_ *corev1.PersistentVolumeClaim, v *corev1.PersistentVolume) { v.Spec.ClaimRef.UID = "different" }},
		{name: "foreign-pv-namespace", change: func(_ *corev1.PersistentVolumeClaim, v *corev1.PersistentVolume) { v.Spec.ClaimRef.Namespace = "tenant-other" }},
		{name: "PV-storage-class-mismatch", change: func(_ *corev1.PersistentVolumeClaim, v *corev1.PersistentVolume) { v.Spec.StorageClassName = "foreign" }},
		{name: "missing-pvc-uid", change: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.UID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tenant := configuredAgentStorageTenant()
			agent := testStorageIdentity(tenant.Name, "spark", StorageRetentionRetain)
			profile := testRuntimeProfile()
			ns := tenantNamespace(tenant.Name)
			name := runtimePVCName(agent.Spec.AgentKey)
			labels := agentLabels(agent, tenant)
			labels[LabelStorageRetention] = StorageRetentionRetain
			uid := types.UID("existing-uid")
			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, UID: uid, Labels: labels},
				Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: stringPtr("storage-rwo-retain"),
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, VolumeName: "retained-pv"},
				Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound}}
			pv := &corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "retained-pv"},
				Spec: corev1.PersistentVolumeSpec{StorageClassName: "storage-rwo-retain",
					PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
					ClaimRef: &corev1.ObjectReference{Namespace: ns, Name: name, UID: uid}}}
			if tc.change != nil { tc.change(pvc, pv) }
			scheme := testScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc, pv).Build()
			r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: scheme}
			err := r.ensurePVC(ctx, agent, tenant, profile, ns)
			if tc.valid {
				if err != nil { t.Fatal(err) }
				if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err != nil { t.Fatalf("second reconcile: %v", err) }
			} else if err == nil { t.Fatal("foreign or incomplete pre-existing RWO PVC adopted") }
			var after corev1.PersistentVolumeClaim
			if err := c.Get(ctx, client.ObjectKeyFromObject(pvc), &after); err != nil { t.Fatal(err) }
			if after.UID != pvc.UID || after.Labels[LabelAgent] != pvc.Labels[LabelAgent] ||
				after.Labels[LabelTenantID] != pvc.Labels[LabelTenantID] ||
				after.Spec.VolumeName != pvc.Spec.VolumeName { t.Fatalf("foreign RWO was mutated: %+v", after) }
		})
	}
}

func TestRWOBrownfieldClassRequiresProvenBoundUID(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "spark", StorageRetentionRetain)
	profile := testRuntimeProfile()
	profile.Spec.Storage.StorageClassName = "storage-rwo-delete"
	labels := agentLabels(agent, tenant)
	labels[LabelStorageRetention] = StorageRetentionRetain
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: runtimePVCName(agent.Spec.AgentKey), Namespace: tenantNamespace(tenant.Name),
		Labels: labels, UID: "uid-old"},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr("storage-rwo-delete"), AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
	r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: scheme}
	if err := r.ensurePVC(ctx, agent, tenant, profile, pvc.Namespace); err == nil {
		t.Fatal("legacy StorageClass without proven bound PV was adopted")
	}
}
