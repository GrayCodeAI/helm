package session

import (
	"encoding/json"
	"fmt"
)

// OpenCodeParser parses OpenCode JSONL session files.
type OpenCodeParser struct{}

// NewOpenCodeParser creates a new OpenCode JSONL parser.
func NewOpenCodeParser() *OpenCodeParser {
	return &OpenCodeParser{}
}

func (p *OpenCodeParser) Provider() string { return "opencode" }

type openCodeLine struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content any    `json:"content"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Status string `json:"status"`
}

func (p *OpenCodeParser) ParseFile(path string) (*Session, []Message, error) {
	sess := &Session{
		Provider: "opencode",
		Status:   "done",
	}
	var messages []Message

	err := ParseJSONL(path, func(line []byte) error {
		var ol openCodeLine
		if err := json.Unmarshal(line, &ol); err != nil {
			return nil
		}

		if ol.Model != "" && sess.Model == "" {
			sess.Model = ol.Model
		}

		if ol.Usage.PromptTokens > 0 {
			sess.InputTokens = int64(ol.Usage.PromptTokens)
		}
		if ol.Usage.CompletionTokens > 0 {
			sess.OutputTokens = int64(ol.Usage.CompletionTokens)
		}

		if ol.Status != "" {
			sess.Status = ol.Status
		}

		if ol.Role != "" {
			content := extractContent(ol.Content)
			if content != "" {
				messages = append(messages, Message{
					Role:    ol.Role,
					Content: content,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("parse opencode session: %w", err)
	}

	if len(messages) > 0 && messages[0].Role == "user" {
		sess.Prompt = messages[0].Content
	}

	return sess, messages, nil
}
