package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPostgreSQLUnsupportedTopologyIsExplicitlyBlocked(t *testing.T) {
	ctx := context.Background()
	scheme := postgresqlTestScheme(t)
	tenant := postgresqlTestTenant()
	profile := postgresqlTestProfile()
	profile.Spec.Topology = "DedicatedCluster"
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant, profile).
		Build()

	reconcilePostgreSQLTenant(t, ctx, c, scheme, tenant.Name, 2)
	current := getTenant(t, ctx, c, tenant.Name)
	condition := conditionByType(current.Status.Conditions, "PersistenceReady")
	if condition == nil || condition.Status != metav1.ConditionFalse || condition.Reason != "UnsupportedTopology" {
		t.Fatalf("unexpected PersistenceReady condition: %#v", condition)
	}
}
