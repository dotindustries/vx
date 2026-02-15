package token

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRenewOnce_NoRenewalNeeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/token/lookup-self" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := tokenLookupResponse{}
		resp.Data.TTL = 7200
		resp.Data.Renewable = true
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	writeTokenTo(tokenPath, "s.valid-token")

	renewer := NewTokenRenewer(srv.URL,
		WithTokenPath(tokenPath),
		WithCheckInterval(time.Second),
	)

	err := renewer.RenewOnce(context.Background())
	if err != nil {
		t.Fatalf("RenewOnce() error = %v", err)
	}

	// Token should remain unchanged since no renewal was needed.
	got, _ := readTokenFrom(tokenPath)
	if got != "s.valid-token" {
		t.Errorf("token = %q, want %q", got, "s.valid-token")
	}
}

func TestRenewOnce_RenewsLowTTL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/token/lookup-self":
			resp := tokenLookupResponse{}
			resp.Data.TTL = 300
			resp.Data.Renewable = true
			json.NewEncoder(w).Encode(resp)
		case "/v1/auth/token/renew-self":
			if r.Method != http.MethodPost {
				t.Errorf("renew-self method = %s, want POST", r.Method)
			}
			resp := tokenRenewResponse{}
			resp.Auth.ClientToken = "s.renewed-token"
			json.NewEncoder(w).Encode(resp)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	writeTokenTo(tokenPath, "s.old-token")

	renewer := NewTokenRenewer(srv.URL, WithTokenPath(tokenPath))

	err := renewer.RenewOnce(context.Background())
	if err != nil {
		t.Fatalf("RenewOnce() error = %v", err)
	}

	got, _ := readTokenFrom(tokenPath)
	if got != "s.renewed-token" {
		t.Errorf("token = %q, want %q", got, "s.renewed-token")
	}
}

func TestRenewOnce_NonRenewableToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := tokenLookupResponse{}
		resp.Data.TTL = 100
		resp.Data.Renewable = false
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	writeTokenTo(tokenPath, "s.non-renewable")

	renewer := NewTokenRenewer(srv.URL, WithTokenPath(tokenPath))

	err := renewer.RenewOnce(context.Background())
	if err != nil {
		t.Fatalf("RenewOnce() error = %v", err)
	}

	// Token should remain unchanged for non-renewable tokens.
	got, _ := readTokenFrom(tokenPath)
	if got != "s.non-renewable" {
		t.Errorf("token = %q, want %q", got, "s.non-renewable")
	}
}

func TestRenewOnce_MissingToken(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "nonexistent")

	renewer := NewTokenRenewer("http://localhost:8200", WithTokenPath(tokenPath))

	err := renewer.RenewOnce(context.Background())
	if err == nil {
		t.Fatal("RenewOnce() expected error for missing token, got nil")
	}
}

func TestRenewOnce_VaultError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	writeTokenTo(tokenPath, "s.bad-token")

	renewer := NewTokenRenewer(srv.URL, WithTokenPath(tokenPath))

	err := renewer.RenewOnce(context.Background())
	if err == nil {
		t.Fatal("RenewOnce() expected error for 403 response, got nil")
	}
}

func TestRenewOnce_VaultTokenHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Vault-Token")
		if got != "s.header-check" {
			t.Errorf("X-Vault-Token = %q, want %q", got, "s.header-check")
		}
		resp := tokenLookupResponse{}
		resp.Data.TTL = 7200
		resp.Data.Renewable = true
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	writeTokenTo(tokenPath, "s.header-check")

	renewer := NewTokenRenewer(srv.URL, WithTokenPath(tokenPath))
	renewer.RenewOnce(context.Background())
}

func TestNeedsReauth_MissingToken(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "nonexistent")

	renewer := NewTokenRenewer("http://localhost:8200", WithTokenPath(tokenPath))

	if !renewer.NeedsReauth() {
		t.Error("NeedsReauth() = false, want true for missing token")
	}
}

func TestNeedsReauth_EmptyToken(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	os.WriteFile(tokenPath, []byte(""), filePerms)

	renewer := NewTokenRenewer("http://localhost:8200", WithTokenPath(tokenPath))

	if !renewer.NeedsReauth() {
		t.Error("NeedsReauth() = false, want true for empty token")
	}
}

func TestNeedsReauth_ExpiredToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := tokenLookupResponse{}
		resp.Data.TTL = 0
		resp.Data.ExpireTime = "2020-01-01T00:00:00Z"
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	writeTokenTo(tokenPath, "s.expired")

	renewer := NewTokenRenewer(srv.URL, WithTokenPath(tokenPath))

	if !renewer.NeedsReauth() {
		t.Error("NeedsReauth() = false, want true for expired token")
	}
}

func TestNeedsReauth_ValidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := tokenLookupResponse{}
		resp.Data.TTL = 3600
		resp.Data.Renewable = true
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	writeTokenTo(tokenPath, "s.valid")

	renewer := NewTokenRenewer(srv.URL, WithTokenPath(tokenPath))

	if renewer.NeedsReauth() {
		t.Error("NeedsReauth() = true, want false for valid token")
	}
}

func TestNeedsRenewal(t *testing.T) {
	tests := []struct {
		name string
		ttl  int
		want bool
	}{
		{name: "high TTL", ttl: 7200, want: false},
		{name: "at threshold", ttl: 1800, want: false},
		{name: "below threshold", ttl: 1799, want: true},
		{name: "very low TTL", ttl: 60, want: true},
		{name: "zero TTL", ttl: 0, want: false},
		{name: "negative TTL", ttl: -1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsRenewal(tt.ttl)
			if got != tt.want {
				t.Errorf("needsRenewal(%d) = %v, want %v", tt.ttl, got, tt.want)
			}
		})
	}
}

func TestNewTokenRenewer_Defaults(t *testing.T) {
	r := NewTokenRenewer("http://vault:8200")

	if r.vaultAddr != "http://vault:8200" {
		t.Errorf("vaultAddr = %q, want %q", r.vaultAddr, "http://vault:8200")
	}
	if r.checkInterval != defaultCheckInterval {
		t.Errorf("checkInterval = %v, want %v", r.checkInterval, defaultCheckInterval)
	}
}

func TestNewTokenRenewer_TrailingSlash(t *testing.T) {
	r := NewTokenRenewer("http://vault:8200/")

	if r.vaultAddr != "http://vault:8200" {
		t.Errorf("vaultAddr = %q, want %q", r.vaultAddr, "http://vault:8200")
	}
}

func TestNewTokenRenewer_Options(t *testing.T) {
	r := NewTokenRenewer("http://vault:8200",
		WithCheckInterval(5*time.Minute),
		WithTokenPath("/custom/path"),
	)

	if r.checkInterval != 5*time.Minute {
		t.Errorf("checkInterval = %v, want %v", r.checkInterval, 5*time.Minute)
	}
	if r.tokenPath != "/custom/path" {
		t.Errorf("tokenPath = %q, want %q", r.tokenPath, "/custom/path")
	}
}
