// Package automation provides task scheduling and automation capabilities
package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Issue represents a GitHub/GitLab issue
type Issue struct {
	ID          int
	Number      int
	Title       string
	Body        string
	State       string
	Labels      []string
	Assignee    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	URL         string
	Repository  string
	Provider    string // "github" or "gitlab"
}

// IssueFetcher fetches issues from GitHub/GitLab
type IssueFetcher struct {
	httpClient *http.Client
	token      string
}

// NewIssueFetcher creates a new issue fetcher
func NewIssueFetcher(token string) *IssueFetcher {
	return &IssueFetcher{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
	}
}

// FetchGitHubIssues fetches open issues from a GitHub repository
func (f *IssueFetcher) FetchGitHubIssues(ctx context.Context, owner, repo string, labels []string) ([]Issue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=open", owner, repo)
	if len(labels) > 0 {
		url += "&labels=" + labels[0] // Simplified: just first label
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var ghIssues []struct {
		Number    int        `json:"number"`
		Title     string     `json:"title"`
		Body      string     `json:"body"`
		State     string     `json:"state"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Assignee  *struct {
			Login string `json:"login"`
		} `json:"assignee"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		HTMLURL   string    `json:"html_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghIssues); err != nil {
		return nil, err
	}

	var issues []Issue
	for _, gi := range ghIssues {
		issue := Issue{
			Number:     gi.Number,
			Title:      gi.Title,
			Body:       gi.Body,
			State:      gi.State,
			CreatedAt:  gi.CreatedAt,
			UpdatedAt:  gi.UpdatedAt,
			URL:        gi.HTMLURL,
			Repository: fmt.Sprintf("%s/%s", owner, repo),
			Provider:   "github",
		}

		for _, l := range gi.Labels {
			issue.Labels = append(issue.Labels, l.Name)
		}

		if gi.Assignee != nil {
			issue.Assignee = gi.Assignee.Login
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// FetchGitLabIssues fetches open issues from a GitLab repository
func (f *IssueFetcher) FetchGitLabIssues(ctx context.Context, projectID string, labels []string) ([]Issue, error) {
	url := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/issues?state=opened", projectID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", f.token)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}

	var glIssues []struct {
		Iid       int       `json:"iid"`
		Title     string    `json:"title"`
		Description string  `json:"description"`
		State     string    `json:"state"`
		Labels    []string  `json:"labels"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		WebURL    string    `json:"web_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glIssues); err != nil {
		return nil, err
	}

	var issues []Issue
	for _, gi := range glIssues {
		issues = append(issues, Issue{
			Number:     gi.Iid,
			Title:      gi.Title,
			Body:       gi.Description,
			State:      gi.State,
			Labels:     gi.Labels,
			CreatedAt:  gi.CreatedAt,
			UpdatedAt:  gi.UpdatedAt,
			URL:        gi.WebURL,
			Repository: projectID,
			Provider:   "gitlab",
		})
	}

	return issues, nil
}

// FilterByLabel filters issues by label
func FilterByLabel(issues []Issue, label string) []Issue {
	var filtered []Issue
	for _, issue := range issues {
		for _, l := range issue.Labels {
			if l == label {
				filtered = append(filtered, issue)
				break
			}
		}
	}
	return filtered
}

// FilterByPriority filters issues by priority labels
func FilterByPriority(issues []Issue, priority string) []Issue {
	priorityLabels := map[string][]string{
		"high":   {"priority/high", "priority:high", "P0", "P1"},
		"medium": {"priority/medium", "priority:medium", "P2"},
		"low":    {"priority/low", "priority:low", "P3"},
	}

	labels := priorityLabels[priority]
	var filtered []Issue
	for _, issue := range issues {
		for _, l := range issue.Labels {
			for _, pl := range labels {
				if l == pl {
					filtered = append(filtered, issue)
					break
				}
			}
		}
	}
	return filtered
}
