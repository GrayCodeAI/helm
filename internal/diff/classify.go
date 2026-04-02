package diff

import (
	"path/filepath"
	"strings"
)

// Classifier automatically classifies changes
type Classifier struct {
	// Patterns for incidental changes
	incidentalPatterns []string
	// File extensions that are typically incidental
	incidentalExtensions []string
}

// NewClassifier creates a new diff classifier
func NewClassifier() *Classifier {
	return &Classifier{
		incidentalPatterns: []string{
			"import ",
			"import(",
			"using ",
			"#include",
			"package ",
			"module ",
			"go.mod",
			"go.sum",
			"package-lock.json",
			"yarn.lock",
			"Cargo.lock",
			"vendor/",
			"node_modules/",
		},
		incidentalExtensions: []string{
			".mod", ".sum", ".lock", ".generated.go",
		},
	}
}

// Classify determines the change type for a file change
func (c *Classifier) Classify(change *FileChange, prompt string) ChangeType {
	// Check for suspicious patterns
	if c.isSuspicious(change, prompt) {
		return Suspicious
	}

	// Check for incidental patterns
	if c.isIncidental(change) {
		return Incidental
	}

	return Essential
}

// ClassifyMulti classifies multiple changes
func (c *Classifier) ClassifyMulti(changes []FileChange, prompt string) []FileChange {
	for i := range changes {
		changes[i].Classification = c.Classify(&changes[i], prompt)
	}
	return changes
}

func (c *Classifier) isIncidental(change *FileChange) bool {
	// Check file extensions
	ext := filepath.Ext(change.FilePath)
	for _, ie := range c.incidentalExtensions {
		if ext == ie {
			return true
		}
	}

	// Check if only imports/formatting changed
	diffLines := strings.Split(change.Diff, "\n")
	allIncidental := true
	hasChanges := false

	for _, line := range diffLines {
		if !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "-") {
			continue
		}
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}

		hasChanges = true
		content := line[1:] // Remove +/- prefix

		// Check if this line matches incidental patterns
		isIncidentalLine := false
		for _, pattern := range c.incidentalPatterns {
			if strings.Contains(content, pattern) {
				isIncidentalLine = true
				break
			}
		}

		// Check for formatting-only changes (whitespace)
		trimmed := strings.TrimSpace(content)
		if trimmed == "" || strings.TrimSpace(strings.TrimPrefix(line, " ")) == "" {
			isIncidentalLine = true
		}

		if !isIncidentalLine {
			allIncidental = false
			break
		}
	}

	return hasChanges && allIncidental
}

func (c *Classifier) isSuspicious(change *FileChange, prompt string) bool {
	// Large changes for small tasks
	if change.Additions+change.Deletions > 500 {
		return true
	}

	// Changes in unrelated files (not mentioned in prompt)
	promptLower := strings.ToLower(prompt)
	fileBase := filepath.Base(change.FilePath)
	fileBase = strings.TrimSuffix(fileBase, filepath.Ext(fileBase))

	// If prompt doesn't mention the file and it's not a config/test file
	if !strings.Contains(promptLower, strings.ToLower(fileBase)) {
		// Check if it's a test file
		if !strings.Contains(fileBase, "_test") && !strings.Contains(fileBase, "test_") {
			// Might be suspicious, but check if it modifies expected patterns
			if change.Additions > 100 {
				return true
			}
		}
	}

	return false
}

// GroupByIntent groups changes by their intended purpose
type IntentGroup struct {
	Name    string
	Changes []FileChange
}

// GroupByIntent groups file changes by inferred intent
func GroupByIntent(changes []FileChange, prompt string) []IntentGroup {
	groups := make(map[string][]FileChange)

	for _, change := range changes {
		intent := inferIntent(change, prompt)
		groups[intent] = append(groups[intent], change)
	}

	var result []IntentGroup
	for name, changes := range groups {
		result = append(result, IntentGroup{Name: name, Changes: changes})
	}

	return result
}

func inferIntent(change FileChange, prompt string) string {
	// Infer intent from file path and prompt
	path := strings.ToLower(change.FilePath)

	// Test files
	if strings.Contains(path, "_test.") || strings.Contains(path, "test_") {
		return "Tests"
	}

	// Configuration files
	if strings.Contains(path, "config") ||
		strings.Contains(path, ".yaml") ||
		strings.Contains(path, ".yml") ||
		strings.Contains(path, ".toml") ||
		strings.Contains(path, ".json") {
		return "Configuration"
	}

	// Documentation
	if strings.Contains(path, "readme") ||
		strings.Contains(path, ".md") ||
		strings.Contains(path, "docs/") {
		return "Documentation"
	}

	// Dependencies
	if strings.Contains(path, "go.mod") ||
		strings.Contains(path, "go.sum") ||
		strings.Contains(path, "package.json") ||
		strings.Contains(path, "requirements.txt") {
		return "Dependencies"
	}

	// Main implementation
	if strings.Contains(strings.ToLower(prompt), "fix") {
		return "Bug Fix"
	}
	if strings.Contains(strings.ToLower(prompt), "feature") ||
		strings.Contains(strings.ToLower(prompt), "add") {
		return "Feature"
	}

	return "Changes"
}
