// Package automation provides task scheduling and automation capabilities
package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CIWatcher monitors CI/CD status
type CIWatcher struct {
	httpClient *http.Client
	token      string
	provider   string // "github" or "gitlab"
}

// CIStatus represents the status of a CI run
type CIStatus struct {
	ID        string
	Branch    string
	Commit    string
	State     string // "pending", "success", "failure", "error"
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewCIWatcher creates a new CI watcher
func NewCIWatcher(token, provider string) *CIWatcher {
	return &CIWatcher{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		provider:   provider,
	}
}

// WatchGitHubActions watches GitHub Actions status
func (w *CIWatcher) WatchGitHubActions(ctx context.Context, owner, repo, branch string) ([]CIStatus, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs?branch=%s", owner, repo, branch)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+w.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var result struct {
		WorkflowRuns []struct {
			ID         int64     `json:"id"`
			Name       string    `json:"name"`
			HeadBranch string    `json:"head_branch"`
			HeadSha    string    `json:"head_sha"`
			Status     string    `json:"status"`
			Conclusion string    `json:"conclusion"`
			HTMLURL    string    `json:"html_url"`
			CreatedAt  time.Time `json:"created_at"`
			UpdatedAt  time.Time `json:"updated_at"`
		} `json:"workflow_runs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var statuses []CIStatus
	for _, run := range result.WorkflowRuns {
		state := run.Status
		if run.Conclusion != "" {
			state = run.Conclusion
		}

		statuses = append(statuses, CIStatus{
			ID:        fmt.Sprintf("%d", run.ID),
			Branch:    run.HeadBranch,
			Commit:    run.HeadSha,
			State:     state,
			URL:       run.HTMLURL,
			CreatedAt: run.CreatedAt,
			UpdatedAt: run.UpdatedAt,
		})
	}

	return statuses, nil
}

// IsFailure checks if CI status is a failure
func (s *CIStatus) IsFailure() bool {
	return s.State == "failure" || s.State == "error"
}

// IsSuccess checks if CI status is successful
func (s *CIStatus) IsSuccess() bool {
	return s.State == "success"
}

// IsPending checks if CI is still running
func (s *CIStatus) IsPending() bool {
	return s.State == "pending" || s.State == "queued" || s.State == "in_progress"
}
