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
		{
			Name:        "UpdatePackageReferences",
			Description: "Updates format package references to output",
			Transform:   m.transformPackageReferences,
			Priority:    10,
		},
	}
}

// transformImportStatement updates v1 imports to v2
func (m *Migrator) transformImportStatement(n ast.Node) (ast.Node, error) {
	// Handle import specs within GenDecl
	if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
		transformed := false
		for _, spec := range genDecl.Specs {
			if importSpec, ok := spec.(*ast.ImportSpec); ok {
				if importSpec.Path != nil {
					path := strings.Trim(importSpec.Path.Value, "\"")
					if strings.Contains(path, "github.com/ArjenSchwarz/go-output") && !strings.Contains(path, "/v2") {
						// Update to v2 import
						var newPath string
						if path == "github.com/ArjenSchwarz/go-output/format" {
							// format subpackage maps to v2 root
							newPath = "github.com/ArjenSchwarz/go-output/v2"
							// Change format to output for v2
							if importSpec.Name == nil || importSpec.Name.Name == "format" {
								importSpec.Name = &ast.Ident{Name: "output"}
							}
						} else {
							// Other imports just add /v2
							newPath = strings.Replace(path, "github.com/ArjenSchwarz/go-output", "github.com/ArjenSchwarz/go-output/v2", 1)
						}
						importSpec.Path.Value = fmt.Sprintf("\"%s\"", newPath)
						transformed = true
					}
				}
			}
		}
		if transformed {
			// Return a new GenDecl to signal transformation
			return &ast.GenDecl{
				Doc:    genDecl.Doc,
				TokPos: genDecl.TokPos,
				Tok:    genDecl.Tok,
				Lparen: genDecl.Lparen,
				Specs:  genDecl.Specs,
				Rparen: genDecl.Rparen,
			}, nil
		}
	}

	return n, nil
}

// transformOutputArrayDeclaration replaces OutputArray with Builder pattern
func (m *Migrator) transformOutputArrayDeclaration(n ast.Node) (ast.Node, error) {
	// Handle assignments like: output := &format.OutputArray{...}
	if assignStmt, ok := n.(*ast.AssignStmt); ok {
		for _, rhs := range assignStmt.Rhs {
			if unaryExpr, ok := rhs.(*ast.UnaryExpr); ok && unaryExpr.Op == token.AND {
				if compositeLit, ok := unaryExpr.X.(*ast.CompositeLit); ok {
					if selectorExpr, ok := compositeLit.Type.(*ast.SelectorExpr); ok {
						if selectorExpr.Sel.Name == "OutputArray" {
							// Extract Keys if present
							var keysExpr ast.Expr
							for _, elt := range compositeLit.Elts {
								if kv, ok := elt.(*ast.KeyValueExpr); ok {
									if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "Keys" {
										keysExpr = kv.Value
										break
									}
								}
							}

							// Store keys for later use in transformations
							if keysExpr != nil {
								m.storedKeys = keysExpr
							}

							// Replace with output.New() call
							newCallExpr := &ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   &ast.Ident{Name: "output"},
									Sel: &ast.Ident{Name: "New"},
								},
							}

							// Create a new assignment statement
							return &ast.AssignStmt{
								Lhs: assignStmt.Lhs,
								Tok: assignStmt.Tok,
								Rhs: []ast.Expr{newCallExpr},
							}, nil
						}
					}
				}
			}
		}
	}

	return n, nil
}

// transformOutputSettings replaces OutputSettings with functional options
func (m *Migrator) transformOutputSettings(n ast.Node) (ast.Node, error) {
	// Handle assignment statements like: settings := format.NewOutputSettings()
	if assignStmt, ok := n.(*ast.AssignStmt); ok {
		for _, rhs := range assignStmt.Rhs {
			if callExpr, ok := rhs.(*ast.CallExpr); ok {
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
			}
		}
	}

	// Handle expression statements containing the call
	if exprStmt, ok := n.(*ast.ExprStmt); ok {
		if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
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
		}
	}

	return n, nil
}

// transformAddContents replaces AddContents calls with builder methods
func (m *Migrator) transformAddContents(n ast.Node) (ast.Node, error) {
	// Handle expression statements like: output.AddContents(data)
	if exprStmt, ok := n.(*ast.ExprStmt); ok {
		if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
			if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if selectorExpr.Sel.Name == "AddContents" && m.isOutputReceiver(selectorExpr.X) {
					// Replace with assignment: output = output.Table(...)
					newCallExpr := &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   selectorExpr.X,
							Sel: &ast.Ident{Name: "Table"},
						},
						Args: []ast.Expr{
							&ast.BasicLit{Kind: token.STRING, Value: "\"\""}, // Empty title
						},
					}
					newCallExpr.Args = append(newCallExpr.Args, callExpr.Args...)

					// Add WithKeys if we have stored keys
					if m.storedKeys != nil {
						withKeysCall := &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   &ast.Ident{Name: "output"},
								Sel: &ast.Ident{Name: "WithKeys"},
							},
						}
						// Convert array to individual arguments
						if arrayLit, ok := m.storedKeys.(*ast.CompositeLit); ok {
							withKeysCall.Args = arrayLit.Elts
						}
						newCallExpr.Args = append(newCallExpr.Args, withKeysCall)
					}

					return &ast.AssignStmt{
						Lhs: []ast.Expr{selectorExpr.X},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{newCallExpr},
					}, nil
				}
			}
		}
	}

	return n, nil
}

// transformAddToBuffer removes AddToBuffer calls
func (m *Migrator) transformAddToBuffer(n ast.Node) (ast.Node, error) {
	// Handle expression statements like: output.AddToBuffer()
	if exprStmt, ok := n.(*ast.ExprStmt); ok {
		if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
			if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if selectorExpr.Sel.Name == "AddToBuffer" && m.isOutputReceiver(selectorExpr.X) {
					// Return an empty statement (effectively removing the call)
					return &ast.EmptyStmt{}, nil
				}
			}
		}
	}

	return n, nil
}

// transformWriteCall replaces Write() with Build() and Render()
func (m *Migrator) transformWriteCall(n ast.Node) (ast.Node, error) {
	// Handle expression statements like: output.Write()
	if exprStmt, ok := n.(*ast.ExprStmt); ok {
		if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
			if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if selectorExpr.Sel.Name == "Write" && m.isOutputReceiver(selectorExpr.X) {
					// Change to: output = output.Build()
					// In reality, we'd need to generate the full render code,
					// but for now just change Write to Build
					selectorExpr.Sel.Name = "Build"
					return &ast.AssignStmt{
						Lhs: []ast.Expr{selectorExpr.X},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{callExpr},
					}, nil
				}
			}
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
			// Create a new call expression with updated method name
			newSelector := &ast.SelectorExpr{
				X:   selectorExpr.X,
				Sel: &ast.Ident{Name: "Header"},
			}
			return &ast.CallExpr{
				Fun:      newSelector,
				Lparen:   callExpr.Lparen,
				Args:     callExpr.Args,
				Ellipsis: callExpr.Ellipsis,
				Rparen:   callExpr.Rparen,
			}, nil
		}
	}

	return n, nil
}

// transformPackageReferences updates format package references to output
func (m *Migrator) transformPackageReferences(n ast.Node) (ast.Node, error) {
	// Update selector expressions like format.OutputArray to output.OutputArray
	if selectorExpr, ok := n.(*ast.SelectorExpr); ok {
		if ident, ok := selectorExpr.X.(*ast.Ident); ok {
			if ident.Name == "format" {
				// Return a new selector expression with updated package name
				return &ast.SelectorExpr{
					X:   &ast.Ident{Name: "output"},
					Sel: selectorExpr.Sel,
				}, nil
			}
		}
	}

	return n, nil
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
				// Handle both format and output package names (in case UpdatePackageReferences ran first)
				if ident.Name == "format" || ident.Name == "output" {
					// Create new nodes to ensure the change is detected
					newIdent := &ast.Ident{Name: "output"}
					newSelector := &ast.SelectorExpr{
						X:   newIdent,
						Sel: selectorExpr.Sel,
					}
					return &ast.CallExpr{
						Fun:      newSelector,
						Lparen:   callExpr.Lparen,
						Args:     callExpr.Args,
						Ellipsis: callExpr.Ellipsis,
						Rparen:   callExpr.Rparen,
					}, nil
				}
			}
		}
	}

	return n, nil
}
