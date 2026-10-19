package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/config"
	"go.dot.industries/vx/internal/resolver"
)

var flagFormat string

func init() {
	listCmd.Flags().StringVar(&flagFormat, "format", "table", "output format: table, dotenv")
	addEnvFileFlags(listCmd)
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List secrets that would be resolved for the current context",
	Long: `Shows all secret mappings for the current environment and workspace.

The default "table" format shows Vault paths without fetching values.
Use --format=dotenv to resolve secrets from Vault and output KEY=VALUE pairs
suitable for piping to a .env file:

  vx list --format=dotenv > .env.docker

Keys overridden by .env.local or --env-file are marked in the table and their
values take precedence in the dotenv output.`,
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

	layers, err := loadEnvFileLayers(cfg, rootDir)
	if err != nil {
		return err
	}

	switch flagFormat {
	case "table":
		_, overrideSources := flattenEnvFileLayers(layers)
		printTable(os.Stdout, merged, env, workspace, overrideSources)
		return nil
	case "dotenv":
		return printDotenv(cfg, merged, layers)
	default:
		return fmt.Errorf("unsupported format %q (use table or dotenv)", flagFormat)
	}
}

// printTable shows the human-readable mapping table (no Vault fetch). Env file
// values are never printed because they commonly hold credentials.
func printTable(w io.Writer, merged *config.MergedConfig, env string, workspace string, overrideSources map[string]string) {
	fmt.Fprintf(w, "Environment: %s\n", env)
	if workspace != "" {
		fmt.Fprintf(w, "Workspace:   %s\n", workspace)
	}
	fmt.Fprintln(w)

	if len(merged.Secrets) > 0 {
		fmt.Fprintf(w, "Secrets (%d):\n", len(merged.Secrets))

		names := sortedKeys(merged.Secrets)
		for _, name := range names {
			path := resolver.Interpolate(merged.Secrets[name], env)
			fmt.Fprintf(w, "  %-35s -> %s%s\n", name, path, overrideMarker(overrideSources, name))
		}
		fmt.Fprintln(w)
	}

	if len(merged.Defaults) > 0 {
		fmt.Fprintf(w, "Defaults (%d):\n", len(merged.Defaults))

		names := sortedKeys(merged.Defaults)
		for _, name := range names {
			fmt.Fprintf(w, "  %-35s = %s%s\n", name, merged.Defaults[name], overrideMarker(overrideSources, name))
		}
		fmt.Fprintln(w)
	}

	if len(overrideSources) > 0 {
		fmt.Fprintf(w, "Env file overrides (%d):\n", len(overrideSources))

		names := sortedKeys(overrideSources)
		for _, name := range names {
			fmt.Fprintf(w, "  %-35s <- %s\n", name, overrideSources[name])
		}
	}
}

func overrideMarker(overrideSources map[string]string, name string) string {
	source, ok := overrideSources[name]
	if !ok {
		return ""
	}
	return "  (overridden by " + source + ")"
}

// printDotenv resolves secrets from Vault and outputs KEY=VALUE lines.
func printDotenv(cfg *config.RootConfig, merged *config.MergedConfig, layers []envFileLayer) error {
	overrides, _ := flattenEnvFileLayers(layers)

	vaultClient, err := authenticatedClient(cfg, merged.Environment)
	if err != nil {
		return err
	}

	secrets, err := resolveSecrets(vaultClient, withoutOverriddenSecrets(merged, overrides))
	if err != nil {
		return err
	}

	logEnvFileLayers(layers)
	all := buildEnv(merged.Defaults, secrets, overrides)

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
