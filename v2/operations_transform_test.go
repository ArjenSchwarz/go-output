package output

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
)

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
