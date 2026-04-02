// Package ui provides the TUI application using Bubbletea.
package ui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/yourname/helm/internal/analytics"
	"github.com/yourname/helm/internal/db"
)

// AnalyticsScreen displays analytics and ROI data
type AnalyticsScreen struct {
	queries     *db.DB
	width       int
	height      int
	roiData     []analytics.ModelROI
	wasteReport *analytics.WasteReport
	trendData   []analytics.DataPoint
	hotspots    []analytics.Hotspot
	activeTab   int // 0=ROI, 1=Waste, 2=Trends, 3=Hotspots
}

// NewAnalyticsScreen creates a new analytics screen.
func NewAnalyticsScreen(queries *db.DB) *AnalyticsScreen {
	return &AnalyticsScreen{
		queries: queries,
	}
}

// Init implements Screen.
func (a *AnalyticsScreen) Init() tea.Cmd {
	return a.loadData()
}

// Update implements Screen.
func (a *AnalyticsScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case analyticsDataMsg:
		a.roiData = msg.roiData
		a.wasteReport = msg.wasteReport
		a.trendData = msg.trendData
		a.hotspots = msg.hotspots

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			a.activeTab = (a.activeTab + 1) % 4
		case "shift+tab":
			a.activeTab = (a.activeTab + 3) % 4
		}
	}

	return nil
}

// SetSize sets the screen size.
func (a *AnalyticsScreen) SetSize(width, height int) {
	a.width = width
	a.height = height
}

// View implements Screen.
func (a *AnalyticsScreen) View() string {
	if a.width == 0 {
		return "Loading analytics..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	tabStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(lipgloss.Color("#666"))

	activeTabStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Bold(true).
		Background(lipgloss.Color("#7D56F4")).
		Foreground(lipgloss.Color("#FFF"))

	// Tabs
	tabs := []string{"ROI", "Waste", "Trends", "Hotspots"}
	var tabViews []string
	for i, tab := range tabs {
		if i == a.activeTab {
			tabViews = append(tabViews, activeTabStyle.Render(tab))
		} else {
			tabViews = append(tabViews, tabStyle.Render(tab))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabViews...)

	// Content based on active tab
	var content string
	switch a.activeTab {
	case 0:
		content = a.renderROI()
	case 1:
		content = a.renderWaste()
	case 2:
		content = a.renderTrends()
	case 3:
		content = a.renderHotspots()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("HELM Analytics"),
		tabBar,
		"",
		content,
	)
}

func (a *AnalyticsScreen) renderROI() string {
	if len(a.roiData) == 0 {
		return "No ROI data available"
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(a.width - 4)

	var rows []string
	rows = append(rows, fmt.Sprintf("%-20s %-10s %-10s %-12s %-10s", "Model", "Tasks", "Success %", "Avg Cost", "ROI"))
	rows = append(rows, string(make([]byte, 60, 60)))

	for _, roi := range a.roiData {
		row := fmt.Sprintf("%-20s %-10d %-10.1f $%-11.4f %-10.2f",
			roi.Model,
			roi.TotalTasks,
			roi.SuccessRate*100,
			roi.AvgCostPerTask,
			roi.ROI,
		)
		rows = append(rows, row)
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (a *AnalyticsScreen) renderWaste() string {
	if a.wasteReport == nil {
		return "No waste data available"
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(a.width - 4)

	var rows []string
	rows = append(rows, fmt.Sprintf("Total Spend: $%.2f", a.wasteReport.TotalSpend))
	rows = append(rows, fmt.Sprintf("Total Waste: $%.2f (%.1f%%)", a.wasteReport.TotalWaste, a.wasteReport.WastePercent))
	rows = append(rows, fmt.Sprintf("Trend: %s", a.wasteReport.Trend))
	rows = append(rows, "")
	rows = append(rows, "Waste Categories:")

	for _, cat := range a.wasteReport.WasteCategories {
		rows = append(rows, fmt.Sprintf("  - %s: %d incidents, $%.2f", cat.Category, cat.Count, cat.Cost))
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (a *AnalyticsScreen) renderTrends() string {
	if len(a.trendData) == 0 {
		return "No trend data available"
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(a.width - 4)

	var rows []string
	rows = append(rows, "Cost Trend (Last 30 days):")
	rows = append(rows, "")

	for _, point := range a.trendData {
		rows = append(rows, fmt.Sprintf("  %s: $%.2f", point.Date.Format("2006-01-02"), point.Value))
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (a *AnalyticsScreen) renderHotspots() string {
	if len(a.hotspots) == 0 {
		return "No hotspot data available"
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(a.width - 4)

	var rows []string
	rows = append(rows, fmt.Sprintf("%-40s %-10s %-10s %-10s", "File", "Changes", "Complexity", "Risk"))
	rows = append(rows, string(make([]byte, 70, 70)))

	for _, h := range a.hotspots[:min(len(a.hotspots), 10)] {
		row := fmt.Sprintf("%-40s %-10d %-10d %-10.1f",
			h.FilePath,
			h.ChangeCount,
			h.Complexity,
			h.RiskScore,
		)
		rows = append(rows, row)
	}

	return style.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (a *AnalyticsScreen) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Load ROI data
		roiDashboard := analytics.NewROIDashboard(a.queries)
		roiData, _ := roiDashboard.GetModelROI(ctx)

		// Load waste data
		wasteDetector := analytics.NewWasteDetector(a.queries)
		wasteReport, _ := wasteDetector.DetectWaste(ctx, "default", time.Now().AddDate(0, 0, -30), time.Now())

		// Load trend data
		trendAnalyzer := analytics.NewTrendAnalyzer(a.queries)
		trendData, _ := trendAnalyzer.GetCostTrend(ctx, "default", 30)

		return analyticsDataMsg{
			roiData:     roiData,
			wasteReport: wasteReport,
			trendData:   trendData,
		}
	}
}

// analyticsDataMsg carries analytics data
type analyticsDataMsg struct {
	roiData     []analytics.ModelROI
	wasteReport *analytics.WasteReport
	trendData   []analytics.DataPoint
	hotspots    []analytics.Hotspot
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
