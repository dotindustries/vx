package vault

import (
	"fmt"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

// Client wraps the official HashiCorp Vault API client with a configured
// base path for KV v2 secret reads.
type Client struct {
	inner    *vaultapi.Client
	basePath string
}

// NewClient creates a new Vault API client pointed at the given address.
// The basePath is the KV v2 mount point (e.g. "secret").
func NewClient(address string, basePath string) (*Client, error) {
	if address == "" {
		return nil, fmt.Errorf("vault address is required")
	}

	cfg := vaultapi.DefaultConfig()
	cfg.Address = address

	inner, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating vault client: %w", err)
	}

	return &Client{
		inner:    inner,
		basePath: basePath,
	}, nil
}

// NewClientWithToken creates a new Vault API client with an existing auth token.
func NewClientWithToken(address string, basePath string, token string) (*Client, error) {
	client, err := NewClient(address, basePath)
	if err != nil {
		return nil, err
	}

	client.inner.SetToken(token)

	return client, nil
}

// Token returns the current authentication token.
func (c *Client) Token() string {
	return c.inner.Token()
}

// SetToken sets the authentication token on the client.
func (c *Client) SetToken(token string) {
	c.inner.SetToken(token)
}

// TokenTTL looks up the current token and returns its remaining TTL.
func (c *Client) TokenTTL() (time.Duration, error) {
	secret, err := c.inner.Auth().Token().LookupSelf()
	if err != nil {
		return 0, fmt.Errorf("looking up token TTL: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return 0, fmt.Errorf("looking up token TTL: empty response")
	}

	ttl, err := secret.TokenTTL()
	if err != nil {
		return 0, fmt.Errorf("parsing token TTL: %w", err)
	}

	return ttl, nil
}

// IsAuthenticated reports whether the client has a token that has not expired.
// Returns false if no token is set or if the token lookup fails.
func (c *Client) IsAuthenticated() bool {
	if c.inner.Token() == "" {
		return false
	}

	ttl, err := c.TokenTTL()
	if err != nil {
		return false
	}

	return ttl > 0
}
