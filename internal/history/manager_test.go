package history

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	assert.NotNil(t, m)
	assert.Equal(t, 0, m.Count())
}

func TestAdd(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	id := m.Add("test prompt", "session-1", "anthropic", "claude-sonnet-4")

	assert.NotEmpty(t, id)
	assert.Equal(t, 1, m.Count())
}

func TestGet(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	id := m.Add("test prompt", "session-1", "anthropic", "claude-sonnet-4")

	entry, ok := m.Get(id)
	assert.True(t, ok)
	assert.Equal(t, "test prompt", entry.Prompt)
	assert.Equal(t, "session-1", entry.SessionID)
}

func TestGetNotFound(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	_, ok := m.Get("nonexistent")
	assert.False(t, ok)
}

func TestList(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	m.Add("prompt 1", "s1", "anthropic", "model")
	m.Add("prompt 2", "s2", "openai", "model")

	entries := m.List(10)
	assert.Len(t, entries, 2)
	// Most recent first
	assert.Equal(t, "prompt 2", entries[0].Prompt)
}

func TestListWithLimit(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	for i := 0; i < 10; i++ {
		m.Add("prompt", "s", "p", "m")
	}

	entries := m.List(5)
	assert.Len(t, entries, 5)
}

func TestSearch(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	m.Add("implement feature X", "s1", "anthropic", "model")
	m.Add("fix bug Y", "s2", "openai", "model")
	m.Add("implement feature Z", "s3", "google", "model")

	results := m.Search("implement")
	assert.Len(t, results, 2)
}

func TestSearchNotFound(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	m.Add("test prompt", "s1", "anthropic", "model")

	results := m.Search("nonexistent")
	assert.Empty(t, results)
}

func TestClear(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	m.Add("prompt", "s", "p", "m")
	m.Clear()
	assert.Equal(t, 0, m.Count())
}

func TestMaxSize(t *testing.T) {
	t.Parallel()
	m := NewManager(5)
	for i := 0; i < 10; i++ {
		m.Add("prompt", "s", "p", "m")
	}
	assert.LessOrEqual(t, m.Count(), 5)
}

func TestEntryTimestamp(t *testing.T) {
	t.Parallel()
	m := NewManager(100)
	before := time.Now()
	id := m.Add("test", "s", "p", "m")
	after := time.Now()

	entry, _ := m.Get(id)
	assert.True(t, entry.Timestamp.After(before) || entry.Timestamp.Equal(before))
	assert.True(t, entry.Timestamp.Before(after) || entry.Timestamp.Equal(after))
}
