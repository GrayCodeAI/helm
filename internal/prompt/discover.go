package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// DiscoverPaths returns the standard directories to search for prompt YAML files.
func DiscoverPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}
	return []string{
		filepath.Join(home, ".helm", "prompts"),
	}
}

// LoadYAML loads a single YAML prompt file.
func LoadYAML(path string) (*PromptTemplate, error) {
	var t PromptTemplate
	if _, err := toml.DecodeFile(path, &t); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	t.Source = "user"
	return &t, nil
}

// Discover loads all YAML prompts from standard directories.
func Discover() ([]*PromptTemplate, error) {
	var all []*PromptTemplate

	for _, dir := range DiscoverPaths() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read dir %s: %w", dir, err)
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
				continue
			}

			path := filepath.Join(dir, name)
			t, err := LoadYAML(path)
			if err != nil {
				return nil, fmt.Errorf("load %s: %w", path, err)
			}
			all = append(all, t)
		}
	}

	return all, nil
}
