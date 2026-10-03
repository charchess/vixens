package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestStorageRetentionPolicyDefaultsToRetain(t *testing.T) {
	agent := &fabricv1alpha1.AgentIdentity{}
	if got := storageRetentionPolicy(agent); got != StorageRetentionRetain {
		t.Fatalf("default retention = %q, want %q", got, StorageRetentionRetain)
	}
	agent.Spec.Runtime.Storage.RetentionPolicy = StorageRetentionDelete
	if got := storageRetentionPolicy(agent); got != StorageRetentionDelete {
		t.Fatalf("explicit retention = %q, want %q", got, StorageRetentionDelete)
	}
}

func TestRetainedPVCPromotesBoundPVReclaimPolicy(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox-tina", UID: types.UID("agent-uid")},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name},
			AgentKey:  "tina",
			Runtime: fabricv1alpha1.AgentRuntimeBinding{Storage: fabricv1alpha1.AgentRuntimeStorageBinding{
				RetentionPolicy: StorageRetentionRetain,
			}},
		},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hermes-tina-data",
			Namespace: "tenant-hairem-sandbox",
			UID:       types.UID("pvc-uid"),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr(profile.Spec.Storage.StorageClassName),
			VolumeName:       "pvc-tina",
		},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-tina"},
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{
				Namespace: pvc.Namespace,
				Name:      pvc.Name,
				UID:       pvc.UID,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc, pv).Build()
	r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: scheme}

	if err := r.ensurePVC(ctx, agent, tenant, profile, pvc.Namespace); err != nil {
		t.Fatal(err)
	}

	var current corev1.PersistentVolume
	if err := c.Get(ctx, types.NamespacedName{Name: pv.Name}, &current); err != nil {
		t.Fatal(err)
	}
	if current.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		t.Fatalf("PV reclaim policy = %q, want %q", current.Spec.PersistentVolumeReclaimPolicy, corev1.PersistentVolumeReclaimRetain)
	}
}

func TestDeletePolicyDoesNotDemoteRetainedPV(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	agent := &fabricv1alpha1.AgentIdentity{
		Spec: fabricv1alpha1.AgentIdentitySpec{
			Runtime: fabricv1alpha1.AgentRuntimeBinding{Storage: fabricv1alpha1.AgentRuntimeStorageBinding{
				RetentionPolicy: StorageRetentionDelete,
			}},
		},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "hermes-probe-data", Namespace: "tenant-fabric-smoke", UID: types.UID("pvc-uid")},
		Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: "pvc-probe"},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-probe"},
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{Namespace: pvc.Namespace, Name: pvc.Name, UID: pvc.UID},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pv).Build()
	r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: scheme}

	if err := r.ensureRetainedPersistentVolume(ctx, agent, pvc); err != nil {
		t.Fatal(err)
	}

	var current corev1.PersistentVolume
	if err := c.Get(ctx, types.NamespacedName{Name: pv.Name}, &current); err != nil {
		t.Fatal(err)
	}
	if current.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		t.Fatalf("Delete policy unexpectedly demoted PV reclaim policy to %q", current.Spec.PersistentVolumeReclaimPolicy)
	}
}

func TestAgentDeleteRetainsPVCAndRemovesOwnerReference(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	controller := true
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "hairem-sandbox-tina",
			UID:        types.UID("agent-uid"),
			Finalizers: []string{AgentFinalizer},
		},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: "hairem-sandbox"},
			AgentKey:  "tina",
		},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hermes-tina-data",
			Namespace: "tenant-hairem-sandbox",
			UID:       types.UID("pvc-uid"),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "fabric.truxonline.io/v1alpha1",
				Kind:       "AgentIdentity",
				Name:       agent.Name,
				UID:        agent.UID,
				Controller: &controller,
			}},
		},
		Spec: corev1.PersistentVolumeClaimSpec{VolumeName: "pvc-tina"},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc-tina"},
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{
				Namespace: pvc.Namespace,
				Name:      pvc.Name,
				UID:       pvc.UID,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, pvc, pv).Build()
	r := &AgentIdentityReconciler{Client: c, APIReader: c, Scheme: scheme}

	if _, err := r.reconcileDelete(ctx, agent); err != nil {
		t.Fatal(err)
	}

	var retained corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &retained); err != nil {
		t.Fatalf("retained PVC missing: %v", err)
	}
	if retained.Labels[LabelStorageRetention] != StorageRetentionRetain {
		t.Fatalf("retention label = %q", retained.Labels[LabelStorageRetention])
	}
	for _, owner := range retained.OwnerReferences {
		if owner.UID == agent.UID {
			t.Fatalf("retained PVC still owned by deleted AgentIdentity: %#v", retained.OwnerReferences)
		}
	}
	var retainedPV corev1.PersistentVolume
	if err := c.Get(ctx, types.NamespacedName{Name: pv.Name}, &retainedPV); err != nil {
		t.Fatal(err)
	}
	if retainedPV.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		t.Fatalf("retained agent deletion left PV reclaim policy = %q", retainedPV.Spec.PersistentVolumeReclaimPolicy)
	}
	var current fabricv1alpha1.AgentIdentity
	if err := c.Get(ctx, types.NamespacedName{Name: agent.Name}, &current); err != nil {
		t.Fatal(err)
	}
	for _, finalizer := range current.Finalizers {
		if finalizer == AgentFinalizer {
			t.Fatal("agent finalizer was not removed after PVC retention")
		}
	}
}

func TestAgentDeletePolicyDeletesPVC(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "fabric-smoke-probe", UID: types.UID("probe-uid"), Finalizers: []string{AgentFinalizer}},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: "fabric-smoke"},
			AgentKey:  "probe",
			Runtime: fabricv1alpha1.AgentRuntimeBinding{Storage: fabricv1alpha1.AgentRuntimeStorageBinding{
				RetentionPolicy: StorageRetentionDelete,
			}},
		},
	}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "hermes-probe-data", Namespace: "tenant-fabric-smoke"}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, pvc).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	if _, err := r.reconcileDelete(ctx, agent); err != nil {
		t.Fatal(err)
	}
	if _, err := r.reconcileDelete(ctx, agent); err != nil {
		t.Fatal(err)
	}

	var currentPVC corev1.PersistentVolumeClaim
	err := c.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &currentPVC)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("Delete policy retained PVC unexpectedly: err=%v", err)
	}
}

func TestRetainedPVCIsReadoptedBySameTenantAgentKey(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox-tina", UID: types.UID("replacement-uid")},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name},
			AgentKey:  "tina",
		},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hermes-tina-data",
			Namespace: "tenant-hairem-sandbox",
			Labels: map[string]string{
				LabelPartOf:           "txo-fabric",
				LabelTenantName:       tenant.Name,
				LabelAgent:            "tina",
				LabelStorageRetention: StorageRetentionRetain,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: stringPtr(profile.Spec.Storage.StorageClassName)},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	if err := r.ensurePVC(ctx, agent, tenant, profile, pvc.Namespace); err != nil {
		t.Fatal(err)
	}
	var adopted corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &adopted); err != nil {
		t.Fatal(err)
	}
	owner := metav1.GetControllerOf(&adopted)
	if owner == nil || owner.UID != agent.UID {
		t.Fatalf("retained PVC was not adopted by replacement identity: %#v", adopted.OwnerReferences)
	}
}

func TestTenantDeleteBlocksOnRetainedAgentPVC(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	bundle := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox", Finalizers: []string{TenantFinalizer}},
		Spec:       fabricv1alpha1.TenantBundleSpec{TenantID: "TEN90001", DisplayName: "hAIrem Sandbox"},
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant-hairem-sandbox"}}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name:      "hermes-tina-data",
		Namespace: ns.Name,
		Labels: map[string]string{
			LabelPartOf:           "txo-fabric",
			LabelTenantName:       bundle.Name,
			LabelAgent:            "tina",
			LabelStorageRetention: StorageRetentionRetain,
		},
	}}
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(bundle, ns, pvc).
		Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	result, err := r.reconcileDelete(ctx, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("tenant deletion with retained PVC should remain blocked")
	}
	var currentNS corev1.Namespace
	if err := c.Get(ctx, types.NamespacedName{Name: ns.Name}, &currentNS); err != nil {
		t.Fatalf("tenant namespace was deleted despite retained PVC: %v", err)
	}
	var current fabricv1alpha1.TenantBundle
	if err := c.Get(ctx, types.NamespacedName{Name: bundle.Name}, &current); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, condition := range current.Status.Conditions {
		if condition.Type == "Ready" && condition.Reason == "RetainedAgentStorage" {
			found = true
		}
	}
	if !found {
		t.Fatalf("RetainedAgentStorage condition missing: %#v", current.Status.Conditions)
	}
}
