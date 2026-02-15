package resolver

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// mockVaultReader is a test double for VaultReader.
type mockVaultReader struct {
	data     map[string]map[string]string
	errPaths map[string]error
	calls    atomic.Int64
}

func newMockVault() *mockVaultReader {
	return &mockVaultReader{
		data:     make(map[string]map[string]string),
		errPaths: make(map[string]error),
	}
}

func (m *mockVaultReader) withData(path string, kv map[string]string) *mockVaultReader {
	m.data[path] = kv
	return m
}

func (m *mockVaultReader) withError(path string, err error) *mockVaultReader {
	m.errPaths[path] = err
	return m
}

func (m *mockVaultReader) ReadKV(path string) (map[string]string, error) {
	m.calls.Add(1)

	if err, ok := m.errPaths[path]; ok {
		return nil, err
	}

	data, ok := m.data[path]
	if !ok {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	// Return a copy to verify resolver does not rely on mutation.
	cp := make(map[string]string, len(data))
	for k, v := range data {
		cp[k] = v
	}

	return cp, nil
}

func TestResolver_Resolve(t *testing.T) {
	vault := newMockVault().
		withData("secrets/dev/database", map[string]string{
			"url":        "postgres://dev:5432",
			"auth_token": "dev-token-123",
		}).
		withData("secrets/dev/stripe", map[string]string{
			"secret_key": "sk_test_xxx",
		})

	r := New(vault, "secrets")

	secrets := map[string]string{
		"DATABASE_URL":        "${env}/database/url",
		"DATABASE_AUTH_TOKEN": "${env}/database/auth_token",
		"STRIPE_SECRET_KEY":   "${env}/stripe/secret_key",
	}

	got, err := r.Resolve(secrets, "dev")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	expected := map[string]string{
		"DATABASE_URL":        "postgres://dev:5432",
		"DATABASE_AUTH_TOKEN": "dev-token-123",
		"STRIPE_SECRET_KEY":   "sk_test_xxx",
	}

	for k, want := range expected {
		if got[k] != want {
			t.Errorf("Resolve()[%q] = %q, want %q", k, got[k], want)
		}
	}
}

func TestResolver_ResolveEmptySecrets(t *testing.T) {
	vault := newMockVault()
	r := New(vault, "secrets")

	got, err := r.Resolve(map[string]string{}, "dev")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}

	if vault.calls.Load() != 0 {
		t.Error("expected no Vault calls for empty secrets")
	}
}

func TestResolver_ResolveSharedPaths(t *testing.T) {
	vault := newMockVault().
		withData("secrets/shared/openai", map[string]string{
			"api_key": "sk-openai-xxx",
		})

	r := New(vault, "secrets")

	secrets := map[string]string{
		"OPENAI_API_KEY": "shared/openai/api_key",
	}

	got, err := r.Resolve(secrets, "dev")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got["OPENAI_API_KEY"] != "sk-openai-xxx" {
		t.Errorf("OPENAI_API_KEY = %q, want %q", got["OPENAI_API_KEY"], "sk-openai-xxx")
	}
}

func TestResolver_ErrorHandling(t *testing.T) {
	vault := newMockVault().
		withData("secrets/dev/database", map[string]string{"url": "pg://localhost"}).
		withError("secrets/dev/stripe", fmt.Errorf("permission denied"))

	r := New(vault, "secrets")

	secrets := map[string]string{
		"DATABASE_URL":      "${env}/database/url",
		"STRIPE_SECRET_KEY": "${env}/stripe/secret_key",
	}

	_, err := r.Resolve(secrets, "dev")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolver_EmptyBasePath(t *testing.T) {
	vault := newMockVault().
		withData("dev/database", map[string]string{"url": "pg://localhost"})

	r := New(vault, "")

	secrets := map[string]string{
		"DATABASE_URL": "${env}/database/url",
	}

	got, err := r.Resolve(secrets, "dev")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got["DATABASE_URL"] != "pg://localhost" {
		t.Errorf("DATABASE_URL = %q, want %q", got["DATABASE_URL"], "pg://localhost")
	}
}

func TestResolver_ConcurrentExecution(t *testing.T) {
	vault := newMockVault()

	// Create many distinct paths to exercise concurrency.
	secrets := make(map[string]string)
	for i := range 20 {
		envVar := fmt.Sprintf("SECRET_%d", i)
		path := fmt.Sprintf("${env}/service%d/key", i)
		secrets[envVar] = path

		vaultPath := fmt.Sprintf("dev/service%d", i)
		vault.withData(vaultPath, map[string]string{"key": fmt.Sprintf("value_%d", i)})
	}

	r := New(vault, "", WithMaxConcurrency(5))

	got, err := r.Resolve(secrets, "dev")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if len(got) != 20 {
		t.Errorf("expected 20 resolved secrets, got %d", len(got))
	}

	for i := range 20 {
		envVar := fmt.Sprintf("SECRET_%d", i)
		want := fmt.Sprintf("value_%d", i)
		if got[envVar] != want {
			t.Errorf("%s = %q, want %q", envVar, got[envVar], want)
		}
	}
}

func TestResolver_WithCache(t *testing.T) {
	vault := newMockVault().
		withData("secrets/dev/database", map[string]string{
			"url": "pg://cached",
		})

	cache := NewCache(time.Minute)
	r := New(vault, "secrets", WithCache(cache))

	secrets := map[string]string{
		"DATABASE_URL": "${env}/database/url",
	}

	// First call should hit Vault.
	got1, err := r.Resolve(secrets, "dev")
	if err != nil {
		t.Fatalf("first Resolve() error = %v", err)
	}

	if got1["DATABASE_URL"] != "pg://cached" {
		t.Errorf("DATABASE_URL = %q, want %q", got1["DATABASE_URL"], "pg://cached")
	}

	firstCalls := vault.calls.Load()

	// Second call should hit cache, not Vault.
	got2, err := r.Resolve(secrets, "dev")
	if err != nil {
		t.Fatalf("second Resolve() error = %v", err)
	}

	if got2["DATABASE_URL"] != "pg://cached" {
		t.Errorf("cached DATABASE_URL = %q, want %q", got2["DATABASE_URL"], "pg://cached")
	}

	if vault.calls.Load() != firstCalls {
		t.Errorf("expected no additional Vault calls on cache hit, got %d total", vault.calls.Load())
	}
}

func TestResolver_MissingKeyInVaultData(t *testing.T) {
	vault := newMockVault().
		withData("secrets/dev/database", map[string]string{
			"url": "pg://localhost",
			// "auth_token" key is intentionally missing.
		})

	r := New(vault, "secrets")

	secrets := map[string]string{
		"DATABASE_URL":        "${env}/database/url",
		"DATABASE_AUTH_TOKEN": "${env}/database/auth_token",
	}

	got, err := r.Resolve(secrets, "dev")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got["DATABASE_URL"] != "pg://localhost" {
		t.Errorf("DATABASE_URL = %q, want %q", got["DATABASE_URL"], "pg://localhost")
	}

	if _, ok := got["DATABASE_AUTH_TOKEN"]; ok {
		t.Error("expected DATABASE_AUTH_TOKEN to be absent when key missing from Vault")
	}
}

func TestWithMaxConcurrency_IgnoresInvalid(t *testing.T) {
	r := New(newMockVault(), "", WithMaxConcurrency(0))
	if r.maxConcurrency != defaultMaxConcurrency {
		t.Errorf("maxConcurrency = %d, want %d", r.maxConcurrency, defaultMaxConcurrency)
	}

	r2 := New(newMockVault(), "", WithMaxConcurrency(-1))
	if r2.maxConcurrency != defaultMaxConcurrency {
		t.Errorf("maxConcurrency = %d, want %d", r2.maxConcurrency, defaultMaxConcurrency)
	}
}

func TestWithCache_IgnoresNil(t *testing.T) {
	r := New(newMockVault(), "", WithCache(nil))
	if r.cache != nil {
		t.Error("expected nil cache when WithCache(nil)")
	}
}
