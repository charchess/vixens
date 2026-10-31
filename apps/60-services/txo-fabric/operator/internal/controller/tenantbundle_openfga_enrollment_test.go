package controller

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/authentik"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type enrollmentAccountSource struct {
	uuid          string
	inactive      bool
	afterInactive bool
	fail          error
	calls         int
	groups        map[string]authentik.GroupMembership
}

func (s *enrollmentAccountSource) SnapshotUserAccount(_ context.Context, uuid string) (authentik.UserAccountSnapshot, error) {
	s.calls++
	if s.fail != nil {
		return authentik.UserAccountSnapshot{}, s.fail
	}
	return authentik.UserAccountSnapshot{UUID: s.uuid, Active: !s.inactive && !(s.afterInactive && s.calls >= 2)}, nil
}

func (s *enrollmentAccountSource) SnapshotGroupMembership(_ context.Context, name string) (authentik.GroupMembership, error) {
	group, ok := s.groups[name]
	if !ok {
		return authentik.GroupMembership{}, errors.New("group unavailable")
	}
	return group, nil
}

func proofForHuman(globalIssuer, globalSubject, uuid string) trustedHumanOIDCProof {
	return trustedHumanOIDCProof{globalIssuer: globalIssuer, globalSubject: globalSubject, tenantIssuer: "https://authentik.truxonline.com/application/o/txo-fabric-indiba/", tenantSubject: "opaque-hashed-sub-not-uuid", authentikUUID: uuid}
}

func setupEnrollment(t *testing.T) (*TenantBundleReconciler, *enrollmentAccountSource, trustedHumanOIDCProof) {
	t.Helper()
	a := humanIdentityTenant("hairem", "TEN00001", "uid-hairem", "client0")
	b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
	r := preparedHumanIdentityReconciler(t, a, b)
	for _, tenant := range []*fabricv1alpha1.TenantBundle{a, b} {
		if err := r.reconcileOpenFGAHumanIdentityRegistry(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	source := &enrollmentAccountSource{
		uuid: identityLedgerUserUUID,
		groups: map[string]authentik.GroupMembership{
			authentikGroupName("indiba", "sales"): makeIAMGroup(authentikGroupName("indiba", "sales"), identityLedgerGroupUUID, identityLedgerUserUUID),
		},
	}
	if err := r.reconcileOpenFGAIAMGroupPins(context.Background(), b, source); err != nil {
		t.Fatal(err)
	}
	proof := proofForHuman("https://authentik.truxonline.com/application/o/txo-fabric-webui/", "opaque-global-sub-not-uuid", identityLedgerUserUUID)
	return r, source, proof
}

func TestVerifiedEnrollmentCASAllocatesOnceAndPreservesTenantIsolation(t *testing.T) {
	ctx := context.Background()
	r, source, proof := setupEnrollment(t)
	b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
	var a fabricv1alpha1.TenantBundle
	if err := r.Get(ctx, types.NamespacedName{Name: "hairem"}, &a); err != nil {
		t.Fatal(err)
	}

	tracker := &countingClient{Client: r.Client}
	r.Client = tracker
	id, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile("^usr[0-9a-f]{32}$").MatchString(id) {
		t.Fatalf("allocated user is not an opaque immutable Fabric ID: %q", id)
	}
	if tracker.updates != 1 || source.calls != 2 {
		t.Fatalf("writes=%d source calls=%d", tracker.updates, source.calls)
	}
	registered, err := r.readOpenFGAHumanIdentityRegistry(ctx, b)
	if err != nil {
		t.Fatal(err)
	}
	got, err := registered.ResolveOIDCSubject(b.Spec.HumanAccess.Web.OIDC.Issuer, "opaque-hashed-sub-not-uuid")
	if err != nil || got != id {
		t.Fatalf("persisted identity not resolvable: %q %v", got, err)
	}
	tracker.updates = 0
	repeated, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source)
	if err != nil || repeated != id || tracker.updates != 0 {
		t.Fatalf("trusted repeated login rotated id or rewrote ledger: %s %v writes=%d", repeated, err, tracker.updates)
	}
	fresh := &TenantBundleReconciler{Client: r.Client, Scheme: r.Scheme, OpenFGAStoreClient: r.OpenFGAStoreClient, OpenFGAModelClient: r.OpenFGAModelClient}
	existing, err := fresh.enrollHumanFromVerifiedIdentity(ctx, b, proof, source)
	if err != nil || existing != id {
		t.Fatalf("restart lost Fabric identity %q: %v", existing, err)
	}
	other, err := fresh.readOpenFGAHumanIdentityRegistry(ctx, &a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.ResolveOIDCSubject(a.Spec.HumanAccess.Web.OIDC.Issuer, "opaque-hashed-sub-not-uuid"); err == nil {
		t.Fatal("Indiba login enrolled hAIrem")
	}
	if _, err := fresh.enrollHumanFromVerifiedIdentity(ctx, &a, proof, source); err == nil {
		t.Fatal("foreign issuer enrolled cross-tenant")
	}
}

func TestVerifiedEnrollmentRejectsIncompleteOrTenantMismatchedProof(t *testing.T) {
	ctx := context.Background()
	for _, mutate := range []func(*trustedHumanOIDCProof){
		func(p *trustedHumanOIDCProof) { p.globalIssuer = "" },
		func(p *trustedHumanOIDCProof) { p.globalSubject = "" },
		func(p *trustedHumanOIDCProof) { p.tenantSubject = "" },
		func(p *trustedHumanOIDCProof) {
			p.tenantIssuer = "https://authentik.truxonline.com/application/o/foreign-tenant/"
		},
		func(p *trustedHumanOIDCProof) { p.authentikUUID = "not-a-uuid" },
	} {
		r, source, proof := setupEnrollment(t)
		modified := proof
		mutate(&modified)
		b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
		if _, err := r.enrollHumanFromVerifiedIdentity(ctx, b, modified, source); err == nil {
			t.Fatalf("incomplete or tenant-mismatched verified proof was accepted: %#v", modified)
		}
	}
}

func TestVerifiedEnrollmentDeniesDisabledOutageOrChangingAccountBeforeWrites(t *testing.T) {
	for _, tc := range []struct {
		name  string
		alter func(*enrollmentAccountSource)
	}{
		{"disabled", func(s *enrollmentAccountSource) { s.inactive = true }},
		{"missing-or-replaced-UUID", func(s *enrollmentAccountSource) { s.uuid = "550e8400-e29b-41d4-a716-446655440002" }},
		{"api-outage", func(s *enrollmentAccountSource) { s.fail = errors.New("Authentik unavailable") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, source, proof := setupEnrollment(t)
			tc.alter(source)
			b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
			tracker := &countingClient{Client: r.Client}
			r.Client = tracker
			if _, err := r.enrollHumanFromVerifiedIdentity(context.Background(), b, proof, source); err == nil {
				t.Fatal("trusted enrollment accepted invalid account")
			}
			if tracker.updates != 0 {
				t.Fatalf("unexpected identity enrollment writes %d", tracker.updates)
			}
		})
	}
}

func TestVerifiedEnrollmentRejectsTamperedRetainedBindingAndRecreatedTenant(t *testing.T) {
	ctx := context.Background()
	r, source, proof := setupEnrollment(t)
	b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
	other := b.DeepCopy()
	other.UID = "recreated-tenant"
	if _, err := r.enrollHumanFromVerifiedIdentity(ctx, other, proof, source); err == nil {
		t.Fatal("recreated tenant adopted previous private identity ledger")
	}
	cm := openFGAModelTestBinding(t, r, b.Spec.TenantID)
	cm.Data["modelFingerprint"] = "tampered"
	if err := r.Update(ctx, &cm); err != nil {
		t.Fatal(err)
	}
	if _, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source); err == nil {
		t.Fatal("unapproved foreign model allowed enrollment")
	}
}

func TestVerifiedEnrollmentPersistsOnlyDeclaredTenantTupleAndIsByteIdempotent(t *testing.T) {
	ctx := context.Background()
	r, source, proof := setupEnrollment(t)
	b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
	id, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source)
	if err != nil {
		t.Fatal(err)
	}
	before := openFGAModelTestBinding(t, r, b.Spec.TenantID)
	var bindings []authentik.IdentityBinding
	if err := json.Unmarshal([]byte(before.Data[identityBindingsKey]), &bindings); err != nil || len(bindings) != 1 {
		t.Fatalf("invalid persisted tenant ledger: %v", err)
	}
	binding := bindings[0]
	if binding.FabricUserID != id || binding.Issuer != b.Spec.HumanAccess.Web.OIDC.Issuer ||
		binding.OIDCSubject != proof.tenantSubject ||
		binding.OIDCSubject == proof.globalSubject || binding.Issuer == proof.globalIssuer {
		t.Fatal("global OIDC proof leaked into the tenant issuer-pinned ledger")
	}
	if repeated, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source); err != nil || repeated != id {
		t.Fatalf("exact global proof was not idempotent: %q %v", repeated, err)
	}
	after := openFGAModelTestBinding(t, r, b.Spec.TenantID)
	if after.ResourceVersion != before.ResourceVersion || after.Data[identityBindingsKey] != before.Data[identityBindingsKey] {
		t.Fatal("idempotent enrollment rewrote tenant ledger bytes")
	}
}

func TestVerifiedEnrollmentRejectsLedgerIssuerMismatch(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		prefill bool
		mutate  func(*fabricv1alpha1.TenantBundle, *TenantBundleReconciler)
	}{
		{"wrong-retained-tenant-issuer", false, func(b *fabricv1alpha1.TenantBundle, r *TenantBundleReconciler) {
			cm := openFGAModelTestBinding(t, r, b.Spec.TenantID)
			cm.Data[identityIssuerKey] = "https://authentik.truxonline.com/application/o/foreign-tenant/"
			if err := r.Update(ctx, &cm); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, source, proof := setupEnrollment(t)
			b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
			if tc.prefill {
				if _, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source); err != nil {
					t.Fatal(err)
				}
			}
			tc.mutate(b, r)
			if _, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source); err == nil {
				t.Fatal("mismatched retained tenant issuer enrolled a proof")
			}
		})
	}
}

type enrollmentConflictClient struct{ client.Client }

func (c *enrollmentConflictClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if _, ok := obj.(*corev1.ConfigMap); ok {
		return apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"}, obj.GetName(), errors.New("another writer updated this tenant ledger"))
	}
	return c.Client.Update(ctx, obj, opts...)
}

func TestVerifiedEnrollmentCASConflictNeverOverwritesAnotherWriter(t *testing.T) {
	ctx := context.Background()
	r, source, proof := setupEnrollment(t)
	b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
	r.Client = &enrollmentConflictClient{Client: r.Client}
	if _, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source); err == nil || !apierrors.IsConflict(errors.Unwrap(err)) {
		// A wrapped k8s Conflict means caller can re-read and retry rather
		// than silently overwriting another actor's new identity binding.
		t.Fatalf("expected wrapped optimistic concurrency conflict, got %v", err)
	}
	registry, err := r.readOpenFGAHumanIdentityRegistry(ctx, b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.ResolveOIDCSubject(b.Spec.HumanAccess.Web.OIDC.Issuer, "opaque-hashed-sub-not-uuid"); err == nil {
		t.Fatal("CAS conflict nevertheless enrolled the user")
	}
}

func TestVerifiedEnrollmentReportsRecheckFailureAfterWriteWithoutGranting(t *testing.T) {
	ctx := context.Background()
	r, source, proof := setupEnrollment(t)
	b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
	source.afterInactive = true
	if _, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source); err == nil {
		t.Fatal("user deactivated during enrollment but operation reported success")
	}
	// The binding may be persisted before the source's deactivation becomes
	// visible. It is NEVER a grant, and the future BFF must independently
	// verify active account+fresh tuple Check on every sensitive operation.
	cm := openFGAModelTestBinding(t, r, b.Spec.TenantID)
	if cm.Data[identityBindingsKey] == "[]" {
		t.Fatal("test did not exercise post-write account recheck")
	}
}

func TestVerifiedEnrollmentUsesSignedTenantProofInsteadOfStaticTenantSubject(t *testing.T) {
	ctx := context.Background()
	r, source, proof := setupEnrollment(t)
	proof.tenantIssuer = "https://authentik.truxonline.com/application/o/txo-fabric-indiba/"
	proof.tenantSubject = "tenant-proof-subject-not-manual"
	var bundle fabricv1alpha1.TenantBundle
	if err := r.Get(ctx, types.NamespacedName{Name: "indiba"}, &bundle); err != nil {
		t.Fatal(err)
	}
	if err := r.Update(ctx, &bundle); err != nil {
		t.Fatal(err)
	}
	id, err := r.enrollHumanFromVerifiedIdentity(ctx, &bundle, proof, source)
	if err != nil {
		t.Fatalf("verified tenant proof was rejected: %v", err)
	}
	ledger, err := r.readOpenFGAHumanIdentityRegistry(ctx, &bundle)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ledger.ResolveOIDCSubject(proof.tenantIssuer, proof.tenantSubject)
	if err != nil || got != id {
		t.Fatalf("signed tenant subject was not the sole ledger binding: %q %v", got, err)
	}
}

func TestVerifiedEnrollmentRequiresFreshDeclaredGroupMembership(t *testing.T) {
	ctx := context.Background()
	r, source, proof := setupEnrollment(t)
	b := humanIdentityTenant("indiba", "TEN00002", "uid-indiba", "sales")
	group := authentikGroupName(b.Name, "sales")
	source.groups[group] = makeIAMGroup(group, identityLedgerGroupUUID)
	if _, err := r.enrollHumanFromVerifiedIdentity(ctx, b, proof, source); err == nil {
		t.Fatal("active account outside tenant declared group enrolled")
	}
}
