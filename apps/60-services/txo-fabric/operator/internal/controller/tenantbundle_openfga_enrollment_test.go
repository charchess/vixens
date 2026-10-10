package controller

import (
    "context"
    "errors"
    "regexp"
    "testing"

    fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
    "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
    corev1 "k8s.io/api/core/v1"
    apierrors "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

type enrollmentAccountSource struct {
    uuid string
    inactive bool
    afterInactive bool
    fail error
    calls int
}
func (s *enrollmentAccountSource) SnapshotUserAccount(_ context.Context,uuid string)(authentik.UserAccountSnapshot,error) {
    s.calls++
    if s.fail!=nil {return authentik.UserAccountSnapshot{},s.fail}
    return authentik.UserAccountSnapshot{UUID:s.uuid,Active:!s.inactive && !(s.afterInactive && s.calls>=2)},nil
}

func proofForHuman(bIssuer,subject,uuid string) trustedHumanOIDCProof {
    return trustedHumanOIDCProof{issuer:bIssuer,subject:subject,authentikUUID:uuid}
}

func setupEnrollment(t *testing.T)(*TenantBundleReconciler,*enrollmentAccountSource,trustedHumanOIDCProof) {
    t.Helper()
    r:=preparedHumanIdentityReconciler(t,
        humanIdentityTenant("hairem","TEN00001","uid-hairem","client0"),
        humanIdentityTenant("indiba","TEN00002","uid-indiba","sales"))
    b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
    a:=humanIdentityTenant("hairem","TEN00001","uid-hairem","client0")
    for _,tenant:=range []*fabricv1alpha1.TenantBundle{a,b} {
        if err:=r.reconcileOpenFGAHumanIdentityRegistry(context.Background(),tenant);err!=nil {t.Fatal(err)}
    }
    source:=&enrollmentAccountSource{uuid:identityLedgerUserUUID}
    proof:=proofForHuman(b.Spec.HumanAccess.Web.OIDC.Issuer,"opaque-hashed-sub-not-uuid",identityLedgerUserUUID)
    return r,source,proof
}

func TestVerifiedEnrollmentCASAllocatesOnceAndPreservesTenantIsolation(t *testing.T) {
    ctx:=context.Background()
    r,source,proof:=setupEnrollment(t)
    b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
    a:=humanIdentityTenant("hairem","TEN00001","uid-hairem","client0")

    tracker:=&countingClient{Client:r.Client}
    r.Client=tracker
    id,err:=r.enrollHumanFromVerifiedIdentity(ctx,b,proof,source)
    if err!=nil {t.Fatal(err)}
    if !regexp.MustCompile("^usr[0-9a-f]{32}$").MatchString(id) {
        t.Fatalf("allocated user is not an opaque immutable Fabric ID: %q",id)
    }
    if tracker.updates!=1 || source.calls!=2 {t.Fatalf("writes=%d source calls=%d",tracker.updates,source.calls)}
    registered,err:=r.readOpenFGAHumanIdentityRegistry(ctx,b)
    if err!=nil {t.Fatal(err)}
    got,err:=registered.ResolveOIDCSubject(proof.issuer,proof.subject)
    if err!=nil || got!=id {t.Fatalf("persisted identity not resolvable: %q %v",got,err)}
    tracker.updates=0
    repeated,err:=r.enrollHumanFromVerifiedIdentity(ctx,b,proof,source)
    if err!=nil || repeated!=id || tracker.updates!=0 {
        t.Fatalf("trusted repeated login rotated id or rewrote ledger: %s %v writes=%d",repeated,err,tracker.updates)
    }
    fresh:=&TenantBundleReconciler{Client:r.Client,Scheme:r.Scheme,OpenFGAStoreClient:r.OpenFGAStoreClient,OpenFGAModelClient:r.OpenFGAModelClient}
    existing,err:=fresh.enrollHumanFromVerifiedIdentity(ctx,b,proof,source)
    if err!=nil || existing!=id {t.Fatalf("restart lost Fabric identity %q: %v",existing,err)}
    other,err:=fresh.readOpenFGAHumanIdentityRegistry(ctx,a)
    if err!=nil {t.Fatal(err)}
    if _,err:=other.ResolveOIDCSubject(a.Spec.HumanAccess.Web.OIDC.Issuer,proof.subject);err==nil {
        t.Fatal("Indiba login enrolled hAIrem")
    }
    if _,err:=fresh.enrollHumanFromVerifiedIdentity(ctx,a,proof,source);err==nil {
        t.Fatal("foreign issuer enrolled cross-tenant")
    }
}

func TestVerifiedEnrollmentRejectsIdentityReplacementOrRebinding(t *testing.T) {
    ctx:=context.Background()
    r,source,proof:=setupEnrollment(t)
    b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
    id,err:=r.enrollHumanFromVerifiedIdentity(ctx,b,proof,source)
    if err!=nil {t.Fatal(err)}
    for _,mutate:=range []func(*trustedHumanOIDCProof){
        func(p *trustedHumanOIDCProof){p.subject="another-oidc-sub"},
        func(p *trustedHumanOIDCProof){p.authentikUUID="550e8400-e29b-41d4-a716-446655440002"},
        func(p *trustedHumanOIDCProof){p.issuer="https://authentik.truxonline.com/application/o/txo-fabric-hairem/"},
        func(p *trustedHumanOIDCProof){p.subject="alice@example.com\nadmin"},
        func(p *trustedHumanOIDCProof){p.authentikUUID="not-a-uuid"},
    } {
        modified:=proof
        mutate(&modified)
        if _,err:=r.enrollHumanFromVerifiedIdentity(ctx,b,modified,source);err==nil {
            t.Fatalf("identity replacement/spoof was accepted: %#v",modified)
        }
    }
    current,err:=r.readOpenFGAHumanIdentityRegistry(ctx,b)
    if err!=nil {t.Fatal(err)}
    if got,err:=current.ResolveOIDCSubject(proof.issuer,proof.subject);err!=nil || got!=id {
        t.Fatalf("failed replacement corrupted immutable identity: %q %v",got,err)
    }
}

func TestVerifiedEnrollmentDeniesDisabledOutageOrChangingAccountBeforeWrites(t *testing.T) {
    for _,tc:=range []struct{
        name string
        alter func(*enrollmentAccountSource)
    }{
        {"disabled",func(s *enrollmentAccountSource){s.inactive=true}},
        {"missing-or-replaced-UUID",func(s *enrollmentAccountSource){s.uuid="550e8400-e29b-41d4-a716-446655440002"}},
        {"api-outage",func(s *enrollmentAccountSource){s.fail=errors.New("Authentik unavailable")}},
    } {
        t.Run(tc.name,func(t *testing.T){
            r,source,proof:=setupEnrollment(t)
            tc.alter(source)
            b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
            tracker:=&countingClient{Client:r.Client}
            r.Client=tracker
            if _,err:=r.enrollHumanFromVerifiedIdentity(context.Background(),b,proof,source);err==nil {
                t.Fatal("trusted enrollment accepted invalid account")
            }
            if tracker.updates!=0 {t.Fatalf("unexpected identity enrollment writes %d",tracker.updates)}
        })
    }
}

func TestVerifiedEnrollmentRejectsTamperedRetainedBindingAndRecreatedTenant(t *testing.T) {
    ctx:=context.Background()
    r,source,proof:=setupEnrollment(t)
    b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
    other:=b.DeepCopy()
    other.UID="recreated-tenant"
    if _,err:=r.enrollHumanFromVerifiedIdentity(ctx,other,proof,source);err==nil {
        t.Fatal("recreated tenant adopted previous private identity ledger")
    }
    cm:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
    cm.Data["modelFingerprint"]="tampered"
    if err:=r.Update(ctx,&cm);err!=nil {t.Fatal(err)}
    if _,err:=r.enrollHumanFromVerifiedIdentity(ctx,b,proof,source);err==nil {
        t.Fatal("unapproved foreign model allowed enrollment")
    }
}

type enrollmentConflictClient struct{client.Client}
func (c *enrollmentConflictClient) Update(ctx context.Context,obj client.Object,opts ...client.UpdateOption)error {
    if _,ok:=obj.(*corev1.ConfigMap);ok {
        return apierrors.NewConflict(schema.GroupResource{Resource:"configmaps"},obj.GetName(),errors.New("another writer updated this tenant ledger"))
    }
    return c.Client.Update(ctx,obj,opts...)
}

func TestVerifiedEnrollmentCASConflictNeverOverwritesAnotherWriter(t *testing.T) {
    ctx:=context.Background()
    r,source,proof:=setupEnrollment(t)
    b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
    r.Client=&enrollmentConflictClient{Client:r.Client}
    if _,err:=r.enrollHumanFromVerifiedIdentity(ctx,b,proof,source);err==nil || !apierrors.IsConflict(errors.Unwrap(err)) {
        // A wrapped k8s Conflict means caller can re-read and retry rather
        // than silently overwriting another actor's new identity binding.
        t.Fatalf("expected wrapped optimistic concurrency conflict, got %v",err)
    }
    registry,err:=r.readOpenFGAHumanIdentityRegistry(ctx,b)
    if err!=nil {t.Fatal(err)}
    if _,err:=registry.ResolveOIDCSubject(proof.issuer,proof.subject);err==nil {
        t.Fatal("CAS conflict nevertheless enrolled the user")
    }
}

func TestVerifiedEnrollmentReportsRecheckFailureAfterWriteWithoutGranting(t *testing.T) {
    ctx:=context.Background()
    r,source,proof:=setupEnrollment(t)
    b:=humanIdentityTenant("indiba","TEN00002","uid-indiba","sales")
    source.afterInactive=true
    if _,err:=r.enrollHumanFromVerifiedIdentity(ctx,b,proof,source);err==nil {
        t.Fatal("user deactivated during enrollment but operation reported success")
    }
    // The binding may be persisted before the source's deactivation becomes
    // visible. It is NEVER a grant, and the future BFF must independently
    // verify active account+fresh tuple Check on every sensitive operation.
    cm:=openFGAModelTestBinding(t,r,b.Spec.TenantID)
    if cm.Data[identityBindingsKey]=="[]" {t.Fatal("test did not exercise post-write account recheck")}
}
