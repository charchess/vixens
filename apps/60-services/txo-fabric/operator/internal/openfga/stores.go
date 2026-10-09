// Package openfga contains the Fabric-only client for the single private OpenFGA
// service. It never accepts a caller-selected endpoint or store name.
package openfga

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	endpoint = "http://txo-fabric-openfga.txo-fabric-system.svc:8080"
	storePrefix = "txo-fabric-tenant-"
	maxPages = 100
	maxResponseBytes = 1048576
)

var (
	tenantIDPattern = regexp.MustCompile("^TEN[0-9]{5,}$")
	storeIDPattern = regexp.MustCompile("^[0-9A-HJKMNP-TV-Z]{26}$")
)

// Store binds an immutable Fabric tenant ID to one private OpenFGA store.
type Store struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

type listStoresResponse struct {
	Stores []Store `json:"stores"`
	ContinuationToken string `json:"continuation_token"`
}

// Client is only instantiated by trusted Fabric operator code, with its secret
// read from the platform-owned Kubernetes Secret (never from tenant intent).
type Client struct {
	token string
	httpClient *http.Client
}

// NewClient fixes the API address to the private ClusterIP name and enforces
// a timeout even if a caller passes a custom HTTP transport for unit tests.
func NewClient(presharedKey string, transport http.RoundTripper) (*Client, error) {
	if len(presharedKey) < 16 || strings.TrimSpace(presharedKey) != presharedKey ||
		strings.ContainsAny(presharedKey, "\r\n") {
		return nil, errors.New("missing or malformed Fabric OpenFGA service credential")
	}
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{token: presharedKey, httpClient: &http.Client{
		Transport: transport,
		Timeout: 3 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}, nil
}

func TenantStoreName(tenantID string) (string, error) {
	if !tenantIDPattern.MatchString(tenantID) {
		return "", errors.New("invalid immutable Fabric tenant ID")
	}
	return storePrefix + strings.ToLower(tenantID), nil
}

// EnsureTenantStore creates a store once, or adopts the uniquely named
// existing store. Store-name uniqueness is NOT guaranteed by OpenFGA: if
// duplicates exist, no result is trusted. Never delete stores on a retry.
func (c *Client) EnsureTenantStore(ctx context.Context, tenantID string) (Store, error) {
	name, err := TenantStoreName(tenantID)
	if err != nil {
		return Store{}, err
	}
	before, err := c.findUnique(ctx, name)
	if err != nil {
		return Store{}, err
	}
	if before.ID != "" {
		return before, nil
	}

	// A request may have succeeded upstream even if the reply was lost.
	// The following retry must first list/adopt, never blindly POST again.
	body, _ := json.Marshal(map[string]string{"name": name})
	data, err := c.request(ctx, http.MethodPost, "/stores", body)
	if err != nil {
		return Store{}, err
	}
	var created Store
	if err := json.Unmarshal(data, &created); err != nil ||
		created.Name != name || !storeIDPattern.MatchString(created.ID) {
		return Store{}, errors.New("OpenFGA create store returned invalid identity")
	}
	// Re-list after creation to detect name collisions/concurrent reconcilers.
	confirmed, err := c.findUnique(ctx, name)
	if err != nil {
		return Store{}, err
	}
	if confirmed.ID != created.ID {
		return Store{}, errors.New("OpenFGA newly created store is not uniquely observable")
	}
	return confirmed, nil
}

// ResolveTenantStore reads, but NEVER creates, the store. This allows the
// trusted Fabric backend to fail closed when provisioning is not complete.
func (c *Client) ResolveTenantStore(ctx context.Context, tenantID string) (Store, error) {
	name, err := TenantStoreName(tenantID)
	if err != nil {
		return Store{}, err
	}
	store, err := c.findUnique(ctx, name)
	if err != nil {
		return Store{}, err
	}
	if store.ID == "" {
		return Store{}, errors.New("Fabric tenant OpenFGA store is not provisioned")
	}
	return store, nil
}

func (c *Client) findUnique(ctx context.Context, name string) (Store, error) {
	var unique Store
	token := ""
	seen := make(map[string]bool)
	for page := 0; page < maxPages; page++ {
		query := url.Values{"name": {name}, "page_size": {"100"}}
		if token != "" {
			query.Set("continuation_token", token)
		}
		payload, err := c.request(ctx, http.MethodGet, "/stores?"+query.Encode(), nil)
		if err != nil {
			return Store{}, err
		}
		var result listStoresResponse
		if err := json.Unmarshal(payload, &result); err != nil {
			return Store{}, errors.New("invalid OpenFGA store discovery response")
		}
		for _, store := range result.Stores {
			if store.Name != name {
				// Defensive check: caller never decides names or accepts a
				// potentially fuzzy server-side filter.
				continue
			}
			if !storeIDPattern.MatchString(store.ID) || unique.ID != "" {
				return Store{}, errors.New("invalid or ambiguous OpenFGA tenant store")
			}
			unique = store
		}
		if result.ContinuationToken == "" {
			return unique, nil
		}
		if seen[result.ContinuationToken] {
			return Store{}, errors.New("OpenFGA discovery pagination repeated a cursor")
		}
		seen[result.ContinuationToken] = true
		token = result.ContinuationToken
	}
	return Store{}, errors.New("OpenFGA store discovery exceeded pagination limit")
}

func (c *Client) request(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if c == nil || c.httpClient == nil || c.token == "" {
		return nil, errors.New("OpenFGA client is uninitialized")
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint+path, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("unable to construct private OpenFGA request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("private OpenFGA service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if (method == http.MethodGet && resp.StatusCode != http.StatusOK) ||
		(method == http.MethodPost && resp.StatusCode != http.StatusCreated &&
			resp.StatusCode != http.StatusOK) {
		return nil, fmt.Errorf("private OpenFGA returned HTTP %d", resp.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil || len(payload) > maxResponseBytes {
		return nil, errors.New("invalid or oversized OpenFGA response")
	}
	return payload, nil
}
