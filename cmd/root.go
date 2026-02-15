package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/config"
)

var (
	flagEnv       string
	flagWorkspace string
	flagConfigDir string
	flagNoDaemon  bool
	flagVerbose   bool
	flagAuth      string
	flagVaultAddr string
	flagRoleID    string
	flagSecretID  string
)

var rootCmd = &cobra.Command{
	Use:   "vx",
	Short: "Vault-backed secret manager for monorepos",
	Long: `vx resolves secrets from HashiCorp Vault and injects them as
environment variables into child processes. It supports workspace-scoped
secret loading, parallel Vault reads, and automatic token renewal.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagEnv, "env", "e", "", "environment to use (overrides config default)")
	rootCmd.PersistentFlags().StringVarP(&flagWorkspace, "workspace", "w", "", "workspace to scope secrets to")
	rootCmd.PersistentFlags().StringVar(&flagConfigDir, "config", "", "path to root vx.toml (auto-detected if omitted)")
	rootCmd.PersistentFlags().BoolVar(&flagNoDaemon, "no-daemon", false, "skip token daemon; authenticate inline")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVar(&flagAuth, "auth", "", "authentication method (oidc, approle); overrides config")
	rootCmd.PersistentFlags().StringVar(&flagVaultAddr, "vault-addr", "", "vault address; overrides config")
	rootCmd.PersistentFlags().StringVar(&flagRoleID, "role-id", "", "AppRole role ID (for --auth approle)")
	rootCmd.PersistentFlags().StringVar(&flagSecretID, "secret-id", "", "AppRole secret ID (for --auth approle)")

	cobra.OnInitialize(initLogger)
}

func initLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	level := zerolog.InfoLevel
	if flagVerbose {
		level = zerolog.DebugLevel
	}

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().Timestamp().Logger().Level(level)
}

// loadConfig finds and parses the root vx.toml and returns the root config,
// the directory it was found in, and optionally the resolved environment name.
func loadConfig() (*config.RootConfig, string, error) {
	configPath := flagConfigDir

	if configPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", fmt.Errorf("getting working directory: %w", err)
		}

		found, err := config.FindRootConfig(cwd)
		if err != nil {
			return nil, "", err
		}
		configPath = found
	}

	cfg, err := config.LoadRootConfig(configPath)
	if err != nil {
		return nil, "", err
	}

	rootDir := filepath.Dir(configPath)

	return cfg, rootDir, nil
}

// resolveEnv returns the environment to use, preferring the CLI flag over the
// config default.
func resolveEnv(cfg *config.RootConfig) string {
	if flagEnv != "" {
		return flagEnv
	}
	return cfg.Environments.Default
}
