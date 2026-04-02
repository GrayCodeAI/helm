// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/db"
)

// costCmd represents the cost command.
var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "View cost tracking and reports",
	Long:  `Track and analyze AI agent costs across sessions and projects.`,
}

// costListCmd shows cost for today.
var costListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show cost breakdown for today",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		// Get today's costs
		records, err := application.DB.ListCostRecords(ctx, db.ListCostRecordsParams{
			Project: project,
			Limit:   100,
		})
		if err != nil {
			return fmt.Errorf("list costs: %w", err)
		}

		var total float64
		var tokens int64

		fmt.Println("Cost Report:")
		fmt.Printf("%-20s %-15s %-10s %-10s\n", "Session", "Provider", "Tokens", "Cost")
		fmt.Println(strings.Repeat("-", 60))

		for _, r := range records {
			if r.TotalCost.Valid {
				total += r.TotalCost.Float64
			}
			tokens += r.InputTokens.Int64 + r.OutputTokens.Int64
			fmt.Printf("%-20s %-15s %-10d $%.2f\n",
				r.SessionID[:min(20, len(r.SessionID))],
				r.Provider,
				r.InputTokens.Int64+r.OutputTokens.Int64,
				r.TotalCost.Float64,
			)
		}

		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("%-20s %-15s %-10d $%.2f\n", "TOTAL", "", tokens, total)

		// Show budget status
		budget, err := application.DB.GetBudget(ctx, project)
		if err == nil && budget.DailyLimit.Valid {
			pct := (total / budget.DailyLimit.Float64) * 100
			fmt.Printf("\nBudget: $%.2f / $%.2f (%.1f%%)\n", total, budget.DailyLimit.Float64, pct)
		}

		return nil
	},
}

// costSetBudgetCmd sets budget limits.
var costSetBudgetCmd = &cobra.Command{
	Use:   "budget <daily> [weekly] [monthly]",
	Short: "Set budget limits",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		var daily, weekly, monthly float64
		fmt.Sscanf(args[0], "%f", &daily)
		if len(args) > 1 {
			fmt.Sscanf(args[1], "%f", &weekly)
		} else {
			weekly = daily * 7
		}
		if len(args) > 2 {
			fmt.Sscanf(args[2], "%f", &monthly)
		} else {
			monthly = daily * 30
		}

		err := application.DB.UpsertBudget(ctx, db.UpsertBudgetParams{
			Project:      project,
			DailyLimit:   db.ToNullFloat64(daily),
			WeeklyLimit:  db.ToNullFloat64(weekly),
			MonthlyLimit: db.ToNullFloat64(monthly),
			WarningPct:   0.8,
		})
		if err != nil {
			return fmt.Errorf("set budget: %w", err)
		}

		fmt.Printf("✓ Budget set: daily=$%.2f, weekly=$%.2f, monthly=$%.2f\n", daily, weekly, monthly)
		return nil
	},
}


func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	costCmd.AddCommand(costListCmd)
	costCmd.AddCommand(costSetBudgetCmd)

	rootCmd.AddCommand(costCmd)
}
