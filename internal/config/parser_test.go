package config

import (
	"path/filepath"
	"testing"
)

func TestLoadRootConfig(t *testing.T) {
	path := filepath.Join("testdata", "root", "vx.toml")

	cfg, err := LoadRootConfig(path)
	if err != nil {
		t.Fatalf("LoadRootConfig() error = %v", err)
	}

	if cfg.Vault.Address != "https://vault.example.com" {
		t.Errorf("Vault.Address = %q, want %q", cfg.Vault.Address, "https://vault.example.com")
	}
	if cfg.Vault.AuthMethod != "oidc" {
		t.Errorf("Vault.AuthMethod = %q, want %q", cfg.Vault.AuthMethod, "oidc")
	}
	if cfg.Vault.AuthRole != "admin" {
		t.Errorf("Vault.AuthRole = %q, want %q", cfg.Vault.AuthRole, "admin")
	}
	if cfg.Vault.BasePath != "secret" {
		t.Errorf("Vault.BasePath = %q, want %q", cfg.Vault.BasePath, "secret")
	}

	if cfg.Environments.Default != "dev" {
		t.Errorf("Environments.Default = %q, want %q", cfg.Environments.Default, "dev")
	}
	if len(cfg.Environments.Available) != 3 {
		t.Errorf("Environments.Available length = %d, want 3", len(cfg.Environments.Available))
	}

	if len(cfg.Workspaces) != 2 {
		t.Errorf("Workspaces length = %d, want 2", len(cfg.Workspaces))
	}

	if cfg.Secrets["DATABASE_URL"] != "${env}/database/url" {
		t.Errorf("Secrets[DATABASE_URL] = %q, want %q", cfg.Secrets["DATABASE_URL"], "${env}/database/url")
	}
	if cfg.Secrets["OPENAI_API_KEY"] != "shared/openai/api_key" {
		t.Errorf("Secrets[OPENAI_API_KEY] = %q, want %q", cfg.Secrets["OPENAI_API_KEY"], "shared/openai/api_key")
	}
}

func TestLoadRootConfig_NotFound(t *testing.T) {
	_, err := LoadRootConfig("nonexistent/vx.toml")
	if err == nil {
		t.Fatal("LoadRootConfig() expected error for missing file")
	}
}

func TestLoadRootConfig_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vx.toml")
	writeTestFile(t, path, "this is not valid [toml")

	_, err := LoadRootConfig(path)
	if err == nil {
		t.Fatal("LoadRootConfig() expected error for invalid TOML")
	}
}

func TestLoadWorkspaceConfig(t *testing.T) {
	path := filepath.Join("testdata", "workspace", "vx.toml")

	cfg, err := LoadWorkspaceConfig(path)
	if err != nil {
		t.Fatalf("LoadWorkspaceConfig() error = %v", err)
	}

	if cfg.Secrets["TURSO_PLATFORM_TOKEN"] != "${env}/database/platform_token" {
		t.Errorf(
			"Secrets[TURSO_PLATFORM_TOKEN] = %q, want %q",
			cfg.Secrets["TURSO_PLATFORM_TOKEN"],
			"${env}/database/platform_token",
		)
	}
	if cfg.Secrets["REDIS_URL"] != "${env}/cache/redis_url" {
		t.Errorf("Secrets[REDIS_URL] = %q, want %q", cfg.Secrets["REDIS_URL"], "${env}/cache/redis_url")
	}
}

func TestLoadWorkspaceConfig_NotFound(t *testing.T) {
	_, err := LoadWorkspaceConfig("nonexistent/vx.toml")
	if err == nil {
		t.Fatal("LoadWorkspaceConfig() expected error for missing file")
	}
}

func TestFindRootConfig(t *testing.T) {
	startDir := filepath.Join("testdata", "findroot", "project", "subdir", "deep")

	found, err := FindRootConfig(startDir)
	if err != nil {
		t.Fatalf("FindRootConfig() error = %v", err)
	}

	expected, err := filepath.Abs(filepath.Join("testdata", "findroot", "project", "vx.toml"))
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	if found != expected {
		t.Errorf("FindRootConfig() = %q, want %q", found, expected)
	}
}

func TestFindRootConfig_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := FindRootConfig(dir)
	if err == nil {
		t.Fatal("FindRootConfig() expected error when no vx.toml exists")
	}
}

func TestFindRootConfig_InCurrentDir(t *testing.T) {
	startDir := filepath.Join("testdata", "findroot", "project")

	found, err := FindRootConfig(startDir)
	if err != nil {
		t.Fatalf("FindRootConfig() error = %v", err)
	}

	expected, err := filepath.Abs(filepath.Join("testdata", "findroot", "project", "vx.toml"))
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	if found != expected {
		t.Errorf("FindRootConfig() = %q, want %q", found, expected)
	}
}
