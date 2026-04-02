// Package oauth provides OAuth2 integration for providers.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Token represents an OAuth2 token
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

// IsExpired checks if token is expired
func (t *Token) IsExpired() bool {
	return time.Now().After(t.Expiry)
}

// Provider represents an OAuth2 provider
type Provider struct {
	Name         string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	Scopes       []string
}

// Manager manages OAuth2 tokens
type Manager struct {
	mu     sync.RWMutex
	tokens map[string]*Token
}

// NewManager creates a new OAuth manager
func NewManager() *Manager {
	return &Manager{
		tokens: make(map[string]*Token),
	}
}

// StoreToken stores a token
func (m *Manager) StoreToken(provider string, token *Token) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[provider] = token
}

// GetToken gets a token, refreshing if expired
func (m *Manager) GetToken(ctx context.Context, provider string, refreshFn func(context.Context, string) (*Token, error)) (*Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	token, ok := m.tokens[provider]
	if !ok || token.IsExpired() {
		// Refresh token
		newToken, err := refreshFn(ctx, provider)
		if err != nil {
			return nil, fmt.Errorf("refresh token: %w", err)
		}
		m.tokens[provider] = newToken
		return newToken, nil
	}

	return token, nil
}

// ExchangeCode exchanges an authorization code for a token
func ExchangeCode(ctx context.Context, tokenURL, clientID, clientSecret, code, redirectURI string) (*Token, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, nil)
	q := req.URL.Query()
	q.Set("grant_type", "authorization_code")
	q.Set("code", code)
	q.Set("client_id", clientID)
	q.Set("client_secret", clientSecret)
	q.Set("redirect_uri", redirectURI)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("exchange failed: %d: %s", resp.StatusCode, string(body))
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}

	return &token, nil
}

// RefreshToken refreshes an access token
func RefreshToken(ctx context.Context, tokenURL, clientID, clientSecret, refreshToken string) (*Token, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, nil)
	q := req.URL.Query()
	q.Set("grant_type", "refresh_token")
	q.Set("refresh_token", refreshToken)
	q.Set("client_id", clientID)
	q.Set("client_secret", clientSecret)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	defer resp.Body.Close()

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}

	return &token, nil
}
