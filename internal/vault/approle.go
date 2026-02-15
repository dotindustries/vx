package vault

import "fmt"

// AppRoleAuth authenticates to Vault using AppRole credentials. This is
// intended for non-interactive environments such as CI pipelines and Docker
// containers. On success the client's token is set to the newly obtained token.
func AppRoleAuth(client *Client, roleID string, secretID string) error {
	if roleID == "" {
		return fmt.Errorf("approle auth: role_id is required")
	}

	if secretID == "" {
		return fmt.Errorf("approle auth: secret_id is required")
	}

	data := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	secret, err := client.inner.Logical().Write("auth/approle/login", data)
	if err != nil {
		return fmt.Errorf("approle auth: %w", err)
	}

	if secret == nil || secret.Auth == nil {
		return fmt.Errorf("approle auth: empty auth response")
	}

	client.SetToken(secret.Auth.ClientToken)

	return nil
}
