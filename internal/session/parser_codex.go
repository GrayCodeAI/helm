package session

import (
	"encoding/json"
	"fmt"
)

// CodexParser parses OpenAI Codex JSONL session files.
type CodexParser struct{}

// NewCodexParser creates a new Codex JSONL parser.
func NewCodexParser() *CodexParser {
	return &CodexParser{}
}

func (p *CodexParser) Provider() string { return "openai" }

type codexLine struct {
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Content any             `json:"content"`
	Model   string          `json:"model"`
	Usage   json.RawMessage `json:"usage"`
}

func (p *CodexParser) ParseFile(path string) (*Session, []Message, error) {
	sess := &Session{
		Provider: "openai",
		Status:   "done",
	}
	var messages []Message

	err := ParseJSONL(path, func(line []byte) error {
		var cl codexLine
		if err := json.Unmarshal(line, &cl); err != nil {
			return nil
		}

		if cl.Model != "" && sess.Model == "" {
			sess.Model = cl.Model
		}

		if cl.Role != "" {
			content := extractContent(cl.Content)
			if content != "" {
				messages = append(messages, Message{
					Role:    cl.Role,
					Content: content,
				})
			}
		}

		if cl.Type == "usage" && len(cl.Usage) > 0 {
			var usage map[string]any
			if err := json.Unmarshal(cl.Usage, &usage); err == nil {
				if v, ok := usage["prompt_tokens"].(float64); ok {
					sess.InputTokens = int64(v)
				}
				if v, ok := usage["completion_tokens"].(float64); ok {
					sess.OutputTokens = int64(v)
				}
			}
		}

		if cl.Type == "result" {
			sess.Status = "done"
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("parse codex session: %w", err)
	}

	if len(messages) > 0 && messages[0].Role == "user" {
		sess.Prompt = messages[0].Content
	}

	return sess, messages, nil
}
