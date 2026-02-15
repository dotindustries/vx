package vault

import (
	"testing"
)

func TestParseListKeys(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]interface{}
		want    []VaultEntry
		wantErr bool
	}{
		{
			name: "mixed directories and keys",
			data: map[string]interface{}{
				"keys": []interface{}{
					"auth/",
					"database/",
					"api_key",
					"app_secret",
				},
			},
			want: []VaultEntry{
				{Name: "auth/", IsDir: true},
				{Name: "database/", IsDir: true},
				{Name: "api_key", IsDir: false},
				{Name: "app_secret", IsDir: false},
			},
		},
		{
			name: "only directories",
			data: map[string]interface{}{
				"keys": []interface{}{
					"auth/",
					"database/",
				},
			},
			want: []VaultEntry{
				{Name: "auth/", IsDir: true},
				{Name: "database/", IsDir: true},
			},
		},
		{
			name: "only leaf keys",
			data: map[string]interface{}{
				"keys": []interface{}{
					"api_key",
					"secret",
				},
			},
			want: []VaultEntry{
				{Name: "api_key", IsDir: false},
				{Name: "secret", IsDir: false},
			},
		},
		{
			name: "empty keys list",
			data: map[string]interface{}{
				"keys": []interface{}{},
			},
			want: []VaultEntry{},
		},
		{
			name: "no keys field",
			data: map[string]interface{}{},
			want: []VaultEntry{},
		},
		{
			name: "invalid keys format",
			data: map[string]interface{}{
				"keys": "not-a-list",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseListKeys(tt.data, "test/path")
			if (err != nil) != tt.wantErr {
				t.Errorf("parseListKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("parseListKeys() returned %d entries, want %d", len(got), len(tt.want))
				return
			}

			for i, entry := range got {
				if entry.Name != tt.want[i].Name {
					t.Errorf("entry[%d].Name = %q, want %q", i, entry.Name, tt.want[i].Name)
				}
				if entry.IsDir != tt.want[i].IsDir {
					t.Errorf("entry[%d].IsDir = %v, want %v", i, entry.IsDir, tt.want[i].IsDir)
				}
			}
		})
	}
}

func TestBuildKV2MetadataPath(t *testing.T) {
	tests := []struct {
		basePath string
		kvPath   string
		want     string
	}{
		{"secret", "dev/database", "secret/metadata/dev/database"},
		{"secret", "", "secret/metadata"},
		{"kv", "auth/tokens", "kv/metadata/auth/tokens"},
	}

	for _, tt := range tests {
		got := buildKV2MetadataPath(tt.basePath, tt.kvPath)
		if got != tt.want {
			t.Errorf("buildKV2MetadataPath(%q, %q) = %q, want %q",
				tt.basePath, tt.kvPath, got, tt.want)
		}
	}
}
