// Package analytics provides analytics and reporting capabilities
package analytics

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// HotspotAnalyzer identifies problematic files
type HotspotAnalyzer struct {
	repoPath string
}

// Hotspot represents a file with high change/error frequency
type Hotspot struct {
	FilePath     string
	ChangeCount  int
	ErrorCount   int
	Complexity   int
	RiskScore    float64
	LastModified string
	Contributors []string
}

// NewHotspotAnalyzer creates a new hotspot analyzer
func NewHotspotAnalyzer(repoPath string) *HotspotAnalyzer {
	return &HotspotAnalyzer{repoPath: repoPath}
}

// Analyze analyzes the repository for hotspots
func (ha *HotspotAnalyzer) Analyze(ctx context.Context, limit int) ([]Hotspot, error) {
	// Get file change frequency from git
	changeFreq, err := ha.getChangeFrequency(ctx)
	if err != nil {
		return nil, fmt.Errorf("get change frequency: %w", err)
	}

	// Get complexity metrics
	complexity, err := ha.getComplexityMetrics(ctx)
	if err != nil {
		// Continue without complexity data
		complexity = make(map[string]int)
	}

	// Build hotspots
	var hotspots []Hotspot
	for file, changes := range changeFreq {
		hotspot := Hotspot{
			FilePath:     file,
			ChangeCount:  changes,
			Complexity:   complexity[file],
			ErrorCount:   ha.estimateErrorCount(file, changes),
			Contributors: ha.getContributors(ctx, file),
		}

		// Calculate risk score
		hotspot.RiskScore = ha.calculateRiskScore(hotspot)

		hotspots = append(hotspots, hotspot)
	}

	// Sort by risk score (descending)
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].RiskScore > hotspots[j].RiskScore
	})

	// Limit results
	if limit > 0 && len(hotspots) > limit {
		hotspots = hotspots[:limit]
	}

	return hotspots, nil
}

func (ha *HotspotAnalyzer) getChangeFrequency(ctx context.Context) (map[string]int, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", ha.repoPath, "log", "--pretty=format:", "--name-only", "--since=90.days")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	freq := make(map[string]int)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "commit") {
			continue
		}
		// Only count source files
		if isSourceFile(line) {
			freq[line]++
		}
	}

	return freq, nil
}

func (ha *HotspotAnalyzer) getComplexityMetrics(ctx context.Context) (map[string]int, error) {
	// This would use a tool like gocyclo for Go, or similar for other languages
	// For now, return empty map
	return make(map[string]int), nil
}

func (ha *HotspotAnalyzer) estimateErrorCount(file string, changes int) int {
	// Simple heuristic: files with many changes are more likely to have errors
	// In a real implementation, this would correlate with test failures, bugs, etc.
	return changes / 5
}

func (ha *HotspotAnalyzer) getContributors(ctx context.Context, file string) []string {
	cmd := exec.CommandContext(ctx, "git", "-C", ha.repoPath, "log", "--pretty=format:%an", "--", file)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	contributorMap := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			contributorMap[line] = true
		}
	}

	var contributors []string
	for c := range contributorMap {
		contributors = append(contributors, c)
	}

	return contributors
}

func (ha *HotspotAnalyzer) calculateRiskScore(hotspot Hotspot) float64 {
	// Risk score formula
	// Higher change count = higher risk
	// Higher error count = higher risk
	// Higher complexity = higher risk
	// More contributors = slightly higher risk (communication overhead)

	changeScore := float64(hotspot.ChangeCount) * 2.0
	errorScore := float64(hotspot.ErrorCount) * 10.0
	complexityScore := float64(hotspot.Complexity) * 0.5
	contributorScore := float64(len(hotspot.Contributors)) * 0.3

	risk := changeScore + errorScore + complexityScore + contributorScore

	// Normalize to 0-100 scale
	if risk > 100 {
		risk = 100
	}

	return risk
}

func isSourceFile(path string) bool {
	ext := filepath.Ext(path)
	sourceExts := []string{".go", ".py", ".js", ".ts", ".java", ".rb", ".rs", ".cpp", ".c", ".h"}
	for _, se := range sourceExts {
		if ext == se {
			return true
		}
	}
	return false
}

// GetRiskLevel returns the risk level for a score
func GetRiskLevel(score float64) string {
	if score >= 70 {
		return "🔴 High"
	}
	if score >= 40 {
		return "🟡 Medium"
	}
	return "🟢 Low"
}

// HotspotReport represents a comprehensive hotspot report
type HotspotReport struct {
	Hotspots        []Hotspot
	HighRiskCount   int
	MediumRiskCount int
	LowRiskCount    int
	TopFiles        []string
	Recommendations []string
}

// GenerateReport generates a comprehensive hotspot report
func (ha *HotspotAnalyzer) GenerateReport(ctx context.Context) (*HotspotReport, error) {
	hotspots, err := ha.Analyze(ctx, 20)
	if err != nil {
		return nil, err
	}

	report := &HotspotReport{
		Hotspots: hotspots,
	}

	for _, h := range hotspots {
		switch GetRiskLevel(h.RiskScore) {
		case "🔴 High":
			report.HighRiskCount++
			report.TopFiles = append(report.TopFiles, h.FilePath)
		case "🟡 Medium":
			report.MediumRiskCount++
		case "🟢 Low":
			report.LowRiskCount++
		}
	}

	report.Recommendations = ha.generateRecommendations(hotspots)

	return report, nil
}

func (ha *HotspotAnalyzer) generateRecommendations(hotspots []Hotspot) []string {
	var recommendations []string

	if len(hotspots) == 0 {
		return recommendations
	}

	topHotspot := hotspots[0]
	if topHotspot.RiskScore >= 70 {
		recommendations = append(recommendations,
			fmt.Sprintf("High-risk file detected: %s - consider refactoring", topHotspot.FilePath),
		)
	}

	if topHotspot.ChangeCount > 20 {
		recommendations = append(recommendations,
			"Consider adding more tests to frequently modified files",
		)
	}

	if len(topHotspot.Contributors) > 5 {
		recommendations = append(recommendations,
			"High-contributor files may benefit from clearer ownership",
		)
	}

	return recommendations
}

// ComplexityAnalyzer calculates code complexity
type ComplexityAnalyzer struct {
	repoPath string
}

// NewComplexityAnalyzer creates a complexity analyzer
func NewComplexityAnalyzer(repoPath string) *ComplexityAnalyzer {
	return &ComplexityAnalyzer{repoPath: repoPath}
}

// FileComplexity represents complexity metrics for a file
type FileComplexity struct {
	FilePath              string
	LinesOfCode           int
	CyclomaticComplexity  int
	CognitiveComplexity   int
	FunctionCount         int
	AverageFunctionLength int
}

// AnalyzeFile analyzes complexity of a single file
func (ca *ComplexityAnalyzer) AnalyzeFile(filePath string) (*FileComplexity, error) {
	// For Go files, we could use gocyclo
	// For now, return basic metrics
	content, err := exec.Command("wc", "-l", filepath.Join(ca.repoPath, filePath)).Output()
	if err != nil {
		return nil, err
	}

	// Parse line count from wc output
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not parse line count")
	}

	lines, _ := strconv.Atoi(matches[1])

	return &FileComplexity{
		FilePath:    filePath,
		LinesOfCode: lines,
	}, nil
}
