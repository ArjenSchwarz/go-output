package output

import (
	"strings"
	"testing"
)

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
