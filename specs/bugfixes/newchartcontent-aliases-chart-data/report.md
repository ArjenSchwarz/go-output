# Bugfix Report: NewChartContent Aliases Known Chart Data

**Date:** 2026-11-02
**Status:** Fixed
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

**Changes made:**
- `v2/graph_content.go` `NewChartContent` - stores `cloneChartData(data)` instead of `data`; godoc states known chart data is copied and other data is stored as provided.
- `v2/graph_content.go` `NewGanttChart` / `NewPieChart` - pass the caller's slice into the fresh `GanttData`/`PieData` and rely on `NewChartContent` for the copy, so the data is copied once instead of twice. Behaviour is unchanged (the T-1305 tests still pass).

**Approach rationale:** `cloneChartData` already holds the ownership rules for the known chart types and is used by `GetData` and `Clone`; reusing it at the constructor makes all entry and exit points share one definition.

**Alternatives considered:**
- Reflection-based deep copy of arbitrary `data` - out of scope; unknown user types keep their current stored-as-provided semantics.
- Leaving the pre-copy in `NewGanttChart`/`NewPieChart` - harmless but copies the slices twice for no benefit.

## Regression Test

**Test file:** `v2/graph_content_mutation_test.go`
**Test names:** `TestNewChartContent_MutateInputGanttDataAfterCreate`, `TestNewChartContent_MutateInputPieDataAfterCreate`

**What it verifies:** After calling `NewChartContent` with a `*GanttData` or
`*PieData`, mutating the caller's struct fields, slice elements, slice header
and (for Gantt) a task's `Dependencies` changes neither `GetData()` nor the
`AppendText` output.

**Run command:** `cd v2 && go test -run 'TestNewChartContent_MutateInput' .`

## Affected Files

| File | Change |
|------|--------|
| `v2/graph_content.go` | Copy known chart data in `NewChartContent`; drop redundant pre-copy in `NewGanttChart`/`NewPieChart` |
| `v2/graph_content_mutation_test.go` | Regression tests for direct `NewChartContent` calls |
| `CHANGELOG.md` | Unreleased "Fixed" entry |

## Verification

**Automated:**
- [x] Regression tests fail before the fix (7 subtests) and pass after
- [x] Full unit and integration suites pass, except two FileWriter read-error subtests that also fail on unmodified `main` in this environment (they rely on `chmod 000` blocking reads, which does not apply when tests run as root)
- [x] `golangci-lint run` reports 0 issues; `go vet ./...` clean

## Prevention

**Recommendations to avoid similar bugs:**
- When a type has a clone helper, apply it at every constructor that accepts caller data, not only the convenience constructors and getters.

## Related

- T-1305 (defensive copies for `NewGanttChart`, `NewPieChart`, `GetData`, `Clone`)
- T-1295, T-1371 (equivalent fixes for graph and Draw.io content)
