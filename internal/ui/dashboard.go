package ui

import (
	"context"
	"database/sql"
	"fmt"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/yourname/helm/internal/db"
)

// DashboardScreen is the main dashboard showing overview of sessions, costs, etc.
type DashboardScreen struct {
	queries   *db.DB
	width     int
	height    int
	sessions  []db.Session
	costToday db.GetCostByProjectTodayRow
	budget    db.Budget
}

// NewDashboardScreen creates a new dashboard screen.
func NewDashboardScreen(queries *db.DB) *DashboardScreen {
	return &DashboardScreen{
		queries: queries,
	}
}

// Init implements Screen.
func (d *DashboardScreen) Init() tea.Cmd {
	return d.loadData()
}

// Update implements Screen.
func (d *DashboardScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case dashboardDataMsg:
		d.sessions = msg.sessions
		d.costToday = msg.costToday
		d.budget = msg.budget

	case tea.KeyMsg:
		// Handle key presses specific to dashboard
	}

	return nil
}

// SetSize sets the screen size.
func (d *DashboardScreen) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View implements Screen.
func (d *DashboardScreen) View() string {
	if d.width == 0 {
		return "Loading..."
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(d.width/2 - 2)

	// Header
	header := titleStyle.Render("HELM — Personal Coding Agent Control Plane")

	// Left column: Sessions
	sessionsContent := d.renderSessions()
	sessionsBox := sectionStyle.Render(sessionsContent)

	// Right column: Cost & Budget
	costContent := d.renderCost()
	costBox := sectionStyle.Render(costContent)

	// Navigation help
	nav := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		Render("[1] Dashboard  [2] Sessions  [3] Cost  [4] Memory  [5] Prompts  [q] Quit")

	// Layout
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sessionsBox,
		costBox,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		"",
		nav,
	)
}

func (d *DashboardScreen) renderSessions() string {
	content := "📋 Recent Sessions\n\n"

	if len(d.sessions) == 0 {
		content += "No sessions yet.\n"
		content += "Run `helm run <prompt>` to start."
		return content
	}

	// Header
	content += fmt.Sprintf("%-8s %-10s %-12s %s\n", "ID", "Status", "Provider", "Cost")
	content += "────────────────────────────────\n"

	// Show up to 5 sessions
	limit := 5
	if len(d.sessions) < limit {
		limit = len(d.sessions)
	}

	for i := 0; i < limit; i++ {
		s := d.sessions[i]
		id := truncate(s.ID, 8)
		status := statusIcon(s.Status)
		content += fmt.Sprintf("%-8s %s %-10s %-12s %s\n",
			id,
			status,
			truncate(s.Status, 8),
			truncate(s.Provider, 10),
			formatCost(s.Cost),
		)
	}

	return content
}

func (d *DashboardScreen) renderCost() string {
	content := "💰 Cost Today\n\n"

	content += fmt.Sprintf("Total: %s\n", formatCost(d.costToday.TotalCost))
	content += fmt.Sprintf("Tokens: %s\n\n", formatTokens(d.costToday.InputTokens+d.costToday.OutputTokens))

	if d.budget.DailyLimit.Valid {
		pct := (d.costToday.TotalCost / d.budget.DailyLimit.Float64) * 100
		content += fmt.Sprintf("Budget: %.1f%% of $%.2f\n", pct, d.budget.DailyLimit.Float64)

		// Progress bar
		barWidth := 20
		filled := int((pct / 100) * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}

		bar := "["
		for i := 0; i < barWidth; i++ {
			if i < filled {
				bar += "█"
			} else {
				bar += "░"
			}
		}
		bar += "]"

		content += bar + "\n"
	} else {
		content += "No budget set.\n"
		content += "Run `helm cost budget <daily>`"
	}

	return content
}

func (d *DashboardScreen) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		project := getProject()

		// Load sessions
		sessions, _ := d.queries.ListSessions(ctx, db.ListSessionsParams{
			Project: project,
			Limit:   10,
		})

		// Load today's cost
		costToday, _ := d.queries.GetCostByProjectToday(ctx, project)

		// Load budget
		budget, _ := d.queries.GetBudget(ctx, project)

		return dashboardDataMsg{
			sessions:  sessions,
			costToday: costToday,
			budget:    budget,
		}
	}
}

// dashboardDataMsg is sent when dashboard data is loaded.
type dashboardDataMsg struct {
	sessions  []db.Session
	costToday db.GetCostByProjectTodayRow
	budget    db.Budget
}

func statusIcon(status string) string {
	switch status {
	case "running":
		return "●"
	case "done":
		return "✓"
	case "failed":
		return "✗"
	case "paused":
		return "⏸"
	default:
		return "○"
	}
}

// SessionScreen displays all sessions with filtering and search
type SessionScreen struct {
	queries  *db.DB
	width    int
	height   int
	sessions []db.Session
	filter   string
	selected int
}

func NewSessionScreen(q *db.DB) *SessionScreen {
	return &SessionScreen{queries: q, selected: -1}
}

func (s *SessionScreen) Init() tea.Cmd {
	return s.loadData()
}

func (s *SessionScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case sessionDataMsg:
		s.sessions = msg.sessions

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if s.selected < len(s.sessions)-1 {
				s.selected++
			}
		case "k", "up":
			if s.selected > 0 {
				s.selected--
			}
		case "/":
			// Could add search input
		}
	}
	return nil
}

func (s *SessionScreen) SetSize(width, height int) {
	s.width = width
	s.height = height
}

func (s *SessionScreen) View() string {
	if s.width == 0 {
		return "Loading sessions..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	header := titleStyle.Render("Sessions")

	content := fmt.Sprintf("%-10s %-8s %-12s %-10s %-8s %s\n", "ID", "Status", "Provider", "Model", "Cost", "Started")
	content += "────────────────────────────────────────────────────────────────────────\n"

	for i, sess := range s.sessions {
		prefix := "  "
		if i == s.selected {
			prefix = "▸ "
		}
		id := truncate(sess.ID, 10)
		status := statusIcon(sess.Status)
		model := truncate(sess.Model, 10)
		content += fmt.Sprintf("%s%-10s %s %-12s %-10s %-10s %s\n",
			prefix, id, status, sess.Provider, model, formatCost(sess.Cost), sess.StartedAt[:10])
	}

	if len(s.sessions) == 0 {
		content += "No sessions found.\nRun `helm run <prompt>` to start a session."
	}

	nav := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		Render("[j/k] Navigate  [/] Search  [enter] View  [esc] Back")

	return lipgloss.JoinVertical(lipgloss.Left, header, content, "", nav)
}

func nullFloat64(v sql.NullFloat64) float64 {
	if !v.Valid {
		return 0
	}
	return v.Float64
}

func (s *SessionScreen) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		sessions, _ := s.queries.ListSessions(ctx, db.ListSessionsParams{
			Project: getProject(),
			Limit:   50,
			Offset:  0,
		})
		return sessionDataMsg{sessions: sessions}
	}
}

type sessionDataMsg struct {
	sessions []db.Session
}

// CostScreen displays cost breakdown and budget management
type CostScreen struct {
	queries *db.DB
	width   int
	height  int
	records []db.CostRecord
	today   db.GetCostByProjectTodayRow
	week    db.GetCostByProjectWeekRow
	month   db.GetCostByProjectMonthRow
	budget  db.Budget
}

func NewCostScreen(q *db.DB) *CostScreen {
	return &CostScreen{queries: q}
}

func (c *CostScreen) Init() tea.Cmd {
	return c.loadData()
}

func (c *CostScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height

	case costDataMsg:
		c.records = msg.records
		c.today = msg.today
		c.week = msg.week
		c.month = msg.month
		c.budget = msg.budget
	}
	return nil
}

func (c *CostScreen) SetSize(width, height int) {
	c.width = width
	c.height = height
}

func (c *CostScreen) View() string {
	if c.width == 0 {
		return "Loading cost data..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	header := titleStyle.Render("Cost Tracking")

	// Summary boxes
	summary := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(c.width/4 - 2)

	todayBox := summary.Render(fmt.Sprintf("Today\n%s\n%s tokens", formatCost(c.today.TotalCost), formatTokens(c.today.InputTokens+c.today.OutputTokens)))
	weekBox := summary.Render(fmt.Sprintf("This Week\n%s\n%s tokens", formatCost(c.week.TotalCost), formatTokens(c.week.InputTokens+c.week.OutputTokens)))
	monthBox := summary.Render(fmt.Sprintf("This Month\n%s\n%s tokens", formatCost(c.month.TotalCost), formatTokens(c.month.InputTokens+c.month.OutputTokens)))

	budgetInfo := ""
	if c.budget.DailyLimit.Valid {
		pct := (c.today.TotalCost / c.budget.DailyLimit.Float64) * 100
		budgetInfo = fmt.Sprintf("Daily Budget: $%.2f (%.0f%% used)", c.budget.DailyLimit.Float64, pct)
	} else {
		budgetInfo = "No budget set"
	}

	summaryRow := lipgloss.JoinHorizontal(lipgloss.Top, todayBox, weekBox, monthBox, summary.Render(budgetInfo))

	// Records table
	content := fmt.Sprintf("\n%-10s %-12s %-10s %-10s %s\n", "Session", "Provider", "Model", "Cost", "Time")
	content += "────────────────────────────────────────────────────────────────────────\n"

	for _, r := range c.records {
		sessionID := truncate(r.SessionID, 10)
		content += fmt.Sprintf("%-10s %-12s %-10s %-10s %s\n",
			sessionID, r.Provider, truncate(r.Model, 10), formatCost(nullFloat64(r.TotalCost)), r.RecordedAt[:16])
	}

	nav := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		Render("[esc] Back")

	return lipgloss.JoinVertical(lipgloss.Left, header, summaryRow, content, "", nav)
}

func (c *CostScreen) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		project := getProject()

		records, _ := c.queries.ListCostRecords(ctx, db.ListCostRecordsParams{
			Project: project,
			Limit:   20,
		})
		today, _ := c.queries.GetCostByProjectToday(ctx, project)
		week, _ := c.queries.GetCostByProjectWeek(ctx, project)
		month, _ := c.queries.GetCostByProjectMonth(ctx, project)
		budget, _ := c.queries.GetBudget(ctx, project)

		return costDataMsg{
			records: records,
			today:   today,
			week:    week,
			month:   month,
			budget:  budget,
		}
	}
}

type costDataMsg struct {
	records []db.CostRecord
	today   db.GetCostByProjectTodayRow
	week    db.GetCostByProjectWeekRow
	month   db.GetCostByProjectMonthRow
	budget  db.Budget
}

// MemoryScreen displays project memory entries with search and management
type MemoryScreen struct {
	queries  *db.DB
	width    int
	height   int
	memories []db.Memory
	selected int
}

func NewMemoryScreen(q *db.DB) *MemoryScreen {
	return &MemoryScreen{queries: q, selected: -1}
}

func (m *MemoryScreen) Init() tea.Cmd {
	return m.loadData()
}

func (m *MemoryScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case memoryDataMsg:
		m.memories = msg.memories

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.memories)-1 {
				m.selected++
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		}
	}
	return nil
}

func (m *MemoryScreen) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *MemoryScreen) View() string {
	if m.width == 0 {
		return "Loading memory..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	header := titleStyle.Render(fmt.Sprintf("Project Memory (%d entries)", len(m.memories)))

	content := fmt.Sprintf("%-4s %-12s %-20s %-30s %s\n", "", "Type", "Key", "Value", "Confidence")
	content += "────────────────────────────────────────────────────────────────────────────────\n"

	for i, mem := range m.memories {
		prefix := "  "
		if i == m.selected {
			prefix = "▸ "
		}
		content += fmt.Sprintf("%s%-12s %-20s %-30s %.0f%%\n",
			prefix, mem.Type, truncate(mem.Key, 20), truncate(mem.Value, 30), mem.Confidence*100)
	}

	if len(m.memories) == 0 {
		content += "No memories yet.\nMemories are auto-learned from sessions or added manually."
	}

	nav := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		Render("[j/k] Navigate  [enter] View  [esc] Back")

	return lipgloss.JoinVertical(lipgloss.Left, header, content, "", nav)
}

func (m *MemoryScreen) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		memories, _ := m.queries.ListMemories(ctx, getProject())
		return memoryDataMsg{memories: memories}
	}
}

type memoryDataMsg struct {
	memories []db.Memory
}

// PromptScreen displays prompt library with search and execution
type PromptScreen struct {
	queries  *db.DB
	width    int
	height   int
	prompts  []db.Prompt
	selected int
}

func NewPromptScreen(q *db.DB) *PromptScreen {
	return &PromptScreen{queries: q, selected: -1}
}

func (p *PromptScreen) Init() tea.Cmd {
	return p.loadData()
}

func (p *PromptScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height

	case promptDataMsg:
		p.prompts = msg.prompts

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if p.selected < len(p.prompts)-1 {
				p.selected++
			}
		case "k", "up":
			if p.selected > 0 {
				p.selected--
			}
		}
	}
	return nil
}

func (p *PromptScreen) SetSize(width, height int) {
	p.width = width
	p.height = height
}

func (p *PromptScreen) View() string {
	if p.width == 0 {
		return "Loading prompts..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	header := titleStyle.Render(fmt.Sprintf("Prompt Library (%d prompts)", len(p.prompts)))

	content := fmt.Sprintf("%-4s %-20s %-30s %-10s %s\n", "", "Name", "Description", "Source", "Tags")
	content += "────────────────────────────────────────────────────────────────────────────────\n"

	for i, prompt := range p.prompts {
		prefix := "  "
		if i == p.selected {
			prefix = "▸ "
		}
		tags := ""
		if prompt.Tags.Valid {
			tags = prompt.Tags.String
		}
		desc := ""
		if prompt.Description.Valid {
			desc = prompt.Description.String
		}
		content += fmt.Sprintf("%s%-20s %-30s %-10s %s\n",
			prefix, prompt.Name, truncate(desc, 30), prompt.Source, tags)
	}

	if len(p.prompts) == 0 {
		content += "No prompts found.\nPrompts are auto-discovered from .helm/prompts/ and built-in."
	}

	nav := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		Render("[j/k] Navigate  [enter] Use  [esc] Back")

	return lipgloss.JoinVertical(lipgloss.Left, header, content, "", nav)
}

func (p *PromptScreen) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		prompts, _ := p.queries.ListPrompts(ctx)
		return promptDataMsg{prompts: prompts}
	}
}

type promptDataMsg struct {
	prompts []db.Prompt
}
