// Package session provides session replay capabilities
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yourname/helm/internal/db"
)

// SessionReplay provides session replay functionality
type SessionReplay struct {
	querier db.Querier
}

// NewSessionReplay creates a new session replay
func NewSessionReplay(querier db.Querier) *SessionReplay {
	return &SessionReplay{querier: querier}
}

// ReplaySession represents a replayable session
type ReplaySession struct {
	SessionID   string
	Session     *Session
	Messages    []ReplayMessage
	FileStates  map[int]FileState // turn index -> file state
	CurrentTurn int
	TotalTurns  int
}

// ReplayMessage represents a message in the replay
type ReplayMessage struct {
	Turn      int
	Timestamp time.Time
	Role      string
	Content   string
	ToolCalls []ToolCall
}

// FileState represents the state of files at a point in time
type FileState struct {
	Timestamp time.Time
	Files     map[string]string // path -> content hash
}

// LoadSession loads a session for replay
func (sr *SessionReplay) LoadSession(ctx context.Context, sessionID string) (*ReplaySession, error) {
	session, err := sr.querier.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	messages, err := sr.querier.GetMessagesBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	replay := &ReplaySession{
		SessionID:  sessionID,
		Session:    fromDBSession(session),
		Messages:   make([]ReplayMessage, len(messages)),
		FileStates: make(map[int]FileState),
		TotalTurns: len(messages),
	}

	for i, msg := range messages {
		var toolCalls []ToolCall
		if msg.ToolCalls.Valid {
			if err := json.Unmarshal([]byte(msg.ToolCalls.String), &toolCalls); err != nil {
				// Silently skip invalid tool calls - they're not critical
			}
		}

		replay.Messages[i] = ReplayMessage{
			Turn:      i,
			Timestamp: parseTimestamp(msg.Timestamp),
			Role:      msg.Role,
			Content:   msg.Content,
			ToolCalls: toolCalls,
		}
	}

	return replay, nil
}

// Play starts playback from current position
func (sr *SessionReplay) Play(replay *ReplaySession, speed float64) chan ReplayMessage {
	output := make(chan ReplayMessage)

	go func() {
		defer close(output)

		for replay.CurrentTurn < replay.TotalTurns {
			msg := replay.Messages[replay.CurrentTurn]
			output <- msg
			replay.CurrentTurn++

			// Simulate delay based on timestamp differences
			if replay.CurrentTurn < replay.TotalTurns {
				nextMsg := replay.Messages[replay.CurrentTurn]
				delay := nextMsg.Timestamp.Sub(msg.Timestamp)
				if delay > 5*time.Second {
					delay = 5 * time.Second
				}
				time.Sleep(time.Duration(float64(delay) / speed))
			}
		}
	}()

	return output
}

// Pause pauses playback
func (sr *SessionReplay) Pause() {
	// Implementation would use a control channel
}

// StepForward advances one turn
func (sr *SessionReplay) StepForward(replay *ReplaySession) *ReplayMessage {
	if replay.CurrentTurn < replay.TotalTurns {
		msg := replay.Messages[replay.CurrentTurn]
		replay.CurrentTurn++
		return &msg
	}
	return nil
}

// StepBack goes back one turn
func (sr *SessionReplay) StepBack(replay *ReplaySession) *ReplayMessage {
	if replay.CurrentTurn > 0 {
		replay.CurrentTurn--
		msg := replay.Messages[replay.CurrentTurn]
		return &msg
	}
	return nil
}

// JumpTo jumps to a specific turn
func (sr *SessionReplay) JumpTo(replaySession *ReplaySession, turn int) *ReplayMessage {
	if turn >= 0 && turn < replaySession.TotalTurns {
		replaySession.CurrentTurn = turn
		return &replaySession.Messages[turn]
	}
	return nil
}

// GetFileState gets file state at a specific turn
func (sr *SessionReplay) GetFileState(replaySession *ReplaySession, turn int) (*FileState, error) {
	if state, exists := replaySession.FileStates[turn]; exists {
		return &state, nil
	}
	return nil, fmt.Errorf("no file state for turn %d", turn)
}

// Export exports replay to JSON
func (sr *SessionReplay) Export(replaySession *ReplaySession) ([]byte, error) {
	return json.MarshalIndent(replaySession, "", "  ")
}

// TimelineEntry represents an entry in the session timeline
type TimelineEntry struct {
	Turn      int
	Timestamp time.Time
	Type      string // "message", "tool_call", "file_change"
	Summary   string
}

// GenerateTimeline generates a session timeline
func (sr *SessionReplay) GenerateTimeline(replaySession *ReplaySession) []TimelineEntry {
	var timeline []TimelineEntry

	for i, msg := range replaySession.Messages {
		entry := TimelineEntry{
			Turn:      i,
			Timestamp: msg.Timestamp,
			Type:      "message",
			Summary:   truncateReplay(msg.Content, 50),
		}

		if len(msg.ToolCalls) > 0 {
			entry.Type = "tool_call"
			entry.Summary = fmt.Sprintf("Tool calls: %d", len(msg.ToolCalls))
		}

		timeline = append(timeline, entry)
	}

	return timeline
}

func truncateReplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func parseTimestamp(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return t
}

func fromDBSession(s db.Session) *Session {
	return &Session{
		ID:       s.ID,
		Provider: s.Provider,
		Model:    s.Model,
		Project:  s.Project,
		Status:   s.Status,
		Cost:     s.Cost,
	}
}
