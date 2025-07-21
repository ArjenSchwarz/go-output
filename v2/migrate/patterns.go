package migrate

import (
	"go/ast"
	"strings"
)

// initializePatterns sets up common v1 usage pattern detectors
func (m *Migrator) initializePatterns() {
	m.patterns = []Pattern{
		{
			Name:        "OutputArrayDeclaration",
			Description: "Detects v1 OutputArray struct declarations",
			Detector:    m.detectOutputArrayDeclaration,
			Example:     "output := &format.OutputArray{}",
		},
		{
			Name:        "OutputSettingsUsage",
			Description: "Detects v1 OutputSettings struct usage",
			Detector:    m.detectOutputSettingsUsage,
			Example:     "settings := format.NewOutputSettings()",
		},
		{
			Name:        "AddContentsCall",
			Description: "Detects v1 AddContents method calls",
			Detector:    m.detectAddContentsCall,
			Example:     "output.AddContents(data)",
		},
		{
			Name:        "AddToBufferCall",
			Description: "Detects v1 AddToBuffer method calls",
			Detector:    m.detectAddToBufferCall,
			Example:     "output.AddToBuffer()",
		},
		{
			Name:        "WriteCall",
			Description: "Detects v1 Write method calls",
			Detector:    m.detectWriteCall,
			Example:     "output.Write()",
		},
		{
			Name:        "KeysFieldAssignment",
			Description: "Detects v1 Keys field assignments",
			Detector:    m.detectKeysFieldAssignment,
			Example:     "output.Keys = []string{\"Name\", \"Age\"}",
		},
		{
			Name:        "SettingsFieldAssignment",
			Description: "Detects v1 Settings field assignments",
			Detector:    m.detectSettingsFieldAssignment,
			Example:     "output.Settings = settings",
		},
		{
			Name:        "V1ImportStatement",
			Description: "Detects v1 import statements",
			Detector:    m.detectV1ImportStatement,
			Example:     "import \"github.com/ArjenSchwarz/go-output\"",
		},
		{
			Name:        "ProgressCreation",
			Description: "Detects v1 progress creation patterns",
			Detector:    m.detectProgressCreation,
			Example:     "p := format.NewProgress(settings)",
		},
		{
			Name:        "AddHeaderCall",
			Description: "Detects v1 AddHeader method calls",
			Detector:    m.detectAddHeaderCall,
			Example:     "output.AddHeader(\"Title\")",
		},
	}
}

// detectOutputArrayDeclaration detects v1 OutputArray declarations
func (m *Migrator) detectOutputArrayDeclaration(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.CompositeLit:
		if selectorExpr, ok := node.Type.(*ast.SelectorExpr); ok {
			return selectorExpr.Sel.Name == "OutputArray"
		}
	case *ast.ValueSpec:
		for _, name := range node.Names {
			if strings.Contains(name.Name, "output") || strings.Contains(name.Name, "Output") {
				if selectorExpr, ok := node.Type.(*ast.SelectorExpr); ok {
					return selectorExpr.Sel.Name == "OutputArray"
				}
			}
		}
	}
	return false
}

// detectOutputSettingsUsage detects v1 OutputSettings usage
func (m *Migrator) detectOutputSettingsUsage(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.CallExpr:
		if selectorExpr, ok := node.Fun.(*ast.SelectorExpr); ok {
			return selectorExpr.Sel.Name == "NewOutputSettings"
		}
	case *ast.SelectorExpr:
		return node.Sel.Name == "OutputSettings"
	}
	return false
}

// detectAddContentsCall detects v1 AddContents method calls
func (m *Migrator) detectAddContentsCall(n ast.Node) bool {
	if callExpr, ok := n.(*ast.CallExpr); ok {
		if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			return selectorExpr.Sel.Name == "AddContents"
		}
	}
	return false
}

// detectAddToBufferCall detects v1 AddToBuffer method calls
func (m *Migrator) detectAddToBufferCall(n ast.Node) bool {
	if callExpr, ok := n.(*ast.CallExpr); ok {
		if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			return selectorExpr.Sel.Name == "AddToBuffer"
		}
	}
	return false
}

// detectWriteCall detects v1 Write method calls
func (m *Migrator) detectWriteCall(n ast.Node) bool {
	if callExpr, ok := n.(*ast.CallExpr); ok {
		if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			return selectorExpr.Sel.Name == "Write" && m.isOutputReceiver(selectorExpr.X)
		}
	}
	return false
}

// detectKeysFieldAssignment detects v1 Keys field assignments
func (m *Migrator) detectKeysFieldAssignment(n ast.Node) bool {
	// Check for assignment like: output.Keys = []string{...}
	if assignStmt, ok := n.(*ast.AssignStmt); ok {
		for _, lhs := range assignStmt.Lhs {
			if selectorExpr, ok := lhs.(*ast.SelectorExpr); ok {
				if selectorExpr.Sel.Name == "Keys" && m.isOutputReceiver(selectorExpr.X) {
					return true
				}
			}
		}
	}

	// Check for composite literal like: &OutputArray{Keys: []string{...}}
	if compositeLit, ok := n.(*ast.CompositeLit); ok {
		if selectorExpr, ok := compositeLit.Type.(*ast.SelectorExpr); ok {
			if selectorExpr.Sel.Name == "OutputArray" {
				// Check if Keys field is present in the composite literal
				for _, elt := range compositeLit.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "Keys" {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// detectSettingsFieldAssignment detects v1 Settings field assignments
func (m *Migrator) detectSettingsFieldAssignment(n ast.Node) bool {
	if assignStmt, ok := n.(*ast.AssignStmt); ok {
		for _, lhs := range assignStmt.Lhs {
			if selectorExpr, ok := lhs.(*ast.SelectorExpr); ok {
				// Check for assignments like: output.Settings = settings
				if selectorExpr.Sel.Name == "Settings" && m.isOutputReceiver(selectorExpr.X) {
					return true
				}
				// Check for assignments like: settings.OutputFormat = "table"
				if ident, ok := selectorExpr.X.(*ast.Ident); ok {
					settingsFields := []string{"OutputFormat", "UseEmoji", "UseColors", "TableStyle", "HasTOC", "ProgressTimeout"}
					for _, field := range settingsFields {
						if selectorExpr.Sel.Name == field && strings.Contains(strings.ToLower(ident.Name), "setting") {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// detectV1ImportStatement detects v1 import statements
func (m *Migrator) detectV1ImportStatement(n ast.Node) bool {
	if importSpec, ok := n.(*ast.ImportSpec); ok {
		if importSpec.Path != nil {
			path := strings.Trim(importSpec.Path.Value, "\"")
			return strings.Contains(path, "github.com/ArjenSchwarz/go-output") && !strings.Contains(path, "/v2")
		}
	}
	return false
}

// detectProgressCreation detects v1 progress creation patterns
func (m *Migrator) detectProgressCreation(n ast.Node) bool {
	if callExpr, ok := n.(*ast.CallExpr); ok {
		if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			return selectorExpr.Sel.Name == "NewProgress"
		}
	}
	return false
}

// detectAddHeaderCall detects v1 AddHeader method calls
func (m *Migrator) detectAddHeaderCall(n ast.Node) bool {
	if callExpr, ok := n.(*ast.CallExpr); ok {
		if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			return selectorExpr.Sel.Name == "AddHeader"
		}
	}
	return false
}

// isOutputReceiver checks if an expression represents an OutputArray receiver
func (m *Migrator) isOutputReceiver(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		// Check if the identifier name suggests it's an output variable
		name := strings.ToLower(e.Name)
		return strings.Contains(name, "output") || strings.Contains(name, "out")
	default:
		return false
	}
}
