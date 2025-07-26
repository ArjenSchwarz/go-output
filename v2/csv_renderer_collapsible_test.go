package output

import (
	"context"
	"strings"
	"testing"
)

func TestCSVRenderer_CollapsibleValue(t *testing.T) {
	tests := []struct {
		name            string
		data            []map[string]any
		fields          []Field
		expectedHeaders []string
		expectedRows    [][]string
		description     string
	}{
		{
			name: "CollapsibleValue with ErrorListFormatter creates detail columns",
			data: []map[string]any{
				{
					"file":   "main.go",
					"errors": []string{"syntax error", "missing import"},
				},
				{
					"file":   "utils.go",
					"errors": []string{},
				},
			},
			fields: []Field{
				{Name: "file", Type: "string"},
				{Name: "errors", Type: "array", Formatter: ErrorListFormatter()},
			},
			expectedHeaders: []string{"file", "errors", "errors_details"},
			expectedRows: [][]string{
				{"main.go", "2 errors (click to expand)", "syntax error; missing import"},
				{"utils.go", "[]", ""}, // Empty array shows as "[]"
			},
			description: "Test Requirements 8.1, 8.2, 8.3: automatic detail column creation",
		},
		{
			name: "CollapsibleValue with FilePathFormatter handles long paths",
			data: []map[string]any{
				{
					"id":   1,
					"path": "/very/long/path/to/some/deeply/nested/file/with/a/very/long/filename.txt",
				},
				{
					"id":   2,
					"path": "short.txt",
				},
			},
			fields: []Field{
				{Name: "id", Type: "int"},
				{Name: "path", Type: "string", Formatter: FilePathFormatter(20)},
			},
			expectedHeaders: []string{"id", "path", "path_details"},
			expectedRows: [][]string{
				{"1", "...ry/long/filename.txt (show full path)", "/very/long/path/to/some/deeply/nested/file/with/a/very/long/filename.txt"},
				{"2", "short.txt", ""}, // No collapsible for short paths
			},
			description: "Test Requirement 8.2: summary in original, details in new column",
		},
		{
			name: "Multiple collapsible fields create adjacent detail columns",
			data: []map[string]any{
				{
					"name":   "test",
					"errors": []string{"error1", "error2"},
					"config": map[string]any{"debug": true, "verbose": false},
				},
			},
			fields: []Field{
				{Name: "name", Type: "string"},
				{Name: "errors", Type: "array", Formatter: ErrorListFormatter()},
				{Name: "config", Type: "object", Formatter: JSONFormatter(100)}, // Large limit to prevent collapsible
			},
			expectedHeaders: []string{"name", "errors", "errors_details", "config"},
			expectedRows: [][]string{
				{"test", "2 errors (click to expand)", "error1; error2", `map[debug:true verbose:false]`}, // No collapsible for large limit
			},
			description: "Test Requirement 8.4: maintain original order, append detail columns adjacently",
		},
		{
			name: "Complex detail structures are flattened for CSV",
			data: []map[string]any{
				{
					"item": "test",
					"metadata": map[string]any{
						"nested": map[string]any{"level": 2},
						"array":  []any{1, 2, 3},
						"simple": "value",
					},
				},
			},
			fields: []Field{
				{Name: "item", Type: "string"},
				{Name: "metadata", Type: "object", Formatter: func(val any) any {
					return NewCollapsibleValue("Complex metadata", val)
				}},
			},
			expectedHeaders: []string{"item", "metadata", "metadata_details"},
			expectedRows: [][]string{
				{"test", "Complex metadata", ""}, // Will be checked separately due to map order
			},
			description: "Test Requirement 8.5: flatten complex structures for CSV compatibility",
		},
		{
			name: "Non-collapsible fields remain unchanged",
			data: []map[string]any{
				{
					"id":   1,
					"name": "test",
					"desc": "description",
				},
			},
			fields: []Field{
				{Name: "id", Type: "int"},
				{Name: "name", Type: "string"},
				{Name: "desc", Type: "string"},
			},
			expectedHeaders: []string{"id", "name", "desc"},
			expectedRows: [][]string{
				{"1", "test", "description"},
			},
			description: "Test baseline: tables without collapsible fields work normally",
		},
		{
			name: "Mixed collapsible and non-collapsible fields",
			data: []map[string]any{
				{
					"id":     1,
					"errors": []string{"error1"},
					"status": "failed",
				},
				{
					"id":     2,
					"errors": []string{}, // Will not be collapsible
					"status": "success",
				},
			},
			fields: []Field{
				{Name: "id", Type: "int"},
				{Name: "errors", Type: "array", Formatter: ErrorListFormatter()},
				{Name: "status", Type: "string"},
			},
			expectedHeaders: []string{"id", "errors", "errors_details", "status"},
			expectedRows: [][]string{
				{"1", "1 errors (click to expand)", "error1", "failed"},
				{"2", "[]", "", "success"}, // Empty array shows as "[]"
			},
			description: "Test Requirements 8.2, 8.3: mixed collapsible and regular processing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create table with schema
			table, err := NewTableContent("Test Table", tt.data, WithSchema(tt.fields...))
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Create document
			doc := New().AddContent(table).Build()

			// Create CSV renderer with collapsible config
			renderer := NewCSVRendererWithCollapsible(DefaultRendererConfig)

			// Render to CSV
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render CSV: %v", err)
			}

			// Parse CSV output
			lines := strings.Split(strings.TrimSpace(string(result)), "\n")
			if len(lines) < 1 {
				t.Fatalf("Expected at least 1 line in CSV output, got %d", len(lines))
			}

			// Verify headers
			headerLine := lines[0]
			actualHeaders := strings.Split(headerLine, ",")
			if len(actualHeaders) != len(tt.expectedHeaders) {
				t.Errorf("Expected %d headers, got %d\nExpected: %v\nActual: %v",
					len(tt.expectedHeaders), len(actualHeaders), tt.expectedHeaders, actualHeaders)
				t.Errorf("Full CSV output:\n%s", string(result))
			}

			for i, expected := range tt.expectedHeaders {
				if i >= len(actualHeaders) {
					t.Errorf("Missing header at index %d: expected %q", i, expected)
					continue
				}
				// Remove quotes from CSV parsing for comparison
				actual := strings.Trim(actualHeaders[i], `"`)
				if actual != expected {
					t.Errorf("Header mismatch at index %d: expected %q, got %q", i, expected, actual)
				}
			}

			// Verify data rows
			dataLines := lines[1:] // Skip header
			if len(dataLines) != len(tt.expectedRows) {
				t.Errorf("Expected %d data rows, got %d", len(tt.expectedRows), len(dataLines))
			}

			for rowIdx, expectedRow := range tt.expectedRows {
				if rowIdx >= len(dataLines) {
					t.Errorf("Missing data row at index %d", rowIdx)
					continue
				}

				actualRow := parseCSVLine(dataLines[rowIdx])
				if len(actualRow) != len(expectedRow) {
					t.Errorf("Row %d: expected %d columns, got %d\nExpected: %v\nActual: %v",
						rowIdx, len(expectedRow), len(actualRow), expectedRow, actualRow)
					continue
				}

				for colIdx, expected := range expectedRow {
					if colIdx >= len(actualRow) {
						t.Errorf("Row %d, missing column at index %d", rowIdx, colIdx)
						continue
					}
					actual := actualRow[colIdx]

					// Special handling for complex structure test due to map iteration order
					if tt.name == "Complex detail structures are flattened for CSV" && colIdx == 2 && expected == "" {
						// Check that the details contain expected key-value pairs
						if !strings.Contains(actual, "array: [1 2 3]") ||
							!strings.Contains(actual, "nested: map[level:2]") ||
							!strings.Contains(actual, "simple: value") {
							t.Errorf("Row %d, column %d: missing expected content in %q", rowIdx, colIdx, actual)
						}
						continue
					}

					if actual != expected {
						t.Errorf("Row %d, column %d: expected %q, got %q", rowIdx, colIdx, expected, actual)
					}
				}
			}

			t.Logf("✓ %s: %s", tt.name, tt.description)
		})
	}
}

func TestCSVRenderer_FlattenDetails(t *testing.T) {
	tests := []struct {
		name     string
		details  any
		expected string
	}{
		{
			name:     "String details remain unchanged",
			details:  "simple string",
			expected: "simple string",
		},
		{
			name:     "String with newlines replaced",
			details:  "line1\nline2\rline3\tline4",
			expected: "line1 line2 line3 line4",
		},
		{
			name:     "String array joined with semicolons",
			details:  []string{"error1", "error2", "error3"},
			expected: "error1; error2; error3",
		},
		{
			name:     "Map converted to key-value pairs",
			details:  map[string]any{"key1": "value1", "key2": 42},
			expected: "", // Will be checked specially due to map order
		},
		{
			name:     "Generic array joined",
			details:  []any{1, "text", true},
			expected: "1; text; true",
		},
		{
			name:     "Complex nested structure",
			details:  map[string]any{"nested": []any{1, 2}, "simple": "value"},
			expected: "", // Will be checked specially due to map order
		},
		{
			name:     "Nil details returns empty string",
			details:  nil,
			expected: "",
		},
	}

	renderer := &csvRenderer{collapsibleConfig: DefaultRendererConfig}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := renderer.flattenDetails(tt.details)

			// Special handling for map-based tests due to iteration order
			if tt.name == "Map converted to key-value pairs" && tt.expected == "" {
				if !strings.Contains(actual, "key1: value1") || !strings.Contains(actual, "key2: 42") {
					t.Errorf("Expected map content not found in %q", actual)
				}
				return
			}

			if tt.name == "Complex nested structure" && tt.expected == "" {
				if !strings.Contains(actual, "nested: [1 2]") || !strings.Contains(actual, "simple: value") {
					t.Errorf("Expected nested structure content not found in %q", actual)
				}
				return
			}

			if actual != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestCSVRenderer_DetectCollapsibleFields(t *testing.T) {
	tests := []struct {
		name     string
		data     []map[string]any
		fields   []Field
		expected map[string]bool
	}{
		{
			name: "Detect ErrorListFormatter field",
			data: []map[string]any{
				{"errors": []string{"error1", "error2"}},
			},
			fields: []Field{
				{Name: "errors", Type: "array", Formatter: ErrorListFormatter()},
			},
			expected: map[string]bool{"errors": true},
		},
		{
			name: "No collapsible fields detected",
			data: []map[string]any{
				{"name": "test", "value": 123},
			},
			fields: []Field{
				{Name: "name", Type: "string"},
				{Name: "value", Type: "int"},
			},
			expected: map[string]bool{},
		},
		{
			name: "Mixed collapsible and non-collapsible fields",
			data: []map[string]any{
				{"name": "test", "errors": []string{"error1"}, "count": 5},
			},
			fields: []Field{
				{Name: "name", Type: "string"},
				{Name: "errors", Type: "array", Formatter: ErrorListFormatter()},
				{Name: "count", Type: "int"},
			},
			expected: map[string]bool{"errors": true},
		},
	}

	renderer := &csvRenderer{collapsibleConfig: DefaultRendererConfig}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, err := NewTableContent("Test", tt.data, WithSchema(tt.fields...))
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			actual := renderer.detectCollapsibleFields(table)

			if len(actual) != len(tt.expected) {
				t.Errorf("Expected %d collapsible fields, got %d", len(tt.expected), len(actual))
			}

			for field, expectedPresent := range tt.expected {
				if actualPresent := actual[field]; actualPresent != expectedPresent {
					t.Errorf("Field %q: expected %v, got %v", field, expectedPresent, actualPresent)
				}
			}
		})
	}
}

// parseCSVLine is a simple CSV line parser for testing purposes
func parseCSVLine(line string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false

	for i, char := range line {
		switch char {
		case '"':
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				// Escaped quote
				current.WriteByte('"')
				i++ // Skip next quote
			} else {
				inQuotes = !inQuotes
			}
		case ',':
			if !inQuotes {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}
	result = append(result, current.String())
	return result
}
