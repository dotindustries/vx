package token

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestStartDaemonProcess_AlreadyRunning(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "daemon.pid")
	overridePIDPath(t, pidPath)

	// Write the current process PID so IsRunning returns true.
	writeTokenTo(pidPath, strconv.Itoa(os.Getpid()))

	pid, err := StartDaemonProcess("/nonexistent/binary")
	if err != nil {
		t.Fatalf("StartDaemonProcess() error = %v, want nil (already running)", err)
	}
	if pid != 0 {
		t.Errorf("StartDaemonProcess() pid = %d, want 0 (already running)", pid)
	}
}

func TestStartDaemonProcess_BadBinary(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "daemon.pid")
	logPath := filepath.Join(dir, "daemon.log")
	overridePIDPath(t, pidPath)
	overrideLogPath(t, logPath)
	overrideDefaultDir(t, dir)

	_, err := StartDaemonProcess("/nonexistent/vx-binary")
	if err == nil {
		t.Fatal("StartDaemonProcess() expected error for bad binary, got nil")
	}
}

// overrideLogPath temporarily overrides LogPath for tests.
func overrideLogPath(t *testing.T, path string) {
	t.Helper()
	orig := LogPath
	LogPath = func() string { return path }
	t.Cleanup(func() { LogPath = orig })
}

// overrideDefaultDir temporarily overrides DefaultDir for tests.
func overrideDefaultDir(t *testing.T, dir string) {
	t.Helper()
	orig := DefaultDir
	DefaultDir = func() string { return dir }
	t.Cleanup(func() { DefaultDir = orig })
}
