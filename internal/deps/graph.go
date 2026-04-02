// Package deps provides dependency graph analysis
package deps

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Graph represents a dependency graph
type Graph struct {
	Nodes map[string]*Node
	Edges []*Edge
}

// Node represents a node in the graph
type Node struct {
	ID    string
	Label string
	Type  string // "internal", "external", "stdlib"
}

// Edge represents a dependency edge
type Edge struct {
	From string
	To   string
}

// NewGraph creates a new graph
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: []*Edge{},
	}
}

// Builder builds dependency graphs
type Builder struct {
	rootPath string
}

// NewBuilder creates a new builder
func NewBuilder(rootPath string) *Builder {
	return &Builder{rootPath: rootPath}
}

// Build builds the dependency graph
func (b *Builder) Build() (*Graph, error) {
	graph := NewGraph()

	err := filepath.Walk(b.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}

		if err := b.parseFile(path, graph); err != nil {
			return err
		}

		return nil
	})

	return graph, err
}

func (b *Builder) parseFile(path string, graph *Graph) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil // Skip unparseable files
	}

	// Get package name
	pkg := filepath.Dir(path)
	relPkg, _ := filepath.Rel(b.rootPath, pkg)

	// Add node for this package
	graph.Nodes[relPkg] = &Node{
		ID:    relPkg,
		Label: f.Name.Name,
		Type:  "internal",
	}

	// Process imports
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var depType string

		if !strings.Contains(path, ".") {
			depType = "stdlib"
		} else if strings.HasPrefix(path, "github.com/yourname/helm") {
			depType = "internal"
		} else {
			depType = "external"
		}

		graph.Nodes[path] = &Node{
			ID:    path,
			Label: filepath.Base(path),
			Type:  depType,
		}

		graph.Edges = append(graph.Edges, &Edge{
			From: relPkg,
			To:   path,
		})
	}

	return nil
}

// CycleDetector detects circular dependencies
type CycleDetector struct {
	graph *Graph
}

// NewCycleDetector creates a cycle detector
func NewCycleDetector(graph *Graph) *CycleDetector {
	return &CycleDetector{graph: graph}
}

// Detect finds cycles
func (cd *CycleDetector) Detect() [][]string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}
	var cycles [][]string

	var dfs func(string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		// Find neighbors
		for _, edge := range cd.graph.Edges {
			if edge.From == node {
				neighbor := edge.To
				if !visited[neighbor] {
					if dfs(neighbor) {
						return true
					}
				} else if recStack[neighbor] {
					// Found cycle
					cycle := extractCycle(path, neighbor)
					cycles = append(cycles, cycle)
				}
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
		return false
	}

	for node := range cd.graph.Nodes {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles
}

func extractCycle(path []string, start string) []string {
	var cycle []string
	started := false
	for _, node := range path {
		if node == start {
			started = true
		}
		if started {
			cycle = append(cycle, node)
		}
	}
	cycle = append(cycle, start)
	return cycle
}

// ExportDOT exports graph to DOT format
func (g *Graph) ExportDOT() string {
	var sb strings.Builder
	sb.WriteString("digraph dependencies {\n")

	// Write nodes
	for id, node := range g.Nodes {
		color := "lightblue"
		if node.Type == "external" {
			color = "lightgreen"
		} else if node.Type == "stdlib" {
			color = "lightgray"
		}
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", style=filled, fillcolor=%s];\n", id, node.Label, color))
	}

	// Write edges
	for _, edge := range g.Edges {
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", edge.From, edge.To))
	}

	sb.WriteString("}\n")
	return sb.String()
}
