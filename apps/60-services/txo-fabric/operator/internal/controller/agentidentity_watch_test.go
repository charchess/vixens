package controller

import (
	"context"
	"reflect"
	"sort"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func requestNames(requests []reconcile.Request) []string {
	names := make([]string, 0, len(requests))
	for _, request := range requests {
		names = append(names, request.Name)
	}
	sort.Strings(names)
	return names
}

func TestRuntimeProfileWatchFansOutOnlyToMatchingAgents(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	agents := []fabricv1alpha1.AgentIdentity{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "hairem-default"},
			Spec: fabricv1alpha1.AgentIdentitySpec{
				TenantRef: fabricv1alpha1.ObjectReference{Name: "hairem"},
				AgentKey:  "usr000001-agt00001",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "indiba-default"},
			Spec: fabricv1alpha1.AgentIdentitySpec{
				TenantRef: fabricv1alpha1.ObjectReference{Name: "indiba"},
				AgentKey:  "usr000001-agt00001",
				Runtime:   fabricv1alpha1.AgentRuntimeBinding{ProfileRef: "hermes-default"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "hairem-restricted"},
			Spec: fabricv1alpha1.AgentIdentitySpec{
				TenantRef: fabricv1alpha1.ObjectReference{Name: "hairem"},
				AgentKey:  "usr000001-agt00002",
				Runtime:   fabricv1alpha1.AgentRuntimeBinding{ProfileRef: "hermes-restricted"},
			},
		},
	}
	objects := make([]client.Object, 0, len(agents))
	for i := range agents {
		objects = append(objects, &agents[i])
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	requests := r.requestsForProfile(ctx, &fabricv1alpha1.AgentRuntimeProfile{ObjectMeta: metav1.ObjectMeta{Name: "hermes-default"}})
	got := requestNames(requests)
	want := []string{"hairem-default", "indiba-default"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default profile fan-out=%v want %v", got, want)
	}

	requests = r.requestsForProfile(ctx, &fabricv1alpha1.AgentRuntimeProfile{ObjectMeta: metav1.ObjectMeta{Name: "hermes-restricted"}})
	got = requestNames(requests)
	want = []string{"hairem-restricted"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restricted profile fan-out=%v want %v", got, want)
	}
}

func TestTenantWatchFansOutOnlyInsideTenant(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	agents := []fabricv1alpha1.AgentIdentity{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "hairem-one"},
			Spec: fabricv1alpha1.AgentIdentitySpec{
				TenantRef: fabricv1alpha1.ObjectReference{Name: "hairem"},
				AgentKey:  "usr000001-agt00001",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "hairem-two"},
			Spec: fabricv1alpha1.AgentIdentitySpec{
				TenantRef: fabricv1alpha1.ObjectReference{Name: "hairem"},
				AgentKey:  "usr000001-agt00002",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "indiba-one"},
			Spec: fabricv1alpha1.AgentIdentitySpec{
				TenantRef: fabricv1alpha1.ObjectReference{Name: "indiba"},
				AgentKey:  "usr000001-agt00001",
			},
		},
	}
	objects := make([]client.Object, 0, len(agents))
	for i := range agents {
		objects = append(objects, &agents[i])
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	requests := r.requestsForTenant(ctx, &fabricv1alpha1.TenantBundle{ObjectMeta: metav1.ObjectMeta{Name: "hairem"}})
	got := requestNames(requests)
	want := []string{"hairem-one", "hairem-two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tenant fan-out=%v want %v", got, want)
	}
}
