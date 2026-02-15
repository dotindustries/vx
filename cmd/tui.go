package cmd

import (
	"github.com/spf13/cobra"

	"go.dot.industries/vx/internal/tui"
)

func init() {
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(browseCmd)
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive terminal UI for browsing and managing secrets",
	Long: `Opens an interactive dual-pane terminal dashboard for browsing
workspaces and secrets, resolving values from Vault on demand, and
managing secret mappings in vx.toml files.`,
	RunE: runTUI,
}

var browseCmd = &cobra.Command{
	Use:    "browse",
	Short:  "Alias for tui",
	Hidden: true,
	RunE:   runTUI,
}

func runTUI(_ *cobra.Command, _ []string) error {
	return tui.Run(flagConfigDir, flagVaultAddr, flagAuth, flagRoleID, flagSecretID)
}
