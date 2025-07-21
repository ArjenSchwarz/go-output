package migrate

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Migrator performs AST-based migration from v1 to v2
type Migrator struct {
	fset           *token.FileSet
	patterns       []Pattern
	transformRules []TransformRule
}

// Pattern represents a v1 usage pattern to detect
type Pattern struct {
	Name        string
	Description string
	Detector    func(ast.Node) bool
	Example     string
}

// TransformRule defines how to transform v1 code to v2
type TransformRule struct {
	Name        string
	Description string
	Transform   func(ast.Node) (ast.Node, error)
	Priority    int
}

// MigrationResult contains the result of a migration operation
type MigrationResult struct {
	OriginalFile    string
	TransformedFile string
	PatternsFound   []string
	RulesApplied    []string
	Errors          []error
	Warnings        []string
}

// New creates a new migrator with default patterns and rules
func New() *Migrator {
	m := &Migrator{
		fset: token.NewFileSet(),
	}

	// Initialize with common v1 patterns
	m.initializePatterns()
	m.initializeTransformRules()

	return m
}

// MigrateFile migrates a single Go file from v1 to v2
func (m *Migrator) MigrateFile(filename string) (*MigrationResult, error) {
	// Read and parse the file
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", filename, err)
	}

	file, err := parser.ParseFile(m.fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing file %s: %w", filename, err)
	}

	result := &MigrationResult{
		OriginalFile: filename,
	}

	// Detect v1 patterns
	result.PatternsFound = m.detectPatterns(file)

	// Apply transformation rules
	transformed, rulesApplied, errs := m.applyTransformRules(file)
	result.RulesApplied = rulesApplied
	result.Errors = errs

	// Generate the transformed code
	var buf strings.Builder
	if err := format.Node(&buf, m.fset, transformed); err != nil {
		return nil, fmt.Errorf("formatting transformed code: %w", err)
	}

	result.TransformedFile = buf.String()

	return result, nil
}

// MigrateDirectory migrates all Go files in a directory
func (m *Migrator) MigrateDirectory(dir string) ([]*MigrationResult, error) {
	var results []*MigrationResult

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			result, migErr := m.MigrateFile(path)
			if migErr != nil {
				return fmt.Errorf("migrating %s: %w", path, migErr)
			}
			results = append(results, result)
		}

		return nil
	})

	return results, err
}

// detectPatterns scans the AST for v1 usage patterns
func (m *Migrator) detectPatterns(file *ast.File) []string {
	var found []string

	ast.Inspect(file, func(n ast.Node) bool {
		for _, pattern := range m.patterns {
			if pattern.Detector(n) {
				found = append(found, pattern.Name)
			}
		}
		return true
	})

	return found
}

// applyTransformRules applies transformation rules to convert v1 to v2
func (m *Migrator) applyTransformRules(file *ast.File) (ast.Node, []string, []error) {
	var rulesApplied []string
	var errors []error

	// Create a copy of the file to transform
	transformed := m.copyNode(file).(*ast.File)

	// Apply rules in priority order
	for _, rule := range m.transformRules {
		applied := false
		ast.Inspect(transformed, func(n ast.Node) bool {
			if newNode, err := rule.Transform(n); err != nil {
				errors = append(errors, fmt.Errorf("applying rule %s: %w", rule.Name, err))
			} else if newNode != n {
				// Replace the node (this is simplified - in practice we'd need more sophisticated replacement)
				applied = true
			}
			return true
		})

		if applied {
			rulesApplied = append(rulesApplied, rule.Name)
		}
	}

	return transformed, rulesApplied, errors
}

// copyNode creates a deep copy of an AST node
func (m *Migrator) copyNode(node ast.Node) ast.Node {
	// This is a simplified implementation
	// In practice, we'd need a more sophisticated deep copy
	return node
}
