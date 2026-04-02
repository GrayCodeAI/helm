// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/db"
)

// reportCmd generates various reports
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate reports",
	Long:  `Generate various reports: cost, usage, model performance, etc.`,
}

var reportCostCmd = &cobra.Command{
	Use:   "cost",
	Short: "Generate cost report",
	Long:  `Generate a detailed cost report for a time period.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		period, _ := cmd.Flags().GetString("period")
		format, _ := cmd.Flags().GetString("format")

		// Get cost data
		var report CostReport
		report.Project = project
		report.Period = period
		report.GeneratedAt = time.Now()

		switch period {
		case "today":
			data, err := application.DB.GetCostByProjectToday(ctx, project)
			if err != nil {
				return err
			}
			report.TotalCost = data.TotalCost
			report.TotalTokens = data.InputTokens + data.OutputTokens
		case "week":
			data, err := application.DB.GetCostByProjectWeek(ctx, project)
			if err != nil {
				return err
			}
			report.TotalCost = data.TotalCost
			report.TotalTokens = data.InputTokens + data.OutputTokens
		case "month":
			data, err := application.DB.GetCostByProjectMonth(ctx, project)
			if err != nil {
				return err
			}
			report.TotalCost = data.TotalCost
			report.TotalTokens = data.InputTokens + data.OutputTokens
		}

		// Output report
		switch format {
		case "json":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(report)
		case "csv":
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{"Period", "Total Cost", "Total Tokens"})
			w.Write([]string{report.Period, fmt.Sprintf("%.4f", report.TotalCost), fmt.Sprintf("%d", report.TotalTokens)})
			w.Flush()
			return nil
		default:
			fmt.Printf("Cost Report - %s\n", report.Period)
			fmt.Printf("Project: %s\n", report.Project)
			fmt.Printf("Generated: %s\n\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Total Cost:   $%.4f\n", report.TotalCost)
			fmt.Printf("Total Tokens: %d\n", report.TotalTokens)
			return nil
		}
	},
}

var reportSessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Generate session report",
	Long:  `Generate a report of recent sessions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		project, _ := os.Getwd()

		limit, _ := cmd.Flags().GetInt("limit")
		format, _ := cmd.Flags().GetString("format")

		sessions, err := application.DB.ListSessions(ctx, db.ListSessionsParams{
			Project: project,
			Limit:   int64(limit),
		})
		if err != nil {
			return err
		}

		// Build report
		report := SessionsReport{
			Project:     project,
			GeneratedAt: time.Now(),
			TotalCount:  int64(len(sessions)),
			Sessions:    make([]SessionSummary, 0, len(sessions)),
		}

		for _, s := range sessions {
			report.Sessions = append(report.Sessions, SessionSummary{
				ID:       s.ID,
				Status:   s.Status,
				Provider: s.Provider,
				Model:    s.Model,
				Cost:     s.Cost,
			})
		}

		// Output
		switch format {
		case "json":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(report)
		case "csv":
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{"ID", "Status", "Provider", "Model", "Cost"})
			for _, s := range report.Sessions {
				w.Write([]string{
					s.ID,
					s.Status,
					s.Provider,
					s.Model,
					fmt.Sprintf("%.4f", s.Cost),
				})
			}
			w.Flush()
			return nil
		default:
			fmt.Printf("Sessions Report\n")
			fmt.Printf("Project: %s\n", report.Project)
			fmt.Printf("Generated: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Total Sessions: %d\n\n", report.TotalCount)

			fmt.Printf("%-12s %-10s %-15s %-20s %s\n", "ID", "STATUS", "PROVIDER", "MODEL", "COST")
			fmt.Println("────────────────────────────────────────────────────────────────")
			for _, s := range report.Sessions {
				fmt.Printf("%-12s %-10s %-15s %-20s $%.4f\n",
					s.ID[:8], s.Status, s.Provider, truncate(s.Model, 20), s.Cost)
			}
			return nil
		}
	},
}

var reportModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Generate model performance report",
	Long:  `Generate a report of model performance metrics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		format, _ := cmd.Flags().GetString("format")

		performance, err := application.DB.ListModelPerformance(ctx)
		if err != nil {
			return err
		}

		switch format {
		case "json":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(performance)
		case "csv":
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{"Model", "Task Type", "Attempts", "Successes", "Total Cost", "Avg Tokens"})
			for _, p := range performance {
				w.Write([]string{
					p.Model,
					p.TaskType,
					fmt.Sprintf("%d", p.Attempts),
					fmt.Sprintf("%d", p.Successes),
					fmt.Sprintf("%.4f", p.TotalCost),
					fmt.Sprintf("%d", p.AvgTokens),
				})
			}
			w.Flush()
			return nil
		default:
			fmt.Println("Model Performance Report")
			fmt.Println("========================")

			fmt.Printf("%-25s %-15s %-10s %-10s %-12s %-10s\n",
				"MODEL", "TASK TYPE", "ATTEMPTS", "SUCCESSES", "COST", "AVG TOKENS")
			fmt.Println("────────────────────────────────────────────────────────────────────────────────")

			for _, p := range performance {
				successRate := float64(0)
				if p.Attempts > 0 {
					successRate = float64(p.Successes) / float64(p.Attempts) * 100
				}

				fmt.Printf("%-25s %-15s %-10d %-10d $%-11.4f %-10d (%.0f%%)\n",
					truncate(p.Model, 25),
					p.TaskType,
					p.Attempts,
					p.Successes,
					p.TotalCost/float64(p.Attempts),
					p.AvgTokens,
					successRate,
				)
			}
			return nil
		}
	},
}

// Report types

type CostReport struct {
	Project     string    `json:"project"`
	Period      string    `json:"period"`
	GeneratedAt time.Time `json:"generated_at"`
	TotalCost   float64   `json:"total_cost"`
	TotalTokens int64     `json:"total_tokens"`
}

type SessionsReport struct {
	Project     string           `json:"project"`
	GeneratedAt time.Time        `json:"generated_at"`
	TotalCount  int64            `json:"total_count"`
	Sessions    []SessionSummary `json:"sessions"`
}

type SessionSummary struct {
	ID       string  `json:"id"`
	Status   string  `json:"status"`
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Cost     float64 `json:"cost"`
}

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.AddCommand(reportCostCmd)
	reportCostCmd.Flags().StringP("period", "p", "today", "Report period (today, week, month)")
	reportCostCmd.Flags().StringP("format", "f", "text", "Output format (text, json, csv)")

	reportCmd.AddCommand(reportSessionsCmd)
	reportSessionsCmd.Flags().IntP("limit", "n", 50, "Number of sessions to include")
	reportSessionsCmd.Flags().StringP("format", "f", "text", "Output format (text, json, csv)")

	reportCmd.AddCommand(reportModelsCmd)
	reportModelsCmd.Flags().StringP("format", "f", "text", "Output format (text, json, csv)")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
