package migrate

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// ConvertResult holds the output of converting a fnox config to vx format.
type ConvertResult struct {
	RootConfig       string
	WorkspaceConfigs map[string]string
}

// vxRoot represents the root vx.toml structure for TOML serialization.
type vxRoot struct {
	Vault        vxVault        `toml:"vault"`
	Environments vxEnvironments `toml:"environments"`
	Workspaces   []string       `toml:"workspaces,omitempty"`
	Secrets      map[string]string `toml:"secrets,omitempty"`
	Defaults     map[string]any    `toml:"defaults,omitempty"`
}

type vxVault struct {
	Address  string `toml:"address"`
	BasePath string `toml:"base_path"`
}

type vxEnvironments struct {
	Default   string   `toml:"default"`
	Available []string `toml:"available"`
}

// Convert transforms a fnox config into vx config format.
// The input config is never mutated.
func Convert(fnox *FnoxConfig, rootDir string) (*ConvertResult, error) {
	if fnox == nil {
		return nil, fmt.Errorf("fnox config is required")
	}

	address := extractVaultAddress(fnox.Providers)
	basePath := extractBasePath(fnox.Providers)
	envs := extractEnvironments(fnox.Providers)
	defaultEnv := resolveDefaultEnv(fnox.DefaultProvider)

	secrets := convertSecrets(fnox.Secrets, fnox.Providers)
	defaults := convertDefaults(fnox.Secrets)
	profileDefaults := convertProfileDefaults(fnox.Profiles, fnox.Secrets)

	mergedDefaults := mergeProfilesIntoDefaults(defaults, profileDefaults)
	workspaces := convertImports(fnox.Import)

	root := vxRoot{
		Vault: vxVault{
			Address:  address,
			BasePath: basePath,
		},
		Environments: vxEnvironments{
			Default:   defaultEnv,
			Available: envs,
		},
		Workspaces: workspaces,
		Secrets:    secrets,
		Defaults:   mergedDefaults,
	}

	rootTOML, err := FormatVxToml(root)
	if err != nil {
		return nil, fmt.Errorf("formatting root config: %w", err)
	}

	return &ConvertResult{
		RootConfig:       rootTOML,
		WorkspaceConfigs: make(map[string]string),
	}, nil
}

// FormatVxToml serializes a config struct as a clean TOML string.
func FormatVxToml(cfg any) (string, error) {
	var buf bytes.Buffer

	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return "", fmt.Errorf("encoding TOML: %w", err)
	}

	return buf.String(), nil
}

// extractVaultAddress returns the vault address from the first provider found.
func extractVaultAddress(providers map[string]FnoxProvider) string {
	for _, p := range providers {
		if p.Address != "" {
			return p.Address
		}
	}

	return ""
}

// extractBasePath determines the common base path prefix across all provider
// paths. For example, "secret/dev" and "secret/staging" yield "secret".
func extractBasePath(providers map[string]FnoxProvider) string {
	paths := collectProviderPaths(providers)
	if len(paths) == 0 {
		return ""
	}

	return commonPrefix(paths)
}

// collectProviderPaths extracts the top-level path segment from each provider.
func collectProviderPaths(providers map[string]FnoxProvider) []string {
	paths := make([]string, 0, len(providers))

	for _, p := range providers {
		if p.Path != "" {
			paths = append(paths, p.Path)
		}
	}

	return paths
}

// commonPrefix finds the longest common path prefix across all paths,
// split by "/". Returns the first segment that is common to all paths.
func commonPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	segments := strings.Split(paths[0], "/")
	prefix := segments[0]

	for _, p := range paths[1:] {
		segs := strings.Split(p, "/")
		if len(segs) == 0 || segs[0] != prefix {
			return ""
		}
	}

	return prefix
}

// extractEnvironments derives environment names from provider names.
// Provider "vault-dev" yields "dev", "vault-staging" yields "staging", etc.
// The "shared" environment is excluded as it is not an environment.
func extractEnvironments(providers map[string]FnoxProvider) []string {
	seen := make(map[string]bool)
	envs := make([]string, 0)

	for name := range providers {
		env := providerToEnv(name)
		if env == "" || env == "shared" || isSubEnvironment(env) {
			continue
		}
		if !seen[env] {
			seen[env] = true
			envs = append(envs, env)
		}
	}

	sort.Strings(envs)

	return envs
}

// providerToEnv extracts the environment name from a provider name.
// "vault-dev" -> "dev", "vault-staging" -> "staging".
func providerToEnv(providerName string) string {
	prefix := "vault-"
	if !strings.HasPrefix(providerName, prefix) {
		return providerName
	}

	return strings.TrimPrefix(providerName, prefix)
}

// isSubEnvironment checks whether an env name contains a sub-path
// like "dev-integrations", which is not a standalone environment.
func isSubEnvironment(env string) bool {
	return strings.Contains(env, "-")
}

// resolveDefaultEnv extracts the environment from the default provider name.
func resolveDefaultEnv(defaultProvider string) string {
	env := providerToEnv(defaultProvider)
	if env == "" {
		return "dev"
	}

	return env
}

// convertSecrets maps fnox secrets with providers to vx secret paths.
// Secrets with only a default value are excluded (handled by convertDefaults).
func convertSecrets(secrets map[string]FnoxSecret, providers map[string]FnoxProvider) map[string]string {
	result := make(map[string]string)

	for name, secret := range secrets {
		if secret.Provider == "" {
			continue
		}

		path := buildSecretPath(secret, providers)
		if path != "" {
			result[name] = path
		}
	}

	return result
}

// buildSecretPath constructs a vx secret path from a fnox secret and its provider.
func buildSecretPath(secret FnoxSecret, providers map[string]FnoxProvider) string {
	provider, ok := providers[secret.Provider]
	if !ok {
		return ""
	}

	envName := providerToEnv(secret.Provider)
	relativePath := extractRelativePath(provider.Path)

	if envName == "shared" {
		return joinPath("shared", secret.Value)
	}

	if strings.Contains(relativePath, "/") {
		subPath := pathAfterEnv(relativePath)
		return joinPath("${env}", subPath, secret.Value)
	}

	return joinPath("${env}", secret.Value)
}

// extractRelativePath removes the base path prefix (e.g. "secret/") from
// a provider path, returning the environment-relative portion.
func extractRelativePath(providerPath string) string {
	idx := strings.Index(providerPath, "/")
	if idx < 0 {
		return providerPath
	}

	return providerPath[idx+1:]
}

// pathAfterEnv returns the sub-path after the environment segment.
// For "dev/integrations" it returns "integrations".
func pathAfterEnv(relativePath string) string {
	idx := strings.Index(relativePath, "/")
	if idx < 0 {
		return ""
	}

	return relativePath[idx+1:]
}

// joinPath joins non-empty path segments with "/".
func joinPath(segments ...string) string {
	nonEmpty := make([]string, 0, len(segments))

	for _, s := range segments {
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}

	return strings.Join(nonEmpty, "/")
}

// convertDefaults extracts secrets that only have a default value (no provider).
func convertDefaults(secrets map[string]FnoxSecret) map[string]any {
	result := make(map[string]any)

	for name, secret := range secrets {
		if secret.Provider == "" && secret.Default != "" {
			result[name] = secret.Default
		}
	}

	return result
}

// convertProfileDefaults extracts per-profile default overrides from fnox profiles.
func convertProfileDefaults(profiles map[string]FnoxProfile, baseSecrets map[string]FnoxSecret) map[string]map[string]string {
	result := make(map[string]map[string]string)

	for profileName, profile := range profiles {
		env := profileName
		overrides := extractDefaultOverrides(profile.Secrets, baseSecrets)

		if len(overrides) > 0 {
			result[env] = overrides
		}
	}

	return result
}

// extractDefaultOverrides finds secrets in a profile that change the default
// value compared to the base secrets.
func extractDefaultOverrides(profileSecrets map[string]FnoxSecret, baseSecrets map[string]FnoxSecret) map[string]string {
	result := make(map[string]string)

	for name, secret := range profileSecrets {
		if secret.Default == "" {
			continue
		}

		base, exists := baseSecrets[name]
		if !exists || base.Default != secret.Default {
			result[name] = secret.Default
		}
	}

	return result
}

// mergeProfilesIntoDefaults combines base defaults with per-environment
// overrides into the vx defaults structure.
func mergeProfilesIntoDefaults(base map[string]any, profiles map[string]map[string]string) map[string]any {
	result := make(map[string]any, len(base)+len(profiles))

	for k, v := range base {
		result[k] = v
	}

	for env, overrides := range profiles {
		envMap := make(map[string]any, len(overrides))
		for k, v := range overrides {
			envMap[k] = v
		}
		result[env] = envMap
	}

	return result
}

// convertImports maps fnox import paths to vx workspace paths.
// Each import "./web/fnox.toml" becomes "web/vx.toml".
func convertImports(imports []string) []string {
	workspaces := make([]string, 0, len(imports))

	for _, imp := range imports {
		ws := importToWorkspace(imp)
		if ws != "" {
			workspaces = append(workspaces, ws)
		}
	}

	return workspaces
}

// importToWorkspace converts a fnox import path to a vx workspace path.
// "./web/fnox.toml" -> "web/vx.toml"
func importToWorkspace(importPath string) string {
	cleaned := strings.TrimPrefix(importPath, "./")
	dir := filepath.Dir(cleaned)

	if dir == "" || dir == "." {
		return ""
	}

	return dir + "/vx.toml"
}
