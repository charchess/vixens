package controller

import (
	"bytes"
	"context"
	"encoding/json"
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

func aiGatewayURL() string {
	if value := strings.TrimSpace(os.Getenv("TXO_AI_GATEWAY_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultAIGatewayURL
}

func aiGatewayAdminToken() string {
	return strings.TrimSpace(os.Getenv("TXO_AI_GATEWAY_ADMIN_TOKEN"))
}

func generateModelAccessKey(ctx context.Context, agent *fabricv1alpha1.AgentIdentity, tenant *fabricv1alpha1.TenantBundle) (string, error) {
	payload := map[string]any{
		"key_alias": modelAccessKeyAlias(agent, tenant),
		"models":    []string{defaultAIGatewayModel},
		"metadata": map[string]string{
			"tenant": tenant.Name, "tenant_id": tenant.Spec.TenantID,
			"agent": agent.Spec.AgentKey, "agent_id": agent.Name,
		},
	}
	var response generateKeyResponse
	if err := aiGatewayJSON(ctx, "/key/generate", payload, &response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.Key) == "" {
		return "", fmt.Errorf("TXO AI gateway returned an empty virtual key")
	}
	return response.Key, nil
}

func revokeModelAccessKey(ctx context.Context, alias string) error {
	return aiGatewayJSON(ctx, "/key/delete", map[string]any{"key_aliases": []string{alias}}, nil)
}

func aiGatewayJSON(ctx context.Context, path string, payload any, out any) error {
	token := aiGatewayAdminToken()
	if token == "" {
		return fmt.Errorf("TXO AI gateway admin credential is not configured")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode TXO AI gateway request: %w", err)
	}
	reqCtx, cancel := context.WithTimeout(ctx, modelAccessRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, aiGatewayURL()+path, bytes.NewReader(body))
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
		return fmt.Errorf("TXO AI gateway request failed with HTTP %d", resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode TXO AI gateway response: %w", err)
	}
	return nil
}
