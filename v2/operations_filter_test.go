package output

import (
	"context"
	"strings"
	"testing"
)

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
