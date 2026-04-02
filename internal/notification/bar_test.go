package notification

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBar(t *testing.T) {
	t.Parallel()
	b := NewBar(5)
	assert.NotNil(t, b)
	assert.Equal(t, 0, b.GetUnreadCount())
}

func TestAdd(t *testing.T) {
	t.Parallel()
	b := NewBar(5)
	b.Add(&Notification{
		ID:      "1",
		Message: "test",
		Type:    "info",
	})
	assert.Equal(t, 1, b.GetUnreadCount())
}

func TestRemove(t *testing.T) {
	t.Parallel()
	b := NewBar(5)
	b.Add(&Notification{ID: "1", Message: "test"})
	b.Remove("1")
	assert.Equal(t, 0, b.GetUnreadCount())
}

func TestMarkRead(t *testing.T) {
	t.Parallel()
	b := NewBar(5)
	b.Add(&Notification{ID: "1", Message: "test"})
	b.MarkRead("1")
	assert.Equal(t, 0, b.GetUnreadCount())
}

func TestGetVisible(t *testing.T) {
	t.Parallel()
	b := NewBar(3)
	for i := 0; i < 5; i++ {
		b.Add(&Notification{
			ID:      string(rune('a' + i)),
			Message: "test",
		})
	}

	visible := b.GetVisible()
	assert.LessOrEqual(t, len(visible), 3)
}

func TestSetMode(t *testing.T) {
	t.Parallel()
	b := NewBar(5)
	b.SetMode("minimal")
	assert.Equal(t, "minimal", b.mode)
}

func TestClear(t *testing.T) {
	t.Parallel()
	b := NewBar(5)
	b.Add(&Notification{ID: "1"})
	b.Add(&Notification{ID: "2"})
	b.Clear()
	assert.Equal(t, 0, b.GetUnreadCount())
}

func TestGetWaitingSessions(t *testing.T) {
	t.Parallel()
	b := NewBar(5)
	b.Add(&Notification{ID: "1", Type: "waiting"})
	b.Add(&Notification{ID: "2", Type: "completed"})

	waiting := b.GetWaitingSessions()
	assert.Len(t, waiting, 1)
	assert.Equal(t, "1", waiting[0].ID)
}

func TestTrimExcess(t *testing.T) {
	t.Parallel()
	b := NewBar(2)
	for i := 0; i < 10; i++ {
		b.Add(&Notification{
			ID:      string(rune('a' + i)),
			Message: "test",
		})
	}
	// Should have trimmed to maxVisible*2 = 4
	assert.LessOrEqual(t, len(b.notifications), 4)
}

func TestNotificationTimestamp(t *testing.T) {
	t.Parallel()
	b := NewBar(5)
	before := time.Now()
	b.Add(&Notification{ID: "1"})
	after := time.Now()

	notifications := b.GetVisible()
	assert.Len(t, notifications, 1)
	assert.True(t, notifications[0].Timestamp.After(before) || notifications[0].Timestamp.Equal(before))
	assert.True(t, notifications[0].Timestamp.Before(after) || notifications[0].Timestamp.Equal(after))
}
