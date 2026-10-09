package authentik

import (
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var (
	canonicalTenantID   = regexp.MustCompile("^TEN[0-9]{5,}$")
	canonicalTenantSlug = regexp.MustCompile("^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$")
	canonicalUserID     = regexp.MustCompile("^usr[a-z0-9-]{1,63}$")
	canonicalGroupKey   = regexp.MustCompile("^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$")
)

// IdentityBinding is an EXPLICIT, durable, trusted Fabric identity association.
// The OIDC subject, Authentik user UUID, and Fabric user ID are distinct domains.
// The source of these bindings must be reviewed, Fabric-owned durable state;
// neither a login claim, username, group membership nor workspace userRef
// may mint or update a binding by itself.
type IdentityBinding struct {
	TenantID          string
	Issuer            string
	OIDCSubject       string
	AuthentikUserUUID string
	FabricUserID      string
}

// IdentityRegistry is an immutable, tenant-scoped projection of trusted Fabric
// identity bindings. It DOES NOT verify OIDC tokens, read IAM memberships,
// allocate user IDs, create grants, or provide a backing persistence layer.
// The future caller must build it from a complete, freshness-checked Fabric
// snapshot, and only call ResolveOIDCSubject after server-side OIDC verification.
type IdentityRegistry struct {
	tenantID       string
	issuer         string
	bySubject      map[string]IdentityBinding
	byAuthentikID  map[string]IdentityBinding
	byFabricID     map[string]IdentityBinding
	approvedGroups map[string]bool
}

func validIssuer(tenantSlug, issuer string) bool {
	parsed, err := url.Parse(issuer)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" &&
		parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" &&
		parsed.Opaque == "" && parsed.RawPath == "" &&
		parsed.Path == "/application/o/txo-fabric-"+tenantSlug+"/" &&
		strings.ToLower(parsed.Host) == parsed.Host && parsed.Host == parsed.Hostname()
}

// OIDC subjects are case-sensitive opaque identifiers. Never normalize them,
// derive them from mutable usernames, or infer that they equal IAM UUIDs.
func validOpaqueSubject(subject string) bool {
	if len(subject) == 0 || len(subject) > 255 {
		return false
	}
	for _, char := range []byte(subject) {
		if char < 0x21 || char > 0x7e {
			return false
		}
	}
	return true
}

// NewIdentityRegistry rejects any inconsistent, ambiguous or foreign-tenant
// binding as a whole. approvedGroupKeys comes from the owning TenantBundle's
// declared humanAccess.web.iamGroups, NOT from a browser request or OIDC claim.
// An empty registry is legitimate for a tenant with no enrolled human identities,
// but it cannot resolve a principal or translate a nonempty group snapshot.
func NewIdentityRegistry(tenantSlug, tenantID, issuer string, approvedGroupKeys []string, bindings []IdentityBinding) (*IdentityRegistry, error) {
	if !canonicalTenantSlug.MatchString(tenantSlug) || !canonicalTenantID.MatchString(tenantID) ||
		!validIssuer(tenantSlug, issuer) || len(approvedGroupKeys) == 0 ||
		len(approvedGroupKeys) > 32 || len(bindings) > 10000 {
		return nil, errors.New("invalid Fabric tenant identity registry scope")
	}
	registry := &IdentityRegistry{
		tenantID: tenantID, issuer: issuer,
		bySubject:      make(map[string]IdentityBinding, len(bindings)),
		byAuthentikID:  make(map[string]IdentityBinding, len(bindings)),
		byFabricID:     make(map[string]IdentityBinding, len(bindings)),
		approvedGroups: make(map[string]bool, len(approvedGroupKeys)),
	}
	for _, key := range approvedGroupKeys {
		groupName := "txo-fabric-" + tenantSlug + "-" + key
		if !canonicalGroupKey.MatchString(key) || len(groupName) > 128 || registry.approvedGroups[groupName] {
			return nil, errors.New("invalid or repeated Fabric IAM group")
		}
		registry.approvedGroups[groupName] = true
	}
	for _, binding := range bindings {
		if binding.TenantID != tenantID || binding.Issuer != issuer ||
			!validOpaqueSubject(binding.OIDCSubject) ||
			!uuidPattern.MatchString(binding.AuthentikUserUUID) ||
			!canonicalUserID.MatchString(binding.FabricUserID) {
			return nil, errors.New("missing, foreign or malformed Fabric human identity binding")
		}
		if _, exists := registry.bySubject[binding.OIDCSubject]; exists {
			return nil, errors.New("duplicate OIDC subject in tenant identity registry")
		}
		if _, exists := registry.byAuthentikID[binding.AuthentikUserUUID]; exists {
			return nil, errors.New("duplicate Authentik UUID in tenant identity registry")
		}
		if _, exists := registry.byFabricID[binding.FabricUserID]; exists {
			return nil, errors.New("duplicate Fabric user ID in tenant identity registry")
		}
		registry.bySubject[binding.OIDCSubject] = binding
		registry.byAuthentikID[binding.AuthentikUserUUID] = binding
		registry.byFabricID[binding.FabricUserID] = binding
	}
	return registry, nil
}

// ResolveOIDCSubject is only a LOOKUP, not an OIDC verification method.
// Its caller must already have verified signature, issuer, audience, expiry,
// nonce/session and the active tenant in the server-side authentication layer.
func (r *IdentityRegistry) ResolveOIDCSubject(issuer, subject string) (string, error) {
	if r == nil || issuer != r.issuer || !validOpaqueSubject(subject) {
		return "", errors.New("untrusted Fabric OIDC principal")
	}
	binding, exists := r.bySubject[subject]
	if !exists || binding.Issuer != issuer || binding.TenantID != r.tenantID {
		return "", errors.New("unmapped Fabric OIDC principal")
	}
	return binding.FabricUserID, nil
}

// ResolveGroupMembership translates one COMPLETE Authentik membership snapshot
// to canonical Fabric user IDs; a missing binding is an ERROR, never a skipped
// user or an empty desired snapshot. It grants nothing and does not write FGA.
// The caller must still verify Authentik freshness and group UUID continuity.
func (r *IdentityRegistry) ResolveGroupMembership(membership GroupMembership) ([]string, error) {
	if r == nil || !r.approvedGroups[membership.Name] ||
		!uuidPattern.MatchString(membership.AuthentikGroupUUID) ||
		membership.ActiveAuthentikUserUUIDs == nil {
		return nil, errors.New("untrusted or unapproved Fabric IAM group snapshot")
	}
	users := make([]string, 0, len(membership.ActiveAuthentikUserUUIDs))
	seen := map[string]bool{}
	for _, id := range membership.ActiveAuthentikUserUUIDs {
		if !uuidPattern.MatchString(id) || seen[id] {
			return nil, errors.New("invalid or duplicate user in Fabric IAM snapshot")
		}
		seen[id] = true
		binding, exists := r.byAuthentikID[id]
		if !exists || binding.TenantID != r.tenantID || binding.Issuer != r.issuer {
			return nil, errors.New("unmapped Authentik group member: refusing partial grants")
		}
		users = append(users, binding.FabricUserID)
	}
	sort.Strings(users)
	return users, nil
}
