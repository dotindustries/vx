package config

import (
	"testing"
)

func TestMerge_RootOnly(t *testing.T) {
	root := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
			AuthRole:   "admin",
			BasePath:   "secret",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev", "staging", "production"},
		},
		Secrets: map[string]string{
			"DATABASE_URL":   "${env}/database/url",
			"OPENAI_API_KEY": "shared/openai/api_key",
		},
		Defaults: map[string]any{
			"NODE_ENV": "development",
			"APP_URL":  "http://localhost:3000",
			"staging": map[string]any{
				"NODE_ENV": "production",
			},
			"production": map[string]any{
				"NODE_ENV": "production",
			},
		},
	}

	merged, err := Merge(root, nil, "dev")
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if merged.Environment != "dev" {
		t.Errorf("Environment = %q, want %q", merged.Environment, "dev")
	}
	if merged.Vault.Address != "https://vault.example.com" {
		t.Errorf("Vault.Address = %q, want %q", merged.Vault.Address, "https://vault.example.com")
	}

	assertMapValue(t, merged.Defaults, "NODE_ENV", "development")
	assertMapValue(t, merged.Defaults, "APP_URL", "http://localhost:3000")
	assertMapValue(t, merged.Secrets, "DATABASE_URL", "${env}/database/url")
	assertMapValue(t, merged.Secrets, "OPENAI_API_KEY", "shared/openai/api_key")
}

func TestMerge_EnvSpecificDefaults(t *testing.T) {
	root := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev", "staging", "production"},
		},
		Secrets: map[string]string{},
		Defaults: map[string]any{
			"NODE_ENV": "development",
			"APP_URL":  "http://localhost:3000",
			"staging": map[string]any{
				"NODE_ENV": "production",
			},
		},
	}

	merged, err := Merge(root, nil, "staging")
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	assertMapValue(t, merged.Defaults, "NODE_ENV", "production")
	assertMapValue(t, merged.Defaults, "APP_URL", "http://localhost:3000")
}

func TestMerge_WithWorkspace(t *testing.T) {
	root := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev", "production"},
		},
		Secrets: map[string]string{
			"DATABASE_URL": "${env}/database/url",
		},
		Defaults: map[string]any{
			"NODE_ENV": "development",
		},
	}

	workspace := &WorkspaceConfig{
		Secrets: map[string]string{
			"TURSO_TOKEN": "${env}/database/turso",
		},
		Defaults: map[string]any{
			"SOME_KEY":  "value",
			"NODE_ENV":  "ws-development",
			"LOG_LEVEL": "debug",
			"production": map[string]any{
				"LOG_LEVEL": "error",
			},
		},
	}

	merged, err := Merge(root, workspace, "dev")
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Root secrets preserved
	assertMapValue(t, merged.Secrets, "DATABASE_URL", "${env}/database/url")
	// Workspace secrets added
	assertMapValue(t, merged.Secrets, "TURSO_TOKEN", "${env}/database/turso")

	// Workspace defaults override root defaults
	assertMapValue(t, merged.Defaults, "NODE_ENV", "ws-development")
	// Workspace-only defaults added
	assertMapValue(t, merged.Defaults, "SOME_KEY", "value")
	assertMapValue(t, merged.Defaults, "LOG_LEVEL", "debug")
}

func TestMerge_WorkspaceEnvSpecificDefaults(t *testing.T) {
	root := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev", "production"},
		},
		Secrets:  map[string]string{},
		Defaults: map[string]any{},
	}

	workspace := &WorkspaceConfig{
		Secrets: map[string]string{},
		Defaults: map[string]any{
			"LOG_LEVEL": "debug",
			"production": map[string]any{
				"LOG_LEVEL": "error",
			},
		},
	}

	merged, err := Merge(root, workspace, "production")
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	assertMapValue(t, merged.Defaults, "LOG_LEVEL", "error")
}

func TestMerge_DefaultEnvUsed(t *testing.T) {
	root := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev", "staging"},
		},
		Secrets:  map[string]string{},
		Defaults: map[string]any{},
	}

	merged, err := Merge(root, nil, "")
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	if merged.Environment != "dev" {
		t.Errorf("Environment = %q, want %q", merged.Environment, "dev")
	}
}

func TestMerge_InvalidEnv(t *testing.T) {
	root := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev"},
		},
	}

	_, err := Merge(root, nil, "nonexistent")
	if err == nil {
		t.Fatal("Merge() expected error for invalid environment")
	}
}

func TestMerge_NilRoot(t *testing.T) {
	_, err := Merge(nil, nil, "dev")
	if err == nil {
		t.Fatal("Merge() expected error for nil root config")
	}
}

func TestMerge_DoesNotMutateInputs(t *testing.T) {
	rootSecrets := map[string]string{
		"DATABASE_URL": "${env}/database/url",
	}
	rootDefaults := map[string]any{
		"NODE_ENV": "development",
	}

	root := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev"},
		},
		Secrets:  rootSecrets,
		Defaults: rootDefaults,
	}

	workspace := &WorkspaceConfig{
		Secrets: map[string]string{
			"WS_SECRET": "path",
		},
		Defaults: map[string]any{
			"WS_KEY": "value",
		},
	}

	_, err := Merge(root, workspace, "dev")
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Verify root secrets were not mutated
	if _, ok := rootSecrets["WS_SECRET"]; ok {
		t.Error("root secrets map was mutated: WS_SECRET was added")
	}
	if len(rootSecrets) != 1 {
		t.Errorf("root secrets map length changed: got %d, want 1", len(rootSecrets))
	}
}

func assertMapValue(t *testing.T, m map[string]string, key string, want string) {
	t.Helper()
	got, ok := m[key]
	if !ok {
		t.Errorf("map missing key %q", key)
		return
	}
	if got != want {
		t.Errorf("map[%q] = %q, want %q", key, got, want)
	}
}
