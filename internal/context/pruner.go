// Package context provides smart context pruning capabilities
package context

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yourname/helm/internal/memory"
	"github.com/yourname/helm/internal/provider"
)

// Pruner intelligently prunes context to fit token budget
type Pruner struct {
	modelCatalog *provider.ModelCatalog
}

// NewPruner creates a new context pruner
func NewPruner(catalog *provider.ModelCatalog) *Pruner {
	return &Pruner{modelCatalog: catalog}
}

// Config configures the pruning behavior
type Config struct {
	Model            string
	Prompt           string
	HardLimit        int // Never exceed this
	SoftLimit        int // Warn when approaching this
	PreservePatterns []string // File patterns to always preserve
}

// File represents a file with relevance scoring
type File struct {
	Path      string
	Content   string
	Tokens    int
	Relevance float64
	Priority  FilePriority
}

// FilePriority defines file importance
type FilePriority int

const (
	PriorityCritical FilePriority = iota // Project memory, user-specified
	PriorityHigh                         // Files mentioned in prompt
	PriorityMedium                       // Recently changed, relevant type
	PriorityLow                          // Other project files
	PriorityNone                         // Can be dropped
)

// Result represents the pruned context
type Result struct {
	Files         []File
	TotalTokens   int
	DroppedFiles  []string
	Warnings      []string
}

// Prune selects relevant files to fit within token budget
func (p *Pruner) Prune(files []File, config Config) (*Result, error) {
	// Get model context window
	modelInfo, ok := p.modelCatalog.Get(config.Model)
	if !ok {
		// Use default if model not found
		modelInfo = provider.ModelInfo{
			ContextWindow:   128000,
			MaxOutputTokens: 4096,
		}
	}

	// Calculate effective limit (leave room for output + prompt + memory)
	promptTokens := estimateTokens(config.Prompt)
	outputBuffer := modelInfo.MaxOutputTokens
	memoryBuffer := 2000 // Reserve for project memory

	effectiveLimit := modelInfo.ContextWindow - outputBuffer - memoryBuffer - promptTokens
	if effectiveLimit < 0 {
		effectiveLimit = modelInfo.ContextWindow / 2
	}

	// Use configured hard limit if specified
	if config.HardLimit > 0 && config.HardLimit < effectiveLimit {
		effectiveLimit = config.HardLimit
	}

	softLimit := int(float64(effectiveLimit) * 0.8)
	if config.SoftLimit > 0 {
		softLimit = config.SoftLimit
	}

	// Score all files
	scoredFiles := p.scoreFiles(files, config)

	// Sort by priority and relevance (descending)
	sort.Slice(scoredFiles, func(i, j int) bool {
		if scoredFiles[i].Priority != scoredFiles[j].Priority {
			return scoredFiles[i].Priority < scoredFiles[j].Priority
		}
		return scoredFiles[i].Relevance > scoredFiles[j].Relevance
	})

	// Select files up to limit
	var selected []File
	var dropped []string
	totalTokens := promptTokens + memoryBuffer
	warnings := []string{}

	for _, f := range scoredFiles {
		// Always include critical files
		if f.Priority == PriorityCritical {
			selected = append(selected, f)
			totalTokens += f.Tokens
			continue
		}

		// Check if adding this file would exceed limit
		if totalTokens+f.Tokens > effectiveLimit {
			dropped = append(dropped, f.Path)
			continue
		}

		selected = append(selected, f)
		totalTokens += f.Tokens
	}

	// Check soft limit warning
	if totalTokens > softLimit {
		warnings = append(warnings, fmt.Sprintf(
			"Approaching token limit: %d/%d tokens (%.1f%%)",
			totalTokens, effectiveLimit, float64(totalTokens)/float64(effectiveLimit)*100))
	}

	return &Result{
		Files:        selected,
		TotalTokens:  totalTokens,
		DroppedFiles: dropped,
		Warnings:     warnings,
	}, nil
}

// scoreFiles calculates relevance scores for all files
func (p *Pruner) scoreFiles(files []File, config Config) []File {
	promptLower := strings.ToLower(config.Prompt)
	promptWords := extractKeywords(promptLower)

	for i := range files {
		f := &files[i]

		// Check if file matches preserve patterns
		for _, pattern := range config.PreservePatterns {
			if matched, _ := filepath.Match(pattern, f.Path); matched {
				f.Priority = PriorityCritical
				f.Relevance = 1.0
				break
			}
		}

		// Skip scoring if already critical
		if f.Priority == PriorityCritical {
			continue
		}

		// Check if file is mentioned in prompt
		if strings.Contains(promptLower, strings.ToLower(f.Path)) {
			f.Priority = PriorityHigh
			f.Relevance = 0.9
			continue
		}

		// Check for filename match
		filename := filepath.Base(f.Path)
		if strings.Contains(promptLower, strings.ToLower(filename)) {
			f.Priority = PriorityHigh
			f.Relevance = 0.85
			continue
		}

		// Calculate keyword overlap
		contentSample := strings.ToLower(f.Content)
		if len(contentSample) > 1000 {
			contentSample = contentSample[:1000]
		}

		overlap := 0
		for _, word := range promptWords {
			if strings.Contains(contentSample, word) {
				overlap++
			}
		}

		if overlap > 0 {
			f.Relevance = float64(overlap) / float64(len(promptWords)) * 0.7
			if f.Relevance > 0.5 {
				f.Priority = PriorityMedium
			} else {
				f.Priority = PriorityLow
			}
		} else {
			f.Priority = PriorityLow
			f.Relevance = 0.1
		}

		// Boost for recent files (if modified recently - this would need git info)
		if isConfigFile(f.Path) {
			f.Relevance *= 0.5 // Deprioritize config files
		}
	}

	return files
}

// estimateTokens estimates token count for text
func estimateTokens(text string) int {
	// Rough estimate: ~4 characters per token for English
	return len(text) / 4
}

// extractKeywords extracts important keywords from text
func extractKeywords(text string) []string {
	words := strings.Fields(text)
	var keywords []string

	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "must": true, "shall": true, "can": true,
		"need": true, "dare": true, "ought": true, "used": true, "to": true, "of": true,
		"in": true, "for": true, "on": true, "with": true, "at": true, "by": true,
		"from": true, "as": true, "into": true, "through": true, "during": true,
		"before": true, "after": true, "above": true, "below": true, "between": true,
		"under": true, "again": true, "further": true, "then": true, "once": true,
		"here": true, "there": true, "when": true, "where": true, "why": true, "how": true,
		"all": true, "any": true, "both": true, "each": true, "few": true, "more": true,
		"most": true, "other": true, "some": true, "such": true, "no": true, "nor": true,
		"not": true, "only": true, "own": true, "same": true, "so": true, "than": true,
		"too": true, "very": true, "just": true, "and": true, "but": true, "if": true,
		"or": true, "because": true, "until": true, "while": true, "this": true, "that": true,
	}

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:\"'()[]{}"))
		if len(word) > 3 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// isConfigFile checks if a file is a config file
func isConfigFile(path string) bool {
	configExts := []string{
		".json", ".yaml", ".yml", ".toml", ".ini", ".conf",
		".config", ".env", ".lock", ".mod", ".sum",
	}
	ext := strings.ToLower(filepath.Ext(path))
	for _, ce := range configExts {
		if ext == ce {
			return true
		}
	}
	return false
}

// Budget manages token budget allocation
type Budget struct {
	TotalAvailable int
	Used           int
	Reserved       int
}

// Remaining returns remaining tokens
func (b *Budget) Remaining() int {
	return b.TotalAvailable - b.Used - b.Reserved
}

// Allocate reserves tokens for a component
func (b *Budget) Allocate(amount int) bool {
	if b.Remaining() >= amount {
		b.Reserved += amount
		return true
	}
	return false
}

// Use marks tokens as used
func (b *Budget) Use(amount int) {
	b.Used += amount
	b.Reserved -= amount
	if b.Reserved < 0 {
		b.Reserved = 0
	}
}

// SmartPruner combines context pruning with budget management
type SmartPruner struct {
	pruner  *Pruner
	memory  *memory.Engine
}

// NewSmartPruner creates a smart pruner with memory awareness
func NewSmartPruner(pruner *Pruner, mem *memory.Engine) *SmartPruner {
	return &SmartPruner{
		pruner: pruner,
		memory: mem,
	}
}

// PruneWithMemory includes project memory in context
func (sp *SmartPruner) PruneWithMemory(ctx context.Context, files []File, config Config, project string) (*Result, error) {
	// Get relevant memories
	if sp.memory != nil {
		memories, err := sp.memory.Recall(ctx, project, config.Prompt, 10)
		if err == nil && len(memories) > 0 {
			// Add memories as a virtual file
			var memoryContent strings.Builder
			memoryContent.WriteString("# Project Context\n\n")
			for _, m := range memories {
				memoryContent.WriteString(fmt.Sprintf("[%s] %s: %s\n", m.Memory.Type, m.Memory.Key, m.Memory.Value))
			}

			files = append([]File{{
				Path:      ".helm/memory",
				Content:   memoryContent.String(),
				Tokens:    estimateTokens(memoryContent.String()),
				Relevance: 1.0,
				Priority:  PriorityCritical,
			}}, files...)
		}
	}

	return sp.pruner.Prune(files, config)
}
