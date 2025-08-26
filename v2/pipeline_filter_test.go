package output

import (
	"slices"
	"strings"
	"testing"
)

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
		if !strings.Contains(err.Error(), "filter predicate function is required") {
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
