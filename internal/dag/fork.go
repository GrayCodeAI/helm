// Package dag provides DAG-based fork detection for session parsing.
package dag

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Node represents a node in the session DAG
type Node struct {
	UUID       string
	ParentUUID string
	Timestamp  time.Time
	Content    string
	Depth      int
}

// DAG represents a directed acyclic graph of session messages
type DAG struct {
	Nodes []*Node
	Roots []*Node
	Forks [][]*Node
	Edges map[string][]string
}

// NewDAG creates a new DAG
func NewDAG() *DAG {
	return &DAG{
		Edges: make(map[string][]string),
	}
}

// AddNode adds a node to the DAG
func (d *DAG) AddNode(uuid, parentUUID string, timestamp time.Time, content string) {
	node := &Node{
		UUID:       uuid,
		ParentUUID: parentUUID,
		Timestamp:  timestamp,
		Content:    content,
	}
	d.Nodes = append(d.Nodes, node)

	if parentUUID != "" {
		d.Edges[parentUUID] = append(d.Edges[parentUUID], uuid)
	}
}

// Build builds the DAG structure
func (d *DAG) Build() {
	// Find roots (nodes with no parent or parent not in nodes)
	uuidSet := make(map[string]bool)
	for _, n := range d.Nodes {
		uuidSet[n.UUID] = true
	}

	for _, n := range d.Nodes {
		if n.ParentUUID == "" || !uuidSet[n.ParentUUID] {
			d.Roots = append(d.Roots, n)
		}
	}

	// Calculate depths
	for _, root := range d.Roots {
		d.calculateDepth(root, 0)
	}

	// Detect forks
	d.detectForks()
}

func (d *DAG) calculateDepth(node *Node, depth int) {
	node.Depth = depth
	children := d.Edges[node.UUID]
	for _, childUUID := range children {
		for _, n := range d.Nodes {
			if n.UUID == childUUID {
				d.calculateDepth(n, depth+1)
				break
			}
		}
	}
}

func (d *DAG) detectForks() {
	// Group nodes by root
	for _, root := range d.Roots {
		fork := d.collectFork(root)
		if len(fork) > 0 {
			d.Forks = append(d.Forks, fork)
		}
	}
}

func (d *DAG) collectFork(root *Node) []*Node {
	var nodes []*Node
	nodes = append(nodes, root)

	queue := []*Node{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		children := d.Edges[current.UUID]
		for _, childUUID := range children {
			for _, n := range d.Nodes {
				if n.UUID == childUUID {
					nodes = append(nodes, n)
					queue = append(queue, n)
					break
				}
			}
		}
	}

	return nodes
}

// DetectForksFromMessages detects forks from a list of messages with uuid/parentUuid metadata
func DetectForksFromMessages(messages []map[string]interface{}, forkThreshold int) [][]map[string]interface{} {
	if len(messages) == 0 {
		return nil
	}

	dag := NewDAG()

	// Build DAG from messages
	for _, msg := range messages {
		uuid, _ := msg["uuid"].(string)
		parentUUID, _ := msg["parentUuid"].(string)
		timestamp, _ := msg["timestamp"].(time.Time)
		content, _ := msg["content"].(string)

		if uuid == "" {
			continue
		}

		dag.AddNode(uuid, parentUUID, timestamp, content)
	}

	dag.Build()

	// Check for forks (multiple roots or large gaps)
	if len(dag.Roots) <= 1 && len(dag.Forks) <= 1 {
		return nil
	}

	// Split into separate conversation branches
	var branches [][]map[string]interface{}
	for _, fork := range dag.Forks {
		var branch []map[string]interface{}
		for _, node := range fork {
			for _, msg := range messages {
				if uuid, _ := msg["uuid"].(string); uuid == node.UUID {
					branch = append(branch, msg)
					break
				}
			}
		}
		if len(branch) > forkThreshold {
			branches = append(branches, branch)
		}
	}

	return branches
}

// DetectSubagents detects subagent sessions from messages
func DetectSubagents(messages []map[string]interface{}) map[string][]map[string]interface{} {
	subagents := make(map[string][]map[string]interface{})

	for _, msg := range messages {
		eventType, _ := msg["type"].(string)
		if strings.Contains(eventType, "subagent") || strings.Contains(eventType, "spawn") {
			parentID, _ := msg["uuid"].(string)
			subagents[parentID] = append(subagents[parentID], msg)
		}
	}

	return subagents
}

// GetParentChildRelationships extracts parent-child session relationships
func GetParentChildRelationships(messages []map[string]interface{}) map[string]string {
	relationships := make(map[string]string)

	for _, msg := range messages {
		uuid, _ := msg["uuid"].(string)
		parentUUID, _ := msg["parentUuid"].(string)

		if uuid != "" && parentUUID != "" {
			relationships[uuid] = parentUUID
		}
	}

	return relationships
}

// SortMessagesByTimestamp sorts messages by timestamp
func SortMessagesByTimestamp(messages []map[string]interface{}) {
	sort.Slice(messages, func(i, j int) bool {
		ti, _ := messages[i]["timestamp"].(time.Time)
		tj, _ := messages[j]["timestamp"].(time.Time)
		return ti.Before(tj)
	})
}

// GetConversationDepth returns the maximum depth of the conversation DAG
func (d *DAG) GetConversationDepth() int {
	maxDepth := 0
	for _, n := range d.Nodes {
		if n.Depth > maxDepth {
			maxDepth = n.Depth
		}
	}
	return maxDepth
}

// GetForkCount returns the number of detected forks
func (d *DAG) GetForkCount() int {
	return len(d.Forks)
}

// GetRootCount returns the number of root nodes
func (d *DAG) GetRootCount() int {
	return len(d.Roots)
}

// HasForks returns true if the DAG has multiple conversation branches
func (d *DAG) HasForks() bool {
	return len(d.Forks) > 1 || len(d.Roots) > 1
}

// GetMainBranch returns the largest conversation branch
func (d *DAG) GetMainBranch() []*Node {
	if len(d.Forks) == 0 {
		return d.Nodes
	}

	mainIdx := 0
	maxLen := len(d.Forks[0])
	for i, fork := range d.Forks {
		if len(fork) > maxLen {
			maxLen = len(fork)
			mainIdx = i
		}
	}

	return d.Forks[mainIdx]
}

// ValidateDAG validates the DAG structure
func (d *DAG) ValidateDAG() error {
	// Check for cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, node := range d.Nodes {
		if !visited[node.UUID] {
			if d.hasCycle(node.UUID, visited, recStack) {
				return fmt.Errorf("cycle detected in DAG")
			}
		}
	}

	return nil
}

func (d *DAG) hasCycle(uuid string, visited, recStack map[string]bool) bool {
	visited[uuid] = true
	recStack[uuid] = true

	for _, childUUID := range d.Edges[uuid] {
		if !visited[childUUID] {
			if d.hasCycle(childUUID, visited, recStack) {
				return true
			}
		} else if recStack[childUUID] {
			return true
		}
	}

	recStack[uuid] = false
	return false
}
