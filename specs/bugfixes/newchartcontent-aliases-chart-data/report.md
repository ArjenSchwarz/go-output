# Bugfix Report: NewChartContent Aliases Known Chart Data

**Date:** 2026-11-02
**Status:** In Progress
**Ticket:** T-1369

## Description of the Issue

`NewChartContent(title, chartType, data)` stored `data` directly in
`ChartContent.data`. T-1305 made `NewGanttChart`, `NewPieChart`, `GetData` and
`Clone` copy chart data, but callers of the generic constructor who passed a
`*GanttData` or `*PieData` still shared that struct with the chart. Changing
the struct, its `Tasks`/`Slices` slices, or a task's `Dependencies` after
construction changed what the chart rendered later.

**Reproduction steps:**
1. Build `data := &GanttData{Tasks: []GanttTask{{Title: "Build", Dependencies: []string{"design"}}}}`
2. Call `c := NewChartContent("p", ChartTypeGantt, data)`
3. Set `data.Tasks[0].Title = "mutated"` (or change `DateFormat`, `Dependencies`, etc.)
4. Observe `c.GetData()` and `c.AppendText(nil)` reflect the mutation

**Impact:** Medium. Callers that reuse or edit a chart data value after adding
it to a document change the earlier chart's output. Only affects direct use of
the generic constructor with the package's own chart data types.

## Investigation Summary

- **Symptoms examined:** Stored chart data changes after the caller edits its own `*GanttData`/`*PieData`.
- **Code inspected:** `v2/graph_content.go` (`NewChartContent`, `NewGanttChart`, `NewPieChart`, `GetData`, `Clone`, `cloneChartData`, `cloneGanttTasks`, `clonePieSlices`); T-1305 tests in `v2/graph_content_mutation_test.go`.
- **Hypotheses tested:** Whether the specialised constructors' copying covered the generic path — it does not; they copy the input slices before calling `NewChartContent`, which then stores the pointer as-is.

## Discovered Root Cause

`NewChartContent` assigns `data: data` without passing it through
`cloneChartData`, the helper `GetData` and `Clone` already use.

**Defect type:** Missing defensive copy (aliasing of caller-owned data).

**Why it occurred:** T-1305 added copying at the specialised constructors
(which build fresh `GanttData`/`PieData` values) and at the read/clone side,
but not at the generic constructor that receives caller-built structs.

**Contributing factors:** `data` is typed `any`, so the copy is easy to miss
and only known types can be copied.

## Resolution for the Issue

_To be completed after the fix._

## Regression Test

**Test file:** `v2/graph_content_mutation_test.go`
**Test names:** `TestNewChartContent_MutateInputGanttDataAfterCreate`, `TestNewChartContent_MutateInputPieDataAfterCreate`

**What it verifies:** After calling `NewChartContent` with a `*GanttData` or
`*PieData`, mutating the caller's struct fields, slice elements, slice header
and (for Gantt) a task's `Dependencies` changes neither `GetData()` nor the
`AppendText` output.

**Run command:** `cd v2 && go test -run 'TestNewChartContent_MutateInput' .`
