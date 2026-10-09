package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	identityLedgerGroupUUID = "550e8400-e29b-41d4-a716-446655440000"
	identityLedgerUserUUID = "550e8400-e29b-41d4-a716-446655440001"
)

func humanIdentityTenant(name, id, uid, group string) *fabricv1alpha1.TenantBundle {
	bundle := openFGATestTenant(name,id,uid)
	bundle.Spec.HumanAccess = &fabricv1alpha1.TenantHumanAccessSpec{
		Web: &fabricv1alpha1.HumanWebAccessSpec{
			IAMGroups: []string{group},
			DomainSuffix: "truxonline.com",
			OIDC: fabricv1alpha1.HumanAccessOIDCSpec{
				Issuer: "https://authentik.truxonline.com/application/o/txo-fabric-"+name+"/",
			},
		},
	}
	return bundle
}

func preparedHumanIdentityReconciler(t *testing.T, tenants ...*fabricv1alpha1.TenantBundle) *TenantBundleReconciler {
	t.Helper()
	storeMap:=map[string]openfga.Store{}
	modelMap:=map[string]string{}
	for i, tenant := range tenants {
		store := openfga.Store{
			ID: "01H0H015178Y2V4CX10C2KGHF" + string(rune('4'+i)),
			Name: "txo-fabric-tenant-"+strings.ToLower(tenant.Spec.TenantID),
		}
		storeMap[tenant.Spec.TenantID]=store
		modelMap[store.ID]= "01H0H015178Y2V4CX10C2KGHF" + string(rune('6'+i))
	}
	provisioner:=&fakeOpenFGAProvisioner{stores:storeMap}
	switch len(tenants) {
	case 1:
		r,_:=openFGATestReconciler(t,provisioner,tenants[0])
		r.OpenFGAModelClient=&fakeOpenFGAModelClient{ids:modelMap}
		for _, tenant:=range tenants {
			store,err:=r.reconcileOpenFGAStore(context.Background(),tenant); if err!=nil {t.Fatal(err)}
			if _,err=r.reconcileOpenFGAModel(context.Background(),tenant,store);err!=nil {t.Fatal(err)}
		}
		return r
	case 2:
		r,_:=openFGATestReconciler(t,provisioner,tenants[0],tenants[1])
		r.OpenFGAModelClient=&fakeOpenFGAModelClient{ids:modelMap}
		for _, tenant:=range tenants {
			store,err:=r.reconcileOpenFGAStore(context.Background(),tenant); if err!=nil {t.Fatal(err)}
			if _,err=r.reconcileOpenFGAModel(context.Background(),tenant,store);err!=nil {t.Fatal(err)}
		}
		return r
	default:
		t.Fatal("unexpected tenant fixture count")
		return nil
	}
}

func TestHumanIdentityLedgerBootstrapsFromTenantBundleAndRemainsStable(t *testing.T) {
	ctx:=context.Background()
	a:=humanIdentityTenant("hairem","TEN00001","original-a","client0")
	b:=humanIdentityTenant("indiba","TEN00002","original-b","sales")
	r:=preparedHumanIdentityReconciler(t,a,b)
	for _,bundle:=range []*fabricv1alpha1.TenantBundle{a,b} {
		if _,err:=r.readOpenFGAHumanIdentityRegistry(ctx,bundle);err==nil {
			t.Fatal("uninitialized identity registry was allowed")
		}
		if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,bundle);err!=nil {t.Fatal(err)}
		ledger:=openFGAModelTestBinding(t,r,bundle.Spec.TenantID)
		if ledger.Data[identityBindingsKey]!="[]" ||
			ledger.Data[identityIssuerKey]!=bundle.Spec.HumanAccess.Web.OIDC.Issuer ||
			len(ledger.Data["modelID"])!=26 ||
			ledger.Data[identitySchemaKey]!="v1" ||
			ledger.Data["tenantUID"]!=string(bundle.UID) {
			t.Fatalf("ledger bootstrap corrupted retained store/model/identity: %v",ledger.Data)
		}
		if _,err:=r.readOpenFGAHumanIdentityRegistry(ctx,bundle);err!=nil {t.Fatal(err)}
		rev:=ledger.ResourceVersion
		if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,bundle);err!=nil {t.Fatal(err)}
		if after:=openFGAModelTestBinding(t,r,bundle.Spec.TenantID); after.ResourceVersion!=rev {
			t.Fatal("repeated reconciliation rewrote complete identity ledger")
		}
	}
	// No OIDC principal is enrolled from tenant workspace users or human
	// group names alone.
	empty,err:=r.readOpenFGAHumanIdentityRegistry(ctx,a)
	if err!=nil {t.Fatal(err)}
	if id,err:=empty.ResolveOIDCSubject(a.Spec.HumanAccess.Web.OIDC.Issuer,"some-oidc-sub");err==nil || id!="" {
		t.Fatal("tenant or IAM group presence auto-enrolled a human")
	}
}

func TestHumanIdentityLedgerReadsDurableMappingsAcrossReconcilerRestarts(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","original-b","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	binding:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	identity:=authentik.IdentityBinding{
		TenantID:b.Spec.TenantID,
		Issuer:b.Spec.HumanAccess.Web.OIDC.Issuer,
		OIDCSubject:"hashed-sub-not-an-authentik-uuid",
		AuthentikUserUUID:identityLedgerUserUUID,
		FabricUserID:"usr000002",
	}
	// Simulate a trusted Fabric-only CAS enrollment (not a production
	// enrollment implementation). Never use username or an OIDC group claim.
	raw,err:=json.Marshal([]authentik.IdentityBinding{identity})
	if err!=nil {t.Fatal(err)}
	binding.Data[identityBindingsKey]=string(raw)
	if err:=r.Update(ctx,&binding);err!=nil {t.Fatal(err)}
	got,err:=r.readOpenFGAHumanIdentityRegistry(ctx,b)
	if err!=nil {t.Fatal(err)}
	id,err:=got.ResolveOIDCSubject(identity.Issuer,identity.OIDCSubject)
	if err!=nil || id!="usr000002" {t.Fatalf("identity mapping not restored: %s %v",id,err)}
	members,err:=got.ResolveGroupMembership(authentik.GroupMembership{
		Name:"txo-fabric-indiba-sales",AuthentikGroupUUID:identityLedgerGroupUUID,
		ActiveAuthentikUserUUIDs:[]string{identityLedgerUserUUID},
	})
	if err!=nil || !reflect.DeepEqual(members,[]string{"usr000002"}) {
		t.Fatalf("trusted Authentik UUID did not resolve to stable Fabric user: %v %v",members,err)
	}
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	after:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	if after.Data[identityBindingsKey]!=string(raw) {
		t.Fatal("reconcile silently dropped durable human identity bindings")
	}
	if _,err:=got.ResolveGroupMembership(authentik.GroupMembership{
		Name:"txo-fabric-indiba-sales",AuthentikGroupUUID:identityLedgerGroupUUID,
		ActiveAuthentikUserUUIDs:[]string{"550e8400-e29b-41d4-a716-446655440002"},
	});err==nil {
		t.Fatal("unmapped identity was silently granted a group")
	}
}

func TestHumanIdentityLedgerRefusesPartialCrossTenantOrCorruptState(t *testing.T) {
	ctx:=context.Background()
	a:=humanIdentityTenant("hairem","TEN00001","uid-a","client0")
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,a,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,a);err!=nil {t.Fatal(err)}
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	original:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	for _,tc:=range []struct{name string;mutate func(*corev1.ConfigMap)}{
		{"issuer-changed",func(cm *corev1.ConfigMap){cm.Data[identityIssuerKey]=a.Spec.HumanAccess.Web.OIDC.Issuer}},
		{"issuer-deleted",func(cm *corev1.ConfigMap){delete(cm.Data,identityIssuerKey)}},
		{"bindings-deleted",func(cm *corev1.ConfigMap){delete(cm.Data,identityBindingsKey)}},
		{"both-fields-deleted-but-schema-retained",func(cm *corev1.ConfigMap){delete(cm.Data,identityBindingsKey);delete(cm.Data,identityIssuerKey)}},
		{"schema-deleted",func(cm *corev1.ConfigMap){delete(cm.Data,identitySchemaKey)}},
		{"schema-upgraded-without-migration",func(cm *corev1.ConfigMap){cm.Data[identitySchemaKey]="v2"}},
		{"broken-json",func(cm *corev1.ConfigMap){cm.Data[identityBindingsKey]="{"}},
		{"json-null",func(cm *corev1.ConfigMap){cm.Data[identityBindingsKey]="null"}},
		{"trailing-json",func(cm *corev1.ConfigMap){cm.Data[identityBindingsKey]="[] []"}},
		{"unknown-field",func(cm *corev1.ConfigMap){cm.Data[identityBindingsKey]=`[{"TenantID":"TEN00002","UnexpectedAdmin":true}]`}},
		{"foreign-tenant",func(cm *corev1.ConfigMap){cm.Data[identityBindingsKey]=`[{"TenantID":"TEN00001","Issuer":"https://authentik.truxonline.com/application/o/txo-fabric-indiba/","OIDCSubject":"hash","AuthentikUserUUID":"550e8400-e29b-41d4-a716-446655440001","FabricUserID":"usr000002"}]`}},
		{"duplicate-subject",func(cm *corev1.ConfigMap){
			one:=authentik.IdentityBinding{TenantID:"TEN00002",Issuer:b.Spec.HumanAccess.Web.OIDC.Issuer,OIDCSubject:"same-sub",AuthentikUserUUID:identityLedgerUserUUID,FabricUserID:"usr000002"}
			two:=one;two.AuthentikUserUUID="550e8400-e29b-41d4-a716-446655440002";two.FabricUserID="usr000003"
			raw,_:=json.Marshal([]authentik.IdentityBinding{one,two});cm.Data[identityBindingsKey]=string(raw)
		}},
		{"oversized-ledger",func(cm *corev1.ConfigMap){cm.Data[identityBindingsKey]=strings.Repeat("x",maxIdentityBindingJSONBytes+1)}},
		{"store-id-tampered",func(cm *corev1.ConfigMap){cm.Data["storeID"]="01H0H015178Y2V4CX10C2KGHF4"}},
		{"model-fingerprint-tampered",func(cm *corev1.ConfigMap){cm.Data["modelFingerprint"]="bad"}},
		{"tenant-uid-tampered",func(cm *corev1.ConfigMap){cm.Data["tenantUID"]="recreated"}},
	}{
		t.Run(tc.name,func(t *testing.T){
			cm:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
			tc.mutate(&cm)
			if err:=r.Update(ctx,&cm);err!=nil {t.Fatal(err)}
			if _,err:=r.readOpenFGAHumanIdentityRegistry(ctx,b);err==nil {t.Fatal("accepted corrupted/foreign identity ledger")}
			if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err==nil {t.Fatal("reconciler silently healed conflicting ledger")}
			current:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
			for k:=range current.Data {delete(current.Data,k)}
			for k,v:=range original.Data {current.Data[k]=v}
			if err:=r.Update(ctx,&current);err!=nil {t.Fatal(err)}
		})
	}
	// A deleted/recreated tenant object cannot adopt the original ledger.
	recreated:=b.DeepCopy();recreated.UID=types.UID("uid-new")
	if _,err:=r.readOpenFGAHumanIdentityRegistry(ctx,recreated);err==nil {t.Fatal("recreated tenant adopted retained ledger")}
}

func TestHumanIdentityLedgerDeniesOnModelVerificationOutage(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	service:=r.OpenFGAModelClient.(*fakeOpenFGAModelClient)
	service.validateErr=fmt.Errorf("upstream unavailable")
	if _,err:=r.readOpenFGAHumanIdentityRegistry(ctx,b);err==nil {
		t.Fatal("accepted identity ledger while OpenFGA model verifier was unavailable")
	}
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err==nil {
		t.Fatal("reconcile claimed healthy identity source during verifier outage")
	}
	service.validateErr=nil
	if _,err:=r.readOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
}

func TestHumanIdentityLedgerBlocksImplicitIssuerMigrationAndDoesNotChangeWorkspaces(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	modified:=b.DeepCopy()
	modified.Spec.HumanAccess.Web.OIDC.Issuer="https://other.truxonline.com/application/o/txo-fabric-indiba/"
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,modified);err==nil {
		t.Fatal("existing identity ledger changed issuer without an explicit migration")
	}
	modified=b.DeepCopy();modified.Spec.HumanAccess.Web.IAMGroups=[]string{"admin"}
	// IAM groups are dynamic *source* definitions, not part of durable user
	// identity. Their authorization tuples must be revoked by later sync.
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,modified);err!=nil {t.Fatal(err)}
	if original:=openFGAModelTestBinding(t,r,b.Spec.TenantID); original.Data[identityIssuerKey]!=b.Spec.HumanAccess.Web.OIDC.Issuer {
		t.Fatal("group rename mutated stable human issuer")
	}
}
