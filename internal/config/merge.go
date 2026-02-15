package config

import (
	"fmt"
)

// Merge combines a root config and an optional workspace config for a specific environment
// into a single MergedConfig. Input configs are never mutated.
func Merge(root *RootConfig, workspace *WorkspaceConfig, env string) (*MergedConfig, error) {
	if root == nil {
		return nil, fmt.Errorf("root config is required")
	}

	if env == "" {
		env = root.Environments.Default
	}

	if !contains(root.Environments.Available, env) {
		return nil, fmt.Errorf("environment %q is not in available environments", env)
	}

	defaults := resolveDefaults(root.Defaults, env)
	defaults = mergeWorkspaceDefaults(defaults, workspace, env)

	secrets := mergeSecrets(root.Secrets, workspace)

	return &MergedConfig{
		Vault:       root.Vault,
		Environment: env,
		Secrets:     secrets,
		Defaults:    defaults,
	}, nil
}

// resolveDefaults extracts base defaults and overlays environment-specific defaults.
// The input map is never mutated.
func resolveDefaults(defaults map[string]any, env string) map[string]string {
	result := make(map[string]string)

	for key, val := range defaults {
		if str, ok := val.(string); ok {
			result[key] = str
		}
	}

	envDefaults := extractEnvDefaults(defaults, env)
	for key, val := range envDefaults {
		result[key] = val
	}

	return result
}

// extractEnvDefaults pulls the environment-specific nested table from a defaults map.
func extractEnvDefaults(defaults map[string]any, env string) map[string]string {
	result := make(map[string]string)

	envSection, ok := defaults[env]
	if !ok {
		return result
	}

	envMap, ok := envSection.(map[string]any)
	if !ok {
		return result
	}

	for key, val := range envMap {
		if str, ok := val.(string); ok {
			result[key] = str
		}
	}

	return result
}

// mergeWorkspaceDefaults overlays workspace defaults on top of existing defaults.
// Neither input is mutated; a new map is returned.
func mergeWorkspaceDefaults(base map[string]string, workspace *WorkspaceConfig, env string) map[string]string {
	if workspace == nil {
		return copyStringMap(base)
	}

	result := copyStringMap(base)

	wsDefaults := resolveDefaults(workspace.Defaults, env)
	for key, val := range wsDefaults {
		result[key] = val
	}

	return result
}

// mergeSecrets combines root and workspace secrets into a new map.
// Workspace secrets override root secrets with the same key.
func mergeSecrets(rootSecrets map[string]string, workspace *WorkspaceConfig) map[string]string {
	result := copyStringMap(rootSecrets)

	if workspace == nil {
		return result
	}

	for key, val := range workspace.Secrets {
		result[key] = val
	}

	return result
}

// copyStringMap creates a shallow copy of a string map.
func copyStringMap(src map[string]string) map[string]string {
	result := make(map[string]string, len(src))
	for k, v := range src {
		result[k] = v
	}
	return result
}
