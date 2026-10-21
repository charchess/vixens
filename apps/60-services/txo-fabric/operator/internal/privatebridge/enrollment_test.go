package privatebridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type enrollmentFunc func(context.Context, string, string, string, string, string, string, time.Time) (string, error)

func (f enrollmentFunc) Enroll(ctx context.Context, tenant, issuer, subject, uuid, globalIssuer, globalSubject string, now time.Time) (string, error) {
	return f(ctx, tenant, issuer, subject, uuid, globalIssuer, globalSubject, now)
}

func TestPrivateEnrollmentRequiresBoundProofAndRejectsReplay(t *testing.T) {
	nonces := &memoryNonceStore{used: map[string]bool{}}
	handler := &EnrollmentHandler{
		Contract: Contract{Audience: testAudience, KubernetesIssuer: testIssuer, BFFServiceAccount: testSA, AssertionKey: []byte(strings.Repeat("k", 32)), Now: func() time.Time { return testNow }},
		Tokens: tokenReviewerFunc(func(_ context.Context, _ string, audience string) (Workload, error) {
			if audience != testAudience {
				return Workload{}, errors.New("wrong audience")
			}
			return Workload{Authenticated: true, Username: testSA, Audiences: []string{testAudience}}, nil
		}),
		Nonces: nonces,
		Enrollments: enrollmentFunc(func(_ context.Context, tenant, issuer, subject, uuid, globalIssuer, globalSubject string, _ time.Time) (string, error) {
			if tenant != "hairem" || issuer != "https://authentik.example/application/o/hairem/" || subject != "opaque-sub" || uuid != testUUID || globalIssuer != "https://authentik.example/application/o/txo-fabric-webui/" || globalSubject != "global-opaque-sub" {
				return "", errors.New("unexpected proof")
			}
			return "usr000001", nil
		}),
	}
	q := EnrollmentRequest{TenantKey: "hairem", OIDCIssuer: "https://authentik.example/application/o/hairem/", OIDCSubject: "opaque-sub", AuthentikUserUUID: testUUID, GlobalOIDCIssuer: "https://authentik.example/application/o/txo-fabric-webui/", GlobalOIDCSubject: "global-opaque-sub"}
	a := EnrollmentAssertion{Method: http.MethodPost, Path: EnrollmentPath, TenantKey: q.TenantKey, OIDCIssuer: q.OIDCIssuer, OIDCSubject: q.OIDCSubject, AuthentikUserUUID: q.AuthentikUserUUID, GlobalOIDCIssuer: q.GlobalOIDCIssuer, GlobalOIDCSubject: q.GlobalOIDCSubject, ServiceAccount: testSA, Audience: testAudience, IssuedAt: testNow.Unix(), ExpiresAt: testNow.Add(20 * time.Second).Unix(), JTI: "enrollment-proof-012345678901234"}
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignEnrollmentAssertion(handler.Contract.AssertionKey, a)
	if err != nil {
		t.Fatal(err)
	}
	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, EnrollmentPath, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+jwtFor(testIssuer, testSA, testAudience))
		req.Header.Set("X-Fabric-Assertion", signed)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	if got := request(); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "usr000001") {
		t.Fatalf("valid enrollment denied: %d %s", got.Code, got.Body.String())
	}
	if got := request(); got.Code != http.StatusForbidden {
		t.Fatalf("replayed enrollment accepted: %d", got.Code)
	}
}

func TestPrivateEnrollmentRejectsMissingGlobalProofBinding(t *testing.T) {
	q := EnrollmentRequest{
		TenantKey: "hairem", OIDCIssuer: "https://authentik.example/application/o/hairem/", OIDCSubject: "tenant-opaque-sub", AuthentikUserUUID: testUUID,
		GlobalOIDCIssuer: "https://authentik.example/application/o/txo-fabric-webui/", GlobalOIDCSubject: "global-opaque-sub",
	}
	a := EnrollmentAssertion{
		Method: http.MethodPost, Path: EnrollmentPath, TenantKey: q.TenantKey, OIDCIssuer: q.OIDCIssuer, OIDCSubject: q.OIDCSubject, AuthentikUserUUID: q.AuthentikUserUUID,
		ServiceAccount: testSA, Audience: testAudience, IssuedAt: testNow.Unix(), ExpiresAt: testNow.Add(20 * time.Second).Unix(), JTI: "global-binding-proof-012345678",
	}
	contract := Contract{Audience: testAudience, BFFServiceAccount: testSA}
	if validEnrollmentAssertion(a, q, contract, testNow) {
		t.Fatal("enrollment assertion without the HMAC-bound global proof was accepted")
	}
}
