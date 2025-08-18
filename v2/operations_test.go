package output

import (
	"context"
	"testing"
	"time"
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
func TestFilterOpValidation(t *testing.T) {
	t.Run("validates predicate is not nil", func(t *testing.T) {
		filterOp := &FilterOp{
			predicate: nil,
		}

		err := filterOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for nil predicate")
		}

		expectedMsg := "filter predicate is required"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("validates when predicate is provided", func(t *testing.T) {
		filterOp := &FilterOp{
			predicate: func(r Record) bool { return true },
		}

		err := filterOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})
}

// TestFilterOpApply tests FilterOp Apply method functionality
func TestFilterOpApply(t *testing.T) {
	t.Run("filters records based on predicate", func(t *testing.T) {
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				return r["status"] == "active"
			},
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "status": "active"},
				{"id": 2, "status": "inactive"},
				{"id": 3, "status": "active"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := filterOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 filtered records, got %d", len(resultTable.records))
		}

		// Check that the correct records were kept
		for _, record := range resultTable.records {
			if record["status"] != "active" {
				t.Errorf("filtered result contains inactive record: %v", record)
			}
		}
	})

	t.Run("returns empty result when no records match", func(t *testing.T) {
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				return r["status"] == "nonexistent"
			},
		}

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

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 0 {
			t.Errorf("expected 0 filtered records, got %d", len(resultTable.records))
		}
	})

	t.Run("handles type assertions in predicate", func(t *testing.T) {
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				if val, ok := r["score"].(int); ok {
					return val > 80
				}
				return false
			},
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "score": 90},
				{"id": 2, "score": "invalid"},
				{"id": 3, "score": 75},
				{"id": 4, "score": 85},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := filterOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 filtered records, got %d", len(resultTable.records))
		}

		// Check that the correct records were kept
		for _, record := range resultTable.records {
			if score, ok := record["score"].(int); !ok || score <= 80 {
				t.Errorf("filtered result contains invalid record: %v", record)
			}
		}
	})

	t.Run("fails when applied to non-table content", func(t *testing.T) {
		filterOp := &FilterOp{
			predicate: func(r Record) bool { return true },
		}

		doc := New().
			Text("Some text content").
			Build()

		textContent := doc.GetContents()[0]
		_, err := filterOp.Apply(context.Background(), textContent)
		if err == nil {
			t.Fatal("expected error when applying filter to non-table content")
		}

		expectedMsg := "filter requires table content"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("preserves original content immutability", func(t *testing.T) {
		filterOp := &FilterOp{
			predicate: func(r Record) bool {
				return r["id"].(int) > 1
			},
		}

		originalRecords := []Record{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
			{"id": 3, "name": "Charlie"},
		}

		doc := New().
			Table("test", originalRecords).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		originalCount := len(tableContent.records)

		_, err := filterOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		// Original should be unchanged
		if len(tableContent.records) != originalCount {
			t.Errorf("original content was modified: had %d records, now has %d",
				originalCount, len(tableContent.records))
		}
	})
}

// TestSortOpValidation tests SortOp validation logic
func TestSortOpValidation(t *testing.T) {
	t.Run("validates when keys are provided", func(t *testing.T) {
		sortOp := &SortOp{
			keys: []SortKey{
				{Column: "id", Direction: Ascending},
			},
		}

		err := sortOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("validates when comparator is provided", func(t *testing.T) {
		sortOp := &SortOp{
			comparator: func(a, b Record) int { return 0 },
		}

		err := sortOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("fails validation when neither keys nor comparator provided", func(t *testing.T) {
		sortOp := &SortOp{}

		err := sortOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for empty sort operation")
		}

		expectedMsg := "sort requires keys or comparator"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("validates empty keys slice", func(t *testing.T) {
		sortOp := &SortOp{
			keys: []SortKey{},
		}

		err := sortOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for empty keys slice")
		}
	})
}

// TestSortOpApply tests SortOp Apply method functionality
func TestSortOpApply(t *testing.T) {
	t.Run("sorts by single column ascending", func(t *testing.T) {
		sortOp := &SortOp{
			keys: []SortKey{
				{Column: "id", Direction: Ascending},
			},
		}

		doc := New().
			Table("test", []Record{
				{"id": 3, "name": "Charlie"},
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := sortOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 3 {
			t.Errorf("expected 3 records, got %d", len(resultTable.records))
		}

		// Check order
		expectedIds := []int{1, 2, 3}
		for i, record := range resultTable.records {
			if record["id"] != expectedIds[i] {
				t.Errorf("record %d: expected id %d, got %v", i, expectedIds[i], record["id"])
			}
		}
	})

	t.Run("sorts by single column descending", func(t *testing.T) {
		sortOp := &SortOp{
			keys: []SortKey{
				{Column: "score", Direction: Descending},
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "score": 85},
				{"name": "Bob", "score": 92},
				{"name": "Charlie", "score": 78},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := sortOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		expectedScores := []int{92, 85, 78}
		for i, record := range resultTable.records {
			if record["score"] != expectedScores[i] {
				t.Errorf("record %d: expected score %d, got %v", i, expectedScores[i], record["score"])
			}
		}
	})

	t.Run("sorts by multiple columns", func(t *testing.T) {
		sortOp := &SortOp{
			keys: []SortKey{
				{Column: "department", Direction: Ascending},
				{Column: "salary", Direction: Descending},
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "salary": 50000},
				{"name": "Bob", "department": "IT", "salary": 75000},
				{"name": "Charlie", "department": "HR", "salary": 60000},
				{"name": "David", "department": "IT", "salary": 80000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := sortOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)

		// Expected order: HR dept (Charlie 60k, Alice 50k), then IT dept (David 80k, Bob 75k)
		expectedOrder := []string{"Charlie", "Alice", "David", "Bob"}
		for i, record := range resultTable.records {
			if record["name"] != expectedOrder[i] {
				t.Errorf("record %d: expected name %s, got %v", i, expectedOrder[i], record["name"])
			}
		}
	})

	t.Run("handles different data types", func(t *testing.T) {
		sortOp := &SortOp{
			keys: []SortKey{
				{Column: "timestamp", Direction: Ascending},
			},
		}

		now := time.Now()
		doc := New().
			Table("test", []Record{
				{"id": 1, "timestamp": now.Add(time.Hour)},
				{"id": 2, "timestamp": now},
				{"id": 3, "timestamp": now.Add(30 * time.Minute)},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := sortOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		expectedIds := []int{2, 3, 1} // Sorted by timestamp
		for i, record := range resultTable.records {
			if record["id"] != expectedIds[i] {
				t.Errorf("record %d: expected id %d, got %v", i, expectedIds[i], record["id"])
			}
		}
	})

	t.Run("uses custom comparator when provided", func(t *testing.T) {
		// Custom comparator that sorts by string length
		sortOp := &SortOp{
			comparator: func(a, b Record) int {
				aName := a["name"].(string)
				bName := b["name"].(string)
				if len(aName) < len(bName) {
					return -1
				} else if len(aName) > len(bName) {
					return 1
				}
				return 0
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Bob"},
				{"name": "Alexander"},
				{"name": "Sam"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := sortOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		expectedNames := []string{"Bob", "Sam", "Alexander"} // Sorted by length
		for i, record := range resultTable.records {
			if record["name"] != expectedNames[i] {
				t.Errorf("record %d: expected name %s, got %v", i, expectedNames[i], record["name"])
			}
		}
	})

	t.Run("maintains stable ordering for equal values", func(t *testing.T) {
		sortOp := &SortOp{
			keys: []SortKey{
				{Column: "score", Direction: Ascending},
			},
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "score": 85},
				{"id": 2, "score": 85},
				{"id": 3, "score": 85},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := sortOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		// Order should be preserved for equal values
		expectedIds := []int{1, 2, 3}
		for i, record := range resultTable.records {
			if record["id"] != expectedIds[i] {
				t.Errorf("record %d: expected id %d, got %v", i, expectedIds[i], record["id"])
			}
		}
	})

	t.Run("fails when applied to non-table content", func(t *testing.T) {
		sortOp := &SortOp{
			keys: []SortKey{
				{Column: "id", Direction: Ascending},
			},
		}

		doc := New().
			Text("Some text content").
			Build()

		textContent := doc.GetContents()[0]
		_, err := sortOp.Apply(context.Background(), textContent)
		if err == nil {
			t.Fatal("expected error when applying sort to non-table content")
		}

		expectedMsg := "sort requires table content"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})
}

// TestLimitOpValidation tests LimitOp validation logic
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

		expectedMsg := "limit count must be non-negative"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
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

		expectedMsg := "limit requires table content"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
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
