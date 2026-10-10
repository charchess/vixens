package controller

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
	"k8s.io/apimachinery/pkg/types"
)

func pinnedTestSource(name, uuid string, users ...string) *stagedIAMSource {
	return &stagedIAMSource{
		groups: map[string]authentik.GroupMembership{name:makeIAMGroup(name,uuid,users...)},
		calls:map[string]int{},fail:map[string]error{},
	}
}

func TestIAMGroupUUIDPinsAutomaticallyInitializeAndRemainStable(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	name:=authentikGroupName(b.Name,"sales")
	source:=pinnedTestSource(name,identityLedgerGroupUUID)
	if _,err:=r.readOpenFGAIAMGroupPins(ctx,b);err==nil {t.Fatal("accepted uninitialized group ledger")}
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,source);err!=nil {t.Fatal(err)}
	pins,err:=r.readOpenFGAIAMGroupPins(ctx,b)
	if err!=nil || pins[name]!=identityLedgerGroupUUID {t.Fatalf("wrong retained group identity: %v %v",pins,err)}
	binding:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	if binding.Data["storeID"]=="" || binding.Data["modelID"]=="" ||
		binding.Data[iamGroupPinsSchemaKey]!="v1" ||
		binding.Data[identityBindingsKey]!="[]" {t.Fatal("group pin overwrote original store/model or identity ledger")}
	revision:=binding.ResourceVersion
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,source);err!=nil {t.Fatal(err)}
	if after:=openFGAModelTestBinding(t,r,b.Spec.TenantID);after.ResourceVersion!=revision {
		t.Fatal("idempotent group reconcile rewrote durable ConfigMap")
	}
	if source.calls[name]<3 {t.Fatal("group was not revalidated against Authentik on retry")}
}

func TestIAMGroupUUIDPinsRefuseRecreationAndDeletionBeforeRevocation(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	name:=authentikGroupName(b.Name,"sales")
	source:=pinnedTestSource(name,identityLedgerGroupUUID)
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,source);err!=nil {t.Fatal(err)}
	before:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	source.groups[name]=makeIAMGroup(name,"550e8400-e29b-41d4-a716-446655440022")
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,source);err==nil {
		t.Fatal("silently adopted recreated Authentik group with matching name")
	}
	// Changing declared group set must never discard a pinned relation that
	// might still contain OpenFGA grants. Revocation requires a separate,
	// explicit lifecycle reconciler before the group is unpinned.
	modified:=b.DeepCopy()
	modified.Spec.HumanAccess.Web.IAMGroups=[]string{"admin"}
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,modified,source);err==nil {
		t.Fatal("dropped old group before group#member tuples were revoked")
	}
	after:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	if before.ResourceVersion!=after.ResourceVersion ||
		before.Data[iamGroupPinsDataKey]!=after.Data[iamGroupPinsDataKey] {
		t.Fatal("failed reconcile mutated retained group UUID pin")
	}
}

func TestIAMGroupUUIDPinsHandleAdditionsAndRejectCrossTenantMix(t *testing.T) {
	ctx:=context.Background()
	a:=humanIdentityTenant("hairem","TEN00001","uid-a","client0")
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,a,b)
	aName:=authentikGroupName(a.Name,"client0")
	bName:=authentikGroupName(b.Name,"sales")
	for _,tc:=range []struct{bundle *fabricv1alpha1.TenantBundle;name string}{
		{a,aName},{b,bName},
	} {
		if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,tc.bundle);err!=nil {t.Fatal(err)}
		if err:=r.reconcileOpenFGAIAMGroupPins(ctx,tc.bundle,pinnedTestSource(tc.name,identityLedgerGroupUUID));err!=nil {t.Fatal(err)}
	}
	b2:=b.DeepCopy()
	b2.Spec.HumanAccess.Web.IAMGroups=[]string{"sales","admin"}
	admin:=authentikGroupName(b.Name,"admin")
	src:=pinnedTestSource(bName,identityLedgerGroupUUID)
	src.groups[admin]=makeIAMGroup(admin,"550e8400-e29b-41d4-a716-446655440011")
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b2,src);err!=nil {t.Fatal(err)}
	pins,err:=r.readOpenFGAIAMGroupPins(ctx,b2)
	if err!=nil || len(pins)!=2 || pins[admin]!="550e8400-e29b-41d4-a716-446655440011" {
		t.Fatalf("safe group addition was not persisted: %v %v",pins,err)
	}
	aPins,err:=r.readOpenFGAIAMGroupPins(ctx,a)
	if err!=nil || len(aPins)!=1 || aPins[aName]!=identityLedgerGroupUUID {
		t.Fatalf("cross-tenant group collision: %v %v",aPins,err)
	}
	// The same UUID attributed to two different groups is ambiguous.
	duplicate:=pinnedTestSource(bName,identityLedgerGroupUUID)
	duplicate.groups[admin]=makeIAMGroup(admin,identityLedgerGroupUUID)
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b2,duplicate);err==nil {
		t.Fatal("duplicate Authentik UUID reused for two group identities")
	}
	newTenant:=b.DeepCopy();newTenant.UID=types.UID("replacement-uid")
	if _,err:=r.readOpenFGAIAMGroupPins(ctx,newTenant);err==nil {
		t.Fatal("new tenant object inherited retained group UUID binding")
	}
}

func TestIAMGroupUUIDPinsFailClosedOnPartialOrUntrustedAuthSource(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	name:=authentikGroupName(b.Name,"sales")
	for _,tc:=range []struct{name string;src *stagedIAMSource}{
		{"outage",func()*stagedIAMSource{s:=pinnedTestSource(name,identityLedgerGroupUUID);s.fail[name]=errors.New("upstream expired");return s}()},
		{"wrong-group",pinnedTestSource("txo-fabric-hairem-sales",identityLedgerGroupUUID)},
		{"malformed-uuid",pinnedTestSource(name,"not-a-uuid")},
		{"nil-users",func()*stagedIAMSource{s:=pinnedTestSource(name,identityLedgerGroupUUID);s.groups[name]=authentik.GroupMembership{Name:name,AuthentikGroupUUID:identityLedgerGroupUUID};return s}()},
		{"raced-group",func()*stagedIAMSource{s:=pinnedTestSource(name,identityLedgerGroupUUID);s.after=map[string]authentik.GroupMembership{name:makeIAMGroup(name,"550e8400-e29b-41d4-a716-446655440023")};return s}()},
	} {
		t.Run(tc.name,func(t *testing.T){
			if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,tc.src);err==nil {t.Fatal("trusted incomplete or racing group source")}
			binding:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
			if _,ok:=binding.Data[iamGroupPinsSchemaKey];ok {t.Fatal("wrote pins after source failure")}
		})
	}
	if _,err:=r.authentikMembershipSource(ctx);err==nil {
		t.Fatal("constructed runtime IAM reader without platform OpenBao token")
	}
}

func TestIAMGroupUUIDPinsRejectMalformedDurableLedger(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	name:=authentikGroupName(b.Name,"sales")
	src:=pinnedTestSource(name,identityLedgerGroupUUID)
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,src);err!=nil {t.Fatal(err)}
	initial:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	for _,tc:=range []struct{name string;mutate func(map[string]string)}{
		{"missing-schema",func(m map[string]string){delete(m,iamGroupPinsSchemaKey)}},
		{"missing-data",func(m map[string]string){delete(m,iamGroupPinsDataKey)}},
		{"schema-drift",func(m map[string]string){m[iamGroupPinsSchemaKey]="v2"}},
		{"malformed",func(m map[string]string){m[iamGroupPinsDataKey]="{"}},
		{"null",func(m map[string]string){m[iamGroupPinsDataKey]="null"}},
		{"duplicate-name",func(m map[string]string){m[iamGroupPinsDataKey]=`[{"name":"txo-fabric-indiba-sales","authentikGroupUUID":"550e8400-e29b-41d4-a716-446655440000"},{"name":"txo-fabric-indiba-sales","authentikGroupUUID":"550e8400-e29b-41d4-a716-446655440011"}]`}},
		{"duplicate-uuid",func(m map[string]string){m[iamGroupPinsDataKey]=`[{"name":"txo-fabric-indiba-sales","authentikGroupUUID":"550e8400-e29b-41d4-a716-446655440000"},{"name":"txo-fabric-hairem-client0","authentikGroupUUID":"550e8400-e29b-41d4-a716-446655440000"}]`}},
		{"foreign-group",func(m map[string]string){m[iamGroupPinsDataKey]=`[{"name":"txo-fabric-hairem-client0","authentikGroupUUID":"550e8400-e29b-41d4-a716-446655440000"}]`}},
		{"trailing-json",func(m map[string]string){m[iamGroupPinsDataKey]=`[] []`}},
		{"unknown-fields",func(m map[string]string){m[iamGroupPinsDataKey]=`[{"name":"txo-fabric-indiba-sales","authentikGroupUUID":"550e8400-e29b-41d4-a716-446655440000","admin":true}]`}},
		{"too-large",func(m map[string]string){m[iamGroupPinsDataKey]=strings.Repeat("x",maxIAMGroupPinsBytes+1)}},
	}{
		t.Run(tc.name,func(t *testing.T){
			current:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
			tc.mutate(current.Data)
			if err:=r.Update(ctx,&current);err!=nil {t.Fatal(err)}
			if _,err:=r.readOpenFGAIAMGroupPins(ctx,b);err==nil {t.Fatal("accepted invalid retained group UUID pins")}
			if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,src);err==nil {t.Fatal("silently repaired damaged group identity pins")}
			restore:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
			for k:=range restore.Data {delete(restore.Data,k)}
			for k,v:=range initial.Data {restore.Data[k]=v}
			if err:=r.Update(ctx,&restore);err!=nil {t.Fatal(err)}
		})
	}
	if _,err:=decodeIAMGroupPins("[]");err!=nil {t.Fatal(err)}
	if keys,err:=decodeIAMGroupPins(initial.Data[iamGroupPinsDataKey]);err!=nil || !reflect.DeepEqual(keys,map[string]string{name:identityLedgerGroupUUID}) {
		t.Fatalf("lost original protected pins %v %v",keys,err)
	}
}

func TestIAMGroupUUIDPinningNeedsApprovedStoreAndModel(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r,_:=openFGATestReconciler(t,&fakeOpenFGAProvisioner{stores:map[string]openfga.Store{}},b)
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,pinnedTestSource(authentikGroupName(b.Name,"sales"),identityLedgerGroupUUID));err==nil {
		t.Fatal("group UUID pinning accepted absent private FGA binding")
	}
}
