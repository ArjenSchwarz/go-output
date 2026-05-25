# Bugfix Report: Graph Renderers Emit Unstable Row Order

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

Graph and chart rendering exposed Go's randomized map iteration order in
user-facing output, making generated diagrams nondeterministic between runs.

Two sites were affected:

1. **Draw.io graph node rows** — `GraphContent.GetNodes` built a `map[string]bool`
   set and ranged it to produce the result slice. The Draw.io renderer
   (`renderGraphAsDrawIO`) wrote one CSV node row per returned node, so node
   rows appeared in a different order on each render. The same `GetNodes` also
   feeds the JSON and YAML renderers, so those were affected too.
2. **Mermaid Gantt sections** — `renderGanttChart` grouped tasks into a
   `map[string][]GanttTask` keyed by section and ranged that map to emit section
   blocks, so section ordering was randomized even when the input task order was
   stable.

**Reproduction steps:**
1. Build a `GraphContent` with several edges, or a Gantt chart with tasks across
   multiple sections.
2. Render to Draw.io (graph) or Mermaid (Gantt) repeatedly.
3. Observe that node rows / section blocks appear in a different order on
   different runs.

**Impact:** Medium. No data loss, but nondeterministic output breaks
reproducible builds, diffing of generated diagrams, and golden-file testing.

## Investigation Summary

- **Symptoms examined:** Draw.io CSV node rows and Mermaid Gantt section blocks
  reordering across renders.
- **Code inspected:** `v2/graph_content.go` (`GetNodes`), `v2/graph_renderers.go`
  (`renderGanttChart`, `renderGraphAsDrawIO`), `v2/json_yaml_renderer.go`
  (other `GetNodes` consumers).
- **Hypotheses tested:** Confirmed both sites rely on ranging a map. Edge order
  and task order are themselves stable (stored as slices and defensively
  cloned), so the only nondeterminism source is the map iteration.

## Discovered Root Cause

Both sites used a map purely as a deduplication/grouping mechanism and then
ranged that map to produce ordered output. Go intentionally randomizes map
iteration order, so any ordering derived from ranging a map is nondeterministic.

**Defect type:** Logic error (relying on map iteration order for user-facing
ordering).

**Why it occurred:** A map is the natural structure for set membership and
grouping, but the code conflated "deduplicate/group" with "produce ordered
output" by ranging the map directly instead of tracking order separately.

**Contributing factors:** None environmental; the input data (edges, tasks) was
already in a stable slice order that simply needed to be preserved.

## Resolution for the Issue

Preserve first-seen (insertion) order by tracking an ordered slice alongside the
map, rather than ranging the map.

**Changes made:**
- `v2/graph_content.go` (`GetNodes`) — replace the set-map range with a `seen`
  map plus an ordered `nodes` slice, appending each node the first time it is
  seen across the edges.
- `v2/graph_renderers.go` (`renderGanttChart`) — keep the section grouping map
  but also build a `sectionOrder` slice recording each section the first time it
  appears, then range `sectionOrder` to emit section blocks.

**Approach rationale:** First-seen order keeps output intuitive (it mirrors the
order the user supplied edges/tasks in) and is the smallest change that makes
output deterministic. Sorting was rejected because it would reorder
user-supplied data, which is contrary to the library's key-order-preservation
philosophy.

**Alternatives considered:**
- **Sort nodes/sections alphabetically** — deterministic, but reorders
  user-supplied data and loses the intuitive input ordering.

## Regression Test

**Test files:**
- `v2/graph_content_mutation_test.go` — `TestGraphContent_GetNodesStableOrder`
- `v2/graph_renderers_test.go` — `TestDrawioRenderer_StableNodeOrder`
- `v2/mermaid_chart_test.go` — `TestMermaidRenderer_GanttSectionOrder`

**What they verify:** `GetNodes` returns nodes in first-seen order; the Draw.io
renderer emits node rows in first-seen order; the Mermaid Gantt renderer emits
section blocks in first-seen order. Each test asserts the expected order across
100 iterations so randomized map iteration cannot coincidentally pass.

**Run command:**
`go test -run 'TestGraphContent_GetNodesStableOrder|TestDrawioRenderer_StableNodeOrder|TestMermaidRenderer_GanttSectionOrder' ./v2/...`

## Affected Files

| File | Change |
|------|--------|
| `v2/graph_content.go` | `GetNodes` returns nodes in first-seen order via a `seen` set + ordered slice |
| `v2/graph_renderers.go` | `renderGanttChart` tracks first-seen section order and ranges that instead of the map |
| `v2/graph_content_mutation_test.go` | Added `TestGraphContent_GetNodesStableOrder` |
| `v2/graph_renderers_test.go` | Added `TestDrawioRenderer_StableNodeOrder` |
| `v2/mermaid_chart_test.go` | Added `TestMermaidRenderer_GanttSectionOrder` |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`make test`)
- [x] Linters pass (`make lint`, 0 issues)

**Manual verification:**
- Confirmed the three new tests fail on the pre-fix code (red) showing randomized
  ordering, then pass after the fix (green).

## Prevention

**Recommendations to avoid similar bugs:**
- Never derive user-facing ordering from ranging a Go map. Use a map only for
  membership/grouping and track order with a separate slice.
- When deduplicating or grouping, default to first-seen order to preserve the
  caller's intent (consistent with the library's key-order-preservation design).
- Order-sensitive output should be covered by tests that run multiple iterations
  to defeat coincidental map-iteration passes.

## Related

- Transit ticket: T-1339
