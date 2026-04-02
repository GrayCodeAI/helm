// Package retention provides data retention policies and automatic cleanup.
package retention

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yourname/helm/internal/logger"
)

// Policy defines a data retention policy
type Policy struct {
	Name        string
	Table       string
	Column      string
	MaxAge      time.Duration
	MaxRows     int64
	Enabled     bool
	Description string
}

// Manager manages data retention
type Manager struct {
	db       *sql.DB
	logger   *logger.Logger
	policies []Policy
}

// NewManager creates a new retention manager
func NewManager(db *sql.DB, log *logger.Logger) *Manager {
	if log == nil {
		log = logger.GetDefault()
	}
	return &Manager{
		db:       db,
		logger:   log,
		policies: make([]Policy, 0),
	}
}

// AddPolicy adds a retention policy
func (m *Manager) AddPolicy(policy Policy) {
	m.policies = append(m.policies, policy)
	m.logger.Info("Added retention policy: %s (max age: %s, max rows: %d)",
		policy.Name, policy.MaxAge, policy.MaxRows)
}

// Enforce enforces all retention policies
func (m *Manager) Enforce(ctx context.Context) error {
	for _, policy := range m.policies {
		if !policy.Enabled {
			continue
		}

		m.logger.Info("Enforcing retention policy: %s", policy.Name)

		if err := m.enforcePolicy(ctx, policy); err != nil {
			m.logger.Error("Failed to enforce policy %s: %v", policy.Name, err)
		}
	}

	return nil
}

// enforcePolicy enforces a single retention policy
func (m *Manager) enforcePolicy(ctx context.Context, policy Policy) error {
	// Delete old records
	if policy.MaxAge > 0 {
		cutoff := time.Now().Add(-policy.MaxAge)
		query := fmt.Sprintf("DELETE FROM %s WHERE %s < ?", policy.Table, policy.Column)

		result, err := m.db.ExecContext(ctx, query, cutoff.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("delete old records: %w", err)
		}

		rows, _ := result.RowsAffected()
		if rows > 0 {
			m.logger.Info("Policy %s: deleted %d old records", policy.Name, rows)
		}
	}

	// Delete excess records
	if policy.MaxRows > 0 {
		query := fmt.Sprintf(`
			DELETE FROM %s WHERE rowid IN (
				SELECT rowid FROM %s 
				ORDER BY %s DESC 
				LIMIT -1 OFFSET ?
			)
		`, policy.Table, policy.Table, policy.Column)

		result, err := m.db.ExecContext(ctx, query, policy.MaxRows)
		if err != nil {
			return fmt.Errorf("delete excess records: %w", err)
		}

		rows, _ := result.RowsAffected()
		if rows > 0 {
			m.logger.Info("Policy %s: deleted %d excess records", policy.Name, rows)
		}
	}

	return nil
}

// ScheduleEnforcement schedules periodic enforcement
func (m *Manager) ScheduleEnforcement(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.Enforce(ctx); err != nil {
				m.logger.Error("Scheduled enforcement failed: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// GetDefaultPolicies returns default retention policies
func GetDefaultPolicies() []Policy {
	return []Policy{
		{
			Name:        "sessions_archive",
			Table:       "sessions",
			Column:      "started_at",
			MaxAge:      90 * 24 * time.Hour,
			MaxRows:     10000,
			Enabled:     true,
			Description: "Archive sessions older than 90 days",
		},
		{
			Name:        "cost_records_cleanup",
			Table:       "cost_records",
			Column:      "recorded_at",
			MaxAge:      365 * 24 * time.Hour,
			MaxRows:     100000,
			Enabled:     true,
			Description: "Clean up cost records older than 1 year",
		},
		{
			Name:        "memory_cleanup",
			Table:       "memories",
			Column:      "updated_at",
			MaxAge:      180 * 24 * time.Hour,
			MaxRows:     5000,
			Enabled:     true,
			Description: "Clean up stale memories",
		},
	}
}

// Stats returns retention statistics
func (m *Manager) Stats(ctx context.Context) map[string]map[string]int64 {
	stats := make(map[string]map[string]int64)

	for _, policy := range m.policies {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", policy.Table)
		m.db.QueryRowContext(ctx, query).Scan(&count)

		stats[policy.Name] = map[string]int64{
			"total_rows": count,
			"max_rows":   policy.MaxRows,
		}
	}

	return stats
}
