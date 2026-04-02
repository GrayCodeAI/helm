package memory

import (
	"context"
	"math"
	"time"
)

const (
	halfLifeDays    = 30
	minConfidence   = 0.1
	decayThreshold  = 0.05
)

// ApplyForgettingCurve applies a forgetting curve to all memories.
// Memories with low usage and age decay in confidence.
func (e *Engine) ApplyForgettingCurve(ctx context.Context, project string) error {
	memories, err := e.List(ctx, project)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, m := range memories {
		newConfidence := e.decayConfidence(m, now)
		if newConfidence < decayThreshold {
			if err := e.Delete(ctx, m.ID); err != nil {
				continue
			}
		} else if newConfidence < m.Confidence {
			if err := e.Update(ctx, m.ID, m.Value, newConfidence); err != nil {
				continue
			}
		}
	}

	return nil
}

func (e *Engine) decayConfidence(m MemoryEntry, now time.Time) float64 {
	if m.UsageCount > 0 && !m.LastUsedAt.IsZero() {
		daysSinceUsed := now.Sub(m.LastUsedAt).Hours() / 24
		halves := daysSinceUsed / halfLifeDays
		decayFactor := math.Pow(0.5, halves)
		return m.Confidence * decayFactor
	}

	if m.UsageCount == 0 {
		daysSinceCreated := now.Sub(m.CreatedAt).Hours() / 24
		if daysSinceCreated > halfLifeDays*2 {
			return minConfidence
		}
		halves := daysSinceCreated / (halfLifeDays * 2)
		return m.Confidence * math.Pow(0.5, halves)
	}

	return m.Confidence
}
