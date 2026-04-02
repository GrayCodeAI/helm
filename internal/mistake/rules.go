// Package mistake provides mistake tracking and learning capabilities
package mistake

import (
	"context"
	"fmt"
	"strings"
)

// RuleEngine manages correction rules derived from mistakes
type RuleEngine struct {
	rules []Rule
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: make([]Rule, 0),
	}
}

// AddRule adds a correction rule
func (re *RuleEngine) AddRule(rule Rule) {
	re.rules = append(re.rules, rule)
}

// FindApplicableRules finds rules applicable to a file
func (re *RuleEngine) FindApplicableRules(filePath string) []Rule {
	var applicable []Rule
	for _, rule := range re.rules {
		for _, pattern := range rule.AppliesTo {
			if matchesPattern(filePath, pattern) {
				applicable = append(applicable, rule)
				break
			}
		}
	}
	return applicable
}

// BuildCorrectionContext builds context string from applicable rules
func (re *RuleEngine) BuildCorrectionContext(filePath string) string {
	rules := re.FindApplicableRules(filePath)
	if len(rules) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "Based on past mistakes in similar files:")

	for _, rule := range rules {
		parts = append(parts, fmt.Sprintf("- %s (confidence: %.0f%%)",
			rule.Correction, rule.Confidence*100))
	}

	return strings.Join(parts, "\n")
}

// LoadRulesFromPatterns generates rules from detected patterns
func (re *RuleEngine) LoadRulesFromPatterns(ctx context.Context, detector *PatternDetector) error {
	patterns, err := detector.Detect(ctx, 3)
	if err != nil {
		return err
	}

	rules := detector.GenerateRules(ctx, patterns)
	for _, rule := range rules {
		re.AddRule(rule)
	}

	return nil
}

func matchesPattern(filePath, pattern string) bool {
	// Simple pattern matching
	if pattern == "" || pattern == "unknown" {
		return true
	}

	// Check extension match
	if strings.HasPrefix(pattern, "*.") {
		ext := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(filePath, "."+ext)
	}

	// Check directory match
	if strings.HasSuffix(pattern, "/") {
		return strings.Contains(filePath, pattern)
	}

	// Exact match
	return filePath == pattern
}

// RuleStore persists rules to storage
type RuleStore struct {
	// In a real implementation, this would use the database
	// For now, in-memory storage
	rules []Rule
}

// NewRuleStore creates a new rule store
func NewRuleStore() *RuleStore {
	return &RuleStore{
		rules: make([]Rule, 0),
	}
}

// Save saves a rule
func (rs *RuleStore) Save(rule Rule) error {
	rs.rules = append(rs.rules, rule)
	return nil
}

// LoadAll loads all rules
func (rs *RuleStore) LoadAll() ([]Rule, error) {
	return rs.rules, nil
}

// LoadByPattern loads rules matching a pattern
func (rs *RuleStore) LoadByPattern(filePattern string) ([]Rule, error) {
	var matching []Rule
	for _, rule := range rs.rules {
		for _, pattern := range rule.AppliesTo {
			if pattern == filePattern {
				matching = append(matching, rule)
				break
			}
		}
	}
	return matching, nil
}

// CorrectionSuggestion represents a specific correction suggestion
type CorrectionSuggestion struct {
	RuleID      string
	Description string
	Action      string
	Confidence  float64
}

// SuggestCorrections suggests corrections for a given context
func (re *RuleEngine) SuggestCorrections(filePath, context string) []CorrectionSuggestion {
	rules := re.FindApplicableRules(filePath)
	var suggestions []CorrectionSuggestion

	for _, rule := range rules {
		// Check if rule pattern matches context
		if strings.Contains(context, rule.Pattern) ||
			strings.Contains(strings.ToLower(context), strings.ToLower(rule.Pattern)) {
			suggestions = append(suggestions, CorrectionSuggestion{
				RuleID:      rule.ID,
				Description: rule.Pattern,
				Action:      rule.Correction,
				Confidence:  rule.Confidence,
			})
		}
	}

	return suggestions
}
