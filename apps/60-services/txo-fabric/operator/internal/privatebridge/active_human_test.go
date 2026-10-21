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

type activeHumanFunc func(context.Context, string, string, string, time.Time) error

func (f activeHumanFunc) VerifyActiveHuman(ctx context.Context, issuer, subject, uuid string, now time.Time) error {
	return f(ctx, issuer, subject, uuid, now)
}

func TestPrivateActiveHumanRequiresBoundProofAndRejectsReplay(t *testing.T) {
	nonces := &memoryNonceStore{used: map[string]bool{}}
	handler := &ActiveHumanHandler{
		Contract: Contract{Audience: testAudience, KubernetesIssuer: testIssuer, BFFServiceAccount: testSA, AssertionKey: []byte(strings.Repeat("k", 32)), Now: func() time.Time { return testNow }},
		Tokens: tokenReviewerFunc(func(_ context.Context, _ string, audience string) (Workload, error) {
			if audience != testAudience {
				return Workload{}, errors.New("wrong audience")
			}
			return Workload{Authenticated: true, Username: testSA, Audiences: []string{testAudience}}, nil
		}),
		Nonces: nonces,
		Accounts: activeHumanFunc(func(_ context.Context, issuer, subject, uuid string, now time.Time) error {
			if issuer != "https://authentik.example/application/o/txo-fabric-webui/" || subject != "opaque-sub" || uuid != testUUID || !now.Equal(testNow) {
				return errors.New("unexpected identity")
			}
			return nil
		}),
	}
	q := ActiveHumanRequest{OIDCIssuer: "https://authentik.example/application/o/txo-fabric-webui/", OIDCSubject: "opaque-sub", AuthentikUserUUID: testUUID}
	a := ActiveHumanAssertion{Method: http.MethodPost, Path: ActiveHumanPath, OIDCIssuer: q.OIDCIssuer, OIDCSubject: q.OIDCSubject, AuthentikUserUUID: q.AuthentikUserUUID, ServiceAccount: testSA, Audience: testAudience, IssuedAt: testNow.Unix(), ExpiresAt: testNow.Add(20 * time.Second).Unix(), JTI: "active-human-proof-012345678901"}
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignActiveHumanAssertion(handler.Contract.AssertionKey, a)
	if err != nil {
		t.Fatal(err)
	}
	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, ActiveHumanPath, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+jwtFor(testIssuer, testSA, testAudience))
		req.Header.Set("X-Fabric-Assertion", signed)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	if got := request(); got.Code != http.StatusOK {
		t.Fatalf("valid active-human proof denied: %d", got.Code)
	} else {
		var body ActiveHumanResponse
		if err := json.Unmarshal(got.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.OIDCIssuer != q.OIDCIssuer || body.OIDCSubject != q.OIDCSubject || body.AuthentikUserUUID != q.AuthentikUserUUID || !body.AccountActive || !body.Fresh {
			t.Fatalf("unexpected active-human response: %#v", body)
		}
	}
	if got := request(); got.Code != http.StatusForbidden {
		t.Fatalf("replayed active-human proof accepted: %d", got.Code)
	}
}
