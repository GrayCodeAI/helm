// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/db"
)

// sessionCmd represents the session command.
var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage sessions",
	Long:  `View and manage agent sessions.`,
}

// sessionListCmd lists all sessions.
var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions for the current project",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		sessions, err := application.DB.ListSessions(ctx, db.ListSessionsParams{
			Project: project,
			Limit:   50,
			Offset:  0,
		})
		if err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No sessions found for this project.")
			return nil
		}

		fmt.Printf("%-12s %-15s %-15s %-10s %-10s %s\n",
			"ID", "Provider", "Model", "Status", "Cost", "Started")
		fmt.Println(string(repeatByte('-', 80)))

		for _, s := range sessions {
			id := s.ID
			if len(id) > 12 {
				id = id[:12]
			}
			started, _ := time.Parse(time.RFC3339, s.StartedAt)
			startedStr := started.Format("Jan 02 15:04")

			statusIcon := "○"
			switch s.Status {
			case "running":
				statusIcon = "●"
			case "done":
				statusIcon = "✓"
			case "failed":
				statusIcon = "✗"
			}

			fmt.Printf("%-12s %-15s %-15s %s %-10s $%.2f %s\n",
				id,
				s.Provider,
				s.Model,
				statusIcon,
				s.Status,
				s.Cost,
				startedStr,
			)
		}

		return nil
	},
}

// sessionShowCmd shows details of a session.
var sessionShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show details of a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		id := args[0]

		s, err := application.DB.GetSession(ctx, id)
		if err != nil {
			return fmt.Errorf("session not found: %w", err)
		}

		fmt.Printf("Session: %s\n", s.ID)
		fmt.Printf("Provider: %s\n", s.Provider)
		fmt.Printf("Model: %s\n", s.Model)
		fmt.Printf("Status: %s\n", s.Status)
		fmt.Printf("Cost: $%.2f\n", s.Cost)
		fmt.Printf("Tokens: %d input, %d output\n", s.InputTokens, s.OutputTokens)

		if s.Prompt.Valid {
			fmt.Printf("Prompt: %.100s...\n", s.Prompt.String)
		}

		if s.Summary.Valid {
			fmt.Printf("Summary: %s\n", s.Summary.String)
		}

		return nil
	},
}

// sessionStopCmd stops a running session.
var sessionStopCmd = &cobra.Command{
	Use:   "stop <id>",
	Short: "Stop a running session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		id := args[0]

		err := application.DB.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
			ID:      id,
			Status:  "stopped",
			EndedAt: db.ToNullString(time.Now().Format(time.RFC3339)),
		})
		if err != nil {
			return fmt.Errorf("stop session: %w", err)
		}

		fmt.Printf("✓ Stopped session: %s\n", id)
		return nil
	},
}

func repeatByte(b byte, count int) []byte {
	result := make([]byte, count)
	for i := range result {
		result[i] = b
	}
	return result
}

func init() {
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)
	sessionCmd.AddCommand(sessionStopCmd)

	rootCmd.AddCommand(sessionCmd)
}
