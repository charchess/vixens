package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func nativeWorkspaceGateFixture() (*fabricv1alpha1.TenantBundle, *fabricv1alpha1.AgentIdentity, *fabricv1alpha1.SharedWorkspaceProfile, *corev1.PersistentVolumeClaim, *corev1.PersistentVolume) {
	tenant := workspaceTestTenant()
	tenant.Name = "customer"
	tenant.Spec.Workspace.Groups = nil
	tenant.Spec.Workspace.Users = nil
	tenant.Spec.Workspace.Organization = fabricv1alpha1.WorkspaceScopeSpec{Reference: true}
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "customer-agent"},
		Spec: fabricv1alpha1.AgentIdentitySpec{TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name}, AgentKey: "probe"},
	}
	profile := &fabricv1alpha1.SharedWorkspaceProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-nfs"},
		Spec: fabricv1alpha1.SharedWorkspaceProfileSpec{StorageClassName: "storage-rwx-retain"},
	}
	scope := desiredWorkspaceScopes(tenant)[0]
	ns := tenantNamespace(tenant.Name)
	pvcName := workspacePVCName(scope)
	pvName := "pv-customer"
	uid := types.UID("fixed-pvc-uid")
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: pvcName, Namespace: ns, UID: uid, Labels: workspaceLabels(tenant, scope)},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr(profile.Spec.StorageClassName),
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			VolumeName: pvName,
		},
		Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: pvName},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: profile.Spec.StorageClassName,
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			ClaimRef: &corev1.ObjectReference{Name: pvcName, Namespace: ns, UID: uid},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{Driver: "test.csi.example", VolumeHandle: "opaque-id"},
			},
		},
	}
	return tenant, agent, profile, pvc, pv
}

func TestNativeWorkspaceMountGateRejectsUnverifiedClaims(t *testing.T) {
	cases := []struct {
		name string
		modify func(*corev1.PersistentVolumeClaim, *corev1.PersistentVolume)
	}{
		{name: "valid"},
		{name: "missing-tenant-label", modify: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) {
			delete(p.Labels, LabelTenantName)
		}},
		{name: "foreign-tenant-id", modify: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) {
			p.Labels[LabelTenantID] = "TEN99999"
		}},
		{name: "missing-scope", modify: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) {
			delete(p.Labels, LabelWorkspaceScope)
		}},
		{name: "wrong-class", modify: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) {
			p.Spec.StorageClassName = stringPtr("wrong-class")
		}},
		{name: "pending", modify: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) {
			p.Status.Phase = corev1.ClaimPending
		}},
		{name: "missing-uid", modify: func(p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) {
			p.UID = ""
		}},
		{name: "foreign-pv-uid", modify: func(_ *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) {
			pv.Spec.ClaimRef.UID = "different"
		}},
		{name: "foreign-pv-namespace", modify: func(_ *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) {
			pv.Spec.ClaimRef.Namespace = "tenant-stranger"
		}},
		{name: "wrong-reclaim", modify: func(_ *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) {
			pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimDelete
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tenant, agent, profile, pvc, pv := nativeWorkspaceGateFixture()
			if tc.modify != nil {
				tc.modify(pvc, pv)
			}
			scheme := testScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile, pvc, pv).Build()
			r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
			err := r.ensureNativeWorkspaceClaimsReady(context.Background(), agent, tenant)
			if tc.name == "valid" {
				if err != nil {
					t.Fatalf("owned bound claim must be available: %v", err)
				}
				if err := r.ensureNativeWorkspaceClaimsReady(context.Background(), agent, tenant); err != nil {
					t.Fatalf("second reconcile must be idempotent: %v", err)
				}
			} else if err == nil {
				t.Fatal("invalid or foreign claim was allowed into an agent mount")
			}
		})
	}
}

func TestNativeWorkspaceMountGateWaitsForProvisioning(t *testing.T) {
	tenant, agent, profile, _, _ := nativeWorkspaceGateFixture()
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := r.ensureNativeWorkspaceClaimsReady(context.Background(), agent, tenant); err == nil {
		t.Fatal("missing newly declared claim must not mount an empty fallback")
	}
}

func TestNativeWorkspaceInvalidClaimWithdrawalPreservesPersistentData(t *testing.T) {
	ctx := context.Background()
	tenant, agent, profile, pvc, pv := nativeWorkspaceGateFixture()
	delete(pvc.Labels, LabelTenantName)
	namespace := tenantNamespace(tenant.Name)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: runtimeName(agent.Spec.AgentKey), Namespace: namespace,
		Labels: map[string]string{LabelInstance: agent.Spec.AgentKey, LabelTenantName: tenant.Name},
	}}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: humanAccessResourceName(agent.Spec.AgentKey), Namespace: namespace}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, profile, pvc, pv, deployment, service).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := r.ensureNativeWorkspaceClaimsReady(ctx, agent, tenant); err == nil {
		t.Fatal("invalid mount identity must be rejected")
	}
	for i := 0; i < 2; i++ {
		if err := r.withdrawInvalidFunctionalRuntime(ctx, agent, tenant, namespace); err != nil {
			t.Fatalf("withdraw %d: %v", i+1, err)
		}
	}
	var after appsv1.Deployment
	if err := c.Get(ctx, client.ObjectKeyFromObject(deployment), &after); err == nil {
		t.Fatal("stale serving Deployment must be withdrawn")
	}
	var afterService corev1.Service
	if err := c.Get(ctx, client.ObjectKeyFromObject(service), &afterService); err == nil {
		t.Fatal("stale human entry must be withdrawn")
	}
	var afterPVC corev1.PersistentVolumeClaim
	if err := c.Get(ctx, client.ObjectKeyFromObject(pvc), &afterPVC); err != nil || afterPVC.UID != pvc.UID {
		t.Fatalf("withdraw must retain business PVC and UID: %v", err)
	}
	var afterPV corev1.PersistentVolume
	if err := c.Get(ctx, client.ObjectKeyFromObject(pv), &afterPV); err != nil {
		t.Fatalf("withdraw must retain PV: %v", err)
	}
}
