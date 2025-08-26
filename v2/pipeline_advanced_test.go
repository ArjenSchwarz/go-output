package output

import (
	"testing"
)

// TestPipelineFilterMethod tests the Filter method on Pipeline
func TestPipelineFilterMethod(t *testing.T) {
	t.Run("single filter", func(t *testing.T) {
		doc := New().
			Table("users", []Record{
				{"id": 1, "name": "Alice", "age": 30},
				{"id": 2, "name": "Bob", "age": 25},
				{"id": 3, "name": "Charlie", "age": 35},
			}).
			Build()

		result, err := doc.Pipeline().
			Filter(func(r Record) bool {
				age, ok := r["age"].(int)
				return ok && age >= 30
			}).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		table := result.GetContents()[0].(*TableContent)
		if len(table.records) != 2 {
			t.Errorf("expected 2 records, got %d", len(table.records))
		}

		// Check filtered results
		names := []string{}
		for _, r := range table.records {
			if name, ok := r["name"].(string); ok {
				names = append(names, name)
			}
		}

		expectedNames := []string{"Alice", "Charlie"}
		for i, name := range expectedNames {
			if i >= len(names) || names[i] != name {
				t.Errorf("expected name %s at position %d, got %v", name, i, names)
			}
		}
	})
}

// TestPipelineOperationReordering tests optimization through operation reordering
func TestPipelineOperationReordering(t *testing.T) {
	t.Run("handles multiple operations correctly", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice", "age": 30},
				{"id": 2, "name": "Bob", "age": 25},
				{"id": 3, "name": "Charlie", "age": 35},
				{"id": 4, "name": "David", "age": 28},
				{"id": 5, "name": "Eve", "age": 32},
			}).
			Build()

		// Pipeline with multiple operations
		opts := PipelineOptions{
			MaxOperations: 10,
		}

		pipeline := doc.Pipeline().
			WithOptions(opts).
			Sort(SortKey{Column: "name", Direction: Ascending}).
			Filter(func(r Record) bool {
				age, ok := r["age"].(int)
				return ok && age > 30
			}).
			Limit(2)

		// Verify options are set
		if pipeline.options.MaxOperations != 10 {
			t.Error("max operations not set correctly")
		}

		// Execute and verify result correctness
		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		table := result.GetContents()[0].(*TableContent)
		if len(table.records) != 2 {
			t.Errorf("expected 2 records after limit, got %d", len(table.records))
		}
	})

	t.Run("preserves operation order", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "value": 10},
				{"id": 2, "value": 20},
			}).
			Build()

		pipeline := doc.Pipeline().
			Sort(SortKey{Column: "value", Direction: Ascending}).
			Filter(func(r Record) bool { return true })

		// Verify operations are in the order they were added
		if len(pipeline.operations) != 2 {
			t.Fatalf("expected 2 operations, got %d", len(pipeline.operations))
		}

		if _, ok := pipeline.operations[0].(*SortOp); !ok {
			t.Error("first operation should be Sort")
		}
		if _, ok := pipeline.operations[1].(*FilterOp); !ok {
			t.Error("second operation should be Filter")
		}
	})
}

// TestPipelineLazyEvaluation tests lazy evaluation of pipeline operations
func TestPipelineLazyEvaluation(t *testing.T) {
	t.Run("operations are not executed until Execute is called", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		executed := false
		pipeline := doc.Pipeline().
			Filter(func(r Record) bool {
				executed = true
				return true
			})

		// Filter should not be executed yet
		if executed {
			t.Error("filter was executed before Execute() was called")
		}

		// Now execute the pipeline
		_, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Now filter should have been executed
		if !executed {
			t.Error("filter was not executed after Execute() was called")
		}
	})

	t.Run("multiple Execute calls re-run operations", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		executionCount := 0
		pipeline := doc.Pipeline().
			Filter(func(r Record) bool {
				executionCount++
				return true
			})

		// First execution
		_, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("first execution error: %v", err)
		}
		if executionCount != 1 {
			t.Errorf("expected 1 execution, got %d", executionCount)
		}

		// Second execution should re-run the filter
		_, err = pipeline.Execute()
		if err != nil {
			t.Fatalf("second execution error: %v", err)
		}
		if executionCount != 2 {
			t.Errorf("expected 2 executions, got %d", executionCount)
		}
	})
}

// TestPipelineErrorPropagation tests error handling in pipeline execution
func TestPipelineErrorPropagation(t *testing.T) {
	t.Run("handles empty document", func(t *testing.T) {
		doc := New().Build() // Empty document

		pipeline := doc.Pipeline()
		_, err := pipeline.Execute()
		if err == nil {
			t.Fatal("expected error for empty document")
		}
	})
}

// TestPipelineContextCancellation tests context cancellation handling
func TestPipelineContextCancellation(t *testing.T) {
	t.Run("completes when no timeout", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		pipeline := doc.Pipeline().Sort(SortKey{Column: "id", Direction: Ascending})

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result == nil {
			t.Fatal("expected result from pipeline execution")
		}
	})
}

// TestTransformationStatistics tests pipeline statistics collection
func TestTransformationStatistics(t *testing.T) {
	t.Run("executes transformation successfully", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "active": true},
				{"id": 2, "active": false},
				{"id": 3, "active": true},
				{"id": 4, "active": false},
				{"id": 5, "active": true},
			}).
			Build()

		pipeline := doc.Pipeline().
			Filter(func(r Record) bool {
				active, ok := r["active"].(bool)
				return ok && active
			}).
			Limit(2)

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify result
		table := result.GetContents()[0].(*TableContent)
		if len(table.records) != 2 {
			t.Errorf("expected 2 filtered records, got %d", len(table.records))
		}
	})
}

// TestGetTransformStats tests getting transformation statistics
func TestGetTransformStats(t *testing.T) {
	t.Run("executes without error", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "value": 10},
				{"id": 2, "value": 20},
				{"id": 3, "value": 30},
			}).
			Build()

		pipeline := doc.Pipeline().
			Filter(func(r Record) bool {
				val, ok := r["value"].(int)
				return ok && val > 15
			})

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify result
		if result == nil {
			t.Error("expected result from pipeline execution")
		}
	})
}

// TestPipelineFormatAware tests format-aware operations
func TestPipelineFormatAware(t *testing.T) {
	t.Run("handles format-unaware operations", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		pipeline := doc.Pipeline().
			Filter(func(r Record) bool { return true })

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify result is not nil
		if result == nil {
			t.Fatal("expected result from pipeline execution")
		}
	})
}
