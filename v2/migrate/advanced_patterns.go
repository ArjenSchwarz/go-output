package migrate

import (
	"go/ast"
	"go/token"
	"strings"
)

// AdvancedPatternDetector provides more sophisticated pattern detection
type AdvancedPatternDetector struct {
	migrator *Migrator
}

// NewAdvancedPatternDetector creates a new advanced pattern detector
func NewAdvancedPatternDetector(m *Migrator) *AdvancedPatternDetector {
	return &AdvancedPatternDetector{migrator: m}
}

// DetectComplexPatterns detects complex v1 usage patterns that require contextual analysis
func (apd *AdvancedPatternDetector) DetectComplexPatterns(file *ast.File) []ComplexPattern {
	var patterns []ComplexPattern

	// Track OutputArray variables and their usage
	outputVars := apd.findOutputArrayVariables(file)

	for _, outputVar := range outputVars {
		pattern := apd.analyzeOutputArrayUsage(file, outputVar)
		if pattern != nil {
			patterns = append(patterns, *pattern)
		}
	}

	// Detect Settings configuration patterns
	settingsPatterns := apd.detectSettingsPatterns(file)
	patterns = append(patterns, settingsPatterns...)

	// Detect Progress usage patterns
	progressPatterns := apd.detectProgressPatterns(file)
	patterns = append(patterns, progressPatterns...)

	return patterns
}

// ComplexPattern represents a detected complex usage pattern
type ComplexPattern struct {
	Type        string
	Description string
	Locations   []PatternLocation
	Severity    string // "info", "warning", "error"
	Suggestion  string
}

// PatternLocation represents where a pattern was found
type PatternLocation struct {
	Line   int
	Column int
	Code   string
}

// findOutputArrayVariables finds all OutputArray variable declarations
func (apd *AdvancedPatternDetector) findOutputArrayVariables(file *ast.File) []string {
	var outputVars []string

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ValueSpec:
			// Check for OutputArray type declarations
			if starExpr, ok := node.Type.(*ast.StarExpr); ok {
				if selectorExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
					if selectorExpr.Sel.Name == "OutputArray" {
						for _, name := range node.Names {
							outputVars = append(outputVars, name.Name)
						}
					}
				}
			}
		case *ast.AssignStmt:
			// Check for OutputArray assignments
			for i, rhs := range node.Rhs {
				if i < len(node.Lhs) {
					if apd.isOutputArrayCreation(rhs) {
						if ident, ok := node.Lhs[i].(*ast.Ident); ok {
							outputVars = append(outputVars, ident.Name)
						}
					}
				}
			}
		}
		return true
	})

	return outputVars
}

// analyzeOutputArrayUsage analyzes how an OutputArray variable is used
func (apd *AdvancedPatternDetector) analyzeOutputArrayUsage(file *ast.File, varName string) *ComplexPattern {
	pattern := &ComplexPattern{
		Type:        "OutputArrayUsage",
		Description: "Complete OutputArray usage pattern",
		Severity:    "info",
	}

	var hasKeysAssignment bool
	var hasAddContents bool
	var hasAddToBuffer bool
	var hasWrite bool
	var locations []PatternLocation

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			// Check for Keys assignment
			for _, lhs := range node.Lhs {
				if selectorExpr, ok := lhs.(*ast.SelectorExpr); ok {
					if apd.isVariableReference(selectorExpr.X, varName) && selectorExpr.Sel.Name == "Keys" {
						hasKeysAssignment = true
						pos := apd.migrator.fset.Position(node.Pos())
						locations = append(locations, PatternLocation{
							Line:   pos.Line,
							Column: pos.Column,
							Code:   "Keys assignment",
						})
					}
				}
			}
		case *ast.CallExpr:
			if selectorExpr, ok := node.Fun.(*ast.SelectorExpr); ok {
				if apd.isVariableReference(selectorExpr.X, varName) {
					switch selectorExpr.Sel.Name {
					case "AddContents":
						hasAddContents = true
						pos := apd.migrator.fset.Position(node.Pos())
						locations = append(locations, PatternLocation{
							Line:   pos.Line,
							Column: pos.Column,
							Code:   "AddContents call",
						})
					case "AddToBuffer":
						hasAddToBuffer = true
						pos := apd.migrator.fset.Position(node.Pos())
						locations = append(locations, PatternLocation{
							Line:   pos.Line,
							Column: pos.Column,
							Code:   "AddToBuffer call",
						})
					case "Write":
						hasWrite = true
						pos := apd.migrator.fset.Position(node.Pos())
						locations = append(locations, PatternLocation{
							Line:   pos.Line,
							Column: pos.Column,
							Code:   "Write call",
						})
					}
				}
			}
		}
		return true
	})

	pattern.Locations = locations

	// Generate suggestions based on usage pattern
	if hasKeysAssignment && hasAddContents && hasWrite {
		if hasAddToBuffer {
			pattern.Suggestion = "Convert to multiple Table() calls with WithKeys() options, then Build() and Render()"
		} else {
			pattern.Suggestion = "Convert to single Table() call with WithKeys() option, then Build() and Render()"
		}
	} else if hasAddContents && hasWrite {
		pattern.Suggestion = "Convert to Table() call with auto-schema detection, then Build() and Render()"
	} else {
		pattern.Suggestion = "Incomplete OutputArray usage - manual migration required"
		pattern.Severity = "warning"
	}

	return pattern
}

// detectSettingsPatterns detects OutputSettings usage patterns
func (apd *AdvancedPatternDetector) detectSettingsPatterns(file *ast.File) []ComplexPattern {
	var patterns []ComplexPattern

	settingsVars := apd.findSettingsVariables(file)

	for _, settingsVar := range settingsVars {
		pattern := apd.analyzeSettingsUsage(file, settingsVar)
		if pattern != nil {
			patterns = append(patterns, *pattern)
		}
	}

	return patterns
}

// findSettingsVariables finds OutputSettings variable declarations
func (apd *AdvancedPatternDetector) findSettingsVariables(file *ast.File) []string {
	var settingsVars []string

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			for i, rhs := range node.Rhs {
				if callExpr, ok := rhs.(*ast.CallExpr); ok {
					if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						if selectorExpr.Sel.Name == "NewOutputSettings" {
							if i < len(node.Lhs) {
								if ident, ok := node.Lhs[i].(*ast.Ident); ok {
									settingsVars = append(settingsVars, ident.Name)
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	return settingsVars
}

// analyzeSettingsUsage analyzes OutputSettings usage
func (apd *AdvancedPatternDetector) analyzeSettingsUsage(file *ast.File, varName string) *ComplexPattern {
	pattern := &ComplexPattern{
		Type:        "OutputSettingsUsage",
		Description: "OutputSettings configuration pattern",
		Severity:    "info",
	}

	var locations []PatternLocation
	var configuredFields []string

	ast.Inspect(file, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			for _, lhs := range assignStmt.Lhs {
				if selectorExpr, ok := lhs.(*ast.SelectorExpr); ok {
					if apd.isVariableReference(selectorExpr.X, varName) {
						fieldName := selectorExpr.Sel.Name
						configuredFields = append(configuredFields, fieldName)

						pos := apd.migrator.fset.Position(assignStmt.Pos())
						locations = append(locations, PatternLocation{
							Line:   pos.Line,
							Column: pos.Column,
							Code:   fieldName + " configuration",
						})
					}
				}
			}
		}
		return true
	})

	pattern.Locations = locations

	// Generate suggestion based on configured fields
	v2Options := apd.mapSettingsToV2Options(configuredFields)
	if len(v2Options) > 0 {
		pattern.Suggestion = "Replace with functional options: " + strings.Join(v2Options, ", ")
	} else {
		pattern.Suggestion = "Review settings configuration for manual migration"
		pattern.Severity = "warning"
	}

	return pattern
}

// mapSettingsToV2Options maps v1 settings fields to v2 options
func (apd *AdvancedPatternDetector) mapSettingsToV2Options(fields []string) []string {
	fieldMapping := map[string]string{
		"OutputFormat":    "WithFormat()",
		"OutputFile":      "WithWriter(NewFileWriter())",
		"UseEmoji":        "WithTransformer(&EmojiTransformer{})",
		"UseColors":       "WithTransformer(&ColorTransformer{})",
		"SortKey":         "WithTransformer(&SortTransformer{})",
		"TableStyle":      "WithTableStyle()",
		"HasTOC":          "WithTOC()",
		"ProgressOptions": "WithProgress()",
	}

	var options []string
	for _, field := range fields {
		if option, exists := fieldMapping[field]; exists {
			options = append(options, option)
		}
	}

	return options
}

// detectProgressPatterns detects Progress usage patterns
func (apd *AdvancedPatternDetector) detectProgressPatterns(file *ast.File) []ComplexPattern {
	var patterns []ComplexPattern

	// Find progress variable declarations
	progressVars := apd.findProgressVariables(file)

	for _, progressVar := range progressVars {
		pattern := apd.analyzeProgressUsage(file, progressVar)
		if pattern != nil {
			patterns = append(patterns, *pattern)
		}
	}

	return patterns
}

// findProgressVariables finds progress variable declarations
func (apd *AdvancedPatternDetector) findProgressVariables(file *ast.File) []string {
	var progressVars []string

	ast.Inspect(file, func(n ast.Node) bool {
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			for i, rhs := range assignStmt.Rhs {
				if callExpr, ok := rhs.(*ast.CallExpr); ok {
					if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						if selectorExpr.Sel.Name == "NewProgress" {
							if i < len(assignStmt.Lhs) {
								if ident, ok := assignStmt.Lhs[i].(*ast.Ident); ok {
									progressVars = append(progressVars, ident.Name)
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	return progressVars
}

// analyzeProgressUsage analyzes progress usage patterns
func (apd *AdvancedPatternDetector) analyzeProgressUsage(file *ast.File, varName string) *ComplexPattern {
	pattern := &ComplexPattern{
		Type:        "ProgressUsage",
		Description: "Progress indicator usage pattern",
		Severity:    "info",
	}

	var locations []PatternLocation
	var methodCalls []string

	ast.Inspect(file, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if apd.isVariableReference(selectorExpr.X, varName) {
					methodName := selectorExpr.Sel.Name
					methodCalls = append(methodCalls, methodName)

					pos := apd.migrator.fset.Position(callExpr.Pos())
					locations = append(locations, PatternLocation{
						Line:   pos.Line,
						Column: pos.Column,
						Code:   methodName + " call",
					})
				}
			}
		}
		return true
	})

	pattern.Locations = locations
	pattern.Suggestion = "Progress API is mostly compatible, update package reference from 'format' to 'output'"

	return pattern
}

// Helper methods

// isOutputArrayCreation checks if an expression creates an OutputArray
func (apd *AdvancedPatternDetector) isOutputArrayCreation(expr ast.Expr) bool {
	if unaryExpr, ok := expr.(*ast.UnaryExpr); ok && unaryExpr.Op == token.AND {
		if compositeLit, ok := unaryExpr.X.(*ast.CompositeLit); ok {
			if selectorExpr, ok := compositeLit.Type.(*ast.SelectorExpr); ok {
				return selectorExpr.Sel.Name == "OutputArray"
			}
		}
	}
	return false
}

// isVariableReference checks if an expression references a specific variable
func (apd *AdvancedPatternDetector) isVariableReference(expr ast.Expr, varName string) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == varName
	}
	return false
}
