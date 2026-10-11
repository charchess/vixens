package controller

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/openfga"
)

type freshAccountSource struct {
	*stagedIAMSource
	inactive bool
	inactiveUUIDs map[string]bool
	badAccountUUID string
	inactiveAfter int
	failAccount error
	failAccountAfter int
	accountCalls int
}

func (s *freshAccountSource) SnapshotUserAccount(_ context.Context, uuid string) (authentik.UserAccountSnapshot, error) {
	s.accountCalls++
	if s.failAccount != nil { return authentik.UserAccountSnapshot{}, s.failAccount }
	if s.failAccountAfter > 0 && s.accountCalls >= s.failAccountAfter {return authentik.UserAccountSnapshot{}, errors.New("live IAM account read failed")}
	if s.badAccountUUID!="" {return authentik.UserAccountSnapshot{UUID:s.badAccountUUID,Active:false},nil}
	return authentik.UserAccountSnapshot{UUID:uuid, Active:!s.inactive && !s.inactiveUUIDs[uuid] && (s.inactiveAfter==0 || s.accountCalls<s.inactiveAfter)}, nil
}

func readyFreshFixture(t *testing.T) (*TenantBundleReconciler, *fabricv1alpha1.TenantBundle, *freshAccountSource, *recordingIAMWriter) {
	t.Helper()
	ctx:=context.Background()
	b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
	r:=preparedHumanIdentityReconciler(t,b)
	if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,b);err!=nil {t.Fatal(err)}
	installIAMIdentityBindings(t,r,b.Spec.TenantID,authentik.IdentityBinding{
		TenantID:b.Spec.TenantID,Issuer:b.Spec.HumanAccess.Web.OIDC.Issuer,
		OIDCSubject:"legacy-issuer-pinned-opaque-sub",AuthentikUserUUID:identityLedgerUserUUID,
		FabricUserID:"usr000002",
	})
	name:=authentikGroupName(b.Name,"sales")
	s:=&freshAccountSource{stagedIAMSource:&stagedIAMSource{
		groups:map[string]authentik.GroupMembership{
			name:makeIAMGroup(name,identityLedgerGroupUUID,identityLedgerUserUUID),
		},calls:map[string]int{},fail:map[string]error{},
	}}
	if err:=r.reconcileOpenFGAIAMGroupPins(ctx,b,s);err!=nil {t.Fatal(err)}
	return r,b,s,newIAMWriter()
}

func tenantMemberTuples(w *recordingIAMWriter, store, tenantID string) []openfga.Tuple {
	return w.current[store+"/tenant:"+tenantID+"/member"]
}

func TestFreshTenantMembershipReconcilesAndRevokesWithoutChangingLedger(t *testing.T) {
	ctx:=context.Background()
	r,b,s,w:=readyFreshFixture(t)
	before:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	approved,err:=r.readOpenFGAAuthorizationBinding(ctx,b)
	if err!=nil {t.Fatal(err)}
	until,reason:=r.reconcileFreshTenantMembers(ctx,b,s,w)
	if reason!="" || !AuthorizationLeaseFresh(until,time.Now()) {t.Fatalf("fresh membership denied: %q %s",reason,until)}
	expected:=[]openfga.Tuple{{Object:"tenant:"+b.Spec.TenantID,Relation:"member",User:"user:usr000002"}}
	if !reflect.DeepEqual(tenantMemberTuples(w,approved.Store.ID,b.Spec.TenantID),expected) {
		t.Fatalf("wrong canonical tenant membership: %#v",w.current)
	}
	if len(w.stores)!=1 || !w.stores[approved.Store.ID] || !w.models[approved.ModelID] {
		t.Fatal("membership crossed server-pinned store/model")
	}
	// Repeat: the persisted identity ledger and pinned issuer remain intact;
	// the production OpenFGA writer makes no HTTP mutation for an exact set.
	until,reason=r.reconcileFreshTenantMembers(ctx,b,s,w)
	if reason!="" || !AuthorizationLeaseFresh(until,time.Now()) {t.Fatal("unchanged snapshot stopped being fresh")}
	after:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	if before.ResourceVersion!=after.ResourceVersion || before.Data[identityBindingsKey]!=after.Data[identityBindingsKey] ||
		before.Data[identityIssuerKey]!=after.Data[identityIssuerKey] {
		t.Fatal("read-only membership verification rewrote the tenant ledger")
	}
	name:=authentikGroupName(b.Name,"sales")
	s.groups[name]=makeIAMGroup(name,identityLedgerGroupUUID)
	until,reason=r.reconcileFreshTenantMembers(ctx,b,s,w)
	if reason!="" || !AuthorizationLeaseFresh(until,time.Now()) {t.Fatalf("empty active group cannot revoke: %s",reason)}
	if len(tenantMemberTuples(w,approved.Store.ID,b.Spec.TenantID))!=0 ||
		len(w.at(approved.Store.ID,name))!=0 {
		t.Fatal("removed Authentik membership persisted in OpenFGA")
	}
}

func TestFreshTenantMembershipPreflightFailsClosed(t *testing.T) {
	tests:=[]struct {
		name string
		alter func(*freshAccountSource,*fabricv1alpha1.TenantBundle)
		want string
	}{
		{"account-inactive",func(s *freshAccountSource,_ *fabricv1alpha1.TenantBundle){s.inactive=true},"AccountInactive"},
		{"account-source-unavailable",func(s *freshAccountSource,_ *fabricv1alpha1.TenantBundle){s.failAccount=errors.New("offline")},"IAMSourceUnavailable"},
		{"account-identity-mismatch",func(s *freshAccountSource,_ *fabricv1alpha1.TenantBundle){s.badAccountUUID="550e8400-e29b-41d4-a716-446655440050"},"IAMSourceUnavailable"},
		{"group-source-unavailable",func(s *freshAccountSource,b *fabricv1alpha1.TenantBundle){s.fail[authentikGroupName(b.Name,"sales")]=errors.New("offline")},"IAMSourceUnavailable"},
		{"wrong-group-uuid",func(s *freshAccountSource,b *fabricv1alpha1.TenantBundle){n:=authentikGroupName(b.Name,"sales");s.groups[n]=makeIAMGroup(n,"550e8400-e29b-41d4-a716-446655440011",identityLedgerUserUUID)},"GroupPinInvalid"},
		{"unmapped-user",func(s *freshAccountSource,b *fabricv1alpha1.TenantBundle){n:=authentikGroupName(b.Name,"sales");s.groups[n]=makeIAMGroup(n,identityLedgerGroupUUID,"550e8400-e29b-41d4-a716-446655440011")},"IdentityLedgerInvalid"},
		{"duplicate-uuid",func(s *freshAccountSource,b *fabricv1alpha1.TenantBundle){n:=authentikGroupName(b.Name,"sales");s.groups[n]=makeIAMGroup(n,identityLedgerGroupUUID,identityLedgerUserUUID,identityLedgerUserUUID)},"IdentityLedgerInvalid"},
	}
	for _,tc:=range tests {
		t.Run(tc.name,func(t *testing.T){
			r,b,s,w:=readyFreshFixture(t)
			tc.alter(s,b)
			until,reason:=r.reconcileFreshTenantMembers(context.Background(),b,s,w)
			if until!="" || reason!=tc.want || len(w.calls)!=0 || len(w.revocations)!=0 {
				t.Fatalf("unsafe preflight %q: until=%s reason=%s calls=%v",tc.name,until,reason,w.calls)
			}
		})
	}
}

func TestFreshTenantMembershipRejectsStaleSnapshotOrPartialWrite(t *testing.T) {
	r,b,s,w:=readyFreshFixture(t)
	base:=time.Date(2026,10,11,15,0,0,0,time.UTC)
	calls:=0
	r.AuthorizationNow=func()time.Time {calls++;return base.Add(time.Duration(calls-1)*40*time.Second)}
	until,reason:=r.reconcileFreshTenantMembers(context.Background(),b,s,w)
	if until!="" || reason!="StaleSnapshot" || len(w.calls)!=0 {t.Fatal("stale preflight wrote FGA")}
	r.AuthorizationNow=nil
	// An upstream failure after group relation reconciliation cannot
	// publish a positive tenant readiness lease.
	w.failFor="tenant:"+b.Spec.TenantID
	until,reason=r.reconcileFreshTenantMembers(context.Background(),b,s,w)
	if until!="" || reason!="TupleMismatch" {t.Fatalf("partial FGA write accepted: %q %q",until,reason)}
	name:=authentikGroupName(b.Name,"sales")
	s.after=map[string]authentik.GroupMembership{name:makeIAMGroup(name,identityLedgerGroupUUID)}
	s.calls[name]=0 // first preflight read old snapshot, subsequent reads observe revocation
	until,reason=r.reconcileFreshTenantMembers(context.Background(),b,s,newIAMWriter())
	if until!="" || (reason!="StaleSnapshot" && reason!="TupleMismatch") {
		t.Fatalf("mid-reconcile membership revocation went unobserved: %q %q",until,reason)
	}
}

func TestFreshTenantMembershipIsolationAndLeaseExpiry(t *testing.T) {
	r,b,s,w:=readyFreshFixture(t)
	// A recreated tenant object, even with the same tenant ID, cannot
	// adopt another UID-bound identity or group/member grant.
	foreign:=b.DeepCopy()
	foreign.UID="recreated-tenant-object"
	until,reason:=r.reconcileFreshTenantMembers(context.Background(),foreign,s,w)
	if until!="" || reason=="" || len(w.calls)!=0 {t.Fatal("foreign tenant accessed retained ledger/store")}
	until,reason=r.reconcileFreshTenantMembers(context.Background(),b,nil,w)
	if until!="" || reason!="IAMCredentialMissing" {t.Fatal("missing IAM source granted membership")}
	until,reason=r.reconcileFreshTenantMembers(context.Background(),b,s,nil)
	if until!="" || reason!="OpenFGAUnavailable" {t.Fatal("missing tuple writer granted membership")}
	now:=time.Now().UTC()
	if !AuthorizationLeaseFresh(now.Add(authorizationLease).Format(time.RFC3339Nano),now) ||
		AuthorizationLeaseFresh(now.Add(-time.Second).Format(time.RFC3339Nano),now) ||
		AuthorizationLeaseFresh(now.Add(2*authorizationLease).Format(time.RFC3339Nano),now) ||
		AuthorizationLeaseFresh("invalid",now) {
		t.Fatal("authorization lease accepted expired, invented or malformed freshness evidence")
	}
	if strings.Contains(until,identityLedgerUserUUID) {t.Fatal("membership evidence leaked identity")}
}

func TestFreshTenantMembershipAccountDisabledDuringFGAWriteDeniesLease(t *testing.T) {
	r,b,s,w:=readyFreshFixture(t)
	s.inactiveAfter=2 // one active source read, then disabled on revalidation
	until,reason:=r.reconcileFreshTenantMembers(context.Background(),b,s,w)
	if until!="" || reason!="AccountInactive" {
		t.Fatalf("account revoked during FGA sync retained positive readiness: until=%q reason=%q",until,reason)
	}
}

func TestFreshTenantMembershipTwoTenantsSameAuthentikUUIDDifferentFabricUsers(t *testing.T) {
	ctx:=context.Background()
	a:=humanIdentityTenant("hairem","TEN00001","uid-hairem","client0")
	b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
	r:=preparedHumanIdentityReconciler(t,a,b)
	shared:=identityLedgerUserUUID
	type tenantFixture struct {
		b *fabricv1alpha1.TenantBundle
		pin string
		user string
		source *freshAccountSource
	}
	cases:=[]tenantFixture{
		{b:a,pin:identityLedgerGroupUUID,user:"usr000001"},
		{b:b,pin:"550e8400-e29b-41d4-a716-446655440019",user:"usr000002"},
	}
	for i:=range cases {
		tc:=&cases[i]
		if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,tc.b);err!=nil {t.Fatal(err)}
		installIAMIdentityBindings(t,r,tc.b.Spec.TenantID,authentik.IdentityBinding{
			TenantID:tc.b.Spec.TenantID,Issuer:tc.b.Spec.HumanAccess.Web.OIDC.Issuer,
			OIDCSubject:"opaque-local-legacy-sub",AuthentikUserUUID:shared,FabricUserID:tc.user,
		})
		name:=authentikGroupName(tc.b.Name,tc.b.Spec.HumanAccess.Web.IAMGroups[0])
		tc.source=&freshAccountSource{stagedIAMSource:&stagedIAMSource{
			groups:map[string]authentik.GroupMembership{name:makeIAMGroup(name,tc.pin,shared)},
			calls:map[string]int{},fail:map[string]error{},
		}}
		if err:=r.reconcileOpenFGAIAMGroupPins(ctx,tc.b,tc.source);err!=nil {t.Fatal(err)}
	}
	writer:=newIAMWriter()
	var stores []string
	for _,tc:=range cases {
		validUntil,reason:=r.reconcileFreshTenantMembers(ctx,tc.b,tc.source,writer)
		if reason!="" || !AuthorizationLeaseFresh(validUntil,time.Now()) {
			t.Fatalf("tenant %s cannot reconcile its own membership: %s",tc.b.Name,reason)
		}
		binding,err:=r.readOpenFGAAuthorizationBinding(ctx,tc.b)
		if err!=nil {t.Fatal(err)}
		stores=append(stores,binding.Store.ID)
		got:=tenantMemberTuples(writer,binding.Store.ID,tc.b.Spec.TenantID)
		want:=[]openfga.Tuple{{Object:"tenant:"+tc.b.Spec.TenantID,Relation:"member",User:"user:"+tc.user}}
		if !reflect.DeepEqual(got,want) {t.Fatalf("tenant %s wrong membership: %v",tc.b.Name,got)}
	}
	if stores[0]==stores[1] ||
		len(tenantMemberTuples(writer,stores[0],cases[1].b.Spec.TenantID))!=0 ||
		len(tenantMemberTuples(writer,stores[1],cases[0].b.Spec.TenantID))!=0 {
		t.Fatal("tenant membership contaminated a foreign store")
	}
}

func TestConfirmedInactiveAccountRevokesExactTwoScopesWithoutCollateral(t *testing.T) {
	ctx:=context.Background()
	r,b,s,w:=readyFreshFixture(t)
	otherUUID:="550e8400-e29b-41d4-a716-446655440002"
	otherID:="usr000003"
	affectedID:="usr000002"
	group:=authentikGroupName(b.Name,"sales")
	s.groups[group]=makeIAMGroup(group,identityLedgerGroupUUID,identityLedgerUserUUID,otherUUID)
	installIAMIdentityBindings(t,r,b.Spec.TenantID,
		authentik.IdentityBinding{TenantID:b.Spec.TenantID,Issuer:b.Spec.HumanAccess.Web.OIDC.Issuer,
			OIDCSubject:"legacy-sub-inactive",AuthentikUserUUID:identityLedgerUserUUID,FabricUserID:affectedID},
		authentik.IdentityBinding{TenantID:b.Spec.TenantID,Issuer:b.Spec.HumanAccess.Web.OIDC.Issuer,
			OIDCSubject:"legacy-sub-other",AuthentikUserUUID:otherUUID,FabricUserID:otherID},
	)
	ledgerBefore:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	approved,err:=r.readOpenFGAAuthorizationBinding(ctx,b)
	if err!=nil {t.Fatal(err)}
	store:=approved.Store.ID
	groupKey:=store+"/group:"+group+"/member"
	tenantKey:=store+"/tenant:"+b.Spec.TenantID+"/member"
	groupAffected:=openfga.Tuple{Object:"group:"+group,Relation:"member",User:"user:"+affectedID}
	groupOther:=openfga.Tuple{Object:"group:"+group,Relation:"member",User:"user:"+otherID}
	tenantAffected:=openfga.Tuple{Object:"tenant:"+b.Spec.TenantID,Relation:"member",User:"user:"+affectedID}
	tenantOther:=openfga.Tuple{Object:"tenant:"+b.Spec.TenantID,Relation:"member",User:"user:"+otherID}
	w.current[groupKey]=[]openfga.Tuple{groupAffected,groupOther}
	w.current[tenantKey]=[]openfga.Tuple{tenantAffected,tenantOther}
	foreignStore:="01H0H015178Y2V4CX10C2KGHF9"
	foreignKey:=foreignStore+"/tenant:TEN00001/member"
	foreign:=openfga.Tuple{Object:"tenant:TEN00001",Relation:"member",User:"user:"+affectedID}
	w.current[foreignKey]=[]openfga.Tuple{foreign}
	anotherGroupKey:=store+"/group:txo-fabric-foreign-sales/member"
	w.current[anotherGroupKey]=[]openfga.Tuple{{Object:"group:txo-fabric-foreign-sales",Relation:"member",User:"user:"+affectedID}}
	s.inactiveUUIDs=map[string]bool{identityLedgerUserUUID:true}
	for pass:=0;pass<2;pass++ {
		until,reason:=r.reconcileFreshTenantMembers(ctx,b,s,w)
		if until!="" || reason!="AccountInactive" {
			t.Fatalf("pass %d granted lease despite confirmed inactivation: until=%q reason=%q",pass,until,reason)
		}
		if !reflect.DeepEqual(w.current[groupKey],[]openfga.Tuple{groupOther}) ||
			!reflect.DeepEqual(w.current[tenantKey],[]openfga.Tuple{tenantOther}) {
			t.Fatalf("pass %d failed precise group and tenant revocation: %+v",pass,w.current)
		}
		if !reflect.DeepEqual(w.current[foreignKey],[]openfga.Tuple{foreign}) ||
			len(w.current[anotherGroupKey])!=1 {
			t.Fatal("revocation leaked into another tenant or unrelated group")
		}
		if len(w.revocations)!=2 {
			t.Fatalf("idempotent revocation made extra writes, pass=%d count=%d",pass,len(w.revocations))
		}
		if len(w.calls)!=0 {t.Fatal("full-scope reconciler should never grant on inactive snapshot")}
	}
	ledgerAfter:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	if ledgerBefore.ResourceVersion!=ledgerAfter.ResourceVersion ||
		ledgerBefore.Data[identityBindingsKey]!=ledgerAfter.Data[identityBindingsKey] ||
		ledgerBefore.Data[identityIssuerKey]!=ledgerAfter.Data[identityIssuerKey] {
		t.Fatal("account revocation rewrote immutable tenant identity provenance")
	}
}

func TestInactiveAccountSourceOutageNeverSpeculativelyDeletes(t *testing.T) {
	r,b,s,w:=readyFreshFixture(t)
	approved,err:=r.readOpenFGAAuthorizationBinding(context.Background(),b)
	if err!=nil {t.Fatal(err)}
	group:=authentikGroupName(b.Name,"sales")
	groupKey:=approved.Store.ID+"/group:"+group+"/member"
	tenantKey:=approved.Store.ID+"/tenant:"+b.Spec.TenantID+"/member"
	existingGroup:=openfga.Tuple{Object:"group:"+group,Relation:"member",User:"user:usr000002"}
	existingTenant:=openfga.Tuple{Object:"tenant:"+b.Spec.TenantID,Relation:"member",User:"user:usr000002"}
	w.current[groupKey]=[]openfga.Tuple{existingGroup}
	w.current[tenantKey]=[]openfga.Tuple{existingTenant}
	s.failAccount=errors.New("read timeout")
	until,reason:=r.reconcileFreshTenantMembers(context.Background(),b,s,w)
	if until!="" || reason!="IAMSourceUnavailable" || len(w.revocations)!=0 ||
		!reflect.DeepEqual(w.current[groupKey],[]openfga.Tuple{existingGroup}) ||
		!reflect.DeepEqual(w.current[tenantKey],[]openfga.Tuple{existingTenant}) {
		t.Fatalf("IAM outage caused speculative deletion or positive lease: %q %q %+v",until,reason,w.current)
	}
}

func TestInactiveAccountRevocationRequiresSecondAuthoritativeAccountRead(t *testing.T) {
	r,b,s,w:=readyFreshFixture(t)
	approved,err:=r.readOpenFGAAuthorizationBinding(context.Background(),b)
	if err!=nil {t.Fatal(err)}
	group:=authentikGroupName(b.Name,"sales")
	groupKey:=approved.Store.ID+"/group:"+group+"/member"
	tenantKey:=approved.Store.ID+"/tenant:"+b.Spec.TenantID+"/member"
	groupMember:=openfga.Tuple{Object:"group:"+group,Relation:"member",User:"user:usr000002"}
	tenantMember:=openfga.Tuple{Object:"tenant:"+b.Spec.TenantID,Relation:"member",User:"user:usr000002"}
	w.current[groupKey]=[]openfga.Tuple{groupMember}
	w.current[tenantKey]=[]openfga.Tuple{tenantMember}
	s.inactive=true
	s.failAccountAfter=2 // first inactive result succeeds, confirmation fails
	until,reason:=r.reconcileFreshTenantMembers(context.Background(),b,s,w)
	if until!="" || reason!="IAMSourceUnavailable" || len(w.revocations)!=0 ||
		!reflect.DeepEqual(w.current[groupKey],[]openfga.Tuple{groupMember}) ||
		!reflect.DeepEqual(w.current[tenantKey],[]openfga.Tuple{tenantMember}) {
		t.Fatalf("partial Authentik confirmation deleted member tuples: until=%s reason=%s",until,reason)
	}
}
