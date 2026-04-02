package diff

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Viewer renders diffs in the terminal
type Viewer struct {
	width       int
	showLineNum bool
	colorize    bool
}

// NewViewer creates a new diff viewer
func NewViewer() *Viewer {
	return &Viewer{
		width:       80,
		showLineNum: true,
		colorize:    true,
	}
}

// SetWidth sets the viewer width
func (v *Viewer) SetWidth(width int) {
	v.width = width
}

// View renders a single file change
func (v *Viewer) View(change FileChange, selectedHunk int) string {
	var b strings.Builder

	// Header styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("#3b82f6")).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 1)

	classificationStyle := lipgloss.NewStyle().
		Italic(true)

	switch change.Classification {
	case Essential:
		classificationStyle = classificationStyle.Foreground(lipgloss.Color("#22c55e"))
	case Incidental:
		classificationStyle = classificationStyle.Foreground(lipgloss.Color("#f59e0b"))
	case Suspicious:
		classificationStyle = classificationStyle.Foreground(lipgloss.Color("#ef4444"))
	}

	// File header
	status := ""
	if change.Accepted != nil {
		if *change.Accepted {
			status = "[✓] "
		} else {
			status = "[✗] "
		}
	} else {
		status = "[ ] "
	}

	fmt.Fprintf(&b, "%s%s %s\n", status, headerStyle.Render(change.FilePath),
		classificationStyle.Render(change.Classification.String()))

	fmt.Fprintf(&b, "  +%d -%d lines\n\n", change.Additions, change.Deletions)

	// Render hunks
	for i, hunk := range change.Hunks {
		isSelected := i == selectedHunk
		b.WriteString(v.viewHunk(hunk, isSelected))
		b.WriteString("\n")
	}

	return b.String()
}

// ViewAll renders multiple file changes
func (v *Viewer) ViewAll(changes []FileChange, selectedFile int) string {
	var b strings.Builder

	// File list
	listStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Width(v.width)

	var listContent strings.Builder
	fmt.Fprintf(&listContent, "Files (%d)\n\n", len(changes))

	for i, change := range changes {
		prefix := "  "
		if i == selectedFile {
			prefix = "> "
		}

		status := "[ ]"
		if change.Accepted != nil {
			if *change.Accepted {
				status = "[✓]"
			} else {
				status = "[✗]"
			}
		}

		classIcon := "●"
		switch change.Classification {
		case Essential:
			classIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Render("●")
		case Incidental:
			classIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("#f59e0b")).Render("●")
		case Suspicious:
			classIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render("●")
		}

		fmt.Fprintf(&listContent, "%s%s %s %s (+%d/-%d)\n",
			prefix, status, classIcon, change.FilePath, change.Additions, change.Deletions)
	}

	b.WriteString(listStyle.Render(listContent.String()))
	b.WriteString("\n\n")

	// Selected file detail
	if selectedFile >= 0 && selectedFile < len(changes) {
		b.WriteString(v.View(changes[selectedFile], 0))
	}

	return b.String()
}

func (v *Viewer) viewHunk(hunk Hunk, isSelected bool) string {
	var b strings.Builder

	// Hunk header style
	var headerStyle lipgloss.Style
	if isSelected {
		headerStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#6b7280")).
			Foreground(lipgloss.Color("#ffffff"))
	} else {
		headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6b7280"))
	}

	fmt.Fprintf(&b, "%s\n", headerStyle.Render(
		fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldLines, hunk.NewStart, hunk.NewLines)))

	// Line styles
	contextStyle := lipgloss.NewStyle()
	addStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#22c55e")).
		Background(lipgloss.Color("#dcfce7"))
		removeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ef4444")).
		Background(lipgloss.Color("#fee2e2"))

	for _, line := range hunk.Lines {
		switch line.Type {
		case LineContext:
			fmt.Fprintf(&b, " %s\n", contextStyle.Render(line.Content))
		case LineAdded:
			fmt.Fprintf(&b, "+%s\n", addStyle.Render(line.Content))
		case LineRemoved:
			fmt.Fprintf(&b, "-%s\n", removeStyle.Render(line.Content))
		}
	}

	return b.String()
}

// Help returns the help text for diff navigation
func (v *Viewer) Help() string {
	return `
Navigation:
  [j/↓] next hunk    [k/↑] previous hunk    [h/←] prev file    [l/→] next file
Actions:
  [a] accept file    [r] reject file        [A] accept hunk    [R] reject hunk
  [q] quit           [?] help
`
}
