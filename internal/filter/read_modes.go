// Package filter provides file filtering and read mode capabilities.
package filter

import (
	"fmt"
	"strings"
)

// ReadMode represents different file reading strategies.
// Inspired by lean-ctx's 6 read modes.
type ReadMode string

const (
	ReadFull       ReadMode = "full"
	ReadMap        ReadMode = "map"
	ReadSignatures ReadMode = "signatures"
	ReadDiff       ReadMode = "diff"
	ReadAggressive ReadMode = "aggressive"
	ReadEntropy    ReadMode = "entropy"
	ReadLines      ReadMode = "lines"
)

// ReadOptions holds options for reading content.
type ReadOptions struct {
	Mode      ReadMode
	StartLine int
	EndLine   int
	MaxTokens int
	Query     string
}

// ReadContent reads content with the specified mode.
func ReadContent(content string, opts ReadOptions) string {
	switch opts.Mode {
	case ReadFull:
		return content
	case ReadMap:
		return readMap(content)
	case ReadSignatures:
		return readSignatures(content)
	case ReadDiff:
		return readDiff(content)
	case ReadAggressive:
		return readAggressive(content)
	case ReadEntropy:
		return readEntropy(content)
	case ReadLines:
		return readLines(content, opts.StartLine, opts.EndLine)
	default:
		return content
	}
}

func readMap(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "type ") || strings.HasPrefix(trimmed, "interface ") ||
			strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "const ") ||
			strings.HasPrefix(trimmed, "let ") || strings.HasPrefix(trimmed, "var ") {
			result = append(result, trimmed)
		}
	}
	return strings.Join(result, "\n")
}

func readSignatures(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "type ") || strings.HasPrefix(trimmed, "interface ") ||
			strings.HasPrefix(trimmed, "def ") {
			result = append(result, trimmed)
			inBlock = true
			continue
		}
		if inBlock && (trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*")) {
			continue
		}
		if inBlock && !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "}") {
			inBlock = false
		}
	}
	return strings.Join(result, "\n")
}

func readDiff(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "@@") {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func readAggressive(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") ||
			strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		result = append(result, trimmed)
	}
	return strings.Join(result, "\n")
}

func lineEntropy(line string) float64 {
	if len(line) == 0 {
		return 0
	}
	freq := make(map[rune]int)
	for _, r := range line {
		freq[r]++
	}
	var entropy float64
	length := float64(len(line))
	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * (p / 0.6931471805599453)
		}
	}
	return entropy
}

func readEntropy(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		if lineEntropy(line) > 0.6 {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func readLines(content string, start, end int) string {
	lines := strings.Split(content, "\n")
	if start < 1 {
		start = 1
	}
	if end > len(lines) || end == 0 {
		end = len(lines)
	}
	if start > end {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

// IncrementalDelta computes the diff between old and new content.
// Inspired by lean-ctx's ctx_delta.
type IncrementalDelta struct {
	Added     []string
	Removed   []string
	Unchanged int
}

// ComputeDelta computes the incremental delta between two versions.
func ComputeDelta(old, new string) IncrementalDelta {
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	oldSet := make(map[string]int)
	for _, l := range oldLines {
		oldSet[l]++
	}

	var delta IncrementalDelta
	newSet := make(map[string]int)
	for _, l := range newLines {
		newSet[l]++
	}

	for line, count := range oldSet {
		newCount := newSet[line]
		if newCount < count {
			for i := 0; i < count-newCount; i++ {
				delta.Removed = append(delta.Removed, line)
			}
		}
	}

	for line, count := range newSet {
		oldCount := oldSet[line]
		if count > oldCount {
			for i := 0; i < count-oldCount; i++ {
				delta.Added = append(delta.Added, line)
			}
		}
		delta.Unchanged += min(count, oldSet[line])
	}

	return delta
}

// FormatDelta returns a human-readable delta string.
func FormatDelta(delta IncrementalDelta) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Delta: +%d -%d (unchanged: %d)\n", len(delta.Added), len(delta.Removed), delta.Unchanged))
	for _, line := range delta.Added {
		sb.WriteString(fmt.Sprintf("+ %s\n", line))
	}
	for _, line := range delta.Removed {
		sb.WriteString(fmt.Sprintf("- %s\n", line))
	}
	return sb.String()
}
