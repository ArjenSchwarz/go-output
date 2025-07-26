# Cross-Format Collapsible Content System

## Overview

This document outlines the design for implementing collapsible/expandable content in the go-output v2 library that works consistently across all output formats (Markdown, JSON, YAML, HTML, Table, CSV, etc.). The system is designed for analysis tools that generate GitHub PR comments and other outputs where users need quick overviews with access to detailed information on demand.

## Background

### Problem Statement
Analysis tools (linters, test runners, security scanners) often generate tables with verbose details that overwhelm users. Current solutions:
- Show all details (information overload)
- Hide details completely (insufficient information)
- Format-specific solutions (inconsistent experience)

### Use Cases
- **GitHub PR Comments**: Tables with collapsible error logs, test details, security findings
- **JSON/YAML APIs**: Structured data with summary and detail separation
- **Terminal Output**: Compact summaries with option to expand
- **HTML Reports**: Interactive expandable sections
- **CSV Export**: Separate summary and detail columns

## Architecture Design

### Core Interface

```go
// CollapsibleValue represents a value that can be expanded/collapsed across formats
type CollapsibleValue interface {
    // Summary returns the collapsed view (what users see initially)
    Summary() string
    
    // Details returns the expanded content
    Details() any
    
    // IsExpanded returns whether this should be expanded by default
    IsExpanded() bool
    
    // FormatHint provides renderer-specific hints
    FormatHint(format string) map[string]any
}

// DefaultCollapsibleValue provides a standard implementation
type DefaultCollapsibleValue struct {
    summary         string
    details         any
    defaultExpanded bool
    formatHints     map[string]map[string]any
}

func (d *DefaultCollapsibleValue) Summary() string {
    return d.summary
}

func (d *DefaultCollapsibleValue) Details() any {
    return d.details
}

func (d *DefaultCollapsibleValue) IsExpanded() bool {
    return d.defaultExpanded
}

func (d *DefaultCollapsibleValue) FormatHint(format string) map[string]any {
    if hints, exists := d.formatHints[format]; exists {
        return hints
    }
    return nil
}
```

### Integration with Existing Field System

The collapsible functionality integrates with the existing `Field.Formatter` pattern without breaking changes:

```go
// Enhanced Field struct (no changes to existing fields)
type Field struct {
    Name           string
    Type           string  
    Formatter      func(any) any  // Enhanced to return CollapsibleValue or regular value
    Hidden         bool
    // No additional fields needed - collapsible logic handled in Formatter
}

// Helper functions for common collapsible patterns
func CollapsibleFormatter(summaryTemplate string, detailFunc func(any) any) func(any) any {
    return func(val any) any {
        if detailFunc == nil {
            return val
        }
        
        detail := detailFunc(val)
        summary := fmt.Sprintf(summaryTemplate, val)
        
        return &DefaultCollapsibleValue{
            summary: summary,
            details: detail,
            defaultExpanded: false,
        }
    }
}

// Specialized formatters for common patterns
func ErrorListFormatter() func(any) any {
    return CollapsibleFormatter(
        "%d errors (click to expand)",
        func(val any) any {
            if errs, ok := val.([]string); ok {
                return strings.Join(errs, "\n")
            }
            return val
        },
    )
}

func FilePathFormatter() func(any) any {
    return CollapsibleFormatter(
        "...%s (show full path)",
        func(val any) any {
            if path, ok := val.(string); ok && len(path) > 50 {
                return path // Show full path in details
            }
            return val
        },
    )
}
```

## Format-Specific Implementations

### 1. Markdown Renderer (GitHub PR Comments)

**Location**: `v2/markdown_renderer.go`

```go
// Enhanced table cell formatting
func (m *markdownRenderer) formatCellValue(val any, field *Field) string {
    if cv, ok := val.(CollapsibleValue); ok {
        // Use GitHub's native <details> support
        openAttr := ""
        if cv.IsExpanded() {
            openAttr = " open"
        }
        
        return fmt.Sprintf("<details%s><summary>%s</summary><br/>%s</details>", 
            openAttr,
            m.escapeMarkdownTableCell(cv.Summary()),
            m.escapeMarkdownTableCell(m.formatDetails(cv.Details())))
    }
    
    // Apply existing field formatter if no collapsible value
    if field != nil && field.Formatter != nil {
        formatted := field.Formatter(val)
        if cv, ok := formatted.(CollapsibleValue); ok {
            return m.formatCellValue(formatted, nil) // Recursive call
        }
        return m.escapeMarkdownTableCell(fmt.Sprint(formatted))
    }
    
    return m.escapeMarkdownTableCell(fmt.Sprint(val))
}

func (m *markdownRenderer) formatDetails(details any) string {
    switch d := details.(type) {
    case string:
        return d
    case []string:
        return strings.Join(d, "<br/>")
    case map[string]any:
        // Format as key-value pairs
        var parts []string
        for k, v := range d {
            parts = append(parts, fmt.Sprintf("<strong>%s:</strong> %v", k, v))
        }
        return strings.Join(parts, "<br/>")
    default:
        return fmt.Sprint(details)
    }
}
```

**Output Example**:
```markdown
| File | Status | Errors |
|------|--------|--------|
| main.go | ❌ Failed | <details><summary>3 errors (click to expand)</summary><br/>syntax error<br/>missing import<br/>undefined var</details> |
```

### 2. JSON Renderer

**Location**: `v2/json_yaml_renderer.go`

```go
// Enhanced record processing in renderTableContentJSON
func (j *jsonRenderer) formatValueForJSON(val any, field *Field) any {
    // Apply field formatter first
    if field != nil && field.Formatter != nil {
        val = field.Formatter(val)
    }
    
    // Check if result is collapsible
    if cv, ok := val.(CollapsibleValue); ok {
        result := map[string]any{
            "type": "collapsible",
            "summary": cv.Summary(),
            "details": cv.Details(),
            "expanded": cv.IsExpanded(),
        }
        
        // Add format-specific hints
        if hints := cv.FormatHint(FormatJSON); hints != nil {
            for k, v := range hints {
                result[k] = v
            }
        }
        
        return result
    }
    
    return val
}
```

**Output Example**:
```json
{
  "title": "Analysis Results",
  "data": [{
    "file": "main.go",
    "status": "❌ Failed",
    "errors": {
      "type": "collapsible",
      "summary": "3 errors (click to expand)",
      "details": ["syntax error", "missing import", "undefined var"],
      "expanded": false
    }
  }],
  "schema": {
    "keys": ["file", "status", "errors"],
    "fields": [...]
  }
}
```

### 3. YAML Renderer

**Location**: `v2/json_yaml_renderer.go`

```go
// Similar to JSON but with YAML-specific formatting
func (y *yamlRenderer) formatValueForYAML(val any, field *Field) any {
    // Apply field formatter first
    if field != nil && field.Formatter != nil {
        val = field.Formatter(val)
    }
    
    if cv, ok := val.(CollapsibleValue); ok {
        result := map[string]any{
            "summary": cv.Summary(),
            "details": cv.Details(),
            "expanded": cv.IsExpanded(),
        }
        
        // YAML-specific formatting hints
        if hints := cv.FormatHint(FormatYAML); hints != nil {
            for k, v := range hints {
                result[k] = v
            }
        }
        
        return result
    }
    
    return val
}
```

**Output Example**:
```yaml
title: Analysis Results
data:
  - file: main.go
    status: ❌ Failed
    errors:
      summary: 3 errors (click to expand)
      details:
        - syntax error
        - missing import
        - undefined var
      expanded: false
```

### 4. Table Renderer (Terminal)

**Location**: `v2/table_renderer.go`

```go
// Enhanced cell formatting for terminal tables
func (t *tableRenderer) formatCellValue(val any, field *Field) string {
    // Apply field formatter first
    if field != nil && field.Formatter != nil {
        val = field.Formatter(val)
    }
    
    if cv, ok := val.(CollapsibleValue); ok {
        if cv.IsExpanded() {
            // Show both summary and details
            details := t.formatDetailsForTable(cv.Details())
            return fmt.Sprintf("%s\n%s", cv.Summary(), details)
        }
        
        // Show summary with expansion hint
        return fmt.Sprintf("%s [details hidden - use --expand for full view]", cv.Summary())
    }
    
    return fmt.Sprint(val)
}

func (t *tableRenderer) formatDetailsForTable(details any) string {
    switch d := details.(type) {
    case string:
        return t.indentText(d)
    case []string:
        return t.indentText(strings.Join(d, "\n"))
    default:
        return t.indentText(fmt.Sprint(d))
    }
}

func (t *tableRenderer) indentText(text string) string {
    lines := strings.Split(text, "\n")
    for i, line := range lines {
        lines[i] = "  " + line
    }
    return strings.Join(lines, "\n")
}
```

**Output Example**:
```
┌─────────┬───────────┬──────────────────────────────────────────┐
│ File    │ Status    │ Errors                                   │
├─────────┼───────────┼──────────────────────────────────────────┤
│ main.go │ ❌ Failed │ 3 errors [details hidden - use --expand] │
└─────────┴───────────┴──────────────────────────────────────────┘
```

**Expanded Output** (with `--expand` flag):
```
┌─────────┬───────────┬──────────────────────────────────────────┐
│ File    │ Status    │ Errors                                   │
├─────────┼───────────┼──────────────────────────────────────────┤
│ main.go │ ❌ Failed │ 3 errors (click to expand)               │
│         │           │   syntax error                           │
│         │           │   missing import                         │
│         │           │   undefined var                          │
└─────────┴───────────┴──────────────────────────────────────────┘
```

### 5. HTML Renderer

**Location**: `v2/html_renderer.go`

```go
// HTML renderer can use JavaScript for enhanced interactivity
func (h *htmlRenderer) formatCellValue(val any, field *Field) string {
    if cv, ok := val.(CollapsibleValue); ok {
        openAttr := ""
        if cv.IsExpanded() {
            openAttr = " open"
        }
        
        // Use HTML5 details element with enhanced styling
        return fmt.Sprintf(`<details%s class="collapsible-cell">
            <summary class="collapsible-summary">%s</summary>
            <div class="collapsible-details">%s</div>
        </details>`, 
            openAttr,
            html.EscapeString(cv.Summary()),
            h.formatDetailsAsHTML(cv.Details()))
    }
    
    return html.EscapeString(fmt.Sprint(val))
}
```

### 6. CSV Renderer

**Location**: `v2/csv_renderer.go`

```go
// CSV splits collapsible content into separate columns
func (c *csvRenderer) handleCollapsibleFields(table *TableContent) (*TableContent, error) {
    // Analyze schema for collapsible fields
    newFields := []Field{}
    keyOrder := []string{}
    
    for _, field := range table.Schema().Fields {
        keyOrder = append(keyOrder, field.Name)
        newFields = append(newFields, field)
        
        // Check if field uses collapsible formatter by testing with sample data
        if c.fieldHasCollapsibleContent(table, field.Name) {
            // Add detail column
            detailField := Field{
                Name: field.Name + "_details",
                Type: "string",
            }
            keyOrder = append(keyOrder, detailField.Name)
            newFields = append(newFields, detailField)
        }
    }
    
    // Transform records to include detail columns
    newRecords := []Record{}
    for _, record := range table.Records() {
        newRecord := make(Record)
        
        for _, key := range table.Schema().GetKeyOrder() {
            val := record[key]
            field := table.Schema().FindField(key)
            
            if field != nil && field.Formatter != nil {
                formatted := field.Formatter(val)
                if cv, ok := formatted.(CollapsibleValue); ok {
                    newRecord[key] = cv.Summary()
                    newRecord[key+"_details"] = cv.Details()
                    continue
                }
            }
            
            newRecord[key] = val
        }
        
        newRecords = append(newRecords, newRecord)
    }
    
    // Return new table with expanded schema
    return &TableContent{
        id:      table.id,
        title:   table.title,
        schema:  &Schema{Fields: newFields, keyOrder: keyOrder},
        records: newRecords,
    }, nil
}
```

## Usage Examples

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/arjenschwarz/go-output/v2"
)

func main() {
    // Analysis results with error details
    results := []map[string]any{
        {
            "file": "/very/long/path/to/some/deeply/nested/file.go",
            "status": "❌ Failed",
            "errors": []string{
                "line 10: syntax error near '{'",
                "line 15: undefined variable 'foo'", 
                "line 20: missing import 'fmt'",
            },
            "warnings": []string{
                "line 5: unused variable 'bar'",
            },
        },
        {
            "file": "main.go", 
            "status": "✅ Passed",
            "errors": []string{},
            "warnings": []string{},
        },
    }
    
    // Create table with collapsible columns
    table, _ := output.NewTableContent("Code Analysis Results", results,
        output.WithSchema(
            output.Field{Name: "file", Type: "string", 
                Formatter: FilePathFormatter()}, // Collapses long paths
            output.Field{Name: "status", Type: "string"},
            output.Field{Name: "errors", Type: "string",
                Formatter: ErrorListFormatter()}, // Collapses error lists
            output.Field{Name: "warnings", Type: "string", 
                Formatter: ErrorListFormatter()}, // Collapses warning lists
        ),
    )
    
    // Create document and render to different formats
    doc := output.NewDocument().AddContent(table).Build()
    
    // GitHub PR comment (Markdown with HTML details)
    markdownBytes, _ := output.Markdown.Renderer.Render(ctx, doc)
    fmt.Println("=== GITHUB PR COMMENT ===")
    fmt.Println(string(markdownBytes))
    
    // API response (JSON with structured data)
    jsonBytes, _ := output.JSON.Renderer.Render(ctx, doc) 
    fmt.Println("\n=== JSON API RESPONSE ===")
    fmt.Println(string(jsonBytes))
    
    // Terminal output (compact table)
    tableBytes, _ := output.Table.Renderer.Render(ctx, doc)
    fmt.Println("\n=== TERMINAL OUTPUT ===") 
    fmt.Println(string(tableBytes))
}
```

### Advanced Usage with Custom Formatters

```go
// Security scan results with nested details
func SecurityScanFormatter() func(any) any {
    return func(val any) any {
        if findings, ok := val.([]SecurityFinding); ok {
            summary := fmt.Sprintf("%d findings", len(findings))
            
            // Group by severity for details
            details := map[string][]string{
                "critical": {},
                "high": {},
                "medium": {},
                "low": {},
            }
            
            for _, finding := range findings {
                details[finding.Severity] = append(
                    details[finding.Severity], 
                    fmt.Sprintf("%s: %s", finding.Rule, finding.Message),
                )
            }
            
            return &output.DefaultCollapsibleValue{
                summary: summary,
                details: details,
                defaultExpanded: len(findings) <= 3, // Auto-expand if few findings
                formatHints: map[string]map[string]any{
                    "markdown": {"use_tables": true},
                    "json": {"include_metadata": true},
                },
            }
        }
        return val
    }
}
```

## Implementation Plan

### Phase 1: Core Interface (Week 1)
- [ ] Implement `CollapsibleValue` interface
- [ ] Create `DefaultCollapsibleValue` struct
- [ ] Add helper functions (`CollapsibleFormatter`, `ErrorListFormatter`, etc.)
- [ ] Write comprehensive tests for core functionality

### Phase 2: Renderer Integration (Week 2)
- [ ] Enhance Markdown renderer with `<details>` support
- [ ] Update JSON renderer for structured collapsible data
- [ ] Implement YAML renderer support
- [ ] Add HTML renderer enhancements

### Phase 3: Advanced Features (Week 3)
- [ ] Table renderer with expansion hints
- [ ] CSV renderer with detail columns
- [ ] Format-specific hint system
- [ ] Performance optimization for large datasets

### Phase 4: CLI and Tools (Week 4)
- [ ] Add `--expand` flag support to table renderer
- [ ] Interactive prompts for terminal usage
- [ ] Documentation and examples
- [ ] Integration tests across all formats

## File Changes Required

### New Files
- `v2/collapsible.go` - Core interface and implementations
- `v2/collapsible_test.go` - Comprehensive test suite
- `v2/formatters.go` - Common formatter helpers

### Modified Files
- `v2/markdown_renderer.go` - Add collapsible cell formatting
- `v2/json_yaml_renderer.go` - Add structured collapsible support
- `v2/table_renderer.go` - Add expansion hints and formatting
- `v2/html_renderer.go` - Add HTML5 details support
- `v2/csv_renderer.go` - Add detail column generation
- `v2/content.go` - Update field value processing (if needed)

### Test Files
- `v2/collapsible_integration_test.go` - Cross-format tests
- Update existing renderer tests to include collapsible scenarios

## Backward Compatibility

This design maintains full backward compatibility:
- Existing `Field.Formatter` functions continue to work unchanged
- Only formatters that return `CollapsibleValue` get special treatment
- All existing APIs remain the same
- No breaking changes to the public interface

## Performance Considerations

- **Memory**: Collapsible values store both summary and details, but only when explicitly created
- **Rendering**: Format detection adds minimal overhead (single type assertion)
- **Streaming**: Large detail content can be streamed in JSON/YAML renderers
- **Caching**: Format hints allow renderer-specific optimizations

## Future Enhancements

1. **Interactive CLI**: Real-time expansion/collapse in terminal
2. **Web UI**: JavaScript-enhanced HTML rendering
3. **Export Options**: Dedicated detail export formats
4. **Accessibility**: Screen reader support for collapsible content
5. **Theming**: Customizable collapse/expand indicators
6. **Nested Collapsible**: Multi-level hierarchical content

## Questions for Implementation

1. Should we add a global expansion setting to Document for easy override?
2. Do we need format-specific CollapsibleValue implementations?
3. Should CSV detail columns be opt-in via options?
4. How should we handle very large detail content (memory/performance)?
5. Should we add built-in formatters for common data types (timestamps, URLs, etc.)?

---

This design provides a comprehensive, format-aware solution for collapsible content that integrates cleanly with the existing v2 architecture while maintaining backward compatibility and enabling powerful new functionality across all output formats.