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
	failAccount error
	accountCalls int
}

func (s *freshAccountSource) SnapshotUserAccount(_ context.Context, uuid string) (authentik.UserAccountSnapshot, error) {
	s.accountCalls++
	if s.failAccount != nil { return authentik.UserAccountSnapshot{}, s.failAccount }
	return authentik.UserAccountSnapshot{UUID:uuid, Active:!s.inactive}, nil
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
			if until!="" || reason!=tc.want || len(w.calls)!=0 {
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
