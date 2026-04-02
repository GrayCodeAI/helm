// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// diffCmd represents the diff command.
var diffCmd = &cobra.Command{
	Use:   "diff [session-id]",
	Short: "Review changes from a session",
	Long:  `Review and accept/reject changes made by an agent session.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Show git diff of current changes
			return showGitDiff()
		}
		return reviewSessionDiff(args[0])
	},
}

func showGitDiff() error {
	cmd := exec.Command("git", "diff", "--stat")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git diff: %w", err)
	}

	fmt.Println("\nUse `helm diff <session-id>` to review a specific session's changes.")
	return nil
}

func reviewSessionDiff(sessionID string) error {
	fmt.Printf("Reviewing changes for session: %s\n", sessionID)
	fmt.Println("Diff review TUI not yet implemented. Use `git diff` to see changes.")
	return nil
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
