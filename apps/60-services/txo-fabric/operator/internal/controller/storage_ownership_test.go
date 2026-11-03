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
	owned.Labels = agentLabels(agent, tenant)
	owned.Labels[LabelStorageRetention] = StorageRetentionRetain
	owned.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
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


func TestLegacyProfileAdoptionCannotProvisionAnEmptyPVC(t *testing.T) {
	for _, policy := range []string{StorageRetentionRetain, StorageRetentionDelete} {
		t.Run(policy, func(t *testing.T) {
			ctx := context.Background()
			scheme := testScheme(t)
			tenant := testTenant()
			profile := testRuntimeProfile()
			agent := testStorageIdentity(tenant.Name, "spark", policy)
			agent.Spec.Runtime.Storage.AdoptLegacyProfile = true
			ns := tenantNamespace(tenant.Name)
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
			if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err == nil {
				t.Fatal("legacy layout must not provision a new empty PVC")
			}
			var pvc corev1.PersistentVolumeClaim
			if err := c.Get(ctx, types.NamespacedName{
				Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns,
			}, &pvc); err == nil {
				t.Fatal("legacy layout created an empty replacement PVC")
			}
		})
	}
}

func TestNativeProfileStillProvisionsCanonicalPVCWithoutHistoricalInput(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := testStorageIdentity(tenant.Name, "spark", StorageRetentionRetain)
	ns := tenantNamespace(tenant.Name)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	for i := 0; i < 2; i++ {
		if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err != nil {
			t.Fatal(err)
		}
	}
	var pvc corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{
		Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns,
	}, &pvc); err != nil {
		t.Fatalf("native PVC not provisioned: %v", err)
	}
	if owner := metav1.GetControllerOf(&pvc); owner != nil {
		t.Fatalf("retained PVC unexpectedly acquired a GC owner: %#v", owner)
	}
}


func TestLegacyProfileCannotAdoptPendingClaim(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	tenant := testTenant()
	profile := testRuntimeProfile()
	agent := testStorageIdentity(tenant.Name, "spark", StorageRetentionRetain)
	agent.Spec.Runtime.Storage.AdoptLegacyProfile = true
	ns := tenantNamespace(tenant.Name)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns,
			UID: types.UID("pending-claim"), Labels: agentLabels(agent, tenant),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: stringPtr(profile.Spec.Storage.StorageClassName),
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
		Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
	}
	pvc.Labels[LabelStorageRetention] = StorageRetentionRetain
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
	r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
	if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err == nil {
		t.Fatal("pending claim must not be treated as historical data")
	}
	var after corev1.PersistentVolumeClaim
	if err := c.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: ns}, &after); err != nil {
		t.Fatal(err)
	}
	if after.UID != pvc.UID || after.Spec.VolumeName != "" ||
		after.Status.Phase != corev1.ClaimPending {
		t.Fatalf("rejected historical claim was mutated: %#v", after)
	}
}



func TestRetiredLegacyProfileCannotMutateExistingBoundPVC(t *testing.T) {
	for _, tenantName := range []string{"fabric-a", "fabric-b"} {
		t.Run(tenantName, func(t *testing.T) {
			ctx := context.Background()
			scheme := testScheme(t)
			tenant := testTenant()
			tenant.Name = tenantName
			tenant.Spec.TenantID = "TEN-" + tenantName
			agent := testStorageIdentity(tenant.Name, "spark", StorageRetentionRetain)
			agent.Spec.Runtime.Storage.AdoptLegacyProfile = true
			profile := testRuntimeProfile()
			ns := tenantNamespace(tenant.Name)
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: runtimePVCName(agent.Spec.AgentKey), Namespace: ns,
					UID: types.UID("retained-claim-" + tenantName),
					Labels: agentLabels(agent, tenant),
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: stringPtr(profile.Spec.Storage.StorageClassName),
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					VolumeName: "existing-pv-" + tenantName,
				},
				Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
			}
			pvc.Labels[LabelStorageRetention] = StorageRetentionRetain
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
			r := &AgentIdentityReconciler{Client: c, Scheme: scheme}
			for i := 0; i < 2; i++ {
				if err := r.ensurePVC(ctx, agent, tenant, profile, ns); err == nil {
					t.Fatal("retired legacy layout must reject even a Bound retained claim")
				}
			}
			var after corev1.PersistentVolumeClaim
			if err := c.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: ns}, &after); err != nil {
				t.Fatal(err)
			}
			if after.UID != pvc.UID || after.Spec.VolumeName != pvc.Spec.VolumeName ||
				after.Status.Phase != corev1.ClaimBound ||
				after.Labels[LabelStorageRetention] != StorageRetentionRetain {
				t.Fatalf("stored retained PVC changed during rejected reconciliation: %#v", after)
			}
		})
	}
}
