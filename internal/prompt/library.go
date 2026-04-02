// Package prompt provides prompt template management and discovery.
package prompt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"
)

// PromptLibrary manages loading, searching, and rendering prompt templates.
type PromptLibrary struct {
	templates map[string]*PromptTemplate
}

// NewLibrary creates an empty prompt library.
func NewLibrary() *PromptLibrary {
	return &PromptLibrary{
		templates: make(map[string]*PromptTemplate),
	}
}

// Add registers a prompt template in the library.
func (lib *PromptLibrary) Add(t *PromptTemplate) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}
	lib.templates[t.Name] = t
	return nil
}

// Get retrieves a prompt template by name.
func (lib *PromptLibrary) Get(name string) (*PromptTemplate, bool) {
	t, ok := lib.templates[name]
	return t, ok
}

// List returns all prompt templates sorted by name.
func (lib *PromptLibrary) List() []*PromptTemplate {
	result := make([]*PromptTemplate, 0, len(lib.templates))
	for _, t := range lib.templates {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Search returns templates matching a query using fuzzy matching.
func (lib *PromptLibrary) Search(query string) []*PromptTemplate {
	if query == "" {
		return lib.List()
	}

	// Build search strings from name, description, and tags
	type searchEntry struct {
		text     string
		template *PromptTemplate
	}
	var entries []searchEntry
	for _, t := range lib.templates {
		searchText := t.Name + " " + t.Description + " " + strings.Join(t.Tags, " ")
		entries = append(entries, searchEntry{text: searchText, template: t})
	}

	// Fuzzy match
	var searchStrings []string
	for _, e := range entries {
		searchStrings = append(searchStrings, e.text)
	}

	matches := fuzzy.Find(query, searchStrings)

	result := make([]*PromptTemplate, 0, len(matches))
	for _, m := range matches {
		result = append(result, entries[m.Index].template)
	}

	return result
}

// ByTag returns all templates with the given tag.
func (lib *PromptLibrary) ByTag(tag string) []*PromptTemplate {
	var result []*PromptTemplate
	for _, t := range lib.templates {
		if t.HasTag(tag) {
			result = append(result, t)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Tags returns all unique tags across templates.
func (lib *PromptLibrary) Tags() []string {
	tagSet := make(map[string]bool)
	for _, t := range lib.templates {
		for _, tag := range t.Tags {
			tagSet[tag] = true
		}
	}
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// Count returns the number of templates in the library.
func (lib *PromptLibrary) Count() int {
	return len(lib.templates)
}
