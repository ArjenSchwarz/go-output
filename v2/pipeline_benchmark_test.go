package output

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkPipelineFilter tests filter operation performance with various data sizes
func BenchmarkPipelineFilter(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			// Create test data
			records := make([]Record, size)
			for i := range size {
				records[i] = Record{
					"id":     i,
					"value":  i * 10,
					"active": i%2 == 0,
				}
			}

			doc := New().Table("test", records).Build()

			b.ResetTimer()
			for b.Loop() {
				_, err := doc.Pipeline().
					Filter(func(r Record) bool {
						return r["active"].(bool)
					}).
					Execute()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPipelineSort tests sort operation performance with different key counts
func BenchmarkPipelineSort(b *testing.B) {
	sizes := []int{10, 100, 1000}
	keyCounts := []int{1, 2, 3}

	for _, size := range sizes {
		for _, keyCount := range keyCounts {
			b.Run(fmt.Sprintf("size_%d_keys_%d", size, keyCount), func(b *testing.B) {
				// Create test data
				records := make([]Record, size)
				for i := range size {
					records[i] = Record{
						"id":    size - i, // Reverse order
						"name":  fmt.Sprintf("name_%d", i%10),
						"value": i % 100,
					}
				}

				doc := New().Table("test", records).Build()

				// Build sort keys based on keyCount
				var pipeline *Pipeline
				switch keyCount {
				case 1:
					pipeline = doc.Pipeline().Sort(SortKey{Column: "id", Direction: Ascending})
				case 2:
					pipeline = doc.Pipeline().
						SortBy("name", Ascending).
						SortBy("id", Ascending)
				case 3:
					pipeline = doc.Pipeline().
						SortBy("value", Descending).
						SortBy("name", Ascending).
						SortBy("id", Ascending)
				}

				b.ResetTimer()
				for b.Loop() {
					_, err := pipeline.Execute()
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkPipelineAggregation tests aggregation operation performance
func BenchmarkPipelineAggregation(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	groupSizes := []int{10, 50, 100}

	for _, size := range sizes {
		for _, groupSize := range groupSizes {
			b.Run(fmt.Sprintf("size_%d_groups_%d", size, groupSize), func(b *testing.B) {
				// Create test data with groupSize different categories
				records := make([]Record, size)
				for i := range size {
					records[i] = Record{
						"category": fmt.Sprintf("cat_%d", i%groupSize),
						"value":    float64(i * 10),
						"count":    1,
					}
				}

				doc := New().Table("test", records).Build()

				b.ResetTimer()
				for b.Loop() {
					_, err := doc.Pipeline().
						GroupBy([]string{"category"}, map[string]AggregateFunc{
							"total": SumAggregate("value"),
							"count": CountAggregate(),
						}).
						Execute()
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkPipelineComplexChain tests complex pipeline with multiple operations
func BenchmarkPipelineComplexChain(b *testing.B) {
	sizes := []int{100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			// Create test data
			records := make([]Record, size)
			for i := range size {
				records[i] = Record{
					"id":       i,
					"category": fmt.Sprintf("cat_%d", i%10),
					"value":    float64(i * 10),
					"active":   i%3 != 0,
					"score":    100 - (i % 100),
				}
			}

			doc := New().Table("test", records).Build()

			b.ResetTimer()
			for b.Loop() {
				_, err := doc.Pipeline().
					Filter(func(r Record) bool {
						return r["active"].(bool)
					}).
					Sort(SortKey{Column: "score", Direction: Descending}).
					Limit(50).
					Execute()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPipelineAddColumn tests calculated field performance
func BenchmarkPipelineAddColumn(b *testing.B) {
	sizes := []int{100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			// Create test data
			records := make([]Record, size)
			for i := range size {
				records[i] = Record{
					"id":    i,
					"price": float64(i * 10),
					"qty":   i % 10,
				}
			}

			doc := New().Table("test", records).Build()

			b.ResetTimer()
			for b.Loop() {
				_, err := doc.Pipeline().
					AddColumn("total", func(r Record) any {
						price := r["price"].(float64)
						qty := r["qty"].(int)
						return price * float64(qty)
					}).
					Execute()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkManualVsPipeline compares manual data manipulation vs pipeline
func BenchmarkManualVsPipeline(b *testing.B) {
	size := 1000

	// Create test data
	records := make([]Record, size)
	for i := range size {
		records[i] = Record{
			"id":     i,
			"value":  i * 10,
			"active": i%2 == 0,
		}
	}

	b.Run("manual", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			// Manual filter
			filtered := make([]Record, 0, len(records))
			for _, r := range records {
				if r["active"].(bool) {
					filtered = append(filtered, r)
				}
			}

			// Manual sort (simple bubble sort for comparison)
			for i := 0; i < len(filtered)-1; i++ {
				for j := 0; j < len(filtered)-i-1; j++ {
					if filtered[j]["value"].(int) > filtered[j+1]["value"].(int) {
						filtered[j], filtered[j+1] = filtered[j+1], filtered[j]
					}
				}
			}

			// Manual limit
			if len(filtered) > 10 {
				filtered = filtered[:10]
			}

			// Create new document
			_ = New().Table("result", filtered).Build()
		}
	})

	b.Run("pipeline", func(b *testing.B) {
		doc := New().Table("test", records).Build()

		b.ResetTimer()
		for b.Loop() {
			_, err := doc.Pipeline().
				Filter(func(r Record) bool {
					return r["active"].(bool)
				}).
				Sort(SortKey{Column: "value", Direction: Ascending}).
				Limit(10).
				Execute()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkPipelineOptimization tests operation optimization effectiveness
func BenchmarkPipelineOptimization(b *testing.B) {
	size := 1000

	// Create test data
	records := make([]Record, size)
	for i := range size {
		records[i] = Record{
			"id":    i,
			"value": i * 10,
			"type":  fmt.Sprintf("type_%d", i%100),
		}
	}

	doc := New().Table("test", records).Build()

	// Test unoptimized order (sort then filter - processes all records for sort)
	b.Run("unoptimized_sort_then_filter", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			// This should ideally be optimized to filter first
			_, err := doc.Pipeline().
				Sort(SortKey{Column: "value", Direction: Descending}).
				Filter(func(r Record) bool {
					return r["id"].(int) < 100
				}).
				Execute()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Test optimized order (filter then sort - processes fewer records for sort)
	b.Run("optimized_filter_then_sort", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, err := doc.Pipeline().
				Filter(func(r Record) bool {
					return r["id"].(int) < 100
				}).
				Sort(SortKey{Column: "value", Direction: Descending}).
				Execute()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkPipelineMemoryAllocation tests memory allocation patterns
func BenchmarkPipelineMemoryAllocation(b *testing.B) {
	sizes := []int{100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			// Create test data
			records := make([]Record, size)
			for i := range size {
				records[i] = Record{
					"id":    i,
					"value": i * 10,
				}
			}

			doc := New().Table("test", records).Build()

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				_, err := doc.Pipeline().
					Filter(func(r Record) bool {
						return r["id"].(int)%2 == 0
					}).
					Execute()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPipelineWithTimeout tests pipeline performance with timeouts
func BenchmarkPipelineWithTimeout(b *testing.B) {
	size := 100

	// Create test data
	records := make([]Record, size)
	for i := range size {
		records[i] = Record{
			"id":    i,
			"value": i * 10,
		}
	}

	doc := New().Table("test", records).Build()

	b.Run("with_timeout", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, err := doc.Pipeline().
				WithOptions(PipelineOptions{
					MaxOperations:    100,
					MaxExecutionTime: 1 * time.Second,
				}).
				Filter(func(r Record) bool {
					return r["id"].(int)%2 == 0
				}).
				Execute()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("without_timeout", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_, err := doc.Pipeline().
				Filter(func(r Record) bool {
					return r["id"].(int)%2 == 0
				}).
				Execute()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkPipelineConcurrent tests concurrent pipeline execution
func BenchmarkPipelineConcurrent(b *testing.B) {
	size := 100

	// Create test data
	records := make([]Record, size)
	for i := range size {
		records[i] = Record{
			"id":    i,
			"value": i * 10,
		}
	}

	doc := New().Table("test", records).Build()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := doc.Pipeline().
				Filter(func(r Record) bool {
					return r["id"].(int)%2 == 0
				}).
				Execute()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkPipelineGroupByWithAggregates tests GroupBy with multiple aggregate functions
func BenchmarkPipelineGroupByWithAggregates(b *testing.B) {
	size := 1000

	// Create test data
	records := make([]Record, size)
	for i := range size {
		records[i] = Record{
			"category": fmt.Sprintf("cat_%d", i%20),
			"value":    float64(i * 10),
			"count":    1,
			"score":    float64(100 - (i % 100)),
		}
	}

	doc := New().Table("test", records).Build()

	b.Run("single_aggregate", func(b *testing.B) {
		pipeline := doc.Pipeline()
		pipeline.operations = []Operation{
			NewGroupByOp([]string{"category"}, map[string]AggregateFunc{
				"total": SumAggregate("value"),
			}),
		}

		b.ResetTimer()
		for b.Loop() {
			_, err := pipeline.Execute()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("multiple_aggregates", func(b *testing.B) {
		pipeline := doc.Pipeline()
		pipeline.operations = []Operation{
			NewGroupByOp([]string{"category"}, map[string]AggregateFunc{
				"total": SumAggregate("value"),
				"count": CountAggregate(),
				"avg":   AverageAggregate("value"),
				"min":   MinAggregate("score"),
				"max":   MaxAggregate("score"),
			}),
		}

		b.ResetTimer()
		for b.Loop() {
			_, err := pipeline.Execute()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkPipelineContextCancellation tests performance impact of context checking
func BenchmarkPipelineContextCancellation(b *testing.B) {
	size := 1000

	// Create test data
	records := make([]Record, size)
	for i := range size {
		records[i] = Record{
			"id":    i,
			"value": i * 10,
		}
	}

	doc := New().Table("test", records).Build()

	b.Run("with_context_checks", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			ctx := context.Background()
			_, err := doc.Pipeline().
				Filter(func(r Record) bool {
					// Context is checked in Apply method
					return r["id"].(int)%2 == 0
				}).
				ExecuteContext(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
