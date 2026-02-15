package config

import (
	"path/filepath"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &RootConfig{
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
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_MissingVaultAddress(t *testing.T) {
	cfg := &RootConfig{
		Vault: VaultConfig{
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev"},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() expected error for missing vault address")
	}
}

func TestValidate_MissingAuthMethod(t *testing.T) {
	cfg := &RootConfig{
		Vault: VaultConfig{
			Address: "https://vault.example.com",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev"},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() expected error for missing auth method")
	}
}

func TestValidate_MissingDefaultEnv(t *testing.T) {
	cfg := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Available: []string{"dev"},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() expected error for missing default environment")
	}
}

func TestValidate_EmptyAvailableEnvs(t *testing.T) {
	cfg := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() expected error for empty available environments")
	}
}

func TestValidate_DefaultNotInAvailable(t *testing.T) {
	cfg := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"staging", "production"},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal("Validate() expected error when default is not in available environments")
	}
}

func TestValidateWithRoot_WorkspacePathsExist(t *testing.T) {
	rootDir := filepath.Join("testdata", "root")
	cfg := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev"},
		},
		Workspaces: []string{"web/vx.toml", "packages/api/vx.toml"},
	}

	if err := ValidateWithRoot(cfg, rootDir); err != nil {
		t.Errorf("ValidateWithRoot() error = %v, want nil", err)
	}
}

func TestValidateWithRoot_WorkspacePathMissing(t *testing.T) {
	rootDir := filepath.Join("testdata", "root")
	cfg := &RootConfig{
		Vault: VaultConfig{
			Address:    "https://vault.example.com",
			AuthMethod: "oidc",
		},
		Environments: EnvironmentConfig{
			Default:   "dev",
			Available: []string{"dev"},
		},
		Workspaces: []string{"nonexistent/vx.toml"},
	}

	if err := ValidateWithRoot(cfg, rootDir); err == nil {
		t.Fatal("ValidateWithRoot() expected error for missing workspace path")
	}
}

func TestValidateWorkspace_Valid(t *testing.T) {
	cfg := &WorkspaceConfig{
		Secrets: map[string]string{"KEY": "path"},
	}

	if err := ValidateWorkspace(cfg); err != nil {
		t.Errorf("ValidateWorkspace() error = %v, want nil", err)
	}
}

func TestValidateWorkspace_Nil(t *testing.T) {
	if err := ValidateWorkspace(nil); err == nil {
		t.Fatal("ValidateWorkspace() expected error for nil config")
	}
}
