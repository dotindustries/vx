package resolver

import "strings"

// Interpolate replaces all occurrences of ${env} in the given path with the
// actual environment name. If env is empty the placeholder is removed.
func Interpolate(path string, env string) string {
	return strings.ReplaceAll(path, "${env}", env)
}

// HasEnvVar reports whether path contains at least one ${env} placeholder.
func HasEnvVar(path string) bool {
	return strings.Contains(path, "${env}")
}
