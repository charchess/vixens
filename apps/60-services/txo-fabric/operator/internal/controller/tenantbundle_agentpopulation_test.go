package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileAgentPopulationCreatesMissingIdentityDeterministically(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	tenant.UID = "tenant-uid"
	tenant.Spec.AgentPopulation = []fabricv1alpha1.TenantAgentPopulationEntry{{
		AgentKey: "tina", DisplayName: "Tina",
	}}
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant).
		Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileAgentPopulation(ctx, tenant); err != nil {
		t.Fatalf("reconcile population: %v", err)
	}
	if err := r.reconcileAgentPopulation(ctx, tenant); err != nil {
		t.Fatalf("reconcile population must be idempotent: %v", err)
	}

	var agent fabricv1alpha1.AgentIdentity
	if err := c.Get(ctx, types.NamespacedName{Name: "hairem-sandbox-tina"}, &agent); err != nil {
		t.Fatalf("generated identity: %v", err)
	}
	if agent.Spec.TenantRef.Name != tenant.Name || agent.Spec.AgentKey != "tina" || agent.Spec.DisplayName != "Tina" {
		t.Fatalf("unexpected generated identity: %#v", agent.Spec)
	}
	if len(agent.OwnerReferences) != 0 {
		t.Fatalf("population identity must not receive garbage-collection ownership: %#v", agent.OwnerReferences)
	}
}

func TestReconcileAgentPopulationFailsClosedWithoutPartialCreation(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	tenant.UID = "tenant-uid"
	tenant.Spec.AgentPopulation = []fabricv1alpha1.TenantAgentPopulationEntry{
		{AgentKey: "tina", DisplayName: "Tina"},
		{AgentKey: "zoe", DisplayName: "Zoe"},
	}
	unmanaged := &fabricv1alpha1.AgentIdentity{ObjectMeta: metav1.ObjectMeta{Name: "hairem-sandbox-tina"}, Spec: fabricv1alpha1.AgentIdentitySpec{TenantRef: fabricv1alpha1.ObjectReference{Name: tenant.Name}, AgentKey: "tina", DisplayName: "Existing Tina"}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).WithObjects(tenant, unmanaged).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}
	err := r.reconcileAgentPopulation(ctx, tenant)
	if err == nil || !strings.Contains(err.Error(), "refusing to adopt or modify") {
		t.Fatalf("want fail-closed unmanaged conflict, got %v", err)
	}
	var absent fabricv1alpha1.AgentIdentity
	if err := c.Get(ctx, types.NamespacedName{Name: "hairem-sandbox-zoe"}, &absent); !apierrors.IsNotFound(err) {
		t.Fatalf("conflict must prevent partial creation, get zoe: %v", err)
	}
}
