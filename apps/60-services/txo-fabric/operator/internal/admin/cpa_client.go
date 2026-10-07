package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CPAClient struct {
	baseURL            string
	managementCredential string
	httpClient         *http.Client
}

type OAuthStart struct {
	Status string `json:"status"`
	URL    string `json:"url"`
	State  string `json:"state"`
}

type OAuthStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func NewCPAClient(baseURL, managementCredential string) *CPAClient {
	return &CPAClient{
		baseURL:            strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		managementCredential: strings.TrimSpace(managementCredential),
		httpClient:         &http.Client{Timeout: 15 * time.Second},
	}
}

func NewCPAClientWithHTTPClient(baseURL, managementCredential string, httpClient *http.Client) *CPAClient {
	client := NewCPAClient(baseURL, managementCredential)
	if httpClient != nil {
		client.httpClient = httpClient
	}
	return client
}

func (c *CPAClient) StartCodexOAuth(ctx context.Context) (OAuthStart, error) {
	var out OAuthStart
	if err := c.getJSON(ctx, "/v0/management/codex-auth-url?is_webui=true", &out); err != nil {
		return OAuthStart{}, err
	}
	if out.Status != "ok" || strings.TrimSpace(out.URL) == "" || strings.TrimSpace(out.State) == "" {
		return OAuthStart{}, fmt.Errorf("CPA returned incomplete Codex OAuth start response")
	}
	return out, nil
}

func (c *CPAClient) SubmitOAuthCallback(ctx context.Context, provider, state, redirectURL string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	state = strings.TrimSpace(state)
	redirectURL = strings.TrimSpace(redirectURL)
	if provider == "" {
		return fmt.Errorf("OAuth provider is required")
	}
	if state == "" {
		return fmt.Errorf("OAuth state is required")
	}
	if redirectURL == "" {
		return fmt.Errorf("OAuth callback URL is required")
	}
	payload, err := json.Marshal(map[string]string{
		"provider":     provider,
		"state":        state,
		"redirect_url": redirectURL,
	})
	if err != nil {
		return fmt.Errorf("encode OAuth callback: %w", err)
	}
	req, err := c.newRequestWithBody(ctx, http.MethodPost, "/v0/management/oauth-callback", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CPA OAuth callback request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("CPA OAuth callback failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *CPAClient) GetOAuthStatus(ctx context.Context, state string) (OAuthStatus, error) {
	state = strings.TrimSpace(state)
	if state == "" {
		return OAuthStatus{}, fmt.Errorf("OAuth state is required")
	}
	var out OAuthStatus
	path := "/v0/management/get-auth-status?state=" + url.QueryEscape(state)
	if err := c.getJSON(ctx, path, &out); err != nil {
		return OAuthStatus{}, err
	}
	if out.Status == "" {
		return OAuthStatus{}, fmt.Errorf("CPA returned OAuth status without status field")
	}
	return out, nil
}

func (c *CPAClient) CancelOAuth(ctx context.Context, state string) error {
	state = strings.TrimSpace(state)
	if state == "" {
		return nil
	}
	req, err := c.newRequest(ctx, http.MethodDelete, "/v0/management/oauth-session?state="+url.QueryEscape(state))
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CPA OAuth cancel request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("CPA OAuth cancel failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *CPAClient) getJSON(ctx context.Context, path string, out any) error {
	req, err := c.newRequest(ctx, http.MethodGet, path)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CPA management request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read CPA management response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("CPA management request failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode CPA management response: %w", err)
	}
	return nil
}

func (c *CPAClient) newRequest(ctx context.Context, method, path string) (*http.Request, error) {
	return c.newRequestWithBody(ctx, method, path, nil)
}

func (c *CPAClient) newRequestWithBody(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	if c == nil || strings.TrimSpace(c.baseURL) == "" {
		return nil, fmt.Errorf("CPA base URL is required")
	}
	if strings.TrimSpace(c.managementCredential) == "" {
		return nil, fmt.Errorf("CPA management password is required")
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("build CPA management request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.managementCredential)
	return req, nil
}
