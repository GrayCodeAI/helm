// Package cost provides cost tracking and budget management.
package cost

import (
	"fmt"
	"strings"
)

// ReportEntry holds a single line in a cost report.
type ReportEntry struct {
	SessionID string
	Provider  string
	Model     string
	Cost      float64
	Tokens    int64
}

// Report aggregates cost data for display.
type Report struct {
	Entries     []ReportEntry
	TotalCost   float64
	TotalTokens int64
	Period      string
}

// FormatReport returns a human-readable cost report.
func FormatReport(r *Report) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Cost Report — %s\n", r.Period))
	sb.WriteString(strings.Repeat("─", 50) + "\n")

	for _, e := range r.Entries {
		sb.WriteString(fmt.Sprintf("  %-20s %6d tok  $%.2f\n",
			shortID(e.SessionID), e.Tokens, e.Cost))
	}

	sb.WriteString(strings.Repeat("─", 50) + "\n")
	sb.WriteString(fmt.Sprintf("  %-20s %6d tok  $%.2f\n",
		"Total", r.TotalTokens, r.TotalCost))

	return sb.String()
}

// FormatBudgetStatus returns a human-readable budget status.
func FormatBudgetStatus(s *BudgetStatus) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Budget — %s\n", s.Project))
	sb.WriteString(strings.Repeat("─", 40) + "\n")

	if s.DailyLimit > 0 {
		sb.WriteString(fmt.Sprintf("  Today:   $%.2f / $%.2f (%.0f%%)\n",
			s.DailyCost, s.DailyLimit, s.DailyPct*100))
	}
	if s.WeeklyLimit > 0 {
		sb.WriteString(fmt.Sprintf("  Week:    $%.2f / $%.2f (%.0f%%)\n",
			s.WeeklyCost, s.WeeklyLimit, s.WeeklyPct*100))
	}
	if s.MonthlyLimit > 0 {
		sb.WriteString(fmt.Sprintf("  Month:   $%.2f / $%.2f (%.0f%%)\n",
			s.MonthlyCost, s.MonthlyLimit, s.MonthlyPct*100))
	}

	if s.HardStop {
		sb.WriteString("\n  ⚠ HARD STOP: Budget exceeded\n")
	} else if s.Warning {
		sb.WriteString("\n  ⚠ WARNING: Approaching budget limit\n")
	}

	return sb.String()
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:8] + "..."
	}
	return id
}
