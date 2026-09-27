package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTenantCapabilityProfilesResolveBeforeControllersExist(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	postgres := testPostgreSQLProfile("Shared")
	hindsight := testHindsightProfile()

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant, postgres, hindsight).
		Build()

	reconcileTenant(t, ctx, c, scheme, tenant.Name)

	var current fabricv1alpha1.TenantBundle
	if err := c.Get(ctx, types.NamespacedName{Name: tenant.Name}, &current); err != nil {
		t.Fatal(err)
	}
	assertCondition(t, current.Status.Conditions, "PersistenceProfileResolved", metav1.ConditionTrue, "Resolved")
	assertCondition(t, current.Status.Conditions, "MemoryProfileResolved", metav1.ConditionTrue, "Resolved")
	assertCondition(t, current.Status.Conditions, "PersistenceReady", metav1.ConditionFalse, "ControllerNotImplemented")
	assertCondition(t, current.Status.Conditions, "MemoryReady", metav1.ConditionFalse, "ControllerNotImplemented")
	if current.Status.Phase != "Provisioning" {
		t.Fatalf("tenant phase = %q, want Provisioning", current.Status.Phase)
	}
}

func TestTenantRejectsPostgreSQLProfileModeMismatch(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	postgres := testPostgreSQLProfile("Dedicated")
	hindsight := testHindsightProfile()

	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant, postgres, hindsight).
		Build()

	reconcileTenant(t, ctx, c, scheme, tenant.Name)

	var current fabricv1alpha1.TenantBundle
	if err := c.Get(ctx, types.NamespacedName{Name: tenant.Name}, &current); err != nil {
		t.Fatal(err)
	}
	assertCondition(t, current.Status.Conditions, "PersistenceProfileResolved", metav1.ConditionFalse, "ProfileModeMismatch")
	assertCondition(t, current.Status.Conditions, "PersistenceReady", metav1.ConditionFalse, "ProfileModeMismatch")
}

func TestCapabilityProfileChangeRequeuesReferencingTenant(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	postgres := testPostgreSQLProfile("Shared")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant, postgres).Build()
	r := &TenantBundleReconciler{Client: c, Scheme: scheme}

	requests := r.tenantBundleRequestsForCapabilityProfile(ctx, postgres)
	if len(requests) != 1 || requests[0].Name != tenant.Name {
		t.Fatalf("profile watch mapped to %#v, want TenantBundle %q", requests, tenant.Name)
	}
}

func testPostgreSQLProfile(mode string) *fabricv1alpha1.PostgreSQLProfile {
	return &fabricv1alpha1.PostgreSQLProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-standard"},
		Spec: fabricv1alpha1.PostgreSQLProfileSpec{
			Provider:           "CloudNativePG",
			Mode:               mode,
			ClusterRef:         &fabricv1alpha1.NamespacedObjectReference{Name: "postgresql-shared", Namespace: "databases"},
			DatabaseNamePrefix: "txo_",
			RequiredExtensions: []string{"vector"},
			Credentials: fabricv1alpha1.PostgreSQLCredentialProfileSpec{
				SecretStoreRef:  fabricv1alpha1.SecretStoreReference{Name: "openbao", Kind: "ClusterSecretStore"},
				RemoteKeyPrefix: "vixens/prod/txo-fabric/tenants",
			},
		},
	}
}

func testHindsightProfile() *fabricv1alpha1.HindsightProfile {
	return &fabricv1alpha1.HindsightProfile{
		ObjectMeta: metav1.ObjectMeta{Name: "hindsight-standard"},
		Spec: fabricv1alpha1.HindsightProfileSpec{
			Image:           "ghcr.io/vectorize-io/hindsight:0.10.1",
			Port:            8888,
			APIAuthMode:     "ApiKey",
			LLMAuthMode:     "Unconfigured",
			FileStorageType: "PostgreSQL",
		},
	}
}

func assertCondition(t *testing.T, conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus, reason string) {
	t.Helper()
	for _, condition := range conditions {
		if condition.Type == conditionType {
			if condition.Status != status || condition.Reason != reason {
				t.Fatalf("condition %s = (%s, %s), want (%s, %s)", conditionType, condition.Status, condition.Reason, status, reason)
			}
			return
		}
	}
	t.Fatalf("condition %s missing: %#v", conditionType, conditions)
}
