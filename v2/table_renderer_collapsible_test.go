package output

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestTableRenderer_CollapsibleValue(t *testing.T) {
	tests := map[string]struct {
		value          any
		formatter      func(any) any
		config         RendererConfig
		expectedOutput string
		description    string
	}{"CollapsibleValue collapsed with default indicator": {

		value: "test data",
		formatter: func(val any) any {
			return NewCollapsibleValue("Summary: test", "Detailed content: test data")
		},
		config:         DefaultRendererConfig,
		expectedOutput: "Summary: test [details hidden - use --expand for full view]",
		description:    "Test Requirements 6.1, 6.6: summary display with default indicator",
	}, "CollapsibleValue expanded by default": {

		value: "test data",
		formatter: func(val any) any {
			return NewCollapsibleValue("Summary: test", "Detailed content here", WithCollapsibleExpanded(true))
		},
		config: DefaultRendererConfig,
		expectedOutput: `Summary: test
  Detailed content here`,
		description: "Test Requirement 6.2: show both summary and indented details when expanded",
	}, "CollapsibleValue with custom indicator": {

		value: "test data",
		formatter: func(val any) any {
			return NewCollapsibleValue("Summary", "Details")
		},
		config: RendererConfig{
			ForceExpansion:       false,
			TableHiddenIndicator: "[expand for details]",
		},
		expectedOutput: "Summary [expand for details]",
		description:    "Test Requirement 6.6: custom indicator text configuration",
	}, "CollapsibleValue with global expansion override": {

		value: "test data",
		formatter: func(val any) any {
			return NewCollapsibleValue("Summary", "Detail content", WithCollapsibleExpanded(false))
		},
		config: RendererConfig{
			ForceExpansion: true,
		},
		expectedOutput: `Summary
  Detail content`,
		description: "Test Requirement 6.7, 13.1: global expansion override",
	}, "CollapsibleValue with string array details": {

		value: []string{"error1", "error2", "error3"},
		formatter: func(val any) any {
			if arr, ok := val.([]string); ok {
				return NewCollapsibleValue("3 errors", arr, WithCollapsibleExpanded(true))
			}
			return val
		},
		config: DefaultRendererConfig,
		expectedOutput: `3 errors
  error1
  error2
  error3`,
		description: "Test Requirement 6.3: proper indentation for multi-line content",
	}, "Non-collapsible value unchanged": {

		value: "regular value",
		formatter: func(val any) any {
			return val // No CollapsibleValue returned
		},
		config:         DefaultRendererConfig,
		expectedOutput: "regular value",
		description:    "Test backward compatibility: non-collapsible values render normally",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create table renderer with test configuration
			renderer := &tableRenderer{
				styleName:         "Default",
				collapsibleConfig: tt.config,
			}

			// Create field with formatter
			field := &Field{
				Name:      "test_field",
				Type:      "string",
				Formatter: tt.formatter,
			}

			// Test formatCellValue method
			result := renderer.formatCellValue(tt.value, field)

			if result != tt.expectedOutput {
				t.Errorf("%s\nExpected:\n%q\nGot:\n%q", tt.description, tt.expectedOutput, result)
			}
		})
	}
}

func TestTableRenderer_DetailFormatting(t *testing.T) {
	renderer := &tableRenderer{
		styleName:         "Default",
		collapsibleConfig: DefaultRendererConfig,
	}

	tests := map[string]struct {
		details  any
		expected string
	}{"Complex data fallback": {

		details:  map[string]string{"key": "value"},
		expected: "  map[key:value]",
	}, "Multi-line string details": {

		details:  "line 1\nline 2\nline 3",
		expected: "  line 1\n  line 2\n  line 3",
	}, "String array details": {

		details:  []string{"item1", "item2", "item3"},
		expected: "  item1\n  item2\n  item3",
	}, "String details with indentation": {

		details:  "single line content",
		expected: "  single line content",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := renderer.formatDetailsForTable(tt.details)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTableRenderer_IndentText(t *testing.T) {
	renderer := &tableRenderer{
		styleName:         "Default",
		collapsibleConfig: DefaultRendererConfig,
	}

	tests := map[string]struct {
		input    string
		expected string
	}{"Empty line in middle": {

		input:    "line 1\n\nline 3",
		expected: "  line 1\n  \n  line 3",
	}, "Empty string": {

		input:    "",
		expected: "  ",
	}, "Multiple lines": {

		input:    "line 1\nline 2\nline 3",
		expected: "  line 1\n  line 2\n  line 3",
	}, "Single line": {

		input:    "single line",
		expected: "  single line",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := renderer.indentText(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTableRenderer_FullTableWithCollapsibleValues(t *testing.T) {
	// Create test data with collapsible formatter
	data := []map[string]any{
		{
			"file":   "/very/long/path/to/file.go",
			"errors": []string{"syntax error", "missing import", "unused variable"},
			"status": "failed",
		},
		{
			"file":   "/short/path.go",
			"errors": []string{},
			"status": "passed",
		},
	}

	// Create error list formatter that creates CollapsibleValue
	errorFormatter := func(val any) any {
		if arr, ok := val.([]string); ok {
			if len(arr) == 0 {
				return "0 errors"
			}
			summary := fmt.Sprintf("%d errors", len(arr))
			return NewCollapsibleValue(summary, arr)
		}
		return val
	}

	// Create table content with schema
	table, err := NewTableContent("Code Analysis", data,
		WithSchema(
			Field{Name: "file", Type: "string"},
			Field{Name: "errors", Type: "array", Formatter: errorFormatter},
			Field{Name: "status", Type: "string"},
		))
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	tests := map[string]struct {
		config         RendererConfig
		expectedInFull string
		description    string
	}{"Collapsed view with default indicator": {

		config:         DefaultRendererConfig,
		expectedInFull: "3 errors [details hidden - use --expand for full view]",
		description:    "Test Requirements 6.1, 6.6: collapsed collapsible values in full table",
	}, "Custom indicator text": {

		config: RendererConfig{
			ForceExpansion:       false,
			TableHiddenIndicator: "[click to expand]",
		},
		expectedInFull: "3 errors [click to expand]",
		description:    "Test Requirement 6.6: custom indicator in full table",
	}, "Expanded view with global expansion": {

		config: RendererConfig{
			ForceExpansion: true,
		},
		expectedInFull: "  syntax error",
		description:    "Test Requirement 6.7: global expansion in full table context",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create renderer with test configuration
			renderer := NewTableRendererWithCollapsible("Default", tt.config)

			// Create document and render
			doc := New().AddContent(table).Build()
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			resultStr := string(result)

			// Check that the expected content appears in the full table output
			if !strings.Contains(resultStr, tt.expectedInFull) {
				t.Errorf("%s\nExpected to find:\n%q\nIn output:\n%s", tt.description, tt.expectedInFull, resultStr)
			}

			// Verify table structure is maintained (should contain headers)
			if !strings.Contains(resultStr, "FILE") || !strings.Contains(resultStr, "ERRORS") || !strings.Contains(resultStr, "STATUS") {
				t.Errorf("Table structure not maintained in output:\n%s", resultStr)
			}
		})
	}
}

func TestTableRenderer_MultipleCollapsibleCells(t *testing.T) {
	// Test Requirement 6.5: consistent formatting across multiple collapsible cells
	data := []map[string]any{
		{
			"errors":   []string{"error1", "error2"},
			"warnings": []string{"warning1", "warning2", "warning3"},
			"file":     "test.go",
		},
		{
			"errors":   []string{"error3"},
			"warnings": []string{},
			"file":     "other.go",
		},
	}

	errorFormatter := func(val any) any {
		if arr, ok := val.([]string); ok {
			if len(arr) == 0 {
				return "0 items"
			}
			return NewCollapsibleValue(fmt.Sprintf("%d items", len(arr)), arr)
		}
		return val
	}

	table, err := NewTableContent("Multiple Collapsible Test", data,
		WithSchema(
			Field{Name: "file", Type: "string"},
			Field{Name: "errors", Type: "array", Formatter: errorFormatter},
			Field{Name: "warnings", Type: "array", Formatter: errorFormatter},
		))
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	renderer := NewTableRendererWithCollapsible("Default", DefaultRendererConfig)
	doc := New().AddContent(table).Build()
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	resultStr := string(result)

	// Verify consistent formatting for multiple collapsible cells
	expectedPatterns := []string{
		"2 items [details hidden - use --expand for full view]", // errors for first row
		"3 items [details hidden - use --expand for full view]", // warnings for first row
		"1 items [details hidden - use --expand for full view]", // errors for second row
		"0 items", // warnings for second row (no collapsible)
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(resultStr, pattern) {
			t.Errorf("Expected pattern %q not found in output:\n%s", pattern, resultStr)
		}
	}
}

func TestTableRenderer_BackwardCompatibility(t *testing.T) {
	// Test Requirement 12: backward compatibility with existing formatters
	data := []map[string]any{
		{"name": "test", "value": 42},
	}

	// Old-style formatter that returns string directly
	oldFormatter := func(val any) any {
		return fmt.Sprintf("Formatted: %v", val)
	}

	table, err := NewTableContent("Compatibility Test", data,
		WithSchema(
			Field{Name: "name", Type: "string"},
			Field{Name: "value", Type: "number", Formatter: oldFormatter},
		))
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	renderer := NewTableRendererWithCollapsible("Default", DefaultRendererConfig)
	doc := New().AddContent(table).Build()
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	resultStr := string(result)

	// Verify old formatter still works
	if !strings.Contains(resultStr, "Formatted: 42") {
		t.Errorf("Old formatter output not found in result:\n%s", resultStr)
	}

	// Verify no collapsible indicators for non-collapsible values
	if strings.Contains(resultStr, "[details hidden") {
		t.Errorf("Unexpected collapsible indicator found for non-collapsible value:\n%s", resultStr)
	}
}
