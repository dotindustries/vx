package migrate

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestConvert_fullConfig(t *testing.T) {
	path := filepath.Join(testdataDir(), "fnox", "fnox.toml")

	fnox, err := LoadFnoxConfig(path)
	if err != nil {
		t.Fatalf("LoadFnoxConfig() error: %v", err)
	}

	result, err := Convert(fnox, "/project")
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	if result.RootConfig == "" {
		t.Fatal("Convert() returned empty RootConfig")
	}

	assertContains(t, result.RootConfig, `address = 'https://vault.example.com'`)
	assertContains(t, result.RootConfig, `base_path = 'secret'`)
}

func TestConvert_environments(t *testing.T) {
	fnox := &FnoxConfig{
		DefaultProvider: "vault-dev",
		Providers: map[string]FnoxProvider{
			"vault-dev":     {Type: "vault", Address: "https://vault.test", Path: "secret/dev"},
			"vault-staging": {Type: "vault", Address: "https://vault.test", Path: "secret/staging"},
			"vault-shared":  {Type: "vault", Address: "https://vault.test", Path: "secret/shared"},
		},
		Secrets:  map[string]FnoxSecret{},
		Profiles: map[string]FnoxProfile{},
	}

	result, err := Convert(fnox, "/project")
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	assertContains(t, result.RootConfig, `default = 'dev'`)
	assertContains(t, result.RootConfig, "dev")
	assertContains(t, result.RootConfig, "staging")
}

func TestConvert_secrets(t *testing.T) {
	fnox := &FnoxConfig{
		DefaultProvider: "vault-dev",
		Providers: map[string]FnoxProvider{
			"vault-dev":    {Type: "vault", Address: "https://vault.test", Path: "secret/dev"},
			"vault-shared": {Type: "vault", Address: "https://vault.test", Path: "secret/shared"},
		},
		Secrets: map[string]FnoxSecret{
			"DATABASE_URL":   {Provider: "vault-dev", Value: "database/url"},
			"OPENAI_API_KEY": {Provider: "vault-shared", Value: "openai/api_key"},
		},
		Profiles: map[string]FnoxProfile{},
	}

	result, err := Convert(fnox, "/project")
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	assertContains(t, result.RootConfig, `DATABASE_URL = '${env}/database/url'`)
	assertContains(t, result.RootConfig, `OPENAI_API_KEY = 'shared/openai/api_key'`)
}

func TestConvert_defaults(t *testing.T) {
	fnox := &FnoxConfig{
		DefaultProvider: "vault-dev",
		Providers: map[string]FnoxProvider{
			"vault-dev": {Type: "vault", Address: "https://vault.test", Path: "secret/dev"},
		},
		Secrets: map[string]FnoxSecret{
			"NODE_ENV": {Default: "development"},
			"APP_URL":  {Default: "http://localhost:3000"},
		},
		Profiles: map[string]FnoxProfile{},
	}

	result, err := Convert(fnox, "/project")
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	assertContains(t, result.RootConfig, `NODE_ENV = 'development'`)
	assertContains(t, result.RootConfig, `APP_URL = 'http://localhost:3000'`)
}

func TestConvert_profileDefaults(t *testing.T) {
	fnox := &FnoxConfig{
		DefaultProvider: "vault-dev",
		Providers: map[string]FnoxProvider{
			"vault-dev":     {Type: "vault", Address: "https://vault.test", Path: "secret/dev"},
			"vault-staging": {Type: "vault", Address: "https://vault.test", Path: "secret/staging"},
		},
		Secrets: map[string]FnoxSecret{
			"NODE_ENV": {Default: "development"},
		},
		Profiles: map[string]FnoxProfile{
			"staging": {
				Secrets: map[string]FnoxSecret{
					"NODE_ENV": {Default: "production"},
				},
			},
		},
	}

	result, err := Convert(fnox, "/project")
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	assertContains(t, result.RootConfig, `NODE_ENV = 'development'`)
	assertContains(t, result.RootConfig, `NODE_ENV = 'production'`)
}

func TestConvert_workspaces(t *testing.T) {
	fnox := &FnoxConfig{
		DefaultProvider: "vault-dev",
		Import:          []string{"./web/fnox.toml", "./packages/api/fnox.toml"},
		Providers: map[string]FnoxProvider{
			"vault-dev": {Type: "vault", Address: "https://vault.test", Path: "secret/dev"},
		},
		Secrets:  map[string]FnoxSecret{},
		Profiles: map[string]FnoxProfile{},
	}

	result, err := Convert(fnox, "/project")
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	assertContains(t, result.RootConfig, "web/vx.toml")
	assertContains(t, result.RootConfig, "packages/api/vx.toml")
}

func TestConvert_integrationsPath(t *testing.T) {
	fnox := &FnoxConfig{
		DefaultProvider: "vault-dev",
		Providers: map[string]FnoxProvider{
			"vault-dev":              {Type: "vault", Address: "https://vault.test", Path: "secret/dev"},
			"vault-dev-integrations": {Type: "vault", Address: "https://vault.test", Path: "secret/dev/integrations"},
		},
		Secrets: map[string]FnoxSecret{
			"NOVU_API_KEY": {Provider: "vault-dev-integrations", Value: "novu/api_key"},
		},
		Profiles: map[string]FnoxProfile{},
	}

	result, err := Convert(fnox, "/project")
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	assertContains(t, result.RootConfig, `NOVU_API_KEY = '${env}/integrations/novu/api_key'`)
}

func TestConvert_nilConfig(t *testing.T) {
	_, err := Convert(nil, "/project")
	if err == nil {
		t.Fatal("Convert(nil) expected error, got nil")
	}
}

func TestConvert_emptyConfig(t *testing.T) {
	fnox := &FnoxConfig{
		Providers: map[string]FnoxProvider{},
		Secrets:   map[string]FnoxSecret{},
		Profiles:  map[string]FnoxProfile{},
	}

	result, err := Convert(fnox, "/project")
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	if result.RootConfig == "" {
		t.Fatal("Convert() returned empty RootConfig for empty config")
	}
}

func TestFormatVxToml(t *testing.T) {
	cfg := vxRoot{
		Vault: vxVault{
			Address:  "https://vault.test",
			BasePath: "secret",
		},
		Environments: vxEnvironments{
			Default:   "dev",
			Available: []string{"dev", "staging"},
		},
	}

	output, err := FormatVxToml(cfg)
	if err != nil {
		t.Fatalf("FormatVxToml() error: %v", err)
	}

	assertContains(t, output, `address = 'https://vault.test'`)
	assertContains(t, output, `base_path = 'secret'`)
	assertContains(t, output, `default = 'dev'`)
}

func TestExtractVaultAddress(t *testing.T) {
	providers := map[string]FnoxProvider{
		"vault-dev": {Address: "https://vault.test"},
	}

	got := extractVaultAddress(providers)
	if got != "https://vault.test" {
		t.Errorf("extractVaultAddress() = %q, want %q", got, "https://vault.test")
	}
}

func TestExtractVaultAddress_empty(t *testing.T) {
	got := extractVaultAddress(map[string]FnoxProvider{})
	if got != "" {
		t.Errorf("extractVaultAddress(empty) = %q, want empty", got)
	}
}

func TestExtractBasePath(t *testing.T) {
	tests := []struct {
		name      string
		providers map[string]FnoxProvider
		want      string
	}{
		{
			name: "common prefix secret",
			providers: map[string]FnoxProvider{
				"vault-dev":     {Path: "secret/dev"},
				"vault-staging": {Path: "secret/staging"},
			},
			want: "secret",
		},
		{
			name:      "empty providers",
			providers: map[string]FnoxProvider{},
			want:      "",
		},
		{
			name: "single provider",
			providers: map[string]FnoxProvider{
				"vault-dev": {Path: "secret/dev"},
			},
			want: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBasePath(tt.providers)
			if got != tt.want {
				t.Errorf("extractBasePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractEnvironments(t *testing.T) {
	providers := map[string]FnoxProvider{
		"vault-dev":              {Path: "secret/dev"},
		"vault-staging":         {Path: "secret/staging"},
		"vault-shared":          {Path: "secret/shared"},
		"vault-dev-integrations": {Path: "secret/dev/integrations"},
	}

	envs := extractEnvironments(providers)

	if len(envs) != 2 {
		t.Fatalf("extractEnvironments() returned %d envs, want 2: %v", len(envs), envs)
	}

	if envs[0] != "dev" || envs[1] != "staging" {
		t.Errorf("extractEnvironments() = %v, want [dev staging]", envs)
	}
}

func TestProviderToEnv(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"vault-dev", "dev"},
		{"vault-staging", "staging"},
		{"vault-shared", "shared"},
		{"vault-dev-integrations", "dev-integrations"},
		{"custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providerToEnv(tt.name)
			if got != tt.want {
				t.Errorf("providerToEnv(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestImportToWorkspace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"./web/fnox.toml", "web/vx.toml"},
		{"./packages/api/fnox.toml", "packages/api/vx.toml"},
		{"fnox.toml", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := importToWorkspace(tt.input)
			if got != tt.want {
				t.Errorf("importToWorkspace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		segments []string
		want     string
	}{
		{[]string{"${env}", "database", "url"}, "${env}/database/url"},
		{[]string{"shared", "openai", "api_key"}, "shared/openai/api_key"},
		{[]string{"${env}", "", "value"}, "${env}/value"},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := joinPath(tt.segments...)
			if got != tt.want {
				t.Errorf("joinPath(%v) = %q, want %q", tt.segments, got, tt.want)
			}
		})
	}
}

func TestCommonPrefix(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{"same prefix", []string{"secret/dev", "secret/staging"}, "secret"},
		{"single path", []string{"secret/dev"}, "secret"},
		{"no common", []string{"secret/dev", "other/staging"}, ""},
		{"empty", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commonPrefix(tt.paths)
			if got != tt.want {
				t.Errorf("commonPrefix(%v) = %q, want %q", tt.paths, got, tt.want)
			}
		})
	}
}

// assertContains checks that haystack contains needle.
func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()

	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q, got:\n%s", needle, haystack)
	}
}
