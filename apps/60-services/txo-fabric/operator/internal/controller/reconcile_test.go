package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPOCContractReconcilesWithRealGatewayCommand(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := fabricv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	tenant := &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox"},
		Spec: fabricv1alpha1.TenantBundleSpec{
			TenantID:    "TEN90001",
			DisplayName: "hAIrem Sandbox",
			Persistence: fabricv1alpha1.TenantPersistenceSpec{PostgreSQL: &fabricv1alpha1.PostgreSQLSpec{Mode: "Shared", ProfileRef: "shared-poc"}},
			Memory:      fabricv1alpha1.TenantMemorySpec{Hindsight: &fabricv1alpha1.HindsightMemorySpec{ProfileRef: "shared-poc"}},
		},
	}
	profile := &fabricv1alpha1.AgentRuntimeProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "hermes-default"},
		Spec: fabricv1alpha1.AgentRuntimeProfileSpec{
			Engine:  "Hermes",
			Image:   "nousresearch/hermes-agent:v2026.9.24",
			Storage: fabricv1alpha1.RuntimeStorageSpec{Size: resource.MustParse("2Gi"), StorageClassName: "truenas-iscsi-delete"},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
				Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("2Gi")},
			},
			Compatibility: fabricv1alpha1.RuntimeCompatibilitySpec{S6Overlay: true},
		},
	}
	agent := &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: "tina"},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef:   fabricv1alpha1.ObjectReference{Name: "hairem-sandbox"},
			DisplayName: "Tina",
			Runtime:     fabricv1alpha1.AgentRuntimeBinding{ProfileRef: "hermes-default"},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}, &fabricv1alpha1.AgentIdentity{}).
		WithObjects(tenant, profile, agent).
		Build()

	tenantReconciler := &TenantBundleReconciler{Client: c, Scheme: scheme}
	reqTenant := ctrl.Request{NamespacedName: types.NamespacedName{Name: tenant.Name}}
	if _, err := tenantReconciler.Reconcile(ctx, reqTenant); err != nil {
		t.Fatal(err)
	}
	if _, err := tenantReconciler.Reconcile(ctx, reqTenant); err != nil {
		t.Fatal(err)
	}

	var ns corev1.Namespace
	if err := c.Get(ctx, types.NamespacedName{Name: "tenant-hairem-sandbox"}, &ns); err != nil {
		t.Fatalf("tenant namespace not reconciled: %v", err)
	}
	var defaultDeny networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: "txo-fabric-default-deny", Namespace: ns.Name}, &defaultDeny); err != nil {
		t.Fatalf("default deny not reconciled: %v", err)
	}

	agentReconciler := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	reqAgent := ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name}}
	if _, err := agentReconciler.Reconcile(ctx, reqAgent); err != nil {
		t.Fatal(err)
	}
	if _, err := agentReconciler.Reconcile(ctx, reqAgent); err != nil {
		t.Fatal(err)
	}

	var deployment appsv1.Deployment
	if err := c.Get(ctx, types.NamespacedName{Name: "hermes-tina", Namespace: ns.Name}, &deployment); err != nil {
		t.Fatalf("runtime deployment not reconciled: %v", err)
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if len(container.Args) != 2 || container.Args[0] != "gateway" || container.Args[1] != "run" {
		t.Fatalf("unexpected Hermes args: %#v", container.Args)
	}
	if container.ReadinessProbe == nil || container.ReadinessProbe.Exec == nil {
		t.Fatal("semantic Hermes readiness probe is missing")
	}
	if deployment.Spec.Template.Annotations["vixens.io/explicitly-allow-root"] != "true" {
		t.Fatal("s6 compatibility annotation is missing")
	}

	var pvc corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: "hermes-tina-data", Namespace: ns.Name}, &pvc); err != nil {
		t.Fatalf("runtime PVC not reconciled: %v", err)
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != "truenas-iscsi-delete" {
		t.Fatalf("unexpected storage class: %#v", pvc.Spec.StorageClassName)
	}

	var egress networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: "hermes-tina-egress", Namespace: ns.Name}, &egress); err != nil {
		t.Fatalf("runtime egress policy not reconciled: %v", err)
	}
	if len(egress.Spec.Egress) != 2 {
		t.Fatalf("expected DNS + tenant Hindsight egress rules, got %d", len(egress.Spec.Egress))
	}
}
