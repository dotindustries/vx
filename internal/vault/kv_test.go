package vault

import (
	"testing"
)

func TestBuildKV2Path(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		kvPath   string
		want     string
	}{
		{
			name:     "standard path",
			basePath: "secret",
			kvPath:   "dev/database",
			want:     "secret/data/dev/database",
		},
		{
			name:     "nested path",
			basePath: "secret",
			kvPath:   "production/services/api/config",
			want:     "secret/data/production/services/api/config",
		},
		{
			name:     "custom mount",
			basePath: "kv",
			kvPath:   "app/credentials",
			want:     "kv/data/app/credentials",
		},
		{
			name:     "single segment path",
			basePath: "secret",
			kvPath:   "keys",
			want:     "secret/data/keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildKV2Path(tt.basePath, tt.kvPath)
			if got != tt.want {
				t.Errorf("buildKV2Path(%q, %q) = %q, want %q", tt.basePath, tt.kvPath, got, tt.want)
			}
		})
	}
}

func TestExtractKV2Data(t *testing.T) {
	tests := []struct {
		name         string
		responseData map[string]interface{}
		wantKeys     map[string]string
		wantErr      bool
	}{
		{
			name: "valid data",
			responseData: map[string]interface{}{
				"data": map[string]interface{}{
					"username": "admin",
					"password": "s3cret",
				},
				"metadata": map[string]interface{}{
					"version": 1,
				},
			},
			wantKeys: map[string]string{
				"username": "admin",
				"password": "s3cret",
			},
			wantErr: false,
		},
		{
			name:         "missing data key",
			responseData: map[string]interface{}{},
			wantKeys:     map[string]string{},
			wantErr:      false,
		},
		{
			name: "non-string values are skipped",
			responseData: map[string]interface{}{
				"data": map[string]interface{}{
					"url":     "postgres://localhost",
					"port":    5432,
					"enabled": true,
				},
			},
			wantKeys: map[string]string{
				"url": "postgres://localhost",
			},
			wantErr: false,
		},
		{
			name: "invalid data type",
			responseData: map[string]interface{}{
				"data": "not-a-map",
			},
			wantKeys: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractKV2Data(tt.responseData, "test/path")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.wantKeys) {
				t.Fatalf("got %d keys, want %d keys", len(got), len(tt.wantKeys))
			}

			for key, wantVal := range tt.wantKeys {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("missing key %q", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("key %q = %q, want %q", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestReadKV_NoServer(t *testing.T) {
	// Client pointed at a non-existent server should return an error
	// when attempting to read.
	client, err := NewClientWithToken("http://127.0.0.1:1", "secret", "test-token")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	_, readErr := client.ReadKV("dev/database")
	if readErr == nil {
		t.Error("expected error reading from non-existent server, got nil")
	}
}
