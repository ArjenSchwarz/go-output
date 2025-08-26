package output

import (
	"context"
	"strings"
	"testing"
)

// TestOperationInterface tests the Operation interface compliance
func TestOperationInterface(t *testing.T) {
	t.Run("FilterOp implements Operation interface", func(t *testing.T) {
		filterOp := &FilterOp{
			predicate: func(r Record) bool { return true },
		}

		var _ Operation = filterOp // Compile-time check

		if filterOp.Name() != "Filter" {
			t.Errorf("expected Name() to return 'Filter', got '%s'", filterOp.Name())
		}

		// Test basic Apply functionality
		doc := New().
			Table("test", []Record{
				{"id": 1, "status": "active"},
				{"id": 2, "status": "inactive"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := filterOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		if result == nil {
			t.Fatal("Apply() returned nil result")
		}
	})

	t.Run("SortOp implements Operation interface", func(t *testing.T) {
		sortOp := &SortOp{
			keys: []SortKey{
				{Column: "id", Direction: Ascending},
			},
		}

		var _ Operation = sortOp // Compile-time check

		if sortOp.Name() != "Sort" {
			t.Errorf("expected Name() to return 'Sort', got '%s'", sortOp.Name())
		}

		// Test basic Apply functionality
		doc := New().
			Table("test", []Record{
				{"id": 2, "name": "Bob"},
				{"id": 1, "name": "Alice"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := sortOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		if result == nil {
			t.Fatal("Apply() returned nil result")
		}
	})

	t.Run("LimitOp implements Operation interface", func(t *testing.T) {
		limitOp := &LimitOp{
			count: 1,
		}

		var _ Operation = limitOp // Compile-time check

		if limitOp.Name() != "Limit" {
			t.Errorf("expected Name() to return 'Limit', got '%s'", limitOp.Name())
		}

		// Test basic Apply functionality
		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := limitOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		if result == nil {
			t.Fatal("Apply() returned nil result")
		}
	})
}

// TestFilterOpValidation tests FilterOp validation logic
func TestLimitOpValidation(t *testing.T) {
	t.Run("validates positive count", func(t *testing.T) {
		limitOp := &LimitOp{
			count: 5,
		}

		err := limitOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("validates zero count", func(t *testing.T) {
		limitOp := &LimitOp{
			count: 0,
		}

		err := limitOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error for zero count: %v", err)
		}
	})

	t.Run("fails validation for negative count", func(t *testing.T) {
		limitOp := &LimitOp{
			count: -1,
		}

		err := limitOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for negative count")
		}

		if !strings.Contains(err.Error(), "limit count must be non-negative") {
			t.Errorf("expected error about limit count being non-negative, got '%s'", err.Error())
		}
	})
}

// TestLimitOpApply tests LimitOp Apply method functionality
func TestLimitOpApply(t *testing.T) {
	t.Run("limits records to specified count", func(t *testing.T) {
		limitOp := &LimitOp{
			count: 2,
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
				{"id": 3, "name": "Charlie"},
				{"id": 4, "name": "David"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := limitOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 limited records, got %d", len(resultTable.records))
		}

		// Check that the first 2 records were kept
		expectedIds := []int{1, 2}
		for i, record := range resultTable.records {
			if record["id"] != expectedIds[i] {
				t.Errorf("record %d: expected id %d, got %v", i, expectedIds[i], record["id"])
			}
		}
	})

	t.Run("handles count larger than data size", func(t *testing.T) {
		limitOp := &LimitOp{
			count: 10,
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := limitOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected all 2 records when limit exceeds data size, got %d", len(resultTable.records))
		}
	})

	t.Run("handles zero count", func(t *testing.T) {
		limitOp := &LimitOp{
			count: 0,
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := limitOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 0 {
			t.Errorf("expected 0 records when limit is 0, got %d", len(resultTable.records))
		}
	})

	t.Run("fails when applied to non-table content", func(t *testing.T) {
		limitOp := &LimitOp{
			count: 1,
		}

		doc := New().
			Text("Some text content").
			Build()

		textContent := doc.GetContents()[0]
		_, err := limitOp.Apply(context.Background(), textContent)
		if err == nil {
			t.Fatal("expected error when applying limit to non-table content")
		}

		if !strings.Contains(err.Error(), "limit operation requires table content") {
			t.Errorf("expected error about limit operation requiring table content, got '%s'", err.Error())
		}
	})
}

// TestOperationCanOptimize tests CanOptimize method for operations
func TestOperationCanOptimize(t *testing.T) {
	t.Run("FilterOp can optimize with other filters", func(t *testing.T) {
		filter1 := &FilterOp{predicate: func(r Record) bool { return true }}
		filter2 := &FilterOp{predicate: func(r Record) bool { return true }}

		// Filters can be combined
		if !filter1.CanOptimize(filter2) {
			t.Error("expected FilterOp to be optimizable with another FilterOp")
		}
	})

	t.Run("SortOp cannot optimize with other operations", func(t *testing.T) {
		sortOp := &SortOp{keys: []SortKey{{Column: "id", Direction: Ascending}}}
		filterOp := &FilterOp{predicate: func(r Record) bool { return true }}

		// Sort typically cannot be optimized with other operations
		if sortOp.CanOptimize(filterOp) {
			t.Error("expected SortOp to not be optimizable with FilterOp")
		}
	})

	t.Run("LimitOp cannot optimize with other operations", func(t *testing.T) {
		limitOp := &LimitOp{count: 10}
		filterOp := &FilterOp{predicate: func(r Record) bool { return true }}

		// Limit typically cannot be optimized with other operations
		if limitOp.CanOptimize(filterOp) {
			t.Error("expected LimitOp to not be optimizable with FilterOp")
		}
	})
}

// TestOperationChaining tests chaining multiple operations
func TestOperationChaining(t *testing.T) {
	t.Run("Filter then Sort then Limit", func(t *testing.T) {
		operations := []Operation{
			&FilterOp{
				predicate: func(r Record) bool {
					return r["status"] == "active"
				},
			},
			&SortOp{
				keys: []SortKey{
					{Column: "score", Direction: Descending},
				},
			},
			&LimitOp{
				count: 2,
			},
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "status": "active", "score": 85},
				{"id": 2, "status": "inactive", "score": 92},
				{"id": 3, "status": "active", "score": 78},
				{"id": 4, "status": "active", "score": 95},
			}).
			Build()

		content := doc.GetContents()[0]

		// Apply operations sequentially
		current := content
		for i, op := range operations {
			// Clone before each operation
			if transformable, ok := current.(TransformableContent); ok {
				current = transformable.Clone()
			}

			result, err := op.Apply(context.Background(), current)
			if err != nil {
				t.Fatalf("operation %d (%s) failed: %v", i, op.Name(), err)
			}
			current = result
		}

		// Check final result
		finalTable := current.(*TableContent)
		if len(finalTable.records) != 2 {
			t.Errorf("expected 2 final records, got %d", len(finalTable.records))
		}

		// Should be top 2 active records by score (95, 85)
		expectedScores := []int{95, 85}
		for i, record := range finalTable.records {
			if record["score"] != expectedScores[i] {
				t.Errorf("record %d: expected score %d, got %v", i, expectedScores[i], record["score"])
			}
			if record["status"] != "active" {
				t.Errorf("record %d: expected status 'active', got %v", i, record["status"])
			}
		}
	})
}

// TestAggregateOpValidation tests GroupByOp and AggregateOp validation logic
