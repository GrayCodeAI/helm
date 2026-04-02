// Package cost provides cost tracking and budget management
package cost

import (
	"fmt"
	"time"
)

// AlertType represents the type of budget alert
type AlertType string

const (
	AlertTypeWarning  AlertType = "warning"
	AlertTypeCritical AlertType = "critical"
	AlertTypeExceeded AlertType = "exceeded"
)

// Alert represents a budget alert
type Alert struct {
	ID           string
	Type         AlertType
	Project      string
	Message      string
	CurrentCost  float64
	Limit        float64
	Percentage   float64
	CreatedAt    time.Time
	Acknowledged bool
}

// AlertManager manages budget alerts
type AlertManager struct {
	alerts       []Alert
	handlers     []AlertHandler
	soundEnabled bool
}

// AlertHandler handles alert notifications
type AlertHandler interface {
	Handle(alert Alert) error
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts:       make([]Alert, 0),
		handlers:     make([]AlertHandler, 0),
		soundEnabled: true,
	}
}

// RegisterHandler registers an alert handler
func (am *AlertManager) RegisterHandler(handler AlertHandler) {
	am.handlers = append(am.handlers, handler)
}

// CheckBudget checks if budget thresholds are exceeded
func (am *AlertManager) CheckBudget(project string, currentCost, limit, warningPct float64) *Alert {
	if limit <= 0 {
		return nil
	}

	percentage := currentCost / limit

	// Check hard limit (100%)
	if percentage >= 1.0 {
		alert := Alert{
			ID:          fmt.Sprintf("exceeded-%s-%d", project, time.Now().Unix()),
			Type:        AlertTypeExceeded,
			Project:     project,
			Message:     fmt.Sprintf("Budget exceeded! $%.2f / $%.2f (%.1f%%)", currentCost, limit, percentage*100),
			CurrentCost: currentCost,
			Limit:       limit,
			Percentage:  percentage,
			CreatedAt:   time.Now(),
		}
		am.fireAlert(alert)
		return &alert
	}

	// Check critical threshold (90%)
	if percentage >= 0.9 {
		alert := Alert{
			ID:          fmt.Sprintf("critical-%s-%d", project, time.Now().Unix()),
			Type:        AlertTypeCritical,
			Project:     project,
			Message:     fmt.Sprintf("Budget critical! $%.2f / $%.2f (%.1f%%)", currentCost, limit, percentage*100),
			CurrentCost: currentCost,
			Limit:       limit,
			Percentage:  percentage,
			CreatedAt:   time.Now(),
		}
		am.fireAlert(alert)
		return &alert
	}

	// Check warning threshold
	if percentage >= warningPct {
		alert := Alert{
			ID:          fmt.Sprintf("warning-%s-%d", project, time.Now().Unix()),
			Type:        AlertTypeWarning,
			Project:     project,
			Message:     fmt.Sprintf("Budget warning! $%.2f / $%.2f (%.1f%%)", currentCost, limit, percentage*100),
			CurrentCost: currentCost,
			Limit:       limit,
			Percentage:  percentage,
			CreatedAt:   time.Now(),
		}
		am.fireAlert(alert)
		return &alert
	}

	return nil
}

// CheckDailyBudget checks daily budget
func (am *AlertManager) CheckDailyBudget(project string, dailyCost, limit, warningPct float64) *Alert {
	return am.CheckBudget(project+"-daily", dailyCost, limit, warningPct)
}

// CheckWeeklyBudget checks weekly budget
func (am *AlertManager) CheckWeeklyBudget(project string, weeklyCost, limit, warningPct float64) *Alert {
	return am.CheckBudget(project+"-weekly", weeklyCost, limit, warningPct)
}

// CheckMonthlyBudget checks monthly budget
func (am *AlertManager) CheckMonthlyBudget(project string, monthlyCost, limit, warningPct float64) *Alert {
	return am.CheckBudget(project+"-monthly", monthlyCost, limit, warningPct)
}

// fireAlert fires an alert to all handlers
func (am *AlertManager) fireAlert(alert Alert) {
	am.alerts = append(am.alerts, alert)

	for _, handler := range am.handlers {
		go handler.Handle(alert)
	}

	if am.soundEnabled && (alert.Type == AlertTypeCritical || alert.Type == AlertTypeExceeded) {
		am.playAlertSound()
	}
}

// playAlertSound plays an alert sound (terminal bell)
func (am *AlertManager) playAlertSound() {
	fmt.Print("\a") // Terminal bell
}

// GetAlerts returns all alerts
func (am *AlertManager) GetAlerts() []Alert {
	return am.alerts
}

// GetUnacknowledgedAlerts returns unacknowledged alerts
func (am *AlertManager) GetUnacknowledgedAlerts() []Alert {
	var unacknowledged []Alert
	for _, alert := range am.alerts {
		if !alert.Acknowledged {
			unacknowledged = append(unacknowledged, alert)
		}
	}
	return unacknowledged
}

// AcknowledgeAlert acknowledges an alert
func (am *AlertManager) AcknowledgeAlert(alertID string) bool {
	for i := range am.alerts {
		if am.alerts[i].ID == alertID {
			am.alerts[i].Acknowledged = true
			return true
		}
	}
	return false
}

// EnableSound enables alert sounds
func (am *AlertManager) EnableSound() {
	am.soundEnabled = true
}

// DisableSound disables alert sounds
func (am *AlertManager) DisableSound() {
	am.soundEnabled = false
}

// ConsoleHandler prints alerts to console
type ConsoleHandler struct{}

// Handle handles an alert by printing to console
func (h *ConsoleHandler) Handle(alert Alert) error {
	var prefix string
	switch alert.Type {
	case AlertTypeWarning:
		prefix = "⚠️  WARNING"
	case AlertTypeCritical:
		prefix = "🚨 CRITICAL"
	case AlertTypeExceeded:
		prefix = "❌ EXCEEDED"
	}

	fmt.Printf("\n%s: %s\n", prefix, alert.Message)
	return nil
}

// EnforcementEngine enforces budget limits
type EnforcementEngine struct {
	alertManager  *AlertManager
	actionOnLimit string // "pause", "stop", "notify"
}

// NewEnforcementEngine creates a new enforcement engine
func NewEnforcementEngine(alertManager *AlertManager, action string) *EnforcementEngine {
	return &EnforcementEngine{
		alertManager:  alertManager,
		actionOnLimit: action,
	}
}

// ShouldAllowSession checks if a new session should be allowed
func (ee *EnforcementEngine) ShouldAllowSession(project string, currentCost, limit float64) (bool, string) {
	if limit <= 0 {
		return true, ""
	}

	if currentCost >= limit {
		switch ee.actionOnLimit {
		case "stop":
			return false, fmt.Sprintf("Budget exceeded ($%.2f / $%.2f). New sessions blocked.", currentCost, limit)
		case "pause":
			return false, fmt.Sprintf("Budget exceeded ($%.2f / $%.2f). Sessions paused until next billing period.", currentCost, limit)
		default: // "notify"
			return true, fmt.Sprintf("Warning: Budget exceeded ($%.2f / $%.2f)", currentCost, limit)
		}
	}

	return true, ""
}
