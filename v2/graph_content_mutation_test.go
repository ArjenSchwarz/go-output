package output

import "testing"

// Regression tests for T-1295: Graph and Draw.io content expose mutable
// caller-owned data. Constructors and getters must defensively copy slices
// and record maps so that mutating caller-owned data after creation, or
// mutating data returned by a getter, cannot change later render output.

// TestGraphContent_MutateInputEdgesAfterCreate verifies that mutating the
// caller's edge slice after NewGraphContent does not affect the stored edges.
func TestGraphContent_MutateInputEdgesAfterCreate(t *testing.T) {
	edges := []Edge{
		{From: "A", To: "B", Label: "edge1"},
		{From: "B", To: "C", Label: "edge2"},
	}

	g := NewGraphContent("graph", edges)

	// Mutate the caller-owned slice after construction.
	edges[0] = Edge{From: "X", To: "Y", Label: "mutated"}

	got := g.GetEdges()
	if got[0].From != "A" || got[0].To != "B" || got[0].Label != "edge1" {
		t.Errorf("mutating input slice changed stored edge: got %+v, want {From:A To:B Label:edge1}", got[0])
	}
}

// TestGraphContent_MutateEdgesThroughGetter verifies that mutating the slice
// returned by GetEdges does not affect the content's internal state.
func TestGraphContent_MutateEdgesThroughGetter(t *testing.T) {
	edges := []Edge{
		{From: "A", To: "B", Label: "edge1"},
	}

	g := NewGraphContent("graph", edges)

	// Mutate the slice returned by the getter.
	returned := g.GetEdges()
	returned[0] = Edge{From: "X", To: "Y", Label: "mutated"}

	got := g.GetEdges()
	if got[0].From != "A" || got[0].To != "B" || got[0].Label != "edge1" {
		t.Errorf("mutating getter result changed stored edge: got %+v, want {From:A To:B Label:edge1}", got[0])
	}
}

// TestDrawIOContent_MutateInputRecordsAfterCreate verifies that mutating the
// caller's records slice (and the maps within it) after NewDrawIOContent does
// not affect the stored records.
func TestDrawIOContent_MutateInputRecordsAfterCreate(t *testing.T) {
	records := []Record{
		{"Name": "Server1", "Type": "Web"},
		{"Name": "Server2", "Type": "DB"},
	}

	d := NewDrawIOContent("infra", records, DefaultDrawIOHeader())

	// Mutate both the slice element and a map value the caller still owns.
	records[0] = Record{"Name": "replaced"}
	records[1]["Type"] = "mutated"

	got := d.GetRecords()
	if got[0]["Name"] != "Server1" {
		t.Errorf("mutating input slice element changed stored record: got %v, want Server1", got[0]["Name"])
	}
	if got[1]["Type"] != "DB" {
		t.Errorf("mutating input record map changed stored record: got %v, want DB", got[1]["Type"])
	}
}

// TestDrawIOContent_MutateRecordsThroughGetter verifies that mutating the
// records slice and inner maps returned by GetRecords does not affect the
// content's internal state.
func TestDrawIOContent_MutateRecordsThroughGetter(t *testing.T) {
	records := []Record{
		{"Name": "Server1", "Type": "Web"},
	}

	d := NewDrawIOContent("infra", records, DefaultDrawIOHeader())

	// Mutate the slice and inner map returned by the getter.
	returned := d.GetRecords()
	returned[0]["Type"] = "mutated"

	got := d.GetRecords()
	if got[0]["Type"] != "Web" {
		t.Errorf("mutating getter result changed stored record: got %v, want Web", got[0]["Type"])
	}
}

// TestDrawIOContentFromTable_MutateSourceTableRecords verifies that mutating
// the source table's records after NewDrawIOContentFromTable does not affect
// the Draw.io content (the from-table constructor must not alias table data).
func TestDrawIOContentFromTable_MutateSourceTableRecords(t *testing.T) {
	records := []Record{
		{"Name": "Server1", "Type": "Web"},
	}
	table, err := NewTableContent("table", records, WithKeys("Name", "Type"))
	if err != nil {
		t.Fatalf("NewTableContent failed: %v", err)
	}

	d := NewDrawIOContentFromTable(table, DefaultDrawIOHeader())

	// Mutate the table's record map through the table's accessor.
	tableRecords := table.records
	tableRecords[0]["Type"] = "mutated"

	got := d.GetRecords()
	if got[0]["Type"] != "Web" {
		t.Errorf("mutating source table record changed Draw.io content: got %v, want Web", got[0]["Type"])
	}
}
