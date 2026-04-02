// Package taxonomy provides session content classification.
package taxonomy

import (
	"regexp"
	"strings"
)

// ContentType represents the type of session content
type ContentType string

const (
	ContentTypeCodeGeneration ContentType = "code_generation"
	ContentTypeBugFix         ContentType = "bug_fix"
	ContentTypeRefactor       ContentType = "refactor"
	ContentTypeDocumentation  ContentType = "documentation"
	ContentTypeTest           ContentType = "test"
	ContentTypeReview         ContentType = "review"
	ContentTypePlanning       ContentType = "planning"
	ContentTypeQuestion       ContentType = "question"
	ContentTypeOther          ContentType = "other"
)

// MessageRole represents the role of a message
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
	MessageRoleSystem    MessageRole = "system"
)

// Classification represents a content classification
type Classification struct {
	Type       ContentType
	Confidence float64
	Keywords   []string
}

// Classifier classifies session content
type Classifier struct {
	patterns map[ContentType][]*regexp.Regexp
}

// NewClassifier creates a new classifier
func NewClassifier() *Classifier {
	c := &Classifier{
		patterns: make(map[ContentType][]*regexp.Regexp),
	}

	// Code generation patterns
	c.patterns[ContentTypeCodeGeneration] = compilePatterns([]string{
		`(?i)implement|create|build|write.*function|write.*code|generate`,
	})

	// Bug fix patterns
	c.patterns[ContentTypeBugFix] = compilePatterns([]string{
		`(?i)fix|bug|error|issue|broken|not working|crash`,
	})

	// Refactor patterns
	c.patterns[ContentTypeRefactor] = compilePatterns([]string{
		`(?i)refactor|restructure|reorganize|clean up|improve.*code`,
	})

	// Documentation patterns
	c.patterns[ContentTypeDocumentation] = compilePatterns([]string{
		`(?i)document|readme|comment|explain|describe`,
	})

	// Test patterns
	c.patterns[ContentTypeTest] = compilePatterns([]string{
		`(?i)test|unit.*test|integration.*test|mock|assert`,
	})

	// Review patterns
	c.patterns[ContentTypeReview] = compilePatterns([]string{
		`(?i)review|analyze|audit|check.*code|inspect`,
	})

	// Planning patterns
	c.patterns[ContentTypePlanning] = compilePatterns([]string{
		`(?i)plan|design|architect|strategy|approach`,
	})

	// Question patterns
	c.patterns[ContentTypeQuestion] = compilePatterns([]string{
		`(?i)\?|how.*do|what.*is|why.*does|can.*you`,
	})

	return c
}

// Classify classifies content
func (c *Classifier) Classify(content string) Classification {
	content = strings.ToLower(content)
	bestType := ContentTypeOther
	bestScore := 0.0
	var bestKeywords []string

	for contentType, patterns := range c.patterns {
		score := 0.0
		var keywords []string

		for _, pattern := range patterns {
			matches := pattern.FindAllString(content, -1)
			if len(matches) > 0 {
				score += float64(len(matches)) * 0.3
				keywords = append(keywords, matches...)
			}
		}

		if score > bestScore {
			bestScore = score
			bestType = contentType
			bestKeywords = keywords
		}
	}

	// Normalize confidence
	confidence := bestScore
	if confidence > 1.0 {
		confidence = 1.0
	}

	return Classification{
		Type:       bestType,
		Confidence: confidence,
		Keywords:   bestKeywords,
	}
}

// ClassifySession classifies an entire session based on prompt and messages
func (c *Classifier) ClassifySession(prompt string, messages []string) Classification {
	// Combine all content
	var allContent strings.Builder
	allContent.WriteString(prompt)
	for _, msg := range messages {
		allContent.WriteString(" ")
		allContent.WriteString(msg)
	}

	return c.Classify(allContent.String())
}

// GetSessionShape analyzes the shape/structure of a session
func GetSessionShape(messages []string) map[string]interface{} {
	userMessages := 0
	assistantMessages := 0
	toolMessages := 0
	totalTokens := 0

	for _, msg := range messages {
		lower := strings.ToLower(msg)
		if strings.HasPrefix(lower, "user:") {
			userMessages++
		} else if strings.HasPrefix(lower, "assistant:") {
			assistantMessages++
		} else if strings.HasPrefix(lower, "tool:") {
			toolMessages++
		}
		totalTokens += len(msg) / 4
	}

	return map[string]interface{}{
		"user_messages":      userMessages,
		"assistant_messages": assistantMessages,
		"tool_messages":      toolMessages,
		"total_messages":     len(messages),
		"estimated_tokens":   totalTokens,
		"avg_message_length": func() int {
			if len(messages) == 0 {
				return 0
			}
			return totalTokens / len(messages)
		}(),
	}
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}
