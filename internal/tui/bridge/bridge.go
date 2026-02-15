// Package bridge adapts existing vx business logic for use by the TUI.
// It wraps config, vault, and resolver functions with explicit parameters
// instead of relying on package-level Cobra flag variables.
package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.dot.industries/vx/internal/config"
	"go.dot.industries/vx/internal/resolver"
	"go.dot.industries/vx/internal/token"
	"go.dot.industries/vx/internal/vault"
)

// FileTarget represents a vx.toml file that can be written to.
type FileTarget struct {
	Label string // display name, e.g. "web" or "[root]"
	Path  string // absolute path to the vx.toml file
}

// Bridge provides TUI access to vx business logic without depending on
// package-level flag variables.
type Bridge struct {
	configPath string
	vaultAddr  string
	authMethod string
	roleID     string
	secretID   string
}

// New creates a Bridge with the given configuration overrides.
// Empty values fall back to config defaults.
func New(configPath, vaultAddr, authMethod, roleID, secretID string) *Bridge {
	return &Bridge{
		configPath: configPath,
		vaultAddr:  vaultAddr,
		authMethod: authMethod,
		roleID:     roleID,
		secretID:   secretID,
	}
}

// LoadConfig finds and parses the root vx.toml. Returns the config and its
// parent directory.
func (b *Bridge) LoadConfig() (*config.RootConfig, string, error) {
	configPath := b.configPath

	if configPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("getting working directory: %w", err)
		}

		found, err := config.FindRootConfig(cwd)
		if err != nil {
			return nil, "", err
		}
		configPath = found
	}

	cfg, err := config.LoadRootConfig(configPath)
	if err != nil {
		return nil, "", err
	}

	rootDir := filepath.Dir(configPath)
	return cfg, rootDir, nil
}

// WorkspaceNames returns human-readable workspace names extracted from the
// configured workspace paths (e.g. "web/vx.toml" -> "web").
func (b *Bridge) WorkspaceNames(cfg *config.RootConfig) []string {
	names := make([]string, 0, len(cfg.Workspaces))
	for _, wp := range cfg.Workspaces {
		dir := filepath.Dir(wp)
		name := filepath.Base(dir)
		names = append(names, name)
	}
	return names
}

// MergeForWorkspace loads and merges the config for a specific workspace and
// environment. Returns the merged secrets (env var -> vault path template).
func (b *Bridge) MergeForWorkspace(
	cfg *config.RootConfig,
	rootDir string,
	workspace string,
	env string,
) (*config.MergedConfig, error) {
	wsPath, err := config.ResolveWorkspacePath(rootDir, workspace, cfg.Workspaces)
	if err != nil {
		return nil, fmt.Errorf("resolving workspace path: %w", err)
	}

	wsCfg, err := config.LoadWorkspaceConfig(wsPath)
	if err != nil {
		return nil, fmt.Errorf("loading workspace config: %w", err)
	}

	return config.Merge(cfg, wsCfg, env)
}

// MergeRootOnly merges just the root config for the given environment (no
// workspace overlay). Used for the "[root]" view.
func (b *Bridge) MergeRootOnly(
	cfg *config.RootConfig,
	env string,
) (*config.MergedConfig, error) {
	return config.Merge(cfg, nil, env)
}

// Authenticate creates an authenticated Vault client. It first tries the
// cached token, then falls back to a fresh auth flow.
func (b *Bridge) Authenticate(cfg *config.RootConfig) (*vault.Client, error) {
	addr := b.vaultAddress(cfg)

	tok, err := token.ReadToken()
	if err == nil {
		client, err := vault.NewClientWithToken(addr, cfg.Vault.BasePath, tok)
		if err != nil {
			return nil, fmt.Errorf("creating vault client: %w", err)
		}
		if client.IsAuthenticated() {
			return client, nil
		}
	}

	return nil, fmt.Errorf("no valid Vault token; run `vx login` first")
}

// ResolveSingle fetches a single secret value from Vault. The vaultPath should
// already be interpolated (no ${env} placeholders).
func (b *Bridge) ResolveSingle(
	client *vault.Client,
	envVar string,
	vaultPath string,
	env string,
) (string, error) {
	interpolated := resolver.Interpolate(vaultPath, env)

	r := resolver.New(client, "")
	secrets := map[string]string{envVar: interpolated}

	result, err := r.Resolve(secrets, "")
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", envVar, err)
	}

	val, ok := result[envVar]
	if !ok {
		return "", fmt.Errorf("secret %s not found at path %s", envVar, interpolated)
	}

	return val, nil
}

// ListVaultKeys lists keys and directories at a Vault KV v2 metadata path.
func (b *Bridge) ListVaultKeys(client *vault.Client, kvPath string) ([]VaultEntry, error) {
	entries, err := client.ListKeys(kvPath)
	if err != nil {
		return nil, err
	}

	result := make([]VaultEntry, len(entries))
	for i, e := range entries {
		result[i] = VaultEntry{
			Name:  e.Name,
			IsDir: e.IsDir,
		}
	}
	return result, nil
}

// VaultEntry represents a key or directory in the Vault KV tree.
type VaultEntry struct {
	Name  string
	IsDir bool
}

// WorkspaceFiles returns all vx.toml files that can be written to, including
// the root config and each workspace config.
func (b *Bridge) WorkspaceFiles(cfg *config.RootConfig, rootDir string) []FileTarget {
	targets := make([]FileTarget, 0, len(cfg.Workspaces)+1)

	targets = append(targets, FileTarget{
		Label: "[root]",
		Path:  filepath.Join(rootDir, "vx.toml"),
	})

	for _, wp := range cfg.Workspaces {
		dir := filepath.Dir(wp)
		name := filepath.Base(dir)
		targets = append(targets, FileTarget{
			Label: name,
			Path:  filepath.Join(rootDir, wp),
		})
	}

	return targets
}

// vaultAddress returns the Vault address, preferring the bridge override.
func (b *Bridge) vaultAddress(cfg *config.RootConfig) string {
	if b.vaultAddr != "" {
		return b.vaultAddr
	}
	return cfg.Vault.Address
}

// WorkspaceForPath returns the workspace name that owns the given vx.toml
// path, or "[root]" if it's the root config.
func (b *Bridge) WorkspaceForPath(cfg *config.RootConfig, rootDir, filePath string) string {
	rootConfigPath := filepath.Join(rootDir, "vx.toml")
	if filePath == rootConfigPath {
		return "[root]"
	}

	for _, wp := range cfg.Workspaces {
		absPath := filepath.Join(rootDir, wp)
		if absPath == filePath {
			dir := filepath.Dir(wp)
			return filepath.Base(dir)
		}
	}

	return filepath.Base(filepath.Dir(filePath))
}

// SecretSource returns the file path where a given secret is defined.
// It checks workspace configs first, then falls back to root.
func (b *Bridge) SecretSource(
	cfg *config.RootConfig,
	rootDir string,
	workspace string,
	envVar string,
) string {
	if workspace != "" && workspace != "[root]" {
		for _, wp := range cfg.Workspaces {
			dir := filepath.Dir(wp)
			name := filepath.Base(dir)
			if name == workspace {
				wsPath := filepath.Join(rootDir, wp)
				wsCfg, err := config.LoadWorkspaceConfig(wsPath)
				if err == nil {
					if _, ok := wsCfg.Secrets[envVar]; ok {
						return wsPath
					}
				}
				break
			}
		}
	}

	if _, ok := cfg.Secrets[envVar]; ok {
		return filepath.Join(rootDir, "vx.toml")
	}

	return ""
}

// InterpolateSecrets takes raw secret mappings and returns them with ${env}
// expanded for display purposes.
func InterpolateSecrets(secrets map[string]string, env string) map[string]string {
	result := make(map[string]string, len(secrets))
	for k, v := range secrets {
		result[k] = resolver.Interpolate(v, env)
	}
	return result
}

// TruncateMiddle truncates a string in the middle if it exceeds maxLen,
// inserting "..." in the center.
func TruncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen || maxLen < 4 {
		return s
	}
	half := (maxLen - 3) / 2
	return s[:half] + "..." + s[len(s)-half:]
}

// ParseWorkspacePath extracts a workspace directory name from a path like
// "packages/api/vx.toml" to "api" or "web/vx.toml" to "web".
func ParseWorkspacePath(wp string) string {
	dir := filepath.Dir(wp)
	return filepath.Base(dir)
}

// contains checks if a string slice contains a specific value.
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, val) {
			return true
		}
	}
	return false
}
