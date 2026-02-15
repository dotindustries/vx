package migrate

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

func TestLoadFnoxConfig(t *testing.T) {
	path := filepath.Join(testdataDir(), "fnox", "fnox.toml")

	cfg, err := LoadFnoxConfig(path)
	if err != nil {
		t.Fatalf("LoadFnoxConfig() error: %v", err)
	}

	if cfg.DefaultProvider != "vault-dev" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "vault-dev")
	}

	if len(cfg.Import) != 2 {
		t.Errorf("Import count = %d, want 2", len(cfg.Import))
	}

	if len(cfg.Providers) != 4 {
		t.Errorf("Providers count = %d, want 4", len(cfg.Providers))
	}

	if len(cfg.Secrets) != 5 {
		t.Errorf("Secrets count = %d, want 5", len(cfg.Secrets))
	}
}

func TestLoadFnoxConfig_providers(t *testing.T) {
	path := filepath.Join(testdataDir(), "fnox", "fnox.toml")

	cfg, err := LoadFnoxConfig(path)
	if err != nil {
		t.Fatalf("LoadFnoxConfig() error: %v", err)
	}

	tests := []struct {
		name    string
		wantType    string
		wantAddress string
		wantPath    string
	}{
		{"vault-dev", "vault", "https://vault.example.com", "secret/dev"},
		{"vault-staging", "vault", "https://vault.example.com", "secret/staging"},
		{"vault-dev-integrations", "vault", "https://vault.example.com", "secret/dev/integrations"},
		{"vault-shared", "vault", "https://vault.example.com", "secret/shared"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := cfg.Providers[tt.name]
			if !ok {
				t.Fatalf("provider %q not found", tt.name)
			}
			if p.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", p.Type, tt.wantType)
			}
			if p.Address != tt.wantAddress {
				t.Errorf("Address = %q, want %q", p.Address, tt.wantAddress)
			}
			if p.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", p.Path, tt.wantPath)
			}
		})
	}
}

func TestLoadFnoxConfig_secrets(t *testing.T) {
	path := filepath.Join(testdataDir(), "fnox", "fnox.toml")

	cfg, err := LoadFnoxConfig(path)
	if err != nil {
		t.Fatalf("LoadFnoxConfig() error: %v", err)
	}

	tests := []struct {
		name        string
		wantProvider string
		wantValue    string
		wantDefault  string
	}{
		{"DATABASE_URL", "vault-dev", "database/url", ""},
		{"DATABASE_AUTH_TOKEN", "vault-dev", "database/auth_token", ""},
		{"OPENAI_API_KEY", "vault-shared", "openai/api_key", ""},
		{"NODE_ENV", "", "", "development"},
		{"APP_URL", "", "", "http://localhost:3000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, ok := cfg.Secrets[tt.name]
			if !ok {
				t.Fatalf("secret %q not found", tt.name)
			}
			if s.Provider != tt.wantProvider {
				t.Errorf("Provider = %q, want %q", s.Provider, tt.wantProvider)
			}
			if s.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", s.Value, tt.wantValue)
			}
			if s.Default != tt.wantDefault {
				t.Errorf("Default = %q, want %q", s.Default, tt.wantDefault)
			}
		})
	}
}

func TestLoadFnoxConfig_profiles(t *testing.T) {
	path := filepath.Join(testdataDir(), "fnox", "fnox.toml")

	cfg, err := LoadFnoxConfig(path)
	if err != nil {
		t.Fatalf("LoadFnoxConfig() error: %v", err)
	}

	if len(cfg.Profiles) != 2 {
		t.Fatalf("Profiles count = %d, want 2", len(cfg.Profiles))
	}

	staging, ok := cfg.Profiles["staging"]
	if !ok {
		t.Fatal("staging profile not found")
	}

	if len(staging.Secrets) != 2 {
		t.Errorf("staging secrets count = %d, want 2", len(staging.Secrets))
	}

	dbURL, ok := staging.Secrets["DATABASE_URL"]
	if !ok {
		t.Fatal("staging DATABASE_URL not found")
	}
	if dbURL.Provider != "vault-staging" {
		t.Errorf("staging DATABASE_URL provider = %q, want %q", dbURL.Provider, "vault-staging")
	}
}

func TestLoadFnoxConfig_fileNotFound(t *testing.T) {
	_, err := LoadFnoxConfig("/nonexistent/fnox.toml")
	if err == nil {
		t.Fatal("LoadFnoxConfig() expected error for missing file, got nil")
	}
}

func TestLoadFnoxConfig_invalidToml(t *testing.T) {
	// Create a temp file with invalid TOML
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")

	if err := writeTestFile(path, "invalid [[ toml"); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := LoadFnoxConfig(path)
	if err == nil {
		t.Fatal("LoadFnoxConfig() expected error for invalid TOML, got nil")
	}
}

func writeTestFile(path string, content string) error {
	return writeFile(path, []byte(content))
}
