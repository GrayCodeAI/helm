// Package replay provides session replay functionality
package replay

import (
	"context"
	"fmt"
	"time"

	"github.com/yourname/helm/internal/db"
)

// Replayer replays session history
type Replayer struct {
	querier db.Querier
}

// NewReplayer creates a new session replayer
func NewReplayer(querier db.Querier) *Replayer {
	return &Replayer{querier: querier}
}

// ReplaySession represents a replayable session
type ReplaySession struct {
	Session  db.Session
	Messages []db.Message
	Changes  []db.FileChange
	Current  int // Current position in timeline
}

// TimelineEvent represents an event in the session timeline
type TimelineEvent struct {
	Timestamp time.Time
	Type      string // "message", "file_change", "tool_call"
	Data      interface{}
}

// LoadSession loads a session for replay
func (r *Replayer) LoadSession(ctx context.Context, sessionID string) (*ReplaySession, error) {
	session, err := r.querier.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	messages, err := r.querier.GetMessagesBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	changes, err := r.querier.ListFileChanges(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get file changes: %w", err)
	}

	return &ReplaySession{
		Session:  session,
		Messages: messages,
		Changes:  changes,
		Current:  0,
	}, nil
}

// BuildTimeline builds a chronological timeline of events
func (r *Replayer) BuildTimeline(replaySession *ReplaySession) []TimelineEvent {
	var events []TimelineEvent

	// Add messages
	for _, msg := range replaySession.Messages {
		ts, _ := time.Parse(time.RFC3339, msg.Timestamp)
		events = append(events, TimelineEvent{
			Timestamp: ts,
			Type:      "message",
			Data:      msg,
		})
	}

	// Add file changes
	for _, change := range replaySession.Changes {
		ts, _ := time.Parse(time.RFC3339, change.CreatedAt)
		events = append(events, TimelineEvent{
			Timestamp: ts,
			Type:      "file_change",
			Data:      change,
		})
	}

	// Sort by timestamp
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].Timestamp.Before(events[i].Timestamp) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	return events
}

// Next advances to the next event
func (r *Replayer) Next(replaySession *ReplaySession) *TimelineEvent {
	events := r.BuildTimeline(replaySession)
	if replaySession.Current >= len(events)-1 {
		return nil
	}
	replaySession.Current++
	return &events[replaySession.Current]
}

// Previous goes back to the previous event
func (r *Replayer) Previous(replaySession *ReplaySession) *TimelineEvent {
	events := r.BuildTimeline(replaySession)
	if replaySession.Current <= 0 {
		return nil
	}
	replaySession.Current--
	return &events[replaySession.Current]
}

// JumpTo jumps to a specific position
func (r *Replayer) JumpTo(replaySession *ReplaySession, position int) *TimelineEvent {
	events := r.BuildTimeline(replaySession)
	if position < 0 || position >= len(events) {
		return nil
	}
	replaySession.Current = position
	return &events[position]
}

// GetStateAt returns the file state at a specific point in time
func (r *Replayer) GetStateAt(replaySession *ReplaySession, position int) map[string]string {
	state := make(map[string]string)

	changes := r.getChangesUpTo(replaySession, position)
	for _, change := range changes {
		// Reconstruct file state from changes
		if change.FilePath != "" {
			state[change.FilePath] = fmt.Sprintf("+%d -%d", change.Additions.Int64, change.Deletions.Int64)
		}
	}

	return state
}

func (r *Replayer) getChangesUpTo(replaySession *ReplaySession, position int) []db.FileChange {
	var changes []db.FileChange

	events := r.BuildTimeline(replaySession)
	for i, event := range events {
		if i > position {
			break
		}
		if event.Type == "file_change" {
			if change, ok := event.Data.(db.FileChange); ok {
				changes = append(changes, change)
			}
		}
	}

	return changes
}

// ReplayOptions configures replay behavior
type ReplayOptions struct {
	Speed     float64 // 1.0 = normal, 2.0 = 2x, etc.
	AutoPlay  bool
	ShowDiffs bool
	ShowTools bool
}

// DefaultReplayOptions returns default options
func DefaultReplayOptions() ReplayOptions {
	return ReplayOptions{
		Speed:     1.0,
		AutoPlay:  false,
		ShowDiffs: true,
		ShowTools: true,
	}
}

// ReplayController controls the replay playback
type ReplayController struct {
	replayer *Replayer
	session  *ReplaySession
	options  ReplayOptions
	playing  bool
	pauseCh  chan bool
}

// NewReplayController creates a new replay controller
func NewReplayController(replayer *Replayer, session *ReplaySession, options ReplayOptions) *ReplayController {
	return &ReplayController{
		replayer: replayer,
		session:  session,
		options:  options,
		pauseCh:  make(chan bool),
	}
}

// Play starts auto-playback
func (rc *ReplayController) Play() {
	rc.playing = true
	// Playback would happen in a goroutine with timing based on Speed
}

// Pause pauses playback
func (rc *ReplayController) Pause() {
	rc.playing = false
	rc.pauseCh <- true
}

// IsPlaying returns if currently playing
func (rc *ReplayController) IsPlaying() bool {
	return rc.playing
}

// ExportSession exports session to various formats
func (r *Replayer) ExportSession(ctx context.Context, sessionID string, format string) (string, error) {
	session, err := r.LoadSession(ctx, sessionID)
	if err != nil {
		return "", err
	}

	switch format {
	case "json":
		return r.exportJSON(session)
	case "markdown":
		return r.exportMarkdown(session)
	case "html":
		return r.exportHTML(session)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func (r *Replayer) exportJSON(session *ReplaySession) (string, error) {
	// Would marshal to JSON
	return "{}", nil
}

func (r *Replayer) exportMarkdown(session *ReplaySession) (string, error) {
	// Would format as markdown
	return "# Session Replay\n", nil
}

func (r *Replayer) exportHTML(session *ReplaySession) (string, error) {
	// Would format as HTML
	return "<html></html>", nil
}
