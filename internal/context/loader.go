// Package context provides context file loading for agent sessions.
package context

import (
	"os"
	"path/filepath"
	"strings"
)

// ContextFile represents a loaded context file
type ContextFile struct {
	Path    string
	Name    string
	Content string
}

// Loader loads context files from the project
type Loader struct {
	contextFiles []string
}

// NewLoader creates a new context file loader
func NewLoader() *Loader {
	return &Loader{
		contextFiles: []string{
			"AGENTS.md",
			"CLAUDE.md",
			"CRUSH.md",
			"GEMINI.md",
			"CODEX.md",
			"README.md",
		},
	}
}

// LoadContextFiles loads all context files from the project
func (l *Loader) LoadContextFiles(projectDir string) []ContextFile {
	var files []ContextFile

	for _, name := range l.contextFiles {
		// Check root
		path := filepath.Join(projectDir, name)
		if content, err := os.ReadFile(path); err == nil {
			files = append(files, ContextFile{
				Path:    path,
				Name:    name,
				Content: string(content),
			})
		}

		// Check .local variant
		localName := strings.TrimSuffix(name, filepath.Ext(name)) + ".local" + filepath.Ext(name)
		localPath := filepath.Join(projectDir, localName)
		if content, err := os.ReadFile(localPath); err == nil {
			files = append(files, ContextFile{
				Path:    localPath,
				Name:    localName,
				Content: string(content),
			})
		}
	}

	return files
}

// BuildSystemPrompt builds a system prompt from context files
func (l *Loader) BuildSystemPrompt(projectDir string) string {
	files := l.LoadContextFiles(projectDir)
	if len(files) == 0 {
		return ""
	}

	var prompt strings.Builder
	prompt.WriteString("# Project Context\n\n")

	for _, f := range files {
		prompt.WriteString("## " + f.Name + "\n\n")
		prompt.WriteString(f.Content)
		prompt.WriteString("\n\n---\n\n")
	}

	return prompt.String()
}
