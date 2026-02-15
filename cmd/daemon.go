package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/token"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the token renewal daemon",
	Long:  `The daemon automatically renews your Vault token before it expires.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the token renewal daemon in the foreground",
	Args:  cobra.NoArgs,
	RunE:  runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running token renewal daemon",
	Args:  cobra.NoArgs,
	RunE:  runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the daemon status",
	Args:  cobra.NoArgs,
	RunE:  runDaemonStatus,
}

func runDaemonStart(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	renewer := token.NewTokenRenewer(cfg.Vault.Address)
	daemon := token.NewDaemon(renewer)

	if daemon.IsRunning() {
		return fmt.Errorf("daemon is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := daemon.Start(ctx); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	log.Info().Msg("daemon started, press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info().Msg("stopping daemon...")
	if err := daemon.Stop(); err != nil {
		log.Warn().Err(err).Msg("error stopping daemon")
	}

	return nil
}

func runDaemonStop(cmd *cobra.Command, args []string) error {
	pidPath := token.PIDPath()

	data, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("daemon is not running (no PID file)")
	}

	pid := 0
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return fmt.Errorf("invalid PID file: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding daemon process: %w", err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending stop signal: %w", err)
	}

	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		log.Warn().Err(err).Msg("removing PID file")
	}

	log.Info().Int("pid", pid).Msg("daemon stopped")

	return nil
}

func runDaemonStatus(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	renewer := token.NewTokenRenewer(cfg.Vault.Address)
	daemon := token.NewDaemon(renewer)

	status, err := daemon.Status()
	if err != nil {
		return fmt.Errorf("checking daemon status: %w", err)
	}

	if !status.Running {
		fmt.Println("Daemon: not running")
		return nil
	}

	fmt.Printf("Daemon: running (PID %d)\n", status.PID)
	if !status.LastRenewal.IsZero() {
		fmt.Printf("Last renewal: %s\n", status.LastRenewal.Format("2006-01-02 15:04:05"))
	}

	return nil
}
