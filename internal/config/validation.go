package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validate checks that a RootConfig has all required fields and valid values.
func Validate(cfg *RootConfig) error {
	if err := validateVault(cfg.Vault); err != nil {
		return fmt.Errorf("vault config: %w", err)
	}

	if err := validateEnvironments(cfg.Environments); err != nil {
		return fmt.Errorf("environments config: %w", err)
	}

	return nil
}

// ValidateWithRoot validates a RootConfig and also checks that workspace paths
// exist relative to rootDir on the filesystem.
func ValidateWithRoot(cfg *RootConfig, rootDir string) error {
	if err := Validate(cfg); err != nil {
		return err
	}

	return validateWorkspacePaths(cfg.Workspaces, rootDir)
}

// ValidateWorkspace checks that a WorkspaceConfig has valid structure.
func ValidateWorkspace(cfg *WorkspaceConfig) error {
	if cfg == nil {
		return fmt.Errorf("workspace config is nil")
	}
	return nil
}

func validateVault(v VaultConfig) error {
	if v.Address == "" {
		return fmt.Errorf("address is required")
	}
	if v.AuthMethod == "" {
		return fmt.Errorf("auth_method is required")
	}
	return nil
}

func validateEnvironments(e EnvironmentConfig) error {
	if e.Default == "" {
		return fmt.Errorf("default environment is required")
	}

	if len(e.Available) == 0 {
		return fmt.Errorf("at least one available environment is required")
	}

	if !contains(e.Available, e.Default) {
		return fmt.Errorf(
			"default environment %q is not in available environments [%s]",
			e.Default,
			strings.Join(e.Available, ", "),
		)
	}

	return nil
}

func validateWorkspacePaths(workspaces []string, rootDir string) error {
	for _, ws := range workspaces {
		absPath := filepath.Join(rootDir, ws)
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("workspace path %q does not exist: %w", ws, err)
		}
	}
	return nil
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
