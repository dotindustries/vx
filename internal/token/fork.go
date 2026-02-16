package token

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// StartDaemonProcess spawns "vx daemon start" as a detached background process.
// It returns the child PID on success. If the daemon is already running it
// returns 0, nil.
//
// Note: there is a small TOCTOU window between the IsRunning check and the
// child's own PID-file write. Concurrent callers may both pass the guard, but
// the child's Daemon.Start will detect the duplicate via its own IsRunning
// check and exit. This is acceptable for a CLI tool; file-locking can be
// added if contention becomes an issue.
func StartDaemonProcess(vxBinary string) (int, error) {
	d := NewDaemon(nil) // only used for IsRunning check
	if d.IsRunning() {
		return 0, nil
	}

	logPath := LogPath()
	if err := os.MkdirAll(DefaultDir(), dirPerms); err != nil {
		return 0, fmt.Errorf("create vx dir: %w", err)
	}

	logF, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, filePerms)
	if err != nil {
		return 0, fmt.Errorf("open daemon log: %w", err)
	}
	defer logF.Close()

	cmd := exec.Command(vxBinary, "daemon", "start")
	cmd.Stdout = logF
	cmd.Stderr = logF
	cmd.SysProcAttr = daemonSysProcAttr()

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start daemon process: %w", err)
	}

	pid := cmd.Process.Pid

	// Release the child so the parent can exit without waiting.
	if err := cmd.Process.Release(); err != nil {
		return pid, fmt.Errorf("release daemon process: %w", err)
	}

	// Brief wait then verify the child is still alive.
	time.Sleep(200 * time.Millisecond)
	if !isProcessAlive(pid) {
		return 0, fmt.Errorf("daemon process exited immediately (check %s)", logPath)
	}

	return pid, nil
}
