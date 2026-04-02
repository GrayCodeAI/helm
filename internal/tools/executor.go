// Package tools provides concrete tool implementations for agents.
package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yourname/helm/internal/logger"
)

// Tool represents an executable agent tool
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	Handler     func(ctx context.Context, args map[string]interface{}, workdir string) (string, error)
}

// Registry manages available tools
type Registry struct {
	tools  map[string]*Tool
	logger *logger.Logger
}

// NewRegistry creates a new tool registry
func NewRegistry(log *logger.Logger) *Registry {
	r := &Registry{
		tools:  make(map[string]*Tool),
		logger: log,
	}
	r.registerBuiltinTools()
	return r
}

// Register registers a tool
func (r *Registry) Register(tool *Tool) {
	r.tools[tool.Name] = tool
}

// Get gets a tool by name
func (r *Registry) Get(name string) (*Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List lists all tools
func (r *Registry) List() []*Tool {
	tools := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Execute executes a tool
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}, workdir string) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return tool.Handler(ctx, args, workdir)
}

func (r *Registry) registerBuiltinTools() {
	// Bash tool - execute shell commands
	r.Register(&Tool{
		Name:        "bash",
		Description: "Execute a bash command. Use for running scripts, git commands, etc.",
		Parameters: map[string]interface{}{
			"command": "The command to execute",
			"timeout": "Optional timeout in seconds (default: 30)",
		},
		Handler: func(ctx context.Context, args map[string]interface{}, workdir string) (string, error) {
			command, ok := args["command"].(string)
			if !ok || command == "" {
				return "", fmt.Errorf("command is required")
			}

			timeout := 30 * time.Second
			if t, ok := args["timeout"].(float64); ok && t > 0 {
				timeout = time.Duration(t) * time.Second
			}

			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			cmd := exec.CommandContext(ctx, "bash", "-c", command)
			cmd.Dir = workdir
			cmd.Env = append(os.Environ(), "TERM=dumb")

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			output := stdout.String()
			if stderr.Len() > 0 {
				output += stderr.String()
			}

			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					return output, fmt.Errorf("command timed out after %s", timeout)
				}
				return output, fmt.Errorf("command failed: %w", err)
			}

			return output, nil
		},
	})

	// Read file tool - read file contents
	r.Register(&Tool{
		Name:        "read_file",
		Description: "Read the contents of a file. Supports line ranges.",
		Parameters: map[string]interface{}{
			"path":       "The file path to read",
			"start_line": "Optional start line (1-indexed)",
			"end_line":   "Optional end line (1-indexed)",
			"max_lines":  "Maximum lines to read (default: 1000)",
		},
		Handler: func(ctx context.Context, args map[string]interface{}, workdir string) (string, error) {
			path, ok := args["path"].(string)
			if !ok || path == "" {
				return "", fmt.Errorf("path is required")
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(workdir, path)
			}

			// Security: prevent reading outside workdir
			absWorkdir, _ := filepath.Abs(workdir)
			absPath, _ := filepath.Abs(path)
			if !strings.HasPrefix(absPath, absWorkdir) {
				return "", fmt.Errorf("access denied: path outside workdir")
			}

			maxLines := 1000
			if ml, ok := args["max_lines"].(float64); ok && ml > 0 {
				maxLines = int(ml)
			}

			startLine := 0
			if sl, ok := args["start_line"].(float64); ok {
				startLine = int(sl)
			}

			endLine := 0
			if el, ok := args["end_line"].(float64); ok {
				endLine = int(el)
			}

			file, err := os.Open(path)
			if err != nil {
				return "", fmt.Errorf("open file: %w", err)
			}
			defer file.Close()

			var lines []string
			scanner := bufio.NewScanner(file)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				if startLine > 0 && lineNum < startLine {
					continue
				}
				if endLine > 0 && lineNum > endLine {
					break
				}
				lines = append(lines, fmt.Sprintf("%d: %s", lineNum, scanner.Text()))
				if len(lines) >= maxLines {
					lines = append(lines, "... (truncated)")
					break
				}
			}

			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("read file: %w", err)
			}

			return strings.Join(lines, "\n"), nil
		},
	})

	// Write file tool - write content to a file
	r.Register(&Tool{
		Name:        "write_file",
		Description: "Write content to a file. Creates directories if needed.",
		Parameters: map[string]interface{}{
			"path":    "The file path to write",
			"content": "The content to write",
		},
		Handler: func(ctx context.Context, args map[string]interface{}, workdir string) (string, error) {
			path, ok := args["path"].(string)
			if !ok || path == "" {
				return "", fmt.Errorf("path is required")
			}

			content, ok := args["content"].(string)
			if !ok {
				return "", fmt.Errorf("content is required")
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(workdir, path)
			}

			// Security: prevent writing outside workdir
			absWorkdir, _ := filepath.Abs(workdir)
			absPath, _ := filepath.Abs(path)
			if !strings.HasPrefix(absPath, absWorkdir) {
				return "", fmt.Errorf("access denied: path outside workdir")
			}

			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return "", fmt.Errorf("create directory: %w", err)
			}

			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return "", fmt.Errorf("write file: %w", err)
			}

			return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
		},
	})

	// Multi-edit tool - apply multiple edits to a file
	r.Register(&Tool{
		Name:        "multi_edit",
		Description: "Apply multiple search-and-replace edits to a file.",
		Parameters: map[string]interface{}{
			"path":  "The file path to edit",
			"edits": "Array of {old_string, new_string} edits",
		},
		Handler: func(ctx context.Context, args map[string]interface{}, workdir string) (string, error) {
			path, ok := args["path"].(string)
			if !ok || path == "" {
				return "", fmt.Errorf("path is required")
			}

			editsRaw, ok := args["edits"]
			if !ok {
				return "", fmt.Errorf("edits is required")
			}

			editsJSON, err := json.Marshal(editsRaw)
			if err != nil {
				return "", fmt.Errorf("invalid edits: %w", err)
			}

			var edits []struct {
				Old string `json:"old_string"`
				New string `json:"new_string"`
			}
			if err := json.Unmarshal(editsJSON, &edits); err != nil {
				return "", fmt.Errorf("invalid edits format: %w", err)
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(workdir, path)
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read file: %w", err)
			}

			original := string(content)
			result := original
			applied := 0

			for _, edit := range edits {
				if strings.Contains(result, edit.Old) {
					result = strings.Replace(result, edit.Old, edit.New, 1)
					applied++
				}
			}

			if applied == 0 {
				return "", fmt.Errorf("no edits matched - old strings not found in file")
			}

			if err := os.WriteFile(path, []byte(result), 0644); err != nil {
				return "", fmt.Errorf("write file: %w", err)
			}

			return fmt.Sprintf("Applied %d/%d edits to %s", applied, len(edits), path), nil
		},
	})

	// Grep tool - search for patterns in files
	r.Register(&Tool{
		Name:        "grep",
		Description: "Search for a pattern in files using grep.",
		Parameters: map[string]interface{}{
			"pattern": "The pattern to search for",
			"path":    "Optional path to search in (default: current dir)",
			"glob":    "Optional glob pattern to filter files",
		},
		Handler: func(ctx context.Context, args map[string]interface{}, workdir string) (string, error) {
			pattern, ok := args["pattern"].(string)
			if !ok || pattern == "" {
				return "", fmt.Errorf("pattern is required")
			}

			searchPath := workdir
			if p, ok := args["path"].(string); ok && p != "" {
				searchPath = p
			}

			cmdArgs := []string{"-rn", "--color=never", pattern, searchPath}

			if glob, ok := args["glob"].(string); ok && glob != "" {
				cmdArgs = append([]string{"--include=" + glob}, cmdArgs...)
			}

			cmd := exec.CommandContext(ctx, "grep", cmdArgs...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				// grep returns exit code 1 if no matches found
				if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
					return "No matches found", nil
				}
				return string(output), fmt.Errorf("grep failed: %w", err)
			}

			return string(output), nil
		},
	})

	// Glob tool - find files matching a pattern
	r.Register(&Tool{
		Name:        "glob",
		Description: "Find files matching a glob pattern.",
		Parameters: map[string]interface{}{
			"pattern": "The glob pattern to match",
			"path":    "Optional base path (default: current dir)",
		},
		Handler: func(ctx context.Context, args map[string]interface{}, workdir string) (string, error) {
			pattern, ok := args["pattern"].(string)
			if !ok || pattern == "" {
				return "", fmt.Errorf("pattern is required")
			}

			basePath := workdir
			if p, ok := args["path"].(string); ok && p != "" {
				basePath = p
			}

			matches, err := filepath.Glob(filepath.Join(basePath, pattern))
			if err != nil {
				return "", fmt.Errorf("glob failed: %w", err)
			}

			if len(matches) == 0 {
				return "No files matched", nil
			}

			var result strings.Builder
			for _, m := range matches {
				rel, _ := filepath.Rel(workdir, m)
				result.WriteString(rel + "\n")
			}

			return result.String(), nil
		},
	})

	// Web search tool - search the web
	r.Register(&Tool{
		Name:        "web_search",
		Description: "Search the web for information.",
		Parameters: map[string]interface{}{
			"query": "The search query",
		},
		Handler: func(ctx context.Context, args map[string]interface{}, workdir string) (string, error) {
			query, ok := args["query"].(string)
			if !ok || query == "" {
				return "", fmt.Errorf("query is required")
			}

			// Use curl to search (would use a proper search API in production)
			cmd := exec.CommandContext(ctx, "curl", "-s", fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", query))
			output, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("search failed: %w", err)
			}

			// Extract text from HTML (simplified)
			text := stripHTML(string(output))
			if len(text) > 2000 {
				text = text[:2000] + "..."
			}

			return text, nil
		},
	})

	// Fetch tool - fetch a URL
	r.Register(&Tool{
		Name:        "fetch",
		Description: "Fetch the contents of a URL.",
		Parameters: map[string]interface{}{
			"url": "The URL to fetch",
		},
		Handler: func(ctx context.Context, args map[string]interface{}, workdir string) (string, error) {
			url, ok := args["url"].(string)
			if !ok || url == "" {
				return "", fmt.Errorf("url is required")
			}

			cmd := exec.CommandContext(ctx, "curl", "-s", "-L", "--max-time", "30", url)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("fetch failed: %w", err)
			}

			text := stripHTML(string(output))
			if len(text) > 5000 {
				text = text[:5000] + "..."
			}

			return text, nil
		},
	})

	// LSP restart tool - restart language server
	r.Register(&Tool{
		Name:        "lsp_restart",
		Description: "Restart the language server for the current file type.",
		Parameters:  map[string]interface{}{},
		Handler: func(ctx context.Context, args map[string]interface{}, workdir string) (string, error) {
			// In production, would communicate with LSP server
			return "LSP restart requested", nil
		},
	})
}

// stripHTML removes HTML tags from a string
func stripHTML(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(html, "")
}

// FormatToolList formats the tool list for display
func FormatToolList(tools []*Tool) string {
	var sb strings.Builder
	for _, t := range tools {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", t.Name, t.Description))
	}
	return sb.String()
}

// GetToolSchemas returns JSON schemas for all tools
func (r *Registry) GetToolSchemas() []map[string]interface{} {
	var schemas []map[string]interface{}
	for _, tool := range r.tools {
		schemas = append(schemas, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		})
	}
	return schemas
}
