package trash

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	m := NewManager()
	assert.NotNil(t, m)
	assert.Equal(t, 0, m.Count())
}

func TestMoveToTrash(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.MoveToTrash("item-1", "session", map[string]string{"id": "1"})

	assert.Equal(t, 1, m.Count())
}

func TestRestore(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.MoveToTrash("item-1", "session", "data")

	itemType, data, ok := m.Restore("item-1")
	assert.True(t, ok)
	assert.Equal(t, "session", itemType)
	assert.Equal(t, "data", data)
	assert.Equal(t, 0, m.Count())
}

func TestRestoreNotFound(t *testing.T) {
	t.Parallel()
	m := NewManager()
	_, _, ok := m.Restore("nonexistent")
	assert.False(t, ok)
}

func TestDeletePermanently(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.MoveToTrash("item-1", "session", "data")

	ok := m.DeletePermanently("item-1")
	assert.True(t, ok)
	assert.Equal(t, 0, m.Count())
}

func TestEmptyTrash(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.MoveToTrash("1", "session", "data")
	m.MoveToTrash("2", "session", "data")

	count := m.EmptyTrash()
	assert.Equal(t, 2, count)
	assert.Equal(t, 0, m.Count())
}

func TestList(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.MoveToTrash("1", "session", "data")
	m.MoveToTrash("2", "memory", "data")

	items := m.List()
	assert.Len(t, items, 2)
}

func TestCleanupOld(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.MoveToTrash("old", "session", "data")
	// Manually set old timestamp
	m.mu.Lock()
	m.items["old"].DeletedAt = time.Now().Add(-24 * time.Hour)
	m.mu.Unlock()

	count := m.CleanupOld(1 * time.Hour)
	assert.Equal(t, 1, count)
	assert.Equal(t, 0, m.Count())
}

func TestCleanupOldKeepsRecent(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.MoveToTrash("recent", "session", "data")

	count := m.CleanupOld(1 * time.Hour)
	assert.Equal(t, 0, count)
	assert.Equal(t, 1, m.Count())
}
