package controller

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type fakeOpenFGAModelClient struct {
	ids map[string]string
	ensureCalls []string
	validateCalls []string
	ensureErr error
	validateErr error
}

func (f *fakeOpenFGAModelClient) EnsureAuthorizationModel(_ context.Context, storeID string) (string, error) {
	f.ensureCalls = append(f.ensureCalls, storeID)
	if f.ensureErr != nil { return "", f.ensureErr }
	return f.ids[storeID], nil
}

func (f *fakeOpenFGAModelClient) ValidateModelBinding(_ context.Context, storeID, modelID string) error {
	f.validateCalls = append(f.validateCalls, storeID+":"+modelID)
	if f.validateErr != nil { return f.validateErr }
	if f.ids[storeID] != modelID {
		return errors.New("OpenFGA model drift")
	}
	return nil
}

func openFGAModelTestBinding(t *testing.T, r *TenantBundleReconciler, tenantID string) corev1.ConfigMap {
	t.Helper()
	name, err := openFGABindingName(tenantID)
	if err != nil { t.Fatal(err) }
	var binding corev1.ConfigMap
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: openFGAPlatformNamespace, Name: name}, &binding); err != nil {
		t.Fatal(err)
	}
	return binding
}

func TestOpenFGAModelBindingPersistsAtomicallyAndIsIdempotentPerTenant(t *testing.T) {
	ctx := context.Background()
	a := openFGATestTenant("hairem","TEN00001","original-a")
	b := openFGATestTenant("indiba","TEN00002","original-b")
	aStore := openfga.Store{ID:"01H0H015178Y2V4CX10C2KGHF4",Name:"txo-fabric-tenant-ten00001"}
	bStore := openfga.Store{ID:"01H0H015178Y2V4CX10C2KGHF5",Name:"txo-fabric-tenant-ten00002"}
	stores := &fakeOpenFGAProvisioner{stores:map[string]openfga.Store{"TEN00001":aStore,"TEN00002":bStore}}
	models := &fakeOpenFGAModelClient{ids:map[string]string{
		aStore.ID:"01H0H015178Y2V4CX10C2KGHF6",
		bStore.ID:"01H0H015178Y2V4CX10C2KGHF7",
	}}
	r,_ := openFGATestReconciler(t,stores,a,b)
	r.OpenFGAModelClient = models
	for _,item := range []struct{ tenantID string; store openfga.Store }{
		{"TEN00001",aStore},{"TEN00002",bStore},
	} {
		tenant := a
		if item.tenantID == "TEN00002" { tenant = b }
		store,err := r.reconcileOpenFGAStore(ctx,tenant)
		if err != nil || store != item.store { t.Fatalf("store: %#v %v",store,err) }
		if _,err:=r.readOpenFGAAuthorizationBinding(ctx,tenant);err==nil {
			t.Fatal("accepted a store-only binding as authorized")
		}
		modelID,err := r.reconcileOpenFGAModel(ctx,tenant,store)
		if err != nil || modelID != models.ids[store.ID] { t.Fatalf("model: %q %v",modelID,err) }
		bound,err := r.readOpenFGAAuthorizationBinding(ctx,tenant)
		if err != nil || bound.Store != store || bound.ModelID != modelID || len(bound.ModelFingerprint)!=64 {
			t.Fatalf("invalid durable binding: %#v %v",bound,err)
		}
		before := openFGAModelTestBinding(t,r,tenant.Spec.TenantID)
		if _,err := r.reconcileOpenFGAModel(ctx,tenant,store);err!=nil { t.Fatal(err) }
		after := openFGAModelTestBinding(t,r,tenant.Spec.TenantID)
		if before.ResourceVersion != after.ResourceVersion { t.Fatal("idempotent model reconcile changed retained ConfigMap") }
	}
	if len(models.ensureCalls)!=2 || len(models.validateCalls)!=4 {
		t.Fatalf("unexpected publication/verification calls: ensure=%v validate=%v",models.ensureCalls,models.validateCalls)
	}
}

func TestOpenFGAModelFailsClosedOnOutageAndIncompleteBindings(t *testing.T) {
	ctx := context.Background()
	tenant := openFGATestTenant("indiba","TEN00002","uid-indiba")
	store := openfga.Store{ID:"01H0H015178Y2V4CX10C2KGHF5",Name:"txo-fabric-tenant-ten00002"}
	stores := &fakeOpenFGAProvisioner{stores:map[string]openfga.Store{"TEN00002":store}}
	models := &fakeOpenFGAModelClient{ids:map[string]string{store.ID:"01H0H015178Y2V4CX10C2KGHF6"},ensureErr:errors.New("service outage")}
	r,_ := openFGATestReconciler(t,stores,tenant)
	r.OpenFGAModelClient = models
	if _,err:=r.reconcileOpenFGAStore(ctx,tenant);err!=nil {t.Fatal(err)}
	if _,err:=r.reconcileOpenFGAModel(ctx,tenant,store);err==nil || !strings.Contains(err.Error(),"service outage") {
		t.Fatalf("OpenFGA outage accepted: %v",err)
	}
	initial := openFGAModelTestBinding(t,r,tenant.Spec.TenantID)
	if initial.Data["modelID"]!="" || initial.Data["modelFingerprint"]!="" {
		t.Fatal("published durable model binding after upstream failure")
	}
	models.ensureErr=nil
	if _,err:=r.reconcileOpenFGAModel(ctx,tenant,store);err!=nil {t.Fatal(err)}
	models.validateErr=errors.New("model changed upstream")
	if _,err:=r.reconcileOpenFGAModel(ctx,tenant,store);err==nil {t.Fatal("accepted a stale model after service drift")}
	models.validateErr=nil

	for _,tc:=range []struct{key,value string}{
		{"modelID",""},{"modelFingerprint","bad-fingerprint"},
	} {
		binding:=openFGAModelTestBinding(t,r,tenant.Spec.TenantID)
		binding.Data[tc.key]=tc.value
		if err:=r.Update(ctx,&binding);err!=nil {t.Fatal(err)}
		calls:=len(models.ensureCalls)
		if _,err:=r.reconcileOpenFGAModel(ctx,tenant,store);err==nil {t.Fatalf("accepted malformed %s binding",tc.key)}
		if _,err:=r.readOpenFGAAuthorizationBinding(ctx,tenant);err==nil {t.Fatal("resolver accepted incomplete or drifting model")}
		if len(models.ensureCalls)!=calls {t.Fatal("model overwrite attempted for retained invalid binding")}
		// Recreate the approved bound value to inspect the next tamper case.
		binding=openFGAModelTestBinding(t,r,tenant.Spec.TenantID)
		binding.Data["modelID"]=models.ids[store.ID]
		fingerprint,err:=openfga.ModelFingerprint()
		if err!=nil {t.Fatal(err)}
		binding.Data["modelFingerprint"]=fingerprint
		if err:=r.Update(ctx,&binding);err!=nil {t.Fatal(err)}
	}
}

func TestOpenFGAModelRefusesCrossTenantStoreAndUnverifiedCreation(t *testing.T) {
	ctx:=context.Background()
	tenant:=openFGATestTenant("hairem","TEN00001","uid-hairem")
	store:=openfga.Store{ID:"01H0H015178Y2V4CX10C2KGHF4",Name:"txo-fabric-tenant-ten00001"}
	foreign:=openfga.Store{ID:"01H0H015178Y2V4CX10C2KGHF5",Name:"txo-fabric-tenant-ten00002"}
	stores:=&fakeOpenFGAProvisioner{stores:map[string]openfga.Store{"TEN00001":store}}
	models:=&fakeOpenFGAModelClient{ids:map[string]string{store.ID:"01H0H015178Y2V4CX10C2KGHF6"}}
	r,_:=openFGATestReconciler(t,stores,tenant)
	r.OpenFGAModelClient=models
	if _,err:=r.reconcileOpenFGAModel(ctx,tenant,store);err==nil {t.Fatal("model provisioned without a bound store")}
	if _,err:=r.reconcileOpenFGAStore(ctx,tenant);err!=nil {t.Fatal(err)}
	if _,err:=r.reconcileOpenFGAModel(ctx,tenant,foreign);err==nil {t.Fatal("forged cross-tenant store was accepted")}
	if len(models.ensureCalls)!=0 {t.Fatal("cross-tenant request reached upstream model API")}
	models.ids[store.ID]=""
	if _,err:=r.reconcileOpenFGAModel(ctx,tenant,store);err==nil {t.Fatal("empty model ID accepted")}
	if _,err:=r.readOpenFGAAuthorizationBinding(ctx,tenant);err==nil {t.Fatal("unconfirmed model resolved")}
}

func TestOpenFGAModelMissingServiceCredentialDenies(t *testing.T) {
	ctx:=context.Background()
	tenant:=openFGATestTenant("hairem","TEN00001","uid-hairem")
	store:=openfga.Store{ID:"01H0H015178Y2V4CX10C2KGHF4",Name:"txo-fabric-tenant-ten00001"}
	stores:=&fakeOpenFGAProvisioner{stores:map[string]openfga.Store{"TEN00001":store}}
	r,_:=openFGATestReconciler(t,stores,tenant)
	if _,err:=r.reconcileOpenFGAStore(ctx,tenant);err!=nil {t.Fatal(err)}
	if _,err:=r.reconcileOpenFGAModel(ctx,tenant,store);err==nil {
		t.Fatal("authorization model accepted without the Fabric-owned service credential")
	}
}
