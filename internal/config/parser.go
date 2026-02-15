package config

import (
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

// LoadRootConfig parses a root vx.toml file at the given path.
func LoadRootConfig(path string) (*RootConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading root config %s: %w", path, err)
	}

	var cfg RootConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing root config %s: %w", path, err)
	}

	return &cfg, nil
}

// LoadWorkspaceConfig parses a workspace-level vx.toml file at the given path.
func LoadWorkspaceConfig(path string) (*WorkspaceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading workspace config %s: %w", path, err)
	}

	var cfg WorkspaceConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing workspace config %s: %w", path, err)
	}

	return &cfg, nil
}

// FindRootConfig walks up directories starting from startDir to locate a vx.toml file.
// Returns the absolute path to the first vx.toml found, or an error if none exists.
func FindRootConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path for %s: %w", startDir, err)
	}

	for {
		candidate := filepath.Join(dir, "vx.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("vx.toml not found in %s or any parent directory", startDir)
}
