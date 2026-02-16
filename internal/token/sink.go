package token

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	dirName    = ".vx"
	tokenFile  = "token"
	pidFile    = "daemon.pid"
	socketFile = "daemon.sock"
	logFile    = "daemon.log"
	dirPerms   = 0700
	filePerms  = 0600
)

// defaultDir returns the default vx configuration directory (~/.vx).
func defaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", dirName)
	}
	return filepath.Join(home, dirName)
}

// DefaultDir returns the default vx configuration directory (~/.vx).
var DefaultDir = defaultDir

// TokenPath returns the path to the token sink file (~/.vx/token).
var TokenPath = func() string {
	return filepath.Join(DefaultDir(), tokenFile)
}

// PIDPath returns the path to the daemon PID file (~/.vx/daemon.pid).
var PIDPath = func() string {
	return filepath.Join(DefaultDir(), pidFile)
}

// SocketPath returns the path to the daemon Unix socket (~/.vx/daemon.sock).
var SocketPath = func() string {
	return filepath.Join(DefaultDir(), socketFile)
}

// LogPath returns the path to the daemon log file (~/.vx/daemon.log).
var LogPath = func() string {
	return filepath.Join(DefaultDir(), logFile)
}

// ReadToken reads the Vault token from the sink file. Returns an error if the
// file does not exist or the token is empty.
func ReadToken() (string, error) {
	return readTokenFrom(TokenPath())
}

// WriteToken writes the Vault token to the sink file with 0600 permissions.
// The parent directory (~/.vx) is created with 0700 permissions if it does not
// exist.
func WriteToken(token string) error {
	return writeTokenTo(TokenPath(), token)
}

// RemoveToken removes the token sink file. Returns nil if the file does not
// exist.
func RemoveToken() error {
	return removeTokenAt(TokenPath())
}

// readTokenFrom reads a token from the given path.
func readTokenFrom(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read token: %w", err)
	}

	tok := strings.TrimSpace(string(data))
	if tok == "" {
		return "", fmt.Errorf("read token: file is empty")
	}

	return tok, nil
}

// writeTokenTo writes a token to the given path, creating the parent directory
// if necessary.
func writeTokenTo(path string, token string) error {
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, dirPerms); err != nil {
		return fmt.Errorf("write token: create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(token+"\n"), filePerms); err != nil {
		return fmt.Errorf("write token: %w", err)
	}

	return nil
}

// removeTokenAt removes the token file at the given path. Returns nil if the
// file does not exist.
func removeTokenAt(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove token: %w", err)
	}
	return nil
}
