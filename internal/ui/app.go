// Package ui provides the TUI application using Bubbletea.
package ui

import (
	"fmt"
	"os"

	"charm.land/bubbletea/v2"
	"github.com/yourname/helm/internal/db"
)

// App is the main Bubbletea application.
type App struct {
	queries *db.DB
	width   int
	height  int

	// Screens
	dashboard *DashboardScreen
	sessions  *SessionScreen
	costs     *CostScreen
	memory    *MemoryScreen
	prompts   *PromptScreen

	// Current screen
	current Screen
}

// Screen is the interface for all screens.
type Screen interface {
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View() string
}

// NewApp creates a new TUI app.
func NewApp(queries *db.DB) *App {
	app := &App{
		queries: queries,
	}

	// Initialize screens
	app.dashboard = NewDashboardScreen(queries)
	app.current = app.dashboard

	return app
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	if a.current != nil {
		return a.current.Init()
	}
	return nil
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "1", "d":
			a.current = a.dashboard
			return a, a.current.Init()
		case "2", "s":
			if a.sessions == nil {
				a.sessions = NewSessionScreen(a.queries)
			}
			a.current = a.sessions
			return a, a.current.Init()
		case "3", "c":
			if a.costs == nil {
				a.costs = NewCostScreen(a.queries)
			}
			a.current = a.costs
			return a, a.current.Init()
		case "4", "m":
			if a.memory == nil {
				a.memory = NewMemoryScreen(a.queries)
			}
			a.current = a.memory
			return a, a.current.Init()
		case "5", "p":
			if a.prompts == nil {
				a.prompts = NewPromptScreen(a.queries)
			}
			a.current = a.prompts
			return a, a.current.Init()
		}
	}

	// Update current screen
	cmd := a.current.Update(msg)
	return a, cmd
}

// View implements tea.Model.
func (a *App) View() tea.View {
	if a.current == nil {
		return tea.NewView("Loading...")
	}
	return tea.NewView(a.current.View())
}

// Run starts the TUI application.
func Run(queries *db.DB) error {
	app := NewApp(queries)
	p := tea.NewProgram(app)
	_, err := p.Run()
	return err
}

// Helper functions
func getProject() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return wd
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}

func formatTokens(tokens int64) string {
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fk", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}

// Ensure screens implement the interface
var _ Screen = (*DashboardScreen)(nil)
