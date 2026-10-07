package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
)

const modelAccessRequestTimeout = 15 * time.Second

type generateKeyResponse struct {
	Key string `json:"key"`
}

type aiGatewayHTTPError struct {
	StatusCode int
}

func (e *aiGatewayHTTPError) Error() string {
	return fmt.Sprintf("TXO AI gateway request failed with HTTP %d", e.StatusCode)
}

func aiGatewayURL() string {
	if value := strings.TrimSpace(os.Getenv("TXO_AI_GATEWAY_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultAIGatewayURL
}

func aiGatewayAdminToken() string {
	return strings.TrimSpace(os.Getenv("TXO_AI_GATEWAY_ADMIN_TOKEN"))
}

func generateModelAccessKey(ctx context.Context, backend modelAccessBackend, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) (string, error) {
	return generateScopedModelAccessKeyWithBackend(ctx, backend, modelAccessKeyAlias(agent, tenant), []string{backend.Model}, nil, map[string]string{
		"tenant": tenant.Name, "tenant_id": tenant.Spec.TenantID,
		"agent": agent.Spec.AgentKey, "agent_id": agent.Name,
	})
}

func generateHindsightEmbeddingAccessKey(ctx context.Context, tenant *fabricv1alpha1.TenantBundle) (string, error) {
	return generateScopedModelAccessKey(ctx, hindsightEmbeddingKeyAlias(tenant), []string{defaultAIEmbeddingModel}, nil, map[string]string{
		"tenant": tenant.Name, "tenant_id": tenant.Spec.TenantID,
		"component": "hindsight", "capability": "embeddings",
	})
}

func generateScopedModelAccessKey(ctx context.Context, alias string, models []string, modelAliases map[string]string, metadata map[string]string) (string, error) {
	return generateScopedModelAccessKeyWithBackend(ctx, sharedModelAccessBackend(), alias, models, modelAliases, metadata)
}

func generateScopedModelAccessKeyWithBackend(ctx context.Context, backend modelAccessBackend, alias string, models []string, modelAliases map[string]string, metadata map[string]string) (string, error) {
	payload := map[string]any{
		"key_alias": alias,
		"models":    models,
		"metadata":  metadata,
	}
	if len(modelAliases) > 0 {
		payload["aliases"] = modelAliases
	}
	var response generateKeyResponse
	if err := aiGatewayJSONWithBackend(ctx, backend, "/key/generate", payload, &response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.Key) == "" {
		return "", fmt.Errorf("TXO AI gateway returned an empty virtual key")
	}
	return response.Key, nil
}

func revokeModelAccessKey(ctx context.Context, alias string) error {
	return revokeModelAccessKeyWithBackend(ctx, sharedModelAccessBackend(), alias)
}

func revokeModelAccessKeyWithBackend(ctx context.Context, backend modelAccessBackend, alias string) error {
	return aiGatewayJSONWithBackend(ctx, backend, "/key/delete", map[string]any{"key_aliases": []string{alias}}, nil)
}

func revokeModelAccessKeyIfExists(ctx context.Context, alias string) error {
	return ignoreAIGatewayNotFound(revokeModelAccessKey(ctx, alias))
}

func revokeModelAccessKeyIfExistsWithBackend(ctx context.Context, backend modelAccessBackend, alias string) error {
	return ignoreAIGatewayNotFound(revokeModelAccessKeyWithBackend(ctx, backend, alias))
}

func revokeModelAccessKeyValueIfExists(ctx context.Context, key string) error {
	return revokeModelAccessKeyValueIfExistsWithBackend(ctx, sharedModelAccessBackend(), key)
}

func revokeModelAccessKeyValueIfExistsWithBackend(ctx context.Context, backend modelAccessBackend, key string) error {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	err := aiGatewayJSONWithBackend(ctx, backend, "/key/delete", map[string]any{"keys": []string{key}}, nil)
	return ignoreAIGatewayNotFound(err)
}

func ignoreAIGatewayNotFound(err error) error {
	if err == nil {
		return nil
	}
	var httpErr *aiGatewayHTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
}

func aiGatewayJSON(ctx context.Context, path string, payload any, out any) error {
	return aiGatewayJSONWithBackend(ctx, sharedModelAccessBackend(), path, payload, out)
}

func aiGatewayJSONWithBackend(ctx context.Context, backend modelAccessBackend, path string, payload any, out any) error {
	token := strings.TrimSpace(backend.AdminToken)
	if token == "" {
		return fmt.Errorf("TXO AI gateway admin credential is not configured for backend %q", backend.ID)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode TXO AI gateway request: %w", err)
	}
	reqCtx, cancel := context.WithTimeout(ctx, modelAccessRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, strings.TrimRight(backend.URL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build TXO AI gateway request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("TXO AI gateway request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &aiGatewayHTTPError{StatusCode: resp.StatusCode}
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode TXO AI gateway response: %w", err)
	}
	return nil
}
