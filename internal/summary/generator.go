// Package summary provides AI session summary generation
package summary

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/yourname/helm/internal/db"
	"github.com/yourname/helm/internal/provider"
	"github.com/yourname/helm/internal/session"
)

// Generator creates session summaries
type Generator struct {
	provider provider.Provider
	querier  db.Querier
}

// NewGenerator creates a new summary generator
func NewGenerator(p provider.Provider, q db.Querier) *Generator {
	return &Generator{
		provider: p,
		querier:  q,
	}
}

// Summary represents a generated session summary
type Summary struct {
	Text         string
	KeyPoints    []string
	FilesChanged []string
	TimeSaved    time.Duration
	Quality      float64 // 0-1 score
}

// Generate creates a summary for a session
func (g *Generator) Generate(ctx context.Context, sess *session.Session) (*Summary, error) {
	// Get messages for the session
	messages, err := g.querier.GetMessagesBySession(ctx, sess.ID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	// Build transcript
	var transcript strings.Builder
	transcript.WriteString(fmt.Sprintf("Session: %s\n", sess.ID))
	transcript.WriteString(fmt.Sprintf("Task: %s\n\n", sess.Prompt))
	transcript.WriteString("Conversation:\n")

	for _, msg := range messages {
		role := msg.Role
		if role == "" {
			role = "unknown"
		}
		transcript.WriteString(fmt.Sprintf("\n%s: %s\n", role, msg.Content))
	}

	// Generate summary using LLM
	prompt := fmt.Sprintf(`Summarize this coding session in 2-3 sentences.
Focus on what was accomplished, key changes made, and outcome.
Be specific but concise.

%s`, transcript.String())

	resp, err := g.provider.Chat(ctx, provider.ChatRequest{
		Model: "claude-haiku-3-5-20241022", // Use cheaper model for summaries
		Messages: []provider.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 150,
	})
	if err != nil {
		// Fallback to simple summary
		return g.generateSimpleSummary(sess), nil
	}

	summary := &Summary{
		Text:         strings.TrimSpace(resp.Content),
		KeyPoints:    g.extractKeyPoints(messages),
		FilesChanged: []string{},
		TimeSaved:    g.estimateTimeSaved(sess),
		Quality:      g.estimateQuality(sess),
	}

	// Cache summary in database
	_ = g.querier.UpdateSessionSummary(ctx, db.UpdateSessionSummaryParams{
		ID:      sess.ID,
		Summary: sql.NullString{String: summary.Text, Valid: true},
	})

	return summary, nil
}

// generateSimpleSummary creates a basic summary without LLM
func (g *Generator) generateSimpleSummary(sess *session.Session) *Summary {
	status := sess.Status
	if status == "" {
		status = "completed"
	}

	summary := fmt.Sprintf("%s: %s. Status: %s. Cost: $%.2f",
		sess.ID[:8],
		truncate(sess.Prompt, 50),
		status,
		sess.Cost,
	)

	return &Summary{
		Text:         summary,
		FilesChanged: []string{},
		TimeSaved:    g.estimateTimeSaved(sess),
		Quality:      g.estimateQuality(sess),
	}
}

// extractKeyPoints extracts key points from messages
func (g *Generator) extractKeyPoints(messages []db.Message) []string {
	var points []string

	for _, msg := range messages {
		if msg.Role == "assistant" {
			content := msg.Content
			// Look for key actions
			if strings.Contains(content, "I'll") || strings.Contains(content, "I will") {
				// Extract the action
				lines := strings.Split(content, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "I'll") || strings.HasPrefix(line, "I will") {
						points = append(points, line)
						break
					}
				}
			}
		}
	}

	return points
}

// estimateTimeSaved estimates time saved by using the agent
func (g *Generator) estimateTimeSaved(sess *session.Session) time.Duration {
	// Rough heuristic: agent work is typically 3-5x faster than manual
	duration := sess.EndedAt.Sub(sess.StartedAt)
	if duration <= 0 {
		duration = time.Minute * 5 // Default estimate
	}

	// Estimate manual time would be 4x longer
	manualTime := duration * 4
	saved := manualTime - duration

	return saved
}

// estimateQuality estimates the quality of the session output
func (g *Generator) estimateQuality(sess *session.Session) float64 {
	score := 0.5 // Base score

	// Boost for successful completion
	if sess.Status == "done" {
		score += 0.3
	}

	// Penalty for high cost (might indicate inefficiency)
	if sess.Cost > 5.0 {
		score -= 0.1
	}

	// Boost for reasonable token usage
	totalTokens := sess.InputTokens + sess.OutputTokens
	if totalTokens > 1000 && totalTokens < 50000 {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// Format formats a summary for display
func Format(s *Summary) string {
	var b strings.Builder

	b.WriteString(s.Text)
	b.WriteString("\n")

	if len(s.KeyPoints) > 0 {
		b.WriteString("\nKey points:\n")
		for _, p := range s.KeyPoints {
			b.WriteString(fmt.Sprintf("  • %s\n", p))
		}
	}

	if len(s.FilesChanged) > 0 {
		b.WriteString(fmt.Sprintf("\nFiles changed: %d\n", len(s.FilesChanged)))
	}

	if s.TimeSaved > 0 {
		b.WriteString(fmt.Sprintf("\nEstimated time saved: %s\n", s.TimeSaved.Round(time.Minute)))
	}

	b.WriteString(fmt.Sprintf("Quality score: %.0f%%\n", s.Quality*100))

	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// BatchGenerator generates summaries for multiple sessions
type BatchGenerator struct {
	generator *Generator
}

// NewBatchGenerator creates a batch generator
func NewBatchGenerator(g *Generator) *BatchGenerator {
	return &BatchGenerator{generator: g}
}

// GenerateAll generates summaries for all sessions without summaries
func (bg *BatchGenerator) GenerateAll(ctx context.Context, sessions []*session.Session) (map[string]*Summary, error) {
	results := make(map[string]*Summary)

	for _, sess := range sessions {
		// Skip if already has summary
		if sess.Summary != "" {
			continue
		}

		summary, err := bg.generator.Generate(ctx, sess)
		if err != nil {
			continue // Skip failed summaries
		}

		results[sess.ID] = summary
	}

	return results, nil
}
