package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CustomProvider implements the Provider interface for any OpenAI-compatible endpoint.
type CustomProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewCustomProvider creates a new custom OpenAI-compatible provider.
func NewCustomProvider(name, baseURL, apiKey string) *CustomProvider {
	if name == "" {
		name = "custom"
	}
	return &CustomProvider{
		name:    name,
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *CustomProvider) Name() string { return p.name }

type customMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type customRequest struct {
	Model     string      `json:"model"`
	MaxTokens int         `json:"max_tokens"`
	Messages  []customMsg `json:"messages"`
}

type customChoice struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

type customResponse struct {
	Choices []customChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (p *CustomProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if p.baseURL == "" {
		return nil, fmt.Errorf("custom: missing base URL")
	}

	body := customRequest{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
	}
	if body.MaxTokens == 0 {
		body.MaxTokens = 4096
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, customMsg{Role: m.Role, Content: m.Content})
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("custom: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("custom: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("custom: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("custom: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("custom: status %d: %s", resp.StatusCode, string(respBody))
	}

	var r customResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return nil, fmt.Errorf("custom: unmarshal response: %w", err)
	}

	if len(r.Choices) == 0 {
		return nil, fmt.Errorf("custom: empty response")
	}

	return &ChatResponse{
		Content:  r.Choices[0].Message.Content,
		Provider: p.Name(),
		Model:    req.Model,
		Usage: Usage{
			InputTokens:  r.Usage.PromptTokens,
			OutputTokens: r.Usage.CompletionTokens,
		},
	}, nil
}
