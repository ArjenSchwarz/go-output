package output

import (
	"context"
	"strings"
	"testing"
	"time"
)

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
