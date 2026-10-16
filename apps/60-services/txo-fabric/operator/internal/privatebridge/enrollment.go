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

// EnrollmentPath is separate from read-only identity resolution. It accepts
// only a BFF-authenticated, server-side verified OIDC proof and cannot be
// reached by a browser token or reused assertion.
const EnrollmentPath = "/internal/v1/identity/enroll"

type EnrollmentRequest struct {
	TenantKey         string `json:"tenantKey"`
	OIDCIssuer        string `json:"oidcIssuer"`
	OIDCSubject       string `json:"oidcSubject"`
	AuthentikUserUUID string `json:"authentikUserUUID"`
}

type EnrollmentAssertion struct {
	Method            string `json:"method"`
	Path              string `json:"path"`
	TenantKey         string `json:"tenantKey"`
	OIDCIssuer        string `json:"oidcIssuer"`
	OIDCSubject       string `json:"oidcSubject"`
	AuthentikUserUUID string `json:"authentikUserUUID"`
	ServiceAccount    string `json:"serviceAccount"`
	Audience          string `json:"audience"`
	IssuedAt          int64  `json:"iat"`
	ExpiresAt         int64  `json:"exp"`
	JTI               string `json:"jti"`
}

// Enroller is implemented by the operator. Its input is trusted only after
// this handler has authenticated the exact BFF ServiceAccount and checked the
// HMAC-bound, one-shot assertion.
type Enroller interface {
	Enroll(context.Context, string, string, string, string, time.Time) (string, error)
}

type EnrollmentHandler struct {
	Contract    Contract
	Tokens      TokenReviewer
	Nonces      NonceStore
	Enrollments Enroller
}

func SignEnrollmentAssertion(key []byte, a EnrollmentAssertion) (string, error) {
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

func verifyEnrollmentAssertion(key []byte, raw string) (EnrollmentAssertion, error) {
	var a EnrollmentAssertion
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
		return EnrollmentAssertion{}, errDenied
	}
	return a, nil
}

func validEnrollmentAssertion(a EnrollmentAssertion, q EnrollmentRequest, c Contract, now time.Time) bool {
	if a.Method != http.MethodPost || a.Path != EnrollmentPath ||
		a.TenantKey != q.TenantKey || a.OIDCIssuer != q.OIDCIssuer ||
		a.OIDCSubject != q.OIDCSubject || a.AuthentikUserUUID != q.AuthentikUserUUID ||
		a.ServiceAccount != c.BFFServiceAccount || a.Audience != c.Audience ||
		!tenantKeyPattern.MatchString(a.TenantKey) || !uuidPattern.MatchString(a.AuthentikUserUUID) ||
		!strings.HasPrefix(a.OIDCIssuer, "https://") || len(a.OIDCIssuer) > 2048 ||
		a.OIDCSubject == "" || len(a.OIDCSubject) > 2048 || !jtiPattern.MatchString(a.JTI) {
		return false
	}
	issued, expires := time.Unix(a.IssuedAt, 0), time.Unix(a.ExpiresAt, 0)
	return a.IssuedAt > 0 && a.ExpiresAt > 0 && expires.After(issued) &&
		expires.Sub(issued) <= MaxAssertionAge && !now.Before(issued.Add(-ClockSkew)) &&
		now.Before(expires) && !now.After(issued.Add(MaxAssertionAge+ClockSkew))
}

func (h *EnrollmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	deny := func() {
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "identity unavailable", http.StatusForbidden)
	}
	if h == nil || r == nil || h.Tokens == nil || h.Nonces == nil || h.Enrollments == nil ||
		h.Contract.Audience == "" || h.Contract.KubernetesIssuer == "" ||
		!strings.HasPrefix(h.Contract.BFFServiceAccount, "system:serviceaccount:") ||
		len(h.Contract.AssertionKey) < 32 || r.Method != http.MethodPost ||
		r.URL.Path != EnrollmentPath || r.URL.RawQuery != "" {
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
	var q EnrollmentRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil || decoder.Decode(new(any)) != io.EOF ||
		!tenantKeyPattern.MatchString(q.TenantKey) || !uuidPattern.MatchString(q.AuthentikUserUUID) ||
		!strings.HasPrefix(q.OIDCIssuer, "https://") || len(q.OIDCIssuer) > 2048 ||
		q.OIDCSubject == "" || len(q.OIDCSubject) > 2048 {
		deny()
		return
	}
	assertion, err := verifyEnrollmentAssertion(h.Contract.AssertionKey, r.Header.Get("X-Fabric-Assertion"))
	now := time.Now().UTC()
	if h.Contract.Now != nil {
		now = h.Contract.Now().UTC()
	}
	if err != nil || !validEnrollmentAssertion(assertion, q, h.Contract, now) {
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
	if h.Nonces.Consume(r.Context(), scopedNonceKey(h.Contract, Assertion{TenantKey: assertion.TenantKey, JTI: assertion.JTI}), time.Unix(assertion.ExpiresAt, 0).Add(ClockSkew)) != nil {
		deny()
		return
	}
	userID, err := h.Enrollments.Enroll(r.Context(), q.TenantKey, q.OIDCIssuer, q.OIDCSubject, q.AuthentikUserUUID, now)
	if err != nil || userID == "" {
		deny()
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Resolution{TenantKey: q.TenantKey, FabricUserID: userID})
}
