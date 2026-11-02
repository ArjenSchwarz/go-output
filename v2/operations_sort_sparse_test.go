package output

import (
	"strings"
	"testing"
)

// TestSortOpSparseFirstRecord is a regression test for T-1465.
// SortOp used to validate sort columns against the first record only, so a
// sparse table whose first row omits the sort column failed with
// "sort column ... does not exist" even though the schema and later rows
// contain it. Missing values must instead sort as nil.
func TestSortOpSparseFirstRecord(t *testing.T) {
	tests := map[string]struct {
		opts    []TableOption
		records []map[string]any
		key     SortKey
		wantIDs []int
	}{
		"column in schema, first row lacks it": {
			opts: []TableOption{WithKeys("id", "score")},
			records: []map[string]any{
				{"id": 1},
				{"id": 2, "score": 10},
				{"id": 3, "score": 5},
			},
			key:     SortKey{Column: "score", Direction: Ascending},
			wantIDs: []int{1, 3, 2}, // nil sorts first ascending
		},
		"column in schema, descending": {
			opts: []TableOption{WithKeys("id", "score")},
			records: []map[string]any{
				{"id": 1},
				{"id": 2, "score": 10},
				{"id": 3, "score": 5},
			},
			key:     SortKey{Column: "score", Direction: Descending},
			wantIDs: []int{2, 3, 1},
		},
		"hidden schema field": {
			opts: []TableOption{WithSchema(
				Field{Name: "id"},
				Field{Name: "score", Hidden: true},
			)},
			records: []map[string]any{
				{"id": 1},
				{"id": 2, "score": 10},
			},
			key:     SortKey{Column: "score", Direction: Descending},
			wantIDs: []int{2, 1},
		},
		"column only present in a later row": {
			opts: []TableOption{WithKeys("id")},
			records: []map[string]any{
				{"id": 1},
				{"id": 2, "score": 10},
			},
			key:     SortKey{Column: "score", Direction: Descending},
			wantIDs: []int{2, 1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			table, err := NewTableContent("sparse", tc.records, tc.opts...)
			if err != nil {
				t.Fatalf("NewTableContent: %v", err)
			}

			got, err := NewSortOp(tc.key).Apply(t.Context(), table)
			if err != nil {
				t.Fatalf("Apply returned error for sparse table: %v", err)
			}

			records := got.(*TableContent).records
			if len(records) != len(tc.wantIDs) {
				t.Fatalf("got %d records, want %d", len(records), len(tc.wantIDs))
			}
			for i, want := range tc.wantIDs {
				if records[i]["id"] != want {
					t.Errorf("record %d: got id %v, want %d", i, records[i]["id"], want)
				}
			}
		})
	}
}

// TestSortOpUnknownColumnStillRejected ensures the T-1465 fix keeps rejecting
// a sort column that is in neither the schema nor any record.
func TestSortOpUnknownColumnStillRejected(t *testing.T) {
	table, err := NewTableContent("t", []map[string]any{
		{"id": 1},
		{"id": 2, "score": 10},
	}, WithKeys("id", "score"))
	if err != nil {
		t.Fatalf("NewTableContent: %v", err)
	}

	_, err = NewSortOp(SortKey{Column: "missing", Direction: Ascending}).Apply(t.Context(), table)
	if err == nil {
		t.Fatal("expected error for unknown sort column, got nil")
	}
	if !strings.Contains(err.Error(), "sort column 'missing' does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}
