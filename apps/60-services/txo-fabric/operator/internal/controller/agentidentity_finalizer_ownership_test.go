package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func deletionOwnershipFixture(t *testing.T) (*fabricv1alpha1.AgentIdentity, *fabricv1alpha1.TenantBundle, string, map[string]string, []metav1.OwnerReference) {
	t.Helper()
	tenant := testTenant()
	agent := testStorageIdentity(tenant.Name, "probe", StorageRetentionDelete)
	agent.Finalizers = []string{AgentFinalizer}
	labels := agentLabels(agent, tenant)
	labels[LabelStorageRetention] = StorageRetentionDelete
	controller := true
	owners := []metav1.OwnerReference{{
		APIVersion: fabricv1alpha1.GroupVersion.String(), Kind: "AgentIdentity",
		Name: agent.Name, UID: agent.UID, Controller: &controller,
	}}
	return agent, tenant, tenantNamespace(tenant.Name), labels, owners
}

func TestAgentDeleteFailsClosedForForeignOrUnownedResources(t *testing.T) {
	cases := []struct {
		name string
		makeObject func(*fabricv1alpha1.AgentIdentity, string, map[string]string, []metav1.OwnerReference) client.Object
	}{
		{"unowned-disposable-pvc", func(a *fabricv1alpha1.AgentIdentity, ns string, labels map[string]string, owners []metav1.OwnerReference) client.Object {
			return &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(a.Spec.AgentKey), Namespace: ns, Labels: labels}}
		}},
		{"foreign-uid-disposable-pvc", func(a *fabricv1alpha1.AgentIdentity, ns string, labels map[string]string, owners []metav1.OwnerReference) client.Object {
			owners[0].UID = types.UID("previous-agent-uid")
			return &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(a.Spec.AgentKey), Namespace: ns, Labels: labels, OwnerReferences: owners}}
		}},
		{"foreign-tenant-disposable-pvc", func(a *fabricv1alpha1.AgentIdentity, ns string, labels map[string]string, owners []metav1.OwnerReference) client.Object {
			labels[LabelTenantName] = "other"
			return &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(a.Spec.AgentKey), Namespace: ns, Labels: labels, OwnerReferences: owners}}
		}},
		{"wrong-retention-disposable-pvc", func(a *fabricv1alpha1.AgentIdentity, ns string, labels map[string]string, owners []metav1.OwnerReference) client.Object {
			labels[LabelStorageRetention] = StorageRetentionRetain
			return &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimePVCName(a.Spec.AgentKey), Namespace: ns, Labels: labels, OwnerReferences: owners}}
		}},
		{"unowned-human-service", func(a *fabricv1alpha1.AgentIdentity, ns string, labels map[string]string, owners []metav1.OwnerReference) client.Object {
			return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: humanAccessResourceName(a.Spec.AgentKey), Namespace: ns, Labels: labels}}
		}},
		{"foreign-uid-human-service", func(a *fabricv1alpha1.AgentIdentity, ns string, labels map[string]string, owners []metav1.OwnerReference) client.Object {
			owners[0].UID = types.UID("different-agent-uid")
			return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: humanAccessResourceName(a.Spec.AgentKey), Namespace: ns, Labels: labels, OwnerReferences: owners}}
		}},
		{"unowned-model-secret", func(a *fabricv1alpha1.AgentIdentity, ns string, labels map[string]string, owners []metav1.OwnerReference) client.Object {
			return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: modelAccessSecretName(a.Spec.AgentKey), Namespace: ns, Labels: labels}}
		}},
		{"foreign-uid-model-secret", func(a *fabricv1alpha1.AgentIdentity, ns string, labels map[string]string, owners []metav1.OwnerReference) client.Object {
			owners[0].UID = types.UID("different-agent-uid")
			return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: modelAccessSecretName(a.Spec.AgentKey), Namespace: ns, Labels: labels, OwnerReferences: owners}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agent, _, ns, labels, owners := deletionOwnershipFixture(t)
			obj := tc.makeObject(agent, ns, labels, owners)
			scheme := testScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, obj).Build()
			r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
			if _, err := r.reconcileDelete(context.Background(), agent); err == nil || !strings.Contains(err.Error(), "refusing to delete") {
				t.Fatalf("unowned object was not rejected: %v", err)
			}
			var kept client.Object
			switch obj.(type) {
			case *corev1.PersistentVolumeClaim:
				kept = &corev1.PersistentVolumeClaim{}
			case *corev1.Service:
				kept = &corev1.Service{}
			default:
				kept = &corev1.Secret{}
			}
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(obj), kept); err != nil {
				t.Fatalf("foreign object was destroyed: %v", err)
			}
			var current fabricv1alpha1.AgentIdentity
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(agent), &current); err != nil {
				t.Fatal(err)
			}
			if len(current.Finalizers) == 0 {
				t.Fatal("finalizer removed despite unverified resources")
			}
		})
	}
}

func TestAgentDeleteOwnedHumanResourcesIsIdempotent(t *testing.T) {
	agent, _, ns, labels, owners := deletionOwnershipFixture(t)
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name: humanAccessResourceName(agent.Spec.AgentKey), Namespace: ns, UID: types.UID("owned-service-uid"),
		Labels: labels, OwnerReferences: owners,
	}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name: modelAccessSecretName(agent.Spec.AgentKey), Namespace: ns, UID: types.UID("owned-secret-uid"),
		Labels: labels, OwnerReferences: owners,
	}}
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent, service, secret).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	for i := 0; i < 3; i++ {
		if _, err := r.reconcileDelete(context.Background(), agent); err != nil {
			t.Fatalf("delete pass %d: %v", i, err)
		}
	}
	for _, obj := range []client.Object{service, secret} {
		var current client.Object
		switch obj.(type) {
		case *corev1.Service:
			current = &corev1.Service{}
		case *corev1.Secret:
			current = &corev1.Secret{}
		}
		if err := c.Get(context.Background(), client.ObjectKeyFromObject(obj), current); err == nil {
			t.Fatalf("owned object %T was not deleted", obj)
		}
	}
}
