package diff

import (
	"path/filepath"
	"strings"
)

// ChangeScore represents the importance score of a change
type ChangeScore struct {
	Score      int
	Category   string
	Reasons    []string
	Confidence float64
}

// Triage provides smart diff triage capabilities
type Triage struct {
	classifier *Classifier
}

// NewTriage creates a new diff triage
func NewTriage() *Triage {
	return &Triage{
		classifier: NewClassifier(),
	}
}

// ScoreChange calculates an importance score for a change
func (t *Triage) ScoreChange(change *FileChange, prompt string) ChangeScore {
	score := 50 // Base score
	reasons := []string{}

	// Check if file is mentioned in prompt (high relevance)
	if t.isPromptRelated(change.FilePath, prompt) {
		score += 30
		reasons = append(reasons, "File mentioned in prompt")
	}

	// Check for essential vs incidental
	classification := t.classifier.Classify(change, prompt)
	switch classification {
	case Essential:
		score += 20
		reasons = append(reasons, "Essential change")
	case Incidental:
		score -= 20
		reasons = append(reasons, "Incidental change (imports/formatting)")
	case Suspicious:
		score -= 10
		reasons = append(reasons, "Suspicious change (unrelated file)")
	}

	// Size-based scoring
	changeSize := change.Additions + change.Deletions
	switch {
	case changeSize > 500:
		score -= 15
		reasons = append(reasons, "Very large change (possible over-engineering)")
	case changeSize > 200:
		score -= 5
		reasons = append(reasons, "Large change")
	case changeSize < 10:
		score += 5
		reasons = append(reasons, "Focused, minimal change")
	}

	// Test file handling
	if t.isTestFile(change.FilePath) {
		if strings.Contains(strings.ToLower(prompt), "test") {
			score += 15
			reasons = append(reasons, "Test changes for testing task")
		} else {
			score += 5
			reasons = append(reasons, "Test file changes")
		}
	}

	// Configuration file handling
	if t.isConfigFile(change.FilePath) {
		if strings.Contains(strings.ToLower(prompt), "config") ||
			strings.Contains(strings.ToLower(prompt), "dependency") ||
			strings.Contains(strings.ToLower(prompt), "dep") {
			score += 10
			reasons = append(reasons, "Config changes for config/deps task")
		} else {
			score -= 5
			reasons = append(reasons, "Incidental config change")
		}
	}

	// Documentation handling
	if t.isDocFile(change.FilePath) {
		if strings.Contains(strings.ToLower(prompt), "doc") {
			score += 10
			reasons = append(reasons, "Documentation changes for docs task")
		} else {
			score -= 10
			reasons = append(reasons, "Documentation change (likely incidental)")
		}
	}

	// Calculate confidence based on evidence
	confidence := 0.5
	if len(reasons) > 0 {
		confidence = 0.5 + (float64(len(reasons)) * 0.1)
		if confidence > 1.0 {
			confidence = 1.0
		}
	}

	// Determine category
	category := t.determineCategory(score, classification)

	return ChangeScore{
		Score:      score,
		Category:   category,
		Reasons:    reasons,
		Confidence: confidence,
	}
}

// isPromptRelated checks if a file is related to the prompt
func (t *Triage) isPromptRelated(filePath, prompt string) bool {
	promptLower := strings.ToLower(prompt)
	fileName := filepath.Base(filePath)
	fileNameLower := strings.ToLower(fileName)
	fileNameNoExt := strings.TrimSuffix(fileNameLower, filepath.Ext(fileNameLower))

	// Check if filename (without extension) is in prompt
	if strings.Contains(promptLower, fileNameNoExt) {
		return true
	}

	// Check for package/directory names in prompt
	dir := filepath.Dir(filePath)
	dirs := strings.Split(dir, string(filepath.Separator))
	for _, d := range dirs {
		if d != "." && d != "/" && len(d) > 2 {
			if strings.Contains(promptLower, strings.ToLower(d)) {
				return true
			}
		}
	}

	// Check for function/type names in prompt (camelCase/PascalCase extraction)
	words := extractWords(promptLower)
	for _, word := range words {
		if len(word) > 3 && strings.Contains(fileNameLower, word) {
			return true
		}
	}

	return false
}

// isImportOrFormatting checks if change is only imports/formatting
func (t *Triage) _isImportOrFormatting(change *FileChange) bool {
	return t.classifier.isIncidental(change)
}

// isUnrelatedFile checks if a file change is unrelated to the task
func (t *Triage) _isUnrelatedFile(change *FileChange, prompt string) bool {
	// If not mentioned in prompt and not a standard file type
	if !t.isPromptRelated(change.FilePath, prompt) {
		// Check if it's a common/generated file
		fileName := filepath.Base(change.FilePath)
		unrelatedPatterns := []string{
			".generated.",
			".pb.go",
			"_gen.go",
			"mock_",
			".min.js",
			".bundle.",
			"vendor/",
			"node_modules/",
			".idea/",
			".vscode/",
		}

		for _, pattern := range unrelatedPatterns {
			if strings.Contains(fileName, pattern) || strings.Contains(change.FilePath, pattern) {
				return true
			}
		}

		// Large changes in unrelated files are suspicious
		if change.Additions+change.Deletions > 100 {
			return true
		}
	}

	return false
}

// isTestFile checks if a file is a test file
func (t *Triage) isTestFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	return strings.Contains(fileName, "_test.") ||
		strings.Contains(fileName, "_spec.") ||
		strings.Contains(fileName, ".test.") ||
		strings.Contains(fileName, "__tests__")
}

// isConfigFile checks if a file is a config file
func (t *Triage) isConfigFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	ext := filepath.Ext(fileName)

	configExts := []string{
		".yaml", ".yml", ".json", ".toml", ".ini", ".conf",
		".config", ".env", ".properties",
	}

	for _, ce := range configExts {
		if ext == ce {
			return true
		}
	}

	configNames := []string{
		"go.mod", "go.sum", "package.json", "package-lock.json",
		"yarn.lock", "Cargo.toml", "Cargo.lock", "requirements.txt",
		"Pipfile", "poetry.lock", "Gemfile", "Gemfile.lock",
	}

	for _, cn := range configNames {
		if fileName == cn {
			return true
		}
	}

	return false
}

// isDocFile checks if a file is a documentation file
func (t *Triage) isDocFile(filePath string) bool {
	fileName := strings.ToLower(filepath.Base(filePath))
	ext := filepath.Ext(fileName)

	docExts := []string{
		".md", ".markdown", ".rst", ".txt",
	}

	for _, de := range docExts {
		if ext == de {
			return true
		}
	}

	docNames := []string{
		"readme", "changelog", "contributing", "license",
		"authors", "notice", "security", "code_of_conduct",
	}

	fileNameNoExt := strings.TrimSuffix(fileName, ext)
	for _, dn := range docNames {
		if fileNameNoExt == dn {
			return true
		}
	}

	return strings.Contains(filePath, "/docs/") ||
		strings.Contains(filePath, "/documentation/")
}

// determineCategory determines the category based on score and classification
func (t *Triage) determineCategory(score int, classification ChangeType) string {
	switch {
	case score >= 70:
		return "Critical"
	case score >= 50:
		return "Important"
	case score >= 30:
		return "Normal"
	case score >= 10:
		return "Low Priority"
	default:
		if classification == Suspicious {
			return "Suspicious"
		}
		return "Incidental"
	}
}

// extractWords extracts words from text
func extractWords(text string) []string {
	// Simple word extraction - split by non-alphanumeric
	var words []string
	current := ""

	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			current += string(r)
		} else {
			if len(current) > 2 {
				words = append(words, current)
			}
			current = ""
		}
	}

	if len(current) > 2 {
		words = append(words, current)
	}

	return words
}

// TriageResult holds the complete triage results for a set of changes
type TriageResult struct {
	Changes    []FileChange
	Scores     map[string]ChangeScore
	ByCategory map[string][]FileChange
	Summary    TriageSummary
}

// TriageSummary provides a summary of the triage
type TriageSummary struct {
	TotalFiles       int
	CriticalCount    int
	ImportantCount   int
	NormalCount      int
	LowPriorityCount int
	SuspiciousCount  int
	IncidentalCount  int
	TotalScore       int
	AvgScore         float64
}

// TriageChanges performs complete triage on a set of changes
func (t *Triage) TriageChanges(changes []FileChange, prompt string) TriageResult {
	result := TriageResult{
		Changes:    changes,
		Scores:     make(map[string]ChangeScore),
		ByCategory: make(map[string][]FileChange),
	}

	totalScore := 0

	for _, change := range changes {
		score := t.ScoreChange(&change, prompt)
		result.Scores[change.FilePath] = score
		result.ByCategory[score.Category] = append(result.ByCategory[score.Category], change)
		totalScore += score.Score

		// Update summary counts
		switch score.Category {
		case "Critical":
			result.Summary.CriticalCount++
		case "Important":
			result.Summary.ImportantCount++
		case "Normal":
			result.Summary.NormalCount++
		case "Low Priority":
			result.Summary.LowPriorityCount++
		case "Suspicious":
			result.Summary.SuspiciousCount++
		case "Incidental":
			result.Summary.IncidentalCount++
		}
	}

	result.Summary.TotalFiles = len(changes)
	result.Summary.TotalScore = totalScore
	if len(changes) > 0 {
		result.Summary.AvgScore = float64(totalScore) / float64(len(changes))
	}

	return result
}

// FilterChanges filters changes by category
func (t *Triage) FilterChanges(result TriageResult, categories ...string) []FileChange {
	var filtered []FileChange

	for _, change := range result.Changes {
		score := result.Scores[change.FilePath]
		for _, cat := range categories {
			if score.Category == cat {
				filtered = append(filtered, change)
				break
			}
		}
	}

	return filtered
}

// GetHighPriorityChanges returns changes that need attention
func (t *Triage) GetHighPriorityChanges(result TriageResult) []FileChange {
	return t.FilterChanges(result, "Critical", "Important", "Suspicious")
}

// GetReviewableChanges returns changes that should be reviewed
func (t *Triage) GetReviewableChanges(result TriageResult) []FileChange {
	// Filter out incidental changes
	var reviewable []FileChange

	for _, change := range result.Changes {
		score := result.Scores[change.FilePath]
		if score.Category != "Incidental" && score.Category != "Low Priority" {
			reviewable = append(reviewable, change)
		}
	}

	return reviewable
}
