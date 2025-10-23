package output

import (
	"context"
	"testing"
)

// Benchmark_100ItemsWith10Transformations benchmarks 100 content items with 10 transformations each
// Performance target: should complete without degradation
func Benchmark_100ItemsWith10Transformations(b *testing.B) {
	// Create 100 content items
	contents := make([]Content, 100)
	for i := range 100 {
		// Create transformations for each content item
		transformations := []Operation{
			NewFilterOp(func(r Record) bool {
				return r["age"].(int) >= 20
			}),
			NewSortOp(SortKey{Column: "age", Direction: Ascending}),
			NewLimitOp(50),
			NewFilterOp(func(r Record) bool {
				return r["age"].(int) <= 60
			}),
			NewSortOp(SortKey{Column: "name", Direction: Ascending}),
			NewLimitOp(40),
			NewFilterOp(func(r Record) bool {
				return r["age"].(int) >= 25
			}),
			NewSortOp(SortKey{Column: "age", Direction: Descending}),
			NewLimitOp(30),
			NewFilterOp(func(r Record) bool {
				return r["age"].(int) <= 55
			}),
		}

		// Create table with 100 records
		records := make([]Record, 100)
		for j := range 100 {
			records[j] = Record{
				"name": string(rune('A' + (j % 26))),
				"age":  20 + (j % 50),
			}
		}

		contents[i] = &TableContent{
			id:      GenerateID(),
			title:   "Test Table",
			records: records,
			schema: &Schema{
				Fields:   []Field{{Name: "name"}, {Name: "age"}},
				keyOrder: []string{"name", "age"},
			},
			transformations: transformations,
		}
	}

	ctx := context.Background()

	// Reset timer before the actual benchmark
	b.ResetTimer()

	// Benchmark loop
	for b.Loop() {
		// Apply transformations to all content items
		for _, content := range contents {
			_, err := applyContentTransformations(ctx, content)
			if err != nil {
				b.Fatalf("Transformation failed: %v", err)
			}
		}
	}
}

// Benchmark_1000RecordsPerTable benchmarks transformation execution on tables with 1000 records
func Benchmark_1000RecordsPerTable(b *testing.B) {
	transformations := []Operation{
		NewFilterOp(func(r Record) bool {
			return r["age"].(int) >= 30
		}),
		NewSortOp(SortKey{Column: "age", Direction: Ascending}),
		NewLimitOp(100),
	}

	// Create table with 1000 records
	records := make([]Record, 1000)
	for i := range 1000 {
		records[i] = Record{
			"name": string(rune('A' + (i % 26))),
			"age":  20 + (i % 60),
		}
	}

	content := &TableContent{
		id:      "benchmark-table",
		title:   "Benchmark Table",
		records: records,
		schema: &Schema{
			Fields:   []Field{{Name: "name"}, {Name: "age"}},
			keyOrder: []string{"name", "age"},
		},
		transformations: transformations,
	}

	ctx := context.Background()

	b.ResetTimer()

	for b.Loop() {
		_, err := applyContentTransformations(ctx, content)
		if err != nil {
			b.Fatalf("Transformation failed: %v", err)
		}
	}
}

// Benchmark_TransformationStorageMemoryOverhead measures memory overhead of transformation storage
func Benchmark_TransformationStorageMemoryOverhead(b *testing.B) {
	transformations := []Operation{
		NewFilterOp(func(r Record) bool { return true }),
		NewSortOp(SortKey{Column: "age", Direction: Ascending}),
		NewLimitOp(10),
		NewFilterOp(func(r Record) bool { return true }),
		NewSortOp(SortKey{Column: "name", Direction: Descending}),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		// Create table with transformations
		_ = &TableContent{
			id:    GenerateID(),
			title: "Test",
			records: []Record{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
			},
			schema: &Schema{
				Fields:   []Field{{Name: "name"}, {Name: "age"}},
				keyOrder: []string{"name", "age"},
			},
			transformations: transformations,
		}
	}
}

// Benchmark_TransformationExecutionTime measures transformation execution time breakdown
func Benchmark_TransformationExecutionTime(b *testing.B) {
	benchmarks := map[string]struct {
		createContent func() Content
		description   string
	}{
		"single_filter": {
			createContent: func() Content {
				return &TableContent{
					id:      "test",
					title:   "Test",
					records: makeTestRecords(1000),
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
					transformations: []Operation{
						NewFilterOp(func(r Record) bool {
							return r["age"].(int) >= 30
						}),
					},
				}
			},
			description: "Single filter on 1000 records",
		},
		"single_sort": {
			createContent: func() Content {
				return &TableContent{
					id:      "test",
					title:   "Test",
					records: makeTestRecords(1000),
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
					transformations: []Operation{
						NewSortOp(SortKey{Column: "age", Direction: Ascending}),
					},
				}
			},
			description: "Single sort on 1000 records",
		},
		"filter_then_sort": {
			createContent: func() Content {
				return &TableContent{
					id:      "test",
					title:   "Test",
					records: makeTestRecords(1000),
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
					transformations: []Operation{
						NewFilterOp(func(r Record) bool {
							return r["age"].(int) >= 30
						}),
						NewSortOp(SortKey{Column: "age", Direction: Ascending}),
					},
				}
			},
			description: "Filter then sort on 1000 records",
		},
		"complex_chain": {
			createContent: func() Content {
				return &TableContent{
					id:      "test",
					title:   "Test",
					records: makeTestRecords(1000),
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
					transformations: []Operation{
						NewFilterOp(func(r Record) bool {
							return r["age"].(int) >= 20
						}),
						NewSortOp(SortKey{Column: "age", Direction: Ascending}),
						NewLimitOp(500),
						NewFilterOp(func(r Record) bool {
							return r["age"].(int) <= 60
						}),
						NewSortOp(SortKey{Column: "name", Direction: Descending}),
					},
				}
			},
			description: "Complex chain of 5 operations on 1000 records",
		},
	}

	ctx := context.Background()

	for name, bm := range benchmarks {
		b.Run(name, func(b *testing.B) {
			content := bm.createContent()
			b.ResetTimer()

			for b.Loop() {
				_, err := applyContentTransformations(ctx, content)
				if err != nil {
					b.Fatalf("Transformation failed: %v", err)
				}
			}
		})
	}
}

// makeTestRecords creates test records for benchmarking
func makeTestRecords(count int) []Record {
	records := make([]Record, count)
	for i := range count {
		records[i] = Record{
			"name": string(rune('A' + (i % 26))),
			"age":  20 + (i % 60),
		}
	}
	return records
}

// Benchmark_CloningOverhead measures the overhead of cloning during transformations
func Benchmark_CloningOverhead(b *testing.B) {
	// Create a table with moderate data
	records := makeTestRecords(100)
	content := &TableContent{
		id:      "test",
		title:   "Test",
		records: records,
		schema: &Schema{
			Fields:   []Field{{Name: "name"}, {Name: "age"}},
			keyOrder: []string{"name", "age"},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = content.Clone()
	}
}

// Benchmark_MultipleClones measures the overhead of multiple clones in a transformation chain
func Benchmark_MultipleClones(b *testing.B) {
	benchmarks := map[string]struct {
		numTransformations int
	}{
		"3_transformations": {
			numTransformations: 3,
		},
		"5_transformations": {
			numTransformations: 5,
		},
		"10_transformations": {
			numTransformations: 10,
		},
	}

	ctx := context.Background()

	for name, bm := range benchmarks {
		b.Run(name, func(b *testing.B) {
			// Create transformations
			transformations := make([]Operation, bm.numTransformations)
			for i := range bm.numTransformations {
				if i%2 == 0 {
					transformations[i] = NewFilterOp(func(r Record) bool { return true })
				} else {
					transformations[i] = NewSortOp(SortKey{Column: "age", Direction: Ascending})
				}
			}

			content := &TableContent{
				id:      "test",
				title:   "Test",
				records: makeTestRecords(100),
				schema: &Schema{
					Fields:   []Field{{Name: "name"}, {Name: "age"}},
					keyOrder: []string{"name", "age"},
				},
				transformations: transformations,
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, err := applyContentTransformations(ctx, content)
				if err != nil {
					b.Fatalf("Transformation failed: %v", err)
				}
			}
		})
	}
}
