package token

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestDaemonStartStop(t *testing.T) {
	srv := newStubVaultServer(t, 7200, 86400, true)
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	pidPath := filepath.Join(dir, "daemon.pid")
	writeTokenTo(tokenPath, "s.daemon-test")

	renewer := NewTokenRenewer(srv.URL,
		WithTokenPath(tokenPath),
		WithCheckInterval(50*time.Millisecond),
	)

	daemon := NewDaemon(renewer)
	// Override PID path for testing by using a helper.
	overridePIDPath(t, pidPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := daemon.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give the loop time to start.
	time.Sleep(100 * time.Millisecond)

	if err := daemon.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestDaemonIsRunning_NoProcess(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "daemon.pid")
	overridePIDPath(t, pidPath)

	renewer := NewTokenRenewer("http://localhost:8200")
	daemon := NewDaemon(renewer)

	if daemon.IsRunning() {
		t.Error("IsRunning() = true, want false when no PID file exists")
	}
}

func TestDaemonIsRunning_StalePID(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "daemon.pid")
	overridePIDPath(t, pidPath)

	// Write a PID that almost certainly doesn't exist.
	writeTokenTo(pidPath, "9999999")

	renewer := NewTokenRenewer("http://localhost:8200")
	daemon := NewDaemon(renewer)

	if daemon.IsRunning() {
		t.Error("IsRunning() = true, want false for stale PID")
	}
}

func TestDaemonStatus_NotRunning(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "daemon.pid")
	overridePIDPath(t, pidPath)

	renewer := NewTokenRenewer("http://localhost:8200")
	daemon := NewDaemon(renewer)

	status, err := daemon.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status.Running {
		t.Error("Status().Running = true, want false")
	}
}

func TestDaemonStatus_Running(t *testing.T) {
	srv := newStubVaultServer(t, 7200, 86400, true)
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	pidPath := filepath.Join(dir, "daemon.pid")
	writeTokenTo(tokenPath, "s.status-test")
	overridePIDPath(t, pidPath)

	renewer := NewTokenRenewer(srv.URL,
		WithTokenPath(tokenPath),
		WithCheckInterval(50*time.Millisecond),
	)

	daemon := NewDaemon(renewer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := daemon.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer daemon.Stop()

	time.Sleep(100 * time.Millisecond)

	status, err := daemon.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if !status.Running {
		t.Error("Status().Running = false, want true")
	}

	if status.PID != os.Getpid() {
		t.Errorf("Status().PID = %d, want %d", status.PID, os.Getpid())
	}
}

func TestDaemonDoubleStart(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "daemon.pid")
	tokenPath := filepath.Join(dir, "token")
	overridePIDPath(t, pidPath)
	writeTokenTo(tokenPath, "s.double")

	// Write our own PID to simulate an already-running daemon.
	writeTokenTo(pidPath, strconv.Itoa(os.Getpid()))

	srv := newStubVaultServer(t, 7200, 86400, true)
	defer srv.Close()

	renewer := NewTokenRenewer(srv.URL,
		WithTokenPath(tokenPath),
		WithCheckInterval(time.Hour),
	)

	daemon := NewDaemon(renewer)

	err := daemon.Start(context.Background())
	if err == nil {
		t.Fatal("Start() expected error for already-running daemon, got nil")
	}
}

func TestDaemonStopWhenNotRunning(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "daemon.pid")
	overridePIDPath(t, pidPath)

	renewer := NewTokenRenewer("http://localhost:8200")
	daemon := NewDaemon(renewer)

	// Close the stop channel to simulate stopped state.
	close(daemon.stop)

	err := daemon.Stop()
	if err == nil {
		t.Fatal("Stop() expected error for already-stopped daemon, got nil")
	}
}

func TestWriteAndReadPIDFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pid")

	if err := writePIDFile(path, 12345); err != nil {
		t.Fatalf("writePIDFile() error = %v", err)
	}

	got, err := readPIDFile(path)
	if err != nil {
		t.Fatalf("readPIDFile() error = %v", err)
	}

	if got != 12345 {
		t.Errorf("readPIDFile() = %d, want %d", got, 12345)
	}
}

func TestReadPIDFile_Missing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.pid")

	_, err := readPIDFile(path)
	if err == nil {
		t.Fatal("readPIDFile() expected error for missing file, got nil")
	}
}

func TestRemovePIDFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pid")

	writePIDFile(path, 99999)

	if err := removePIDFile(path); err != nil {
		t.Fatalf("removePIDFile() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("removePIDFile() file still exists after removal")
	}
}

func TestRemovePIDFile_Missing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.pid")

	if err := removePIDFile(path); err != nil {
		t.Errorf("removePIDFile() error = %v, want nil for missing file", err)
	}
}

func TestDaemonContextCancellation(t *testing.T) {
	srv := newStubVaultServer(t, 7200, 86400, true)
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token")
	pidPath := filepath.Join(dir, "daemon.pid")
	writeTokenTo(tokenPath, "s.ctx-cancel")
	overridePIDPath(t, pidPath)

	renewer := NewTokenRenewer(srv.URL,
		WithTokenPath(tokenPath),
		WithCheckInterval(50*time.Millisecond),
	)

	daemon := NewDaemon(renewer)

	ctx, cancel := context.WithCancel(context.Background())

	if err := daemon.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	// PID file should be cleaned up after context cancellation.
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file should be removed after context cancellation")
	}
}

// newStubVaultServer creates a test HTTP server that responds to Vault
// lookup-self and renew-self endpoints.
func newStubVaultServer(t *testing.T, ttl int, creationTTL int, renewable bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/token/lookup-self":
			resp := tokenLookupResponse{}
			resp.Data.TTL = ttl
			resp.Data.CreationTTL = creationTTL
			resp.Data.Renewable = renewable
			json.NewEncoder(w).Encode(resp)
		case "/v1/auth/token/renew-self":
			resp := tokenRenewResponse{}
			resp.Auth.ClientToken = "s.renewed"
			json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// overridePIDPath is a test helper that temporarily overrides the PIDPath
// function by setting a custom PID path via the environment. Since the actual
// implementation uses a fixed path, we use a file-based approach: the test
// creates PID files at known paths and the daemon methods read from those
// paths. This helper ensures the daemon uses the temp directory path.
//
// Note: This works because IsRunning, Start, Stop, and Status all call
// PIDPath() which returns a fixed path. For tests, we work around this
// by directly manipulating files at the expected paths.
func overridePIDPath(t *testing.T, path string) {
	t.Helper()

	origPIDPath := PIDPath
	PIDPath = func() string { return path }
	t.Cleanup(func() { PIDPath = origPIDPath })
}
