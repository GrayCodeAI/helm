// Package analytics provides analytics and reporting capabilities
package analytics

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ArchitectureAnalyzer analyzes codebase architecture
type ArchitectureAnalyzer struct {
	repoPath string
}

// NewArchitectureAnalyzer creates a new architecture analyzer
func NewArchitectureAnalyzer(repoPath string) *ArchitectureAnalyzer {
	return &ArchitectureAnalyzer{repoPath: repoPath}
}

// ArchitectureMap represents the architecture of a codebase
type ArchitectureMap struct {
	Packages     []Package
	Dependencies map[string][]string
	Layers       map[string][]string
	CyclicDeps   [][]string
}

// Package represents a package in the codebase
type Package struct {
	Path      string
	Name      string
	Files     []File
	Imports   []string
	Layer     string
}

// File represents a source file
type File struct {
	Path       string
	Structs    []string
	Interfaces []string
	Functions  []string
}

// Analyze analyzes the codebase architecture
func (aa *ArchitectureAnalyzer) Analyze(ctx context.Context) (*ArchitectureMap, error) {
	arch := &ArchitectureMap{
		Dependencies: make(map[string][]string),
		Layers:       make(map[string][]string),
	}

	// Walk the repository
	err := filepath.Walk(aa.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, node_modules, etc.
		if info.IsDir() && (info.Name() == "vendor" || info.Name() == "node_modules" || info.Name() == ".git") {
			return filepath.SkipDir
		}

		// Parse Go files
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			pkg, err := aa.parseGoFile(path)
			if err == nil && pkg != nil {
				arch.Packages = append(arch.Packages, *pkg)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk repository: %w", err)
	}

	// Build dependency graph
	aa.buildDependencyGraph(arch)

	// Detect layers
	aa.detectLayers(arch)

	// Detect cycles
	arch.CyclicDeps = aa.detectCycles(arch)

	return arch, nil
}

func (aa *ArchitectureAnalyzer) parseGoFile(path string) (*Package, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	pkg := &Package{
		Path:    filepath.Dir(path),
		Name:    f.Name.Name,
		Files:   []File{{Path: path}},
		Imports: make([]string, 0, len(f.Imports)),
	}

	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		pkg.Imports = append(pkg.Imports, path)
	}

	return pkg, nil
}

func (aa *ArchitectureAnalyzer) buildDependencyGraph(arch *ArchitectureMap) {
	for _, pkg := range arch.Packages {
		arch.Dependencies[pkg.Path] = pkg.Imports
	}
}

func (aa *ArchitectureAnalyzer) detectLayers(arch *ArchitectureMap) {
	// Simple layer detection based on naming conventions
	for _, pkg := range arch.Packages {
		layer := aa.classifyLayer(pkg.Path)
		pkg.Layer = layer
		arch.Layers[layer] = append(arch.Layers[layer], pkg.Path)
	}
}

func (aa *ArchitectureAnalyzer) classifyLayer(path string) string {
	pathLower := strings.ToLower(path)

	if strings.Contains(pathLower, "handler") || strings.Contains(pathLower, "controller") || strings.Contains(pathLower, "api") {
		return "handler"
	}
	if strings.Contains(pathLower, "service") || strings.Contains(pathLower, "usecase") || strings.Contains(pathLower, "business") {
		return "service"
	}
	if strings.Contains(pathLower, "repository") || strings.Contains(pathLower, "store") || strings.Contains(pathLower, "db") {
		return "repository"
	}
	if strings.Contains(pathLower, "model") || strings.Contains(pathLower, "entity") || strings.Contains(pathLower, "domain") {
		return "model"
	}
	if strings.Contains(pathLower, "internal") {
		return "internal"
	}

	return "other"
}

func (aa *ArchitectureAnalyzer) detectCycles(arch *ArchitectureMap) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	var dfs func(string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range arch.Dependencies[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle
				cycleStart := 0
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				cycle := append([]string{}, path[cycleStart:]...)
				cycle = append(cycle, neighbor)
				cycles = append(cycles, cycle)
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
		return false
	}

	for node := range arch.Dependencies {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles
}

// ArchitectureReport represents an architecture analysis report
type ArchitectureReport struct {
	TotalPackages  int
	TotalFiles     int
	LayerCounts    map[string]int
	CyclicDeps     [][]string
	HasCycles      bool
	Recommendations []string
}

// GenerateReport generates an architecture report
func (aa *ArchitectureAnalyzer) GenerateReport(ctx context.Context) (*ArchitectureReport, error) {
	arch, err := aa.Analyze(ctx)
	if err != nil {
		return nil, err
	}

	report := &ArchitectureReport{
		TotalPackages: len(arch.Packages),
		LayerCounts:   make(map[string]int),
		CyclicDeps:    arch.CyclicDeps,
		HasCycles:     len(arch.CyclicDeps) > 0,
	}

	for layer, packages := range arch.Layers {
		report.LayerCounts[layer] = len(packages)
	}

	for _, pkg := range arch.Packages {
		report.TotalFiles += len(pkg.Files)
	}

	report.Recommendations = aa.generateRecommendations(arch)

	return report, nil
}

func (aa *ArchitectureAnalyzer) generateRecommendations(arch *ArchitectureMap) []string {
	var recs []string

	if len(arch.CyclicDeps) > 0 {
		recs = append(recs, fmt.Sprintf("Found %d circular dependencies - consider refactoring", len(arch.CyclicDeps)))
	}

	// Check for missing layers
	if len(arch.Layers["handler"]) > 0 && len(arch.Layers["service"]) == 0 {
		recs = append(recs, "Handlers detected but no service layer - consider adding business logic layer")
	}

	if len(arch.Layers["service"]) > 0 && len(arch.Layers["repository"]) == 0 {
		recs = append(recs, "Service layer detected but no repository layer - data access may be mixed with business logic")
	}

	return recs
}

// ImportGraphBuilder builds an import dependency graph
type ImportGraphBuilder struct {
	repoPath string
}

// NewImportGraphBuilder creates a new import graph builder
func NewImportGraphBuilder(repoPath string) *ImportGraphBuilder {
	return &ImportGraphBuilder{repoPath: repoPath}
}

// Node represents a node in the import graph
type Node struct {
	ID       string
	Label    string
	Group    string
}

// Edge represents an edge in the import graph
type Edge struct {
	From string
	To   string
}

// Graph represents the import graph
type Graph struct {
	Nodes []Node
	Edges []Edge
}

// BuildGraph builds the import graph
func (igb *ImportGraphBuilder) BuildGraph(ctx context.Context) (*Graph, error) {
	graph := &Graph{
		Nodes: []Node{},
		Edges: []Edge{},
	}

	nodeMap := make(map[string]bool)

	err := filepath.Walk(igb.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}

		pkgPath := filepath.Dir(path)
		if !nodeMap[pkgPath] {
			nodeMap[pkgPath] = true
			graph.Nodes = append(graph.Nodes, Node{
				ID:    pkgPath,
				Label: f.Name.Name,
				Group: "internal",
			})
		}

		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			graph.Edges = append(graph.Edges, Edge{
				From: pkgPath,
				To:   impPath,
			})

			if !nodeMap[impPath] && !isStandardLib(impPath) {
				nodeMap[impPath] = true
				graph.Nodes = append(graph.Nodes, Node{
					ID:    impPath,
					Label: filepath.Base(impPath),
					Group: "external",
				})
			}
		}

		return nil
	})

	return graph, err
}

func isStandardLib(path string) bool {
	// Standard library packages don't contain dots
	return !strings.Contains(path, ".")
}
