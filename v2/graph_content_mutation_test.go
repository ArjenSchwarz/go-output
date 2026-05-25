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

// Regression tests for T-1305: ChartContent/GanttData/PieData expose mutable
// caller-owned data. NewGanttChart and NewPieChart must defensively copy their
// input slices (including each Gantt task's Dependencies slice), GetData must
// return an independent copy of the chart data, and Clone must produce a deep
// copy so the clone is independent of the original.

// TestNewGanttChart_MutateInputTasksAfterCreate verifies that mutating the
// caller's task slice after NewGanttChart does not affect the stored tasks.
func TestNewGanttChart_MutateInputTasksAfterCreate(t *testing.T) {
	tasks := []GanttTask{
		{ID: "1", Title: "Design", StartDate: "2026-01-01"},
		{ID: "2", Title: "Build", StartDate: "2026-01-02"},
	}

	c := NewGanttChart("project", tasks)

	// Mutate the caller-owned slice element after construction.
	tasks[0] = GanttTask{ID: "X", Title: "mutated"}

	data, ok := c.GetData().(*GanttData)
	if !ok {
		t.Fatalf("GetData did not return *GanttData, got %T", c.GetData())
	}
	if data.Tasks[0].Title != "Design" {
		t.Errorf("mutating input slice changed stored task: got %q, want Design", data.Tasks[0].Title)
	}
}

// TestNewGanttChart_MutateInputTaskDependencies verifies that mutating a task's
// Dependencies slice the caller still owns does not affect the stored task.
func TestNewGanttChart_MutateInputTaskDependencies(t *testing.T) {
	tasks := []GanttTask{
		{ID: "1", Title: "Build", Dependencies: []string{"design"}},
	}

	c := NewGanttChart("project", tasks)

	// Mutate the nested Dependencies slice the caller still owns.
	tasks[0].Dependencies[0] = "mutated"

	data := c.GetData().(*GanttData)
	if data.Tasks[0].Dependencies[0] != "design" {
		t.Errorf("mutating input task dependency changed stored task: got %q, want design", data.Tasks[0].Dependencies[0])
	}
}

// TestNewGanttChart_MutateTasksThroughGetData verifies that mutating the tasks
// reachable through GetData does not affect the chart's internal state.
func TestNewGanttChart_MutateTasksThroughGetData(t *testing.T) {
	tasks := []GanttTask{
		{ID: "1", Title: "Design"},
	}

	c := NewGanttChart("project", tasks)

	// Mutate the data returned by the getter.
	returned := c.GetData().(*GanttData)
	returned.Tasks[0].Title = "mutated"

	again := c.GetData().(*GanttData)
	if again.Tasks[0].Title != "Design" {
		t.Errorf("mutating GetData result changed stored task: got %q, want Design", again.Tasks[0].Title)
	}
}

// TestNewPieChart_MutateInputSlicesAfterCreate verifies that mutating the
// caller's slice after NewPieChart does not affect the stored slices.
func TestNewPieChart_MutateInputSlicesAfterCreate(t *testing.T) {
	slices := []PieSlice{
		{Label: "A", Value: 1},
		{Label: "B", Value: 2},
	}

	c := NewPieChart("dist", slices, true)

	// Mutate the caller-owned slice element after construction.
	slices[0] = PieSlice{Label: "mutated", Value: 99}

	data, ok := c.GetData().(*PieData)
	if !ok {
		t.Fatalf("GetData did not return *PieData, got %T", c.GetData())
	}
	if data.Slices[0].Label != "A" || data.Slices[0].Value != 1 {
		t.Errorf("mutating input slice changed stored slice: got %+v, want {Label:A Value:1}", data.Slices[0])
	}
}

// TestNewPieChart_MutateSlicesThroughGetData verifies that mutating the slices
// reachable through GetData does not affect the chart's internal state.
func TestNewPieChart_MutateSlicesThroughGetData(t *testing.T) {
	slices := []PieSlice{
		{Label: "A", Value: 1},
	}

	c := NewPieChart("dist", slices, true)

	// Mutate the data returned by the getter.
	returned := c.GetData().(*PieData)
	returned.Slices[0].Label = "mutated"

	again := c.GetData().(*PieData)
	if again.Slices[0].Label != "A" {
		t.Errorf("mutating GetData result changed stored slice: got %q, want A", again.Slices[0].Label)
	}
}

// TestChartContent_CloneIndependentGantt verifies that mutating a cloned Gantt
// chart's data does not affect the original chart.
func TestChartContent_CloneIndependentGantt(t *testing.T) {
	tasks := []GanttTask{
		{ID: "1", Title: "Design", Dependencies: []string{"spec"}},
	}

	c := NewGanttChart("project", tasks)
	clone := c.Clone().(*ChartContent)

	// Mutate the clone's data.
	cloneData := clone.GetData().(*GanttData)
	cloneData.Tasks[0].Title = "mutated"
	cloneData.Tasks[0].Dependencies[0] = "mutated"

	origData := c.GetData().(*GanttData)
	if origData.Tasks[0].Title != "Design" {
		t.Errorf("mutating clone changed original task title: got %q, want Design", origData.Tasks[0].Title)
	}
	if origData.Tasks[0].Dependencies[0] != "spec" {
		t.Errorf("mutating clone changed original task dependency: got %q, want spec", origData.Tasks[0].Dependencies[0])
	}
}

// TestChartContent_CloneIndependentPie verifies that mutating a cloned pie
// chart's data does not affect the original chart.
func TestChartContent_CloneIndependentPie(t *testing.T) {
	slices := []PieSlice{
		{Label: "A", Value: 1},
	}

	c := NewPieChart("dist", slices, true)
	clone := c.Clone().(*ChartContent)

	// Mutate the clone's data.
	cloneData := clone.GetData().(*PieData)
	cloneData.Slices[0].Label = "mutated"

	origData := c.GetData().(*PieData)
	if origData.Slices[0].Label != "A" {
		t.Errorf("mutating clone changed original slice: got %q, want A", origData.Slices[0].Label)
	}
}
