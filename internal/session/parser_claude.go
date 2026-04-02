package session

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ClaudeParser parses Claude Code JSONL session files.
type ClaudeParser struct{}

// NewClaudeParser creates a new Claude Code JSONL parser.
func NewClaudeParser() *ClaudeParser {
	return &ClaudeParser{}
}

func (p *ClaudeParser) Provider() string { return "anthropic" }

type claudeLine struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
	Subtype string          `json:"subtype"`
}

type claudeMessage struct {
	Type    string `json:"type"`
	Message struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	} `json:"message"`
	Timestamp string `json:"timestamp"`
}

func (p *ClaudeParser) ParseFile(path string) (*Session, []Message, error) {
	sess := &Session{
		Provider: "anthropic",
		Status:   "done",
	}
	var messages []Message

	err := ParseJSONL(path, func(line []byte) error {
		var cl claudeLine
		if err := json.Unmarshal(line, &cl); err != nil {
			return nil
		}

		switch cl.Type {
		case "message":
			var cm claudeMessage
			if err := json.Unmarshal(line, &cm); err != nil {
				return nil
			}
			content := extractContent(cm.Message.Content)
			if content != "" {
				messages = append(messages, Message{
					Role:      cm.Message.Role,
					Content:   content,
					Timestamp: parseTime(cm.Timestamp),
				})
			}
		case "session_start":
			var meta map[string]any
			if err := json.Unmarshal(line, &meta); err != nil {
				return nil
			}
			if model, ok := meta["model"].(string); ok {
				sess.Model = model
			}
		case "result":
			var meta map[string]any
			if err := json.Unmarshal(line, &meta); err != nil {
				return nil
			}
			if status, ok := meta["status"].(string); ok {
				sess.Status = status
			}
			if usage, ok := meta["usage"].(map[string]any); ok {
				if v, ok := usage["input_tokens"].(float64); ok {
					sess.InputTokens = int64(v)
				}
				if v, ok := usage["output_tokens"].(float64); ok {
					sess.OutputTokens = int64(v)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("parse claude session: %w", err)
	}

	if len(messages) > 0 && messages[0].Role == "user" {
		sess.Prompt = messages[0].Content
	}

	return sess, messages, nil
}

func extractContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		if text, ok := v["text"].(string); ok {
			return text
		}
	}
	return ""
}
