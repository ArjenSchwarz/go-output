package output

import (
	"context"
	"fmt"
	"slices"
	"strings"
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

		if !strings.Contains(err.Error(), "filter predicate function is required") {
			t.Errorf("expected error about filter predicate, got '%s'", err.Error())
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

		if !strings.Contains(err.Error(), "filter operation requires table content") {
			t.Errorf("expected error about filter operation requiring table content, got '%s'", err.Error())
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

		if !strings.Contains(err.Error(), "sort operation requires either sort keys or a custom comparator function") {
			t.Errorf("expected error about sort operation requiring keys or comparator, got '%s'", err.Error())
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

		if !strings.Contains(err.Error(), "sort operation requires table content") {
			t.Errorf("expected error about sort operation requiring table content, got '%s'", err.Error())
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
func TestAggregateOpValidation(t *testing.T) {
	t.Run("validates GroupBy with single column", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count": CountAggregate(),
			},
		}

		err := groupByOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("validates GroupBy with multiple columns", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department", "level"},
			aggregates: map[string]AggregateFunc{
				"total_salary": SumAggregate("salary"),
				"count":        CountAggregate(),
			},
		}

		err := groupByOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("fails validation when no groupBy columns provided", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{},
			aggregates: map[string]AggregateFunc{
				"count": CountAggregate(),
			},
		}

		err := groupByOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for empty groupBy columns")
		}

		if !strings.Contains(err.Error(), "groupBy operation requires at least one grouping column") {
			t.Errorf("expected error about groupBy requiring at least one column, got '%s'", err.Error())
		}
	})

	t.Run("fails validation when no aggregates provided", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy:    []string{"department"},
			aggregates: map[string]AggregateFunc{},
		}

		err := groupByOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for empty aggregates")
		}

		if !strings.Contains(err.Error(), "groupBy operation requires at least one aggregate function") {
			t.Errorf("expected error about groupBy requiring at least one aggregate function, got '%s'", err.Error())
		}
	})
}

// TestAggregateOpApply tests GroupByOp Apply method functionality
func TestAggregateOpApply(t *testing.T) {
	t.Run("groups by single column with count aggregate", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count": CountAggregate(),
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
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 groups, got %d", len(resultTable.records))
		}

		// Verify groups and counts
		groupCounts := make(map[string]int)
		for _, record := range resultTable.records {
			department := record["department"].(string)
			count := record["count"].(int)
			groupCounts[department] = count
		}

		if groupCounts["HR"] != 2 {
			t.Errorf("expected HR count 2, got %d", groupCounts["HR"])
		}
		if groupCounts["IT"] != 2 {
			t.Errorf("expected IT count 2, got %d", groupCounts["IT"])
		}
	})

	t.Run("groups by single column with sum aggregate", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"total_salary": SumAggregate("salary"),
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
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 groups, got %d", len(resultTable.records))
		}

		// Verify groups and sums
		groupSums := make(map[string]float64)
		for _, record := range resultTable.records {
			department := record["department"].(string)
			totalSalary := record["total_salary"].(float64)
			groupSums[department] = totalSalary
		}

		if groupSums["HR"] != 110000 {
			t.Errorf("expected HR total salary 110000, got %f", groupSums["HR"])
		}
		if groupSums["IT"] != 155000 {
			t.Errorf("expected IT total salary 155000, got %f", groupSums["IT"])
		}
	})

	t.Run("groups by multiple columns", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department", "level"},
			aggregates: map[string]AggregateFunc{
				"avg_salary": AverageAggregate("salary"),
				"count":      CountAggregate(),
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "level": "Junior", "salary": 50000},
				{"name": "Bob", "department": "IT", "level": "Senior", "salary": 80000},
				{"name": "Charlie", "department": "HR", "level": "Senior", "salary": 70000},
				{"name": "David", "department": "IT", "level": "Junior", "salary": 60000},
				{"name": "Eve", "department": "HR", "level": "Junior", "salary": 45000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 4 {
			t.Errorf("expected 4 groups, got %d", len(resultTable.records))
		}

		// Verify we have the expected department/level combinations
		groups := make(map[string]map[string]bool)
		for _, record := range resultTable.records {
			department := record["department"].(string)
			level := record["level"].(string)
			count := record["count"].(int)

			if groups[department] == nil {
				groups[department] = make(map[string]bool)
			}
			groups[department][level] = true

			// HR Junior should have 2 people (avg 47500)
			if department == "HR" && level == "Junior" && count != 2 {
				t.Errorf("expected HR Junior count 2, got %d", count)
			}
			// IT Senior should have 1 person (avg 80000)
			if department == "IT" && level == "Senior" && count != 1 {
				t.Errorf("expected IT Senior count 1, got %d", count)
			}
		}
	})

	t.Run("supports multiple aggregate functions", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count":      CountAggregate(),
				"total":      SumAggregate("salary"),
				"average":    AverageAggregate("salary"),
				"min_salary": MinAggregate("salary"),
				"max_salary": MaxAggregate("salary"),
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "salary": 50000},
				{"name": "Bob", "department": "HR", "salary": 60000},
				{"name": "Charlie", "department": "HR", "salary": 70000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 1 {
			t.Errorf("expected 1 group, got %d", len(resultTable.records))
		}

		record := resultTable.records[0]
		if record["count"] != 3 {
			t.Errorf("expected count 3, got %v", record["count"])
		}
		if record["total"] != float64(180000) {
			t.Errorf("expected total 180000, got %v", record["total"])
		}
		if record["average"] != float64(60000) {
			t.Errorf("expected average 60000, got %v", record["average"])
		}
		if record["min_salary"] != float64(50000) {
			t.Errorf("expected min_salary 50000, got %v", record["min_salary"])
		}
		if record["max_salary"] != float64(70000) {
			t.Errorf("expected max_salary 70000, got %v", record["max_salary"])
		}
	})

	t.Run("handles different numeric types correctly", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"category"},
			aggregates: map[string]AggregateFunc{
				"sum_int":   SumAggregate("value_int"),
				"sum_float": SumAggregate("value_float"),
				"sum_int64": SumAggregate("value_int64"),
			},
		}

		doc := New().
			Table("test", []Record{
				{"category": "A", "value_int": 10, "value_float": 10.5, "value_int64": int64(100)},
				{"category": "A", "value_int": 20, "value_float": 20.5, "value_int64": int64(200)},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		record := resultTable.records[0]

		if record["sum_int"] != float64(30) {
			t.Errorf("expected sum_int 30, got %v", record["sum_int"])
		}
		if record["sum_float"] != float64(31) {
			t.Errorf("expected sum_float 31, got %v", record["sum_float"])
		}
		if record["sum_int64"] != float64(300) {
			t.Errorf("expected sum_int64 300, got %v", record["sum_int64"])
		}
	})

	t.Run("handles custom aggregate functions", func(t *testing.T) {
		// Custom aggregate that concatenates strings
		concatAggregate := func(records []Record, field string) any {
			var result []string
			for _, record := range records {
				if val, ok := record[field].(string); ok {
					result = append(result, val)
				}
			}
			if len(result) == 0 {
				return ""
			}
			// Simple string join
			joined := result[0]
			for i := 1; i < len(result); i++ {
				joined += "," + result[i]
			}
			return joined
		}

		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"names": concatAggregate,
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR"},
				{"name": "Bob", "department": "HR"},
				{"name": "Charlie", "department": "IT"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 groups, got %d", len(resultTable.records))
		}

		// Verify custom aggregation worked
		namesByDept := make(map[string]string)
		for _, record := range resultTable.records {
			department := record["department"].(string)
			names := record["names"].(string)
			namesByDept[department] = names
		}

		// Note: order may vary, but we should have both names for HR
		hrNames := namesByDept["HR"]
		if !(hrNames == "Alice,Bob" || hrNames == "Bob,Alice") {
			t.Errorf("expected HR names to contain Alice and Bob, got '%s'", hrNames)
		}
		if namesByDept["IT"] != "Charlie" {
			t.Errorf("expected IT names 'Charlie', got '%s'", namesByDept["IT"])
		}
	})

	t.Run("preserves key order in generated schema", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count":        CountAggregate(),
				"total_salary": SumAggregate("salary"),
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "department": "HR", "salary": 50000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := groupByOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if resultTable.schema == nil {
			t.Fatal("expected schema to be generated")
		}

		keyOrder := resultTable.schema.GetKeyOrder()
		// Should start with groupBy columns, followed by aggregate columns
		expectedStart := []string{"department"}
		for i, expected := range expectedStart {
			if i >= len(keyOrder) || keyOrder[i] != expected {
				t.Errorf("expected key order to start with %v, got %v", expectedStart, keyOrder)
				break
			}
		}

		// Should contain all aggregate columns
		hasCount := false
		hasTotalSalary := false
		for _, key := range keyOrder {
			if key == "count" {
				hasCount = true
			}
			if key == "total_salary" {
				hasTotalSalary = true
			}
		}
		if !hasCount {
			t.Error("expected 'count' in key order")
		}
		if !hasTotalSalary {
			t.Error("expected 'total_salary' in key order")
		}
	})

	t.Run("fails when applied to non-table content", func(t *testing.T) {
		groupByOp := &GroupByOp{
			groupBy: []string{"department"},
			aggregates: map[string]AggregateFunc{
				"count": CountAggregate(),
			},
		}

		doc := New().
			Text("Some text content").
			Build()

		textContent := doc.GetContents()[0]
		_, err := groupByOp.Apply(context.Background(), textContent)
		if err == nil {
			t.Fatal("expected error when applying groupBy to non-table content")
		}

		if !strings.Contains(err.Error(), "groupBy operation requires table content") {
			t.Errorf("expected error about groupBy operation requiring table content, got '%s'", err.Error())
		}
	})
}

// TestAddColumnOpValidation tests AddColumnOp validation logic
func TestAddColumnOpValidation(t *testing.T) {
	t.Run("validates when column name and function are provided", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "calculated_field",
			fn:   func(r Record) any { return "test" },
		}

		err := addColumnOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("fails validation when column name is empty", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "",
			fn:   func(r Record) any { return "test" },
		}

		err := addColumnOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for empty column name")
		}

		if !strings.Contains(err.Error(), "addColumn operation requires a non-empty column name") {
			t.Errorf("expected error about addColumn requiring non-empty column name, got '%s'", err.Error())
		}
	})

	t.Run("fails validation when calculation function is nil", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "calculated_field",
			fn:   nil,
		}

		err := addColumnOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for nil function")
		}

		if !strings.Contains(err.Error(), "addColumn operation requires a calculation function") {
			t.Errorf("expected error about addColumn requiring calculation function, got '%s'", err.Error())
		}
	})

	t.Run("validates position when specified", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name:     "calculated_field",
			fn:       func(r Record) any { return "test" },
			position: intPtr(1),
		}

		err := addColumnOp.Validate()
		if err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("fails validation for negative position", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name:     "calculated_field",
			fn:       func(r Record) any { return "test" },
			position: intPtr(-1),
		}

		err := addColumnOp.Validate()
		if err == nil {
			t.Fatal("expected validation error for negative position")
		}

		if !strings.Contains(err.Error(), "addColumn position must be non-negative") {
			t.Errorf("expected error about addColumn position being non-negative, got '%s'", err.Error())
		}
	})
}

// TestAddColumnOpApply tests AddColumnOp Apply method functionality
func TestAddColumnOpApply(t *testing.T) {
	t.Run("adds column with string data type", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "full_name",
			fn: func(r Record) any {
				first, _ := r["first_name"].(string)
				last, _ := r["last_name"].(string)
				return first + " " + last
			},
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "first_name": "John", "last_name": "Doe"},
				{"id": 2, "first_name": "Jane", "last_name": "Smith"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		if len(resultTable.records) != 2 {
			t.Errorf("expected 2 records, got %d", len(resultTable.records))
		}

		// Check that the new column was added
		for i, record := range resultTable.records {
			if _, exists := record["full_name"]; !exists {
				t.Errorf("record %d missing 'full_name' column", i)
			}
		}

		// Check calculated values
		if resultTable.records[0]["full_name"] != "John Doe" {
			t.Errorf("expected 'John Doe', got %v", resultTable.records[0]["full_name"])
		}
		if resultTable.records[1]["full_name"] != "Jane Smith" {
			t.Errorf("expected 'Jane Smith', got %v", resultTable.records[1]["full_name"])
		}
	})

	t.Run("adds column with numeric data type", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "total_score",
			fn: func(r Record) any {
				math, _ := r["math_score"].(int)
				english, _ := r["english_score"].(int)
				return math + english
			},
		}

		doc := New().
			Table("test", []Record{
				{"student": "Alice", "math_score": 85, "english_score": 92},
				{"student": "Bob", "math_score": 78, "english_score": 88},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)

		// Check calculated values
		if resultTable.records[0]["total_score"] != 177 {
			t.Errorf("expected 177, got %v", resultTable.records[0]["total_score"])
		}
		if resultTable.records[1]["total_score"] != 166 {
			t.Errorf("expected 166, got %v", resultTable.records[1]["total_score"])
		}
	})

	t.Run("calculated field accesses all record data", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "summary",
			fn: func(r Record) any {
				// Access multiple fields from the record
				name, _ := r["name"].(string)
				age, _ := r["age"].(int)
				department, _ := r["department"].(string)
				salary, _ := r["salary"].(int)

				return fmt.Sprintf("%s (%d) works in %s earning %d", name, age, department, salary)
			},
		}

		doc := New().
			Table("test", []Record{
				{"name": "Alice", "age": 30, "department": "Engineering", "salary": 75000},
				{"name": "Bob", "age": 25, "department": "Marketing", "salary": 60000},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)

		expected1 := "Alice (30) works in Engineering earning 75000"
		if resultTable.records[0]["summary"] != expected1 {
			t.Errorf("expected '%s', got %v", expected1, resultTable.records[0]["summary"])
		}

		expected2 := "Bob (25) works in Marketing earning 60000"
		if resultTable.records[1]["summary"] != expected2 {
			t.Errorf("expected '%s', got %v", expected2, resultTable.records[1]["summary"])
		}
	})

	t.Run("updates schema with new field", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "calculated_field",
			fn:   func(r Record) any { return "test_value" },
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		originalFieldCount := len(tableContent.schema.Fields)

		result, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)

		// Check that schema was updated
		if resultTable.schema == nil {
			t.Fatal("expected schema to be present")
		}

		if len(resultTable.schema.Fields) != originalFieldCount+1 {
			t.Errorf("expected %d fields in schema, got %d",
				originalFieldCount+1, len(resultTable.schema.Fields))
		}

		// Check that the new field exists in schema
		newField := resultTable.schema.FindField("calculated_field")
		if newField == nil {
			t.Error("new field 'calculated_field' not found in schema")
		} else {
			if newField.Name != "calculated_field" {
				t.Errorf("expected field name 'calculated_field', got '%s'", newField.Name)
			}
		}

		// Check that key order was updated
		keyOrder := resultTable.schema.GetKeyOrder()
		found := slices.Contains(keyOrder, "calculated_field")
		if !found {
			t.Error("new field 'calculated_field' not found in key order")
		}
	})

	t.Run("appends field by default when no position specified", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "new_field",
			fn:   func(r Record) any { return "new_value" },
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		keyOrder := resultTable.schema.GetKeyOrder()

		// New field should be at the end
		expectedOrder := []string{"id", "name", "new_field"}
		if len(keyOrder) != len(expectedOrder) {
			t.Errorf("expected %d keys, got %d", len(expectedOrder), len(keyOrder))
		}

		for i, expected := range expectedOrder {
			if i >= len(keyOrder) || keyOrder[i] != expected {
				t.Errorf("expected key order %v, got %v", expectedOrder, keyOrder)
				break
			}
		}
	})

	t.Run("inserts field at specified position", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name:     "middle_field",
			fn:       func(r Record) any { return "middle_value" },
			position: intPtr(1), // Insert between id and name
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		keyOrder := resultTable.schema.GetKeyOrder()

		// New field should be at position 1 (between id and name)
		expectedOrder := []string{"id", "middle_field", "name"}
		if len(keyOrder) != len(expectedOrder) {
			t.Errorf("expected %d keys, got %d", len(expectedOrder), len(keyOrder))
		}

		for i, expected := range expectedOrder {
			if i >= len(keyOrder) || keyOrder[i] != expected {
				t.Errorf("expected key order %v, got %v", expectedOrder, keyOrder)
				break
			}
		}
	})

	t.Run("handles position at beginning", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name:     "first_field",
			fn:       func(r Record) any { return "first_value" },
			position: intPtr(0), // Insert at beginning
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		keyOrder := resultTable.schema.GetKeyOrder()

		// New field should be first
		expectedOrder := []string{"first_field", "id", "name"}
		if len(keyOrder) != len(expectedOrder) {
			t.Errorf("expected %d keys, got %d", len(expectedOrder), len(keyOrder))
		}

		for i, expected := range expectedOrder {
			if i >= len(keyOrder) || keyOrder[i] != expected {
				t.Errorf("expected key order %v, got %v", expectedOrder, keyOrder)
				break
			}
		}
	})

	t.Run("handles position beyond end (appends)", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name:     "last_field",
			fn:       func(r Record) any { return "last_value" },
			position: intPtr(10), // Beyond end, should append
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)
		keyOrder := resultTable.schema.GetKeyOrder()

		// Should be appended to end
		expectedOrder := []string{"id", "name", "last_field"}
		if len(keyOrder) != len(expectedOrder) {
			t.Errorf("expected %d keys, got %d", len(expectedOrder), len(keyOrder))
		}

		for i, expected := range expectedOrder {
			if i >= len(keyOrder) || keyOrder[i] != expected {
				t.Errorf("expected key order %v, got %v", expectedOrder, keyOrder)
				break
			}
		}
	})

	t.Run("supports different data types in calculation", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "mixed_calculation",
			fn: func(r Record) any {
				// Test with different data types
				id, _ := r["id"].(int)
				score, _ := r["score"].(float64)
				active, _ := r["active"].(bool)

				if active {
					return float64(id) + score
				}
				return 0.0
			},
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "score": 85.5, "active": true},
				{"id": 2, "score": 92.0, "active": false},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		result, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		resultTable := result.(*TableContent)

		// First record: 1 + 85.5 = 86.5 (active is true)
		if resultTable.records[0]["mixed_calculation"] != 86.5 {
			t.Errorf("expected 86.5, got %v", resultTable.records[0]["mixed_calculation"])
		}

		// Second record: 0.0 (active is false)
		if resultTable.records[1]["mixed_calculation"] != 0.0 {
			t.Errorf("expected 0.0, got %v", resultTable.records[1]["mixed_calculation"])
		}
	})

	t.Run("fails when applied to non-table content", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "calculated_field",
			fn:   func(r Record) any { return "test" },
		}

		doc := New().
			Text("Some text content").
			Build()

		textContent := doc.GetContents()[0]
		_, err := addColumnOp.Apply(context.Background(), textContent)
		if err == nil {
			t.Fatal("expected error when applying addColumn to non-table content")
		}

		if !strings.Contains(err.Error(), "addColumn operation requires table content") {
			t.Errorf("expected error about addColumn operation requiring table content, got '%s'", err.Error())
		}
	})

	t.Run("preserves original content immutability", func(t *testing.T) {
		addColumnOp := &AddColumnOp{
			name: "new_field",
			fn:   func(r Record) any { return "new_value" },
		}

		doc := New().
			Table("test", []Record{
				{"id": 1, "name": "Alice"},
			}).
			Build()

		tableContent := doc.GetContents()[0].(*TableContent)
		originalFieldCount := len(tableContent.schema.Fields)
		originalRecordFieldCount := len(tableContent.records[0])

		_, err := addColumnOp.Apply(context.Background(), tableContent)
		if err != nil {
			t.Fatalf("Apply() failed: %v", err)
		}

		// Original should be unchanged
		if len(tableContent.schema.Fields) != originalFieldCount {
			t.Errorf("original schema was modified: had %d fields, now has %d",
				originalFieldCount, len(tableContent.schema.Fields))
		}
		if len(tableContent.records[0]) != originalRecordFieldCount {
			t.Errorf("original record was modified: had %d fields, now has %d",
				originalRecordFieldCount, len(tableContent.records[0]))
		}
	})
}

// Helper function for tests
func intPtr(i int) *int {
	return &i
}

// TestBuiltInAggregateFunctions tests the built-in aggregate functions
func TestBuiltInAggregateFunctions(t *testing.T) {
	t.Run("CountAggregate counts records", func(t *testing.T) {
		countFunc := CountAggregate()
		records := []Record{
			{"id": 1}, {"id": 2}, {"id": 3},
		}

		result := countFunc(records, "")
		if result != 3 {
			t.Errorf("expected count 3, got %v", result)
		}
	})

	t.Run("SumAggregate sums numeric values", func(t *testing.T) {
		sumFunc := SumAggregate("value")
		records := []Record{
			{"value": 10},
			{"value": 20},
			{"value": 30},
		}

		result := sumFunc(records, "value")
		if result != float64(60) {
			t.Errorf("expected sum 60, got %v", result)
		}
	})

	t.Run("SumAggregate handles mixed numeric types", func(t *testing.T) {
		sumFunc := SumAggregate("value")
		records := []Record{
			{"value": 10},        // int
			{"value": 20.5},      // float64
			{"value": int64(30)}, // int64
		}

		result := sumFunc(records, "value")
		if result != float64(60.5) {
			t.Errorf("expected sum 60.5, got %v", result)
		}
	})

	t.Run("SumAggregate ignores non-numeric values", func(t *testing.T) {
		sumFunc := SumAggregate("value")
		records := []Record{
			{"value": 10},
			{"value": "invalid"},
			{"value": 20},
		}

		result := sumFunc(records, "value")
		if result != float64(30) {
			t.Errorf("expected sum 30 (ignoring invalid), got %v", result)
		}
	})

	t.Run("AverageAggregate calculates mean", func(t *testing.T) {
		avgFunc := AverageAggregate("value")
		records := []Record{
			{"value": 10},
			{"value": 20},
			{"value": 30},
		}

		result := avgFunc(records, "value")
		if result != float64(20) {
			t.Errorf("expected average 20, got %v", result)
		}
	})

	t.Run("MinAggregate finds minimum", func(t *testing.T) {
		minFunc := MinAggregate("value")
		records := []Record{
			{"value": 30},
			{"value": 10},
			{"value": 20},
		}

		result := minFunc(records, "value")
		if result != float64(10) {
			t.Errorf("expected min 10, got %v", result)
		}
	})

	t.Run("MaxAggregate finds maximum", func(t *testing.T) {
		maxFunc := MaxAggregate("value")
		records := []Record{
			{"value": 30},
			{"value": 10},
			{"value": 20},
		}

		result := maxFunc(records, "value")
		if result != float64(30) {
			t.Errorf("expected max 30, got %v", result)
		}
	})

	t.Run("aggregates handle empty record sets", func(t *testing.T) {
		records := []Record{}

		if CountAggregate()(records, "") != 0 {
			t.Error("expected count 0 for empty records")
		}
		if SumAggregate("value")(records, "value") != float64(0) {
			t.Error("expected sum 0 for empty records")
		}
		if AverageAggregate("value")(records, "value") != float64(0) {
			t.Error("expected average 0 for empty records")
		}
		if MinAggregate("value")(records, "value") != float64(0) {
			t.Error("expected min 0 for empty records")
		}
		if MaxAggregate("value")(records, "value") != float64(0) {
			t.Error("expected max 0 for empty records")
		}
	})
}
