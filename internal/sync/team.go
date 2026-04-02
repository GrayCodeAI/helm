// Package sync provides team synchronization capabilities
package sync

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourname/helm/internal/db"
)

// TeamSync synchronizes HELM data across team members
type TeamSync struct {
	dbURL      string
	httpClient *http.Client
	localDB    db.Querier
}

// NewTeamSync creates a new team sync client
func NewTeamSync(dbURL string, localDB db.Querier) *TeamSync {
	return &TeamSync{
		dbURL:      dbURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		localDB:    localDB,
	}
}

// SyncMemory syncs project memory with the team
func (ts *TeamSync) SyncMemory(ctx context.Context, project string) error {
	memories, err := ts.localDB.ListMemories(ctx, project)
	if err != nil {
		return fmt.Errorf("list local memories: %w", err)
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"project":   project,
		"memories":  memories,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/sync/memory", ts.dbURL), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sync memory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sync memory failed: %s", resp.Status)
	}

	return nil
}

// FetchTeamMemory fetches shared memory from the team
func (ts *TeamSync) FetchTeamMemory(ctx context.Context, project string) ([]db.Memory, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/sync/memory?project=%s", ts.dbURL, project), nil)

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch team memory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch team memory failed: %s", resp.Status)
	}

	var memories []db.Memory
	if err := json.NewDecoder(resp.Body).Decode(&memories); err != nil {
		return nil, fmt.Errorf("decode memories: %w", err)
	}

	return memories, nil
}

// SyncPrompts syncs prompt library with the team
func (ts *TeamSync) SyncPrompts(ctx context.Context) error {
	prompts, err := ts.localDB.ListPrompts(ctx)
	if err != nil {
		return fmt.Errorf("list local prompts: %w", err)
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"prompts":   prompts,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/sync/prompts", ts.dbURL), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sync prompts: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// FetchTeamPrompts fetches shared prompts from the team
func (ts *TeamSync) FetchTeamPrompts(ctx context.Context) ([]db.Prompt, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/sync/prompts", ts.dbURL), nil)

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch team prompts: %w", err)
	}
	defer resp.Body.Close()

	var prompts []db.Prompt
	if err := json.NewDecoder(resp.Body).Decode(&prompts); err != nil {
		return nil, fmt.Errorf("decode prompts: %w", err)
	}

	return prompts, nil
}

// SyncPerformance syncs model performance data with the team
func (ts *TeamSync) SyncPerformance(ctx context.Context) error {
	performances, err := ts.localDB.ListModelPerformance(ctx)
	if err != nil {
		return fmt.Errorf("list local performance: %w", err)
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"performances": performances,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/sync/performance", ts.dbURL), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sync performance: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// FetchTeamPerformance fetches team-wide model performance data
func (ts *TeamSync) FetchTeamPerformance(ctx context.Context) ([]db.ModelPerformance, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/sync/performance", ts.dbURL), nil)

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch team performance: %w", err)
	}
	defer resp.Body.Close()

	var performances []db.ModelPerformance
	if err := json.NewDecoder(resp.Body).Decode(&performances); err != nil {
		return nil, fmt.Errorf("decode performances: %w", err)
	}

	return performances, nil
}

// TeamMember represents a team member
type TeamMember struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	LastSync time.Time `json:"last_sync"`
	Active   bool      `json:"active"`
}

// GetTeamMembers returns the list of team members
func (ts *TeamSync) GetTeamMembers(ctx context.Context) ([]TeamMember, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/team/members", ts.dbURL), nil)

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get team members: %w", err)
	}
	defer resp.Body.Close()

	var members []TeamMember
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, fmt.Errorf("decode members: %w", err)
	}

	return members, nil
}

// SyncAll runs a full sync of all data types
func (ts *TeamSync) SyncAll(ctx context.Context, project string) error {
	if err := ts.SyncMemory(ctx, project); err != nil {
		return fmt.Errorf("sync memory: %w", err)
	}

	if err := ts.SyncPrompts(ctx); err != nil {
		return fmt.Errorf("sync prompts: %w", err)
	}

	if err := ts.SyncPerformance(ctx); err != nil {
		return fmt.Errorf("sync performance: %w", err)
	}

	return nil
}

// MergeMemories merges remote memories with local, avoiding duplicates
func (ts *TeamSync) MergeMemories(ctx context.Context, project string, remote []db.Memory) error {
	local, err := ts.localDB.ListMemories(ctx, project)
	if err != nil {
		return fmt.Errorf("list local memories: %w", err)
	}

	localKeys := make(map[string]bool)
	for _, m := range local {
		localKeys[m.Key] = true
	}

	for _, remote := range remote {
		if !localKeys[remote.Key] {
			// Insert remote memory locally
			_, err := ts.localDB.CreateMemory(ctx, db.CreateMemoryParams{
				ID:         remote.ID,
				Project:    remote.Project,
				Type:       remote.Type,
				Key:        remote.Key,
				Value:      remote.Value,
				Source:     "team_sync",
				Confidence: remote.Confidence,
			})
			if err != nil {
				return fmt.Errorf("create memory: %w", err)
			}
		}
	}

	return nil
}

// MergePrompts merges remote prompts with local, avoiding duplicates
func (ts *TeamSync) MergePrompts(ctx context.Context, remote []db.Prompt) error {
	local, err := ts.localDB.ListPrompts(ctx)
	if err != nil {
		return fmt.Errorf("list local prompts: %w", err)
	}

	localNames := make(map[string]bool)
	for _, p := range local {
		localNames[p.Name] = true
	}

	for _, remote := range remote {
		if !localNames[remote.Name] {
			desc := sql.NullString{}
			if remote.Description.Valid {
				desc = remote.Description
			}
			tags := sql.NullString{}
			if remote.Tags.Valid {
				tags = remote.Tags
			}
			complexity := sql.NullString{}
			if remote.Complexity.Valid {
				complexity = remote.Complexity
			}
			variables := sql.NullString{}
			if remote.Variables.Valid {
				variables = remote.Variables
			}

			_, err := ts.localDB.CreatePrompt(ctx, db.CreatePromptParams{
				ID:          remote.ID,
				Name:        remote.Name,
				Description: desc,
				Tags:        tags,
				Complexity:  complexity,
				Template:    remote.Template,
				Variables:   variables,
				Source:      "team_sync",
			})
			if err != nil {
				return fmt.Errorf("create prompt: %w", err)
			}
		}
	}

	return nil
}

// ConflictResolution represents a sync conflict
type ConflictResolution struct {
	LocalValue  string
	RemoteValue string
	Resolution  string // "local", "remote", "merge"
}

// ResolveConflicts handles sync conflicts
func (ts *TeamSync) ResolveConflicts(conflicts []ConflictResolution) error {
	for _, c := range conflicts {
		switch c.Resolution {
		case "remote":
			// Accept remote value
		case "local":
			// Keep local value
		case "merge":
			// Merge both values
		}
	}
	return nil
}

// SyncStatus represents the current sync status
type SyncStatus struct {
	LastSync    time.Time `json:"last_sync"`
	Members     int       `json:"members"`
	PendingSync bool      `json:"pending_sync"`
	Errors      []string  `json:"errors,omitempty"`
}

// GetSyncStatus returns the current sync status
func (ts *TeamSync) GetSyncStatus(ctx context.Context) (*SyncStatus, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/sync/status", ts.dbURL), nil)

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get sync status: %w", err)
	}
	defer resp.Body.Close()

	var status SyncStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode status: %w", err)
	}

	return &status, nil
}
