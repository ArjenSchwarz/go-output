# Bugfix Report: Graph Renderers Skip Table Transformations

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

The DOT, Mermaid, and Draw.io renderers bypassed per-content transformations
(filter/sort/limit) on their graph/table-specific render paths. As a result,
a table built with `WithTransformations(...)` rendered transformed rows through
JSON/CSV/HTML/Table/Markdown (which go through `baseRenderer.renderTransformedDocument`),
but rendered the original, untransformed rows through DOT/Mermaid/Draw.io.

**Reproduction steps:**
1. Build a document with a `Table` that has from/to columns and attach
   `WithTransformations(NewFilterOp(...), NewLimitOp(2))`.
2. Render the document with the DOT, Mermaid, or Draw.io renderer.
3. Observe that rows which should have been filtered out or removed by the limit
   still appear in the diagram/CSV output.

**Impact:** Medium. Diagrams and Draw.io CSV exports produced from the same
document as other formats could contain incorrect (extra, unsorted, or unlimited)
data, silently disagreeing with every other output format.

## Investigation Summary

- **Symptoms examined:** Graph outputs included rows that the attached filter and
  limit operations should have removed.
- **Code inspected:** `v2/base_renderer.go` (where ordinary renderers apply
  `applyContentTransformations`), `v2/renderer.go` (`applyContentTransformations`
  helper), `v2/graph_renderers.go` (the three custom renderers), `v2/operations.go`
  (filter/sort/limit `Apply`).
- **Hypotheses tested:** Confirmed that `applyContentTransformations` is only
  invoked inside `baseRenderer.renderTransformedDocument` / `renderDocumentTo`.
  The graph renderers implement their own `Render` loops that iterate
  `doc.GetContents()` and type-switch on the raw content without ever calling
  `applyContentTransformations`.

## Discovered Root Cause

The DOT, Mermaid, and Draw.io renderers each have bespoke `Render` methods that
read `*TableContent.records`/`schema` (or `*GraphContent`/`*ChartContent`) directly
from `doc.GetContents()`. Per-content transformations live only in
`applyContentTransformations`, which these custom paths never call, so the
transformations attached to the content are ignored.

**Defect type:** Missing call to existing transformation step on alternate code paths.

**Why it occurred:** Per-content transformations were centralised in
`baseRenderer.renderTransformedDocument`. The graph renderers build a single
combined output (one digraph / one mermaid graph / one CSV) instead of
concatenating per-content output, so they did not reuse that base path and
therefore skipped the transformation step.

**Contributing factors:** The transformation logic is not enforced by the
`Content` interface; it is the renderer's responsibility to call it, making it
easy to omit on a custom path.

## Resolution for the Issue

**Changes made:**
- `v2/graph_renderers.go` — DOT `Render`: apply `applyContentTransformations(ctx, content)`
  at the top of the content loop and switch on the transformed content.
- `v2/graph_renderers.go` — Mermaid `Render`: transform each content once up front
  into a transformed slice; use the transformed slice for both the detection pass
  and the flowchart render pass.
- `v2/graph_renderers.go` — Draw.io `Render`: transform each content once up front
  into a transformed slice; use it for both the compatibility-detection pass and
  the render pass.

**Approach rationale:** Reuse the existing `applyContentTransformations` helper
(the same one the base renderer uses) so graph output matches every other format.
For Mermaid and Draw.io, which scan content twice (once to decide headers/compatibility,
once to render), the content is transformed a single time into a reused slice to keep
the two passes consistent and avoid transforming twice.

**Alternatives considered:**
- Route graph renderers through `baseRenderer.renderTransformedDocument` — rejected
  because those renderers intentionally produce a single combined document
  (one digraph/graph/CSV) rather than per-content concatenation, so the base loop
  does not fit.

## Regression Test

**Test file:** `v2/graph_renderers_test.go`
**Test names:** `TestDOTRenderer_AppliesTableTransformations`,
`TestMermaidRenderer_AppliesTableTransformations`,
`TestDrawioRenderer_AppliesTableTransformations`

**What it verifies:** A table with from/to/label columns and a filter (`keep == true`)
plus a limit (2) renders only the two surviving rows (`keep1`, `keep2`) through each
renderer, and the filtered-out (`drop1`) and limited-off (`keep3`) rows are absent.

**Run command:** `go test -run 'AppliesTableTransformations' ./v2/`

## Affected Files

| File | Change |
|------|--------|
| `v2/graph_renderers.go` | Apply `applyContentTransformations` before graph detection/render in DOT, Mermaid, Draw.io |
| `v2/graph_renderers_test.go` | Added three regression tests |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed red state before the fix (all rows present) and green after.

## Prevention

**Recommendations to avoid similar bugs:**
- Any renderer with a custom `Render` path that reads content data directly must
  call `applyContentTransformations` first, matching `baseRenderer`.
- Consider a shared helper that yields transformed contents so custom renderers
  cannot forget the transformation step.

## Related

- Transit ticket T-1091
