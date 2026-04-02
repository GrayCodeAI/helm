// Package setup provides project setup and initialization.
package setup

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectInfo contains detected project information.
type ProjectInfo struct {
	Language       string
	Framework      string
	PackageManager string
	TestFramework  string
	HasDocker      bool
	HasCI          bool
}

// DetectProject analyzes the current directory and returns project info.
func DetectProject() ProjectInfo {
	info := ProjectInfo{
		Language:       "Unknown",
		Framework:      "None",
		PackageManager: "None",
		TestFramework:  "Unknown",
	}

	// Check for Go
	if fileExists("go.mod") {
		info.Language = "Go"
		info.PackageManager = "go modules"
		if fileExists("go.sum") {
			// Read go.mod for more info
		}
		// Check for test framework
		if dirContains("_test.go") {
			info.TestFramework = "testing (stdlib)"
		}
		// Check for popular frameworks
		if fileExists("main.go") {
			content, _ := os.ReadFile("main.go")
			if strings.Contains(string(content), "github.com/charmbracelet/bubbletea") {
				info.Framework = "Bubbletea TUI"
			}
		}
	}

	// Check for Node.js
	if fileExists("package.json") {
		info.Language = "JavaScript/TypeScript"
		if fileExists("yarn.lock") {
			info.PackageManager = "yarn"
		} else if fileExists("pnpm-lock.yaml") {
			info.PackageManager = "pnpm"
		} else if fileExists("bun.lockb") {
			info.PackageManager = "bun"
		} else {
			info.PackageManager = "npm"
		}
		// Check for TypeScript
		if fileExists("tsconfig.json") {
			info.Language = "TypeScript"
		}
		// Check for frameworks
		if fileExists("next.config.js") || fileExists("next.config.ts") {
			info.Framework = "Next.js"
		} else if fileExists("vite.config.ts") || fileExists("vite.config.js") {
			info.Framework = "Vite"
		}
	}

	// Check for Python
	if fileExists("requirements.txt") || fileExists("pyproject.toml") || fileExists("setup.py") {
		info.Language = "Python"
		if fileExists("poetry.lock") {
			info.PackageManager = "poetry"
		} else if fileExists("Pipfile") {
			info.PackageManager = "pipenv"
		} else if fileExists("uv.lock") {
			info.PackageManager = "uv"
		} else {
			info.PackageManager = "pip"
		}
		// Check for frameworks
		if fileExists("django") {
			info.Framework = "Django"
		} else if fileExists("fastapi") {
			info.Framework = "FastAPI"
		} else if fileExists("flask") {
			info.Framework = "Flask"
		}
	}

	// Check for Rust
	if fileExists("Cargo.toml") {
		info.Language = "Rust"
		info.PackageManager = "cargo"
	}

	// Check for Docker
	if fileExists("Dockerfile") || fileExists("docker-compose.yml") || fileExists("compose.yaml") {
		info.HasDocker = true
	}

	// Check for CI
	if dirExists(".github/workflows") || fileExists(".gitlab-ci.yml") || fileExists("Jenkinsfile") {
		info.HasCI = true
	}

	return info
}

// CheckProviders checks which providers have API keys configured.
func CheckProviders() map[string]bool {
	providers := map[string]bool{
		"Anthropic":  os.Getenv("ANTHROPIC_API_KEY") != "",
		"OpenAI":     os.Getenv("OPENAI_API_KEY") != "",
		"Google":     os.Getenv("GOOGLE_API_KEY") != "",
		"OpenRouter": os.Getenv("OPENROUTER_API_KEY") != "",
		"Ollama":     checkOllama(),
	}
	return providers
}

// checkOllama checks if Ollama is running locally.
func checkOllama() bool {
	// Try to connect to default Ollama port
	// This is a simple check - in production would actually try to connect
	return false // Conservative default
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// dirContains checks if any file in the directory matches the pattern.
func dirContains(pattern string) bool {
	matches, _ := filepath.Glob(pattern)
	return len(matches) > 0
}

// BuildProjectMemory extracts initial project memory from codebase.
func BuildProjectMemory(info ProjectInfo) []MemoryItem {
	var memories []MemoryItem

	// Language-specific conventions
	switch info.Language {
	case "Go":
		memories = append(memories, MemoryItem{
			Type:  "convention",
			Key:   "language",
			Value: "Go",
		})
		if info.Framework == "Bubbletea TUI" {
			memories = append(memories, MemoryItem{
				Type:  "framework",
				Key:   "ui_framework",
				Value: "Bubbletea v2",
			})
		}
	case "TypeScript", "JavaScript":
		memories = append(memories, MemoryItem{
			Type:  "convention",
			Key:   "language",
			Value: info.Language,
		})
	}

	return memories
}

// SuggestPrompts suggests relevant prompts based on project type.
func SuggestPrompts(info ProjectInfo) []string {
	var suggestions []string

	suggestions = append(suggestions, "add-feature", "fix-bug")

	if info.Language == "Go" {
		suggestions = append(suggestions, "write-tests (testify patterns)")
	} else {
		suggestions = append(suggestions, "write-tests")
	}

	suggestions = append(suggestions, "refactor", "review")

	if info.HasDocker {
		suggestions = append(suggestions, "update-deps")
	}

	return suggestions
}

// MemoryItem represents a memory entry to be created.
type MemoryItem struct {
	Type  string
	Key   string
	Value string
}
