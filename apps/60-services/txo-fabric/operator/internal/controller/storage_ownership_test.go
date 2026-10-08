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

func testStorageIdentity(tenant string, key string, policy string) *fabricv1alpha1.AgentIdentity {
	return &fabricv1alpha1.AgentIdentity{
		ObjectMeta: metav1.ObjectMeta{Name: tenant + "-" + key, UID: types.UID(tenant + "-" + key + "-uid")},
		Spec: fabricv1alpha1.AgentIdentitySpec{
			TenantRef: fabricv1alpha1.ObjectReference{Name: tenant},
			AgentKey: key,
			Runtime: fabricv1alpha1.AgentRuntimeBinding{
				Storage: fabricv1alpha1.AgentRuntimeStorageBinding{RetentionPolicy: policy},
			},
		},
	}
}

func TestRetainedPVCNeverGetsGCOwnerOnCreationOrReconcile(t *testing.T) {
	ctx := context.Background()
	tenant := testTenant()
	agent := testStorageIdentity(tenant.Name, "spark", StorageRetentionRetain)
	profile := testRuntimeProfile()
	ns := tenantNamespace(tenant.Name)
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: testScheme(t)}
	for i := 0; i < 3; i++ {
		if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err != nil { t.Fatal(err) }
		var pvc corev1.PersistentVolumeClaim
		if err := c.Get(ctx, types.NamespacedName{Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns}, &pvc); err != nil { t.Fatal(err) }
		if owner := metav1.GetControllerOf(&pvc); owner != nil { t.Fatalf("retained pvc has controller owner on iteration %d: %#v", i, owner) }
		if pvc.Labels[LabelStorageRetention] != StorageRetentionRetain { t.Fatalf("unexpected retention label: %#v", pvc.Labels) }
	}
}

func TestRetainedPVCExistingOwnerRemovedBeforeAgentDeletion(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	agent := testStorageIdentity(tenant.Name, "spark", StorageRetentionRetain)
	profile := testRuntimeProfile()
	ns := tenantNamespace(tenant.Name)
	owned := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns, UID: types.UID("same-pvc-uid"),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "fabric.truxonline.io/v1alpha1", Kind: "AgentIdentity",
				Name: agent.Name, UID: agent.UID, Controller: func() *bool { x:=true; return &x }(),
			}},
		},
		Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: stringPtr(profile.Spec.Storage.StorageClassName)},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(owned).Build()
	r := &AgentIdentityReconciler{Client:c, Scheme:scheme}
	if err:=r.ensurePVC(ctx, agent, tenant, profile, ns); err!=nil { t.Fatal(err) }
	var current corev1.PersistentVolumeClaim
	if err:=c.Get(ctx, types.NamespacedName{Name:owned.Name,Namespace:ns}, &current); err!=nil {t.Fatal(err)}
	if current.UID!=owned.UID { t.Fatalf("retained PVC UID changed: %q vs %q",current.UID,owned.UID) }
	if owner:=metav1.GetControllerOf(&current); owner!=nil { t.Fatalf("legacy owner survived ordinary reconciliation: %#v",owner) }
	// A second reconcile must never reintroduce a GC edge.
	if err:=r.ensurePVC(ctx,agent,tenant,profile,ns); err!=nil {t.Fatal(err)}
	if err:=c.Get(ctx,types.NamespacedName{Name:owned.Name,Namespace:ns},&current);err!=nil {t.Fatal(err)}
	if len(current.OwnerReferences)!=0 {t.Fatalf("retained PVC owner reappeared: %#v",current.OwnerReferences)}
}

func TestDisposablePVCKeepsControllerOwnership(t *testing.T) {
	ctx:=context.Background()
	scheme:=testScheme(t)
	tenant:=testTenant()
	agent:=testStorageIdentity(tenant.Name, "probe", StorageRetentionDelete)
	profile:=testRuntimeProfile()
	ns:=tenantNamespace(tenant.Name)
	c:=fake.NewClientBuilder().WithScheme(scheme).Build()
	r:=&AgentIdentityReconciler{Client:c,Scheme:scheme}
	for i:=0;i<2;i++ {
		if err:=r.ensurePVC(ctx,agent,tenant,profile,ns);err!=nil {t.Fatal(err)}
		var pvc corev1.PersistentVolumeClaim
		if err:=c.Get(ctx,types.NamespacedName{Name:runtimePVCName(agent.Spec.AgentKey),Namespace:ns},&pvc);err!=nil{t.Fatal(err)}
		owner:=metav1.GetControllerOf(&pvc)
		if owner==nil || owner.UID!=agent.UID {t.Fatalf("Delete PVC needs its controller owner: %#v",pvc.OwnerReferences)}
	}
}

func TestUnownedRetainedPVCBindingEventRequeuesOnlyMatchingIdentity(t *testing.T) {
	ctx:=context.Background()
	scheme:=testScheme(t)
	tenant:=testTenant()
	spark:=testStorageIdentity(tenant.Name,"spark",StorageRetentionRetain)
	other:=testStorageIdentity(tenant.Name,"other",StorageRetentionRetain)
	foreign:=testStorageIdentity("indiba","spark",StorageRetentionRetain)
	c:=fake.NewClientBuilder().WithScheme(scheme).WithObjects(spark,other,foreign).Build()
	r:=&AgentIdentityReconciler{Client:c,Scheme:scheme}
	pvc:=&corev1.PersistentVolumeClaim{ObjectMeta:metav1.ObjectMeta{
		Name:runtimePVCName("spark"),Namespace:tenantNamespace(tenant.Name),
		Labels:map[string]string{
			LabelPartOf:"txo-fabric",LabelTenantName:tenant.Name,LabelAgent:"spark",
			LabelStorageRetention:StorageRetentionRetain,
		},
	}}
	got:=r.requestsForRuntimePVC(ctx,pvc)
	if len(got)!=1 || got[0].NamespacedName.Name!=spark.Name {t.Fatalf("PVC watch cross-tenant/nonmatched enqueue: %#v",got)}
	pvc.Namespace=tenantNamespace("indiba")
	if got:=r.requestsForRuntimePVC(ctx,pvc);len(got)!=0 {t.Fatalf("forged namespace triggered enqueue: %#v",got)}
	pvc.Namespace=tenantNamespace(tenant.Name)
	pvc.Labels[LabelPartOf]="not-fabric"
	if got:=r.requestsForRuntimePVC(ctx,pvc);len(got)!=0 {t.Fatalf("foreign PVC triggered enqueue: %#v",got)}
}
