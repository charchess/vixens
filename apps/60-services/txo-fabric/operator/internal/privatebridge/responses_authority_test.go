package privatebridge

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type responsesResolutionFunc func(context.Context, ResponsesAuthorityRequest, time.Time) (ResponsesAuthorityResponse, error)

func (f responsesResolutionFunc) ResolveResponsesAuthority(ctx context.Context, q ResponsesAuthorityRequest, now time.Time) (ResponsesAuthorityResponse, error) {
	return f(ctx, q, now)
}

func TestResponsesAuthorityAcceptsOnlySignedServerScope(t *testing.T) {
	_, nonces := fixture()
	calls := 0
	h := &ResponsesAuthorityHandler{
		Contract: Contract{Audience: testAudience, KubernetesIssuer: testIssuer, BFFServiceAccount: testSA, AssertionKey: []byte(strings.Repeat("k", 32)), Now: func() time.Time { return testNow }},
		Tokens: tokenReviewerFunc(func(context.Context, string, string) (Workload, error) {
			return Workload{Authenticated: true, Username: testSA, Audiences: []string{testAudience}}, nil
		}),
		Nonces: nonces,
		Authorities: responsesResolutionFunc(func(_ context.Context, q ResponsesAuthorityRequest, _ time.Time) (ResponsesAuthorityResponse, error) {
			calls++
			if q.Operation != ResponsesAuthorityResolveTenant {
				return ResponsesAuthorityResponse{}, errors.New("unexpected operation")
			}
			return ResponsesAuthorityResponse{Tenant: &ResponsesTenantAuthority{
				TenantKey: q.TenantKey, FabricTenantID: q.FabricTenantID,
				TenantBundleUID: "6d0a9f4e-8024-4d39-97a5-cc1560f98dc7", Status: "Ready",
			}}, nil
		}),
	}
	q := ResponsesAuthorityRequest{Operation: ResponsesAuthorityResolveTenant, TenantKey: "hairem", FabricTenantID: "TEN00001"}
	a := ResponsesAuthorityAssertion{Method: http.MethodPost, Path: ResponsesAuthorityPath, Operation: q.Operation, TenantKey: q.TenantKey, FabricTenantID: q.FabricTenantID, ServiceAccount: testSA, Audience: testAudience, IssuedAt: testNow.Unix(), ExpiresAt: testNow.Add(20 * time.Second).Unix(), JTI: "responses-proof-012345678901234"}
	proof, err := SignResponsesAuthorityAssertion(h.Contract.AssertionKey, a)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, ResponsesAuthorityPath, strings.NewReader(`{"operation":"resolve-tenant","tenantKey":"hairem","fabricTenantId":"TEN00001"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtFor(testIssuer, testSA, testAudience))
	req.Header.Set("X-Fabric-Assertion", proof)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"tenantBundleUID":"6d0a9f4e-8024-4d39-97a5-cc1560f98dc7"`) || calls != 1 {
		t.Fatalf("valid private response authority denied or malformed: code=%d calls=%d body=%s", w.Code, calls, w.Body.String())
	}

	injected := httptest.NewRequest(http.MethodPost, ResponsesAuthorityPath, strings.NewReader(`{"operation":"resolve-tenant","tenantKey":"hairem","fabricTenantId":"TEN00001","endpoint":"http://evil"}`))
	injected.Header.Set("Content-Type", "application/json")
	injected.Header.Set("Authorization", "Bearer "+jwtFor(testIssuer, testSA, testAudience))
	injected.Header.Set("X-Fabric-Assertion", proof)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, injected)
	if w.Code != http.StatusForbidden || calls != 1 {
		t.Fatalf("client injection reached response authority: code=%d calls=%d", w.Code, calls)
	}
}

func TestResponsesAuthorityRejectsCrossOperationAndSnapshotConfusion(t *testing.T) {
	base := ResponsesAuthorityRequest{Operation: ResponsesAuthorityResolveAgent, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra"}
	if !validResponsesAuthorityRequest(base) {
		t.Fatal("valid agent-resolution request rejected")
	}
	cases := []ResponsesAuthorityRequest{
		{Operation: ResponsesAuthorityResolveTenant, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra"},
		{Operation: ResponsesAuthorityResolveAgent, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra", FabricUserID: "usr0123456789abcdef0123456789abcdef"},
		{Operation: ResponsesAuthorityAuthorizeChat, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra"},
		{Operation: ResponsesAuthorityVerifyRuntime, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra", TenantBundleUID: "6d0a9f4e-8024-4d39-97a5-cc1560f98dc7", TenantGeneration: 3, AgentIdentityUID: "9d0a9f4e-8024-4d39-97a5-cc1560f98dc7"},
	}
	for _, q := range cases {
		if validResponsesAuthorityRequest(q) {
			t.Fatalf("invalid responses authority request accepted: %#v", q)
		}
	}

	q := ResponsesAuthorityRequest{Operation: ResponsesAuthorityAuthorizeChat, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra", FabricUserID: "usr0123456789abcdef0123456789abcdef"}
	a := ResponsesAuthorityAssertion{Operation: q.Operation, TenantKey: q.TenantKey, FabricTenantID: q.FabricTenantID, AgentKey: q.AgentKey, FabricUserID: q.FabricUserID}
	if !sameResponsesAuthorityScope(a, q) {
		t.Fatal("matching authority scope rejected")
	}
	a.AgentKey = "other-agent"
	if sameResponsesAuthorityScope(a, q) {
		t.Fatal("agent assertion replay accepted for another agent")
	}
}

func TestResponsesAuthorityResponseIsOperationScoped(t *testing.T) {
	tenant := &ResponsesTenantAuthority{TenantKey: "hairem", FabricTenantID: "TEN00001", TenantBundleUID: "6d0a9f4e-8024-4d39-97a5-cc1560f98dc7", Status: "Ready"}
	agent := &ResponsesAgentAuthority{TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra", AgentIdentityUID: "9d0a9f4e-8024-4d39-97a5-cc1560f98dc7", AgentGeneration: 7, Status: "Ready"}
	cases := []struct {
		name string
		q    ResponsesAuthorityRequest
		out  ResponsesAuthorityResponse
		want bool
	}{
		{"tenant", ResponsesAuthorityRequest{Operation: ResponsesAuthorityResolveTenant, TenantKey: "hairem", FabricTenantID: "TEN00001"}, ResponsesAuthorityResponse{Tenant: tenant}, true},
		{"agent", ResponsesAuthorityRequest{Operation: ResponsesAuthorityResolveAgent, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra"}, ResponsesAuthorityResponse{Agent: agent}, true},
		{"authorization", ResponsesAuthorityRequest{Operation: ResponsesAuthorityAuthorizeChat, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra", FabricUserID: "usr0123456789abcdef0123456789abcdef"}, ResponsesAuthorityResponse{Authorization: &ResponsesAuthorizationAuthority{Allowed: true}}, true},
		{"runtime", ResponsesAuthorityRequest{Operation: ResponsesAuthorityVerifyRuntime, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra", TenantBundleUID: tenant.TenantBundleUID, TenantGeneration: 3, AgentIdentityUID: agent.AgentIdentityUID, AgentGeneration: 7}, ResponsesAuthorityResponse{Runtime: &ResponsesRuntimeAuthority{Attested: true}}, true},
		{"cross-operation tenant", ResponsesAuthorityRequest{Operation: ResponsesAuthorityResolveAgent, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra"}, ResponsesAuthorityResponse{Tenant: tenant}, false},
		{"mixed authority", ResponsesAuthorityRequest{Operation: ResponsesAuthorityResolveTenant, TenantKey: "hairem", FabricTenantID: "TEN00001"}, ResponsesAuthorityResponse{Tenant: tenant, Agent: agent}, false},
		{"authorization denied", ResponsesAuthorityRequest{Operation: ResponsesAuthorityAuthorizeChat, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra", FabricUserID: "usr0123456789abcdef0123456789abcdef"}, ResponsesAuthorityResponse{Authorization: &ResponsesAuthorizationAuthority{}}, false},
		{"runtime unverified", ResponsesAuthorityRequest{Operation: ResponsesAuthorityVerifyRuntime, TenantKey: "hairem", FabricTenantID: "TEN00001", AgentKey: "electra", TenantBundleUID: tenant.TenantBundleUID, TenantGeneration: 3, AgentIdentityUID: agent.AgentIdentityUID, AgentGeneration: 7}, ResponsesAuthorityResponse{Runtime: &ResponsesRuntimeAuthority{}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validResponsesAuthorityResponse(tc.q, tc.out); got != tc.want {
				t.Fatalf("validResponsesAuthorityResponse()=%t want %t for %#v", got, tc.want, tc.out)
			}
		})
	}
}
