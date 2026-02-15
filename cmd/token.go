package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/token"
	"go.dot.industries/vx/internal/vault"
)

func init() {
	rootCmd.AddCommand(tokenCmd)
	tokenCmd.AddCommand(tokenStatusCmd)
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage Vault tokens",
}

var tokenStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current Vault token status and TTL",
	Args:  cobra.NoArgs,
	RunE:  runTokenStatus,
}

func runTokenStatus(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	tok, err := token.ReadToken()
	if err != nil {
		fmt.Println("Token: not found")
		fmt.Printf("Token path: %s\n", token.TokenPath())
		return nil
	}

	client, err := vault.NewClientWithToken(cfg.Vault.Address, cfg.Vault.BasePath, tok)
	if err != nil {
		return fmt.Errorf("creating vault client: %w", err)
	}

	ttl, err := client.TokenTTL()
	if err != nil {
		fmt.Println("Token: present but cannot verify (lookup failed)")
		return nil
	}

	if ttl <= 0 {
		fmt.Println("Token: expired")
		return nil
	}

	fmt.Println("Token: valid")
	fmt.Printf("TTL: %s\n", formatDuration(ttl))
	fmt.Printf("Expires: %s\n", time.Now().Add(ttl).Format("2006-01-02 15:04:05"))

	return nil
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}
