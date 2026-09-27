package controller

import (
	"context"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTenantBaselineStaysOutsideGarbageCollectionOwnership(t *testing.T) {
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
	requests := tenantBundleRequestsForManagedObject(ctx, &ns)
	if len(requests) != 1 || requests[0].Name != tenant.Name {
		t.Fatalf("namespace watch mapped to %#v, want TenantBundle %q", requests, tenant.Name)
	}

	var baseline networkingv1.NetworkPolicy
	if err := c.Get(ctx, types.NamespacedName{Name: "txo-fabric-default-deny", Namespace: ns.Name}, &baseline); err != nil {
		t.Fatalf("tenant default deny not reconciled: %v", err)
	}
	if len(baseline.OwnerReferences) != 0 {
		t.Fatalf("tenant network baseline must survive while TenantBundle finalizer is blocked: %#v", baseline.OwnerReferences)
	}
	requests = tenantBundleRequestsForManagedObject(ctx, &baseline)
	if len(requests) != 1 || requests[0].Name != tenant.Name {
		t.Fatalf("network baseline watch mapped to %#v, want TenantBundle %q", requests, tenant.Name)
	}
}

func TestUnmanagedObjectDoesNotEnqueueTenantBundle(t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: "unmanaged",
		Labels: map[string]string{
			LabelTenantName: "hairem-sandbox",
		},
	}}
	if requests := tenantBundleRequestsForManagedObject(context.Background(), ns); len(requests) != 0 {
		t.Fatalf("unmanaged object unexpectedly enqueued TenantBundle: %#v", requests)
	}
}
