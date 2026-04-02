// Package pruner provides smart context pruning for agent sessions.
package pruner

import (
	"sort"
	"strings"
	"time"
)

// Message represents a message that can be pruned
type Message struct {
	ID        string
	Role      string
	Content   string
	Timestamp time.Time
	Tokens    int
	Relevance float64 // 0-1 relevance score
}

// Pruner smartly prunes context to fit within limits
type Pruner struct {
	maxTokens   int
	minMessages int
}

// NewPruner creates a new context pruner
func NewPruner(maxTokens, minMessages int) *Pruner {
	if maxTokens == 0 {
		maxTokens = 128000
	}
	if minMessages == 0 {
		minMessages = 5
	}
	return &Pruner{
		maxTokens:   maxTokens,
		minMessages: minMessages,
	}
}

// Prune prunes messages to fit within token limit
func (p *Pruner) Prune(messages []Message) []Message {
	if len(messages) <= p.minMessages {
		return messages
	}

	// Calculate total tokens
	totalTokens := 0
	for _, m := range messages {
		totalTokens += m.Tokens
	}

	if totalTokens <= p.maxTokens {
		return messages
	}

	// Score messages by relevance
	scored := make([]scoredMessage, len(messages))
	for i, m := range messages {
		scored[i] = scoredMessage{
			Message: m,
			Score:   p.calculateRelevance(m, i, len(messages)),
		}
	}

	// Sort by score (lowest first for removal)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score < scored[j].Score
	})

	// Remove lowest scoring messages until under limit
	removed := make(map[int]bool)
	currentTokens := totalTokens
	remainingCount := len(messages)

	for i := 0; i < len(scored) && currentTokens > p.maxTokens && remainingCount > p.minMessages; i++ {
		// Never remove system messages or the last few messages
		if scored[i].Role == "system" {
			continue
		}
		if i >= len(scored)-3 {
			continue
		}

		originalIdx := p.findOriginalIndex(messages, scored[i].ID)
		if removed[originalIdx] {
			continue
		}

		removed[originalIdx] = true
		currentTokens -= scored[i].Tokens
		remainingCount--
	}

	// Build result
	var result []Message
	for i, m := range messages {
		if !removed[i] {
			result = append(result, m)
		}
	}

	return result
}

func (p *Pruner) calculateRelevance(msg Message, index, total int) float64 {
	score := msg.Relevance

	// Recent messages are more relevant
	recency := float64(index) / float64(total)
	score += recency * 0.3

	// User messages are more important
	if msg.Role == "user" {
		score += 0.2
	}

	// System messages are critical
	if msg.Role == "system" {
		score += 0.5
	}

	// Tool results are less important
	if msg.Role == "tool" {
		score -= 0.1
	}

	// Short messages are less important
	if len(msg.Content) < 50 {
		score -= 0.1
	}

	// Messages with code are more important
	if strings.Contains(msg.Content, "```") {
		score += 0.1
	}

	return score
}

func (p *Pruner) findOriginalIndex(messages []Message, id string) int {
	for i, m := range messages {
		if m.ID == id {
			return i
		}
	}
	return 0
}

type scoredMessage struct {
	Message
	Score float64
}

// SummarizeOldMessages summarizes old messages to save tokens
func SummarizeOldMessages(messages []Message, keepRecent int) []Message {
	if len(messages) <= keepRecent {
		return messages
	}

	// Keep system message
	var result []Message
	for _, m := range messages {
		if m.Role == "system" {
			result = append(result, m)
			break
		}
	}

	// Add summary of old messages
	var oldContent strings.Builder
	for i := 0; i < len(messages)-keepRecent; i++ {
		if messages[i].Role == "system" {
			continue
		}
		oldContent.WriteString(messages[i].Content)
		oldContent.WriteString("\n")
	}

	if oldContent.Len() > 0 {
		summary := "[Previous conversation summarized: " + oldContent.String()[:min(500, oldContent.Len())] + "...]"
		result = append(result, Message{
			Role:    "assistant",
			Content: summary,
			Tokens:  50,
		})
	}

	// Keep recent messages
	result = append(result, messages[len(messages)-keepRecent:]...)

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
