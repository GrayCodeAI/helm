package memory

import (
	"context"
	"strings"
)

// AutoLearn extracts and stores memories from session outcomes.
type AutoLearn struct {
	engine *Engine
}

// NewAutoLearn creates an auto-learn processor.
func NewAutoLearn(engine *Engine) *AutoLearn {
	return &AutoLearn{engine: engine}
}

// LearnFromRejectedDiff stores a correction when a user rejects a diff.
func (al *AutoLearn) LearnFromRejectedDiff(ctx context.Context, project, filePath, description string) error {
	key := "rejected:" + filePath
	return al.engine.Upsert(ctx, project, TypeCorrection, key, description, "auto", 0.6)
}

// LearnFromConvention extracts a convention from repeated patterns.
func (al *AutoLearn) LearnFromConvention(ctx context.Context, project, key, value string) error {
	return al.engine.Upsert(ctx, project, TypeConvention, key, value, "auto", 0.5)
}

// LearnFromFact stores a project fact.
func (al *AutoLearn) LearnFromFact(ctx context.Context, project, key, value string) error {
	return al.engine.Upsert(ctx, project, TypeFact, key, value, "auto", 0.7)
}

// LearnFromSkill stores a reusable skill.
func (al *AutoLearn) LearnFromSkill(ctx context.Context, project, key, value string) error {
	return al.engine.Upsert(ctx, project, TypeSkill, key, value, "auto", 0.6)
}

// ExtractConventions analyzes text for repeated patterns and stores them.
func (al *AutoLearn) ExtractConventions(ctx context.Context, project, text string) error {
	lines := strings.Split(text, "\n")
	patterns := make(map[string]int)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 10 {
			continue
		}
		patterns[line]++
	}

	for pattern, count := range patterns {
		if count >= 3 {
			key := "pattern:" + pattern[:min(len(pattern), 50)]
			if err := al.LearnFromConvention(ctx, project, key, pattern); err != nil {
				return err
			}
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
