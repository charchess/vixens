package privatebridge

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// BrowserAuthorityPath is a separate read-only browser presentation contract.
// It deliberately cannot resolve endpoint, credential, store/model, namespace,
// service, group, or any deployment authority.
const BrowserAuthorityPath = "/internal/v1/browser-authority/resolve"

const (
	BrowserAuthorityCatalog = "catalog"
	BrowserAuthorityTenant  = "tenant"
	BrowserAuthorityAgent   = "agent"
)

var agentKeyPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

type BrowserAuthorityRequest struct {
	Kind              string `json:"kind"`
	AuthentikUserUUID string `json:"authentikUserUUID"`
	TenantKey         string `json:"tenantKey,omitempty"`
	AgentKey          string `json:"agentKey,omitempty"`
}

type BrowserAuthorityAssertion struct {
	Method            string `json:"method"`
	Path              string `json:"path"`
	Kind              string `json:"kind"`
	AuthentikUserUUID string `json:"authentikUserUUID"`
	TenantKey         string `json:"tenantKey,omitempty"`
	AgentKey          string `json:"agentKey,omitempty"`
	ServiceAccount    string `json:"serviceAccount"`
	Audience          string `json:"audience"`
	IssuedAt          int64  `json:"iat"`
	ExpiresAt         int64  `json:"exp"`
	JTI               string `json:"jti"`
}

// BrowserTenant is intentionally the complete public tenant record.
type BrowserTenant struct {
	TenantKey      string `json:"tenantKey"`
	FabricTenantID string `json:"fabricTenantId"`
	DisplayName    string `json:"displayName"`
}

type BrowserTenantAuthority struct {
	BrowserTenant
	FabricUserID  string `json:"fabricUserId"`
	AccountActive bool   `json:"accountActive"`
	Fresh         bool   `json:"fresh"`
	CanEnter      bool   `json:"canEnter"`
}

type BrowserAgentAuthority struct {
	AgentKey    string `json:"agentKey"`
	DisplayName string `json:"displayName"`
	AgentUID    string `json:"agentUID"`
	Generation  int64  `json:"generation"`
}

type BrowserAuthorityResponse struct {
	Tenants []BrowserTenant          `json:"tenants,omitempty"`
	Tenant  *BrowserTenantAuthority  `json:"tenant,omitempty"`
	Agent   *BrowserAgentAuthority   `json:"agent,omitempty"`
}

// BrowserAuthorityResolver receives only an authenticated, one-shot, signed
// server-side request. It is responsible for canonical Kubernetes/ledger/FGA
// reads; browser input is never a source of Fabric infrastructure authority.
type BrowserAuthorityResolver interface {
	ResolveBrowserAuthority(context.Context, BrowserAuthorityRequest, time.Time) (BrowserAuthorityResponse, error)
}

type BrowserAuthorityHandler struct {
	Contract   Contract
	Tokens     TokenReviewer
	Nonces     NonceStore
	Authorities BrowserAuthorityResolver
}

// SignBrowserAuthorityAssertion has a distinct payload type, so a valid
// legacy identity assertion cannot be replayed against this route.
func SignBrowserAuthorityAssertion(key []byte, a BrowserAuthorityAssertion) (string, error) {
	if len(key) < 32 { return "", errDenied }
	data, err := json.Marshal(a); if err != nil { return "", errDenied }
	payload := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, key); _, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func verifyBrowserAssertion(key []byte, raw string) (BrowserAuthorityAssertion, error) {
	var a BrowserAuthorityAssertion
	if len(key) < 32 || len(raw) > 8192 { return a, errDenied }
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || len(parts[1]) != 43 { return a, errDenied }
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(sig) != sha256.Size { return a, errDenied }
	mac := hmac.New(sha256.New, key); _, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(sig, mac.Sum(nil)) { return a, errDenied }
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(payload) > 4096 { return a, errDenied }
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&a) != nil || decoder.Decode(new(any)) != io.EOF { return a, errDenied }
	return a, nil
}

func validBrowserRequest(q BrowserAuthorityRequest) bool {
	if !uuidPattern.MatchString(q.AuthentikUserUUID) { return false }
	switch q.Kind {
	case BrowserAuthorityCatalog:
		return q.TenantKey == "" && q.AgentKey == ""
	case BrowserAuthorityTenant:
		return tenantKeyPattern.MatchString(q.TenantKey) && q.AgentKey == ""
	case BrowserAuthorityAgent:
		return tenantKeyPattern.MatchString(q.TenantKey) && agentKeyPattern.MatchString(q.AgentKey)
	default:
		return false
	}
}

func validBrowserAssertion(a BrowserAuthorityAssertion, q BrowserAuthorityRequest, c Contract, now time.Time) bool {
	if a.Method != http.MethodPost || a.Path != BrowserAuthorityPath || a.Kind != q.Kind ||
		a.AuthentikUserUUID != q.AuthentikUserUUID || a.TenantKey != q.TenantKey || a.AgentKey != q.AgentKey ||
		a.ServiceAccount != c.BFFServiceAccount || a.Audience != c.Audience || !validBrowserRequest(q) || !jtiPattern.MatchString(a.JTI) { return false }
	issued, expires := time.Unix(a.IssuedAt, 0), time.Unix(a.ExpiresAt, 0)
	return a.IssuedAt > 0 && a.ExpiresAt > 0 && expires.After(issued) && expires.Sub(issued) <= MaxAssertionAge &&
		!now.Before(issued.Add(-ClockSkew)) && now.Before(expires) && !now.After(issued.Add(MaxAssertionAge+ClockSkew))
}

func (h *BrowserAuthorityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	deny := func() { w.Header().Set("Cache-Control", "no-store"); http.Error(w, "browser authority unavailable", http.StatusForbidden) }
	if h == nil || r == nil || h.Tokens == nil || h.Nonces == nil || h.Authorities == nil || h.Contract.Audience == "" || h.Contract.KubernetesIssuer == "" || !strings.HasPrefix(h.Contract.BFFServiceAccount, "system:serviceaccount:") || len(h.Contract.AssertionKey) < 32 || r.Method != http.MethodPost || r.URL.Path != BrowserAuthorityPath || r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/json" { deny(); return }
	rawToken := r.Header.Get("Authorization")
	if len(rawToken) > 16391 || !strings.HasPrefix(rawToken, "Bearer ") || strings.TrimSpace(rawToken[7:]) != rawToken[7:] || rawToken[7:] == "" { deny(); return }
	var q BrowserAuthorityRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)); decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !validBrowserRequest(q) { deny(); return }
	a, err := verifyBrowserAssertion(h.Contract.AssertionKey, r.Header.Get("X-Fabric-Assertion"))
	now := time.Now().UTC()
	if h.Contract.Now != nil { now = h.Contract.Now().UTC() }
	if err != nil || !validBrowserAssertion(a, q, h.Contract, now) { deny(); return }
	review, err := h.Tokens.Review(r.Context(), rawToken[7:], h.Contract.Audience)
	if err != nil || !review.Authenticated || review.Username != h.Contract.BFFServiceAccount || !exactlyOne(review.Audiences, h.Contract.Audience) || !tokenClaimsMatch(rawToken[7:], h.Contract.KubernetesIssuer, h.Contract.Audience, h.Contract.BFFServiceAccount, now) { deny(); return }
	if h.Nonces.Consume(r.Context(), scopedNonceKey(h.Contract, Assertion{TenantKey: q.TenantKey, JTI: a.JTI}), time.Unix(a.ExpiresAt, 0).Add(ClockSkew)) != nil { deny(); return }
	out, err := h.Authorities.ResolveBrowserAuthority(r.Context(), q, now)
	if err != nil || !validBrowserResponse(q.Kind, out) { deny(); return }
	w.Header().Set("Cache-Control", "no-store"); w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func validBrowserResponse(kind string, out BrowserAuthorityResponse) bool {
	switch kind {
	case BrowserAuthorityCatalog:
		return out.Tenant == nil && out.Agent == nil && out.Tenants != nil
	case BrowserAuthorityTenant:
		return out.Tenants == nil && out.Agent == nil && out.Tenant != nil && out.Tenant.AccountActive && out.Tenant.Fresh && out.Tenant.CanEnter
	case BrowserAuthorityAgent:
		return out.Tenants == nil && out.Tenant == nil && out.Agent != nil
	default: return false
	}
}
