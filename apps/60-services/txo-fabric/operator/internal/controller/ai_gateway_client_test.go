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
