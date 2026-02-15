package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddMapping(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		envVar   string
		path     string
		wantKey  string
		wantVal  string
	}{
		{
			name: "add to existing secrets section",
			initial: `# Root config
[vault]
address = "https://vault.example.com"

[secrets]
# Existing secret
DATABASE_URL = "${env}/database/url"
`,
			envVar:  "API_KEY",
			path:    "${env}/api/key",
			wantKey: "API_KEY",
			wantVal: `"${env}/api/key"`,
		},
		{
			name: "create secrets section when missing",
			initial: `[vault]
address = "https://vault.example.com"
`,
			envVar:  "DATABASE_URL",
			path:    "${env}/database/url",
			wantKey: "DATABASE_URL",
			wantVal: `"${env}/database/url"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "vx.toml")
			if err := os.WriteFile(filePath, []byte(tt.initial), 0644); err != nil {
				t.Fatal(err)
			}

			b := New("", "", "", "", "")
			if err := b.AddMapping(filePath, tt.envVar, tt.path); err != nil {
				t.Fatal(err)
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatal(err)
			}

			content := string(data)
			if !strings.Contains(content, tt.wantKey) {
				t.Errorf("output missing key %q:\n%s", tt.wantKey, content)
			}
			if !strings.Contains(content, tt.wantVal) {
				t.Errorf("output missing value %q:\n%s", tt.wantVal, content)
			}
		})
	}
}

func TestAddMapping_PreservesComments(t *testing.T) {
	initial := `# Root configuration file
# for the vx secret manager

[vault]
address = "https://vault.example.com"

[secrets]
# Database credentials
DATABASE_URL = "${env}/database/url"
# Auth secrets
AUTH_SECRET = "${env}/auth/secret"
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "vx.toml")
	if err := os.WriteFile(filePath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	b := New("", "", "", "", "")
	if err := b.AddMapping(filePath, "NEW_KEY", "${env}/new/key"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	// Comments should be preserved
	if !strings.Contains(content, "# Root configuration file") {
		t.Error("top-level comment was not preserved")
	}
	if !strings.Contains(content, "# Database credentials") {
		t.Error("inline comment was not preserved")
	}
	if !strings.Contains(content, "# Auth secrets") {
		t.Error("auth comment was not preserved")
	}
	// New key should be added
	if !strings.Contains(content, "NEW_KEY") {
		t.Error("new key was not added")
	}
}

func TestEditMapping(t *testing.T) {
	initial := `[secrets]
DATABASE_URL = "${env}/database/url"
API_KEY = "${env}/api/key"
`

	tests := []struct {
		name      string
		oldEnvVar string
		newEnvVar string
		newPath   string
		wantKey   string
		wantVal   string
	}{
		{
			name:      "update value only",
			oldEnvVar: "DATABASE_URL",
			newEnvVar: "DATABASE_URL",
			newPath:   "${env}/database/new_url",
			wantKey:   "DATABASE_URL",
			wantVal:   `"${env}/database/new_url"`,
		},
		{
			name:      "rename key and update value",
			oldEnvVar: "API_KEY",
			newEnvVar: "OPENAI_KEY",
			newPath:   "${env}/openai/key",
			wantKey:   "OPENAI_KEY",
			wantVal:   `"${env}/openai/key"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "vx.toml")
			if err := os.WriteFile(filePath, []byte(initial), 0644); err != nil {
				t.Fatal(err)
			}

			b := New("", "", "", "", "")
			if err := b.EditMapping(filePath, tt.oldEnvVar, tt.newEnvVar, tt.newPath); err != nil {
				t.Fatal(err)
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatal(err)
			}

			content := string(data)
			if !strings.Contains(content, tt.wantKey) {
				t.Errorf("output missing key %q:\n%s", tt.wantKey, content)
			}
			if !strings.Contains(content, tt.wantVal) {
				t.Errorf("output missing value %q:\n%s", tt.wantVal, content)
			}
		})
	}
}

func TestEditMapping_NotFound(t *testing.T) {
	initial := `[secrets]
DATABASE_URL = "${env}/database/url"
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "vx.toml")
	if err := os.WriteFile(filePath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	b := New("", "", "", "", "")
	err := b.EditMapping(filePath, "NONEXISTENT", "NEW_NAME", "new/path")
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestDeleteMapping(t *testing.T) {
	initial := `[secrets]
DATABASE_URL = "${env}/database/url"
API_KEY = "${env}/api/key"
AUTH_SECRET = "${env}/auth/secret"
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "vx.toml")
	if err := os.WriteFile(filePath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	b := New("", "", "", "", "")
	if err := b.DeleteMapping(filePath, "API_KEY"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if strings.Contains(content, "API_KEY") {
		t.Error("deleted key should not be present")
	}
	if !strings.Contains(content, "DATABASE_URL") {
		t.Error("other keys should be preserved")
	}
	if !strings.Contains(content, "AUTH_SECRET") {
		t.Error("other keys should be preserved")
	}
}

func TestDeleteMapping_NotFound(t *testing.T) {
	initial := `[secrets]
DATABASE_URL = "${env}/database/url"
`

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "vx.toml")
	if err := os.WriteFile(filePath, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	b := New("", "", "", "", "")
	err := b.DeleteMapping(filePath, "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for non-existent key")
	}
}
