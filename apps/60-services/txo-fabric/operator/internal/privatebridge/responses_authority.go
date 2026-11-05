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

// ResponsesAuthorityPath is a private, BFF-to-operator contract. Unlike the
// browser-authority route it returns canonical lifecycle facts required by the
// stateless Hermes Responses adapter, never browser presentation data.
const ResponsesAuthorityPath = "/internal/v1/responses-authority/resolve"

const (
	ResponsesAuthorityResolveTenant = "resolve-tenant"
	ResponsesAuthorityResolveAgent  = "resolve-agent"
	ResponsesAuthorityAuthorizeChat = "authorize-chat"
	ResponsesAuthorityVerifyRuntime = "verify-runtime"
)

var (
	fabricTenantIDPattern    = regexp.MustCompile(`^TEN[0-9]{5,}$`)
	fabricUserIDPattern      = regexp.MustCompile(`^usr[0-9a-f]{32}$`)
	responsesAgentKeyPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,54}[a-z0-9])?$`)
)

type ResponsesAuthorityRequest struct {
	Operation        string `json:"operation"`
	TenantKey        string `json:"tenantKey"`
	FabricTenantID   string `json:"fabricTenantId"`
	AgentKey         string `json:"agentKey,omitempty"`
	FabricUserID     string `json:"fabricUserId,omitempty"`
	TenantBundleUID  string `json:"tenantBundleUID,omitempty"`
	TenantGeneration int64  `json:"tenantGeneration,omitempty"`
	AgentIdentityUID string `json:"agentIdentityUID,omitempty"`
	AgentGeneration  int64  `json:"agentGeneration,omitempty"`
}

type ResponsesAuthorityAssertion struct {
	Method           string `json:"method"`
	Path             string `json:"path"`
	Operation        string `json:"operation"`
	TenantKey        string `json:"tenantKey"`
	FabricTenantID   string `json:"fabricTenantId"`
	AgentKey         string `json:"agentKey,omitempty"`
	FabricUserID     string `json:"fabricUserId,omitempty"`
	TenantBundleUID  string `json:"tenantBundleUID,omitempty"`
	TenantGeneration int64  `json:"tenantGeneration,omitempty"`
	AgentIdentityUID string `json:"agentIdentityUID,omitempty"`
	AgentGeneration  int64  `json:"agentGeneration,omitempty"`
	ServiceAccount   string `json:"serviceAccount"`
	Audience         string `json:"audience"`
	IssuedAt         int64  `json:"iat"`
	ExpiresAt        int64  `json:"exp"`
	JTI              string `json:"jti"`
}

type ResponsesTenantAuthority struct {
	TenantKey       string `json:"tenantKey"`
	FabricTenantID  string `json:"fabricTenantId"`
	TenantBundleUID string `json:"tenantBundleUID"`
	Status          string `json:"status"`
}

type ResponsesAuthorityResponse struct {
	Tenant *ResponsesTenantAuthority `json:"tenant,omitempty"`
}

// ResponsesAuthorityResolver owns all canonical Fabric/Kubernetes reads. The
// handler never turns a BFF request into infrastructure authority by itself.
type ResponsesAuthorityResolver interface {
	ResolveResponsesAuthority(context.Context, ResponsesAuthorityRequest, time.Time) (ResponsesAuthorityResponse, error)
}

type ResponsesAuthorityHandler struct {
	Contract    Contract
	Tokens      TokenReviewer
	Nonces      NonceStore
	Authorities ResponsesAuthorityResolver
}

func SignResponsesAuthorityAssertion(key []byte, a ResponsesAuthorityAssertion) (string, error) {
	if len(key) < 32 {
		return "", errDenied
	}
	data, err := json.Marshal(a)
	if err != nil {
		return "", errDenied
	}
	payload := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func verifyResponsesAuthorityAssertion(key []byte, raw string) (ResponsesAuthorityAssertion, error) {
	var a ResponsesAuthorityAssertion
	if len(key) < 32 || len(raw) > 8192 {
		return a, errDenied
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || len(parts[1]) != 43 {
		return a, errDenied
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(sig) != sha256.Size {
		return a, errDenied
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return a, errDenied
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(payload) > 4096 {
		return a, errDenied
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&a) != nil || decoder.Decode(new(any)) != io.EOF {
		return a, errDenied
	}
	return a, nil
}

func validResponsesAuthorityRequest(q ResponsesAuthorityRequest) bool {
	base := tenantKeyPattern.MatchString(q.TenantKey) && fabricTenantIDPattern.MatchString(q.FabricTenantID)
	if !base {
		return false
	}
	switch q.Operation {
	case ResponsesAuthorityResolveTenant:
		return q.AgentKey == "" && q.FabricUserID == "" && q.TenantBundleUID == "" && q.TenantGeneration == 0 && q.AgentIdentityUID == "" && q.AgentGeneration == 0
	case ResponsesAuthorityResolveAgent:
		return responsesAgentKeyPattern.MatchString(q.AgentKey) && q.FabricUserID == "" && q.TenantBundleUID == "" && q.TenantGeneration == 0 && q.AgentIdentityUID == "" && q.AgentGeneration == 0
	case ResponsesAuthorityAuthorizeChat:
		return responsesAgentKeyPattern.MatchString(q.AgentKey) && fabricUserIDPattern.MatchString(q.FabricUserID) && q.TenantBundleUID == "" && q.TenantGeneration == 0 && q.AgentIdentityUID == "" && q.AgentGeneration == 0
	case ResponsesAuthorityVerifyRuntime:
		return responsesAgentKeyPattern.MatchString(q.AgentKey) && q.FabricUserID == "" && uuidPattern.MatchString(q.TenantBundleUID) && q.TenantGeneration > 0 && uuidPattern.MatchString(q.AgentIdentityUID) && q.AgentGeneration > 0
	default:
		return false
	}
}

func sameResponsesAuthorityScope(a ResponsesAuthorityAssertion, q ResponsesAuthorityRequest) bool {
	return a.Operation == q.Operation && a.TenantKey == q.TenantKey && a.FabricTenantID == q.FabricTenantID &&
		a.AgentKey == q.AgentKey && a.FabricUserID == q.FabricUserID &&
		a.TenantBundleUID == q.TenantBundleUID && a.TenantGeneration == q.TenantGeneration &&
		a.AgentIdentityUID == q.AgentIdentityUID && a.AgentGeneration == q.AgentGeneration
}

func validResponsesAuthorityAssertion(a ResponsesAuthorityAssertion, q ResponsesAuthorityRequest, c Contract, now time.Time) bool {
	if a.Method != http.MethodPost || a.Path != ResponsesAuthorityPath || !sameResponsesAuthorityScope(a, q) || a.ServiceAccount != c.BFFServiceAccount || a.Audience != c.Audience || !validResponsesAuthorityRequest(q) || !jtiPattern.MatchString(a.JTI) {
		return false
	}
	issued, expires := time.Unix(a.IssuedAt, 0), time.Unix(a.ExpiresAt, 0)
	return a.IssuedAt > 0 && a.ExpiresAt > 0 && expires.After(issued) && expires.Sub(issued) <= MaxAssertionAge && !now.Before(issued.Add(-ClockSkew)) && now.Before(expires) && !now.After(issued.Add(MaxAssertionAge+ClockSkew))
}

func validResponsesAuthorityResponse(q ResponsesAuthorityRequest, out ResponsesAuthorityResponse) bool {
	return out.Tenant != nil && out.Tenant.TenantKey == q.TenantKey && out.Tenant.FabricTenantID == q.FabricTenantID && uuidPattern.MatchString(out.Tenant.TenantBundleUID) && out.Tenant.Status == "Ready"
}

func (h *ResponsesAuthorityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	deny := func() {
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "responses authority unavailable", http.StatusForbidden)
	}
	if h == nil || r == nil || h.Tokens == nil || h.Nonces == nil || h.Authorities == nil || h.Contract.Audience == "" || h.Contract.KubernetesIssuer == "" || !strings.HasPrefix(h.Contract.BFFServiceAccount, "system:serviceaccount:") || len(h.Contract.AssertionKey) < 32 || r.Method != http.MethodPost || r.URL.Path != ResponsesAuthorityPath || r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/json" {
		deny()
		return
	}
	rawToken := r.Header.Get("Authorization")
	if len(rawToken) > 16391 || !strings.HasPrefix(rawToken, "Bearer ") || strings.TrimSpace(rawToken[7:]) != rawToken[7:] || rawToken[7:] == "" {
		deny()
		return
	}
	var q ResponsesAuthorityRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF || !validResponsesAuthorityRequest(q) {
		deny()
		return
	}
	a, err := verifyResponsesAuthorityAssertion(h.Contract.AssertionKey, r.Header.Get("X-Fabric-Assertion"))
	now := time.Now().UTC()
	if h.Contract.Now != nil {
		now = h.Contract.Now().UTC()
	}
	if err != nil || !validResponsesAuthorityAssertion(a, q, h.Contract, now) {
		deny()
		return
	}
	review, err := h.Tokens.Review(r.Context(), rawToken[7:], h.Contract.Audience)
	if err != nil || !review.Authenticated || review.Username != h.Contract.BFFServiceAccount || !exactlyOne(review.Audiences, h.Contract.Audience) || !tokenClaimsMatch(rawToken[7:], h.Contract.KubernetesIssuer, h.Contract.Audience, h.Contract.BFFServiceAccount, now) {
		deny()
		return
	}
	if h.Nonces.Consume(r.Context(), scopedNonceKey(h.Contract, Assertion{TenantKey: q.TenantKey, JTI: a.JTI}), time.Unix(a.ExpiresAt, 0).Add(ClockSkew)) != nil {
		deny()
		return
	}
	out, err := h.Authorities.ResolveResponsesAuthority(r.Context(), q, now)
	if err != nil || !validResponsesAuthorityResponse(q, out) {
		deny()
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
