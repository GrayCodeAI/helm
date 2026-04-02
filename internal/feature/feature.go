// Package feature provides feature flags system.
package feature

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// Flag represents a feature flag
type Flag struct {
	Name        string                 `json:"name"`
	Enabled     bool                   `json:"enabled"`
	Description string                 `json:"description"`
	Rules       []Rule                 `json:"rules,omitempty"`
	Variants    map[string]interface{} `json:"variants,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Rule represents a flag rule
type Rule struct {
	Attribute string
	Operator  string
	Value     interface{}
	Enabled   bool
}

// Manager manages feature flags
type Manager struct {
	flags    map[string]*Flag
	mu       sync.RWMutex
	onChange func()
}

// NewManager creates a new feature flags manager
func NewManager() *Manager {
	return &Manager{
		flags: make(map[string]*Flag),
	}
}

// Register registers a feature flag
func (m *Manager) Register(flag Flag) {
	m.mu.Lock()
	defer m.mu.Unlock()

	flag.CreatedAt = time.Now()
	flag.UpdatedAt = time.Now()
	m.flags[flag.Name] = &flag
}

// IsEnabled checks if a flag is enabled
func (m *Manager) IsEnabled(name string, attrs map[string]interface{}) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, ok := m.flags[name]
	if !ok {
		return false
	}

	// Check rules
	if len(flag.Rules) > 0 {
		for _, rule := range flag.Rules {
			if evaluateRule(rule, attrs) {
				return rule.Enabled
			}
		}
	}

	return flag.Enabled
}

// GetVariant gets a variant value
func (m *Manager) GetVariant(name string, attrs map[string]interface{}) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, ok := m.flags[name]
	if !ok || !flag.Enabled {
		return nil
	}

	if flag.Variants == nil {
		return nil
	}

	// Check rules for variant
	if len(flag.Rules) > 0 {
		for _, rule := range flag.Rules {
			if evaluateRule(rule, attrs) {
				if val, ok := flag.Variants[rule.Attribute]; ok {
					return val
				}
			}
		}
	}

	// Return default variant
	if val, ok := flag.Variants["default"]; ok {
		return val
	}

	return nil
}

// Enable enables a flag
func (m *Manager) Enable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	flag, ok := m.flags[name]
	if !ok {
		return fmt.Errorf("flag not found: %s", name)
	}

	flag.Enabled = true
	flag.UpdatedAt = time.Now()

	if m.onChange != nil {
		m.onChange()
	}

	return nil
}

// Disable disables a flag
func (m *Manager) Disable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	flag, ok := m.flags[name]
	if !ok {
		return fmt.Errorf("flag not found: %s", name)
	}

	flag.Enabled = false
	flag.UpdatedAt = time.Now()

	if m.onChange != nil {
		m.onChange()
	}

	return nil
}

// List returns all flags
func (m *Manager) List() []*Flag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flags := make([]*Flag, 0, len(m.flags))
	for _, f := range m.flags {
		flags = append(flags, f)
	}
	return flags
}

// Export exports flags to JSON
func (m *Manager) Export() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return json.MarshalIndent(m.flags, "", "  ")
}

// Import imports flags from JSON
func (m *Manager) Import(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var flags map[string]*Flag
	if err := json.Unmarshal(data, &flags); err != nil {
		return err
	}

	m.flags = flags
	return nil
}

// LoadFromFile loads flags from file
func (m *Manager) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return m.Import(data)
}

// SaveToFile saves flags to file
func (m *Manager) SaveToFile(path string) error {
	data, err := m.Export()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// OnChange sets callback for flag changes
func (m *Manager) OnChange(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = fn
}

// evaluateRule evaluates a rule against attributes
func evaluateRule(rule Rule, attrs map[string]interface{}) bool {
	val, ok := attrs[rule.Attribute]
	if !ok {
		return false
	}

	switch rule.Operator {
	case "equals":
		return val == rule.Value
	case "not_equals":
		return val != rule.Value
	case "contains":
		if str, ok := val.(string); ok {
			if ruleStr, ok := rule.Value.(string); ok {
				return len(str) > 0 && len(ruleStr) > 0 && contains(str, ruleStr)
			}
		}
	case "gt":
		if f1, ok := val.(float64); ok {
			if f2, ok := rule.Value.(float64); ok {
				return f1 > f2
			}
		}
	case "lt":
		if f1, ok := val.(float64); ok {
			if f2, ok := rule.Value.(float64); ok {
				return f1 < f2
			}
		}
	}

	return false
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Middleware returns feature flag middleware
func (m *Manager) Middleware(requiredFlags ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, flag := range requiredFlags {
				if !m.IsEnabled(flag, nil) {
					http.Error(w, `{"error":"feature not available"}`, http.StatusNotImplemented)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Check checks if flag is enabled, panics if not found
func (m *Manager) Check(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, ok := m.flags[name]
	if !ok {
		panic(fmt.Sprintf("feature flag not found: %s", name))
	}

	return flag.Enabled
}

// DefaultFlags returns common default flags
func DefaultFlags() []Flag {
	return []Flag{
		{
			Name:        "web_dashboard",
			Enabled:     true,
			Description: "Enable web dashboard",
		},
		{
			Name:        "voice_notes",
			Enabled:     false,
			Description: "Enable voice notes feature",
		},
		{
			Name:        "team_sync",
			Enabled:     false,
			Description: "Enable team synchronization",
		},
		{
			Name:        "advanced_analytics",
			Enabled:     true,
			Description: "Enable advanced analytics",
		},
	}
}
