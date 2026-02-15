package cmd

import (
	"fmt"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/config"
	"go.dot.industries/vx/internal/resolver"
)

var flagFormat string

func init() {
	listCmd.Flags().StringVar(&flagFormat, "format", "table", "output format: table, dotenv")
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List secrets that would be resolved for the current context",
	Long: `Shows all secret mappings for the current environment and workspace.

The default "table" format shows Vault paths without fetching values.
Use --format=dotenv to resolve secrets from Vault and output KEY=VALUE pairs
suitable for piping to a .env file:

  vx list --format=dotenv > .env.docker`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, rootDir, err := loadConfig()
	if err != nil {
		return err
	}

	env := resolveEnv(cfg)

	workspace, err := detectWorkspace(cfg, rootDir, []string{})
	if err != nil {
		return err
	}

	merged, err := mergeForWorkspace(cfg, rootDir, workspace, env)
	if err != nil {
		return err
	}

	log.Debug().
		Str("env", env).
		Str("workspace", workspace).
		Int("secrets", len(merged.Secrets)).
		Int("defaults", len(merged.Defaults)).
		Msg("resolved config")

	switch flagFormat {
	case "table":
		return printTable(merged, env, workspace)
	case "dotenv":
		return printDotenv(cfg, merged)
	default:
		return fmt.Errorf("unsupported format %q (use table or dotenv)", flagFormat)
	}
}

// printTable shows the human-readable mapping table (no Vault fetch).
func printTable(merged *config.MergedConfig, env string, workspace string) error {
	fmt.Printf("Environment: %s\n", env)
	if workspace != "" {
		fmt.Printf("Workspace:   %s\n", workspace)
	}
	fmt.Println()

	if len(merged.Secrets) > 0 {
		fmt.Printf("Secrets (%d):\n", len(merged.Secrets))

		names := sortedKeys(merged.Secrets)
		for _, name := range names {
			path := resolver.Interpolate(merged.Secrets[name], env)
			fmt.Printf("  %-35s -> %s\n", name, path)
		}
		fmt.Println()
	}

	if len(merged.Defaults) > 0 {
		fmt.Printf("Defaults (%d):\n", len(merged.Defaults))

		names := sortedKeys(merged.Defaults)
		for _, name := range names {
			fmt.Printf("  %-35s = %s\n", name, merged.Defaults[name])
		}
	}

	return nil
}

// printDotenv resolves secrets from Vault and outputs KEY=VALUE lines.
func printDotenv(cfg *config.RootConfig, merged *config.MergedConfig) error {
	vaultClient, err := authenticatedClient(cfg, merged.Environment)
	if err != nil {
		return err
	}

	secrets, err := resolveSecrets(vaultClient, merged)
	if err != nil {
		return err
	}

	// Merge: defaults first, secrets override.
	all := make(map[string]string, len(merged.Defaults)+len(secrets))
	for k, v := range merged.Defaults {
		all[k] = v
	}
	for k, v := range secrets {
		all[k] = v
	}

	names := sortedKeys(all)
	for _, name := range names {
		fmt.Printf("%s=%s\n", name, all[name])
	}

	return nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
