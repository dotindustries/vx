package migrate

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// FnoxConfig represents the top-level fnox.toml configuration file.
type FnoxConfig struct {
	DefaultProvider string                    `toml:"default_provider"`
	Import          []string                  `toml:"import"`
	Providers       map[string]FnoxProvider   `toml:"providers"`
	Secrets         map[string]FnoxSecret     `toml:"secrets"`
	Profiles        map[string]FnoxProfile    `toml:"profiles"`
}

// FnoxProvider describes a secret provider (e.g. HashiCorp Vault).
type FnoxProvider struct {
	Type    string `toml:"type"`
	Address string `toml:"address"`
	Path    string `toml:"path"`
}

// FnoxSecret represents a single secret entry in fnox.toml.
// A secret either references a provider+value or specifies a default.
type FnoxSecret struct {
	Provider string `toml:"provider"`
	Value    string `toml:"value"`
	Default  string `toml:"default"`
}

// FnoxProfile holds environment-specific secret overrides.
type FnoxProfile struct {
	Secrets map[string]FnoxSecret `toml:"secrets"`
}

// LoadFnoxConfig parses a fnox.toml file at the given path.
func LoadFnoxConfig(path string) (*FnoxConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading fnox config %s: %w", path, err)
	}

	var cfg FnoxConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing fnox config %s: %w", path, err)
	}

	return &cfg, nil
}
