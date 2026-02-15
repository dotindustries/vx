package vault

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		basePath string
		wantErr  bool
	}{
		{
			name:     "valid address",
			address:  "http://127.0.0.1:8200",
			basePath: "secret",
			wantErr:  false,
		},
		{
			name:     "empty address",
			address:  "",
			basePath: "secret",
			wantErr:  true,
		},
		{
			name:     "empty base path is allowed",
			address:  "http://127.0.0.1:8200",
			basePath: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.address, tt.basePath)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if client == nil {
				t.Fatal("expected non-nil client")
			}

			if client.basePath != tt.basePath {
				t.Errorf("basePath = %q, want %q", client.basePath, tt.basePath)
			}
		})
	}
}

func TestNewClientWithToken(t *testing.T) {
	client, err := NewClientWithToken("http://127.0.0.1:8200", "secret", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := client.Token(); got != "test-token" {
		t.Errorf("Token() = %q, want %q", got, "test-token")
	}
}

func TestTokenSetAndGet(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "")

	client, err := NewClient("http://127.0.0.1:8200", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := client.Token(); got != "" {
		t.Errorf("Token() = %q, want empty string", got)
	}

	client.SetToken("my-token")

	if got := client.Token(); got != "my-token" {
		t.Errorf("Token() = %q, want %q", got, "my-token")
	}
}

func TestIsAuthenticated_NoToken(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:8200", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.IsAuthenticated() {
		t.Error("IsAuthenticated() = true, want false for client without token")
	}
}

func TestIsAuthenticated_InvalidToken(t *testing.T) {
	client, err := NewClientWithToken("http://127.0.0.1:8200", "secret", "invalid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With a non-reachable server and invalid token, IsAuthenticated should
	// return false because the token lookup will fail.
	if client.IsAuthenticated() {
		t.Error("IsAuthenticated() = true, want false for invalid token")
	}
}
