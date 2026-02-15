package vault

import (
	"testing"
)

func TestAppRoleAuth_EmptyRoleID(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:8200", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	err = AppRoleAuth(client, "", "some-secret-id")
	if err == nil {
		t.Fatal("expected error for empty role_id, got nil")
	}
}

func TestAppRoleAuth_EmptySecretID(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:8200", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	err = AppRoleAuth(client, "some-role-id", "")
	if err == nil {
		t.Fatal("expected error for empty secret_id, got nil")
	}
}

func TestAppRoleAuth_NoServer(t *testing.T) {
	// With a non-reachable server, AppRoleAuth should return an error.
	client, err := NewClient("http://127.0.0.1:1", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	err = AppRoleAuth(client, "role-id", "secret-id")
	if err == nil {
		t.Fatal("expected error for non-reachable server, got nil")
	}
}
