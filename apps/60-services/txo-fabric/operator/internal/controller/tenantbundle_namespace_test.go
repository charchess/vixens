package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTenantNamespaceStaysOutsideGarbageCollectionOwnership(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&fabricv1alpha1.TenantBundle{}).
		WithObjects(tenant).
		Build()

	reconcileTenant(t, ctx, c, scheme, tenant.Name)

	var ns corev1.Namespace
	if err := c.Get(ctx, types.NamespacedName{Name: "tenant-hairem-sandbox"}, &ns); err != nil {
		t.Fatalf("tenant namespace not reconciled: %v", err)
	}
	if len(ns.OwnerReferences) != 0 {
		t.Fatalf("tenant namespace must not be garbage-collected with TenantBundle: %#v", ns.OwnerReferences)
	}

	requests := tenantBundleRequestsForNamespace(ctx, &ns)
	if len(requests) != 1 || requests[0].Name != tenant.Name {
		t.Fatalf("namespace watch mapped to %#v, want TenantBundle %q", requests, tenant.Name)
	}
}

func TestUnmanagedNamespaceDoesNotEnqueueTenantBundle(t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "unmanaged",
		Labels: map[string]string{
			LabelTenantName: "hairem-sandbox",
		},
	}}
	if requests := tenantBundleRequestsForNamespace(context.Background(), ns); len(requests) != 0 {
		t.Fatalf("unmanaged namespace unexpectedly enqueued TenantBundle: %#v", requests)
	}
}
