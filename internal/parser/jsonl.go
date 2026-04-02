// Package parser provides JSONL session parsing with DAG/subagent detection.
package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// JSONLMessage represents a raw JSONL message from Claude Code
type JSONLMessage struct {
	Type       string                 `json:"type"`
	UUID       string                 `json:"uuid"`
	ParentUUID string                 `json:"parent_uuid"`
	Role       string                 `json:"role"`
	Content    string                 `json:"content"`
	Model      string                 `json:"model"`
	Timestamp  string                 `json:"timestamp"`
	ToolCalls  []JSONLToolCall        `json:"tool_calls,omitempty"`
	Thinking   string                 `json:"thinking,omitempty"`
	Usage      *JSONLUsage            `json:"usage,omitempty"`
	Subagent   *JSONLSubagent         `json:"subagent,omitempty"`
	Extra      map[string]interface{} `json:"-"`
}

// JSONLToolCall represents a tool call in a JSONL message
type JSONLToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

// JSONLUsage represents token usage
type JSONLUsage struct {
	InputTokens      int64 `json:"input_tokens"`
	OutputTokens     int64 `json:"output_tokens"`
	CacheReadTokens  int64 `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int64 `json:"cache_write_tokens,omitempty"`
}

// JSONLSubagent represents a subagent spawn event
type JSONLSubagent struct {
	ParentSessionID string `json:"parent_session_id"`
	SubagentID      string `json:"subagent_id"`
	Status          string `json:"status"` // spawned, completed, failed
}

// ParsedSession represents a fully parsed session
type ParsedSession struct {
	SessionID        string
	Provider         string
	Model            string
	Status           string
	Prompt           string
	Messages         []JSONLMessage
	ToolCalls        []JSONLToolCall
	ThinkingBlocks   []string
	Subagents        []JSONLSubagent
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	Cost             float64
	StartedAt        time.Time
	EndedAt          time.Time
	DAG              *DAGStructure
}

// DAGStructure represents the session's message DAG
type DAGStructure struct {
	Roots    []string
	Branches map[string][]string
	Forks    [][]string
	MaxDepth int
}

// ParseJSONL parses a JSONL file into a ParsedSession
func ParseJSONL(filePath string) (*ParsedSession, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	session := &ParsedSession{
		DAG: &DAGStructure{
			Branches: make(map[string][]string),
		},
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB buffer

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg JSONLMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Skip malformed lines
			continue
		}

		// Extract extra fields
		msg.Extra = make(map[string]interface{})
		json.Unmarshal([]byte(line), &msg.Extra)

		session.Messages = append(session.Messages, msg)

		// Process message
		session.processMessage(msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Build DAG
	session.buildDAG()

	// Detect forks
	session.detectForks()

	return session, nil
}

func (s *ParsedSession) processMessage(msg JSONLMessage) {
	// Track model
	if msg.Model != "" {
		s.Model = msg.Model
	}

	// Track timestamps
	if msg.Timestamp != "" {
		t, err := time.Parse(time.RFC3339, msg.Timestamp)
		if err == nil {
			if s.StartedAt.IsZero() || t.Before(s.StartedAt) {
				s.StartedAt = t
			}
			if s.EndedAt.IsZero() || t.After(s.EndedAt) {
				s.EndedAt = t
			}
		}
	}

	// Track usage
	if msg.Usage != nil {
		s.InputTokens += msg.Usage.InputTokens
		s.OutputTokens += msg.Usage.OutputTokens
		s.CacheReadTokens += msg.Usage.CacheReadTokens
		s.CacheWriteTokens += msg.Usage.CacheWriteTokens
	}

	// Track tool calls
	for _, tc := range msg.ToolCalls {
		s.ToolCalls = append(s.ToolCalls, tc)
	}

	// Track thinking blocks
	if msg.Thinking != "" {
		s.ThinkingBlocks = append(s.ThinkingBlocks, msg.Thinking)
	}

	// Track subagents
	if msg.Subagent != nil {
		s.Subagents = append(s.Subagents, *msg.Subagent)
	}

	// Detect prompt (first user message)
	if msg.Role == "user" && s.Prompt == "" {
		s.Prompt = msg.Content
	}

	// Detect status from message types
	if msg.Type == "session_end" || msg.Type == "result" {
		s.Status = "done"
	} else if msg.Type == "error" {
		s.Status = "failed"
	} else if msg.Type == "session_start" {
		s.Status = "running"
	}
}

func (s *ParsedSession) buildDAG() {
	// Build parent-child relationships
	for _, msg := range s.Messages {
		if msg.UUID == "" {
			continue
		}

		if msg.ParentUUID == "" {
			s.DAG.Roots = append(s.DAG.Roots, msg.UUID)
		} else {
			s.DAG.Branches[msg.ParentUUID] = append(s.DAG.Branches[msg.ParentUUID], msg.UUID)
		}
	}

	// Calculate max depth
	s.DAG.MaxDepth = s.calculateMaxDepth()
}

func (s *ParsedSession) calculateMaxDepth() int {
	maxDepth := 0
	for _, root := range s.DAG.Roots {
		depth := s.depthFromNode(root, make(map[string]bool))
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func (s *ParsedSession) depthFromNode(uuid string, visited map[string]bool) int {
	if visited[uuid] {
		return 0
	}
	visited[uuid] = true

	children := s.DAG.Branches[uuid]
	if len(children) == 0 {
		return 1
	}

	maxChildDepth := 0
	for _, child := range children {
		d := s.depthFromNode(child, visited)
		if d > maxChildDepth {
			maxChildDepth = d
		}
	}

	return 1 + maxChildDepth
}

func (s *ParsedSession) detectForks() {
	// A fork occurs when a message has multiple children (branching conversation)
	for parent, children := range s.DAG.Branches {
		if len(children) > 1 {
			// This is a fork point
			fork := []string{parent}
			fork = append(fork, children...)
			s.DAG.Forks = append(s.DAG.Forks, fork)
		}
	}
}

// GetMainBranch returns the longest conversation branch
func (s *ParsedSession) GetMainBranch() []JSONLMessage {
	if len(s.DAG.Roots) == 0 {
		return s.Messages
	}

	// Follow the longest path from the first root
	root := s.DAG.Roots[0]
	var branch []JSONLMessage

	current := root
	for current != "" {
		// Find message with this UUID
		for _, msg := range s.Messages {
			if msg.UUID == current {
				branch = append(branch, msg)
				break
			}
		}

		// Move to first child
		children := s.DAG.Branches[current]
		if len(children) > 0 {
			current = children[0]
		} else {
			break
		}
	}

	return branch
}

// GetSubagentSessions returns subagent session IDs
func (s *ParsedSession) GetSubagentSessions() []string {
	var ids []string
	seen := make(map[string]bool)
	for _, sub := range s.Subagents {
		if !seen[sub.SubagentID] {
			seen[sub.SubagentID] = true
			ids = append(ids, sub.SubagentID)
		}
	}
	return ids
}

// GetThinkingSummary returns a summary of thinking blocks
func (s *ParsedSession) GetThinkingSummary() string {
	if len(s.ThinkingBlocks) == 0 {
		return ""
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Thinking blocks: %d\n", len(s.ThinkingBlocks)))
	for i, block := range s.ThinkingBlocks {
		if len(block) > 200 {
			block = block[:200] + "..."
		}
		summary.WriteString(fmt.Sprintf("Block %d: %s\n", i+1, block))
	}
	return summary.String()
}

// GetToolCallSummary returns a summary of tool calls
func (s *ParsedSession) GetToolCallSummary() string {
	toolCount := make(map[string]int)
	for _, tc := range s.ToolCalls {
		toolCount[tc.Name]++
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Tool calls: %d total\n", len(s.ToolCalls)))
	for name, count := range toolCount {
		summary.WriteString(fmt.Sprintf("  %s: %d\n", name, count))
	}
	return summary.String()
}
