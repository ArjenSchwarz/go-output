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
	storedKeys     ast.Expr // Temporary storage for Keys during transformation
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

	// Apply rules in priority order
	for _, rule := range m.transformRules {
		rewriter := &astRewriter{
			transform: rule.Transform,
			applied:   false,
		}

		result := rewriter.rewrite(file)

		if rewriter.err != nil {
			errors = append(errors, fmt.Errorf("applying rule %s: %w", rule.Name, rewriter.err))
		} else if rewriter.applied {
			rulesApplied = append(rulesApplied, rule.Name)
			file = result.(*ast.File)
		}
	}

	return file, rulesApplied, errors
}

// astRewriter implements AST rewriting with proper node replacement
type astRewriter struct {
	transform func(ast.Node) (ast.Node, error)
	applied   bool
	err       error
}

// rewrite recursively rewrites the AST, replacing nodes as needed
func (r *astRewriter) rewrite(node ast.Node) ast.Node {
	if node == nil || r.err != nil {
		return node
	}

	// First, try to transform this node
	newNode, err := r.transform(node)
	if err != nil {
		r.err = err
		return node
	}

	if newNode != node {
		r.applied = true
		// If the node was transformed, we still need to rewrite its children
		node = newNode
	}

	// Rewrite children based on node type
	switch n := node.(type) {
	case *ast.File:
		for i, decl := range n.Decls {
			n.Decls[i] = r.rewrite(decl).(ast.Decl)
		}
	case *ast.FuncDecl:
		if n.Body != nil {
			n.Body = r.rewrite(n.Body).(*ast.BlockStmt)
		}
	case *ast.BlockStmt:
		for i, stmt := range n.List {
			n.List[i] = r.rewrite(stmt).(ast.Stmt)
		}
	case *ast.AssignStmt:
		for i, expr := range n.Lhs {
			n.Lhs[i] = r.rewrite(expr).(ast.Expr)
		}
		for i, expr := range n.Rhs {
			n.Rhs[i] = r.rewrite(expr).(ast.Expr)
		}
	case *ast.ExprStmt:
		n.X = r.rewrite(n.X).(ast.Expr)
	case *ast.CallExpr:
		n.Fun = r.rewrite(n.Fun).(ast.Expr)
		for i, arg := range n.Args {
			n.Args[i] = r.rewrite(arg).(ast.Expr)
		}
	case *ast.CompositeLit:
		if n.Type != nil {
			n.Type = r.rewrite(n.Type).(ast.Expr)
		}
		for i, elt := range n.Elts {
			n.Elts[i] = r.rewrite(elt).(ast.Expr)
		}
	case *ast.SelectorExpr:
		n.X = r.rewrite(n.X).(ast.Expr)
	case *ast.ValueSpec:
		if n.Type != nil {
			n.Type = r.rewrite(n.Type).(ast.Expr)
		}
		for i, value := range n.Values {
			n.Values[i] = r.rewrite(value).(ast.Expr)
		}
	case *ast.GenDecl:
		for i, spec := range n.Specs {
			n.Specs[i] = r.rewrite(spec).(ast.Spec)
		}
	case *ast.IfStmt:
		if n.Init != nil {
			n.Init = r.rewrite(n.Init).(ast.Stmt)
		}
		if n.Cond != nil {
			n.Cond = r.rewrite(n.Cond).(ast.Expr)
		}
		if n.Body != nil {
			n.Body = r.rewrite(n.Body).(*ast.BlockStmt)
		}
		if n.Else != nil {
			n.Else = r.rewrite(n.Else).(ast.Stmt)
		}
	case *ast.ReturnStmt:
		for i, expr := range n.Results {
			n.Results[i] = r.rewrite(expr).(ast.Expr)
		}
	case *ast.DeclStmt:
		n.Decl = r.rewrite(n.Decl).(ast.Decl)
	case *ast.UnaryExpr:
		n.X = r.rewrite(n.X).(ast.Expr)
	case *ast.KeyValueExpr:
		n.Key = r.rewrite(n.Key).(ast.Expr)
		n.Value = r.rewrite(n.Value).(ast.Expr)
	case *ast.ArrayType:
		if n.Elt != nil {
			n.Elt = r.rewrite(n.Elt).(ast.Expr)
		}
	case *ast.StarExpr:
		n.X = r.rewrite(n.X).(ast.Expr)
	}

	return node
}
