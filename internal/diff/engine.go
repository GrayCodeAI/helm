// Package diff provides diff viewing and classification capabilities
package diff

import (
	"fmt"
	"strings"

	"github.com/aymanbagabas/go-udiff"
)

// ChangeType classifies the importance of a change
type ChangeType int

const (
	Essential ChangeType = iota
	Incidental
	Suspicious
)

func (c ChangeType) String() string {
	switch c {
	case Essential:
		return "essential"
	case Incidental:
		return "incidental"
	case Suspicious:
		return "suspicious"
	default:
		return "unknown"
	}
}

// FileChange represents changes to a single file
type FileChange struct {
	FilePath       string
	OldContent     string
	NewContent     string
	Diff           string
	Additions      int
	Deletions      int
	Classification ChangeType
	Accepted       *bool
	Hunks          []Hunk
}

// Hunk represents a single contiguous change
type Hunk struct {
	OldStart   int
	OldLines   int
	NewStart   int
	NewLines   int
	Lines      []Line
	Accepted   *bool
}

// Line represents a single line in a diff
type Line struct {
	Type     LineType
	Content  string
	OldNum   int
	NewNum   int
}

// LineType indicates the type of line in a diff
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
)

// Engine provides diff generation and manipulation
type Engine struct{}

// NewEngine creates a new diff engine
func NewEngine() *Engine {
	return &Engine{}
}

// Generate creates a unified diff between old and new content
func (e *Engine) Generate(filePath, oldContent, newContent string) (*FileChange, error) {
	diff := udiff.Unified(filePath, filePath, oldContent, newContent)

	additions, deletions := countChanges(diff)

	return &FileChange{
		FilePath:   filePath,
		OldContent: oldContent,
		NewContent: newContent,
		Diff:       diff,
		Additions:  additions,
		Deletions:  deletions,
		Hunks:      parseHunks(diff),
	}, nil
}

// GenerateMulti creates diffs for multiple files
func (e *Engine) GenerateMulti(files map[string]struct{ Old, New string }) ([]FileChange, error) {
	var changes []FileChange
	for path, content := range files {
		change, err := e.Generate(path, content.Old, content.New)
		if err != nil {
			return nil, fmt.Errorf("diff %s: %w", path, err)
		}
		changes = append(changes, *change)
	}
	return changes, nil
}

// Apply applies accepted changes to produce final content
func (e *Engine) Apply(changes []FileChange) map[string]string {
	result := make(map[string]string)
	for _, c := range changes {
		if c.Accepted != nil && *c.Accepted {
			result[c.FilePath] = c.NewContent
		} else if c.Accepted == nil {
			// Partial acceptance - apply accepted hunks
			result[c.FilePath] = e.applyHunks(c)
		} else {
			result[c.FilePath] = c.OldContent
		}
	}
	return result
}

func (e *Engine) applyHunks(change FileChange) string {
	// If no hunks are individually marked, return new content
	hasExplicitHunks := false
	for _, h := range change.Hunks {
		if h.Accepted != nil {
			hasExplicitHunks = true
			break
		}
	}
	if !hasExplicitHunks {
		return change.NewContent
	}

	// Build result from accepted hunks
	lines := strings.Split(change.OldContent, "\n")
	var result []string
	lastOldLine := 0

	for _, hunk := range change.Hunks {
		if hunk.Accepted != nil && !*hunk.Accepted {
			continue // Skip rejected hunks
		}

		// Add context before this hunk
		for i := lastOldLine; i < hunk.OldStart-1 && i < len(lines); i++ {
			result = append(result, lines[i])
		}

		// Add hunk lines
		for _, line := range hunk.Lines {
			switch line.Type {
			case LineContext, LineAdded:
				result = append(result, line.Content)
			case LineRemoved:
				// Skip removed lines
			}
		}

		lastOldLine = hunk.OldStart + hunk.OldLines - 1
	}

	// Add remaining lines
	for i := lastOldLine; i < len(lines); i++ {
		result = append(result, lines[i])
	}

	return strings.Join(result, "\n")
}

func countChanges(diff string) (additions, deletions int) {
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}
	return
}

func parseHunks(diff string) []Hunk {
	var hunks []Hunk
	lines := strings.Split(diff, "\n")

	var currentHunk *Hunk
	oldNum := 0
	newNum := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				hunks = append(hunks, *currentHunk)
			}
			// Parse hunk header: @@ -oldStart,oldLines +newStart,newLines @@
			var h Hunk
			fmt.Sscanf(line, "@@ -%d,%d +%d,%d @@", &h.OldStart, &h.OldLines, &h.NewStart, &h.NewLines)
			currentHunk = &h
			oldNum = h.OldStart
			newNum = h.NewStart
			continue
		}

		if currentHunk == nil {
			continue
		}

		if len(line) == 0 {
			continue
		}

		switch line[0] {
		case ' ':
			currentHunk.Lines = append(currentHunk.Lines, Line{
				Type:    LineContext,
				Content: line[1:],
				OldNum:  oldNum,
				NewNum:  newNum,
			})
			oldNum++
			newNum++
		case '+':
			if !strings.HasPrefix(line, "+++") {
				currentHunk.Lines = append(currentHunk.Lines, Line{
					Type:    LineAdded,
					Content: line[1:],
					NewNum:  newNum,
				})
				newNum++
			}
		case '-':
			if !strings.HasPrefix(line, "---") {
				currentHunk.Lines = append(currentHunk.Lines, Line{
					Type:    LineRemoved,
					Content: line[1:],
					OldNum:  oldNum,
				})
				oldNum++
			}
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}
