package resolver

import "strings"

// SecretMapping maps an environment variable name to a key within a Vault
// KV v2 path. For example, env var DATABASE_URL may map to key "url" under
// the Vault path "dev/database".
type SecretMapping struct {
	EnvVar string
	Key    string
}

// GroupByPath groups secrets by their Vault KV v2 path prefix after
// interpolating the environment. The path is split at the last "/" separator:
// the prefix becomes the Vault read path, the suffix becomes the key name
// within that path's data.
//
// The input map is not mutated.
func GroupByPath(secrets map[string]string, env string) map[string][]SecretMapping {
	groups := make(map[string][]SecretMapping, len(secrets))

	for envVar, rawPath := range secrets {
		resolved := Interpolate(rawPath, env)

		vaultPath, key := splitPath(resolved)
		if vaultPath == "" || key == "" {
			continue
		}

		groups[vaultPath] = append(groups[vaultPath], SecretMapping{
			EnvVar: envVar,
			Key:    key,
		})
	}

	return groups
}

// splitPath splits a resolved path at the last "/" into a Vault path prefix
// and a key suffix. Returns empty strings if there is no "/" separator.
func splitPath(path string) (string, string) {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "", ""
	}

	return path[:idx], path[idx+1:]
}
