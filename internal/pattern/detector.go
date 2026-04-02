// Package pattern provides pattern learning capabilities
package pattern

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Detector detects code patterns
type Detector struct {
	repoPath string
}

// NewDetector creates a pattern detector
func NewDetector(repoPath string) *Detector {
	return &Detector{repoPath: repoPath}
}

// Patterns holds detected patterns
type Patterns struct {
	NamingConvention   string
	ErrorHandling      string
	TestStyle          string
	FileOrganization   string
	ImportStyle        string
	CommentStyle       string
	Indentation        string
	LineLength         int
}

// Detect detects patterns from the codebase
func (d *Detector) Detect() (*Patterns, error) {
	p := &Patterns{
		Indentation:  "tabs",
		LineLength:   80,
	}

	// Would walk the repository and analyze code
	// For now, return defaults

	return p, nil
}

// detectNamingConvention detects naming patterns
func (d *Detector) detectNamingConvention(filePath string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return "", err
	}

	camelCount := 0
	snakeCount := 0
	pascalCount := 0

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			name := x.Name.Name
			if isCamelCase(name) {
				camelCount++
			} else if isSnakeCase(name) {
				snakeCount++
			} else if isPascalCase(name) {
				pascalCount++
			}
		case *ast.TypeSpec:
			name := x.Name.Name
			if isPascalCase(name) {
				pascalCount++
			}
		}
		return true
	})

	// Determine convention
	if pascalCount > camelCount && pascalCount > snakeCount {
		return "Go-style (Pascal for exported, camel for unexported)", nil
	}
	if snakeCount > camelCount {
		return "snake_case", nil
	}
	return "camelCase", nil
}

// detectErrorHandling detects error handling style
func (d *Detector) detectErrorHandling(filePath string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return "", err
	}

	returnErr := 0
	panicCount := 0

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ReturnStmt:
			// Check for error returns
			for _, r := range x.Results {
				if ident, ok := r.(*ast.Ident); ok && ident.Name == "err" {
					returnErr++
				}
			}
		case *ast.CallExpr:
			if ident, ok := x.Fun.(*ast.Ident); ok {
				if ident.Name == "panic" {
					panicCount++
				}
			}
		}
		return true
	})

	if panicCount > returnErr {
		return "panic-heavy", nil
	}
	return "error-return", nil
}

// detectTestStyle detects testing patterns
func (d *Detector) detectTestStyle() string {
	// Check for test files
	// Look for table-driven tests vs simple tests
	return "table-driven"
}

func isCamelCase(s string) bool {
	return s != "" && s[0] >= 'a' && s[0] <= 'z'
}

func isPascalCase(s string) bool {
	return s != "" && s[0] >= 'A' && s[0] <= 'Z'
}

func isSnakeCase(s string) bool {
	return strings.Contains(s, "_")
}

// Apply applies patterns to new code
func Apply(patterns *Patterns, code string) string {
	// Would transform code to match patterns
	return code
}

// Convention represents a coding convention
type Convention struct {
	Name        string
	Description string
	Examples    []string
}

// ExtractConventions extracts conventions from code
func ExtractConventions(repoPath string) []Convention {
	return []Convention{
		{
			Name:        "naming",
			Description: "Use camelCase for unexported, PascalCase for exported",
			Examples:    []string{"func doSomething()", "func DoSomething()"},
		},
		{
			Name:        "errors",
			Description: "Return errors, don't panic",
			Examples:    []string{"return err"},
		},
	}
}
