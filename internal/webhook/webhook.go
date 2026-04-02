// Package webhook provides webhook system for event notifications.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/yourname/helm/internal/logger"
	"github.com/yourname/helm/internal/retry"
)

// Event represents a webhook event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Source    string                 `json:"source"`
}

// Endpoint represents a webhook endpoint
type Endpoint struct {
	ID           string
	URL          string
	Secret       string
	Events       []string
	Active       bool
	LastDelivery *time.Time
	Failures     int
	MaxRetries   int
	Timeout      time.Duration
}

// Delivery represents a webhook delivery attempt
type Delivery struct {
	ID          string
	EventID     string
	EndpointID  string
	Status      string
	StatusCode  int
	Response    string
	Attempts    int
	NextRetry   *time.Time
	DeliveredAt *time.Time
}

// Manager manages webhooks
type Manager struct {
	endpoints  map[string]*Endpoint
	deliveries []Delivery
	logger     *logger.Logger
	client     *http.Client
	mu         sync.RWMutex
}

// NewManager creates a new webhook manager
func NewManager(log *logger.Logger) *Manager {
	if log == nil {
		log = logger.GetDefault()
	}
	return &Manager{
		endpoints:  make(map[string]*Endpoint),
		deliveries: make([]Delivery, 0),
		logger:     log,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterEndpoint registers a webhook endpoint
func (m *Manager) RegisterEndpoint(ep *Endpoint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ep.Timeout == 0 {
		ep.Timeout = 30 * time.Second
	}
	if ep.MaxRetries == 0 {
		ep.MaxRetries = 3
	}

	m.endpoints[ep.ID] = ep
	m.logger.Info("Registered webhook endpoint: %s -> %s", ep.ID, ep.URL)
}

// RemoveEndpoint removes a webhook endpoint
func (m *Manager) RemoveEndpoint(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.endpoints, id)
}

// ListEndpoints lists all endpoints
func (m *Manager) ListEndpoints() []*Endpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	endpoints := make([]*Endpoint, 0, len(m.endpoints))
	for _, ep := range m.endpoints {
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

// GetEndpoint gets an endpoint by ID
func (m *Manager) GetEndpoint(id string) (*Endpoint, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ep, ok := m.endpoints[id]
	return ep, ok
}

// SendEvent sends an event to all matching endpoints
func (m *Manager) SendEvent(ctx context.Context, event Event) error {
	m.mu.RLock()
	endpoints := make([]*Endpoint, 0)
	for _, ep := range m.endpoints {
		if !ep.Active {
			continue
		}

		// Check if endpoint wants this event type
		if len(ep.Events) > 0 {
			matches := false
			for _, e := range ep.Events {
				if e == event.Type || e == "*" {
					matches = true
					break
				}
			}
			if !matches {
				continue
			}
		}

		endpoints = append(endpoints, ep)
	}
	m.mu.RUnlock()

	// Send to all matching endpoints concurrently
	var wg sync.WaitGroup
	for _, ep := range endpoints {
		wg.Add(1)
		go func(endpoint *Endpoint) {
			defer wg.Done()
			if err := m.deliver(ctx, endpoint, event); err != nil {
				m.logger.Error("Webhook delivery failed to %s: %v", endpoint.URL, err)
			}
		}(ep)
	}

	wg.Wait()
	return nil
}

// deliver sends an event to a single endpoint
func (m *Manager) deliver(ctx context.Context, ep *Endpoint, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-ID", ep.ID)
	req.Header.Set("X-Webhook-Event", event.Type)
	req.Header.Set("X-Webhook-Timestamp", event.Timestamp.Format(time.RFC3339))

	// Add signature
	if ep.Secret != "" {
		signature := signPayload(payload, []byte(ep.Secret))
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// Send with retry
	delivery := Delivery{
		ID:         generateID(),
		EventID:    event.ID,
		EndpointID: ep.ID,
		Status:     "pending",
	}

	err = retry.Do(ctx, func(ctx context.Context) error {
		delivery.Attempts++

		resp, err := m.client.Do(req)
		if err != nil {
			delivery.Status = "failed"
			delivery.Response = err.Error()
			return err
		}
		defer resp.Body.Close()

		delivery.StatusCode = resp.StatusCode

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			delivery.Status = "delivered"
			now := time.Now()
			delivery.DeliveredAt = &now
			m.logger.Info("Webhook delivered to %s (status: %d)", ep.URL, resp.StatusCode)
			return nil
		}

		delivery.Status = "failed"
		delivery.Response = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	},
		retry.WithMaxRetries(ep.MaxRetries),
		retry.WithBaseDelay(1*time.Second),
		retry.WithMaxDelay(30*time.Second),
	)

	// Record delivery
	m.mu.Lock()
	if delivery.Status == "failed" {
		ep.Failures++
		if ep.Failures >= 10 {
			ep.Active = false
			m.logger.Warn("Deactivated webhook endpoint %s due to too many failures", ep.ID)
		}
	} else {
		ep.Failures = 0
		now := time.Now()
		ep.LastDelivery = &now
	}
	m.deliveries = append(m.deliveries, delivery)
	m.mu.Unlock()

	return err
}

// GetDeliveries returns recent deliveries
func (m *Manager) GetDeliveries(limit int) []Delivery {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit > len(m.deliveries) {
		limit = len(m.deliveries)
	}

	return m.deliveries[len(m.deliveries)-limit:]
}

// GetEndpointDeliveries returns deliveries for a specific endpoint
func (m *Manager) GetEndpointDeliveries(endpointID string, limit int) []Delivery {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var deliveries []Delivery
	for _, d := range m.deliveries {
		if d.EndpointID == endpointID {
			deliveries = append(deliveries, d)
		}
		if len(deliveries) >= limit {
			break
		}
	}

	return deliveries
}

// RetryFailedDeliveries retries failed deliveries
func (m *Manager) RetryFailedDeliveries(ctx context.Context) error {
	m.mu.RLock()
	failed := make([]Delivery, 0)
	for _, d := range m.deliveries {
		if d.Status == "failed" && d.Attempts < 3 {
			failed = append(failed, d)
		}
	}
	m.mu.RUnlock()

	for _, d := range failed {
		ep, ok := m.GetEndpoint(d.EndpointID)
		if !ok {
			continue
		}

		// Get event and resend
		// In production, would fetch event from storage
		_ = ep
	}

	return nil
}

// Helper functions

func signPayload(payload []byte, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func generateID() string {
	return fmt.Sprintf("wh_%d", time.Now().UnixNano())
}

// VerifySignature verifies a webhook signature
func VerifySignature(payload []byte, signature string, secret string) bool {
	expected := signPayload(payload, []byte(secret))
	return hmac.Equal([]byte(signature), []byte(expected))
}

// Common event types
const (
	EventSessionCreated = "session.created"
	EventSessionDone    = "session.done"
	EventSessionFailed  = "session.failed"
	EventCostThreshold  = "cost.threshold"
	EventMemoryAdded    = "memory.added"
	EventError          = "error"
)
