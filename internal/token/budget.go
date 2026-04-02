// Package token provides token budget management
package token

import (
	"fmt"
)

// Budget represents a token budget
type Budget struct {
	TotalTokens    int
	UsedTokens     int
	ReservedTokens int
	SessionID      string
}

// Remaining returns remaining tokens
func (b *Budget) Remaining() int {
	return b.TotalTokens - b.UsedTokens - b.ReservedTokens
}

// Use marks tokens as used
func (b *Budget) Use(tokens int) error {
	if tokens > b.Remaining() {
		return fmt.Errorf("token budget exceeded: need %d, have %d", tokens, b.Remaining())
	}
	b.UsedTokens += tokens
	return nil
}

// Reserve reserves tokens
func (b *Budget) Reserve(tokens int) bool {
	if tokens > b.Remaining() {
		return false
	}
	b.ReservedTokens += tokens
	return true
}

// Release releases reserved tokens
func (b *Budget) Release(tokens int) {
	b.ReservedTokens -= tokens
	if b.ReservedTokens < 0 {
		b.ReservedTokens = 0
	}
}

// PercentUsed returns percentage used
func (b *Budget) PercentUsed() float64 {
	if b.TotalTokens == 0 {
		return 0
	}
	return float64(b.UsedTokens) / float64(b.TotalTokens) * 100
}

// Manager manages token budgets
type Manager struct {
	budgets map[string]*Budget
}

// NewManager creates a budget manager
func NewManager() *Manager {
	return &Manager{
		budgets: make(map[string]*Budget),
	}
}

// CreateBudget creates a new budget
func (m *Manager) CreateBudget(sessionID string, totalTokens int) *Budget {
	budget := &Budget{
		TotalTokens: totalTokens,
		SessionID:   sessionID,
	}
	m.budgets[sessionID] = budget
	return budget
}

// GetBudget gets a budget
func (m *Manager) GetBudget(sessionID string) (*Budget, bool) {
	budget, ok := m.budgets[sessionID]
	return budget, ok
}

// CheckAndReserve checks and reserves tokens
func (m *Manager) CheckAndReserve(sessionID string, tokens int) (*Budget, error) {
	budget, ok := m.budgets[sessionID]
	if !ok {
		return nil, fmt.Errorf("no budget for session %s", sessionID)
	}

	if !budget.Reserve(tokens) {
		return budget, fmt.Errorf("insufficient tokens: need %d, have %d", tokens, budget.Remaining())
	}

	return budget, nil
}

// ReleaseTokens releases reserved tokens
func (m *Manager) ReleaseTokens(sessionID string, tokens int) {
	if budget, ok := m.budgets[sessionID]; ok {
		budget.Release(tokens)
	}
}

// UseTokens marks tokens as used
func (m *Manager) UseTokens(sessionID string, tokens int) error {
	if budget, ok := m.budgets[sessionID]; ok {
		return budget.Use(tokens)
	}
	return fmt.Errorf("no budget for session %s", sessionID)
}

// AlertLevel represents budget alert level
type AlertLevel int

const (
	AlertNone AlertLevel = iota
	AlertWarning
	AlertCritical
	AlertExceeded
)

// CheckAlert checks if budget needs alerting
func (m *Manager) CheckAlert(sessionID string) (AlertLevel, string) {
	budget, ok := m.budgets[sessionID]
	if !ok {
		return AlertNone, ""
	}

	percent := budget.PercentUsed()

	if percent >= 100 {
		return AlertExceeded, fmt.Sprintf("Token budget exceeded: %d/%d", budget.UsedTokens, budget.TotalTokens)
	}
	if percent >= 90 {
		return AlertCritical, fmt.Sprintf("Token budget critical: %.0f%% used", percent)
	}
	if percent >= 75 {
		return AlertWarning, fmt.Sprintf("Token budget warning: %.0f%% used", percent)
	}

	return AlertNone, ""
}

// BudgetConfig configures token budgets
type BudgetConfig struct {
	MaxInputTokens  int
	MaxOutputTokens int
	WarningPercent  float64
	HardStop        bool
}

// DefaultBudgetConfig returns default config
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		MaxInputTokens:  100000,
		MaxOutputTokens: 20000,
		WarningPercent:  0.8,
		HardStop:        true,
	}
}

// Enforcer enforces token budgets
type Enforcer struct {
	config  BudgetConfig
	manager *Manager
}

// NewEnforcer creates a budget enforcer
func NewEnforcer(config BudgetConfig, manager *Manager) *Enforcer {
	return &Enforcer{
		config:  config,
		manager: manager,
	}
}

// CheckRequest checks if request is within budget
func (e *Enforcer) CheckRequest(sessionID string, inputTokens int) (*Budget, error) {
	budget, err := e.manager.CheckAndReserve(sessionID, inputTokens)
	if err != nil {
		if e.config.HardStop {
			return budget, fmt.Errorf("hard stop: %w", err)
		}
		return budget, nil // Soft stop - allow but warn
	}
	return budget, nil
}

// CheckResponse checks response tokens
func (e *Enforcer) CheckResponse(sessionID string, outputTokens int) (*Budget, error) {
	if outputTokens > e.config.MaxOutputTokens {
		return nil, fmt.Errorf("output exceeds max: %d > %d", outputTokens, e.config.MaxOutputTokens)
	}

	budget, err := e.manager.CheckAndReserve(sessionID, outputTokens)
	if err != nil {
		if e.config.HardStop {
			return budget, fmt.Errorf("hard stop: %w", err)
		}
	}

	return budget, nil
}
