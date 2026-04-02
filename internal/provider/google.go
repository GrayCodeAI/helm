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

// GoogleProvider implements the Provider interface for Google's Gemini API.
type GoogleProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewGoogleProvider creates a new Google provider.
func NewGoogleProvider(apiKey string) *GoogleProvider {
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	return &GoogleProvider{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com",
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *GoogleProvider) Name() string { return "google" }

type googleRequest struct {
	Contents []googleContent `json:"contents"`
}

type googleContent struct {
	Role  string       `json:"role"`
	Parts []googlePart `json:"parts"`
}

type googlePart struct {
	Text string `json:"text"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func (p *GoogleProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("google: missing API key")
	}

	var contents []googleContent
	for _, m := range req.Messages {
		role := m.Role
		if role == "system" {
			role = "user"
		}
		contents = append(contents, googleContent{
			Role:  role,
			Parts: []googlePart{{Text: m.Content}},
		})
	}

	body := googleRequest{Contents: contents}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("google: marshal request: %w", err)
	}

	model := req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("google: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("google: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("google: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google: status %d: %s", resp.StatusCode, string(respBody))
	}

	var gr googleResponse
	if err := json.Unmarshal(respBody, &gr); err != nil {
		return nil, fmt.Errorf("google: unmarshal response: %w", err)
	}

	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("google: empty response")
	}

	return &ChatResponse{
		Content:  gr.Candidates[0].Content.Parts[0].Text,
		Provider: p.Name(),
		Model:    req.Model,
		Usage: Usage{
			InputTokens:  gr.UsageMetadata.PromptTokenCount,
			OutputTokens: gr.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}
