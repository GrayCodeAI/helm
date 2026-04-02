package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OpenRouterProvider implements the Provider interface for OpenRouter API.
type OpenRouterProvider struct {
	apiKey string
	client *http.Client
}

// NewOpenRouterProvider creates a new OpenRouter provider.
func NewOpenRouterProvider(apiKey string) *OpenRouterProvider {
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	return &OpenRouterProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *OpenRouterProvider) Name() string { return "openrouter" }

type openRouterRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []openRouterMsg `json:"messages"`
}

type openRouterMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (p *OpenRouterProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("openrouter: missing API key")
	}

	body := openRouterRequest{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
	}
	if body.MaxTokens == 0 {
		body.MaxTokens = 4096
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, openRouterMsg{Role: m.Role, Content: m.Content})
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openrouter: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("openrouter: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://github.com/yourname/helm")
	httpReq.Header.Set("X-Title", "HELM")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openrouter: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter: status %d: %s", resp.StatusCode, string(respBody))
	}

	var or openRouterResponse
	if err := json.Unmarshal(respBody, &or); err != nil {
		return nil, fmt.Errorf("openrouter: unmarshal response: %w", err)
	}

	if len(or.Choices) == 0 {
		return nil, fmt.Errorf("openrouter: empty response")
	}

	return &ChatResponse{
		Content:  or.Choices[0].Message.Content,
		Provider: p.Name(),
		Model:    req.Model,
	}, nil
}
