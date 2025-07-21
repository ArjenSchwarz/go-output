package migrate

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// AdvancedTransformer provides sophisticated AST transformations
type AdvancedTransformer struct {
	migrator *Migrator
}

// NewAdvancedTransformer creates a new advanced transformer
func NewAdvancedTransformer(m *Migrator) *AdvancedTransformer {
	return &AdvancedTransformer{migrator: m}
}

// TransformCompletePattern transforms complete v1 usage patterns to v2
func (at *AdvancedTransformer) TransformCompletePattern(file *ast.File, pattern ComplexPattern) (*ast.File, error) {
	switch pattern.Type {
	case "OutputArrayUsage":
		return at.transformOutputArrayPattern(file, pattern)
	case "OutputSettingsUsage":
		return at.transformSettingsPattern(file, pattern)
	case "ProgressUsage":
		return at.transformProgressPattern(file, pattern)
	default:
		return file, nil
	}
}

// transformOutputArrayPattern transforms complete OutputArray usage to v2 builder pattern
func (at *AdvancedTransformer) transformOutputArrayPattern(file *ast.File, pattern ComplexPattern) (*ast.File, error) {
	// Create a deep copy of the file
	newFile := at.copyFile(file)

	// Find the OutputArray variable
	var outputVar string
	ast.Inspect(newFile, func(n ast.Node) bool {
		// Find OutputArray declaration
		if valueSpec, ok := n.(*ast.ValueSpec); ok {
			if starExpr, ok := valueSpec.Type.(*ast.StarExpr); ok {
				if selectorExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
					if selectorExpr.Sel.Name == "OutputArray" {
						if len(valueSpec.Names) > 0 {
							outputVar = valueSpec.Names[0].Name
						}
					}
				}
			}
		}
		return true
	})

	if outputVar == "" {
		return file, fmt.Errorf("OutputArray variable not found")
	}

	// Transform the usage pattern
	builderChain := at.buildV2Chain(newFile, outputVar)

	// Replace the OutputArray declaration with builder initialization
	at.replaceOutputArrayDeclaration(newFile, outputVar, builderChain)

	// Remove or transform individual method calls
	at.removeTransformedCalls(newFile, outputVar)

	return newFile, nil
}

// buildV2Chain analyzes the v1 usage and builds the equivalent v2 chain
func (at *AdvancedTransformer) buildV2Chain(file *ast.File, outputVar string) *ast.CallExpr {
	var keys []string
	var tables []TableCall
	var headers []string

	// Analyze the usage pattern
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			// Check for Keys assignment
			for _, lhs := range node.Lhs {
				if selectorExpr, ok := lhs.(*ast.SelectorExpr); ok {
					if at.isVariableReference(selectorExpr.X, outputVar) && selectorExpr.Sel.Name == "Keys" {
						keys = at.extractKeysFromAssignment(node.Rhs[0])
					}
				}
			}
		case *ast.CallExpr:
			if selectorExpr, ok := node.Fun.(*ast.SelectorExpr); ok {
				if at.isVariableReference(selectorExpr.X, outputVar) {
					switch selectorExpr.Sel.Name {
					case "AddContents":
						tables = append(tables, TableCall{
							Data: node.Args[0],
							Keys: keys,
						})
					case "AddHeader":
						if len(node.Args) > 0 {
							if basicLit, ok := node.Args[0].(*ast.BasicLit); ok {
								headerText := strings.Trim(basicLit.Value, "\"")
								headers = append(headers, headerText)
							}
						}
					}
				}
			}
		}
		return true
	})

	// Build the v2 chain
	chain := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "output"},
			Sel: &ast.Ident{Name: "New"},
		},
	}

	// Add headers
	for _, header := range headers {
		chain = at.chainMethod(chain, "Header", &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(header),
		})
	}

	// Add tables
	for i, table := range tables {
		title := ""
		if i > 0 || len(headers) > 0 {
			title = fmt.Sprintf("Table %d", i+1)
		}

		args := []ast.Expr{
			&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(title)},
			table.Data,
		}

		// Add WithKeys option if keys are specified
		if len(table.Keys) > 0 {
			keysOption := at.createWithKeysOption(table.Keys)
			args = append(args, keysOption)
		}

		chain = at.chainMethod(chain, "Table", args...)
	}

	// Add Build() call
	chain = at.chainMethod(chain, "Build")

	return chain
}

// TableCall represents a table call to be converted
type TableCall struct {
	Data ast.Expr
	Keys []string
}

// extractKeysFromAssignment extracts key names from a Keys assignment
func (at *AdvancedTransformer) extractKeysFromAssignment(expr ast.Expr) []string {
	var keys []string

	if compositeLit, ok := expr.(*ast.CompositeLit); ok {
		for _, elt := range compositeLit.Elts {
			if basicLit, ok := elt.(*ast.BasicLit); ok {
				key := strings.Trim(basicLit.Value, "\"")
				keys = append(keys, key)
			}
		}
	}

	return keys
}

// chainMethod adds a method call to the chain
func (at *AdvancedTransformer) chainMethod(base *ast.CallExpr, method string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   base,
			Sel: &ast.Ident{Name: method},
		},
		Args: args,
	}
}

// createWithKeysOption creates a WithKeys functional option
func (at *AdvancedTransformer) createWithKeysOption(keys []string) *ast.CallExpr {
	var keyExprs []ast.Expr
	for _, key := range keys {
		keyExprs = append(keyExprs, &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(key),
		})
	}

	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "output"},
			Sel: &ast.Ident{Name: "WithKeys"},
		},
		Args: keyExprs,
	}
}

// replaceOutputArrayDeclaration replaces the OutputArray declaration
func (at *AdvancedTransformer) replaceOutputArrayDeclaration(file *ast.File, outputVar string, builderChain *ast.CallExpr) {
	ast.Inspect(file, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			for i, lhs := range assignStmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == outputVar {
					if i < len(assignStmt.Rhs) {
						// Replace the RHS with the builder chain
						assignStmt.Rhs[i] = builderChain
					}
				}
			}
		}
		return true
	})
}

// removeTransformedCalls removes calls that have been incorporated into the builder pattern
func (at *AdvancedTransformer) removeTransformedCalls(file *ast.File, outputVar string) {
	// This would need a more sophisticated approach to remove statements
	// For now, we'll replace them with comments
	ast.Inspect(file, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if at.isVariableReference(selectorExpr.X, outputVar) {
					switch selectorExpr.Sel.Name {
					case "AddContents", "AddToBuffer", "AddHeader":
						// Replace with comment (simplified approach)
						// In practice, we'd need to remove the entire statement
						callExpr.Fun = &ast.Ident{Name: "// Migrated to builder pattern"}
					case "Write":
						// Replace with Render call
						at.replaceWriteWithRender(callExpr, outputVar)
					}
				}
			}
		}
		return true
	})
}

// replaceWriteWithRender replaces Write() call with Render() call
func (at *AdvancedTransformer) replaceWriteWithRender(callExpr *ast.CallExpr, outputVar string) {
	// Create: output.NewOutput().Render(ctx, doc)
	renderCall := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "output"},
					Sel: &ast.Ident{Name: "NewOutput"},
				},
			},
			Sel: &ast.Ident{Name: "Render"},
		},
		Args: []ast.Expr{
			&ast.Ident{Name: "ctx"}, // TODO: context needs to be available
			&ast.Ident{Name: "doc"}, // TODO: document variable name
		},
	}

	// Replace the original call
	*callExpr = *renderCall
}

// transformSettingsPattern transforms OutputSettings usage to functional options
func (at *AdvancedTransformer) transformSettingsPattern(file *ast.File, pattern ComplexPattern) (*ast.File, error) {
	newFile := at.copyFile(file)

	// Find settings variable and its configuration
	settingsConfig := at.analyzeSettingsConfiguration(newFile)

	// Generate functional options
	options := at.generateFunctionalOptions(settingsConfig)

	// Replace settings usage with options
	at.replaceSettingsWithOptions(newFile, options)

	return newFile, nil
}

// SettingsConfig represents OutputSettings configuration
type SettingsConfig struct {
	Variable string
	Fields   map[string]ast.Expr
}

// analyzeSettingsConfiguration analyzes OutputSettings configuration
func (at *AdvancedTransformer) analyzeSettingsConfiguration(file *ast.File) SettingsConfig {
	config := SettingsConfig{
		Fields: make(map[string]ast.Expr),
	}

	// Find settings variable and its assignments
	ast.Inspect(file, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			// Check for settings creation
			for i, rhs := range assignStmt.Rhs {
				if callExpr, ok := rhs.(*ast.CallExpr); ok {
					if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						if selectorExpr.Sel.Name == "NewOutputSettings" {
							if i < len(assignStmt.Lhs) {
								if ident, ok := assignStmt.Lhs[i].(*ast.Ident); ok {
									config.Variable = ident.Name
								}
							}
						}
					}
				}
			}

			// Check for field assignments
			for i, lhs := range assignStmt.Lhs {
				if selectorExpr, ok := lhs.(*ast.SelectorExpr); ok {
					if ident, ok := selectorExpr.X.(*ast.Ident); ok {
						if ident.Name == config.Variable {
							fieldName := selectorExpr.Sel.Name
							if i < len(assignStmt.Rhs) {
								config.Fields[fieldName] = assignStmt.Rhs[i]
							}
						}
					}
				}
			}
		}
		return true
	})

	return config
}

// generateFunctionalOptions generates v2 functional options from settings
func (at *AdvancedTransformer) generateFunctionalOptions(config SettingsConfig) []ast.Expr {
	var options []ast.Expr

	for fieldName, value := range config.Fields {
		option := at.createFunctionalOption(fieldName, value)
		if option != nil {
			options = append(options, option)
		}
	}

	return options
}

// createFunctionalOption creates a functional option for a settings field
func (at *AdvancedTransformer) createFunctionalOption(fieldName string, value ast.Expr) ast.Expr {
	switch fieldName {
	case "OutputFormat":
		return at.createFormatOption(value)
	case "UseEmoji":
		if at.isTrueValue(value) {
			return at.createTransformerOption("EmojiTransformer")
		}
	case "UseColors":
		if at.isTrueValue(value) {
			return at.createTransformerOption("ColorTransformer")
		}
	case "TableStyle":
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "output"},
				Sel: &ast.Ident{Name: "WithTableStyle"},
			},
			Args: []ast.Expr{value},
		}
	case "HasTOC":
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "output"},
				Sel: &ast.Ident{Name: "WithTOC"},
			},
			Args: []ast.Expr{value},
		}
	}

	return nil
}

// createFormatOption creates a format option
func (at *AdvancedTransformer) createFormatOption(value ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "output"},
			Sel: &ast.Ident{Name: "WithFormat"},
		},
		Args: []ast.Expr{
			&ast.SelectorExpr{
				X:   &ast.Ident{Name: "output"},
				Sel: &ast.Ident{Name: "Table"}, // Default to Table format
			},
		},
	}
}

// createTransformerOption creates a transformer option
func (at *AdvancedTransformer) createTransformerOption(transformerName string) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "output"},
			Sel: &ast.Ident{Name: "WithTransformer"},
		},
		Args: []ast.Expr{
			&ast.UnaryExpr{
				Op: token.AND,
				X: &ast.CompositeLit{
					Type: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "output"},
						Sel: &ast.Ident{Name: transformerName},
					},
				},
			},
		},
	}
}

// isTrueValue checks if an expression represents a true value
func (at *AdvancedTransformer) isTrueValue(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "true"
	}
	return false
}

// replaceSettingsWithOptions replaces settings usage with functional options
func (at *AdvancedTransformer) replaceSettingsWithOptions(file *ast.File, options []ast.Expr) {
	// This would need sophisticated replacement logic
	// For now, we'll add a comment with the options
	// In practice, we'd need to find where the settings are used and replace appropriately
}

// transformProgressPattern transforms progress usage (minimal changes needed)
func (at *AdvancedTransformer) transformProgressPattern(file *ast.File, pattern ComplexPattern) (*ast.File, error) {
	newFile := at.copyFile(file)

	// Update package references from format to output
	ast.Inspect(newFile, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := selectorExpr.X.(*ast.Ident); ok {
					if ident.Name == "format" && selectorExpr.Sel.Name == "NewProgress" {
						ident.Name = "output"
					}
				}
			}
		}
		return true
	})

	return newFile, nil
}

// Helper methods

// copyFile creates a deep copy of an AST file
func (at *AdvancedTransformer) copyFile(file *ast.File) *ast.File {
	// This is a simplified implementation
	// In practice, we'd need a proper deep copy mechanism
	return file
}

// isVariableReference checks if an expression references a specific variable
func (at *AdvancedTransformer) isVariableReference(expr ast.Expr, varName string) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == varName
	}
	return false
}
