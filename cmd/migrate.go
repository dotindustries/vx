package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/migrate"
)

var flagMigrateWrite bool

func init() {
	migrateCmd.Flags().BoolVar(&flagMigrateWrite, "write", false, "write vx.toml files to disk (default: dry-run)")
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate [path-to-fnox.toml]",
	Short: "Convert fnox.toml configuration to vx.toml format",
	Long: `Reads an existing fnox.toml and generates equivalent vx.toml files.
By default runs in dry-run mode showing what would be generated.
Use --write to actually write the files to disk.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMigrate,
}

func runMigrate(cmd *cobra.Command, args []string) error {
	fnoxPath := "fnox.toml"
	if len(args) > 0 {
		fnoxPath = args[0]
	}

	absPath, err := filepath.Abs(fnoxPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	rootDir := filepath.Dir(absPath)

	log.Debug().Str("path", absPath).Msg("loading fnox config")

	fnoxCfg, err := migrate.LoadFnoxConfig(absPath)
	if err != nil {
		return fmt.Errorf("loading fnox config: %w", err)
	}

	result, err := migrate.Convert(fnoxCfg, rootDir)
	if err != nil {
		return fmt.Errorf("converting config: %w", err)
	}

	if !flagMigrateWrite {
		fmt.Println("# Dry run — use --write to create files")
		fmt.Println()
		fmt.Println("# vx.toml (root)")
		fmt.Println(result.RootConfig)

		for wsPath, content := range result.WorkspaceConfigs {
			fmt.Printf("# %s\n", wsPath)
			fmt.Println(content)
		}

		return nil
	}

	rootOutput := filepath.Join(rootDir, "vx.toml")
	if err := writeConfigFile(rootOutput, result.RootConfig); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", rootOutput)

	for wsPath, content := range result.WorkspaceConfigs {
		outPath := filepath.Join(rootDir, wsPath)
		if err := writeConfigFile(outPath, content); err != nil {
			return err
		}
		fmt.Printf("wrote %s\n", outPath)
	}

	return nil
}

func writeConfigFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}
