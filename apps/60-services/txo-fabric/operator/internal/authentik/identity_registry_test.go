package authentik

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const (
	registryIssuer = "https://authentik.truxonline.com/application/o/txo-fabric-indiba/"
	registryTenant = "TEN00002"
	registrySub1 = "a-hashed-sub-not-equal-to-user-uuid"
	registrySub2 = "b-another-opaque-sub"
)

func identityFixtures() []IdentityBinding {
	return []IdentityBinding{
		{registryTenant, registryIssuer, registrySub1, userOne, "usr000001"},
		{registryTenant, registryIssuer, registrySub2, userTwo, "usr000002"},
	}
}

func mustIdentityRegistry(t *testing.T) *IdentityRegistry {
	t.Helper()
	r, err := NewIdentityRegistry("indiba", registryTenant, registryIssuer, []string{"sales"}, identityFixtures())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestIdentityRegistryExactIssuerSubjectAndCanonicalUser(t *testing.T) {
	r := mustIdentityRegistry(t)
	user, err := r.ResolveOIDCSubject(registryIssuer, registrySub1)
	if err != nil || user != "usr000001" {
		t.Fatalf("lost verified principal mapping: %q %v", user, err)
	}
	for _, input := range []struct{ issuer, sub string }{
		{registryIssuer, userOne}, // UUID must not be confused with a hashed OIDC subject
		{registryIssuer, "edfoley"}, // login is not a subject
		{registryIssuer, registrySub1 + " "},
		{registryIssuer, registrySub1 + "\n"},
		{"https://attacker.example/application/o/txo-fabric-indiba/", registrySub1},
		{"https://authentik.truxonline.com/application/o/txo-fabric-hairem/", registrySub1},
	} {
		if got, err := r.ResolveOIDCSubject(input.issuer, input.sub); err == nil || got != "" {
			t.Fatalf("accepted foreign or mutable OIDC identity: %#v => %q", input, got)
		}
	}
	var nilRegistry *IdentityRegistry
	if got, err := nilRegistry.ResolveOIDCSubject(registryIssuer, registrySub1); err == nil || got != "" {
		t.Fatal("nil registry authorized a principal")
	}
}

func TestIdentityRegistryCompleteGroupSnapshotOrDeny(t *testing.T) {
	r := mustIdentityRegistry(t)
	snapshot := GroupMembership{
		Name: "txo-fabric-indiba-sales", AuthentikGroupUUID: groupUUID,
		ActiveAuthentikUserUUIDs: []string{userTwo, userOne},
	}
	got, err := r.ResolveGroupMembership(snapshot)
	if err != nil || !reflect.DeepEqual(got, []string{"usr000001", "usr000002"}) {
		t.Fatalf("incorrect group translation: %v %v", got, err)
	}
	// A changed complete snapshot projects only the remaining active member,
	// never a stale OIDC group claim. OpenFGA tuple removal is a later stage.
	snapshot.ActiveAuthentikUserUUIDs = []string{userTwo}
	got, err = r.ResolveGroupMembership(snapshot)
	if err != nil || !reflect.DeepEqual(got, []string{"usr000002"}) {
		t.Fatalf("membership removal lost: %v %v", got, err)
	}
	snapshot.ActiveAuthentikUserUUIDs = []string{}
	got, err = r.ResolveGroupMembership(snapshot)
	if err != nil || len(got) != 0 {
		t.Fatalf("valid empty group denied: %v %v", got, err)
	}

	for _, malformed := range []GroupMembership{
		{Name: "txo-fabric-indiba-sales", AuthentikGroupUUID: groupUUID, ActiveAuthentikUserUUIDs: nil},
		{Name: "txo-fabric-hairem-client0", AuthentikGroupUUID: groupUUID, ActiveAuthentikUserUUIDs: []string{userOne}},
		{Name: "txo-fabric-indiba-sales-admin", AuthentikGroupUUID: groupUUID, ActiveAuthentikUserUUIDs: []string{userOne}},
		{Name: "txo-fabric-indiba-sales", AuthentikGroupUUID: "forged", ActiveAuthentikUserUUIDs: []string{userOne}},
		{Name: "txo-fabric-indiba-sales", AuthentikGroupUUID: groupUUID, ActiveAuthentikUserUUIDs: []string{userOne,userOne}},
		{Name: "txo-fabric-indiba-sales", AuthentikGroupUUID: groupUUID, ActiveAuthentikUserUUIDs: []string{userOne,"not-a-uuid"}},
		{Name: "txo-fabric-indiba-sales", AuthentikGroupUUID: groupUUID, ActiveAuthentikUserUUIDs: []string{"550e8400-e29b-41d4-a716-446655440999"}},
	} {
		if got, err := r.ResolveGroupMembership(malformed); err == nil || got != nil {
			t.Fatalf("accepted partial or foreign group membership: %#v => %v", malformed, got)
		}
	}
	var nilRegistry *IdentityRegistry
	if got, err := nilRegistry.ResolveGroupMembership(snapshot); err == nil || got != nil {
		t.Fatal("nil registry accepted group")
	}
}

func TestIdentityRegistryRejectsAmbiguityAndForeignSubjects(t *testing.T) {
	good := identityFixtures()
	for _, tc := range []struct {name string; mutate func([]IdentityBinding) []IdentityBinding}{
		{"duplicate-subject", func(bs []IdentityBinding) []IdentityBinding { bs[1].OIDCSubject = bs[0].OIDCSubject; return bs }},
		{"duplicate-authentik", func(bs []IdentityBinding) []IdentityBinding { bs[1].AuthentikUserUUID = bs[0].AuthentikUserUUID; return bs }},
		{"duplicate-user", func(bs []IdentityBinding) []IdentityBinding { bs[1].FabricUserID = bs[0].FabricUserID; return bs }},
		{"cross-tenant", func(bs []IdentityBinding) []IdentityBinding { bs[1].TenantID = "TEN00001"; return bs }},
		{"cross-issuer", func(bs []IdentityBinding) []IdentityBinding { bs[1].Issuer = "https://authentik.truxonline.com/application/o/txo-fabric-hairem/"; return bs }},
		{"username-as-fabric-id", func(bs []IdentityBinding) []IdentityBinding { bs[1].FabricUserID = "edfoley"; return bs }},
		{"missing-fabric-id", func(bs []IdentityBinding) []IdentityBinding { bs[1].FabricUserID = ""; return bs }},
		{"bad-user-uuid", func(bs []IdentityBinding) []IdentityBinding { bs[1].AuthentikUserUUID = "invalid"; return bs }},
		{"missing-sub", func(bs []IdentityBinding) []IdentityBinding { bs[1].OIDCSubject = ""; return bs }},
		{"control-in-sub", func(bs []IdentityBinding) []IdentityBinding { bs[1].OIDCSubject = "bad\nsub"; return bs }},
		{"oversized-sub", func(bs []IdentityBinding) []IdentityBinding { bs[1].OIDCSubject = strings.Repeat("a", 256); return bs }},
	} {
		t.Run(tc.name,func(t *testing.T) {
			bs := append([]IdentityBinding(nil), good...)
			if _, err := NewIdentityRegistry("indiba", registryTenant, registryIssuer, []string{"sales"}, tc.mutate(bs)); err == nil {
				t.Fatal("accepted inconsistent or ambiguous identity registry")
			}
		})
	}
}

func TestIdentityRegistryRejectsUntrustedTenantAndIAMGroups(t *testing.T) {
	for _, tc := range []struct {tenantSlug,tenantID,issuer string; groups []string}{
		{"indiba", "TEN00003", registryIssuer, []string{"sales"}},
		{"indiba", registryTenant, "http://authentik.truxonline.com/application/o/txo-fabric-indiba/", []string{"sales"}},
		{"indiba", registryTenant, registryIssuer + "?x=1", []string{"sales"}},
		{"indiba", registryTenant, "https://authentik.truxonline.com/application/o/txo-fabric-hairem/", []string{"sales"}},
		{"Indiba", registryTenant, registryIssuer, []string{"sales"}},
		{"indiba", registryTenant, registryIssuer, []string{}},
		{"indiba", registryTenant, registryIssuer, []string{"sales", "sales"}},
		{"indiba", registryTenant, registryIssuer, []string{"sales", "../hairem"}},
	} {
		if _, err := NewIdentityRegistry(tc.tenantSlug, tc.tenantID, tc.issuer, tc.groups, identityFixtures()); err == nil {
			t.Fatalf("accepted invalid tenant/group scope: %#v", tc)
		}
	}
}

func TestIdentityRegistryRejectsGroupNamesBeyondAPIReadLimit(t *testing.T) {
	tenant := strings.Repeat("a", 63)
	issuer := fmt.Sprintf("https://authentik.truxonline.com/application/o/txo-fabric-%s/", tenant)
	if _, err := NewIdentityRegistry(tenant, registryTenant, issuer, []string{strings.Repeat("b", 63)}, nil); err == nil {
		t.Fatal("accepted a group name the Authentik client refuses to query")
	}
}

func TestIdentityRegistryNoImplicitEnrollmentAndTenantIsolation(t *testing.T) {
	indiba := mustIdentityRegistry(t)
	hairIssuer := "https://authentik.truxonline.com/application/o/txo-fabric-hairem/"
	empty, err := NewIdentityRegistry("hairem", "TEN00001", hairIssuer, []string{"client0"}, nil)
	if err != nil { t.Fatal(err) }
	if got, err := empty.ResolveOIDCSubject(registryIssuer, registrySub1); err == nil || got != "" {
		t.Fatal("cross-tenant OIDC subject accepted")
	}
	if got, err := empty.ResolveGroupMembership(GroupMembership{
		Name: "txo-fabric-hairem-client0", AuthentikGroupUUID: groupUUID,
		ActiveAuthentikUserUUIDs: []string{userOne},
	}); err == nil || got != nil {
		t.Fatal("unknown human enrollment silently granted a tenant membership")
	}
	// An Authentik UUID may map independently per tenant; scope is never global.
	hair, err := NewIdentityRegistry("hairem", "TEN00001", hairIssuer, []string{"client0"},
		[]IdentityBinding{{"TEN00001", hairIssuer, registrySub1, userOne, "usr000999"}})
	if err != nil { t.Fatal(err) }
	if got, err := hair.ResolveOIDCSubject(hairIssuer, registrySub1); err != nil || got != "usr000999" {
		t.Fatal("independent tenant mapping rejected")
	}
	if got, err := indiba.ResolveOIDCSubject(hairIssuer, registrySub1); err == nil || got != "" {
		t.Fatal("foreign issuer crossed tenant")
	}
}
