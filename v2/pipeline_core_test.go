package output

import (
	"testing"
)

// TestPipelineStructInitialization tests the Pipeline struct initialization
func TestPipelineStructInitialization(t *testing.T) {
	t.Run("creates pipeline from document", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		pipeline := doc.Pipeline()
		if pipeline == nil {
			t.Fatal("expected pipeline to be created")
		}

		if pipeline.document != doc {
			t.Error("expected pipeline to reference the document")
		}

		if pipeline.operations == nil {
			t.Error("expected operations slice to be initialized")
		}

		if len(pipeline.operations) != 0 {
			t.Error("expected operations to be empty initially")
		}
	})

	t.Run("validates document has table content", func(t *testing.T) {
		// Document with only text content
		doc := New().
			Text("Some text").
			Build()

		pipeline := doc.Pipeline()
		err := pipeline.Validate()
		if err == nil {
			t.Fatal("expected error for non-table content")
		}
		// Just verify we get an error - the exact message may vary
	})

	t.Run("validates document with table content", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		pipeline := doc.Pipeline()
		err := pipeline.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("validates document with mixed content", func(t *testing.T) {
		doc := New().
			Text("Header").
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		pipeline := doc.Pipeline()
		err := pipeline.Validate()
		if err != nil {
			t.Errorf("unexpected error for mixed content: %v", err)
		}
	})
}

// TestPipelineOperationChaining tests chaining of pipeline operations
func TestPipelineOperationChaining(t *testing.T) {
	t.Run("chains filter operations", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice", "age": 30},
				{"id": 2, "name": "Bob", "age": 25},
				{"id": 3, "name": "Charlie", "age": 35},
			}).
			Build()

		pipeline := doc.Pipeline().
			Filter(func(r Record) bool {
				age, ok := r["age"].(int)
				return ok && age > 25
			}).
			Filter(func(r Record) bool {
				name, ok := r["name"].(string)
				return ok && name != "Charlie"
			})

		if len(pipeline.operations) != 2 {
			t.Errorf("expected 2 operations, got %d", len(pipeline.operations))
		}

		// Verify both operations are FilterOps
		for i, op := range pipeline.operations {
			if _, ok := op.(*FilterOp); !ok {
				t.Errorf("operation %d is not a FilterOp: %T", i, op)
			}
		}
	})

	t.Run("chains mixed operations", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 3, "name": "Charlie", "age": 35},
				{"id": 1, "name": "Alice", "age": 30},
				{"id": 2, "name": "Bob", "age": 25},
			}).
			Build()

		pipeline := doc.Pipeline().
			Sort(SortKey{Column: "id", Direction: Ascending}).
			Filter(func(r Record) bool {
				age, ok := r["age"].(int)
				return ok && age >= 30
			}).
			Limit(1)

		if len(pipeline.operations) != 3 {
			t.Errorf("expected 3 operations, got %d", len(pipeline.operations))
		}

		// Verify operation types
		if _, ok := pipeline.operations[0].(*SortOp); !ok {
			t.Errorf("first operation is not SortOp: %T", pipeline.operations[0])
		}
		if _, ok := pipeline.operations[1].(*FilterOp); !ok {
			t.Errorf("second operation is not FilterOp: %T", pipeline.operations[1])
		}
		if _, ok := pipeline.operations[2].(*LimitOp); !ok {
			t.Errorf("third operation is not LimitOp: %T", pipeline.operations[2])
		}
	})
}

// TestPipelineImmutability tests that pipeline operations don't modify original document
func TestPipelineImmutability(t *testing.T) {
	t.Run("original document remains unchanged", func(t *testing.T) {
		originalData := []Record{
			{"id": 3, "name": "Charlie"},
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
		}

		doc := New().
			Table("test", originalData).
			Build()

		// Create pipeline and add operations
		pipeline := doc.Pipeline().
			Sort(SortKey{Column: "id", Direction: Ascending}).
			Filter(func(r Record) bool {
				id, ok := r["id"].(int)
				return ok && id <= 2
			})

		// Execute pipeline
		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify original document is unchanged
		origContent := doc.GetContents()[0].(*TableContent)
		if len(origContent.records) != 3 {
			t.Error("original document was modified")
		}

		// Check first record is still Charlie (id: 3)
		if origContent.records[0]["id"] != 3 {
			t.Error("original document order was modified")
		}

		// Verify result is transformed
		resultContent := result.GetContents()[0].(*TableContent)
		if len(resultContent.records) != 2 {
			t.Errorf("expected 2 records in result, got %d", len(resultContent.records))
		}
		if resultContent.records[0]["id"] != 1 {
			t.Error("result is not sorted correctly")
		}
	})

	t.Run("pipeline instances are independent", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "value": 10},
				{"id": 2, "value": 20},
				{"id": 3, "value": 30},
			}).
			Build()

		// Create two independent pipelines
		pipeline1 := doc.Pipeline().Filter(func(r Record) bool {
			val, _ := r["value"].(int)
			return val > 15
		})

		pipeline2 := doc.Pipeline().Sort(SortKey{Column: "value", Direction: Ascending})

		// Verify they have different operations
		if len(pipeline1.operations) != 1 {
			t.Errorf("pipeline1 should have 1 operation, has %d", len(pipeline1.operations))
		}
		if len(pipeline2.operations) != 1 {
			t.Errorf("pipeline2 should have 1 operation, has %d", len(pipeline2.operations))
		}

		// Verify operation types are different
		if _, ok := pipeline1.operations[0].(*FilterOp); !ok {
			t.Error("pipeline1 should have FilterOp")
		}
		if _, ok := pipeline2.operations[0].(*SortOp); !ok {
			t.Error("pipeline2 should have SortOp")
		}
	})
}

// TestPipelineOptions tests pipeline configuration options
func TestPipelineOptions(t *testing.T) {
	t.Run("applies optimization option", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		opts := PipelineOptions{
			MaxOperations: 50,
		}

		pipeline := doc.Pipeline().
			WithOptions(opts).
			Filter(func(r Record) bool { return true }).
			Sort(SortKey{Column: "id", Direction: Ascending})

		if pipeline.options.MaxOperations != 50 {
			t.Error("max operations not set correctly")
		}
	})

	t.Run("applies max operations option", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		opts := PipelineOptions{
			MaxOperations: 200,
		}

		pipeline := doc.Pipeline().WithOptions(opts)

		if pipeline.options.MaxOperations != 200 {
			t.Errorf("expected max operations 200, got %d", pipeline.options.MaxOperations)
		}
	})

	t.Run("default options when not specified", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		pipeline := doc.Pipeline()

		if pipeline.options.MaxOperations != 100 {
			t.Error("expected default max operations 100")
		}
	})
}

// TestPipelineValidation tests validation of pipeline operations
func TestPipelineValidation(t *testing.T) {
	t.Run("validates empty pipeline", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		pipeline := doc.Pipeline()
		err := pipeline.Validate()
		if err != nil {
			t.Errorf("empty pipeline should be valid: %v", err)
		}
	})

	t.Run("validates operation sequence", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		// Valid sequence
		pipeline := doc.Pipeline().
			Filter(func(r Record) bool { return true }).
			Sort(SortKey{Column: "id", Direction: Ascending}).
			Limit(1)

		err := pipeline.Validate()
		if err != nil {
			t.Errorf("valid pipeline should pass validation: %v", err)
		}
	})
}
