package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCPAClientRemoteCodexOAuthFlow(t *testing.T) {
	const managementCredential = "management-secret"
	var callbackSeen bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+managementCredential {
			t.Fatalf("Authorization=%q", got)
		}
		switch r.URL.Path {
		case "/v0/management/codex-auth-url":
			if r.URL.Query().Get("is_webui") != "true" {
				t.Fatalf("is_webui=%q want true", r.URL.Query().Get("is_webui"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"url":    "https://auth.openai.example/authorize",
				"state":  "oauth-state",
			})
		case "/v0/management/oauth-callback":
			if r.Method != http.MethodPost {
				t.Fatalf("callback method=%s", r.Method)
			}
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["provider"] != "codex" || body["state"] != "oauth-state" || body["redirect_url"] != "http://localhost:1455/auth/callback?code=abc&state=oauth-state" {
				t.Fatalf("callback body=%#v", body)
			}
			callbackSeen = true
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/v0/management/get-auth-status":
			if r.URL.Query().Get("state") != "oauth-state" {
				t.Fatalf("status state=%q", r.URL.Query().Get("state"))
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewCPAClientWithHTTPClient(server.URL, managementCredential, server.Client())
	start, err := client.StartCodexOAuth(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if start.State != "oauth-state" || start.URL != "https://auth.openai.example/authorize" {
		t.Fatalf("start=%#v", start)
	}
	if err := client.SubmitOAuthCallback(context.Background(), "codex", start.State, "http://localhost:1455/auth/callback?code=abc&state=oauth-state"); err != nil {
		t.Fatal(err)
	}
	if !callbackSeen {
		t.Fatal("OAuth callback was not submitted")
	}
	status, err := client.GetOAuthStatus(context.Background(), start.State)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "ok" {
		t.Fatalf("status=%#v", status)
	}
}

func TestCPAClientRequiresManagementCredential(t *testing.T) {
	client := NewCPAClient("http://example.invalid", "")
	if _, err := client.StartCodexOAuth(context.Background()); err == nil {
		t.Fatal("expected missing management password to fail closed")
	}
}
