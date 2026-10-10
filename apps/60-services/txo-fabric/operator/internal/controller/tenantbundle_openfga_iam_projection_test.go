package controller

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
)

type stagedIAMSource struct {
	groups map[string]authentik.GroupMembership
	after map[string]authentik.GroupMembership
	fail map[string]error
	calls map[string]int
}
func (s *stagedIAMSource) SnapshotGroupMembership(_ context.Context, name string) (authentik.GroupMembership,error) {
	s.calls[name]++
	if err:=s.fail[name];err!=nil { return authentik.GroupMembership{},err }
	if s.after!=nil && s.calls[name]>=2 {
		if m,ok:=s.after[name];ok {return m,nil}
	}
	if m,ok:=s.groups[name];ok {return m,nil}
	return authentik.GroupMembership{},errors.New("no authoritative group")
}

type recordingIAMWriter struct {
	current map[string][]openfga.Tuple
	stores map[string]bool
	models map[string]bool
	calls []openfga.TupleScope
	failFor string
}
func (w *recordingIAMWriter) ReconcileTupleScope(_ context.Context, storeID,modelID string, scope openfga.TupleScope, desired []openfga.Tuple) error {
	w.calls=append(w.calls,scope)
	if w.failFor==scope.Object {return errors.New("OpenFGA unavailable")}
	w.stores[storeID]=true
	w.models[modelID]=true
	key:=storeID+"/"+scope.Object+"/"+scope.Relation
	// Simulate exact-set reconciliation: anything missing from the new
	// authoritative source snapshot is revoked, not accumulated.
	w.current[key]=append([]openfga.Tuple(nil),desired...)
	return nil
}
func (w *recordingIAMWriter) at(storeID,groupName string) []openfga.Tuple {
	return w.current[storeID+"/group:"+groupName+"/member"]
}

func newIAMWriter() *recordingIAMWriter {
	return &recordingIAMWriter{
		current:map[string][]openfga.Tuple{},stores:map[string]bool{},models:map[string]bool{},
	}
}
func enrolledIAMTenant(t *testing.T,r *TenantBundleReconciler,tenantName,tenantID string, users ...struct{sub,uuid,id string}) {
	t.Helper()
	ctx:=context.Background()
	bundle:=humanIdentityTenant(tenantName,tenantID,"ignored","sales")
	// read/reconcile ownership is tested against the actual fixture UID,
	// not the ignored fixture copy.
	var aBinding struct{}
	_ = aBinding
	_ = bundle
	_ = ctx
}

// installIAMIdentityBindings deliberately represents a trusted Fabric-only
// verified enrollment fixture: production has NO automatic subject/UUID
// enrollment mechanism yet. In particular, no username is converted to sub.
func installIAMIdentityBindings(t *testing.T,r *TenantBundleReconciler,tenantName,tenantID,issuer string, identities ...authentik.IdentityBinding) {
	t.Helper()
	ctx:=context.Background()
	binding:=openFGAModelTestBinding(t,r,tenantID)
	raw,err:=json.Marshal(identities)
	if err!=nil {t.Fatal(err)}
	binding.Data[identityBindingsKey]=string(raw)
	if err:=r.Update(ctx,&binding);err!=nil {t.Fatal(err)}
}

func makeIAMGroup(name,uuid string,users ...string) authentik.GroupMembership {
	if users==nil {users=[]string{}}
	sort.Strings(users)
	return authentik.GroupMembership{Name:name,AuthentikGroupUUID:uuid,ActiveAuthentikUserUUIDs:users}
}

func TestVerifiedIAMMembershipConvergesThenRevokesRemovedUser(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	issuer:=b.Spec.HumanAccess.Web.OIDC.Issuer
	installIAMIdentityBindings(t,r,b.Name,b.Spec.TenantID,issuer,
		authentik.IdentityBinding{TenantID:b.Spec.TenantID,Issuer:issuer,OIDCSubject:"hashed-sub-one",AuthentikUserUUID:identityLedgerUserUUID,FabricUserID:"usr000002"},
		authentik.IdentityBinding{TenantID:b.Spec.TenantID,Issuer:issuer,OIDCSubject:"hashed-sub-two",AuthentikUserUUID:"550e8400-e29b-41d4-a716-446655440002",FabricUserID:"usr000003"},
	)
	name:=authentikGroupName(b.Name,"sales")
	src:=&stagedIAMSource{
		groups:map[string]authentik.GroupMembership{name:makeIAMGroup(name,identityLedgerGroupUUID,identityLedgerUserUUID,"550e8400-e29b-41d4-a716-446655440002")},
		calls:map[string]int{},fail:map[string]error{},
	}
	pin:=map[string]string{name:identityLedgerGroupUUID}
	writer:=newIAMWriter()
	if err:=r.reconcileVerifiedIAMGroupMemberships(ctx,b,pin,src,writer);err!=nil {t.Fatal(err)}
	store,err:=r.readOpenFGAAuthorizationBinding(ctx,b)
	if err!=nil {t.Fatal(err)}
	expected:=[]openfga.Tuple{
		{User:"user:usr000002",Relation:"member",Object:"group:"+name},
		{User:"user:usr000003",Relation:"member",Object:"group:"+name},
	}
	if !reflect.DeepEqual(writer.at(store.Store.ID,name),expected) {t.Fatalf("wrong grants: %v",writer.at(store.Store.ID,name))}
	if len(writer.stores)!=1 || !writer.stores[store.Store.ID] ||
		len(writer.models)!=1 || !writer.models[store.ModelID] {
		t.Fatal("FGA writer did not use server-retained tenant store/model")
	}
	// Authentik removed a user: the next complete snapshot must revoke the
	// user:usr000002 membership, NOT leave it in the FGA relation.
	src.groups[name]=makeIAMGroup(name,identityLedgerGroupUUID,"550e8400-e29b-41d4-a716-446655440002")
	if err:=r.reconcileVerifiedIAMGroupMemberships(ctx,b,pin,src,writer);err!=nil {t.Fatal(err)}
	want:=[]openfga.Tuple{{User:"user:usr000003",Relation:"member",Object:"group:"+name}}
	if !reflect.DeepEqual(writer.at(store.Store.ID,name),want) {t.Fatalf("failed revocation: %v",writer.at(store.Store.ID,name))}
	// Legitimate EMPTY membership snapshot deletes all grants.
	src.groups[name]=makeIAMGroup(name,identityLedgerGroupUUID)
	if err:=r.reconcileVerifiedIAMGroupMemberships(ctx,b,pin,src,writer);err!=nil {t.Fatal(err)}
	if len(writer.at(store.Store.ID,name))!=0 {t.Fatal("empty verified group failed to revoke all members")}
}

func TestVerifiedIAMMembershipNeverWritesFromPartialUnmappedOrForgedInput(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	issuer:=b.Spec.HumanAccess.Web.OIDC.Issuer
	installIAMIdentityBindings(t,r,b.Name,b.Spec.TenantID,issuer,
		authentik.IdentityBinding{TenantID:b.Spec.TenantID,Issuer:issuer,OIDCSubject:"verified-hash",AuthentikUserUUID:identityLedgerUserUUID,FabricUserID:"usr000002"})
	name:=authentikGroupName(b.Name,"sales")
	good:=makeIAMGroup(name,identityLedgerGroupUUID,identityLedgerUserUUID)
	for _,tc:=range []struct{
		name string
		pin map[string]string
		snapshot authentik.GroupMembership
		err error
	}{
		{"missing-pin",nil,good,nil},
		{"pin-missing-entry",map[string]string{"txo-fabric-hairem-sales":identityLedgerGroupUUID},good,nil},
		{"group-recreated",map[string]string{name:identityLedgerGroupUUID},makeIAMGroup(name,"550e8400-e29b-41d4-a716-446655440019",identityLedgerUserUUID),nil},
		{"foreign-name",map[string]string{name:identityLedgerGroupUUID},makeIAMGroup("txo-fabric-hairem-sales",identityLedgerGroupUUID,identityLedgerUserUUID),nil},
		{"unmapped-member",map[string]string{name:identityLedgerGroupUUID},makeIAMGroup(name,identityLedgerGroupUUID,"550e8400-e29b-41d4-a716-446655440017"),nil},
		{"nil-members",map[string]string{name:identityLedgerGroupUUID},authentik.GroupMembership{Name:name,AuthentikGroupUUID:identityLedgerGroupUUID},nil},
		{"authentik-outage",map[string]string{name:identityLedgerGroupUUID},good,errors.New("token expired")},
	}{
		t.Run(tc.name,func(t *testing.T){
			source:=&stagedIAMSource{groups:map[string]authentik.GroupMembership{name:tc.snapshot},
				fail:map[string]error{},calls:map[string]int{}}
			if tc.err!=nil {source.fail[name]=tc.err}
			writer:=newIAMWriter()
			if err:=r.reconcileVerifiedIAMGroupMemberships(ctx,b,tc.pin,source,writer);err==nil {
				t.Fatal("incomplete/foreign IAM membership was accepted")
			}
			if len(writer.calls)!=0 {t.Fatal("untrusted membership reached FGA write API")}
		})
	}
	// A replaced TenantBundle UID cannot reuse the original tenant graph.
	replaced:=b.DeepCopy();replaced.UID="new-tenant-object"
	src:=&stagedIAMSource{groups:map[string]authentik.GroupMembership{name:good},calls:map[string]int{},fail:map[string]error{}}
	writer:=newIAMWriter()
	if err:=r.reconcileVerifiedIAMGroupMemberships(ctx,replaced,map[string]string{name:identityLedgerGroupUUID},src,writer);err==nil {
		t.Fatal("recreated tenant adopted old store and memberships")
	}
	if len(writer.calls)!=0 {t.Fatal("recreated tenant wrote old grants")}
}

func TestVerifiedIAMGroupPreflightIsAllOrNothingAndRereadsSource(t *testing.T) {
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	b.Spec.HumanAccess.Web.IAMGroups=[]string{"sales","admin"}
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	issuer:=b.Spec.HumanAccess.Web.OIDC.Issuer
	installIAMIdentityBindings(t,r,b.Name,b.Spec.TenantID,issuer,
		authentik.IdentityBinding{TenantID:b.Spec.TenantID,Issuer:issuer,OIDCSubject:"verified-hash",AuthentikUserUUID:identityLedgerUserUUID,FabricUserID:"usr000002"})
	sales:=authentikGroupName(b.Name,"sales")
	admin:=authentikGroupName(b.Name,"admin")
	adminUUID:="550e8400-e29b-41d4-a716-446655440011"
	pins:=map[string]string{sales:identityLedgerGroupUUID,admin:adminUUID}
	writer:=newIAMWriter()
	// Input for the second group is malformed. Even the valid first group
	// must NOT be sent to OpenFGA before every snapshot is verified.
	bad:=&stagedIAMSource{groups:map[string]authentik.GroupMembership{
		sales:makeIAMGroup(sales,identityLedgerGroupUUID,identityLedgerUserUUID),
		admin:makeIAMGroup(admin,adminUUID,"550e8400-e29b-41d4-a716-446655440018"),
	},calls:map[string]int{},fail:map[string]error{}}
	if err:=r.reconcileVerifiedIAMGroupMemberships(ctx,b,pins,bad,writer);err==nil {t.Fatal("accepted partial tenant IAM inventory")}
	if len(writer.calls)!=0 {t.Fatal("wrote partial IAM inventory")}
	// Source changes after the writes. This is NOT a successful complete
	// convergence, so the caller must deny until it revalidates/resyncs.
	goodAdmin:=makeIAMGroup(admin,adminUUID,identityLedgerUserUUID)
	changed:=&stagedIAMSource{
		groups:map[string]authentik.GroupMembership{sales:makeIAMGroup(sales,identityLedgerGroupUUID,identityLedgerUserUUID),admin:goodAdmin},
		after:map[string]authentik.GroupMembership{sales:makeIAMGroup(sales,identityLedgerGroupUUID)},
		calls:map[string]int{},fail:map[string]error{},
	}
	if err:=r.reconcileVerifiedIAMGroupMemberships(ctx,b,pins,changed,writer);err==nil {
		t.Fatal("claimed success after race against Authentik membership changes")
	}
	if len(writer.calls)!=2 {t.Fatalf("expected two scoped reconciliation attempts, got %d",len(writer.calls))}
	if writer.calls[0].Object!="group:"+admin || writer.calls[1].Object!="group:"+sales {
		t.Fatalf("unexpected stable group order: %v",writer.calls)
	}
}

func TestVerifiedIAMGroupWriterOutageFailsAndForeignTenantCannotCrossStore(t *testing.T) {
	ctx:=context.Background()
	a:=humanIdentityTenant("hairem","TEN00001","uid-a","client0")
	b:=humanIdentityTenant("indiba","TEN00002","uid-b","sales")
	r:=preparedHumanIdentityReconciler(t,a,b)
	for _,bundle:=range []*fabricv1alpha1.TenantBundle{a,b} {
		if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,bundle);err!=nil {t.Fatal(err)}
		issuer:=bundle.Spec.HumanAccess.Web.OIDC.Issuer
		installIAMIdentityBindings(t,r,bundle.Name,bundle.Spec.TenantID,issuer,
			authentik.IdentityBinding{TenantID:bundle.Spec.TenantID,Issuer:issuer,OIDCSubject:"signed-sub-"+bundle.Name,
				AuthentikUserUUID:identityLedgerUserUUID,FabricUserID:"usr000002"})
	}
	writer:=newIAMWriter()
	writer.failFor="group:txo-fabric-indiba-sales"
	for _,bundle:=range []*fabricv1alpha1.TenantBundle{a,b} {
		key:=bundle.Spec.HumanAccess.Web.IAMGroups[0]
		name:=authentikGroupName(bundle.Name,key)
		src:=&stagedIAMSource{groups:map[string]authentik.GroupMembership{name:makeIAMGroup(name,identityLedgerGroupUUID,identityLedgerUserUUID)},
			calls:map[string]int{},fail:map[string]error{}}
		err:=r.reconcileVerifiedIAMGroupMemberships(ctx,bundle,map[string]string{name:identityLedgerGroupUUID},src,writer)
		if bundle.Name=="indiba" && err==nil {t.Fatal("accepted OpenFGA write failure")}
		if bundle.Name=="hairem" && err!=nil {t.Fatal(err)}
	}
	storeA,err:=r.readOpenFGAAuthorizationBinding(ctx,a);if err!=nil {t.Fatal(err)}
	storeB,err:=r.readOpenFGAAuthorizationBinding(ctx,b);if err!=nil {t.Fatal(err)}
	if storeA.Store.ID==storeB.Store.ID {t.Fatal("cross-tenant store ID collision")}
	if len(writer.at(storeA.Store.ID,authentikGroupName(a.Name,"client0")))!=1 ||
		len(writer.at(storeB.Store.ID,authentikGroupName(b.Name,"sales")))!=0 {
		t.Fatal("cross-tenant grant leakage or ignored upstream failure")
	}
}
