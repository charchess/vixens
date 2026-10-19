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

type browserResolutionFunc func(context.Context, BrowserAuthorityRequest, time.Time) (BrowserAuthorityResponse, error)
func (f browserResolutionFunc) ResolveBrowserAuthority(ctx context.Context, q BrowserAuthorityRequest, now time.Time) (BrowserAuthorityResponse, error) { return f(ctx, q, now) }

func browserProof(kind, tenant, agent, jti string) BrowserAuthorityAssertion {
	return BrowserAuthorityAssertion{Method: http.MethodPost, Path: BrowserAuthorityPath, Kind: kind, AuthentikUserUUID: testUUID, TenantKey: tenant, AgentKey: agent, ServiceAccount: testSA, Audience: testAudience, IssuedAt: testNow.Unix(), ExpiresAt: testNow.Add(20*time.Second).Unix(), JTI: jti}
}

func browserHandler() (*BrowserAuthorityHandler, *memoryNonceStore) {
	_, nonces := fixture()
	return &BrowserAuthorityHandler{Contract: Contract{Audience: testAudience, KubernetesIssuer: testIssuer, BFFServiceAccount: testSA, AssertionKey: []byte(strings.Repeat("k", 32)), Now: func() time.Time { return testNow }}, Tokens: tokenReviewerFunc(func(context.Context, string, string) (Workload, error) { return Workload{Authenticated: true, Username: testSA, Audiences: []string{testAudience}}, nil }), Nonces: nonces, Authorities: browserResolutionFunc(func(_ context.Context, q BrowserAuthorityRequest, _ time.Time) (BrowserAuthorityResponse, error) {
		switch q.Kind {
		case BrowserAuthorityCatalog:
			return BrowserAuthorityResponse{Tenants: []BrowserTenant{{TenantKey: "hairem", FabricTenantID: "TEN00001", DisplayName: "hAIrem"}}}, nil
		case BrowserAuthorityTenant:
			return BrowserAuthorityResponse{Tenant: &BrowserTenantAuthority{BrowserTenant: BrowserTenant{TenantKey: q.TenantKey, FabricTenantID: "TEN00001", DisplayName: "hAIrem"}, FabricUserID: "usr000001", AccountActive: true, Fresh: true, CanEnter: true}}, nil
		case BrowserAuthorityAgent:
			return BrowserAuthorityResponse{Agent: &BrowserAgentAuthority{AgentKey: q.AgentKey, DisplayName: "Sam", AgentUID: "uid-1", Generation: 1}}, nil
		}
		return BrowserAuthorityResponse{}, errors.New("unexpected")
	})}, nonces
}

func serveBrowser(t *testing.T, h *BrowserAuthorityHandler, q BrowserAuthorityRequest, a BrowserAuthorityAssertion) *httptest.ResponseRecorder {
	t.Helper()
	proof, err := SignBrowserAuthorityAssertion(h.Contract.AssertionKey, a); if err != nil { t.Fatal(err) }
	raw := []byte(`{"kind":"` + q.Kind + `","authentikUserUUID":"` + q.AuthentikUserUUID + `","tenantKey":"` + q.TenantKey + `","agentKey":"` + q.AgentKey + `"}`)
	// Omit empty optional fields because strict JSON decoding distinguishes absent
	// from a browser trying to inject an unsupported field.
	if q.Kind == BrowserAuthorityCatalog { raw = []byte(`{"kind":"catalog","authentikUserUUID":"` + q.AuthentikUserUUID + `"}`) }
	if q.Kind == BrowserAuthorityTenant { raw = []byte(`{"kind":"tenant","authentikUserUUID":"` + q.AuthentikUserUUID + `","tenantKey":"` + q.TenantKey + `"}`) }
	req := httptest.NewRequest(http.MethodPost, BrowserAuthorityPath, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json"); req.Header.Set("Authorization", "Bearer "+jwtFor(testIssuer, testSA, testAudience)); req.Header.Set("X-Fabric-Assertion", proof)
	w := httptest.NewRecorder(); h.ServeHTTP(w, req); return w
}

func TestBrowserAuthorityRouteReturnsOnlyStrictNarrowShapes(t *testing.T) {
	h, _ := browserHandler()
	for _, tc := range []struct { kind, tenant, agent, want string }{
		{BrowserAuthorityCatalog, "", "", `"tenants"`},
		{BrowserAuthorityTenant, "hairem", "", `"fabricUserId":"usr000001"`},
		{BrowserAuthorityAgent, "hairem", "sam", `"agentUID":"uid-1"`},
	} {
		q := BrowserAuthorityRequest{Kind: tc.kind, AuthentikUserUUID: testUUID, TenantKey: tc.tenant, AgentKey: tc.agent}
		w := serveBrowser(t, h, q, browserProof(tc.kind, tc.tenant, tc.agent, "browser-proof-01234567890123456"+tc.kind))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), tc.want) || strings.Contains(w.Body.String(), "endpoint") || strings.Contains(w.Body.String(), "credential") { t.Fatalf("unsafe browser output: %d %s", w.Code, w.Body.String()) }
	}
}

func TestBrowserAuthorityRejectsLegacyAssertionAndClientInjection(t *testing.T) {
	h, store := browserHandler()
	q := BrowserAuthorityRequest{Kind: BrowserAuthorityTenant, AuthentikUserUUID: testUUID, TenantKey: "hairem"}
	legacy, err := SignAssertion(h.Contract.AssertionKey, proofFor("hairem", testUUID, "legacy-proof-01234567890123456")); if err != nil { t.Fatal(err) }
	req := httptest.NewRequest(http.MethodPost, BrowserAuthorityPath, strings.NewReader(`{"kind":"tenant","authentikUserUUID":"`+testUUID+`","tenantKey":"hairem","endpoint":"http://evil"}`))
	req.Header.Set("Content-Type", "application/json"); req.Header.Set("Authorization", "Bearer "+jwtFor(testIssuer, testSA, testAudience)); req.Header.Set("X-Fabric-Assertion", legacy)
	w := httptest.NewRecorder(); h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden || store.calls != 0 { t.Fatal("legacy assertion or injected field reached authority") }
	w = serveBrowser(t, h, q, browserProof(BrowserAuthorityTenant, "hairem", "", "browser-proof-01234567890123456"))
	if w.Code != http.StatusOK { t.Fatalf("valid narrow assertion denied: %d", w.Code) }
}
