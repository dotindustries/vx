package config

// RootConfig represents the top-level vx.toml configuration file.
type RootConfig struct {
	Vault        VaultConfig       `toml:"vault"`
	Environments EnvironmentConfig `toml:"environments"`
	Workspaces   []string          `toml:"workspaces"`
	Secrets      map[string]string `toml:"secrets"`
	Defaults     map[string]any    `toml:"defaults"`
	EnvFile      EnvFileConfig     `toml:"env_file"`
}

// EnvFileConfig controls loading of local dotenv override files.
type EnvFileConfig struct {
	AutoLoad *bool `toml:"auto_load"`
}

// AutoLoadEnabled reports whether .env.local next to the root vx.toml should
// be loaded automatically. It is enabled unless explicitly turned off.
func (c EnvFileConfig) AutoLoadEnabled() bool {
	return c.AutoLoad == nil || *c.AutoLoad
}

// VaultConfig holds Vault server connection settings.
type VaultConfig struct {
	Address    string `toml:"address"`
	AuthMethod string `toml:"auth_method"`
	AuthRole   string `toml:"auth_role"`
	BasePath   string `toml:"base_path"`
}

// EnvironmentConfig defines available environments and the default selection.
type EnvironmentConfig struct {
	Default   string   `toml:"default"`
	Available []string `toml:"available"`
}

// WorkspaceConfig represents a workspace-level vx.toml with only secrets and defaults.
type WorkspaceConfig struct {
	Secrets  map[string]string `toml:"secrets"`
	Defaults map[string]any    `toml:"defaults"`
}

// MergedConfig is the fully resolved configuration after merging root and workspace
// configs for a specific environment.
type MergedConfig struct {
	Vault       VaultConfig
	Environment string
	Secrets     map[string]string
	Defaults    map[string]string
}
