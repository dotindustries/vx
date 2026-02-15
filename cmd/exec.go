package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/config"
	vxexec "go.dot.industries/vx/internal/exec"
	"go.dot.industries/vx/internal/resolver"
	"go.dot.industries/vx/internal/token"
	"go.dot.industries/vx/internal/vault"
)

func init() {
	rootCmd.AddCommand(execCmd)
}

var execCmd = &cobra.Command{
	Use:   "exec -- <command> [args...]",
	Short: "Run a command with secrets injected as environment variables",
	Long: `Resolves secrets from Vault and executes the given command with them
injected as environment variables. Secrets are scoped to the detected or
specified workspace.`,
	DisableFlagParsing: false,
	Args:               cobra.MinimumNArgs(1),
	RunE:               runExec,
}

func runExec(cmd *cobra.Command, args []string) error {
	cfg, rootDir, err := loadConfig()
	if err != nil {
		return err
	}

	env := resolveEnv(cfg)
	log.Debug().Str("env", env).Msg("resolved environment")

	workspace, err := detectWorkspace(cfg, rootDir, args)
	if err != nil {
		return err
	}

	merged, err := mergeForWorkspace(cfg, rootDir, workspace, env)
	if err != nil {
		return err
	}

	vaultClient, err := authenticatedClient(cfg, env)
	if err != nil {
		return err
	}

	secrets, err := resolveSecrets(vaultClient, merged)
	if err != nil {
		return err
	}

	// Overlay defaults under secrets (secrets take precedence).
	envVars := make(map[string]string, len(merged.Defaults)+len(secrets))
	for k, v := range merged.Defaults {
		envVars[k] = v
	}
	for k, v := range secrets {
		envVars[k] = v
	}

	log.Info().
		Int("secrets", len(secrets)).
		Int("defaults", len(merged.Defaults)).
		Str("workspace", workspace).
		Msg("injecting environment")

	ctx := context.Background()
	if err := vxexec.Run(ctx, args, envVars); err != nil {
		os.Exit(vxexec.ExitCode(err))
	}

	return nil
}

// detectWorkspace determines the workspace using CLI flags, command args, or cwd.
func detectWorkspace(cfg *config.RootConfig, rootDir string, args []string) (string, error) {
	if flagWorkspace != "" {
		log.Debug().Str("workspace", flagWorkspace).Msg("using explicit workspace flag")
		return flagWorkspace, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	ws, err := config.DetectWorkspace(args, cwd, cfg.Workspaces)
	if err != nil {
		return "", fmt.Errorf("detecting workspace: %w", err)
	}

	if ws != "" {
		log.Debug().Str("workspace", ws).Msg("auto-detected workspace")
	} else {
		log.Debug().Msg("no workspace detected, loading all secrets")
	}

	return ws, nil
}

// mergeForWorkspace loads the workspace config (if any) and merges it with root.
func mergeForWorkspace(cfg *config.RootConfig, rootDir string, workspace string, env string) (*config.MergedConfig, error) {
	if workspace == "" {
		return mergeAllWorkspaces(cfg, rootDir, env)
	}

	wsPath, err := config.ResolveWorkspacePath(rootDir, workspace, cfg.Workspaces)
	if err != nil {
		return nil, fmt.Errorf("resolving workspace path: %w", err)
	}

	wsCfg, err := config.LoadWorkspaceConfig(wsPath)
	if err != nil {
		return nil, fmt.Errorf("loading workspace config: %w", err)
	}

	return config.Merge(cfg, wsCfg, env)
}

// mergeAllWorkspaces loads all workspace configs and merges them with root.
func mergeAllWorkspaces(cfg *config.RootConfig, rootDir string, env string) (*config.MergedConfig, error) {
	merged, err := config.Merge(cfg, nil, env)
	if err != nil {
		return nil, err
	}

	for _, wsRelPath := range cfg.Workspaces {
		wsPath := filepath.Join(rootDir, wsRelPath)

		wsCfg, err := config.LoadWorkspaceConfig(wsPath)
		if err != nil {
			log.Warn().Err(err).Str("path", wsRelPath).Msg("skipping workspace")
			continue
		}

		wsMerged, err := config.Merge(cfg, wsCfg, env)
		if err != nil {
			log.Warn().Err(err).Str("path", wsRelPath).Msg("skipping workspace merge")
			continue
		}

		for k, v := range wsMerged.Secrets {
			merged.Secrets[k] = v
		}
		for k, v := range wsMerged.Defaults {
			merged.Defaults[k] = v
		}
	}

	return merged, nil
}

// authenticatedClient creates a Vault client with a valid token.
func authenticatedClient(cfg *config.RootConfig, env string) (*vault.Client, error) {
	tok, err := token.ReadToken()
	if err != nil {
		log.Debug().Msg("no cached token, authenticating")
		return authenticateNew(cfg)
	}

	client, err := vault.NewClientWithToken(cfg.Vault.Address, cfg.Vault.BasePath, tok)
	if err != nil {
		return nil, fmt.Errorf("creating vault client: %w", err)
	}

	if !client.IsAuthenticated() {
		log.Debug().Msg("cached token expired, re-authenticating")
		return authenticateNew(cfg)
	}

	log.Debug().Msg("using cached vault token")
	return client, nil
}

// authenticateNew performs a fresh authentication against Vault.
func authenticateNew(cfg *config.RootConfig) (*vault.Client, error) {
	client, err := vault.NewClient(cfg.Vault.Address, cfg.Vault.BasePath)
	if err != nil {
		return nil, fmt.Errorf("creating vault client: %w", err)
	}

	switch cfg.Vault.AuthMethod {
	case "oidc":
		if err := vault.OIDCAuth(client, cfg.Vault.AuthRole); err != nil {
			return nil, fmt.Errorf("OIDC authentication: %w", err)
		}
	case "approle":
		roleID := os.Getenv("VX_ROLE_ID")
		secretID := os.Getenv("VX_SECRET_ID")
		if err := vault.AppRoleAuth(client, roleID, secretID); err != nil {
			return nil, fmt.Errorf("AppRole authentication: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", cfg.Vault.AuthMethod)
	}

	if err := token.WriteToken(client.Token()); err != nil {
		log.Warn().Err(err).Msg("failed to cache token")
	}

	return client, nil
}

// resolveSecrets uses the resolver to fetch all secrets from Vault concurrently.
func resolveSecrets(client *vault.Client, merged *config.MergedConfig) (map[string]string, error) {
	r := resolver.New(client, merged.Vault.BasePath)

	secrets, err := r.Resolve(merged.Secrets, merged.Environment)
	if err != nil {
		return nil, fmt.Errorf("resolving secrets: %w", err)
	}

	return secrets, nil
}
