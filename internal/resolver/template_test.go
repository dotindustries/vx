package resolver

import "testing"

func TestInterpolate(t *testing.T) {
	tests := []struct {
		name string
		path string
		env  string
		want string
	}{
		{
			name: "single env placeholder",
			path: "${env}/database/url",
			env:  "dev",
			want: "dev/database/url",
		},
		{
			name: "no placeholder (shared path)",
			path: "shared/openai/api_key",
			env:  "dev",
			want: "shared/openai/api_key",
		},
		{
			name: "multiple env placeholders",
			path: "${env}/prefix/${env}/suffix",
			env:  "staging",
			want: "staging/prefix/staging/suffix",
		},
		{
			name: "empty env",
			path: "${env}/database/url",
			env:  "",
			want: "/database/url",
		},
		{
			name: "empty path",
			path: "",
			env:  "dev",
			want: "",
		},
		{
			name: "production environment",
			path: "${env}/stripe/secret_key",
			env:  "production",
			want: "production/stripe/secret_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Interpolate(tt.path, tt.env)
			if got != tt.want {
				t.Errorf("Interpolate(%q, %q) = %q, want %q", tt.path, tt.env, got, tt.want)
			}
		})
	}
}

func TestHasEnvVar(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "has env var",
			path: "${env}/database/url",
			want: true,
		},
		{
			name: "no env var",
			path: "shared/openai/api_key",
			want: false,
		},
		{
			name: "empty path",
			path: "",
			want: false,
		},
		{
			name: "partial match",
			path: "$env/database/url",
			want: false,
		},
		{
			name: "multiple env vars",
			path: "${env}/${env}",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasEnvVar(tt.path)
			if got != tt.want {
				t.Errorf("HasEnvVar(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
