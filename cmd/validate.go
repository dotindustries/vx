package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/config"
)

func init() {
	rootCmd.AddCommand(validateCmd)
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate all vx.toml configuration files",
	Long: `Checks the root vx.toml and all referenced workspace configs for
structural validity. Reports errors for missing fields, invalid values,
and workspace paths that don't exist on disk.`,
	Args: cobra.NoArgs,
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	cfg, rootDir, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading root config: %w", err)
	}

	if err := config.ValidateWithRoot(cfg, rootDir); err != nil {
		return fmt.Errorf("root vx.toml: %w", err)
	}

	log.Debug().Str("root", rootDir).Msg("root config valid")
	fmt.Println("root vx.toml: valid")

	errors := 0
	for _, wsRelPath := range cfg.Workspaces {
		wsPath := filepath.Join(rootDir, wsRelPath)

		wsCfg, err := config.LoadWorkspaceConfig(wsPath)
		if err != nil {
			fmt.Printf("%s: ERROR - %s\n", wsRelPath, err)
			errors++
			continue
		}

		if err := config.ValidateWorkspace(wsCfg); err != nil {
			fmt.Printf("%s: ERROR - %s\n", wsRelPath, err)
			errors++
			continue
		}

		fmt.Printf("%s: valid\n", wsRelPath)
	}

	if errors > 0 {
		return fmt.Errorf("%d workspace config(s) have errors", errors)
	}

	fmt.Printf("\nAll %d config files are valid.\n", 1+len(cfg.Workspaces))

	return nil
}
