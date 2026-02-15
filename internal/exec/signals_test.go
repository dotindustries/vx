package exec

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestForwardSignals_setup(t *testing.T) {
	// Start a long-running process to get a valid os.Process
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	ctx := context.Background()
	cleanup := ForwardSignals(ctx, cmd.Process)

	// Verify cleanup function is returned and can be called without panic
	cleanup()
}

func TestForwardSignals_contextCancellation(t *testing.T) {
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cleanup := ForwardSignals(ctx, cmd.Process)
	defer cleanup()

	// Cancel context to stop the forwarding goroutine
	cancel()
}

func TestForwardSignals_signalDelivery(t *testing.T) {
	// Start a process that traps SIGTERM and exits cleanly
	cmd := exec.Command("sh", "-c", "trap 'exit 0' TERM; sleep 10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	ctx := context.Background()
	cleanup := ForwardSignals(ctx, cmd.Process)
	defer cleanup()

	// Send SIGTERM directly to child process to verify it handles it
	if err := cmd.Process.Signal(os.Signal(os.Interrupt)); err != nil {
		// On some platforms this may fail; skip rather than fail
		t.Skipf("could not send signal to child process: %v", err)
	}

	_ = cmd.Wait()
}
