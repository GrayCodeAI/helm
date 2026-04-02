package mistake

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Pattern represents a recurring mistake pattern
type Pattern struct {
	Type         Type
	Description  string
	FilePattern  string
	Count        int
	LastOccurred string
	Suggestion   string
}

// PatternDetector analyzes mistakes to find recurring patterns
type PatternDetector struct {
	journal *Journal
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector(journal *Journal) *PatternDetector {
	return &PatternDetector{journal: journal}
}

// Detect finds recurring mistake patterns
func (pd *PatternDetector) Detect(ctx context.Context, minOccurrences int) ([]Pattern, error) {
	if minOccurrences < 2 {
		minOccurrences = 2
	}

	// Get all mistake stats
	stats, err := pd.journal.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	var patterns []Pattern

	for mistakeType, count := range stats {
		if count < int64(minOccurrences) {
			continue
		}

		// Get mistakes of this type
		mistakes, err := pd.journal.ListByType(ctx, mistakeType)
		if err != nil {
			continue
		}

		// Group by file path patterns
		byFile := make(map[string][]*Entry)
		for _, m := range mistakes {
			filePattern := extractFilePattern(m.FilePath)
			byFile[filePattern] = append(byFile[filePattern], m)
		}

		// Create patterns for files with multiple occurrences
		for filePattern, entries := range byFile {
			if len(entries) < minOccurrences {
				continue
			}

			// Find common descriptions
			desc := findCommonDescription(entries)

			pattern := Pattern{
				Type:         mistakeType,
				Description:  desc,
				FilePattern:  filePattern,
				Count:        len(entries),
				LastOccurred: entries[0].CreatedAt.Format("2006-01-02"),
				Suggestion:   generateSuggestion(mistakeType, desc),
			}
			patterns = append(patterns, pattern)
		}
	}

	// Sort by count descending
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	return patterns, nil
}

// DetectForSession finds patterns specific to a session
func (pd *PatternDetector) DetectForSession(ctx context.Context, sessionID string) ([]Pattern, error) {
	mistakes, err := pd.journal.List(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if len(mistakes) < 2 {
		return nil, nil
	}

	// Group by type
	byType := make(map[Type][]*Entry)
	for _, m := range mistakes {
		byType[m.Type] = append(byType[m.Type], m)
	}

	var patterns []Pattern
	for mistakeType, entries := range byType {
		if len(entries) < 2 {
			continue
		}

		desc := findCommonDescription(entries)
		pattern := Pattern{
			Type:        mistakeType,
			Description: desc,
			Count:       len(entries),
			Suggestion:  generateSuggestion(mistakeType, desc),
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

func extractFilePattern(path string) string {
	if path == "" {
		return "unknown"
	}

	// Extract file extension
	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		ext := parts[len(parts)-1]
		return "*." + ext
	}

	// Extract directory pattern
	parts = strings.Split(path, "/")
	if len(parts) > 2 {
		return parts[len(parts)-2] + "/"
	}

	return path
}

func findCommonDescription(entries []*Entry) string {
	if len(entries) == 0 {
		return ""
	}

	// Use the most common description
	descCount := make(map[string]int)
	for _, e := range entries {
		descCount[e.Description]++
	}

	var maxCount int
	var mostCommon string
	for desc, count := range descCount {
		if count > maxCount {
			maxCount = count
			mostCommon = desc
		}
	}

	return mostCommon
}

func generateSuggestion(t Type, description string) string {
	switch t {
	case TypeRejectedDiff:
		return "Consider reviewing requirements more carefully before making changes"
	case TypeTestFailure:
		return "Run tests locally before submitting changes"
	case TypeLintError:
		return "Run linter before committing: `make lint` or equivalent"
	case TypeCompileError:
		return "Ensure code compiles before running tests"
	case TypeTimeout:
		return "Break task into smaller chunks to avoid timeouts"
	case TypeLoopDetected:
		return "Add explicit termination conditions to loops"
	case TypeWrongFile:
		return "Double-check file paths before making changes"
	case TypeRuntimeError:
		return "Add error handling and validation"
	case TypeSecurityIssue:
		return "Review security best practices for this type of change"
	default:
		return "Review similar past mistakes before proceeding"
	}
}

// Rule represents a correction rule derived from mistakes
type Rule struct {
	ID         string
	Pattern    string
	Correction string
	Confidence float64
	AppliesTo  []string // file patterns
}

// GenerateRules creates correction rules from patterns
func (pd *PatternDetector) GenerateRules(ctx context.Context, patterns []Pattern) []Rule {
	var rules []Rule

	for _, p := range patterns {
		if p.Count < 3 {
			continue // Need at least 3 occurrences for a rule
		}

		rule := Rule{
			ID:         fmt.Sprintf("rule-%s-%s", p.Type, p.FilePattern),
			Pattern:    p.Description,
			Correction: p.Suggestion,
			Confidence: float64(p.Count) / 10.0, // Simple confidence calculation
			AppliesTo:  []string{p.FilePattern},
		}

		if rule.Confidence > 1.0 {
			rule.Confidence = 1.0
		}

		rules = append(rules, rule)
	}

	return rules
}
