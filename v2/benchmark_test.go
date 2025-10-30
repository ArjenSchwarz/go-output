package output

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// BenchmarkBuilder_Table benchmarks table creation
func BenchmarkBuilder_Table(b *testing.B) {
	data := []map[string]any{
		{"ID": 1, "Name": "Alice", "Email": "alice@example.com"},
		{"ID": 2, "Name": "Bob", "Email": "bob@example.com"},
		{"ID": 3, "Name": "Charlie", "Email": "charlie@example.com"},
	}
	keys := []string{"ID", "Name", "Email"}

	for b.Loop() {
		builder := New()
		builder.Table("Users", data, WithKeys(keys...))
		builder.Build()
	}
}

// BenchmarkBuilder_LargeTable benchmarks large table creation
func BenchmarkBuilder_LargeTable(b *testing.B) {
	// Create large dataset
	rows := 1000
	data := make([]map[string]any, rows)
	for i := range rows {
		data[i] = map[string]any{
			"ID":     i,
			"Name":   "User" + string(rune('A'+i%26)),
			"Email":  "user@example.com",
			"Active": i%2 == 0,
			"Score":  i * 10,
		}
	}
	keys := []string{"ID", "Name", "Email", "Active", "Score"}

	for b.Loop() {
		builder := New()
		builder.Table("LargeTable", data, WithKeys(keys...))
		builder.Build()
	}
}

// BenchmarkBuilder_MixedContent benchmarks building documents with mixed content
func BenchmarkBuilder_MixedContent(b *testing.B) {

	for b.Loop() {
		builder := New()
		builder.
			Header("Document Title").
			Text("Some text content").
			Table("Data", []map[string]any{
				{"A": 1, "B": 2},
				{"A": 3, "B": 4},
			}, WithKeys("A", "B")).
			Section("Details", func(b *Builder) {
				b.Text("Section text")
				b.Table("Nested", []map[string]any{{"X": "x"}}, WithKeys("X"))
			}).
			Raw("html", []byte("<p>HTML</p>"))
		builder.Build()
	}
}

// BenchmarkKeyOrderPreservation benchmarks key order preservation overhead
func BenchmarkKeyOrderPreservation(b *testing.B) {
	// Test with different key orders
	data := []map[string]any{
		{"Z": 1, "A": 2, "M": 3, "B": 4, "Y": 5},
	}

	b.Run("ForwardOrder", func(b *testing.B) {
		keys := []string{"A", "B", "M", "Y", "Z"}
		for b.Loop() {
			builder := New()
			builder.Table("Test", data, WithKeys(keys...))
			builder.Build()
		}
	})

	b.Run("ReverseOrder", func(b *testing.B) {
		keys := []string{"Z", "Y", "M", "B", "A"}
		for b.Loop() {
			builder := New()
			builder.Table("Test", data, WithKeys(keys...))
			builder.Build()
		}
	})

	b.Run("RandomOrder", func(b *testing.B) {
		keys := []string{"M", "Z", "A", "Y", "B"}
		for b.Loop() {
			builder := New()
			builder.Table("Test", data, WithKeys(keys...))
			builder.Build()
		}
	})
}

// BenchmarkConcurrentBuilding benchmarks concurrent document building
func BenchmarkConcurrentBuilding(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			builder := New()
			for i := range 10 {
				builder.Text("Content " + string(rune('A'+i)))
			}
			builder.Build()
		}
	})
}

// BenchmarkRendering_JSON benchmarks JSON rendering
func BenchmarkRendering_JSON(b *testing.B) {
	// Create a document
	doc := New().
		Table("Benchmark", []map[string]any{
			{"ID": 1, "Value": "A"},
			{"ID": 2, "Value": "B"},
			{"ID": 3, "Value": "C"},
		}, WithKeys("ID", "Value")).
		Build()

	output := NewOutput(
		WithFormat(JSON()),
		WithWriter(&benchmarkWriter{}),
	)
	ctx := context.Background()

	for b.Loop() {
		output.Render(ctx, doc)
	}
}

// BenchmarkRendering_Table benchmarks table rendering
func BenchmarkRendering_Table(b *testing.B) {
	// Create a document
	doc := New().
		Table("Benchmark", []map[string]any{
			{"Name": "Alice", "Age": 30, "City": "New York"},
			{"Name": "Bob", "Age": 25, "City": "London"},
			{"Name": "Charlie", "Age": 35, "City": "Tokyo"},
		}, WithKeys("Name", "Age", "City")).
		Build()

	output := NewOutput(
		WithFormat(Table()),
		WithWriter(&benchmarkWriter{}),
	)
	ctx := context.Background()

	for b.Loop() {
		output.Render(ctx, doc)
	}
}

// BenchmarkTransformers benchmarks transformer overhead
func BenchmarkTransformers(b *testing.B) {
	doc := New().
		Table("Data", []map[string]any{
			{"Name": "Zebra", "Score": 10},
			{"Name": "Apple", "Score": 30},
			{"Name": "Mango", "Score": 20},
		}, WithKeys("Name", "Score")).
		Build()

	ctx := context.Background()

	b.Run("NoTransformers", func(b *testing.B) {
		output := NewOutput(
			WithFormat(JSON()),
			WithWriter(&benchmarkWriter{}),
		)

		b.ResetTimer()
		for b.Loop() {
			output.Render(ctx, doc)
		}
	})

	b.Run("WithSort", func(b *testing.B) {
		output := NewOutput(
			WithFormat(JSON()),
			WithTransformer(NewSortTransformer("Name", true)),
			WithWriter(&benchmarkWriter{}),
		)

		b.ResetTimer()
		for b.Loop() {
			output.Render(ctx, doc)
		}
	})

	b.Run("MultipleTransformers", func(b *testing.B) {
		output := NewOutput(
			WithFormat(JSON()),
			WithTransformer(NewSortTransformer("Score", false)),
			WithTransformer(&EmojiTransformer{}),
			WithTransformer(NewColorTransformer()),
			WithWriter(&benchmarkWriter{}),
		)

		b.ResetTimer()
		for b.Loop() {
			output.Render(ctx, doc)
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocations
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("SmallDocument", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			doc := New().
				Text("Hello").
				Build()
			_ = doc
		}
	})

	b.Run("LargeDocument", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			builder := New()
			for j := range 100 {
				builder.Table("Table", []map[string]any{
					{"A": j, "B": j * 2},
				}, WithKeys("A", "B"))
			}
			doc := builder.Build()
			_ = doc
		}
	})
}

// Benchmark writer that discards output
type benchmarkWriter struct{}

func (b *benchmarkWriter) Write(ctx context.Context, format string, data []byte) error {
	// Discard output for benchmarking
	return nil
}

// BenchmarkSchemaDetection benchmarks schema detection performance
func BenchmarkSchemaDetection(b *testing.B) {
	data := map[string]any{
		"ID":          123,
		"Name":        "Test",
		"Email":       "test@example.com",
		"Active":      true,
		"Score":       95.5,
		"Tags":        []string{"a", "b", "c"},
		"Metadata":    map[string]string{"key": "value"},
		"LastUpdated": "2024-01-01",
	}

	b.Run("DetectSchemaFromMap", func(b *testing.B) {
		for b.Loop() {
			_ = DetectSchemaFromMap(data)
		}
	})

	b.Run("DetectType", func(b *testing.B) {
		values := []any{
			"string", 123, 45.67, true, nil,
			[]int{1, 2, 3}, map[string]string{"a": "b"},
		}

		b.ResetTimer()
		for b.Loop() {
			for _, v := range values {
				_ = DetectType(v)
			}
		}
	})
}

// BenchmarkOptions benchmarks option application
func BenchmarkOptions(b *testing.B) {
	b.Run("TableOptions", func(b *testing.B) {
		opts := []TableOption{
			WithKeys("A", "B", "C"),
			WithAutoSchema(),
		}

		for b.Loop() {
			_ = ApplyTableOptions(opts...)
		}
	})

	b.Run("TextOptions", func(b *testing.B) {
		opts := []TextOption{
			WithBold(true),
			WithItalic(true),
			WithColor("red"),
			WithSize(16),
		}

		for b.Loop() {
			_ = ApplyTextOptions(opts...)
		}
	})
}

// BenchmarkAppendText benchmarks the encoding.TextAppender interface
func BenchmarkAppendText(b *testing.B) {
	table := &TableContent{
		title: "Benchmark",
		schema: &Schema{
			Fields: []Field{
				{Name: "ID", Type: "int"},
				{Name: "Name", Type: "string"},
				{Name: "Score", Type: "float"},
			},
			keyOrder: []string{"ID", "Name", "Score"},
		},
		records: []Record{
			{"ID": 1, "Name": "Alice", "Score": 95.5},
			{"ID": 2, "Name": "Bob", "Score": 87.3},
			{"ID": 3, "Name": "Charlie", "Score": 92.1},
		},
	}

	buf := make([]byte, 0, 1024)

	for b.Loop() {
		buf = buf[:0] // Reset buffer
		_, _ = table.AppendText(buf)
	}
}

// ===== COLLAPSIBLE PERFORMANCE BENCHMARKS =====
// These benchmarks verify performance requirements for Task 13.3

// BenchmarkCollapsibleValue benchmarks CollapsibleValue processing overhead
// This verifies Requirement 10.1: minimal overhead for collapsible features
func BenchmarkCollapsibleValue(b *testing.B) {
	b.Run("Creation", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			cv := NewCollapsibleValue("Summary text", "Detail content")
			_ = cv
		}
	})

	b.Run("Summary_Access", func(b *testing.B) {
		cv := NewCollapsibleValue("Summary text", "Detail content")
		b.ResetTimer()
		for b.Loop() {
			_ = cv.Summary()
		}
	})

	b.Run("Details_First_Access", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			cv := NewCollapsibleValue("Summary", "Detail content")
			_ = cv.Details()
		}
	})

	b.Run("Details_Cached_Access", func(b *testing.B) {
		cv := NewCollapsibleValue("Summary", "Detail content")
		cv.Details() // First access to cache
		b.ResetTimer()
		for b.Loop() {
			_ = cv.Details()
		}
	})

	b.Run("FormatHint_Access", func(b *testing.B) {
		cv := NewCollapsibleValue("Summary", "Detail content",
			WithFormatHint("json", map[string]any{"style": "compact"}))
		b.ResetTimer()
		for b.Loop() {
			_ = cv.FormatHint("json")
		}
	})
}

// BenchmarkCollapsibleValue_LargeContent benchmarks memory usage with large detail content
// This verifies Requirements 10.2, 10.4: efficient processing of large datasets
func BenchmarkCollapsibleValue_LargeContent(b *testing.B) {
	// Create large content of different types
	largeString := strings.Repeat("This is a large string content. ", 1000) // ~32KB
	largeArray := make([]string, 1000)
	for i := range largeArray {
		largeArray[i] = fmt.Sprintf("Item %d with some content", i)
	}
	largeMap := make(map[string]any, 100)
	for i := range 100 {
		largeMap[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d_with_content", i)
	}

	b.Run("LargeString_Processing", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			cv := NewCollapsibleValue("Large content", largeString)
			_ = cv.Details()
		}
	})

	b.Run("LargeArray_Processing", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			cv := NewCollapsibleValue("Large array", largeArray)
			_ = cv.Details()
		}
	})

	b.Run("LargeMap_Processing", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			cv := NewCollapsibleValue("Large map", largeMap)
			_ = cv.Details()
		}
	})
}

// BenchmarkCollapsibleValue_Truncation benchmarks character limit truncation
// This verifies Requirements 10.6, 10.7: configurable character limits
func BenchmarkCollapsibleValue_Truncation(b *testing.B) {
	longContent := strings.Repeat("Long content that will be truncated. ", 100) // ~3.7KB

	b.Run("No_Truncation", func(b *testing.B) {
		for b.Loop() {
			cv := NewCollapsibleValue("Summary", longContent, WithMaxLength(0))
			_ = cv.Details()
		}
	})

	b.Run("With_Truncation_500", func(b *testing.B) {
		for b.Loop() {
			cv := NewCollapsibleValue("Summary", longContent, WithMaxLength(500))
			_ = cv.Details()
		}
	})

	b.Run("With_Truncation_100", func(b *testing.B) {
		for b.Loop() {
			cv := NewCollapsibleValue("Summary", longContent, WithMaxLength(100))
			_ = cv.Details()
		}
	})
}

// BenchmarkMemoryOptimizedProcessor benchmarks the memory optimization system
// This verifies Requirements 10.4: avoid redundant transformations
func BenchmarkMemoryOptimizedProcessor(b *testing.B) {
	config := DefaultRendererConfig
	processor := NewMemoryOptimizedProcessor(config)

	largeString := strings.Repeat("Content for memory optimization testing. ", 500) // ~20KB
	largeArray := make([]string, 500)
	for i := range largeArray {
		largeArray[i] = fmt.Sprintf("Array item %d", i)
	}

	b.Run("ProcessLargeString", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = processor.ProcessLargeDetails(largeString, 1000)
		}
	})

	b.Run("ProcessLargeArray", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = processor.ProcessLargeDetails(largeArray, 5000)
		}
	})

	b.Run("BufferPooling", func(b *testing.B) {
		for b.Loop() {
			buf := processor.GetBuffer()
			buf.WriteString("Test content")
			processor.ReturnBuffer(buf)
		}
	})

	b.Run("StringSlicePooling", func(b *testing.B) {
		for b.Loop() {
			slice := processor.GetStringSlice()
			slice = append(slice, "item1", "item2", "item3")
			processor.ReturnStringSlice(slice)
		}
	})
}

// BenchmarkRendering_WithCollapsible benchmarks rendering with collapsible content
// This verifies Requirements 12.4: minimal overhead when collapsible features unused
func BenchmarkRendering_WithCollapsible(b *testing.B) {
	// Create test data with formatters
	formatter := ErrorListFormatter()
	data := []map[string]any{
		{"file": "main.go", "errors": []string{"error1", "error2", "error3"}},
		{"file": "utils.go", "errors": []string{"warning1"}},
		{"file": "config.go", "errors": []string{}},
	}

	schema := &Schema{
		Fields: []Field{
			{Name: "file", Type: "string"},
			{Name: "errors", Type: "array", Formatter: formatter},
		},
		keyOrder: []string{"file", "errors"},
	}

	doc := New().
		Table("ErrorReport", data, WithSchema(schema.Fields...)).
		Build()

	ctx := context.Background()

	b.Run("Markdown_Rendering", func(b *testing.B) {
		output := NewOutput(
			WithFormat(Markdown()),
			WithWriter(&benchmarkWriter{}),
		)
		b.ResetTimer()
		for b.Loop() {
			output.Render(ctx, doc)
		}
	})

	b.Run("JSON_Rendering", func(b *testing.B) {
		output := NewOutput(
			WithFormat(JSON()),
			WithWriter(&benchmarkWriter{}),
		)
		b.ResetTimer()
		for b.Loop() {
			output.Render(ctx, doc)
		}
	})

	b.Run("Table_Rendering", func(b *testing.B) {
		output := NewOutput(
			WithFormat(Table()),
			WithWriter(&benchmarkWriter{}),
		)
		b.ResetTimer()
		for b.Loop() {
			output.Render(ctx, doc)
		}
	})
}

// BenchmarkTypeAssertion benchmarks type assertion overhead
// This verifies Requirement 10.1: minimize type assertions to one per value
func BenchmarkTypeAssertion(b *testing.B) {
	values := []any{
		"string value",
		NewCollapsibleValue("summary", "details"),
		123,
		[]string{"a", "b", "c"},
		map[string]any{"key": "value"},
	}

	b.Run("DirectTypeAssertion", func(b *testing.B) {
		for b.Loop() {
			for _, val := range values {
				if cv, ok := val.(CollapsibleValue); ok {
					_ = cv.Summary()
				}
			}
		}
	})

	b.Run("OptimizedProcessing", func(b *testing.B) {
		baseRenderer := &baseRenderer{}
		for b.Loop() {
			for _, val := range values {
				processed := baseRenderer.processFieldValueOptimized(val, nil)
				if processed.IsCollapsible {
					_ = processed.CollapsibleValue.Summary()
				}
			}
		}
	})
}

// BenchmarkStreamingProcessor benchmarks streaming capabilities for large datasets
// This verifies Requirement 10.2: maintain streaming capabilities
func BenchmarkStreamingProcessor(b *testing.B) {
	config := DefaultRendererConfig
	processor := NewStreamingValueProcessor(config)

	// Create large dataset
	values := make([]any, 1000)
	fields := make([]*Field, 1000)
	for i := range values {
		values[i] = fmt.Sprintf("Value %d", i)
		fields[i] = &Field{Name: fmt.Sprintf("field_%d", i), Type: "string"}
	}

	processFunc := func(pv *ProcessedValue) error {
		_ = pv.String() // Force string conversion
		return nil
	}

	b.Run("BatchProcessing", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = processor.ProcessBatch(values, fields, processFunc)
		}
	})
}

// BenchmarkBackwardCompatibility benchmarks overhead when collapsible features are not used
// This verifies Requirement 12.4: zero overhead when collapsible features unused
func BenchmarkBackwardCompatibility(b *testing.B) {
	data := []map[string]any{
		{"ID": 1, "Name": "Alice", "Value": "A"},
		{"ID": 2, "Name": "Bob", "Value": "B"},
		{"ID": 3, "Name": "Charlie", "Value": "C"},
	}

	// Test with regular formatters (no collapsible)
	b.Run("WithoutCollapsible", func(b *testing.B) {
		doc := New().
			Table("Regular", data, WithKeys("ID", "Name", "Value")).
			Build()

		output := NewOutput(
			WithFormat(JSON()),
			WithWriter(&benchmarkWriter{}),
		)
		ctx := context.Background()

		b.ResetTimer()
		for b.Loop() {
			output.Render(ctx, doc)
		}
	})

	// Test with collapsible formatters
	b.Run("WithCollapsible", func(b *testing.B) {
		schema := &Schema{
			Fields: []Field{
				{Name: "ID", Type: "int"},
				{Name: "Name", Type: "string"},
				{Name: "Value", Type: "string", Formatter: func(val any) any {
					return NewCollapsibleValue(fmt.Sprint(val), "Details for "+fmt.Sprint(val))
				}},
			},
			keyOrder: []string{"ID", "Name", "Value"},
		}

		doc := New().
			Table("Collapsible", data, WithSchema(schema.Fields...)).
			Build()

		output := NewOutput(
			WithFormat(JSON()),
			WithWriter(&benchmarkWriter{}),
		)
		ctx := context.Background()

		b.ResetTimer()
		for b.Loop() {
			output.Render(ctx, doc)
		}
	})
}
