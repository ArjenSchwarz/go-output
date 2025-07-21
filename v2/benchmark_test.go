package output

import (
	"context"
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
	for i := 0; i < rows; i++ {
		data[i] = map[string]any{
			"ID":     i,
			"Name":   "User" + string(rune('A'+i%26)),
			"Email":  "user@example.com",
			"Active": i%2 == 0,
			"Score":  i * 10,
		}
	}
	keys := []string{"ID", "Name", "Email", "Active", "Score"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := New()
		builder.Table("LargeTable", data, WithKeys(keys...))
		builder.Build()
	}
}

// BenchmarkBuilder_MixedContent benchmarks building documents with mixed content
func BenchmarkBuilder_MixedContent(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
		for i := 0; i < b.N; i++ {
			builder := New()
			builder.Table("Test", data, WithKeys(keys...))
			builder.Build()
		}
	})

	b.Run("ReverseOrder", func(b *testing.B) {
		keys := []string{"Z", "Y", "M", "B", "A"}
		for i := 0; i < b.N; i++ {
			builder := New()
			builder.Table("Test", data, WithKeys(keys...))
			builder.Build()
		}
	})

	b.Run("RandomOrder", func(b *testing.B) {
		keys := []string{"M", "Z", "A", "Y", "B"}
		for i := 0; i < b.N; i++ {
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
			for i := 0; i < 10; i++ {
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
		WithFormat(JSON),
		WithWriter(&benchmarkWriter{}),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
		WithFormat(Table),
		WithWriter(&benchmarkWriter{}),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
			WithFormat(JSON),
			WithWriter(&benchmarkWriter{}),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			output.Render(ctx, doc)
		}
	})

	b.Run("WithSort", func(b *testing.B) {
		output := NewOutput(
			WithFormat(JSON),
			WithTransformer(NewSortTransformer("Name", true)),
			WithWriter(&benchmarkWriter{}),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			output.Render(ctx, doc)
		}
	})

	b.Run("MultipleTransformers", func(b *testing.B) {
		output := NewOutput(
			WithFormat(JSON),
			WithTransformer(NewSortTransformer("Score", false)),
			WithTransformer(&EmojiTransformer{}),
			WithTransformer(NewColorTransformer()),
			WithWriter(&benchmarkWriter{}),
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			output.Render(ctx, doc)
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocations
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("SmallDocument", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			doc := New().
				Text("Hello").
				Build()
			_ = doc
		}
	})

	b.Run("LargeDocument", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			builder := New()
			for j := 0; j < 100; j++ {
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
		for i := 0; i < b.N; i++ {
			_ = DetectSchemaFromMap(data)
		}
	})

	b.Run("DetectType", func(b *testing.B) {
		values := []any{
			"string", 123, 45.67, true, nil,
			[]int{1, 2, 3}, map[string]string{"a": "b"},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
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

		for i := 0; i < b.N; i++ {
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

		for i := 0; i < b.N; i++ {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = buf[:0] // Reset buffer
		_, _ = table.AppendText(buf)
	}
}
