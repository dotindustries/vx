package token

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	if err := writeTokenTo(path, "s.abc123"); err != nil {
		t.Fatalf("writeTokenTo() error = %v", err)
	}

	got, err := readTokenFrom(path)
	if err != nil {
		t.Fatalf("readTokenFrom() error = %v", err)
	}

	if got != "s.abc123" {
		t.Errorf("readTokenFrom() = %q, want %q", got, "s.abc123")
	}
}

func TestWriteTokenCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "sub", "deep", "token")

	if err := writeTokenTo(nested, "s.xyz789"); err != nil {
		t.Fatalf("writeTokenTo() error = %v", err)
	}

	got, err := readTokenFrom(nested)
	if err != nil {
		t.Fatalf("readTokenFrom() error = %v", err)
	}

	if got != "s.xyz789" {
		t.Errorf("readTokenFrom() = %q, want %q", got, "s.xyz789")
	}
}

func TestWriteTokenPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	if err := writeTokenTo(path, "s.secret"); err != nil {
		t.Fatalf("writeTokenTo() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}

	perm := info.Mode().Perm()
	if perm != filePerms {
		t.Errorf("file permissions = %o, want %o", perm, filePerms)
	}
}

func TestWriteTokenDirectoryPermissions(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "newdir")
	path := filepath.Join(subDir, "token")

	if err := writeTokenTo(path, "s.perm"); err != nil {
		t.Fatalf("writeTokenTo() error = %v", err)
	}

	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}

	perm := info.Mode().Perm()
	if perm != dirPerms {
		t.Errorf("directory permissions = %o, want %o", perm, dirPerms)
	}
}

func TestReadTokenMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent")

	_, err := readTokenFrom(path)
	if err == nil {
		t.Fatal("readTokenFrom() expected error for missing file, got nil")
	}
}

func TestReadTokenEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	if err := os.WriteFile(path, []byte(""), filePerms); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := readTokenFrom(path)
	if err == nil {
		t.Fatal("readTokenFrom() expected error for empty file, got nil")
	}
}

func TestReadTokenWhitespaceOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	if err := os.WriteFile(path, []byte("  \n\t\n  "), filePerms); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := readTokenFrom(path)
	if err == nil {
		t.Fatal("readTokenFrom() expected error for whitespace-only file, got nil")
	}
}

func TestRemoveToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	if err := writeTokenTo(path, "s.delete-me"); err != nil {
		t.Fatalf("writeTokenTo() error = %v", err)
	}

	if err := removeTokenAt(path); err != nil {
		t.Fatalf("removeTokenAt() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("removeTokenAt() file still exists after removal")
	}
}

func TestRemoveTokenMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent")

	if err := removeTokenAt(path); err != nil {
		t.Errorf("removeTokenAt() error = %v, want nil for missing file", err)
	}
}

func TestReadTokenTrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	if err := os.WriteFile(path, []byte("  s.padded  \n"), filePerms); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	got, err := readTokenFrom(path)
	if err != nil {
		t.Fatalf("readTokenFrom() error = %v", err)
	}

	if got != "s.padded" {
		t.Errorf("readTokenFrom() = %q, want %q", got, "s.padded")
	}
}
