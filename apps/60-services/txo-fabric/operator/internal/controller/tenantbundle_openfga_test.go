package controller

import (
	"context"
	"errors"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeOpenFGAProvisioner struct {
	calls []string
	stores map[string]openfga.Store
	err error
}
func (f *fakeOpenFGAProvisioner) EnsureTenantStore(_ context.Context, tenantID string) (openfga.Store,error) {
	f.calls = append(f.calls,tenantID)
	if f.err != nil { return openfga.Store{},f.err }
	if s,ok:=f.stores[tenantID];ok { return s,nil }
	return openfga.Store{},errors.New("not configured")
}

func openFGATestTenant(name, id, uid string) *fabricv1alpha1.TenantBundle {
	return &fabricv1alpha1.TenantBundle{
		ObjectMeta: metav1.ObjectMeta{Name:name,UID:types.UID(uid)},
		Spec: fabricv1alpha1.TenantBundleSpec{TenantID:id,DisplayName:name},
	}
}

func openFGATestReconciler(t *testing.T, f *fakeOpenFGAProvisioner, objs ...client.Object) (*TenantBundleReconciler,client.Client) {
	t.Helper()
	scheme:=postgresqlTestScheme(t)
	c:=fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
	r:=&TenantBundleReconciler{Client:c,Scheme:scheme}
	if f!=nil { r.OpenFGAStoreClient=f }
	return r,c
}

func TestOpenFGATenantBindingIdempotentAndIsolated(t *testing.T) {
	ctx:=context.Background()
	hairem:=openFGATestTenant("hairem","TEN00001","uid-hairem")
	indiba:=openFGATestTenant("indiba","TEN00002","uid-indiba")
	f:=&fakeOpenFGAProvisioner{stores:map[string]openfga.Store{
		"TEN00001": {ID:"01H0H015178Y2V4CX10C2KGHF4",Name:"txo-fabric-tenant-ten00001"},
		"TEN00002": {ID:"01H0H015178Y2V4CX10C2KGHF5",Name:"txo-fabric-tenant-ten00002"},
	}}
	r,c:=openFGATestReconciler(t,f,hairem,indiba)
	for _, tenant:=range []*fabricv1alpha1.TenantBundle{hairem,indiba,hairem} {
		if _,err:=r.reconcileOpenFGAStore(ctx,tenant);err!=nil {t.Fatal(err)}
	}
	var all corev1.ConfigMapList
	if err:=c.List(ctx,&all,client.InNamespace(openFGAPlatformNamespace));err!=nil {t.Fatal(err)}
	if len(all.Items)!=2 {t.Fatalf("expected exactly two bindings, got %d",len(all.Items))}
	for _, tenant:=range []*fabricv1alpha1.TenantBundle{hairem,indiba} {
		store,err:=r.readOpenFGAStoreBinding(ctx,tenant)
		if err!=nil {t.Fatal(err)}
		if store!=f.stores[tenant.Spec.TenantID] {t.Fatalf("unexpected binding for %s: %#v",tenant.Name,store)}
	}
}

func TestOpenFGARejectsDuplicateBusinessIdentityBeforeServiceCall(t *testing.T) {
	a:=openFGATestTenant("hairem","TEN00001","uid1")
	b:=openFGATestTenant("indiba","TEN00001","uid2")
	f:=&fakeOpenFGAProvisioner{stores:map[string]openfga.Store{}}
	r,c:=openFGATestReconciler(t,f,a,b)
	if _,err:=r.reconcileOpenFGAStore(context.Background(),a);err==nil {t.Fatal("duplicate business ID accepted")}
	if len(f.calls)!=0 {t.Fatal("reconciler called OpenFGA before validating tenant identity")}
	var bindings corev1.ConfigMapList
	_ = c.List(context.Background(),&bindings,client.InNamespace(openFGAPlatformNamespace))
	if len(bindings.Items)!=0 {t.Fatal("binding written for conflicting tenants")}
}

func TestOpenFGARetainedBindingBlocksImplicitNewCustomerAdoption(t *testing.T) {
	ctx:=context.Background()
	tenant:=openFGATestTenant("hairem","TEN00001","uid-original")
	f:=&fakeOpenFGAProvisioner{stores:map[string]openfga.Store{
		"TEN00001": {ID:"01H0H015178Y2V4CX10C2KGHF4",Name:"txo-fabric-tenant-ten00001"},
	}}
	r,_:=openFGATestReconciler(t,f,tenant)
	if _,err:=r.reconcileOpenFGAStore(ctx,tenant);err!=nil {t.Fatal(err)}

	// A removed/recreated TenantBundle must not silently inherit old store,
	// permissions, historical group memberships or credential relations.
	replacement:=tenant.DeepCopy()
	replacement.UID=types.UID("uid-recreated")
	callsBefore:=len(f.calls)
	if _,err:=r.reconcileOpenFGAStore(ctx,replacement);err==nil {t.Fatal("implicit store adoption after tenant recreation")}
	if len(f.calls)!=callsBefore {t.Fatal("called upstream despite rejected binding")}
	if _,err:=r.readOpenFGAStoreBinding(ctx,replacement);err==nil {t.Fatal("recreated tenant read retained store")}
}

func TestOpenFGARejectsBindingDriftAndMalformedResults(t *testing.T) {
	ctx:=context.Background()
	tenant:=openFGATestTenant("indiba","TEN00002","uid2")
	name,_:=openFGABindingName(tenant.Spec.TenantID)
	cases:=[]struct{name string; result openfga.Store; binding *corev1.ConfigMap}{
		{name:"unexpected-name",result:openfga.Store{ID:"01H0H015178Y2V4CX10C2KGHF4",Name:"txo-fabric-tenant-ten00001"}},
		{name:"no-id",result:openfga.Store{Name:"txo-fabric-tenant-ten00002"}},
		{name:"changed-id",result:openfga.Store{ID:"01H0H015178Y2V4CX10C2KGHF5",Name:"txo-fabric-tenant-ten00002"},binding:&corev1.ConfigMap{
			ObjectMeta:metav1.ObjectMeta{Namespace:openFGAPlatformNamespace,Name:name},
			Data:map[string]string{"tenantID":"TEN00002","tenantName":"indiba","tenantUID":"uid2","storeID":"01H0H015178Y2V4CX10C2KGHF4"},
		}},
	}
	for _,tc:=range cases {
		t.Run(tc.name,func(t *testing.T){
			f:=&fakeOpenFGAProvisioner{stores:map[string]openfga.Store{"TEN00002":tc.result}}
			obj:=[]client.Object{tenant}
			if tc.binding!=nil {obj=append(obj,tc.binding)}
			r,c:=openFGATestReconciler(t,f,obj...)
			if _,err:=r.reconcileOpenFGAStore(ctx,tenant);err==nil {t.Fatal("accepted malformed or drifting store")}
			binding:=&corev1.ConfigMap{}
			err:=c.Get(ctx,types.NamespacedName{Namespace:openFGAPlatformNamespace,Name:name},binding)
			if tc.binding==nil && err==nil {t.Fatal("created a binding from invalid service response")}
			if tc.binding!=nil && (err!=nil || binding.Data["storeID"]!="01H0H015178Y2V4CX10C2KGHF4") {
				t.Fatal("overwrote immutable store binding")
			}
		})
	}
}

func TestOpenFGAServiceOutageAndAbsentCredentialAreFailClosed(t *testing.T) {
	tenant:=openFGATestTenant("hairem","TEN00001","uid-hairem")
	f:=&fakeOpenFGAProvisioner{err:errors.New("network unavailable")}
	r,c:=openFGATestReconciler(t,f,tenant)
	if _,err:=r.reconcileOpenFGAStore(context.Background(),tenant);err==nil || !strings.Contains(err.Error(),"network unavailable"){
		t.Fatalf("outage not propagated: %v",err)
	}
	var list corev1.ConfigMapList
	_ = c.List(context.Background(),&list,client.InNamespace(openFGAPlatformNamespace))
	if len(list.Items)!=0 {t.Fatal("wrote binding despite outage")}

	withoutSecret,_:=openFGATestReconciler(t,nil,tenant)
	if _,err:=withoutSecret.reconcileOpenFGAStore(context.Background(),tenant);err==nil {
		t.Fatal("missing OpenBao-synced secret did not deny")
	}
}
