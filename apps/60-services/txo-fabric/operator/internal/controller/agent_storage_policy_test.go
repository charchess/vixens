package controller

import (
	"context"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func configuredAgentStorageTenant() *fabricv1alpha1.TenantBundle {
	tenant := testTenant()
	tenant.Spec.AgentStorage = &fabricv1alpha1.TenantAgentStorageSpec{
		RetainedStorageClassName: "truenas-iscsi-retain",
		DisposableStorageClassName: "truenas-iscsi-delete",
	}
	return tenant
}

func TestNewRetainedAgentPVCUsesTenantStorageNotHermesProfile(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "spark", StorageRetentionRetain)
	profile := testRuntimeProfile()
	profile.Spec.Storage.StorageClassName = "truenas-iscsi-delete" // obsolete release profile class
	ns := tenantNamespace(tenant.Name)
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}

	if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err != nil { t.Fatal(err) }
	var original corev1.PersistentVolumeClaim
	key := types.NamespacedName{Namespace: ns, Name: runtimePVCName(agent.Spec.AgentKey)}
	if err := c.Get(ctx, key, &original); err != nil { t.Fatal(err) }
	if original.Spec.StorageClassName == nil || *original.Spec.StorageClassName != "truenas-iscsi-retain" {
		t.Fatalf("durable PVC must be born with Retain StorageClass, got %#v", original.Spec.StorageClassName)
	}
	if owner := metav1.GetControllerOf(&original); owner != nil { t.Fatalf("Retain PVC must not carry GC owner: %#v", owner) }

	// Switching Hermes stable -> edge may change runtime-profile fields. It must
	// not change or replace the tenant-managed PVC.
	profile.Name = "hermes-edge"
	profile.Spec.Storage.StorageClassName = "some-other-runtime-class"
	for i := 0; i < 2; i++ {
		if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err != nil { t.Fatal(err) }
		var current corev1.PersistentVolumeClaim
		if err := c.Get(ctx, key, &current); err != nil { t.Fatal(err) }
		if *current.Spec.StorageClassName != "truenas-iscsi-retain" || current.UID != original.UID {
			t.Fatalf("Hermes release/channel changed PVC: %+v", current)
		}
	}
}

func TestDisposableAgentPVCDirectlyUsesDeleteClass(t *testing.T) {
	ctx := context.Background()
	tenant := configuredAgentStorageTenant()
	agent := testStorageIdentity(tenant.Name, "probe", StorageRetentionDelete)
	profile := testRuntimeProfile()
	profile.Spec.Storage.StorageClassName = "truenas-iscsi-retain"
	ns := tenantNamespace(tenant.Name)
	scheme := testScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &AgentIdentityReconciler{Client:c,Scheme:scheme}
	if err := r.ensurePVC(ctx,agent,tenant,profile,ns);err!=nil{t.Fatal(err)}
	var pvc corev1.PersistentVolumeClaim
	if err:=c.Get(ctx,types.NamespacedName{Namespace:ns,Name:runtimePVCName(agent.Spec.AgentKey)},&pvc);err!=nil{t.Fatal(err)}
	if pvc.Spec.StorageClassName==nil || *pvc.Spec.StorageClassName!="truenas-iscsi-delete"{t.Fatalf("disposable PVC storage=%#v",pvc.Spec.StorageClassName)}
	if owner:=metav1.GetControllerOf(&pvc);owner==nil || owner.UID!=agent.UID {t.Fatalf("disposable PVC needs GC owner: %#v",pvc.OwnerReferences)}
}

func TestExistingBrownfieldRetainPVCStaysBoundAndPromotesPV(t *testing.T) {
	ctx:=context.Background()
	tenant:=configuredAgentStorageTenant()
	agent:=testStorageIdentity(tenant.Name,"spark",StorageRetentionRetain)
	profile:=testRuntimeProfile()
	profile.Spec.Storage.StorageClassName="truenas-iscsi-delete"
	ns:=tenantNamespace(tenant.Name)
	uid:=types.UID("legacy-uid")
	name:=runtimePVCName(agent.Spec.AgentKey)
	pvc:=&corev1.PersistentVolumeClaim{
		ObjectMeta:metav1.ObjectMeta{Name:name,Namespace:ns,UID:uid},
		Spec:corev1.PersistentVolumeClaimSpec{
			StorageClassName:stringPtr("truenas-iscsi-delete"),
			VolumeName:"old-pv",
		},
	}
	pv:=&corev1.PersistentVolume{
		ObjectMeta:metav1.ObjectMeta{Name:"old-pv"},
		Spec:corev1.PersistentVolumeSpec{
			ClaimRef:&corev1.ObjectReference{Namespace:ns,Name:name,UID:uid},
			PersistentVolumeReclaimPolicy:corev1.PersistentVolumeReclaimDelete,
		},
	}
	scheme:=testScheme(t)
	c:=fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc,pv).Build()
	r:=&AgentIdentityReconciler{Client:c,APIReader:c,Scheme:scheme}
	if err:=r.ensurePVC(ctx,agent,tenant,profile,ns);err!=nil{t.Fatal(err)}
	var current corev1.PersistentVolumeClaim
	if err:=c.Get(ctx,types.NamespacedName{Namespace:ns,Name:name},&current);err!=nil{t.Fatal(err)}
	if current.UID!=uid || *current.Spec.StorageClassName!="truenas-iscsi-delete" || current.Spec.VolumeName!="old-pv"{
		t.Fatalf("legacy retained PVC mutated: %#v",current)
	}
	var bound corev1.PersistentVolume
	if err:=c.Get(ctx,types.NamespacedName{Name:"old-pv"},&bound);err!=nil{t.Fatal(err)}
	if bound.Spec.PersistentVolumeReclaimPolicy!=corev1.PersistentVolumeReclaimRetain {t.Fatalf("legacy PV wasn't protected: %s",bound.Spec.PersistentVolumeReclaimPolicy)}
}

func TestUnexpectedExistingStorageClassFailsClosedWithoutPVCCreation(t *testing.T){
	ctx:=context.Background()
	tenant:=configuredAgentStorageTenant()
	agent:=testStorageIdentity(tenant.Name,"spark",StorageRetentionRetain)
	profile:=testRuntimeProfile()
	pvc:=&corev1.PersistentVolumeClaim{
		ObjectMeta:metav1.ObjectMeta{Name:runtimePVCName(agent.Spec.AgentKey),Namespace:tenantNamespace(tenant.Name),UID:types.UID("foreign-uid")},
		Spec:corev1.PersistentVolumeClaimSpec{StorageClassName:stringPtr("unapproved-class")},
	}
	scheme:=testScheme(t)
	c:=fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
	r:=&AgentIdentityReconciler{Client:c,Scheme:scheme}
	err:=r.ensurePVC(ctx,agent,tenant,profile,pvc.Namespace)
	if err==nil || !strings.Contains(err.Error(),"immutable storageClass"){t.Fatalf("unexpected class accepted: %v",err)}
	var current corev1.PersistentVolumeClaim
	if err:=c.Get(ctx,types.NamespacedName{Name:pvc.Name,Namespace:pvc.Namespace},&current);err!=nil{t.Fatal(err)}
	if current.UID!=pvc.UID{t.Fatal("foreign PVC replaced")}
}

func TestDisposableExistingWrongClassFailsClosed(t *testing.T){
	ctx:=context.Background()
	tenant:=configuredAgentStorageTenant()
	agent:=testStorageIdentity(tenant.Name,"probe",StorageRetentionDelete)
	profile:=testRuntimeProfile()
	ns:=tenantNamespace(tenant.Name)
	pvc:=&corev1.PersistentVolumeClaim{
		ObjectMeta:metav1.ObjectMeta{Name:runtimePVCName(agent.Spec.AgentKey),Namespace:ns},
		Spec:corev1.PersistentVolumeClaimSpec{StorageClassName:stringPtr("truenas-iscsi-retain")},
	}
	scheme:=testScheme(t)
	c:=fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
	r:=&AgentIdentityReconciler{Client:c,Scheme:scheme}
	if err:=r.ensurePVC(ctx,agent,tenant,profile,ns);err==nil {t.Fatal("Delete agent adopted retained class")}
}

func TestAgentStoragePolicyValidationAndLegacyFallback(t *testing.T){
	tenant:=configuredAgentStorageTenant()
	agent:=testStorageIdentity(tenant.Name,"x",StorageRetentionRetain)
	profile:=testRuntimeProfile()
	profile.Spec.Storage.StorageClassName="legacy-default"
	if got,err:=desiredAgentPVCStorageClass(agent,tenant,profile);err!=nil || got!="truenas-iscsi-retain"{t.Fatalf("Retain resolution: %s %v",got,err)}
	agent.Spec.Runtime.Storage.RetentionPolicy=StorageRetentionDelete
	if got,err:=desiredAgentPVCStorageClass(agent,tenant,profile);err!=nil || got!="truenas-iscsi-delete"{t.Fatalf("Delete resolution: %s %v",got,err)}
	tenant.Spec.AgentStorage.DisposableStorageClassName=""
	if _,err:=desiredAgentPVCStorageClass(agent,tenant,profile);err==nil{t.Fatal("incomplete tenant policy accepted")}
	tenant.Spec.AgentStorage.DisposableStorageClassName=tenant.Spec.AgentStorage.RetainedStorageClassName
	if _,err:=desiredAgentPVCStorageClass(agent,tenant,profile);err==nil{t.Fatal("identical Retain/Delete storage classes accepted")}
	tenant.Spec.AgentStorage=nil
	if got,err:=desiredAgentPVCStorageClass(agent,tenant,profile);err!=nil || got!="legacy-default"{t.Fatalf("legacy tenant fallback broken: %s %v",got,err)}
}
