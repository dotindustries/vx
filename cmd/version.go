package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set by goreleaser at build time via ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("vx %s (%s) built %s\n", version, commit, date)
	},
}
