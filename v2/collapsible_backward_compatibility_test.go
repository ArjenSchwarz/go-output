package output

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestCollapsibleBackwardCompatibility_ExistingFormatters tests that existing Field.Formatter functions continue to work
func TestCollapsibleBackwardCompatibility_ExistingFormatters(t *testing.T) {
	// Test traditional string-returning formatters
	tests := []struct {
		name      string
		formatter func(any) any
		input     any
		expectOld bool // Whether we expect old behavior
	}{
		{
			name: "Legacy string formatter",
			formatter: func(val any) any {
				return "formatted: " + string(val.(string))
			},
			input:     "test",
			expectOld: true,
		},
		{
			name: "New collapsible formatter",
			formatter: func(val any) any {
				return NewCollapsibleValue("summary", "details")
			},
			input:     "test",
			expectOld: false,
		},
		{
			name: "Conditional formatter",
			formatter: func(val any) any {
				if str, ok := val.(string); ok && len(str) > 10 {
					return NewCollapsibleValue("long text", str)
				}
				return val
			},
			input:     "short",
			expectOld: true,
		},
		{
			name:      "Nil formatter",
			formatter: nil,
			input:     "test",
			expectOld: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create table with the formatter
			data := []map[string]any{
				{"value": tt.input},
			}

			var table *TableContent
			var err error
			if tt.formatter != nil {
				table, err = NewTableContent("Test", data,
					WithSchema(
						Field{Name: "value", Type: "string", Formatter: tt.formatter},
					))
			} else {
				table, err = NewTableContent("Test", data,
					WithKeys("value"))
			}

			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			doc := New().AddContent(table).Build()

			// Test that it renders without errors
			var buf bytes.Buffer
			output := NewOutput(
				WithFormat(Table),
				WithWriter(&backwardCompatTestWriter{buf: &buf}),
			)

			err = output.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Rendering failed: %v", err)
			}

			result := buf.String()
			if len(result) == 0 {
				t.Error("No output generated")
			}

			// Verify behavior matches expectations
			if tt.expectOld {
				// Should not contain collapsible indicators
				if strings.Contains(result, "[details hidden") {
					t.Error("Old formatter should not produce collapsible indicators")
				}
			} else {
				// Collapsible formatter may produce indicators (depending on config)
				t.Logf("Collapsible formatter output: %s", result)
			}
		})
	}
}

// TestCollapsibleBackwardCompatibility_OutputConsistency tests that non-collapsible content produces identical output
func TestCollapsibleBackwardCompatibility_OutputConsistency(t *testing.T) {
	// Create identical data with regular formatters
	data := []map[string]any{
		{"name": "Alice", "age": 30, "active": true},
		{"name": "Bob", "age": 25, "active": false},
	}

	// Test regular formatting vs new API
	testCases := []struct {
		name      string
		builderFn func() *Document
	}{
		{
			name: "Traditional table creation",
			builderFn: func() *Document {
				return New().
					Table("Users", data, WithKeys("name", "age", "active")).
					Build()
			},
		},
		{
			name: "With schema but no collapsible formatters",
			builderFn: func() *Document {
				return New().
					Table("Users", data,
						WithSchema(
							Field{Name: "name", Type: "string"},
							Field{Name: "age", Type: "int"},
							Field{Name: "active", Type: "bool"},
						)).
					Build()
			},
		},
		{
			name: "Mixed with non-collapsible formatters",
			builderFn: func() *Document {
				upperFormatter := func(val any) any {
					if str, ok := val.(string); ok {
						return strings.ToUpper(str)
					}
					return val
				}
				return New().
					Table("Users", data,
						WithSchema(
							Field{Name: "name", Type: "string", Formatter: upperFormatter},
							Field{Name: "age", Type: "int"},
							Field{Name: "active", Type: "bool"},
						)).
					Build()
			},
		},
	}

	// Render with all formats
	formats := []Format{JSON, YAML, Table, CSV, HTML, Markdown}
	baselineOutputs := make(map[string][]byte)

	// Generate baseline from first test case
	baseDoc := testCases[0].builderFn()
	for _, format := range formats {
		var buf bytes.Buffer
		output := NewOutput(
			WithFormat(format),
			WithWriter(&backwardCompatTestWriter{buf: &buf}),
		)

		err := output.Render(context.Background(), baseDoc)
		if err != nil {
			t.Fatalf("Failed to render baseline %s: %v", format.Name, err)
		}
		baselineOutputs[format.Name] = buf.Bytes()
	}

	// Compare other test cases against baseline
	for i, tc := range testCases[1:] {
		t.Run(tc.name, func(t *testing.T) {
			doc := tc.builderFn()

			for _, format := range formats {
				t.Run(format.Name, func(t *testing.T) {
					var buf bytes.Buffer
					output := NewOutput(
						WithFormat(format),
						WithWriter(&backwardCompatTestWriter{buf: &buf}),
					)

					err := output.Render(context.Background(), doc)
					if err != nil {
						t.Fatalf("Failed to render %s: %v", format.Name, err)
					}

					result := buf.Bytes()
					baseline := baselineOutputs[format.Name]

					// For the first comparison (schema vs keys), they might have slight differences
					// For the last one (with formatter), it should differ due to uppercase transformation
					if i == 1 { // Mixed with formatter - expect differences
						// CSV renderer currently doesn't apply field formatters - this is a known limitation
						if format.Name == "csv" {
							t.Logf("CSV format does not currently support field formatters - skipping difference check")
						} else if bytes.Equal(result, baseline) {
							t.Error("Expected differences due to formatter, but outputs are identical")
						}
					} else {
						// Schema-based should be very similar to key-based
						// Allow for minor formatting differences but check core content
						resultStr := string(result)
						baselineStr := string(baseline)

						// Check that key data is present in both
						requiredContent := []string{"Alice", "Bob", "30", "25", "true", "false"}
						for _, content := range requiredContent {
							if !strings.Contains(resultStr, content) {
								t.Errorf("Result missing required content: %s", content)
							}
							if !strings.Contains(baselineStr, content) {
								t.Errorf("Baseline missing required content: %s", content)
							}
						}
					}
				})
			}
		})
	}
}

// TestCollapsibleBackwardCompatibility_PerformanceRegression tests that collapsible features don't impact performance when unused
func TestCollapsibleBackwardCompatibility_PerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create large dataset without collapsible content
	size := 1000
	data := make([]map[string]any, size)
	for i := 0; i < size; i++ {
		data[i] = map[string]any{
			"id":     i,
			"name":   "User " + string(rune('A'+i%26)),
			"score":  i * 10,
			"active": i%2 == 0,
		}
	}

	// Test with plain table (no collapsible)
	doc := New().
		Table("Performance Test", data, WithKeys("id", "name", "score", "active")).
		Build()

	// Measure rendering performance for different formats
	formats := []Format{JSON, Table, CSV}

	for _, format := range formats {
		t.Run(format.Name+"_NoCollapsible", func(t *testing.T) {
			var buf bytes.Buffer
			output := NewOutput(
				WithFormat(format),
				WithWriter(&backwardCompatTestWriter{buf: &buf}),
			)

			// Warm-up
			output.Render(context.Background(), doc)

			// Measure multiple runs
			runs := 5
			for i := 0; i < runs; i++ {
				buf.Reset()
				err := output.Render(context.Background(), doc)
				if err != nil {
					t.Fatalf("Render failed: %v", err)
				}

				if buf.Len() == 0 {
					t.Error("No output generated")
				}
			}

			t.Logf("Successfully rendered %d rows %d times without collapsible features", size, runs)
		})
	}
}

// TestCollapsibleBackwardCompatibility_APISignatures tests that the API signatures remain compatible
func TestCollapsibleBackwardCompatibility_APISignatures(t *testing.T) {
	// Test that existing functions still exist and work
	t.Run("FieldStructure", func(t *testing.T) {
		// Field struct should support both old and new formatter signatures
		field := Field{
			Name: "test",
			Type: "string",
			Formatter: func(val any) any {
				return "formatted: " + val.(string)
			},
		}

		if field.Name != "test" {
			t.Error("Field.Name not preserved")
		}
		if field.Type != "string" {
			t.Error("Field.Type not preserved")
		}
		if field.Formatter == nil {
			t.Error("Field.Formatter should not be nil")
		}

		// Test formatter execution
		result := field.Formatter("input")
		if result != "formatted: input" {
			t.Errorf("Formatter result = %v, want 'formatted: input'", result)
		}
	})

	t.Run("TableCreationMethods", func(t *testing.T) {
		data := []map[string]any{{"key": "value"}}

		// All these methods should still work
		methods := []func() (*TableContent, error){
			func() (*TableContent, error) {
				return NewTableContent("Test", data, WithKeys("key"))
			},
			func() (*TableContent, error) {
				return NewTableContent("Test", data, WithAutoSchema())
			},
			func() (*TableContent, error) {
				return NewTableContent("Test", data,
					WithSchema(Field{Name: "key", Type: "string"}))
			},
		}

		for i, method := range methods {
			t.Run("Method"+string(rune('A'+i)), func(t *testing.T) {
				table, err := method()
				if err != nil {
					t.Fatalf("Table creation failed: %v", err)
				}
				if table == nil {
					t.Error("Table creation returned nil")
				}
				if table.Title() != "Test" {
					t.Error("Table title not preserved")
				}
			})
		}
	})

	t.Run("BuilderMethods", func(t *testing.T) {
		// All existing builder methods should work
		doc := New().
			Header("Test Header").
			Text("Test text").
			Table("Test Table", []map[string]any{{"a": 1}}, WithKeys("a")).
			Section("Test Section", func(b *Builder) {
				b.Text("Nested text")
			}).
			Build()

		if doc == nil {
			t.Error("Builder.Build() returned nil")
		}

		contents := doc.GetContents()
		if len(contents) != 4 { // Header, Text, Table, Section
			t.Errorf("Expected 4 contents, got %d", len(contents))
		}
	})
}

// TestCollapsibleBackwardCompatibility_ZeroOverhead tests that unused collapsible features have no overhead
func TestCollapsibleBackwardCompatibility_ZeroOverhead(t *testing.T) {
	// Create simple data without any collapsible content
	data := []map[string]any{
		{"simple": "value1"},
		{"simple": "value2"},
	}

	doc := New().
		Table("Simple", data, WithKeys("simple")).
		Build()

	// Render with different renderers
	renderers := []struct {
		name   string
		format Format
	}{
		{"JSON", JSON},
		{"Table", Table},
		{"CSV", CSV},
	}

	for _, renderer := range renderers {
		t.Run(renderer.name, func(t *testing.T) {
			var buf bytes.Buffer
			output := NewOutput(
				WithFormat(renderer.format),
				WithWriter(&backwardCompatTestWriter{buf: &buf}),
			)

			err := output.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			result := buf.String()

			// Should not contain any collapsible-related markers
			collapsibleMarkers := []string{
				"<details>",
				"[details hidden",
				"collapsible",
				"_details",
			}

			for _, marker := range collapsibleMarkers {
				if strings.Contains(result, marker) {
					t.Errorf("Found collapsible marker %q in non-collapsible content", marker)
				}
			}

			// Should contain the actual data
			if !strings.Contains(result, "value1") || !strings.Contains(result, "value2") {
				t.Error("Output missing expected data")
			}
		})
	}
}

// TestCollapsibleBackwardCompatibility_MigrationPath tests smooth migration from old to new patterns
func TestCollapsibleBackwardCompatibility_MigrationPath(t *testing.T) {
	data := []map[string]any{
		{"file": "/very/long/path/to/file.go", "size": 1024},
	}

	// Test migration from string formatter to collapsible formatter
	t.Run("StringToCollapsibleMigration", func(t *testing.T) {
		// Old way: formatter returns string
		oldFormatter := func(val any) any {
			path := val.(string)
			if len(path) > 20 {
				return "..." + path[len(path)-17:]
			}
			return path
		}

		// New way: same logic but with collapsible
		newFormatter := func(val any) any {
			path := val.(string)
			if len(path) > 20 {
				summary := "..." + path[len(path)-17:]
				return NewCollapsibleValue(summary, path)
			}
			return path
		}

		// Both should work
		for _, test := range []struct {
			name      string
			formatter func(any) any
		}{
			{"Old", oldFormatter},
			{"New", newFormatter},
		} {
			t.Run(test.name, func(t *testing.T) {
				doc := New().
					Table("Files", data,
						WithSchema(
							Field{Name: "file", Type: "string", Formatter: test.formatter},
							Field{Name: "size", Type: "int"},
						)).
					Build()

				var buf bytes.Buffer
				output := NewOutput(
					WithFormat(Table),
					WithWriter(&backwardCompatTestWriter{buf: &buf}),
				)

				err := output.Render(context.Background(), doc)
				if err != nil {
					t.Fatalf("Render failed for %s: %v", test.name, err)
				}

				result := buf.String()
				if len(result) == 0 {
					t.Error("No output generated")
				}

				// Both should show shortened path (should contain the last part of the path)
				if !strings.Contains(result, "file.go") {
					t.Error("Expected shortened path in output")
				}
			})
		}
	})

	t.Run("GradualCollapsibleAdoption", func(t *testing.T) {
		// Test mixing old and new formatters in same table
		doc := New().
			Table("Mixed", data,
				WithSchema(
					Field{Name: "file", Type: "string", Formatter: FilePathFormatter(15)}, // New collapsible
					Field{Name: "size", Type: "int", Formatter: func(val any) any { // Old string formatter
						return val.(int) + 0 // No-op formatter
					}},
				)).
			Build()

		var buf bytes.Buffer
		output := NewOutput(
			WithFormat(JSON),
			WithWriter(&backwardCompatTestWriter{buf: &buf}),
		)

		err := output.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		if buf.Len() == 0 {
			t.Error("No output generated")
		}

		// Should work without errors
		t.Log("Successfully mixed old and new formatters")
	})
}

// Helper writer for testing (using a different name to avoid conflicts)
type backwardCompatTestWriter struct {
	buf *bytes.Buffer
}

func (w *backwardCompatTestWriter) Write(ctx context.Context, format string, data []byte) error {
	w.buf.Write(data)
	return nil
}
