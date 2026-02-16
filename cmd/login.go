package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/token"
	"go.dot.industries/vx/internal/vault"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Vault via OIDC and start the token daemon",
	Long: `Opens a browser for OIDC authentication with Vault. On success the
token is saved to ~/.vx/token and the background renewal daemon is started.`,
	Args: cobra.NoArgs,
	RunE: runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	addr := cfg.Vault.Address
	if flagVaultAddr != "" {
		addr = flagVaultAddr
	}

	client, err := newClientForAuth(addr, cfg.Vault.BasePath, "oidc")
	if err != nil {
		return fmt.Errorf("creating vault client: %w", err)
	}

	log.Info().Msg("opening browser for OIDC authentication...")

	if err := vault.OIDCAuth(client, cfg.Vault.AuthRole); err != nil {
		return fmt.Errorf("OIDC authentication failed: %w", err)
	}

	if err := token.WriteToken(client.Token()); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	log.Info().Msg("authenticated successfully")

	if flagNoDaemon {
		log.Debug().Msg("skipping daemon start (--no-daemon)")
		return nil
	}

	startDaemonBackground()

	return nil
}
