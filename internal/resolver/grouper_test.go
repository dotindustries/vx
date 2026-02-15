package resolver

import (
	"testing"
)

func TestGroupByPath(t *testing.T) {
	tests := []struct {
		name    string
		secrets map[string]string
		env     string
		want    map[string][]SecretMapping
	}{
		{
			name: "groups by parent path",
			secrets: map[string]string{
				"DATABASE_URL":        "${env}/database/url",
				"DATABASE_AUTH_TOKEN": "${env}/database/auth_token",
				"STRIPE_SECRET_KEY":   "${env}/stripe/secret_key",
			},
			env: "dev",
			want: map[string][]SecretMapping{
				"dev/database": {
					{EnvVar: "DATABASE_URL", Key: "url"},
					{EnvVar: "DATABASE_AUTH_TOKEN", Key: "auth_token"},
				},
				"dev/stripe": {
					{EnvVar: "STRIPE_SECRET_KEY", Key: "secret_key"},
				},
			},
		},
		{
			name: "nested paths",
			secrets: map[string]string{
				"NOVU_API_KEY": "${env}/integrations/novu/api_key",
			},
			env: "staging",
			want: map[string][]SecretMapping{
				"staging/integrations/novu": {
					{EnvVar: "NOVU_API_KEY", Key: "api_key"},
				},
			},
		},
		{
			name: "shared paths without env",
			secrets: map[string]string{
				"OPENAI_API_KEY": "shared/openai/api_key",
			},
			env: "dev",
			want: map[string][]SecretMapping{
				"shared/openai": {
					{EnvVar: "OPENAI_API_KEY", Key: "api_key"},
				},
			},
		},
		{
			name:    "empty secrets",
			secrets: map[string]string{},
			env:     "dev",
			want:    map[string][]SecretMapping{},
		},
		{
			name: "path without separator is skipped",
			secrets: map[string]string{
				"BAD_SECRET": "noslash",
			},
			env:  "dev",
			want: map[string][]SecretMapping{},
		},
		{
			name: "mixed env and shared paths",
			secrets: map[string]string{
				"DB_URL":         "${env}/database/url",
				"OPENAI_API_KEY": "shared/openai/api_key",
			},
			env: "production",
			want: map[string][]SecretMapping{
				"production/database": {
					{EnvVar: "DB_URL", Key: "url"},
				},
				"shared/openai": {
					{EnvVar: "OPENAI_API_KEY", Key: "api_key"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupByPath(tt.secrets, tt.env)

			if len(got) != len(tt.want) {
				t.Fatalf("GroupByPath() returned %d groups, want %d", len(got), len(tt.want))
			}

			for path, wantMappings := range tt.want {
				gotMappings, ok := got[path]
				if !ok {
					t.Errorf("missing group for path %q", path)
					continue
				}

				if !mappingsContainAll(gotMappings, wantMappings) {
					t.Errorf("GroupByPath()[%q] = %v, want %v", path, gotMappings, wantMappings)
				}
			}
		})
	}
}

// mappingsContainAll checks that all expected mappings appear in actual,
// ignoring order.
func mappingsContainAll(actual, expected []SecretMapping) bool {
	if len(actual) != len(expected) {
		return false
	}

	seen := make(map[SecretMapping]bool, len(actual))
	for _, m := range actual {
		seen[m] = true
	}

	for _, m := range expected {
		if !seen[m] {
			return false
		}
	}

	return true
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path     string
		wantPath string
		wantKey  string
	}{
		{"dev/database/url", "dev/database", "url"},
		{"shared/openai/api_key", "shared/openai", "api_key"},
		{"a/b/c/d", "a/b/c", "d"},
		{"noslash", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotPath, gotKey := splitPath(tt.path)
			if gotPath != tt.wantPath || gotKey != tt.wantKey {
				t.Errorf("splitPath(%q) = (%q, %q), want (%q, %q)",
					tt.path, gotPath, gotKey, tt.wantPath, tt.wantKey)
			}
		})
	}
}
