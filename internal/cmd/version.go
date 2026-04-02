// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/version"
)

// versionCmd shows version information.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("HELM %s (commit: %s, built: %s)\n",
			version.Version,
			version.Commit,
			version.Date,
		)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
