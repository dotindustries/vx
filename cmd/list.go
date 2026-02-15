package cmd

import (
	"fmt"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/resolver"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List secrets that would be resolved for the current context",
	Long: `Shows all secret mappings for the current environment and workspace
without actually fetching them from Vault. Useful for debugging configuration.`,
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

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
