package token

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// DaemonStatus represents the current state of the background renewal daemon.
type DaemonStatus struct {
	Running     bool
	PID         int
	TokenTTL    time.Duration
	LastRenewal time.Time
}

// Daemon manages a background token renewal process.
type Daemon struct {
	renewer     *TokenRenewer
	stop        chan struct{}
	mu          sync.Mutex
	lastRenewal time.Time
}

// NewDaemon creates a new Daemon with the given TokenRenewer.
func NewDaemon(renewer *TokenRenewer) *Daemon {
	return &Daemon{
		renewer: renewer,
		stop:    make(chan struct{}),
	}
}

// Start begins the daemon renewal loop. It writes a PID file, periodically
// checks for token renewal, and cleans up on exit.
func (d *Daemon) Start(ctx context.Context) error {
	if d.IsRunning() {
		return fmt.Errorf("daemon: already running")
	}

	if err := writePIDFile(PIDPath(), os.Getpid()); err != nil {
		return fmt.Errorf("daemon: %w", err)
	}

	go d.loop(ctx)

	return nil
}

// Stop signals the daemon to stop and cleans up the PID file.
func (d *Daemon) Stop() error {
	select {
	case <-d.stop:
		return fmt.Errorf("daemon: not running")
	default:
		close(d.stop)
	}

	return removePIDFile(PIDPath())
}

// IsRunning reports whether the daemon process is alive by checking the PID
// file and sending a zero signal to the process.
func (d *Daemon) IsRunning() bool {
	pid, err := readPIDFile(PIDPath())
	if err != nil {
		return false
	}

	return isProcessAlive(pid)
}

// Status returns the current daemon status including PID, token TTL, and last
// renewal time.
func (d *Daemon) Status() (DaemonStatus, error) {
	pid, err := readPIDFile(PIDPath())
	if err != nil {
		return DaemonStatus{Running: false}, nil
	}

	alive := isProcessAlive(pid)

	d.mu.Lock()
	lastRenewal := d.lastRenewal
	d.mu.Unlock()

	return DaemonStatus{
		Running:     alive,
		PID:         pid,
		LastRenewal: lastRenewal,
	}, nil
}

// loop runs the periodic renewal check until stopped or the context is
// cancelled.
func (d *Daemon) loop(ctx context.Context) {
	defer removePIDFile(PIDPath())

	ticker := time.NewTicker(d.renewer.checkInterval)
	defer ticker.Stop()

	// Perform an immediate check on startup.
	d.tryRenew(ctx)

	for {
		select {
		case <-d.stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.tryRenew(ctx)
		}
	}
}

// tryRenew attempts a single renewal and records the time on success.
func (d *Daemon) tryRenew(ctx context.Context) {
	if err := d.renewer.RenewOnce(ctx); err != nil {
		return
	}

	d.mu.Lock()
	d.lastRenewal = time.Now()
	d.mu.Unlock()
}

// writePIDFile writes the process ID to the given path.
func writePIDFile(path string, pid int) error {
	return writeTokenTo(path, strconv.Itoa(pid))
}

// readPIDFile reads and parses the PID from the given path.
func readPIDFile(path string) (int, error) {
	data, err := readTokenFrom(path)
	if err != nil {
		return 0, fmt.Errorf("read pid: %w", err)
	}

	pid, err := strconv.Atoi(data)
	if err != nil {
		return 0, fmt.Errorf("parse pid: %w", err)
	}

	return pid, nil
}

// removePIDFile removes the PID file. Returns nil if the file does not exist.
func removePIDFile(path string) error {
	return removeTokenAt(path)
}

// isProcessAlive checks whether a process with the given PID exists by sending
// signal 0.
func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
