package output

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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

		expectedErr := "pipeline operations require table content"
		var validationErr *ValidationError
		if !AsError(err, &validationErr) {
			t.Errorf("expected ValidationError, got %T", err)
		}
		if !strings.Contains(err.Error(), expectedErr) {
			t.Errorf("expected error to contain '%s', got '%s'", expectedErr, err.Error())
		}
	})

	t.Run("accepts document with table content", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		pipeline := doc.Pipeline()
		err := pipeline.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("pipeline with mixed content validates correctly", func(t *testing.T) {
		doc := New().
			Text("Header").
			Table("data", []Record{
				{"id": 1, "value": "test"},
			}).
			Text("Footer").
			Build()

		pipeline := doc.Pipeline()
		err := pipeline.Validate()
		if err != nil {
			t.Fatalf("unexpected error for mixed content with table: %v", err)
		}
	})
}

// TestPipelineOperationChaining tests the operation chaining mechanism
func TestPipelineOperationChaining(t *testing.T) {
	t.Run("chains multiple operations", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice", "status": "active"},
				{"id": 2, "name": "Bob", "status": "inactive"},
				{"id": 3, "name": "Charlie", "status": "active"},
			}).
			Build()

		pipeline := doc.Pipeline()

		// Chain operations (methods will be added when implementing)
		// For now, we'll add operations manually to test the structure
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				return r["status"] == "active"
			},
		}
		sortOp := &SortOp{
			keys: []SortKey{
				{Column: "name", Direction: Ascending},
			},
		}
		limitOp := &LimitOp{
			count: 2,
		}

		pipeline.operations = append(pipeline.operations, filterOp, sortOp, limitOp)

		if len(pipeline.operations) != 3 {
			t.Errorf("expected 3 operations, got %d", len(pipeline.operations))
		}

		// Verify operation order
		if _, ok := pipeline.operations[0].(*FilterOp); !ok {
			t.Error("expected first operation to be FilterOp")
		}
		if _, ok := pipeline.operations[1].(*SortOp); !ok {
			t.Error("expected second operation to be SortOp")
		}
		if _, ok := pipeline.operations[2].(*LimitOp); !ok {
			t.Error("expected third operation to be LimitOp")
		}
	})

	t.Run("maintains operation order", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "value": 10},
			}).
			Build()

		pipeline := doc.Pipeline()

		// Add operations in specific order
		ops := []Operation{
			&FilterOp{predicate: func(r Record) bool { return true }},
			&SortOp{keys: []SortKey{{Column: "id", Direction: Ascending}}},
			&LimitOp{count: 1},
		}

		for _, op := range ops {
			pipeline.operations = append(pipeline.operations, op)
		}

		// Verify order is maintained
		for i, op := range ops {
			if pipeline.operations[i] != op {
				t.Errorf("operation at index %d does not match", i)
			}
		}
	})
}

// TestPipelineImmutability tests that pipeline operations preserve document immutability
func TestPipelineImmutability(t *testing.T) {
	t.Run("Execute returns new document", func(t *testing.T) {
		originalRecords := []Record{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
		}

		doc := New().
			Table("test", originalRecords).
			SetMetadata("version", "1.0").
			Build()

		pipeline := doc.Pipeline()

		// Execute pipeline (even without operations)
		newDoc, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if newDoc == doc {
			t.Error("expected new document instance, got same instance")
		}

		// Original document should be unchanged
		originalContents := doc.GetContents()
		if len(originalContents) != 1 {
			t.Error("original document contents changed")
		}

		// New document should have same structure
		newContents := newDoc.GetContents()
		if len(newContents) != 1 {
			t.Error("new document has different content count")
		}
	})

	t.Run("original document remains unchanged after transformation", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "status": "active"},
				{"id": 2, "status": "inactive"},
				{"id": 3, "status": "active"},
			}).
			Build()

		// Get original content count
		originalContents := doc.GetContents()
		originalTable := originalContents[0].(*TableContent)
		originalCount := len(originalTable.records)

		pipeline := doc.Pipeline()
		// Add filter operation
		pipeline.operations = append(pipeline.operations, &FilterOp{
			predicate: func(r Record) bool {
				return r["status"] == "active"
			},
		})

		_, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check original is unchanged
		afterContents := doc.GetContents()
		afterTable := afterContents[0].(*TableContent)
		afterCount := len(afterTable.records)

		if originalCount != afterCount {
			t.Errorf("original document was modified: had %d records, now has %d",
				originalCount, afterCount)
		}
	})
}

// TestPipelineOptions tests pipeline configuration options
func TestPipelineOptions(t *testing.T) {
	t.Run("respects MaxOperations limit", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1},
			}).
			Build()

		pipeline := doc.Pipeline()
		pipeline.options = PipelineOptions{
			MaxOperations: 2,
		}

		// Add operations up to limit
		pipeline.operations = append(pipeline.operations,
			&FilterOp{predicate: func(r Record) bool { return true }},
			&SortOp{keys: []SortKey{{Column: "id", Direction: Ascending}}},
		)

		// Validation should pass
		err := pipeline.Validate()
		if err != nil {
			t.Fatalf("unexpected error with %d operations: %v", len(pipeline.operations), err)
		}

		// Add one more operation
		pipeline.operations = append(pipeline.operations,
			&LimitOp{count: 1},
		)

		// Validation should fail
		err = pipeline.Validate()
		if err == nil {
			t.Fatal("expected error when exceeding MaxOperations")
		}

		var validationErr *ValidationError
		if !AsError(err, &validationErr) {
			t.Errorf("expected ValidationError, got %T", err)
		}
		if !strings.Contains(err.Error(), "pipeline exceeds maximum operations limit: 3 > 2") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("respects MaxExecutionTime", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1},
			}).
			Build()

		pipeline := doc.Pipeline()
		pipeline.options = PipelineOptions{
			MaxExecutionTime: 100 * time.Millisecond,
		}

		// Add a slow operation for testing
		pipeline.operations = append(pipeline.operations, &slowTestOperation{
			delay: 200 * time.Millisecond,
		})

		ctx := context.Background()
		_, err := pipeline.ExecuteContext(ctx)

		if err == nil {
			t.Fatal("expected timeout error")
		}

		// Check if it's a cancelled error
		if !IsCancelled(err) && !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected cancelled error or timeout, got %v", err)
		}
	})
}

// TestPipelineValidation tests operation validation
func TestPipelineValidation(t *testing.T) {
	t.Run("validates empty pipeline", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1},
			}).
			Build()

		pipeline := doc.Pipeline()
		err := pipeline.Validate()
		if err != nil {
			t.Fatalf("empty pipeline should be valid: %v", err)
		}
	})

	t.Run("validates operation compatibility", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1},
			}).
			Build()

		pipeline := doc.Pipeline()

		// Add an invalid operation (missing required fields)
		pipeline.operations = append(pipeline.operations, &invalidTestOperation{})

		err := pipeline.Validate()
		if err == nil {
			t.Fatal("expected validation error for invalid operation")
		}

		var contextErr *ContextError
		if !AsError(err, &contextErr) {
			t.Errorf("expected ContextError, got %T", err)
		}
		if !strings.Contains(err.Error(), "operation is invalid") {
			t.Errorf("expected error to contain 'operation is invalid', got '%s'", err.Error())
		}
	})

	t.Run("validates all operations", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1},
			}).
			Build()

		pipeline := doc.Pipeline()

		// Add mix of valid and invalid operations
		pipeline.operations = append(pipeline.operations,
			&FilterOp{predicate: func(r Record) bool { return true }}, // valid
			&invalidTestOperation{},                                   // invalid
			&SortOp{keys: []SortKey{{Column: "id"}}},                  // valid
		)

		err := pipeline.Validate()
		if err == nil {
			t.Fatal("expected validation error")
		}

		// Should report the first invalid operation
		if !errors.Is(err, errInvalidOperation) {
			t.Errorf("expected wrapped error to be errInvalidOperation")
		}
	})
}

// Test helpers

// slowTestOperation simulates a slow operation for timeout testing
type slowTestOperation struct {
	delay time.Duration
}

func (o *slowTestOperation) Name() string { return "SlowTest" }
func (o *slowTestOperation) Apply(ctx context.Context, content Content) (Content, error) {
	select {
	case <-time.After(o.delay):
		return content, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (o *slowTestOperation) CanOptimize(with Operation) bool { return false }
func (o *slowTestOperation) Validate() error                 { return nil }

// invalidTestOperation simulates an invalid operation for validation testing
type invalidTestOperation struct{}

var errInvalidOperation = errors.New("operation is invalid")

func (o *invalidTestOperation) Name() string { return "TestInvalid" }
func (o *invalidTestOperation) Apply(ctx context.Context, content Content) (Content, error) {
	return nil, errInvalidOperation
}
func (o *invalidTestOperation) CanOptimize(with Operation) bool { return false }
func (o *invalidTestOperation) Validate() error                 { return errInvalidOperation }

// Mock operation types for testing (will be replaced with real implementations)
type FilterOp struct {
	predicate func(Record) bool
}

func (o *FilterOp) Name() string { return "Filter" }
func (o *FilterOp) Apply(ctx context.Context, content Content) (Content, error) {
	// Basic implementation for testing
	tableContent, ok := content.(*TableContent)
	if !ok {
		return nil, errors.New("filter requires table content")
	}

	// Clone the content
	cloned := tableContent.Clone().(*TableContent)

	// Apply filter
	filtered := make([]Record, 0)
	for _, record := range cloned.records {
		if o.predicate(record) {
			filtered = append(filtered, record)
		}
	}
	cloned.records = filtered

	return cloned, nil
}
func (o *FilterOp) CanOptimize(with Operation) bool { return false }
func (o *FilterOp) Validate() error {
	if o.predicate == nil {
		return errors.New("filter predicate is required")
	}
	return nil
}

type SortOp struct {
	keys       []SortKey
	comparator func(a, b Record) int
}

type SortKey struct {
	Column    string
	Direction SortDirection
}

type SortDirection int

const (
	Ascending SortDirection = iota
	Descending
)

func (o *SortOp) Name() string { return "Sort" }
func (o *SortOp) Apply(ctx context.Context, content Content) (Content, error) {
	// Basic implementation for testing
	return content, nil
}
func (o *SortOp) CanOptimize(with Operation) bool { return false }
func (o *SortOp) Validate() error {
	if len(o.keys) == 0 && o.comparator == nil {
		return errors.New("sort requires keys or comparator")
	}
	return nil
}

type LimitOp struct {
	count int
}

func (o *LimitOp) Name() string { return "Limit" }
func (o *LimitOp) Apply(ctx context.Context, content Content) (Content, error) {
	// Basic implementation for testing
	return content, nil
}
func (o *LimitOp) CanOptimize(with Operation) bool { return false }
func (o *LimitOp) Validate() error {
	if o.count < 0 {
		return errors.New("limit count must be non-negative")
	}
	return nil
}
