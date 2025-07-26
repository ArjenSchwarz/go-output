package output

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestCollapsibleIntegration_RealWorldScenarios tests complete end-to-end collapsible workflows
func TestCollapsibleIntegration_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name             string
		scenarioBuilder  func() *Document
		expectedBehavior map[string]func([]byte, *testing.T)
	}{
		{
			name: "GitHub PR Comment Analysis",
			scenarioBuilder: func() *Document {
				// Simulate analysis tool generating PR comment
				data := []map[string]any{
					{
						"file":   "/very/long/path/to/src/components/UserProfile.tsx",
						"errors": []string{"Missing import for React", "Unused variable 'userData'", "Type annotation missing for 'props'"},
						"config": map[string]any{"eslint": true, "typescript": true, "prettier": false},
						"lines":  150,
					},
					{
						"file":   "/another/long/path/to/utils/helpers.ts",
						"errors": []string{"Deprecated method usage"},
						"config": map[string]any{"eslint": true, "typescript": true, "prettier": true},
						"lines":  75,
					},
				}

				return New().
					Header("Code Analysis Results").
					Text("Found issues in 2 files requiring attention.").
					Table("File Analysis", data,
						WithSchema(
							Field{Name: "file", Type: "string", Formatter: FilePathFormatter(30)},
							Field{Name: "errors", Type: "array", Formatter: ErrorListFormatter()},
							Field{Name: "config", Type: "object", Formatter: JSONFormatter(50)},
							Field{Name: "lines", Type: "int"},
						)).
					Build()
			},
			expectedBehavior: map[string]func([]byte, *testing.T){
				"markdown": func(output []byte, t *testing.T) {
					str := string(output)
					// Should contain details elements for collapsible content
					if !strings.Contains(str, "<details>") {
						t.Error("Markdown should contain <details> elements for collapsible content")
					}
					if !strings.Contains(str, "<summary>") {
						t.Error("Markdown should contain <summary> elements")
					}
					// File paths should be shortened in summary
					if !strings.Contains(str, "UserProfile.tsx (show full path)") {
						t.Error("File paths should be shortened in summary view")
					}
				},
				"json": func(output []byte, t *testing.T) {
					var result []any
					if err := json.Unmarshal(output, &result); err != nil {
						t.Fatalf("Failed to parse JSON output: %v", err)
					}
					// Should contain multiple content items (header, text, table)
					if len(result) < 3 {
						t.Error("JSON should contain structured contents array with multiple items")
					}
				},
				"table": func(output []byte, t *testing.T) {
					str := string(output)
					// Should show expansion indicators for hidden details
					if !strings.Contains(str, "[details hidden") {
						t.Error("Table should show expansion indicators for collapsible content")
					}
				},
			},
		},
		{
			name: "API Response with Nested Data",
			scenarioBuilder: func() *Document {
				// Simulate API analysis with complex nested data
				responseData := []map[string]any{
					{
						"endpoint": "/api/users",
						"requests": 1500,
						"errors":   []string{"Rate limit exceeded", "Invalid auth token", "Missing user ID"},
						"performance": map[string]any{
							"avg_response_time": "250ms",
							"p95_response_time": "500ms",
							"success_rate":      0.95,
						},
					},
					{
						"endpoint": "/api/orders",
						"requests": 800,
						"errors":   []string{},
						"performance": map[string]any{
							"avg_response_time": "150ms",
							"p95_response_time": "300ms",
							"success_rate":      0.99,
						},
					},
				}

				return New().
					Header("API Performance Analysis").
					Table("Endpoint Statistics", responseData,
						WithSchema(
							Field{Name: "endpoint", Type: "string"},
							Field{Name: "requests", Type: "int"},
							Field{Name: "errors", Type: "array", Formatter: ErrorListFormatter(WithExpanded(false))},
							Field{Name: "performance", Type: "object", Formatter: JSONFormatter(100, WithExpanded(true))},
						)).
					Build()
			},
			expectedBehavior: map[string]func([]byte, *testing.T){
				"json": func(output []byte, t *testing.T) {
					var result []any
					if err := json.Unmarshal(output, &result); err != nil {
						t.Fatalf("Failed to parse JSON: %v", err)
					}
					// Check for collapsible structure in JSON (should have multiple content items)
					if len(result) == 0 {
						t.Error("JSON should contain content items")
					}
				},
				"yaml": func(output []byte, t *testing.T) {
					var result []any
					if err := yaml.Unmarshal(output, &result); err != nil {
						t.Fatalf("Failed to parse YAML: %v", err)
					}
					// Should maintain YAML structure with collapsible data
					if len(result) == 0 {
						t.Error("YAML should contain content items")
					}
				},
			},
		},
		{
			name: "CSV Export with Detail Columns",
			scenarioBuilder: func() *Document {
				// Simulate data export scenario
				exportData := []map[string]any{
					{
						"product":      "Laptop Pro",
						"specs":        map[string]any{"cpu": "M2 Pro", "ram": "16GB", "storage": "512GB SSD"},
						"reviews":      []string{"Excellent performance", "Great battery life", "Expensive but worth it"},
						"price":        2499.99,
						"availability": true,
					},
					{
						"product":      "Desktop Workstation",
						"specs":        map[string]any{"cpu": "Intel i9", "ram": "32GB", "storage": "1TB NVMe"},
						"reviews":      []string{"Powerful for development", "Quiet operation"},
						"price":        3999.99,
						"availability": false,
					},
				}

				return New().
					Table("Product Catalog", exportData,
						WithSchema(
							Field{Name: "product", Type: "string"},
							Field{Name: "specs", Type: "object", Formatter: JSONFormatter(50)},
							Field{Name: "reviews", Type: "array", Formatter: ErrorListFormatter()}, // Reusing for string arrays
							Field{Name: "price", Type: "float"},
							Field{Name: "availability", Type: "bool"},
						)).
					Build()
			},
			expectedBehavior: map[string]func([]byte, *testing.T){
				"csv": func(output []byte, t *testing.T) {
					lines := strings.Split(string(output), "\n")
					if len(lines) < 2 {
						t.Fatal("CSV should have header and data rows")
					}

					// Check for detail columns
					header := lines[0]
					if !strings.Contains(header, "specs_details") {
						t.Error("CSV should contain specs_details column for collapsible specs field")
					}
					if !strings.Contains(header, "reviews_details") {
						t.Error("CSV should contain reviews_details column for collapsible reviews field")
					}

					// Check data rows have summary and detail content
					if len(lines) > 1 && lines[1] != "" {
						// Should have summary in main column and details in detail column
						if !strings.Contains(lines[1], "JSON data") {
							t.Error("CSV should contain summary information in main columns")
						}
					}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := tt.scenarioBuilder()

			// Test each format with expected behaviors
			for formatName, behaviorTest := range tt.expectedBehavior {
				t.Run(formatName, func(t *testing.T) {
					var buf bytes.Buffer
					var format Format
					switch formatName {
					case "json":
						format = JSON
					case "yaml":
						format = YAML
					case "markdown":
						format = Markdown
					case "table":
						format = Table
					case "html":
						format = HTML
					case "csv":
						format = CSV
					default:
						t.Fatalf("Unknown format: %s", formatName)
					}

					output := NewOutput(
						WithFormat(format),
						WithWriter(&testWriter{buf: &buf}),
					)

					err := output.Render(context.Background(), doc)
					if err != nil {
						t.Fatalf("Failed to render %s: %v", formatName, err)
					}

					if buf.Len() == 0 {
						t.Fatal("No output generated")
					}

					behaviorTest(buf.Bytes(), t)
				})
			}
		})
	}
}

// TestCollapsibleIntegration_CrossFormatConsistency tests that collapsible content behaves consistently across formats
func TestCollapsibleIntegration_CrossFormatConsistency(t *testing.T) {
	// Create document with known collapsible content
	testData := []map[string]any{
		{
			"name":   "Test Item",
			"errors": []string{"error1", "error2", "error3"},
			"config": map[string]any{"debug": true, "verbose": false},
			"path":   "/very/long/path/that/should/be/truncated/in/summary/view/file.go",
		},
	}

	doc := New().
		Table("Cross-Format Test", testData,
			WithSchema(
				Field{Name: "name", Type: "string"},
				Field{Name: "errors", Type: "array", Formatter: ErrorListFormatter()},
				Field{Name: "config", Type: "object", Formatter: JSONFormatter(30)},
				Field{Name: "path", Type: "string", Formatter: FilePathFormatter(25)},
			)).
		Build()

	formats := []Format{JSON, YAML, Markdown, Table, HTML, CSV}
	outputs := make(map[string][]byte)

	// Render in all formats
	for _, format := range formats {
		var buf bytes.Buffer
		output := NewOutput(
			WithFormat(format),
			WithWriter(&testWriter{buf: &buf}),
		)

		err := output.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Failed to render %s format: %v", format.Name, err)
		}

		outputs[format.Name] = buf.Bytes()
	}

	// Verify consistency across formats
	t.Run("CollapsibleContentPresent", func(t *testing.T) {
		for format, output := range outputs {
			str := string(output)
			switch format {
			case "markdown":
				if !strings.Contains(str, "<details>") {
					t.Errorf("%s: should contain collapsible details elements", format)
				}
			case "json":
				if !strings.Contains(str, `"type":"collapsible"`) && !strings.Contains(str, `"type": "collapsible"`) {
					t.Errorf("%s: should contain collapsible type indicators", format)
				}
			case "table":
				if !strings.Contains(str, "[details hidden") {
					t.Errorf("%s: should contain expansion indicators", format)
				}
			case "html":
				if !strings.Contains(str, "<details") {
					t.Errorf("%s: should contain HTML details elements", format)
				}
			case "csv":
				if !strings.Contains(str, "_details") {
					t.Errorf("%s: should contain detail columns", format)
				}
			}
		}
	})

	t.Run("SummaryConsistency", func(t *testing.T) {
		// All formats should show some form of summary for collapsible content
		summaryPatterns := map[string][]string{
			"markdown": {"3 errors (click to expand)", "file.go (show full path)", "JSON data"},
			"json":     {"3 errors (click to expand)", "file.go (show full path)", "JSON data"},
			"yaml":     {"3 errors (click to expand)", "file.go (show full path)", "JSON data"},
			"table":    {"3 errors (click to expand)", "file.go (show full path)", "JSON data"},
			"html":     {"3 errors (click to expand)", "file.go (show full path)", "JSON data"},
			"csv":      {"3 errors (click to expand)", "file.go (show full path)", "JSON data"},
		}

		for format, patterns := range summaryPatterns {
			output := string(outputs[format])
			for _, pattern := range patterns {
				if !strings.Contains(output, pattern) {
					t.Errorf("%s: should contain summary pattern %q", format, pattern)
				}
			}
		}
	})
}

// TestCollapsibleIntegration_GlobalExpansionControl tests global expansion settings across renderers
func TestCollapsibleIntegration_GlobalExpansionControl(t *testing.T) {
	testData := []map[string]any{
		{
			"item":    "Test",
			"details": []string{"detail1", "detail2"},
		},
	}

	doc := New().
		Table("Expansion Test", testData,
			WithSchema(
				Field{Name: "item", Type: "string"},
				Field{Name: "details", Type: "array", Formatter: ErrorListFormatter(WithExpanded(false))}, // Explicitly collapsed
			)).
		Build()

	t.Run("DefaultBehavior", func(t *testing.T) {
		// Default: should respect individual CollapsibleValue settings (collapsed)
		var buf bytes.Buffer
		tableRenderer := NewTableRendererWithCollapsible("Default", DefaultRendererConfig)
		format := Format{Name: "table", Renderer: tableRenderer}

		output := NewOutput(
			WithFormat(format),
			WithWriter(&testWriter{buf: &buf}),
		)

		err := output.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Failed to render: %v", err)
		}

		result := string(buf.Bytes())
		if !strings.Contains(result, "[details hidden") {
			t.Error("Should show expansion indicators when collapsed by default")
		}
	})

	t.Run("GlobalExpansionEnabled", func(t *testing.T) {
		// Global expansion: should override individual settings
		var buf bytes.Buffer
		config := RendererConfig{
			ForceExpansion:       true,
			MaxDetailLength:      500,
			TruncateIndicator:    "[...truncated]",
			TableHiddenIndicator: "[details hidden - use --expand for full view]",
		}
		tableRenderer := NewTableRendererWithCollapsible("Default", config)
		format := Format{Name: "table", Renderer: tableRenderer}

		output := NewOutput(
			WithFormat(format),
			WithWriter(&testWriter{buf: &buf}),
		)

		err := output.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Failed to render: %v", err)
		}

		result := string(buf.Bytes())
		if strings.Contains(result, "[details hidden") {
			t.Error("Should not show expansion indicators when global expansion is enabled")
		}
		// Should show the actual details
		if !strings.Contains(result, "detail1") || !strings.Contains(result, "detail2") {
			t.Error("Should show expanded details when global expansion is enabled")
		}
	})
}

// TestCollapsibleIntegration_ErrorRecovery tests error handling and graceful degradation
func TestCollapsibleIntegration_ErrorRecovery(t *testing.T) {
	// Create problematic data
	testData := []map[string]any{
		{
			"normal":        "regular value",
			"nil_details":   "value with nil details formatter",
			"empty_summary": "",
		},
	}

	// Formatter that returns nil details (should fallback gracefully)
	nilDetailsFormatter := func(val any) any {
		return NewCollapsibleValue("summary", nil) // nil details
	}

	// Formatter that returns empty summary
	emptySummaryFormatter := func(val any) any {
		return NewCollapsibleValue("", "some details") // empty summary
	}

	doc := New().
		Table("Error Recovery Test", testData,
			WithSchema(
				Field{Name: "normal", Type: "string"},
				Field{Name: "nil_details", Type: "string", Formatter: nilDetailsFormatter},
				Field{Name: "empty_summary", Type: "string", Formatter: emptySummaryFormatter},
			)).
		Build()

	formats := []Format{JSON, Markdown, Table, HTML}

	for _, format := range formats {
		t.Run(format.Name, func(t *testing.T) {
			var buf bytes.Buffer
			output := NewOutput(
				WithFormat(format),
				WithWriter(&testWriter{buf: &buf}),
			)

			// Should not panic or fail despite problematic collapsible values
			err := output.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Rendering should not fail due to collapsible errors: %v", err)
			}

			result := string(buf.Bytes())

			// Should contain fallback placeholders
			switch format.Name {
			case "json", "markdown", "table", "html":
				if strings.Contains(result, "[no summary]") {
					t.Log("Good: Found fallback placeholder for empty summary")
				}
			}

			// Should not be empty
			if len(result) == 0 {
				t.Error("Output should not be empty even with problematic collapsible values")
			}
		})
	}
}

// TestCollapsibleIntegration_PerformanceWithLargeData tests performance with large collapsible datasets
func TestCollapsibleIntegration_PerformanceWithLargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create large dataset with collapsible content
	size := 1000
	data := make([]map[string]any, size)
	for i := 0; i < size; i++ {
		data[i] = map[string]any{
			"id":     i,
			"errors": []string{"error1", "error2", "error3", "error4", "error5"},
			"config": map[string]any{"debug": true, "verbose": i%2 == 0, "level": i % 5},
			"path":   "/very/long/path/to/file/number/" + string(rune('A'+i%26)) + "/test.go",
		}
	}

	doc := New().
		Table("Performance Test", data,
			WithSchema(
				Field{Name: "id", Type: "int"},
				Field{Name: "errors", Type: "array", Formatter: ErrorListFormatter()},
				Field{Name: "config", Type: "object", Formatter: JSONFormatter(100)},
				Field{Name: "path", Type: "string", Formatter: FilePathFormatter(30)},
			)).
		Build()

	// Test performance across formats
	formats := []Format{JSON, Table, Markdown}
	for _, format := range formats {
		t.Run(format.Name, func(t *testing.T) {
			var buf bytes.Buffer
			output := NewOutput(
				WithFormat(format),
				WithWriter(&testWriter{buf: &buf}),
			)

			err := output.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render large dataset in %s: %v", format.Name, err)
			}

			if buf.Len() == 0 {
				t.Error("No output generated for large dataset")
			}

			t.Logf("%s format generated %d bytes for %d rows with collapsible content",
				format.Name, buf.Len(), size)
		})
	}
}

// Test helper writer
type testWriter struct {
	buf *bytes.Buffer
}

func (w *testWriter) Write(ctx context.Context, format string, data []byte) error {
	w.buf.Write(data)
	return nil
}
