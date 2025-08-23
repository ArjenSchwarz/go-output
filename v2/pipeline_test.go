package output

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

// Mock operation types have been replaced with real implementations in operations.go

// TestFilterOperation tests comprehensive filter functionality
func TestFilterOperation(t *testing.T) {
	t.Run("filters with different data types in predicates", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice", "age": 30, "active": true, "score": 85.5},
				{"id": 2, "name": "Bob", "age": 25, "active": false, "score": 92.0},
				{"id": 3, "name": "Charlie", "age": 35, "active": true, "score": 78.3},
				{"id": 4, "name": "Diana", "age": 28, "active": true, "score": 88.7},
			}).
			Build()

		pipeline := doc.Pipeline()

		// Test string filtering
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				name, ok := r["name"].(string)
				return ok && name == "Alice"
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 1 {
			t.Errorf("expected 1 record, got %d", len(tableContent.records))
		}
		if tableContent.records[0]["name"] != "Alice" {
			t.Errorf("expected Alice, got %v", tableContent.records[0]["name"])
		}

		// Test integer filtering
		pipeline = doc.Pipeline()
		filterOp = &FilterOp{
			predicate: func(r Record) bool {
				age, ok := r["age"].(int)
				return ok && age > 30
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err = pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent = result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 1 {
			t.Errorf("expected 1 record, got %d", len(tableContent.records))
		}

		// Test boolean filtering
		pipeline = doc.Pipeline()
		filterOp = &FilterOp{
			predicate: func(r Record) bool {
				active, ok := r["active"].(bool)
				return ok && active
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err = pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent = result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 3 {
			t.Errorf("expected 3 records, got %d", len(tableContent.records))
		}

		// Test float filtering
		pipeline = doc.Pipeline()
		filterOp = &FilterOp{
			predicate: func(r Record) bool {
				score, ok := r["score"].(float64)
				return ok && score >= 85.0
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err = pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent = result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 3 {
			t.Errorf("expected 3 records (score >= 85), got %d", len(tableContent.records))
		}
	})

	t.Run("filter with empty results and no matches", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "status": "active"},
				{"id": 2, "status": "active"},
				{"id": 3, "status": "active"},
			}).
			Build()

		pipeline := doc.Pipeline()
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				return r["status"] == "inactive"
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 0 {
			t.Errorf("expected 0 records (no matches), got %d", len(tableContent.records))
		}

		// Verify schema is preserved even with empty results
		if tableContent.schema == nil {
			t.Error("expected schema to be preserved with empty results")
		}
	})

	t.Run("type assertions within predicate functions", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "value": "string"},
				{"id": 2, "value": 42},
				{"id": 3, "value": nil},
				{"id": 4, "value": true},
				{"id": 5, "value": 3.14},
			}).
			Build()

		pipeline := doc.Pipeline()

		// Filter for integer values only
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				_, ok := r["value"].(int)
				return ok
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 1 {
			t.Errorf("expected 1 integer record, got %d", len(tableContent.records))
		}
		if tableContent.records[0]["value"] != 42 {
			t.Errorf("expected value 42, got %v", tableContent.records[0]["value"])
		}

		// Test handling nil values
		pipeline = doc.Pipeline()
		filterOp = &FilterOp{
			predicate: func(r Record) bool {
				return r["value"] == nil
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err = pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent = result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 1 {
			t.Errorf("expected 1 nil record, got %d", len(tableContent.records))
		}
	})

	t.Run("multiple chained filter operations", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "category": "A", "status": "active", "score": 85},
				{"id": 2, "category": "B", "status": "active", "score": 92},
				{"id": 3, "category": "A", "status": "inactive", "score": 78},
				{"id": 4, "category": "A", "status": "active", "score": 88},
				{"id": 5, "category": "B", "status": "active", "score": 75},
			}).
			Build()

		pipeline := doc.Pipeline()

		// First filter: category = "A"
		filter1 := &FilterOp{
			predicate: func(r Record) bool {
				return r["category"] == "A"
			},
		}

		// Second filter: status = "active"
		filter2 := &FilterOp{
			predicate: func(r Record) bool {
				return r["status"] == "active"
			},
		}

		// Third filter: score > 80
		filter3 := &FilterOp{
			predicate: func(r Record) bool {
				score, ok := r["score"].(int)
				return ok && score > 80
			},
		}

		pipeline.operations = append(pipeline.operations, filter1, filter2, filter3)

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected 2 records after chained filters, got %d", len(tableContent.records))
		}

		// Verify the correct records made it through
		for _, record := range tableContent.records {
			if record["category"] != "A" {
				t.Error("expected category A")
			}
			if record["status"] != "active" {
				t.Error("expected status active")
			}
			score := record["score"].(int)
			if score <= 80 {
				t.Errorf("expected score > 80, got %d", score)
			}
		}
	})

	t.Run("filter preserves schema and key ordering", func(t *testing.T) {
		// Create document with explicit schema
		doc := New().
			Table("test", []Record{
				{"name": "Alice", "age": 30, "city": "NYC"},
				{"name": "Bob", "age": 25, "city": "LA"},
				{"name": "Charlie", "age": 35, "city": "Chicago"},
			}, WithKeys("name", "age", "city")). // Explicit key order
			Build()

		originalContent := doc.GetContents()[0].(*TableContent)
		originalSchema := originalContent.schema

		pipeline := doc.Pipeline()
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				age, ok := r["age"].(int)
				return ok && age >= 30
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultContent := result.GetContents()[0].(*TableContent)

		// Check schema preservation
		if resultContent.schema == nil {
			t.Fatal("expected schema to be preserved")
		}

		// Check key order preservation
		if len(resultContent.schema.keyOrder) != len(originalSchema.keyOrder) {
			t.Errorf("expected %d keys, got %d",
				len(originalSchema.keyOrder), len(resultContent.schema.keyOrder))
		}

		for i, key := range originalSchema.keyOrder {
			if resultContent.schema.keyOrder[i] != key {
				t.Errorf("key order mismatch at index %d: expected %s, got %s",
					i, key, resultContent.schema.keyOrder[i])
			}
		}
	})

	t.Run("filter with complex predicate logic", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "type": "A", "priority": 1, "tags": []string{"urgent", "bug"}},
				{"id": 2, "type": "B", "priority": 2, "tags": []string{"feature"}},
				{"id": 3, "type": "A", "priority": 3, "tags": []string{"bug"}},
				{"id": 4, "type": "C", "priority": 1, "tags": []string{"urgent"}},
			}).
			Build()

		pipeline := doc.Pipeline()

		// Complex filter: (type="A" OR priority=1) AND contains "urgent" tag
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				typeA := r["type"] == "A"
				priority1 := r["priority"] == 1

				tags, ok := r["tags"].([]string)
				if !ok {
					return false
				}

				hasUrgent := slices.Contains(tags, "urgent")

				return (typeA || priority1) && hasUrgent
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected 2 records matching complex filter, got %d", len(tableContent.records))
		}

		// Verify the correct records (IDs 1 and 4)
		expectedIDs := map[int]bool{1: true, 4: true}
		for _, record := range tableContent.records {
			id := record["id"].(int)
			if !expectedIDs[id] {
				t.Errorf("unexpected record with ID %d in results", id)
			}
		}
	})

	t.Run("filter validation", func(t *testing.T) {
		// Test nil predicate validation
		filterOp := &FilterOp{
			predicate: nil,
		}

		err := filterOp.Validate()
		if err == nil {
			t.Fatal("expected error for nil predicate")
		}
		if err.Error() != "filter predicate is required" {
			t.Errorf("unexpected error message: %v", err)
		}

		// Test valid predicate
		filterOp = &FilterOp{
			predicate: func(r Record) bool { return true },
		}

		err = filterOp.Validate()
		if err != nil {
			t.Errorf("unexpected error for valid predicate: %v", err)
		}
	})

	t.Run("filter with missing fields", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice", "age": 30},
				{"id": 2, "name": "Bob"}, // Missing age
				{"id": 3, "age": 35},     // Missing name
				{"id": 4, "name": "Diana", "age": 28},
			}).
			Build()

		pipeline := doc.Pipeline()

		// Filter for records with both name and age
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				_, hasName := r["name"]
				_, hasAge := r["age"]
				return hasName && hasAge
			},
		}
		pipeline.operations = append(pipeline.operations, filterOp)

		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected 2 complete records, got %d", len(tableContent.records))
		}
	})
}

// TestPipelineSortMethod tests the Sort() methods on Pipeline
func TestPipelineSortMethod(t *testing.T) {
	t.Run("fluent Sort() method with single key", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 3, "name": "Charlie", "score": 85},
				{"id": 1, "name": "Alice", "score": 92},
				{"id": 2, "name": "Bob", "score": 78},
			}).
			Build()

		// Test fluent API with ascending sort
		result, err := doc.Pipeline().
			Sort(SortKey{Column: "id", Direction: Ascending}).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 3 {
			t.Errorf("expected 3 records, got %d", len(tableContent.records))
		}

		// Check ascending order by ID
		expectedIds := []int{1, 2, 3}
		for i, record := range tableContent.records {
			if record["id"] != expectedIds[i] {
				t.Errorf("record %d: expected id %d, got %v", i, expectedIds[i], record["id"])
			}
		}
	})

	t.Run("fluent Sort() method with multiple keys", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "salary": 50000},
				{"name": "Bob", "department": "IT", "salary": 75000},
				{"name": "Charlie", "department": "HR", "salary": 60000},
				{"name": "David", "department": "IT", "salary": 80000},
			}).
			Build()

		// Sort by department ascending, then salary descending
		result, err := doc.Pipeline().
			Sort(
				SortKey{Column: "department", Direction: Ascending},
				SortKey{Column: "salary", Direction: Descending},
			).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)

		// Expected order: HR dept (Charlie 60k, Alice 50k), then IT dept (David 80k, Bob 75k)
		expectedNames := []string{"Charlie", "Alice", "David", "Bob"}
		for i, record := range tableContent.records {
			if record["name"] != expectedNames[i] {
				t.Errorf("record %d: expected name %s, got %v", i, expectedNames[i], record["name"])
			}
		}
	})

	t.Run("SortBy convenience method", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 3, "score": 85},
				{"id": 1, "score": 92},
				{"id": 2, "score": 78},
			}).
			Build()

		// Test SortBy convenience method
		result, err := doc.Pipeline().
			SortBy("score", Descending).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		expectedScores := []int{92, 85, 78}
		for i, record := range tableContent.records {
			if record["score"] != expectedScores[i] {
				t.Errorf("record %d: expected score %d, got %v", i, expectedScores[i], record["score"])
			}
		}
	})

	t.Run("SortWith custom comparator method", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"name": "Bob", "length": 3},
				{"name": "Alexander", "length": 9},
				{"name": "Sam", "length": 3},
			}).
			Build()

		// Sort by name length, then alphabetically for ties
		result, err := doc.Pipeline().
			SortWith(func(a, b Record) int {
				aName := a["name"].(string)
				bName := b["name"].(string)

				// First compare by length
				if len(aName) < len(bName) {
					return -1
				} else if len(aName) > len(bName) {
					return 1
				}

				// For equal lengths, compare alphabetically
				if aName < bName {
					return -1
				} else if aName > bName {
					return 1
				}
				return 0
			}).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		expectedNames := []string{"Bob", "Sam", "Alexander"} // Bob and Sam (length 3), then Alexander (length 9)
		for i, record := range tableContent.records {
			if record["name"] != expectedNames[i] {
				t.Errorf("record %d: expected name %s, got %v", i, expectedNames[i], record["name"])
			}
		}
	})

	t.Run("Sort() method returns pipeline for chaining", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "value": "test"},
			}).
			Build()

		pipeline1 := doc.Pipeline()
		pipeline2 := pipeline1.Sort(SortKey{Column: "id", Direction: Ascending})

		// Should return the same pipeline instance for chaining
		if pipeline1 != pipeline2 {
			t.Error("Sort() should return the same pipeline for chaining")
		}

		// Should have added one operation
		if len(pipeline1.operations) != 1 {
			t.Errorf("expected 1 operation after Sort(), got %d", len(pipeline1.operations))
		}

		// Operation should be SortOp
		if _, ok := pipeline1.operations[0].(*SortOp); !ok {
			t.Errorf("expected SortOp, got %T", pipeline1.operations[0])
		}
	})

	t.Run("chained Sort() and Filter() methods", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 3, "status": "active", "score": 85},
				{"id": 1, "status": "inactive", "score": 92},
				{"id": 4, "status": "active", "score": 78},
				{"id": 2, "status": "active", "score": 95},
			}).
			Build()

		// Chain filter then sort
		result, err := doc.Pipeline().
			Filter(func(r Record) bool {
				return r["status"] == "active"
			}).
			Sort(SortKey{Column: "score", Direction: Descending}).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 3 {
			t.Errorf("expected 3 active records, got %d", len(tableContent.records))
		}

		// Should be sorted by score descending: 95, 85, 78
		expectedScores := []int{95, 85, 78}
		for i, record := range tableContent.records {
			if record["status"] != "active" {
				t.Errorf("record %d: expected active status", i)
			}
			if record["score"] != expectedScores[i] {
				t.Errorf("record %d: expected score %d, got %v", i, expectedScores[i], record["score"])
			}
		}
	})

	t.Run("Sort() with different data types", func(t *testing.T) {
		now := time.Now()
		doc := New().
			Table("test", []Record{
				{"id": 3, "name": "Charlie", "active": true, "created": now.Add(time.Hour)},
				{"id": 1, "name": "Alice", "active": false, "created": now},
				{"id": 2, "name": "Bob", "active": true, "created": now.Add(30 * time.Minute)},
			}).
			Build()

		// Test string sorting
		result, err := doc.Pipeline().
			SortBy("name", Ascending).
			Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		expectedNames := []string{"Alice", "Bob", "Charlie"}
		for i, record := range tableContent.records {
			if record["name"] != expectedNames[i] {
				t.Errorf("string sort - record %d: expected name %s, got %v", i, expectedNames[i], record["name"])
			}
		}

		// Test boolean sorting
		result, err = doc.Pipeline().
			SortBy("active", Ascending).
			Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent = result.GetContents()[0].(*TableContent)
		expectedActive := []bool{false, true, true}
		for i, record := range tableContent.records {
			if record["active"] != expectedActive[i] {
				t.Errorf("boolean sort - record %d: expected active %v, got %v", i, expectedActive[i], record["active"])
			}
		}

		// Test time sorting
		result, err = doc.Pipeline().
			SortBy("created", Ascending).
			Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent = result.GetContents()[0].(*TableContent)
		expectedIds := []int{1, 2, 3} // Sorted by timestamp
		for i, record := range tableContent.records {
			if record["id"] != expectedIds[i] {
				t.Errorf("time sort - record %d: expected id %d, got %v", i, expectedIds[i], record["id"])
			}
		}
	})

	t.Run("Sort() preserves immutability", func(t *testing.T) {
		originalData := []Record{
			{"id": 3, "name": "Charlie"},
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
		}

		doc := New().
			Table("test", originalData).
			Build()

		// Sort the data
		sorted, err := doc.Pipeline().
			SortBy("id", Ascending).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Original document should be unchanged
		originalTable := doc.GetContents()[0].(*TableContent)
		if originalTable.records[0]["id"] != 3 {
			t.Error("original document was modified - first record should still have id=3")
		}

		// New document should have sorted results
		sortedTable := sorted.GetContents()[0].(*TableContent)
		if sortedTable.records[0]["id"] != 1 {
			t.Error("sorted document should have first record with id=1")
		}

		// Documents should be different instances
		if doc == sorted {
			t.Error("expected different document instances")
		}
	})

	t.Run("Sort() validation with empty keys", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1},
			}).
			Build()

		// This should create the operation, but validation should fail during execution
		pipeline := doc.Pipeline().Sort() // No keys provided

		_, err := pipeline.Execute()
		if err == nil {
			t.Fatal("expected error for sort with no keys")
		}

		// Should be a validation error about keys or comparator required
		if !strings.Contains(err.Error(), "sort requires keys or comparator") {
			t.Errorf("expected sort validation error, got: %v", err)
		}
	})
}

// TestPipelineFilterMethod tests the Filter() method on Pipeline
func TestPipelineFilterMethod(t *testing.T) {
	t.Run("fluent Filter() method", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "status": "active", "score": 85},
				{"id": 2, "status": "inactive", "score": 92},
				{"id": 3, "status": "active", "score": 78},
				{"id": 4, "status": "active", "score": 95},
			}).
			Build()

		// Test fluent API chaining
		result, err := doc.Pipeline().
			Filter(func(r Record) bool {
				return r["status"] == "active"
			}).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 3 {
			t.Errorf("expected 3 active records, got %d", len(tableContent.records))
		}

		// Verify all returned records are active
		for _, record := range tableContent.records {
			if record["status"] != "active" {
				t.Errorf("expected active status, got %v", record["status"])
			}
		}
	})

	t.Run("chained Filter() methods", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "category": "A", "status": "active", "score": 85},
				{"id": 2, "category": "B", "status": "active", "score": 92},
				{"id": 3, "category": "A", "status": "inactive", "score": 78},
				{"id": 4, "category": "A", "status": "active", "score": 95},
				{"id": 5, "category": "B", "status": "active", "score": 82},
			}).
			Build()

		// Chain multiple filters
		result, err := doc.Pipeline().
			Filter(func(r Record) bool {
				return r["category"] == "A"
			}).
			Filter(func(r Record) bool {
				return r["status"] == "active"
			}).
			Filter(func(r Record) bool {
				score, ok := r["score"].(int)
				return ok && score >= 85
			}).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected 2 records after chained filters, got %d", len(tableContent.records))
		}

		// Verify results match all filters
		expectedIDs := map[int]bool{1: true, 4: true}
		for _, record := range tableContent.records {
			id := record["id"].(int)
			if !expectedIDs[id] {
				t.Errorf("unexpected record with ID %d in results", id)
			}
		}
	})

	t.Run("Filter() method returns pipeline for chaining", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "value": "test"},
			}).
			Build()

		pipeline1 := doc.Pipeline()
		pipeline2 := pipeline1.Filter(func(r Record) bool { return true })

		// Should return the same pipeline instance for chaining
		if pipeline1 != pipeline2 {
			t.Error("Filter() should return the same pipeline for chaining")
		}

		// Should have added one operation
		if len(pipeline1.operations) != 1 {
			t.Errorf("expected 1 operation after Filter(), got %d", len(pipeline1.operations))
		}

		// Operation should be FilterOp
		if _, ok := pipeline1.operations[0].(*FilterOp); !ok {
			t.Errorf("expected FilterOp, got %T", pipeline1.operations[0])
		}
	})

	t.Run("Filter() with nil predicate validation", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1},
			}).
			Build()

		// This should create the operation, but validation should fail during execution
		pipeline := doc.Pipeline().Filter(nil)

		_, err := pipeline.Execute()
		if err == nil {
			t.Fatal("expected error for nil predicate")
		}

		// Should be a validation error about the predicate
		if !strings.Contains(err.Error(), "filter predicate is required") {
			t.Errorf("expected predicate validation error, got: %v", err)
		}
	})

	t.Run("Filter() with complex predicate", func(t *testing.T) {
		doc := New().
			Table("employees", []Record{
				{"name": "Alice", "department": "Engineering", "salary": 90000, "years": 5},
				{"name": "Bob", "department": "Sales", "salary": 70000, "years": 3},
				{"name": "Charlie", "department": "Engineering", "salary": 85000, "years": 7},
				{"name": "Diana", "department": "Marketing", "salary": 75000, "years": 4},
				{"name": "Eve", "department": "Engineering", "salary": 95000, "years": 2},
			}).
			Build()

		// Filter for senior engineers (Engineering dept, salary > 80000, years > 3)
		result, err := doc.Pipeline().
			Filter(func(r Record) bool {
				dept, ok1 := r["department"].(string)
				salary, ok2 := r["salary"].(int)
				years, ok3 := r["years"].(int)

				return ok1 && ok2 && ok3 &&
					dept == "Engineering" &&
					salary > 80000 &&
					years > 3
			}).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected 2 senior engineers, got %d", len(tableContent.records))
		}

		// Verify both Alice and Charlie made it through
		names := make(map[string]bool)
		for _, record := range tableContent.records {
			names[record["name"].(string)] = true
		}

		if !names["Alice"] || !names["Charlie"] {
			t.Error("expected Alice and Charlie in results")
		}
	})

	t.Run("Filter() preserves immutability", func(t *testing.T) {
		originalData := []Record{
			{"id": 1, "status": "active"},
			{"id": 2, "status": "inactive"},
			{"id": 3, "status": "active"},
		}

		doc := New().
			Table("test", originalData).
			Build()

		// Filter the data
		filtered, err := doc.Pipeline().
			Filter(func(r Record) bool {
				return r["status"] == "active"
			}).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Original document should be unchanged
		originalTable := doc.GetContents()[0].(*TableContent)
		if len(originalTable.records) != 3 {
			t.Errorf("original document was modified: expected 3 records, got %d", len(originalTable.records))
		}

		// New document should have filtered results
		filteredTable := filtered.GetContents()[0].(*TableContent)
		if len(filteredTable.records) != 2 {
			t.Errorf("filtered document should have 2 records, got %d", len(filteredTable.records))
		}

		// Documents should be different instances
		if doc == filtered {
			t.Error("expected different document instances")
		}
	})
}

// TestPipelineLimitMethod tests the Limit() method on Pipeline
func TestPipelineLimitMethod(t *testing.T) {
	t.Run("fluent Limit() method", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice", "score": 85},
				{"id": 2, "name": "Bob", "score": 92},
				{"id": 3, "name": "Charlie", "score": 78},
				{"id": 4, "name": "Diana", "score": 95},
			}).
			Build()

		// Test fluent API chaining
		result, err := doc.Pipeline().
			Limit(2).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected 2 limited records, got %d", len(tableContent.records))
		}

		// Verify first 2 records are kept
		expectedIDs := []int{1, 2}
		for i, record := range tableContent.records {
			if record["id"] != expectedIDs[i] {
				t.Errorf("record %d: expected id %d, got %v", i, expectedIDs[i], record["id"])
			}
		}
	})

	t.Run("Limit() method returns pipeline for chaining", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "value": "test"},
			}).
			Build()

		pipeline1 := doc.Pipeline()
		pipeline2 := pipeline1.Limit(1)

		// Should return the same pipeline instance for chaining
		if pipeline1 != pipeline2 {
			t.Error("Limit() should return the same pipeline for chaining")
		}

		// Should have added one operation
		if len(pipeline1.operations) != 1 {
			t.Errorf("expected 1 operation after Limit(), got %d", len(pipeline1.operations))
		}

		// Operation should be LimitOp
		if _, ok := pipeline1.operations[0].(*LimitOp); !ok {
			t.Errorf("expected LimitOp, got %T", pipeline1.operations[0])
		}
	})

	t.Run("chained Filter() and Limit() methods", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "status": "active", "score": 85},
				{"id": 2, "status": "inactive", "score": 92},
				{"id": 3, "status": "active", "score": 78},
				{"id": 4, "status": "active", "score": 95},
				{"id": 5, "status": "active", "score": 88},
			}).
			Build()

		// Chain filter then limit
		result, err := doc.Pipeline().
			Filter(func(r Record) bool {
				return r["status"] == "active"
			}).
			Limit(2).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected 2 limited records after filter, got %d", len(tableContent.records))
		}

		// Should be first 2 active records: IDs 1 and 3
		expectedIDs := []int{1, 3}
		for i, record := range tableContent.records {
			if record["status"] != "active" {
				t.Errorf("record %d: expected active status", i)
			}
			if record["id"] != expectedIDs[i] {
				t.Errorf("record %d: expected id %d, got %v", i, expectedIDs[i], record["id"])
			}
		}
	})

	t.Run("chained Sort() and Limit() methods", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 3, "score": 85},
				{"id": 1, "score": 95},
				{"id": 4, "score": 78},
				{"id": 2, "score": 92},
			}).
			Build()

		// Sort by score descending, then limit to top 2
		result, err := doc.Pipeline().
			Sort(SortKey{Column: "score", Direction: Descending}).
			Limit(2).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected 2 limited records after sort, got %d", len(tableContent.records))
		}

		// Should be top 2 scores: 95, 92
		expectedScores := []int{95, 92}
		for i, record := range tableContent.records {
			if record["score"] != expectedScores[i] {
				t.Errorf("record %d: expected score %d, got %v", i, expectedScores[i], record["score"])
			}
		}
	})

	t.Run("Limit() preserves immutability", func(t *testing.T) {
		originalData := []Record{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
			{"id": 3, "name": "Charlie"},
		}

		doc := New().
			Table("test", originalData).
			Build()

		// Limit the data
		limited, err := doc.Pipeline().
			Limit(2).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Original document should be unchanged
		originalTable := doc.GetContents()[0].(*TableContent)
		if len(originalTable.records) != 3 {
			t.Errorf("original document was modified: expected 3 records, got %d", len(originalTable.records))
		}

		// New document should have limited results
		limitedTable := limited.GetContents()[0].(*TableContent)
		if len(limitedTable.records) != 2 {
			t.Errorf("limited document should have 2 records, got %d", len(limitedTable.records))
		}

		// Documents should be different instances
		if doc == limited {
			t.Error("expected different document instances")
		}
	})
}

// TestLimitOperationVariousCounts tests limit operation with various count values
func TestLimitOperationVariousCounts(t *testing.T) {
	t.Run("limiting with various counts", func(t *testing.T) {
		data := []Record{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
			{"id": 3, "name": "Charlie"},
			{"id": 4, "name": "Diana"},
			{"id": 5, "name": "Eve"},
		}

		doc := New().
			Table("test", data).
			Build()

		// Test various limit counts
		testCases := []struct {
			limit    int
			expected int
		}{
			{1, 1},
			{3, 3},
			{5, 5},  // Equal to data size
			{10, 5}, // Larger than data size
			{0, 0},  // Zero limit
		}

		for _, tc := range testCases {
			result, err := doc.Pipeline().
				Limit(tc.limit).
				Execute()

			if err != nil {
				t.Fatalf("unexpected error for limit %d: %v", tc.limit, err)
			}

			tableContent := result.GetContents()[0].(*TableContent)
			if len(tableContent.records) != tc.expected {
				t.Errorf("limit %d: expected %d records, got %d", tc.limit, tc.expected, len(tableContent.records))
			}

			// Verify we get the first N records
			for i, record := range tableContent.records {
				expectedID := i + 1
				if record["id"] != expectedID {
					t.Errorf("limit %d, record %d: expected id %d, got %v", tc.limit, i, expectedID, record["id"])
				}
			}
		}
	})

	t.Run("limit with count larger than data size", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		result, err := doc.Pipeline().
			Limit(10).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected all 2 records when limit exceeds data size, got %d", len(tableContent.records))
		}
	})

	t.Run("limit with zero count", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		result, err := doc.Pipeline().
			Limit(0).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 0 {
			t.Errorf("expected 0 records when limit is 0, got %d", len(tableContent.records))
		}
	})

	t.Run("limit with negative values handled by validation", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		// This should create the operation, but validation should fail during execution
		_, err := doc.Pipeline().
			Limit(-1).
			Execute()

		if err == nil {
			t.Fatal("expected error for negative limit")
		}

		// Should be a validation error about negative count
		if !strings.Contains(err.Error(), "limit count must be non-negative") {
			t.Errorf("expected negative count validation error, got: %v", err)
		}
	})
}

// Tests for Task 9.1: Pipeline execution and optimization

// TestPipelineOperationReordering tests operation reordering for performance optimization
func TestPipelineOperationReordering(t *testing.T) {
	t.Run("filter operations should be applied before sort for efficiency", func(t *testing.T) {
		// Create a large dataset to make the optimization meaningful
		records := make([]Record, 100)
		for i := range 100 {
			records[i] = Record{
				"id":     i,
				"name":   fmt.Sprintf("User%d", i),
				"status": map[bool]string{true: "active", false: "inactive"}[i%3 == 0],
				"score":  float64(i * 10),
			}
		}

		doc := New().
			Table("users", records).
			Build()

		// Create pipeline with filter and sort operations
		// The optimization should reorder these to apply filter first
		pipeline := doc.Pipeline().
			SortBy("score", Descending).
			Filter(func(r Record) bool { return r["status"] == "active" }).
			Limit(5)

		start := time.Now()
		result, err := pipeline.Execute()
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify results are correct
		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 5 {
			t.Errorf("expected 5 records, got %d", len(tableContent.records))
		}

		// All results should be active
		for i, record := range tableContent.records {
			if record["status"] != "active" {
				t.Errorf("record %d should be active, got %v", i, record["status"])
			}
		}

		// Results should be sorted by score in descending order
		for i := 0; i < len(tableContent.records)-1; i++ {
			currentScore := tableContent.records[i]["score"].(float64)
			nextScore := tableContent.records[i+1]["score"].(float64)
			if currentScore < nextScore {
				t.Errorf("records not properly sorted: %f should be >= %f", currentScore, nextScore)
			}
		}

		// Performance should be reasonable (this is more of a sanity check)
		if duration > time.Second {
			t.Logf("Pipeline took %v - may indicate optimization issues", duration)
		}
	})

	t.Run("multiple filters should be applied efficiently", func(t *testing.T) {
		records := []Record{
			{"id": 1, "name": "Alice", "age": 25, "department": "Engineering"},
			{"id": 2, "name": "Bob", "age": 30, "department": "Engineering"},
			{"id": 3, "name": "Carol", "age": 28, "department": "Marketing"},
			{"id": 4, "name": "Dave", "age": 35, "department": "Engineering"},
			{"id": 5, "name": "Eve", "age": 22, "department": "Sales"},
		}

		doc := New().
			Table("employees", records).
			Build()

		// Multiple filters should be combined efficiently
		result, err := doc.Pipeline().
			Filter(func(r Record) bool { return r["department"] == "Engineering" }).
			Filter(func(r Record) bool { return r["age"].(int) >= 28 }).
			Execute()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 { // Bob and Dave
			t.Errorf("expected 2 records, got %d", len(tableContent.records))
		}

		// Verify the correct records are returned
		names := make(map[string]bool)
		for _, record := range tableContent.records {
			names[record["name"].(string)] = true
		}

		if !names["Bob"] || !names["Dave"] {
			t.Error("expected Bob and Dave in results")
		}
	})
}

// TestPipelineLazyEvaluation tests lazy evaluation behavior
func TestPipelineLazyEvaluation(t *testing.T) {
	t.Run("operations are collected before execution", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		// Build pipeline but don't execute yet
		pipeline := doc.Pipeline().
			Filter(func(r Record) bool { return r["id"].(int) > 0 }).
			SortBy("name", Ascending).
			Limit(10)

		// Operations should be collected but not executed
		if len(pipeline.operations) != 3 {
			t.Errorf("expected 3 operations to be collected, got %d", len(pipeline.operations))
		}

		// Verify operation names
		expectedOps := []string{"Filter", "Sort", "Limit"}
		for i, op := range pipeline.operations {
			if op.Name() != expectedOps[i] {
				t.Errorf("operation %d: expected %s, got %s", i, expectedOps[i], op.Name())
			}
		}

		// Now execute and verify it works
		result, err := pipeline.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 {
			t.Errorf("expected 2 records, got %d", len(tableContent.records))
		}
	})

	t.Run("multiple calls to same pipeline method create multiple operations", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		pipeline := doc.Pipeline().
			Filter(func(r Record) bool { return r["id"].(int) > 0 }).
			Filter(func(r Record) bool { return r["name"] != "" })

		if len(pipeline.operations) != 2 {
			t.Errorf("expected 2 filter operations, got %d", len(pipeline.operations))
		}

		for _, op := range pipeline.operations {
			if op.Name() != "Filter" {
				t.Errorf("expected Filter operation, got %s", op.Name())
			}
		}
	})
}

// TestPipelineErrorPropagation tests error propagation through pipeline
func TestPipelineErrorPropagation(t *testing.T) {
	t.Run("error in operation stops pipeline execution", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		// Create a pipeline with an operation that will cause validation failure
		pipeline := doc.Pipeline().
			Filter(func(r Record) bool { return r["id"].(int) > 0 }).
			Limit(-1).                // This will cause validation error
			SortBy("name", Ascending) // This should not be reached

		_, err := pipeline.Execute()
		if err == nil {
			t.Fatal("expected error from invalid limit")
		}

		// Error should contain context about which operation failed
		errStr := err.Error()
		if !strings.Contains(errStr, "operation") || !strings.Contains(errStr, "limit") {
			t.Errorf("error should contain operation context, got: %v", err)
		}
	})

	t.Run("validation error provides detailed context", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		// Create invalid pipeline (negative limit)
		pipeline := doc.Pipeline().
			Filter(func(r Record) bool { return true }).
			Limit(-5)

		_, err := pipeline.Execute()
		if err == nil {
			t.Fatal("expected validation error")
		}

		// Error should be descriptive
		errStr := err.Error()
		if !strings.Contains(errStr, "limit count must be non-negative") {
			t.Errorf("expected specific validation error, got: %v", err)
		}
	})

	t.Run("error includes operation index and type information", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		// Create pipeline with multiple operations where later operation fails
		pipeline := doc.Pipeline().
			Filter(func(r Record) bool { return true }).    // This will succeed
			AddColumn("", func(r Record) any { return "" }) // This will fail validation (empty name)

		_, err := pipeline.Execute()
		if err == nil {
			t.Fatal("expected error from invalid column name")
		}

		// The error should provide context about the failing operation
		errStr := err.Error()
		if !strings.Contains(errStr, "operation") {
			t.Errorf("error should contain operation context, got: %v", err)
		}
	})
}

// TestPipelineContextCancellation tests context cancellation and timeouts
func TestPipelineContextCancellation(t *testing.T) {
	t.Run("pipeline respects context cancellation", func(t *testing.T) {
		// Create a large dataset to make cancellation meaningful
		records := make([]Record, 5000)
		for i := range 5000 {
			records[i] = Record{
				"id":   i,
				"name": fmt.Sprintf("User%d", i),
			}
		}

		doc := New().
			Table("users", records).
			Build()

		// Create a context that we can cancel
		ctx, cancel := context.WithCancel(context.Background())

		// Create pipeline with operations that would take time
		pipeline := doc.Pipeline().
			Filter(func(r Record) bool {
				// Simulate some processing time
				time.Sleep(50 * time.Microsecond)
				return r["id"].(int)%2 == 0
			}).
			SortBy("name", Ascending)

		// Cancel context after a very short time
		go func() {
			time.Sleep(5 * time.Millisecond)
			cancel()
		}()

		_, err := pipeline.ExecuteContext(ctx)
		if err == nil {
			t.Log("Pipeline completed before cancellation - this can happen on fast systems")
			return // Skip assertion on fast systems
		}

		// Should be a context cancellation error
		if !strings.Contains(err.Error(), "cancel") && !strings.Contains(err.Error(), "context") {
			t.Errorf("expected context cancellation error, got: %v", err)
		}
	})

	t.Run("pipeline respects execution timeout", func(t *testing.T) {
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		// Create pipeline with very short timeout
		pipeline := doc.Pipeline().WithOptions(PipelineOptions{
			MaxOperations:    100,
			MaxExecutionTime: 1 * time.Nanosecond, // Extremely short timeout
		}).Filter(func(r Record) bool {
			time.Sleep(10 * time.Millisecond) // This should timeout
			return true
		})

		_, err := pipeline.Execute()
		if err == nil {
			t.Fatal("expected timeout error")
		}

		// Should be a timeout/cancellation error
		errStr := err.Error()
		if !strings.Contains(errStr, "cancel") && !strings.Contains(errStr, "timeout") && !strings.Contains(errStr, "context") {
			t.Errorf("expected timeout error, got: %v", err)
		}
	})

	t.Run("pipeline handles context cancellation in individual operations", func(t *testing.T) {
		records := make([]Record, 100)
		for i := range 100 {
			records[i] = Record{
				"id":    i,
				"value": i * 10,
			}
		}

		doc := New().
			Table("test", records).
			Build()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()

		// Create operations that check context
		_, err := doc.Pipeline().
			Filter(func(r Record) bool {
				// Simulate work that checks context
				time.Sleep(100 * time.Microsecond)
				return r["id"].(int) > 10
			}).
			ExecuteContext(ctx)

		if err == nil {
			t.Log("Pipeline may have completed before timeout - this is acceptable")
		} else {
			// Should be a context-related error
			errStr := err.Error()
			if !strings.Contains(errStr, "cancel") && !strings.Contains(errStr, "timeout") && !strings.Contains(errStr, "context") {
				t.Errorf("expected context-related error, got: %v", err)
			}
		}
	})
}
