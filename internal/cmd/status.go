// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/db"
)

// statusCmd shows the current status.
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show HELM status for current project",
	Long:  `Display current project status: running sessions, today's cost, and active configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		fmt.Printf("HELM Status for: %s\n\n", project)

		// Show running sessions
		sessions, err := application.DB.ListSessions(ctx, db.ListSessionsParams{
			Project: project,
			Limit:   10,
		})
		if err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}

		var running int
		for _, s := range sessions {
			if s.Status == "running" {
				running++
			}
		}

		if running > 0 {
			fmt.Printf("🟢 Running Sessions: %d\n", running)
			for _, s := range sessions {
				if s.Status == "running" {
					fmt.Printf("   - %s (%s)\n", s.ID[:8], s.Provider)
				}
			}
		} else {
			fmt.Println("⚪ No running sessions")
		}
		fmt.Println()

		// Show today's cost
		costs, err := application.DB.ListCostRecords(ctx, db.ListCostRecordsParams{
			Project: project,
			Limit:   100,
		})
		if err != nil {
			return fmt.Errorf("list costs: %w", err)
		}

		var totalCost float64
		var totalTokens int64
		for _, c := range costs {
			if c.TotalCost.Valid {
				totalCost += c.TotalCost.Float64
			}
			totalTokens += c.InputTokens.Int64 + c.OutputTokens.Int64
		}

		fmt.Printf("💰 Today's Cost: $%.2f (%d tokens)\n", totalCost, totalTokens)

		// Show budget
		budget, err := application.DB.GetBudget(ctx, project)
		if err == nil && budget.DailyLimit.Valid {
			pct := (totalCost / budget.DailyLimit.Float64) * 100
			fmt.Printf("📊 Budget: %.1f%% of $%.2f daily limit\n", pct, budget.DailyLimit.Float64)
		}
		fmt.Println()

		// Show active provider
		fmt.Printf("⚙️  Active Provider: %s\n", application.Config.Router.FallbackChain[0])
		fmt.Printf("   Fallback Chain: %v\n", application.Config.Router.FallbackChain)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
