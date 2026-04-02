package dag

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDAG(t *testing.T) {
	t.Parallel()
	d := NewDAG()
	assert.NotNil(t, d)
	assert.Empty(t, d.Nodes)
	assert.Empty(t, d.Roots)
	assert.Empty(t, d.Forks)
}

func TestAddNode(t *testing.T) {
	t.Parallel()
	d := NewDAG()
	now := time.Now()

	d.AddNode("uuid-1", "", now, "message 1")
	d.AddNode("uuid-2", "uuid-1", now.Add(time.Second), "message 2")

	assert.Len(t, d.Nodes, 2)
	assert.Len(t, d.Edges["uuid-1"], 1)
}

func TestBuild(t *testing.T) {
	t.Parallel()
	d := NewDAG()
	now := time.Now()

	d.AddNode("root", "", now, "root message")
	d.AddNode("child1", "root", now.Add(time.Second), "child 1")
	d.AddNode("child2", "root", now.Add(2*time.Second), "child 2")

	d.Build()

	assert.Len(t, d.Roots, 1)
	assert.Equal(t, "root", d.Roots[0].UUID)
	assert.Equal(t, 0, d.Roots[0].Depth)
}

func TestDetectForksFromMessages(t *testing.T) {
	t.Parallel()
	messages := []map[string]interface{}{
		{"uuid": "1", "parentUuid": "", "timestamp": time.Now(), "content": "msg1", "type": "message"},
		{"uuid": "2", "parentUuid": "1", "timestamp": time.Now().Add(time.Second), "content": "msg2", "type": "message"},
	}

	branches := DetectForksFromMessages(messages, 1)
	// No forks since all messages are in one chain
	assert.Nil(t, branches)
}

func TestDetectSubagents(t *testing.T) {
	t.Parallel()
	messages := []map[string]interface{}{
		{"uuid": "1", "type": "message"},
		{"uuid": "2", "type": "subagent_spawn"},
		{"uuid": "3", "type": "message"},
	}

	subagents := DetectSubagents(messages)
	assert.Len(t, subagents, 1)
	assert.Contains(t, subagents, "2")
}

func TestGetParentChildRelationships(t *testing.T) {
	t.Parallel()
	messages := []map[string]interface{}{
		{"uuid": "1", "parentUuid": ""},
		{"uuid": "2", "parentUuid": "1"},
		{"uuid": "3", "parentUuid": "2"},
	}

	rels := GetParentChildRelationships(messages)
	assert.Equal(t, "1", rels["2"])
	assert.Equal(t, "2", rels["3"])
	assert.NotContains(t, rels, "1")
}

func TestSortMessagesByTimestamp(t *testing.T) {
	t.Parallel()
	now := time.Now()
	messages := []map[string]interface{}{
		{"timestamp": now.Add(2 * time.Second)},
		{"timestamp": now},
		{"timestamp": now.Add(time.Second)},
	}

	SortMessagesByTimestamp(messages)

	t0, _ := messages[0]["timestamp"].(time.Time)
	t1, _ := messages[1]["timestamp"].(time.Time)
	t2, _ := messages[2]["timestamp"].(time.Time)

	assert.True(t, t0.Before(t1) || t0.Equal(t1))
	assert.True(t, t1.Before(t2) || t1.Equal(t2))
}

func TestHasForks(t *testing.T) {
	t.Parallel()
	d := NewDAG()
	now := time.Now()

	d.AddNode("root", "", now, "root")
	d.AddNode("child1", "root", now.Add(time.Second), "child1")
	d.Build()

	assert.False(t, d.HasForks())
}

func TestGetMainBranch(t *testing.T) {
	t.Parallel()
	d := NewDAG()
	now := time.Now()

	d.AddNode("root", "", now, "root")
	d.AddNode("child", "root", now.Add(time.Second), "child")
	d.Build()

	main := d.GetMainBranch()
	assert.Len(t, main, 2)
}

func TestValidateDAG(t *testing.T) {
	t.Parallel()
	d := NewDAG()
	now := time.Now()

	d.AddNode("1", "", now, "msg1")
	d.AddNode("2", "1", now.Add(time.Second), "msg2")
	d.Build()

	err := d.ValidateDAG()
	assert.NoError(t, err)
}

func TestGetConversationDepth(t *testing.T) {
	t.Parallel()
	d := NewDAG()
	now := time.Now()

	d.AddNode("root", "", now, "root")
	d.AddNode("child", "root", now.Add(time.Second), "child")
	d.AddNode("grandchild", "child", now.Add(2*time.Second), "grandchild")
	d.Build()

	depth := d.GetConversationDepth()
	assert.Equal(t, 2, depth)
}

func TestGetForkCount(t *testing.T) {
	t.Parallel()
	d := NewDAG()
	d.Build()
	assert.Equal(t, 0, d.GetForkCount())
}

func TestGetRootCount(t *testing.T) {
	t.Parallel()
	d := NewDAG()
	now := time.Now()

	d.AddNode("root", "", now, "root")
	d.Build()

	assert.Equal(t, 1, d.GetRootCount())
}
