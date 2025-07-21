package migrate

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// initializeTransformRules sets up transformation rules for v1 to v2 conversion
func (m *Migrator) initializeTransformRules() {
	m.transformRules = []TransformRule{
		{
			Name:        "UpdateImportStatement",
			Description: "Updates v1 import to v2 import",
			Transform:   m.transformImportStatement,
			Priority:    1, // Highest priority - do imports first
		},
		{
			Name:        "ReplaceOutputArrayDeclaration",
			Description: "Replaces OutputArray declarations with Builder pattern",
			Transform:   m.transformOutputArrayDeclaration,
			Priority:    2,
		},
		{
			Name:        "ReplaceOutputSettings",
			Description: "Replaces OutputSettings with functional options",
			Transform:   m.transformOutputSettings,
			Priority:    3,
		},
		{
			Name:        "ReplaceAddContents",
			Description: "Replaces AddContents calls with Table/Text methods",
			Transform:   m.transformAddContents,
			Priority:    4,
		},
		{
			Name:        "ReplaceAddToBuffer",
			Description: "Removes AddToBuffer calls (handled by builder pattern)",
			Transform:   m.transformAddToBuffer,
			Priority:    5,
		},
		{
			Name:        "ReplaceWriteCall",
			Description: "Replaces Write() with Build() and Render()",
			Transform:   m.transformWriteCall,
			Priority:    6,
		},
		{
			Name:        "ReplaceKeysAssignment",
			Description: "Converts Keys assignments to WithKeys() options",
			Transform:   m.transformKeysAssignment,
			Priority:    7,
		},
		{
			Name:        "ReplaceAddHeader",
			Description: "Replaces AddHeader with Header() method",
			Transform:   m.transformAddHeader,
			Priority:    8,
		},
		{
			Name:        "ReplaceProgressCreation",
			Description: "Updates progress creation to v2 API",
			Transform:   m.transformProgressCreation,
			Priority:    9,
		},
	}
}

// transformImportStatement updates v1 imports to v2
func (m *Migrator) transformImportStatement(n ast.Node) (ast.Node, error) {
	importSpec, ok := n.(*ast.ImportSpec)
	if !ok {
		return n, nil
	}

	if importSpec.Path != nil {
		path := strings.Trim(importSpec.Path.Value, "\"")
		if strings.Contains(path, "github.com/ArjenSchwarz/go-output") && !strings.Contains(path, "/v2") {
			// Update to v2 import
			newPath := strings.Replace(path, "github.com/ArjenSchwarz/go-output", "github.com/ArjenSchwarz/go-output/v2", 1)
			importSpec.Path.Value = fmt.Sprintf("\"%s\"", newPath)
		}
	}

	return importSpec, nil
}

// transformOutputArrayDeclaration replaces OutputArray with Builder pattern
func (m *Migrator) transformOutputArrayDeclaration(n ast.Node) (ast.Node, error) {
	switch node := n.(type) {
	case *ast.CompositeLit:
		if selectorExpr, ok := node.Type.(*ast.SelectorExpr); ok {
			if selectorExpr.Sel.Name == "OutputArray" {
				// Replace with output.New() call
				return &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "output"},
						Sel: &ast.Ident{Name: "New"},
					},
				}, nil
			}
		}
	case *ast.ValueSpec:
		// Handle variable declarations like: var output *format.OutputArray
		for _ = range node.Names {
			if selectorExpr, ok := node.Type.(*ast.SelectorExpr); ok {
				if selectorExpr.Sel.Name == "OutputArray" {
					// Change type to *output.Builder
					node.Type = &ast.StarExpr{
						X: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "output"},
							Sel: &ast.Ident{Name: "Builder"},
						},
					}
				}
			}
		}
	}

	return n, nil
}

// transformOutputSettings replaces OutputSettings with functional options
func (m *Migrator) transformOutputSettings(n ast.Node) (ast.Node, error) {
	callExpr, ok := n.(*ast.CallExpr)
	if !ok {
		return n, nil
	}

	if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if selectorExpr.Sel.Name == "NewOutputSettings" {
			// Replace with a comment explaining the migration
			return &ast.ExprStmt{
				X: &ast.BasicLit{
					Kind:  token.STRING,
					Value: "// TODO: Replace OutputSettings with functional options in Output.NewOutput()",
				},
			}, nil
		}
	}

	return n, nil
}

// transformAddContents replaces AddContents calls with builder methods
func (m *Migrator) transformAddContents(n ast.Node) (ast.Node, error) {
	callExpr, ok := n.(*ast.CallExpr)
	if !ok {
		return n, nil
	}

	if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if selectorExpr.Sel.Name == "AddContents" {
			// Replace with Table() method call
			selectorExpr.Sel.Name = "Table"

			// Add empty title as first argument
			newArgs := []ast.Expr{
				&ast.BasicLit{Kind: token.STRING, Value: "\"\""}, // Empty title
			}
			newArgs = append(newArgs, callExpr.Args...)
			callExpr.Args = newArgs
		}
	}

	return callExpr, nil
}

// transformAddToBuffer removes AddToBuffer calls
func (m *Migrator) transformAddToBuffer(n ast.Node) (ast.Node, error) {
	callExpr, ok := n.(*ast.CallExpr)
	if !ok {
		return n, nil
	}

	if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if selectorExpr.Sel.Name == "AddToBuffer" {
			// Return an empty statement (effectively removing the call)
			return &ast.EmptyStmt{}, nil
		}
	}

	return n, nil
}

// transformWriteCall replaces Write() with Build() and Render()
func (m *Migrator) transformWriteCall(n ast.Node) (ast.Node, error) {
	callExpr, ok := n.(*ast.CallExpr)
	if !ok {
		return n, nil
	}

	if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if selectorExpr.Sel.Name == "Write" && m.isOutputReceiver(selectorExpr.X) {
			// Create a block with Build() and Render() calls
			return &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   selectorExpr.X,
								Sel: &ast.Ident{Name: "Build"},
							},
						},
					},
					&ast.ExprStmt{
						X: &ast.BasicLit{
							Kind:  token.STRING,
							Value: "// TODO: Add Output.NewOutput().Render(ctx, doc) call",
						},
					},
				},
			}, nil
		}
	}

	return n, nil
}

// transformKeysAssignment converts Keys assignments to WithKeys() options
func (m *Migrator) transformKeysAssignment(n ast.Node) (ast.Node, error) {
	assignStmt, ok := n.(*ast.AssignStmt)
	if !ok {
		return n, nil
	}

	for _, lhs := range assignStmt.Lhs {
		if selectorExpr, ok := lhs.(*ast.SelectorExpr); ok {
			if selectorExpr.Sel.Name == "Keys" && m.isOutputReceiver(selectorExpr.X) {
				// Convert to a comment with WithKeys suggestion
				return &ast.ExprStmt{
					X: &ast.BasicLit{
						Kind:  token.STRING,
						Value: "// TODO: Use WithKeys() option in Table() method instead of Keys assignment",
					},
				}, nil
			}
		}
	}

	return n, nil
}

// transformAddHeader replaces AddHeader with Header() method
func (m *Migrator) transformAddHeader(n ast.Node) (ast.Node, error) {
	callExpr, ok := n.(*ast.CallExpr)
	if !ok {
		return n, nil
	}

	if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if selectorExpr.Sel.Name == "AddHeader" {
			// Replace with Header() method call
			selectorExpr.Sel.Name = "Header"
		}
	}

	return callExpr, nil
}

// transformProgressCreation updates progress creation to v2 API
func (m *Migrator) transformProgressCreation(n ast.Node) (ast.Node, error) {
	callExpr, ok := n.(*ast.CallExpr)
	if !ok {
		return n, nil
	}

	if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if selectorExpr.Sel.Name == "NewProgress" {
			// Update package reference from format to output
			if ident, ok := selectorExpr.X.(*ast.Ident); ok {
				if ident.Name == "format" {
					ident.Name = "output"
				}
			}
		}
	}

	return callExpr, nil
}
