package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRevokeModelAccessKeyIfExistsTreatsNotFoundAsSuccess(t *testing.T) {
	ctx := context.Background()
	calls := 0
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		if req.URL.Path != "/key/delete" {
			t.Fatalf("unexpected gateway path %q", req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer test-admin-token" {
			t.Fatal("missing gateway admin authorization")
		}
		var payload map[string]any
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer gateway.Close()
	t.Setenv("TXO_AI_GATEWAY_URL", gateway.URL)
	t.Setenv("TXO_AI_GATEWAY_ADMIN_TOKEN", "test-admin-token")

	if err := revokeModelAccessKeyIfExists(ctx, "txo-fabric:test:agent"); err != nil {
		t.Fatalf("alias not-found revocation must be idempotent: %v", err)
	}
	if err := revokeModelAccessKeyValueIfExists(ctx, "sk-test-old-key"); err != nil {
		t.Fatalf("key not-found revocation must be idempotent: %v", err)
	}
	if calls != 2 {
		t.Fatalf("delete calls=%d, want 2", calls)
	}
}


func TestGenerateScopedModelAccessKeySupportsTrustedModelAliases(t *testing.T) {
	var request map[string]any
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/key/generate" {
			t.Fatalf("unexpected gateway path %q", req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer test-admin-token" {
			t.Fatalf("unexpected gateway authorization header")
		}
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"sk-agent-test"}`))
	}))
	defer gateway.Close()

	t.Setenv("TXO_AI_GATEWAY_URL", gateway.URL)
	t.Setenv("TXO_AI_GATEWAY_ADMIN_TOKEN", "test-admin-token")

	key, err := generateScopedModelAccessKey(
		context.Background(),
		"txo-fabric:hairem:usr000001-agt00001",
		[]string{"txo-default"},
		map[string]string{"txo-default": "txo-tenant-hairem-default"},
		map[string]string{
			"tenant":    "hairem",
			"tenant_id": "TEN00001",
			"agent":     "usr000001-agt00001",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if key != "sk-agent-test" {
		t.Fatalf("generated key=%q, want sk-agent-test", key)
	}

	if got := request["key_alias"]; got != "txo-fabric:hairem:usr000001-agt00001" {
		t.Fatalf("key_alias=%#v", got)
	}
	models, ok := request["models"].([]any)
	if !ok || len(models) != 1 || models[0] != "txo-default" {
		t.Fatalf("models=%#v, want only public logical model txo-default", request["models"])
	}
	aliases, ok := request["aliases"].(map[string]any)
	if !ok || len(aliases) != 1 || aliases["txo-default"] != "txo-tenant-hairem-default" {
		t.Fatalf("aliases=%#v", request["aliases"])
	}
	metadata, ok := request["metadata"].(map[string]any)
	if !ok || metadata["tenant"] != "hairem" || metadata["tenant_id"] != "TEN00001" {
		t.Fatalf("metadata=%#v", request["metadata"])
	}

	for _, forbidden := range []string{"access_token", "refresh_token", "oauth_token", "provider_api_key"} {
		if _, exists := request[forbidden]; exists {
			t.Fatalf("gateway key payload unexpectedly contains secret field %q", forbidden)
		}
	}
}

func TestGenerateScopedModelAccessKeyOmitsEmptyAliases(t *testing.T) {
	var request map[string]any
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"sk-embedding-test"}`))
	}))
	defer gateway.Close()

	t.Setenv("TXO_AI_GATEWAY_URL", gateway.URL)
	t.Setenv("TXO_AI_GATEWAY_ADMIN_TOKEN", "test-admin-token")

	if _, err := generateScopedModelAccessKey(
		context.Background(),
		"txo-fabric:hairem:hindsight-embeddings",
		[]string{"txo-embedding"},
		nil,
		map[string]string{"tenant": "hairem", "component": "hindsight"},
	); err != nil {
		t.Fatal(err)
	}
	if _, exists := request["aliases"]; exists {
		t.Fatalf("empty aliases must not change the existing key contract: %#v", request["aliases"])
	}
}
