package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func activeRuntimeFixture() (*fabricv1alpha1.TenantBundle, *fabricv1alpha1.AgentIdentity, *corev1.PersistentVolumeClaim, *corev1.PersistentVolume) {
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "active-canary", StorageRetentionRetain)
	agent.UID = types.UID("agent-uid")
	agent.Spec.Runtime.Storage.Migration = &fabricv1alpha1.AgentRuntimeStorageMigrationBinding{
		TargetClaimSuffix: "synelia", TargetStorageClassName: "storage-rwo-retain", Cutover: true,
	}
	name := runtimeMigrationTargetPVCName(agent.Spec.AgentKey, "synelia")
	agent.Spec.Runtime.Storage.ActiveVolume = &fabricv1alpha1.ActiveVolumeBinding{
		ClaimName: name, PVCUID: "pvc-uid", VolumeName: "pv-active-canary",
		StorageClassName: "storage-rwo-retain", AccessMode: "ReadWriteOnce", RetentionPolicy: "Retain",
	}
	labels := agentLabels(agent, tenant)
	labels[LabelStorageRetention] = StorageRetentionRetain
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: tenantNamespace(tenant.Name), UID: types.UID("pvc-uid"), Labels: labels},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr("storage-rwo-retain"), AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, VolumeName: "pv-active-canary",
		},
		Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pv-active-canary"},
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{Name: name, Namespace: tenantNamespace(tenant.Name), UID: pvc.UID},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
		},
		Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound},
	}
	return tenant, agent, pvc, pv
}

func TestActiveRuntimeVolumeSurvivesMissingMigrationHistoryAndIsReadOnly(t *testing.T) {
	ctx := context.Background()
	tenant, agent, pvc, pv := activeRuntimeFixture()
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc, pv).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	ns := tenantNamespace(tenant.Name)

	for i := 0; i < 2; i++ {
		if err := r.ensureRuntimeActiveVolume(ctx, agent, tenant, ns); err != nil {
			t.Fatalf("reconcile #%d depends on retired source/job: %v", i+1, err)
		}
	}
	if got := runtimeActivePVCName(agent); got != pvc.Name {
		t.Fatalf("selected %q instead of pinned PVC %q", got, pvc.Name)
	}
	var after corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: ns}, &after); err != nil || after.UID != pvc.UID {
		t.Fatalf("active PVC mutated/replaced: %#v %v", after, err)
	}
	var source corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns}, &source); err == nil {
		t.Fatal("created a historical source PVC")
	}
	if err := r.ensureDeploymentRuntime(ctx, agent, tenant, testRuntimeProfile(), ns, "", "", effectiveToolsetPolicy{}, integrationResolution{}); err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: runtimeName(agent.Spec.AgentKey), Namespace: ns}, &deployment); err != nil {
		t.Fatal(err)
	}
	if got := deployment.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName; got != pvc.Name {
		t.Fatalf("Deployment mounted %q, expected exact pinned claim %q", got, pvc.Name)
	}
}

func TestActiveRuntimeVolumeFailsClosedOnIdentityAndOwnershipDrift(t *testing.T) {
	cases := []struct {
		name string
		mutate func(*fabricv1alpha1.TenantBundle, *fabricv1alpha1.AgentIdentity, *corev1.PersistentVolumeClaim, *corev1.PersistentVolume)
		want string
	}{
		{"wrong-pvc-uid", func(_ *fabricv1alpha1.TenantBundle, _ *fabricv1alpha1.AgentIdentity, p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.UID = types.UID("replacement") }, "UID/Bound/lifecycle"},
		{"pending-empty-claim", func(_ *fabricv1alpha1.TenantBundle, _ *fabricv1alpha1.AgentIdentity, p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Status.Phase = corev1.ClaimPending; p.Spec.VolumeName = "" }, "UID/Bound/lifecycle"},
		{"foreign-tenant", func(_ *fabricv1alpha1.TenantBundle, _ *fabricv1alpha1.AgentIdentity, p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Labels[LabelTenantName] = "foreign" }, "owner/scope"},
		{"foreign-agent", func(_ *fabricv1alpha1.TenantBundle, _ *fabricv1alpha1.AgentIdentity, p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Labels[LabelAgent] = "foreign" }, "owner/scope"},
		{"wrong-class", func(_ *fabricv1alpha1.TenantBundle, _ *fabricv1alpha1.AgentIdentity, p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Spec.StorageClassName = stringPtr("other") }, "mismatched PV"},
		{"wrong-access", func(_ *fabricv1alpha1.TenantBundle, _ *fabricv1alpha1.AgentIdentity, p *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { p.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany} }, "mismatched PV"},
		{"wrong-retention", func(_ *fabricv1alpha1.TenantBundle, _ *fabricv1alpha1.AgentIdentity, _ *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) { pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimDelete }, "reclaim policy"},
		{"wrong-pv-claimref", func(_ *fabricv1alpha1.TenantBundle, _ *fabricv1alpha1.AgentIdentity, _ *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume) { pv.Spec.ClaimRef.UID = types.UID("other") }, "pinned PVC UID"},
		{"preparation-not-finalized", func(_ *fabricv1alpha1.TenantBundle, a *fabricv1alpha1.AgentIdentity, _ *corev1.PersistentVolumeClaim, _ *corev1.PersistentVolume) { a.Spec.Runtime.Storage.Migration.Cutover = false }, "still in preparation"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tenant, agent, pvc, pv := activeRuntimeFixture()
			tc.mutate(tenant, agent, pvc, pv)
			scheme := testScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc, pv).Build()
			r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
			err := r.ensureRuntimeActiveVolume(context.Background(), agent, tenant, tenantNamespace(tenant.Name))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("fail-closed check expected %q, got %v", tc.want, err)
			}
		})
	}
}
