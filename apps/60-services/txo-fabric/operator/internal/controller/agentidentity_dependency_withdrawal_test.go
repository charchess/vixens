package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func invalidDependencyRuntimeFixture(t *testing.T) (*fabricv1alpha1.TenantBundle, *fabricv1alpha1.AgentIdentity, *fabricv1alpha1.AgentRuntimeProfile, *corev1.Namespace, *appsv1.Deployment, *corev1.Service, *corev1.PersistentVolumeClaim) {
	t.Helper()
	tenant := testTenant()
	agent := testStorageIdentity(tenant.Name, "probe", StorageRetentionRetain)
	agent.Finalizers = []string{AgentFinalizer}
	agent.Spec.Runtime.ProfileRef = testRuntimeProfile().Name
	agent.Status.Runtime.HumanEndpoint = "https://stale.invalid"
	profile := testRuntimeProfile()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNamespace(tenant.Name)}}
	controller := true
	owners := []metav1.OwnerReference{{
		APIVersion: fabricv1alpha1.GroupVersion.String(), Kind: "AgentIdentity",
		Name: agent.Name, UID: agent.UID, Controller: &controller,
	}}
	labels := agentLabels(agent, tenant)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: runtimeName(agent.Spec.AgentKey), Namespace: ns.Name, UID: "runtime-uid",
		Labels: labels, OwnerReferences: owners,
	}}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name: humanAccessResourceName(agent.Spec.AgentKey), Namespace: ns.Name, UID: "human-uid",
		Labels: labels, OwnerReferences: owners,
	}}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns.Name, UID: types.UID("retained-data-uid"),
	}}
	return tenant, agent, profile, ns, deployment, service, pvc
}

func TestInvalidAgentDependenciesWithdrawExistingRuntime(t *testing.T) {
	cases := []string{"missing-tenant", "missing-profile", "unsupported-runtime", "duplicate-agent-key", "missing-ai-gateway-backend"}
	for _, scenario := range cases {
		t.Run(scenario, func(t *testing.T) {
			tenant, agent, profile, ns, deployment, service, pvc := invalidDependencyRuntimeFixture(t)
			objects := []client.Object{agent, ns, deployment, service, pvc}
			switch scenario {
			case "missing-tenant":
				objects = append(objects, profile)
			case "missing-profile":
				objects = append(objects, tenant)
			case "unsupported-runtime":
				profile.Spec.Engine = "Unsupported"
				objects = append(objects, tenant, profile)
			case "missing-ai-gateway-backend":
				// No tenant AI gateway profile/Secret; the previous agent must
				// stop serving rather than fall back to the old model backend.
				objects = append(objects, tenant, profile)
			case "duplicate-agent-key":
				other := agent.DeepCopy()
				other.Name = "another-agent"
				other.UID = "other-agent-uid"
				other.Finalizers = nil
				objects = append(objects, tenant, profile, other)
			}
			scheme := testScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&fabricv1alpha1.AgentIdentity{}).
				WithObjects(objects...).Build()
			r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
			for i := 0; i < 2; i++ {
				if _, err := r.Reconcile(context.Background(), ctrl.Request{
					NamespacedName: types.NamespacedName{Name: agent.Name},
				}); err != nil {
					t.Fatalf("invalid dependency reconcile %d: %v", i, err)
				}
			}
			for _, obj := range []client.Object{deployment, service} {
				var current client.Object
				switch obj.(type) {
				case *appsv1.Deployment:
					current = &appsv1.Deployment{}
				case *corev1.Service:
					current = &corev1.Service{}
				}
				if err := c.Get(context.Background(), client.ObjectKeyFromObject(obj), current); !apierrors.IsNotFound(err) {
					t.Fatalf("stale serving object survived: %T error=%v", obj, err)
				}
			}
			var kept corev1.PersistentVolumeClaim
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(pvc), &kept); err != nil ||
				kept.UID != pvc.UID {
				t.Fatalf("retained agent data was modified: %v", err)
			}
			var current fabricv1alpha1.AgentIdentity
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(agent), &current); err != nil {
				t.Fatal(err)
			}
			if current.Status.Runtime.HumanEndpoint != "" {
				t.Fatalf("stale human endpoint remains: %s", current.Status.Runtime.HumanEndpoint)
			}
		})
	}
}

func TestDuplicateAgentNeverWithdrawsForeignOwnerRuntime(t *testing.T) {
	tenant, agent, profile, ns, deployment, service, _ := invalidDependencyRuntimeFixture(t)
	other := agent.DeepCopy()
	other.Name = "actual-owner"
	other.UID = types.UID("actual-owner-uid")
	deployment.OwnerReferences[0].Name = other.Name
	deployment.OwnerReferences[0].UID = other.UID
	service.OwnerReferences[0].Name = other.Name
	service.OwnerReferences[0].UID = other.UID
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.AgentIdentity{}).
		WithObjects(tenant, profile, agent, other, ns, deployment, service).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: agent.Name}}); err == nil {
		t.Fatal("duplicate AgentIdentity should not withdraw another owner's runtime")
	}
	for _, obj := range []client.Object{deployment, service} {
		var current client.Object
		if _, ok := obj.(*appsv1.Deployment); ok {
			current = &appsv1.Deployment{}
		} else {
			current = &corev1.Service{}
		}
		if err := c.Get(context.Background(), client.ObjectKeyFromObject(obj), current); err != nil {
			t.Fatalf("foreign owner's serving object was deleted: %v", err)
		}
	}
}
