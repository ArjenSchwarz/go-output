package output

import (
	"strings"
	"testing"
	"time"
)

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
		if !strings.Contains(err.Error(), "sort operation requires either sort keys or a custom comparator function") {
			t.Errorf("expected sort validation error, got: %v", err)
		}
	})
}
