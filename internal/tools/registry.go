// Package tools provides tool execution for the agent.
package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Registry manages available tools
type Registry struct {
	tools map[string]Tool
}

// Tool represents an executable tool
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]string
	Handler     func(ctx context.Context, args map[string]string) (string, error)
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	r := &Registry{
		tools: make(map[string]Tool),
	}
	r.registerBuiltinTools()
	return r
}

// Register registers a tool
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name] = tool
}

// Get gets a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List lists all tools
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Execute executes a tool
func (r *Registry) Execute(ctx context.Context, name string, args map[string]string) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return tool.Handler(ctx, args)
}

func (r *Registry) registerBuiltinTools() {
	// Bash tool
	r.Register(Tool{
		Name:        "bash",
		Description: "Execute a bash command",
		Parameters:  map[string]string{"command": "The command to execute"},
		Handler: func(ctx context.Context, args map[string]string) (string, error) {
			cmd := args["command"]
			if cmd == "" {
				return "", fmt.Errorf("command is required")
			}
			c := exec.CommandContext(ctx, "bash", "-c", cmd)
			output, err := c.CombinedOutput()
			if err != nil {
				return string(output), fmt.Errorf("command failed: %w", err)
			}
			return string(output), nil
		},
	})

	// Read file tool
	r.Register(Tool{
		Name:        "read_file",
		Description: "Read the contents of a file",
		Parameters:  map[string]string{"path": "The file path to read"},
		Handler: func(ctx context.Context, args map[string]string) (string, error) {
			path := args["path"]
			if path == "" {
				return "", fmt.Errorf("path is required")
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read file: %w", err)
			}
			return string(content), nil
		},
	})

	// Write file tool
	r.Register(Tool{
		Name:        "write_file",
		Description: "Write content to a file",
		Parameters:  map[string]string{"path": "The file path", "content": "The content to write"},
		Handler: func(ctx context.Context, args map[string]string) (string, error) {
			path := args["path"]
			content := args["content"]
			if path == "" {
				return "", fmt.Errorf("path is required")
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

	// Grep tool
	r.Register(Tool{
		Name:        "grep",
		Description: "Search for a pattern in files",
		Parameters:  map[string]string{"pattern": "The pattern to search for", "path": "The path to search in"},
		Handler: func(ctx context.Context, args map[string]string) (string, error) {
			pattern := args["pattern"]
			path := args["path"]
			if pattern == "" {
				return "", fmt.Errorf("pattern is required")
			}
			if path == "" {
				path = "."
			}
			c := exec.CommandContext(ctx, "grep", "-rn", "--include=*.go", "--include=*.md", "--include=*.txt", pattern, path)
			output, err := c.CombinedOutput()
			if err != nil {
				return string(output), nil
			}
			return string(output), nil
		},
	})

	// List files tool
	r.Register(Tool{
		Name:        "list_files",
		Description: "List files in a directory",
		Parameters:  map[string]string{"path": "The directory path"},
		Handler: func(ctx context.Context, args map[string]string) (string, error) {
			path := args["path"]
			if path == "" {
				path = "."
			}
			entries, err := os.ReadDir(path)
			if err != nil {
				return "", fmt.Errorf("read directory: %w", err)
			}
			var lines []string
			for _, e := range entries {
				typeStr := "file"
				if e.IsDir() {
					typeStr = "dir"
				}
				lines = append(lines, fmt.Sprintf("%s  %s", typeStr, e.Name()))
			}
			return strings.Join(lines, "\n"), nil
		},
	})
}
