package memory

import (
	"context"
	"sort"
	"strings"
)

// RecallResult holds memories ranked by relevance.
type RecallResult struct {
	Memory   MemoryEntry
	Score    float64
}

// Recall retrieves relevant memories for a session based on the prompt.
func (e *Engine) Recall(ctx context.Context, project, prompt string, limit int) ([]RecallResult, error) {
	all, err := e.List(ctx, project)
	if err != nil {
		return nil, err
	}

	promptLower := strings.ToLower(prompt)
	var results []RecallResult

	for _, m := range all {
		score := e.scoreRelevance(m, promptLower)
		if score > 0 {
			results = append(results, RecallResult{Memory: m, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (e *Engine) scoreRelevance(m MemoryEntry, prompt string) float64 {
	score := 0.0

	keyLower := strings.ToLower(m.Key)
	valueLower := strings.ToLower(m.Value)

	// Keyword overlap scoring
	promptWords := strings.Fields(prompt)
	for _, word := range promptWords {
		if len(word) < 3 {
			continue
		}
		if strings.Contains(keyLower, word) {
			score += 2.0
		}
		if strings.Contains(valueLower, word) {
			score += 1.0
		}
	}

	// Boost by confidence
	score *= m.Confidence

	// Boost by usage count (proven memories)
	score += float64(m.UsageCount) * 0.1

	// Boost by type priority
	switch m.Type {
	case TypeConvention:
		score *= 1.5
	case TypeCorrection:
		score *= 1.3
	case TypeFact:
		score *= 1.1
	}

	return score
}

// RecallByType retrieves memories of specific types for context injection.
func (e *Engine) RecallByType(ctx context.Context, project string, types []MemoryType, limit int) ([]MemoryEntry, error) {
	var all []MemoryEntry
	for _, t := range types {
		mems, err := e.ListByType(ctx, project, t)
		if err != nil {
			continue
		}
		all = append(all, mems...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].UsageCount > all[j].UsageCount
	})

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	return all, nil
}
