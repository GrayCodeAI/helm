// Package thinking provides reasoning/thinking block support for models.
package thinking

import (
	"regexp"
	"strings"
)

// ThinkingBlock represents a model's reasoning/thinking content
type ThinkingBlock struct {
	Content string
	Type    string // thinking, reasoning, chain_of_thought
}

// ExtractThinkingBlocks extracts thinking blocks from content
func ExtractThinkingBlocks(content string) ([]ThinkingBlock, string) {
	var blocks []ThinkingBlock
	remaining := content

	// Anthropic-style thinking blocks
	re := regexp.MustCompile(`(?s)<thinking>(.*?)</thinking>`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		blocks = append(blocks, ThinkingBlock{
			Content: strings.TrimSpace(m[1]),
			Type:    "thinking",
		})
		remaining = strings.Replace(remaining, m[0], "", 1)
	}

	// OpenAI-style reasoning
	re2 := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
	matches2 := re2.FindAllStringSubmatch(content, -1)
	for _, m := range matches2 {
		blocks = append(blocks, ThinkingBlock{
			Content: strings.TrimSpace(m[1]),
			Type:    "reasoning",
		})
		remaining = strings.Replace(remaining, m[0], "", 1)
	}

	return blocks, strings.TrimSpace(remaining)
}

// HasThinking checks if content contains thinking blocks
func HasThinking(content string) bool {
	return strings.Contains(content, "<thinking>") ||
		strings.Contains(content, "<think>") ||
		strings.Contains(content, "Let me think") ||
		strings.Contains(content, "Let me reason")
}

// StripThinking removes thinking blocks from content
func StripThinking(content string) string {
	content = regexp.MustCompile(`(?s)<thinking>.*?</thinking>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?s)<think>.*?</think>`).ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

// CountThinkingTokens estimates token count for thinking content
func CountThinkingTokens(content string) int {
	blocks, _ := ExtractThinkingBlocks(content)
	total := 0
	for _, b := range blocks {
		// Rough estimate: 1 token per 4 characters
		total += len(b.Content) / 4
	}
	return total
}
