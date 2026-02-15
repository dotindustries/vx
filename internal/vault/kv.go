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

// isPermissionDenied checks whether a Vault API error is a 403 permission denied.
func isPermissionDenied(err error) bool {
	var respErr *vaultapi.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == http.StatusForbidden
	}
	return false
}
