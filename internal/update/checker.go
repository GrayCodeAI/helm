// Package update provides self-update functionality.
package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Release represents a GitHub release
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a release asset
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// Checker checks for updates
type Checker struct {
	repo        string
	currentVer  string
	lastChecked time.Time
	cache       *Release
}

// NewChecker creates a new update checker
func NewChecker(repo, currentVer string) *Checker {
	return &Checker{
		repo:       repo,
		currentVer: currentVer,
	}
}

// CheckLatest checks for the latest release
func (c *Checker) CheckLatest() (*Release, error) {
	// Check cache (5 minutes)
	if c.cache != nil && time.Since(c.lastChecked) < 5*time.Minute {
		return c.cache, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", c.repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}

	c.cache = &release
	c.lastChecked = time.Now()
	return &release, nil
}

// IsUpdateAvailable checks if an update is available
func (c *Checker) IsUpdateAvailable() (bool, *Release, error) {
	latest, err := c.CheckLatest()
	if err != nil {
		return false, nil, err
	}

	current := strings.TrimPrefix(c.currentVer, "v")
	latestTag := strings.TrimPrefix(latest.TagName, "v")

	return current != latestTag, latest, nil
}

// GetDownloadURL gets the download URL for the current platform
func (r *Release) GetDownloadURL() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	pattern := fmt.Sprintf("helm-%s-%s", goos, goarch)

	for _, asset := range r.Assets {
		if strings.Contains(asset.Name, pattern) {
			return asset.DownloadURL
		}
	}

	return ""
}

// Download downloads the latest release
func Download(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	// In production, would extract and replace binary
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}
