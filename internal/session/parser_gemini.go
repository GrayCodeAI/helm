package session

import (
	"encoding/json"
	"fmt"
)

// GeminiParser parses Google Gemini JSONL session files.
type GeminiParser struct{}

// NewGeminiParser creates a new Gemini JSONL parser.
func NewGeminiParser() *GeminiParser {
	return &GeminiParser{}
}

func (p *GeminiParser) Provider() string { return "google" }

type geminiLine struct {
	Role  string `json:"role"`
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
	Model         string `json:"model"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func (p *GeminiParser) ParseFile(path string) (*Session, []Message, error) {
	sess := &Session{
		Provider: "google",
		Status:   "done",
	}
	var messages []Message

	err := ParseJSONL(path, func(line []byte) error {
		var gl geminiLine
		if err := json.Unmarshal(line, &gl); err != nil {
			return nil
		}

		if gl.Model != "" && sess.Model == "" {
			sess.Model = gl.Model
		}

		if gl.UsageMetadata.PromptTokenCount > 0 {
			sess.InputTokens = int64(gl.UsageMetadata.PromptTokenCount)
		}
		if gl.UsageMetadata.CandidatesTokenCount > 0 {
			sess.OutputTokens = int64(gl.UsageMetadata.CandidatesTokenCount)
		}

		if gl.Role != "" {
			var content string
			for _, part := range gl.Parts {
				if part.Text != "" {
					content = part.Text
					break
				}
			}
			if content != "" {
				messages = append(messages, Message{
					Role:    gl.Role,
					Content: content,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("parse gemini session: %w", err)
	}

	if len(messages) > 0 && messages[0].Role == "user" {
		sess.Prompt = messages[0].Content
	}

	return sess, messages, nil
}
