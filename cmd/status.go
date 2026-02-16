package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/config"
	"go.dot.industries/vx/internal/token"
	"go.dot.industries/vx/internal/vault"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show token and daemon health at a glance",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	printTokenStatus(cfg)
	printDaemonStatus(cfg)

	return nil
}

func printTokenStatus(cfg *config.RootConfig) {
	addr := cfg.Vault.Address
	if flagVaultAddr != "" {
		addr = flagVaultAddr
	}

	tok, err := token.ReadToken()
	if err != nil {
		fmt.Println("Token:  not found")
		return
	}

	client, err := vault.NewClientWithToken(addr, cfg.Vault.BasePath, tok)
	if err != nil {
		fmt.Println("Token:  error (cannot create client)")
		return
	}

	ttl, err := client.TokenTTL()
	if err != nil {
		fmt.Println("Token:  present but unverifiable")
		return
	}

	if ttl <= 0 {
		fmt.Println("Token:  expired")
		return
	}

	expires := time.Now().Add(ttl).Format("15:04:05")
	fmt.Printf("Token:  valid (%s remaining, expires %s)\n", formatDuration(ttl), expires)
}

func printDaemonStatus(cfg *config.RootConfig) {
	addr := cfg.Vault.Address
	if flagVaultAddr != "" {
		addr = flagVaultAddr
	}

	renewer := token.NewTokenRenewer(addr)
	daemon := token.NewDaemon(renewer)

	status, err := daemon.Status()
	if err != nil {
		fmt.Println("Daemon: error")
		return
	}

	if !status.Running {
		fmt.Println("Daemon: not running")
		return
	}

	fmt.Printf("Daemon: running (PID %d)\n", status.PID)
}
