package vault

import (
	"errors"
	"fmt"
	"net/http"
	"path"

	vaultapi "github.com/hashicorp/vault/api"
)

// ReadKV reads all key-value pairs at the given KV v2 path. The path is
// relative to the client's basePath mount. For example, with basePath "secret"
// and path "dev/database", the full API path is "secret/data/dev/database".
//
// Returns an empty map when the path does not exist (404).
// Returns a wrapped error on permission denied or other failures.
func (c *Client) ReadKV(kvPath string) (map[string]string, error) {
	fullPath := buildKV2Path(c.basePath, kvPath)

	secret, err := c.inner.Logical().Read(fullPath)
	if err != nil {
		if isPermissionDenied(err) {
			return nil, fmt.Errorf("reading KV path %q: permission denied: %w", kvPath, err)
		}
		return nil, fmt.Errorf("reading KV path %q: %w", kvPath, err)
	}

	if secret == nil || secret.Data == nil {
		return make(map[string]string), nil
	}

	return extractKV2Data(secret.Data, kvPath)
}

// buildKV2Path constructs the full KV v2 API path by inserting "data" between
// the mount point and the secret path.
func buildKV2Path(basePath string, kvPath string) string {
	return path.Join(basePath, "data", kvPath)
}

// extractKV2Data parses the nested KV v2 response structure. The Vault KV v2
// API returns data in response.Data["data"] as a nested map.
func extractKV2Data(responseData map[string]interface{}, kvPath string) (map[string]string, error) {
	dataRaw, ok := responseData["data"]
	if !ok {
		return make(map[string]string), nil
	}

	dataMap, ok := dataRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("reading KV path %q: unexpected data format", kvPath)
	}

	result := make(map[string]string, len(dataMap))
	for key, val := range dataMap {
		str, ok := val.(string)
		if !ok {
			continue
		}
		result[key] = str
	}

	return result, nil
}

// VaultEntry represents a key or directory in the Vault KV tree.
type VaultEntry struct {
	Name  string // e.g. "database" or "auth/"
	IsDir bool   // trailing "/" in Vault LIST response indicates a directory
}

// ListKeys lists keys and directories at a KV v2 metadata path. This uses the
// Vault LIST HTTP method on {basePath}/metadata/{kvPath}. Keys ending with "/"
// are directories; others are leaf secrets.
//
// Requires the "list" capability on the metadata path. Returns an empty slice
// when the path does not exist.
func (c *Client) ListKeys(kvPath string) ([]VaultEntry, error) {
	fullPath := buildKV2MetadataPath(c.basePath, kvPath)

	secret, err := c.inner.Logical().List(fullPath)
	if err != nil {
		if isPermissionDenied(err) {
			return nil, fmt.Errorf("listing KV path %q: permission denied: %w", kvPath, err)
		}
		return nil, fmt.Errorf("listing KV path %q: %w", kvPath, err)
	}

	if secret == nil || secret.Data == nil {
		return []VaultEntry{}, nil
	}

	return parseListKeys(secret.Data, kvPath)
}

// buildKV2MetadataPath constructs the KV v2 metadata path for LIST operations.
func buildKV2MetadataPath(basePath string, kvPath string) string {
	return path.Join(basePath, "metadata", kvPath)
}

// parseListKeys extracts VaultEntry items from the LIST response data.
// The Vault KV v2 LIST response returns keys in Data["keys"] as a []interface{}.
func parseListKeys(data map[string]interface{}, kvPath string) ([]VaultEntry, error) {
	keysRaw, ok := data["keys"]
	if !ok {
		return []VaultEntry{}, nil
	}

	keysList, ok := keysRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("listing KV path %q: unexpected keys format", kvPath)
	}

	entries := make([]VaultEntry, 0, len(keysList))
	for _, k := range keysList {
		name, ok := k.(string)
		if !ok {
			continue
		}

		isDir := len(name) > 0 && name[len(name)-1] == '/'
		entries = append(entries, VaultEntry{
			Name:  name,
			IsDir: isDir,
		})
	}

	return entries, nil
}

// isPermissionDenied checks whether a Vault API error is a 403 permission denied.
func isPermissionDenied(err error) bool {
	var respErr *vaultapi.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == http.StatusForbidden
	}
	return false
}
