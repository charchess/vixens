package privatebridge

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// ActiveHumanPath is a BFF-only identity-liveness check. It grants no tenant,
// Fabric user ID, authorization relation, agent, or chat capability.
const ActiveHumanPath = "/internal/v1/identity/active"

type ActiveHumanRequest struct {
	OIDCIssuer        string `json:"oidcIssuer"`
	OIDCSubject       string `json:"oidcSubject"`
	AuthentikUserUUID string `json:"authentikUserUUID"`
}

type ActiveHumanAssertion struct {
	Method            string `json:"method"`
	Path              string `json:"path"`
	OIDCIssuer        string `json:"oidcIssuer"`
	OIDCSubject       string `json:"oidcSubject"`
	AuthentikUserUUID string `json:"authentikUserUUID"`
	ServiceAccount    string `json:"serviceAccount"`
	Audience          string `json:"audience"`
	IssuedAt          int64  `json:"iat"`
	ExpiresAt         int64  `json:"exp"`
	JTI               string `json:"jti"`
}

type ActiveHumanResponse struct {
	OIDCIssuer        string `json:"oidcIssuer"`
	OIDCSubject       string `json:"oidcSubject"`
	AuthentikUserUUID string `json:"authentikUserUUID"`
	AccountActive     bool   `json:"accountActive"`
	Fresh             bool   `json:"fresh"`
}

// ActiveHumanVerifier is implemented by the operator. The BFF has already
// verified its OIDC ID token; this rechecks immutable-account liveness.
type ActiveHumanVerifier interface {
	VerifyActiveHuman(context.Context, string, string, string, time.Time) error
}

type ActiveHumanHandler struct {
	Contract Contract
	Tokens   TokenReviewer
	Nonces   NonceStore
	Accounts ActiveHumanVerifier
}

func SignActiveHumanAssertion(key []byte, a ActiveHumanAssertion) (string, error) {
	if len(key) < 32 {
		return "", errDenied
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return "", errDenied
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func verifyActiveHumanAssertion(key []byte, raw string) (ActiveHumanAssertion, error) {
	var a ActiveHumanAssertion
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
	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.DisallowUnknownFields()
	if dec.Decode(&a) != nil || dec.Decode(new(any)) != io.EOF {
		return ActiveHumanAssertion{}, errDenied
	}
	return a, nil
}

func validActiveHumanAssertion(a ActiveHumanAssertion, q ActiveHumanRequest, c Contract, now time.Time) bool {
	if a.Method != http.MethodPost || a.Path != ActiveHumanPath ||
		a.OIDCIssuer != q.OIDCIssuer || a.OIDCSubject != q.OIDCSubject ||
		a.AuthentikUserUUID != q.AuthentikUserUUID || a.ServiceAccount != c.BFFServiceAccount ||
		a.Audience != c.Audience || !uuidPattern.MatchString(a.AuthentikUserUUID) ||
		!strings.HasPrefix(a.OIDCIssuer, "https://") || len(a.OIDCIssuer) > 2048 ||
		a.OIDCSubject == "" || len(a.OIDCSubject) > 2048 || !jtiPattern.MatchString(a.JTI) {
		return false
	}
	issued, expires := time.Unix(a.IssuedAt, 0), time.Unix(a.ExpiresAt, 0)
	return a.IssuedAt > 0 && a.ExpiresAt > 0 && expires.After(issued) &&
		expires.Sub(issued) <= MaxAssertionAge && !now.Before(issued.Add(-ClockSkew)) &&
		now.Before(expires) && !now.After(issued.Add(MaxAssertionAge+ClockSkew))
}

func activeHumanNonceKey(c Contract, a ActiveHumanAssertion) string {
	scope := c.KubernetesIssuer + "\x00" + c.Audience + "\x00" + c.BFFServiceAccount + "\x00" + a.JTI
	sum := sha256.Sum256([]byte(scope))
	return "bff-active-jti-" + base64.RawURLEncoding.EncodeToString(sum[:24])
}

func (h *ActiveHumanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	deny := func() {
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "identity unavailable", http.StatusForbidden)
	}
	if h == nil || r == nil || h.Tokens == nil || h.Nonces == nil || h.Accounts == nil ||
		h.Contract.Audience == "" || h.Contract.KubernetesIssuer == "" ||
		!strings.HasPrefix(h.Contract.BFFServiceAccount, "system:serviceaccount:") ||
		len(h.Contract.AssertionKey) < 32 || r.Method != http.MethodPost ||
		r.URL.Path != ActiveHumanPath || r.URL.RawQuery != "" {
		deny()
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		deny()
		return
	}
	rawToken := r.Header.Get("Authorization")
	if len(rawToken) > 16391 || !strings.HasPrefix(rawToken, "Bearer ") ||
		strings.TrimSpace(rawToken[7:]) != rawToken[7:] || rawToken[7:] == "" {
		deny()
		return
	}
	var q ActiveHumanRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	dec.DisallowUnknownFields()
	if dec.Decode(&q) != nil || dec.Decode(new(any)) != io.EOF ||
		!uuidPattern.MatchString(q.AuthentikUserUUID) || !strings.HasPrefix(q.OIDCIssuer, "https://") ||
		len(q.OIDCIssuer) > 2048 || q.OIDCSubject == "" || len(q.OIDCSubject) > 2048 {
		deny()
		return
	}
	assertion, err := verifyActiveHumanAssertion(h.Contract.AssertionKey, r.Header.Get("X-Fabric-Assertion"))
	now := time.Now().UTC()
	if h.Contract.Now != nil {
		now = h.Contract.Now().UTC()
	}
	if err != nil || !validActiveHumanAssertion(assertion, q, h.Contract, now) {
		deny()
		return
	}
	review, err := h.Tokens.Review(r.Context(), rawToken[7:], h.Contract.Audience)
	if err != nil || !review.Authenticated || review.Username != h.Contract.BFFServiceAccount ||
		!exactlyOne(review.Audiences, h.Contract.Audience) ||
		!tokenClaimsMatch(rawToken[7:], h.Contract.KubernetesIssuer, h.Contract.Audience, h.Contract.BFFServiceAccount, now) {
		deny()
		return
	}
	if h.Nonces.Consume(r.Context(), activeHumanNonceKey(h.Contract, assertion), time.Unix(assertion.ExpiresAt, 0).Add(ClockSkew)) != nil {
		deny()
		return
	}
	if h.Accounts.VerifyActiveHuman(r.Context(), q.OIDCIssuer, q.OIDCSubject, q.AuthentikUserUUID, now) != nil {
		deny()
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ActiveHumanResponse{OIDCIssuer: q.OIDCIssuer, OIDCSubject: q.OIDCSubject, AuthentikUserUUID: q.AuthentikUserUUID, AccountActive: true, Fresh: true})
}
