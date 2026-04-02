package prompt

import (
	"bytes"
	"fmt"
	"text/template"
)

// PromptVariable defines a variable that can be injected into a prompt template.
type PromptVariable struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default,omitempty"`
}

// PromptTemplate represents a YAML-defined prompt template with variable injection.
type PromptTemplate struct {
	Name        string           `yaml:"name"`
	Description string           `yaml:"description"`
	Tags        []string         `yaml:"tags,omitempty"`
	Complexity  string           `yaml:"complexity,omitempty"`
	Context     []string         `yaml:"context,omitempty"`
	Template    string           `yaml:"template"`
	Variables   []PromptVariable `yaml:"variables,omitempty"`
	Source      string           `yaml:"source,omitempty"` // "builtin" or "user"
}

// Validate checks the template for required variables and valid template syntax.
func (pt *PromptTemplate) Validate() error {
	if pt.Name == "" {
		return fmt.Errorf("prompt template requires a name")
	}
	if pt.Template == "" {
		return fmt.Errorf("prompt template %q requires a template body", pt.Name)
	}

	// Check template syntax
	_, err := template.New(pt.Name).Parse(pt.Template)
	if err != nil {
		return fmt.Errorf("prompt template %q: invalid template syntax: %w", pt.Name, err)
	}

	// Check for missing required variable definitions
	for _, v := range pt.Variables {
		if v.Name == "" {
			return fmt.Errorf("prompt template %q: variable has empty name", pt.Name)
		}
	}

	return nil
}

// Render executes the template with the provided variables.
func (pt *PromptTemplate) Render(vars map[string]string) (string, error) {
	tmpl, err := template.New(pt.Name).Parse(pt.Template)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", pt.Name, err)
	}

	data := make(map[string]string)
	for _, v := range pt.Variables {
		if val, ok := vars[v.Name]; ok {
			data[v.Name] = val
		} else if v.Default != "" {
			data[v.Name] = v.Default
		} else if v.Required {
			return "", fmt.Errorf("required variable %q not provided", v.Name)
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template %q: %w", pt.Name, err)
	}

	return buf.String(), nil
}

// RequiredVariables returns variables that must be provided.
func (pt *PromptTemplate) RequiredVariables() []PromptVariable {
	var result []PromptVariable
	for _, v := range pt.Variables {
		if v.Required {
			result = append(result, v)
		}
	}
	return result
}

// HasTag checks if the template has the given tag.
func (pt *PromptTemplate) HasTag(tag string) bool {
	for _, t := range pt.Tags {
		if t == tag {
			return true
		}
	}
	return false
}
