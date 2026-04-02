// Package skills provides agent skills system (SKILL.md).
package skills

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents an agent skill
type Skill struct {
	Name           string   `yaml:"name"`
	Description    string   `yaml:"description"`
	Version        string   `yaml:"version"`
	CompatibleWith []string `yaml:"compatible_with"`
	Instructions   string   `yaml:"-"`
	Path           string   `yaml:"-"`
}

// Manager manages agent skills
type Manager struct {
	skills []Skill
}

// NewManager creates a new skills manager
func NewManager() *Manager {
	return &Manager{}
}

// DiscoverSkills discovers skills from .skills/ directories
func (m *Manager) DiscoverSkills(dirs ...string) error {
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillPath := filepath.Join(dir, entry.Name())
			skillFile := filepath.Join(skillPath, "SKILL.md")

			content, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}

			skill, err := parseSkillMD(string(content))
			if err != nil {
				continue
			}

			skill.Path = skillPath
			m.skills = append(m.skills, skill)
		}
	}

	return nil
}

// ListSkills lists all discovered skills
func (m *Manager) ListSkills() []Skill {
	return m.skills
}

// GetSkill gets a skill by name
func (m *Manager) GetSkill(name string) (*Skill, bool) {
	for _, s := range m.skills {
		if s.Name == name {
			return &s, true
		}
	}
	return nil, false
}

// BuildContext builds the context from all skills
func (m *Manager) BuildContext() string {
	if len(m.skills) == 0 {
		return ""
	}

	var ctx strings.Builder
	ctx.WriteString("# Agent Skills\n\n")

	for _, s := range m.skills {
		ctx.WriteString("## " + s.Name + "\n\n")
		ctx.WriteString(s.Description + "\n\n")
		if s.Instructions != "" {
			ctx.WriteString(s.Instructions + "\n\n")
		}
	}

	return ctx.String()
}

func parseSkillMD(content string) (Skill, error) {
	// Parse YAML frontmatter
	if !strings.HasPrefix(content, "---") {
		return Skill{}, nil
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return Skill{}, nil
	}

	var skill Skill
	if err := yaml.Unmarshal([]byte(parts[1]), &skill); err != nil {
		return Skill{}, err
	}

	skill.Instructions = strings.TrimSpace(parts[2])
	return skill, nil
}
