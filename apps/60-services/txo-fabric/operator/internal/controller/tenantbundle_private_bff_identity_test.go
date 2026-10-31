package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func markPrivateMembershipReady(t *testing.T,r *TenantBundleReconciler,b *fabricv1alpha1.TenantBundle,now time.Time) {
	t.Helper()
	var saved fabricv1alpha1.TenantBundle
	if err:=r.Get(context.Background(),types.NamespacedName{Name:b.Name},&saved);err!=nil {t.Fatal(err)}
	saved.Status.ObservedGeneration=saved.Generation
	saved.Status.Conditions=[]metav1.Condition{{
		Type:"OpenFGAAuthorizationReady", Status:metav1.ConditionTrue,
		Reason:"TenantMembershipReconciled",ObservedGeneration:saved.Generation,
		LastTransitionTime:metav1.NewTime(now),
		Message:"validUntil="+now.Add(45*time.Second).UTC().Format(time.RFC3339Nano)+"; tenant membership only; agent grants not included",
	}}
	if err:=r.Update(context.Background(),&saved);err!=nil {t.Fatal(err)}
	*b=*saved.DeepCopy()
}

func TestPrivateAuthorizationConditionFreshFailsClosed(t *testing.T) {
	now:=time.Date(2026,10,11,20,0,0,0,time.UTC)
	good:=metav1.Condition{Type:"OpenFGAAuthorizationReady",Status:metav1.ConditionTrue,
		Reason:"TenantMembershipReconciled",ObservedGeneration:5,
		Message:"validUntil="+now.Add(40*time.Second).Format(time.RFC3339Nano)+"; tenant membership only"}
	if !privateAuthorizationConditionFresh([]metav1.Condition{good},5,now) {t.Fatal("fresh condition rejected")}
	for _,tc:=range []struct{name string; mutate func(*metav1.Condition)}{
		{"false",func(c *metav1.Condition){c.Status=metav1.ConditionFalse}},
		{"unknown",func(c *metav1.Condition){c.Status=metav1.ConditionUnknown}},
		{"old-generation",func(c *metav1.Condition){c.ObservedGeneration=4}},
		{"bad-reason",func(c *metav1.Condition){c.Reason="RolloutDisabled"}},
		{"no-lease",func(c *metav1.Condition){c.Message="authorization ready"}},
		{"expired",func(c *metav1.Condition){c.Message="validUntil="+now.Add(-time.Second).Format(time.RFC3339Nano)+"; membership"}},
		{"implausible-future",func(c *metav1.Condition){c.Message="validUntil="+now.Add(2*time.Minute).Format(time.RFC3339Nano)+"; membership"}},
		{"bad-time",func(c *metav1.Condition){c.Message="validUntil=notatime; membership"}},
	}{
		t.Run(tc.name,func(t *testing.T){
			c:=good;tc.mutate(&c)
			if privateAuthorizationConditionFresh([]metav1.Condition{c},5,now) {t.Fatal("accepted untrusted freshness")}
		})
	}
	if privateAuthorizationConditionFresh(nil,5,now) ||
		privateAuthorizationConditionFresh([]metav1.Condition{good},6,now) {
		t.Fatal("missing/stale authorization accepted")
	}
}

func TestPrivateBFFIdentityUsesExistingTenantIssuerPinnedUUIDOnly(t *testing.T) {
	ctx:=context.Background()
	now:=time.Date(2026,10,11,20,0,0,0,time.UTC)
	a:=humanIdentityTenant("hairem","TEN00001","uid-hairem","client0")
	b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
	a.Generation=5;b.Generation=6
	r:=preparedHumanIdentityReconciler(t,a,b)
	r.AuthorizationNow=func()time.Time{return now}
	uuid:=identityLedgerUserUUID
	aGroup:=authentikGroupName(a.Name,"client0")
	bGroup:=authentikGroupName(b.Name,"sales")
	source:=&freshAccountSource{stagedIAMSource:&stagedIAMSource{
		groups:map[string]authentik.GroupMembership{
			aGroup:makeIAMGroup(aGroup,identityLedgerGroupUUID,uuid),
			bGroup:makeIAMGroup(bGroup,"550e8400-e29b-41d4-a716-446655440019",uuid),
		},calls:map[string]int{},fail:map[string]error{},
	}}
	for _,tc:=range []struct{bundle *fabricv1alpha1.TenantBundle;user string}{
		{a,"usr000001"},{b,"usr000002"},
	}{
		if err:=r.reconcileOpenFGAHumanIdentityRegistry(ctx,tc.bundle);err!=nil {t.Fatal(err)}
		installIAMIdentityBindings(t,r,tc.bundle.Spec.TenantID,authentik.IdentityBinding{
			TenantID:tc.bundle.Spec.TenantID,Issuer:tc.bundle.Spec.HumanAccess.Web.OIDC.Issuer,
			OIDCSubject:"legacy-local-sub-not-global",AuthentikUserUUID:uuid,FabricUserID:tc.user,
		})
		if err:=r.reconcileOpenFGAIAMGroupPins(ctx,tc.bundle,source);err!=nil {t.Fatal(err)}
		markPrivateMembershipReady(t,r,tc.bundle,now)
	}
	r.AuthentikMembershipClient=source
	ledgerA:=openFGAModelTestBinding(t,r,a.Spec.TenantID)
	ledgerB:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
	for _,tc:=range []struct{tenant,user string}{
		{"hairem","usr000001"},{"indiba","usr000002"},
	}{
		id,err:=r.ResolvePrivateBFFIdentity(ctx,tc.tenant,uuid,now)
		if err!=nil || id!=tc.user {t.Fatalf("%s identity mapping wrong: %q %v",tc.tenant,id,err)}
	}
	for _,tenant:=range []string{"unknown","hairem","indiba"} {
		id,err:=r.ResolvePrivateBFFIdentity(ctx,tenant,"550e8400-e29b-41d4-a716-446655440011",now)
		if err==nil || id!="" {t.Fatalf("enumerated identity for tenant %s",tenant)}
	}
	if after:=openFGAModelTestBinding(t,r,a.Spec.TenantID);after.Data[identityBindingsKey]!=ledgerA.Data[identityBindingsKey] ||
		after.Data[identityIssuerKey]!=ledgerA.Data[identityIssuerKey] ||
		after.ResourceVersion!=ledgerA.ResourceVersion {
		t.Fatal("private global login changed historical tenant A provenance")
	}
	if after:=openFGAModelTestBinding(t,r,b.Spec.TenantID);after.Data[identityBindingsKey]!=ledgerB.Data[identityBindingsKey] ||
		after.Data[identityIssuerKey]!=ledgerB.Data[identityIssuerKey] ||
		after.ResourceVersion!=ledgerB.ResourceVersion {
		t.Fatal("private global login changed historical tenant B provenance")
	}
	// Existing ledger is present, but active accounts/group and freshness
	// remain independent mandatory gates, including on auth provider outage.
	source.inactive=true
	if id,err:=r.ResolvePrivateBFFIdentity(ctx,a.Name,uuid,now);err==nil || id!="" {t.Fatal("inactive Authentik account resolved")}
	source.inactive=false
	source.failAccount=errors.New("authentik outage")
	if id,err:=r.ResolvePrivateBFFIdentity(ctx,a.Name,uuid,now);err==nil || id!="" {t.Fatal("IAM outage resolved")}
	source.failAccount=nil
	source.fail[aGroup]=errors.New("groups outage")
	if id,err:=r.ResolvePrivateBFFIdentity(ctx,a.Name,uuid,now);err==nil || id!="" {t.Fatal("incomplete group source resolved")}
	source.fail[aGroup]=nil
	source.groups[aGroup]=makeIAMGroup(aGroup,identityLedgerGroupUUID) // user removed from group
	if id,err:=r.ResolvePrivateBFFIdentity(ctx,a.Name,uuid,now);err==nil || id!="" {t.Fatal("revoked tenant membership resolved")}
	source.groups[aGroup]=makeIAMGroup(aGroup,identityLedgerGroupUUID,uuid)
	// First group inventory is complete, but IAM revokes the user while
	// the resolver is re-checking it. A torn snapshot MUST NOT resolve.
	source.after=map[string]authentik.GroupMembership{
		aGroup:makeIAMGroup(aGroup,identityLedgerGroupUUID),
	}
	source.calls[aGroup]=0
	if id,err:=r.ResolvePrivateBFFIdentity(ctx,a.Name,uuid,now);err==nil || id!="" {
		t.Fatal("group membership revoked during resolution still granted identity")
	}
	source.after=nil
	source.calls[aGroup]=0
	source.groups[aGroup]=makeIAMGroup(aGroup,"550e8400-e29b-41d4-a716-446655440099",uuid)
	if id,err:=r.ResolvePrivateBFFIdentity(ctx,a.Name,uuid,now);err==nil || id!="" {
		t.Fatal("foreign Authentik group pin authorized tenant identity")
	}
	source.groups[aGroup]=makeIAMGroup(aGroup,identityLedgerGroupUUID,uuid)
	model:=r.OpenFGAModelClient
	r.OpenFGAModelClient=nil // missing platform FGA verifier credential
	if id,err:=r.ResolvePrivateBFFIdentity(ctx,a.Name,uuid,now);err==nil || id!="" {
		t.Fatal("unavailable OpenFGA model verifier authorized identity")
	}
	r.OpenFGAModelClient=model

	for _,tc:=range []struct{reason string;mutate func(*fabricv1alpha1.TenantBundle)}{
		{"condition-false",func(b *fabricv1alpha1.TenantBundle){b.Status.Conditions[0].Status=metav1.ConditionFalse}},
		{"observed-generation-stale",func(b *fabricv1alpha1.TenantBundle){b.Status.ObservedGeneration=3}},
		{"lease-expired",func(b *fabricv1alpha1.TenantBundle){b.Status.Conditions[0].Message="validUntil="+now.Add(-time.Second).Format(time.RFC3339Nano)+"; tenant membership only"}},
	}{
		t.Run(tc.reason,func(t *testing.T){
			var stored fabricv1alpha1.TenantBundle
			if err:=r.Get(ctx,typesNamespacedName(a.Name),&stored);err!=nil {t.Fatal(err)}
			tc.mutate(&stored)
			if err:=r.Update(ctx,&stored);err!=nil {t.Fatal(err)}
			if id,err:=r.ResolvePrivateBFFIdentity(ctx,a.Name,uuid,now);err==nil || id!="" {
				t.Fatal("bypassed stale/unavailable authorization condition")
			}
			markPrivateMembershipReady(t,r,&stored,now)
		})
	}
}

// Avoid handcrafting namespace fields: TenantBundle is cluster-scoped.
func typesNamespacedName(name string) types.NamespacedName {
	return types.NamespacedName{Name:name}
}

func TestPrivateBFFEnrollmentRequiresSignedSixFieldsAndFreshDeclaredGroup(t *testing.T) {
	ctx := context.Background()
	r, source, proof := setupEnrollment(t)
	r.AuthentikMembershipClient = source
	now := time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)

	id, err := r.EnrollPrivateBFFIdentity(ctx, "indiba", proof.tenantIssuer, proof.tenantSubject,
		proof.authentikUUID, proof.globalIssuer, proof.globalSubject, now)
	if err != nil || id == "" {
		t.Fatalf("signed six-field proof for active declared member denied: %q %v", id, err)
	}

	for _, mutate := range []func(*trustedHumanOIDCProof){
		func(p *trustedHumanOIDCProof) { p.globalIssuer = "" },
		func(p *trustedHumanOIDCProof) { p.globalSubject = "" },
		func(p *trustedHumanOIDCProof) { p.tenantIssuer = "https://authentik.truxonline.com/application/o/foreign/" },
		func(p *trustedHumanOIDCProof) { p.tenantSubject = "" },
		func(p *trustedHumanOIDCProof) { p.authentikUUID = "" },
	} {
		modified := proof
		mutate(&modified)
		if got, err := r.EnrollPrivateBFFIdentity(ctx, "indiba", modified.tenantIssuer, modified.tenantSubject,
			modified.authentikUUID, modified.globalIssuer, modified.globalSubject, now); err == nil || got != "" {
			t.Fatalf("incomplete or tenant-mismatched bridge proof enrolled: %#v", modified)
		}
	}

	group := authentikGroupName("indiba", "sales")
	source.groups[group] = makeIAMGroup(group, identityLedgerGroupUUID)
	if got, err := r.EnrollPrivateBFFIdentity(ctx, "indiba", proof.tenantIssuer, proof.tenantSubject,
		proof.authentikUUID, proof.globalIssuer, proof.globalSubject, now); err == nil || got != "" {
		t.Fatal("active account removed from declared tenant group enrolled")
	}
}
