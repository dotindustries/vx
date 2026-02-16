package token

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultCheckInterval = 60 * time.Second

// TokenRenewer handles automatic renewal of Vault tokens before they expire.
type TokenRenewer struct {
	vaultAddr     string
	tokenPath     string
	checkInterval time.Duration
	httpClient    *http.Client
}

// RenewerOption configures a TokenRenewer.
type RenewerOption func(*TokenRenewer)

// WithCheckInterval sets how frequently the renewer checks token TTL.
func WithCheckInterval(d time.Duration) RenewerOption {
	return func(r *TokenRenewer) {
		r.checkInterval = d
	}
}

// WithTokenPath overrides the default token sink file path.
func WithTokenPath(path string) RenewerOption {
	return func(r *TokenRenewer) {
		r.tokenPath = path
	}
}

// withHTTPClient overrides the HTTP client used for Vault API calls. This is
// intended for testing only.
func withHTTPClient(c *http.Client) RenewerOption {
	return func(r *TokenRenewer) {
		r.httpClient = c
	}
}

// NewTokenRenewer creates a TokenRenewer configured for the given Vault address.
func NewTokenRenewer(vaultAddr string, opts ...RenewerOption) *TokenRenewer {
	r := &TokenRenewer{
		vaultAddr:     strings.TrimRight(vaultAddr, "/"),
		tokenPath:     TokenPath(),
		checkInterval: defaultCheckInterval,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// tokenLookupResponse represents the relevant fields from Vault's
// auth/token/lookup-self response.
type tokenLookupResponse struct {
	Data struct {
		TTL         int  `json:"ttl"`
		CreationTTL int  `json:"creation_ttl"`
		ExpireTime  any  `json:"expire_time"`
		Renewable   bool `json:"renewable"`
	} `json:"data"`
}

// tokenRenewResponse represents the relevant fields from Vault's
// auth/token/renew-self response.
type tokenRenewResponse struct {
	Auth struct {
		ClientToken string `json:"client_token"`
	} `json:"auth"`
}

// RenewOnce performs a single renewal check. It reads the current token, looks
// up its TTL, and renews it if the remaining TTL is below 50% of the max TTL.
// Returns nil if no renewal was needed.
func (r *TokenRenewer) RenewOnce(ctx context.Context) error {
	tok, err := readTokenFrom(r.tokenPath)
	if err != nil {
		return fmt.Errorf("renew: %w", err)
	}

	lookup, err := r.lookupToken(ctx, tok)
	if err != nil {
		return fmt.Errorf("renew: lookup: %w", err)
	}

	if !lookup.Data.Renewable {
		return nil
	}

	if !needsRenewal(lookup.Data.TTL, lookup.Data.CreationTTL) {
		return nil
	}

	newToken, err := r.renewToken(ctx, tok)
	if err != nil {
		return fmt.Errorf("renew: renew-self: %w", err)
	}

	if err := writeTokenTo(r.tokenPath, newToken); err != nil {
		return fmt.Errorf("renew: write: %w", err)
	}

	return nil
}

// NeedsReauth reports whether the token is missing, empty, or expired and
// cannot be renewed (requiring a full re-authentication).
func (r *TokenRenewer) NeedsReauth() bool {
	tok, err := readTokenFrom(r.tokenPath)
	if err != nil || tok == "" {
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	lookup, err := r.lookupToken(ctx, tok)
	if err != nil {
		return true
	}

	return lookup.Data.TTL <= 0 && lookup.Data.ExpireTime != nil
}

// needsRenewal returns true when the remaining TTL is below the renewal
// threshold. When creationTTL is known (> 0) the threshold is 50% of the
// original lease. When creationTTL is unknown (0) we always renew — Vault
// renewal is idempotent so an extra POST /renew-self per check interval is
// harmless and avoids missing the window for tokens with unknown lifetimes.
func needsRenewal(ttlSeconds, creationTTL int) bool {
	if ttlSeconds <= 0 {
		return false
	}
	if creationTTL > 0 {
		return ttlSeconds < creationTTL/2
	}
	return true // unknown lifetime — always renew to be safe
}

// lookupToken calls Vault's auth/token/lookup-self endpoint.
func (r *TokenRenewer) lookupToken(ctx context.Context, tok string) (*tokenLookupResponse, error) {
	url := r.vaultAddr + "/v1/auth/token/lookup-self"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Vault-Token", tok)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result tokenLookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// renewToken calls Vault's auth/token/renew-self endpoint and returns the new
// client token.
func (r *TokenRenewer) renewToken(ctx context.Context, tok string) (string, error) {
	url := r.vaultAddr + "/v1/auth/token/renew-self"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Vault-Token", tok)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result tokenRenewResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.Auth.ClientToken == "" {
		return "", fmt.Errorf("empty client token in response")
	}

	return result.Auth.ClientToken, nil
}
