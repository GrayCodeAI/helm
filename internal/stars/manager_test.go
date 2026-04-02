package stars

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	m := NewManager()
	assert.NotNil(t, m)
}

func TestStar(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.Star("session-1", "great session")

	assert.True(t, m.IsStarred("session-1"))
	assert.False(t, m.IsStarred("session-2"))
}

func TestUnstar(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.Star("session-1", "")
	m.Unstar("session-1")
	assert.False(t, m.IsStarred("session-1"))
}

func TestListStars(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.Star("s1", "")
	m.Star("s2", "")

	stars := m.ListStars()
	assert.Len(t, stars, 2)
}

func TestPin(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.Pin("session-1", "important", 0)

	assert.True(t, m.IsPinned("session-1"))
	assert.False(t, m.IsPinned("session-2"))
}

func TestUnpin(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.Pin("session-1", "", 0)
	m.Unpin("session-1")
	assert.False(t, m.IsPinned("session-1"))
}

func TestListPins(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.Pin("s1", "", 0)
	m.Pin("s2", "", 1)

	pins := m.ListPins()
	assert.Len(t, pins, 2)
}

func TestStarTimestamp(t *testing.T) {
	t.Parallel()
	m := NewManager()
	before := time.Now()
	m.Star("s1", "")
	after := time.Now()

	stars := m.ListStars()
	assert.Len(t, stars, 1)
	assert.True(t, stars[0].StarredAt.After(before) || stars[0].StarredAt.Equal(before))
	assert.True(t, stars[0].StarredAt.Before(after) || stars[0].StarredAt.Equal(after))
}

func TestPinTimestamp(t *testing.T) {
	t.Parallel()
	m := NewManager()
	before := time.Now()
	m.Pin("s1", "", 0)
	after := time.Now()

	pins := m.ListPins()
	assert.Len(t, pins, 1)
	assert.True(t, pins[0].PinnedAt.After(before) || pins[0].PinnedAt.Equal(before))
	assert.True(t, pins[0].PinnedAt.Before(after) || pins[0].PinnedAt.Equal(after))
}
