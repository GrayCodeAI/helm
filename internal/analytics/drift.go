// Package analytics provides analytics and reporting capabilities
package analytics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DriftDetector detects when code diverges from documented architecture
type DriftDetector struct {
	repoPath string
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(repoPath string) *DriftDetector {
	return &DriftDetector{repoPath: repoPath}
}

// DriftReport represents architecture drift findings
type DriftReport struct {
	Deviations      []Deviation
	MissingLayers   []string
	PatternViolations []PatternViolation
	DocUpdatesNeeded []string
}

// Deviation represents a deviation from documented architecture
type Deviation struct {
	FilePath      string
	ExpectedLayer string
	ActualLayer   string
	Description   string
	Severity      string // "low", "medium", "high"
}

// PatternViolation represents a pattern violation
type PatternViolation struct {
	Pattern       string
	FilePath      string
	Violation     string
	SuggestedFix  string
}

// DetectDrift detects architecture drift
func (dd *DriftDetector) DetectDrift(ctx context.Context) (*DriftReport, error) {
	report := &DriftReport{
		Deviations:        []Deviation{},
		MissingLayers:     []string{},
		PatternViolations: []PatternViolation{},
		DocUpdatesNeeded:  []string{},
	}

	// Load architecture documentation
	archDocs, err := dd.loadArchitectureDocs()
	if err != nil {
		return nil, fmt.Errorf("load architecture docs: %w", err)
	}

	// Analyze actual code structure
	analyzer := NewArchitectureAnalyzer(dd.repoPath)
	archMap, err := analyzer.Analyze(ctx)
	if err != nil {
		return nil, fmt.Errorf("analyze architecture: %w", err)
	}

	// Compare documented vs actual
	report.Deviations = dd.findDeviations(archDocs, archMap)
	report.MissingLayers = dd.findMissingLayers(archDocs, archMap)
	report.PatternViolations = dd.findPatternViolations(archDocs, archMap)

	return report, nil
}

// ArchitectureDoc represents documented architecture
type ArchitectureDoc struct {
	Layers          []string
	Patterns        []ArchitecturePattern
	Dependencies    map[string][]string
}

// ArchitecturePattern represents a documented pattern
type ArchitecturePattern struct {
	Name        string
	Description string
	AppliesTo   string // glob pattern
}

func (dd *DriftDetector) loadArchitectureDocs() (*ArchitectureDoc, error) {
	doc := &ArchitectureDoc{
		Layers:       []string{},
		Patterns:     []ArchitecturePattern{},
		Dependencies: make(map[string][]string),
	}

	// Try to read ARCHITECTURE.md
	archPath := filepath.Join(dd.repoPath, "ARCHITECTURE.md")
	if _, err := os.Stat(archPath); err == nil {
		content, err := os.ReadFile(archPath)
		if err == nil {
			doc = dd.parseArchitectureDoc(string(content))
		}
	}

	// If no doc found, create defaults
	if len(doc.Layers) == 0 {
		doc.Layers = []string{"handler", "service", "repository", "model"}
	}

	return doc, nil
}

func (dd *DriftDetector) parseArchitectureDoc(content string) *ArchitectureDoc {
	doc := &ArchitectureDoc{
		Layers:       []string{},
		Patterns:     []ArchitecturePattern{},
		Dependencies: make(map[string][]string),
	}

	lines := strings.Split(content, "\n")
	inLayers := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect layers section
		if strings.Contains(strings.ToLower(line), "layers") || strings.Contains(strings.ToLower(line), "architecture") {
			inLayers = true
			continue
		}

		// Parse layer items
		if inLayers && strings.HasPrefix(line, "-") {
			layer := strings.TrimPrefix(line, "-")
			layer = strings.TrimSpace(layer)
			layer = strings.ToLower(strings.Split(layer, " ")[0])
			doc.Layers = append(doc.Layers, layer)
		}

		// End of layers section
		if inLayers && line == "" && len(doc.Layers) > 0 {
			inLayers = false
		}
	}

	return doc
}

func (dd *DriftDetector) findDeviations(doc *ArchitectureDoc, arch *ArchitectureMap) []Deviation {
	var deviations []Deviation

	// Check if files are in unexpected layers
	for _, pkg := range arch.Packages {
		expectedLayer := dd.expectedLayerForPackage(doc, pkg.Path)
		if expectedLayer != "" && pkg.Layer != expectedLayer && pkg.Layer != "other" {
			deviations = append(deviations, Deviation{
				FilePath:      pkg.Path,
				ExpectedLayer: expectedLayer,
				ActualLayer:   pkg.Layer,
				Description:   fmt.Sprintf("Package should be in %s layer but appears to be in %s", expectedLayer, pkg.Layer),
				Severity:      "medium",
			})
		}
	}

	return deviations
}

func (dd *DriftDetector) expectedLayerForPackage(doc *ArchitectureDoc, pkgPath string) string {
	pathLower := strings.ToLower(pkgPath)

	for _, layer := range doc.Layers {
		if strings.Contains(pathLower, layer) {
			return layer
		}
	}

	return ""
}

func (dd *DriftDetector) findMissingLayers(doc *ArchitectureDoc, arch *ArchitectureMap) []string {
	var missing []string

	for _, docLayer := range doc.Layers {
		found := false
		for layer := range arch.Layers {
			if layer == docLayer {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, docLayer)
		}
	}

	return missing
}

func (dd *DriftDetector) findPatternViolations(doc *ArchitectureDoc, arch *ArchitectureMap) []PatternViolation {
	var violations []PatternViolation

	// Check for common pattern violations
	for _, pkg := range arch.Packages {
		// Check for handler depending directly on repository (skipping service)
		if pkg.Layer == "handler" {
			for _, imp := range pkg.Imports {
				if strings.Contains(imp, "repository") || strings.Contains(imp, "repo") {
					violations = append(violations, PatternViolation{
						Pattern:      "Layered Architecture",
						FilePath:     pkg.Path,
						Violation:    "Handler imports repository directly",
						SuggestedFix: "Route through service layer",
					})
				}
			}
		}
	}

	return violations
}

// GenerateDriftAlert generates an alert for significant drift
func (dd *DriftDetector) GenerateDriftAlert(report *DriftReport) string {
	if len(report.Deviations) == 0 && len(report.PatternViolations) == 0 {
		return ""
	}

	alert := "🚨 Architecture Drift Detected\n\n"

	if len(report.Deviations) > 0 {
		alert += fmt.Sprintf("Found %d deviations from documented architecture:\n", len(report.Deviations))
		for _, d := range report.Deviations[:min(3, len(report.Deviations))] {
			alert += fmt.Sprintf("  - %s: %s\n", d.FilePath, d.Description)
		}
		if len(report.Deviations) > 3 {
			alert += fmt.Sprintf("  ... and %d more\n", len(report.Deviations)-3)
		}
		alert += "\n"
	}

	if len(report.PatternViolations) > 0 {
		alert += fmt.Sprintf("Found %d pattern violations:\n", len(report.PatternViolations))
		for _, v := range report.PatternViolations[:min(3, len(report.PatternViolations))] {
			alert += fmt.Sprintf("  - %s: %s\n", v.FilePath, v.Violation)
		}
		if len(report.PatternViolations) > 3 {
			alert += fmt.Sprintf("  ... and %d more\n", len(report.PatternViolations)-3)
		}
	}

	return alert
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
