package exec

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// ForwardSignals starts a goroutine that forwards SIGINT, SIGTERM, and
// SIGHUP to the given child process. Returns a cleanup function that
// stops signal forwarding and must be called when the child exits.
func ForwardSignals(ctx context.Context, process *os.Process) func() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	done := make(chan struct{})

	go forwardLoop(ctx, process, sigChan, done)

	return func() {
		signal.Stop(sigChan)
		close(done)
	}
}

// forwardLoop receives signals from sigChan and sends them to the child
// process. It exits when done is closed or the context is cancelled.
func forwardLoop(ctx context.Context, process *os.Process, sigChan <-chan os.Signal, done <-chan struct{}) {
	for {
		select {
		case sig := <-sigChan:
			_ = process.Signal(sig)
		case <-done:
			return
		case <-ctx.Done():
			return
		}
	}
}
