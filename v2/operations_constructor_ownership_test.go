package output

import (
	"testing"
)

// Regression tests for T-1467: operation constructors must take ownership of
// caller-provided slices, maps and pointers so that mutating the inputs after
// construction cannot change how the operation behaves when it is applied
// later during rendering.

func TestNewSortOpCopiesKeys(t *testing.T) {
	keys := []SortKey{{Column: "id", Direction: Ascending}}
	op := NewSortOp(keys...)

	keys[0] = SortKey{Column: "id", Direction: Descending}

	table, err := NewTableContent("t", []map[string]any{
		{"id": 2}, {"id": 1}, {"id": 3},
	}, WithKeys("id"))
	if err != nil {
		t.Fatalf("NewTableContent: %v", err)
	}

	got, err := op.Apply(t.Context(), table)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	records := got.(*TableContent).records
	for i, want := range []int{1, 2, 3} {
		if records[i]["id"] != want {
			t.Errorf("record %d: got id %v, want %d", i, records[i]["id"], want)
		}
	}
}

func TestNewGroupByOpCopiesInputs(t *testing.T) {
	groupBy := []string{"team"}
	aggregates := map[string]AggregateFunc{"count": CountAggregate()}
	op := NewGroupByOp(groupBy, aggregates)

	groupBy[0] = "name"
	aggregates["sum_score"] = SumAggregate("score")
	delete(aggregates, "count")

	table, err := NewTableContent("t", []map[string]any{
		{"team": "a", "name": "x", "score": 1},
		{"team": "a", "name": "y", "score": 2},
		{"team": "b", "name": "z", "score": 3},
	}, WithKeys("team", "name", "score"))
	if err != nil {
		t.Fatalf("NewTableContent: %v", err)
	}

	got, err := op.Apply(t.Context(), table)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	result := got.(*TableContent)

	wantKeys := []string{"team", "count"}
	gotKeys := result.schema.GetKeyOrder()
	if len(gotKeys) != len(wantKeys) {
		t.Fatalf("got key order %v, want %v", gotKeys, wantKeys)
	}
	for i := range wantKeys {
		if gotKeys[i] != wantKeys[i] {
			t.Fatalf("got key order %v, want %v", gotKeys, wantKeys)
		}
	}
	if len(result.records) != 2 {
		t.Fatalf("got %d groups, want 2", len(result.records))
	}
}

func TestNewAddColumnOpCopiesPosition(t *testing.T) {
	position := 0
	op := NewAddColumnOp("extra", func(Record) any { return "v" }, &position)

	position = 2

	table, err := NewTableContent("t", []map[string]any{
		{"a": 1, "b": 2},
	}, WithKeys("a", "b"))
	if err != nil {
		t.Fatalf("NewTableContent: %v", err)
	}

	got, err := op.Apply(t.Context(), table)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	gotKeys := got.(*TableContent).schema.GetKeyOrder()
	wantKeys := []string{"extra", "a", "b"}
	for i := range wantKeys {
		if i >= len(gotKeys) || gotKeys[i] != wantKeys[i] {
			t.Fatalf("got key order %v, want %v", gotKeys, wantKeys)
		}
	}
}
