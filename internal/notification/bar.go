// Package notification provides session notification bar system.
package notification

import (
	"sync"
	"time"
)

// Notification represents a session notification
type Notification struct {
	ID        string
	SessionID string
	Message   string
	Type      string // waiting, completed, failed, info
	Timestamp time.Time
	Read      bool
	Key       string // keyboard shortcut
}

// Bar manages the notification bar
type Bar struct {
	mu            sync.RWMutex
	notifications []*Notification
	maxVisible    int
	mode          string // minimal, compact, full
}

// NewBar creates a new notification bar
func NewBar(maxVisible int) *Bar {
	return &Bar{
		notifications: make([]*Notification, 0),
		maxVisible:    maxVisible,
		mode:          "full",
	}
}

// Add adds a notification
func (b *Bar) Add(n *Notification) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n.Timestamp = time.Now()
	b.notifications = append(b.notifications, n)

	// Trim if exceeds max
	if len(b.notifications) > b.maxVisible*2 {
		b.notifications = b.notifications[len(b.notifications)-b.maxVisible:]
	}
}

// Remove removes a notification
func (b *Bar) Remove(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, n := range b.notifications {
		if n.ID == id {
			b.notifications = append(b.notifications[:i], b.notifications[i+1:]...)
			return
		}
	}
}

// MarkRead marks a notification as read
func (b *Bar) MarkRead(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, n := range b.notifications {
		if n.ID == id {
			n.Read = true
			return
		}
	}
}

// GetUnreadCount returns count of unread notifications
func (b *Bar) GetUnreadCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	count := 0
	for _, n := range b.notifications {
		if !n.Read {
			count++
		}
	}
	return count
}

// GetVisible returns visible notifications based on mode
func (b *Bar) GetVisible() []*Notification {
	b.mu.RLock()
	defer b.mu.RUnlock()

	switch b.mode {
	case "minimal":
		// Return only count
		return nil
	case "compact":
		// Return last 3
		if len(b.notifications) > 3 {
			return b.notifications[len(b.notifications)-3:]
		}
		return b.notifications
	default:
		// Return all visible
		if len(b.notifications) > b.maxVisible {
			return b.notifications[len(b.notifications)-b.maxVisible:]
		}
		return b.notifications
	}
}

// SetMode sets the display mode
func (b *Bar) SetMode(mode string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.mode = mode
}

// Clear clears all notifications
func (b *Bar) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.notifications = nil
}

// GetWaitingSessions returns sessions waiting for attention
func (b *Bar) GetWaitingSessions() []*Notification {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var waiting []*Notification
	for _, n := range b.notifications {
		if n.Type == "waiting" && !n.Read {
			waiting = append(waiting, n)
		}
	}
	return waiting
}
