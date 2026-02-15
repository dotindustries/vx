package config

import (
	"os"
	"testing"
)

// writeTestFile is a test helper that writes content to a file path.
func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file %s: %v", path, err)
	}
}
