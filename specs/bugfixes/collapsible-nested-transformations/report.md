# Bugfix Report: CSV and Table Renderers Skip Nested Transformations in Collapsible Sections

**Ticket:** T-1635 (consolidates T-1637)
**Date:** 2026-08-22
**Status:** Fixed

## Description of the Issue

A `*TableContent` carrying per-content transformations via `WithTransformations(...)` (e.g. `NewFilterOp`, `NewSortOp`, `NewLimitOp`) rendered its untransformed rows in CSV and terminal-table output whenever the table was nested inside a `DefaultCollapsibleSection` (`NewCollapsibleSection`, `NewCollapsibleTable`, `Builder.AddCollapsibleSection`, ...). The same table rendered correctly transformed at the top level of a document or inside a regular `SectionContent`. Nested non-table content in collapsible sections was likewise rendered without its transformations.

**Reproduction steps:**
1. Build a table with `WithTransformations(NewFilterOp(...), NewLimitOp(2))`.
2. Wrap it in `NewCollapsibleSection("Details", []Content{table}, WithSectionExpanded(true))` and build a document with it.
3. Render the document with the CSV renderer or the terminal-table renderer.
4. Observe that filtered-out and limited-out rows appear in the output; rendering the identical table at top level omits them.

**Impact:** Any consumer relying on per-content transformations for filtering data inside collapsible sections silently got unfiltered output in CSV and terminal-table formats. Since filtering is commonly used to exclude rows, this could leak data the user intended to remove. Markdown, HTML, JSON, and YAML renderers were unaffected.

## Investigation Summary

- **Symptoms examined:** Filtered rows (`keep == false`) and rows beyond a `LimitOp` bound appeared in CSV and terminal-table output for tables inside collapsible sections; the same document rendered correctly in markdown.
- **Code inspected:** `v2/csv_renderer.go` (`renderDocumentCSVTo`, `renderSectionTablesCSV`, `renderCollapsibleSectionCSV`), `v2/table_renderer.go` (`renderDocumentTable`, `renderSectionTable`, `renderCollapsibleSection`), `v2/renderer.go` (`applyContentTransformations`), and the markdown/HTML renderers' collapsible paths as the reference implementations.
- **Hypotheses tested:** Confirmed transformations are applied at both document level and regular-section level in both renderers (so the table content itself carries its operations correctly through `Build()`); the gap is isolated to the `DefaultCollapsibleSection` dispatch branch, whose helper functions do not accept a `context.Context` and never call `applyContentTransformations`.

## Discovered Root Cause

`csvRenderer.renderCollapsibleSectionCSV` and `tableRenderer.renderCollapsibleSection` iterate `section.Content()` and render each nested item directly, switching on the raw content value. Every other render path in these two renderers — top-level dispatch and regular `SectionContent` recursion — first calls `applyContentTransformations(ctx, content)` and renders the transformed result. The markdown and HTML renderers' collapsible-section paths already apply transformations, confirming the intended contract.

**Defect type:** Logic omission (missing transformation application) in a duplicated dispatch branch.

**Why it occurred:** The collapsible helpers were written for Requirements 15.7/15.8 before per-content transformations were threaded through the renderers, and their signatures never gained a `ctx` parameter, so they could not call the ctx-requiring helper. Later transformation fixes (T-1091 for graph renderers, T-1239/T-1522 for nested sections) each targeted the specific paths that were reported, leaving the collapsible branch behind.

**Contributing factors:** Each renderer duplicates its content-type dispatch across three sites (top level, regular section, collapsible section), so a cross-cutting behaviour like transformation application must be added in every copy independently. Chore T-2031 tracks unifying these switches.

## Resolution for the Issue

**Changes made:**
- `v2/csv_renderer.go` — `renderCollapsibleSectionCSV` now accepts `ctx context.Context` and the document-level `flushCSV` helper. Each non-nil nested item is passed through `applyContentTransformations(ctx, content)` before the type switch, and the switch (including the non-table fallback) operates on the transformed value. On a transformation error, buffered rows are flushed first and a write error takes precedence, matching `renderDocumentCSVTo` and `renderSectionTablesCSV` (T-1186).
- `v2/table_renderer.go` — `renderCollapsibleSection` now accepts `ctx context.Context`. Each non-nil nested item is passed through `applyContentTransformations(ctx, content)` before the type switch, and the switch — including the `AppendText` fallback — renders the transformed value, not the original (avoiding the T-1448 bug shape). Both call sites (`renderDocumentTable`, `renderSectionTable`) pass their `ctx` through.

**Approach rationale:** This mirrors the pattern already used by the markdown and HTML renderers' collapsible paths and by every other dispatch site in the CSV and table renderers, so all render paths now share one transformation contract. The T-1472 nil-entry guards and rendered-counter separator logic are preserved unchanged, and the change keeps each dispatch switch shaped like the others to ease the planned T-2031 unification.

**Alternatives considered:**
- Applying transformations inside `renderTable`/`renderTableContentCSV` (the leaf table-rendering helpers) — rejected: it would double-apply transformations on paths that already transform, and would not fix nested non-table content.
- Pre-transforming section contents when the document is built — rejected: transformations are defined to run at render time on an immutable document; eagerly materialising them would change observable semantics and mutate shared state.

## Regression Test

**Test file:** `v2/renderer_collapsible_transform_test.go`
**Test names:**
- `TestCSVRenderer_CollapsibleSectionAppliesTransformations`
- `TestTableRenderer_CollapsibleSectionAppliesTransformations`
- `TestTableRenderer_NestedCollapsibleSectionAppliesTransformations`

**What it verifies:** A table with `NewFilterOp` + `NewLimitOp` transformations inside an expanded `NewCollapsibleSection` renders only the surviving rows (`keep1`, `keep2`) and none of the transformed-out rows (`drop1`, `keep3`) in CSV output, in terminal-table output, and in terminal-table output when the collapsible section is itself nested inside a regular section (the second `renderCollapsibleSection` call site).

**Run command:** `go test -run 'CollapsibleSectionAppliesTransformations' ./...` (from `v2/`)

## Affected Files

| File | Change |
|------|--------|
| `v2/csv_renderer.go` | Thread `ctx` + `flushCSV` into `renderCollapsibleSectionCSV`; apply per-content transformations to nested items |
| `v2/table_renderer.go` | Thread `ctx` into `renderCollapsibleSection`; apply per-content transformations to nested items |
| `v2/renderer_collapsible_transform_test.go` | New regression tests |
| `CHANGELOG.md` | Entry under Unreleased → Fixed |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`make test-all`)
- [x] Linters/validators pass (`make check`)

**Manual verification:**
- Confirmed before the fix that the same document renders transformed rows in markdown output but untransformed rows in CSV/table output; after the fix all formats agree.

## Prevention

**Recommendations to avoid similar bugs:**
- Unify the three per-renderer content-dispatch switches so cross-cutting behaviour (transformations, nil guards, cancellation) is implemented once per renderer — tracked as chore T-2031.
- When adding a renderer behaviour that must apply to "all content", audit every dispatch site per renderer (top level, section, collapsible section), not just the reported one.
- Always render the transformed value after calling `applyContentTransformations`; switching on or falling back to the raw content silently discards transformations (T-1448 bug shape).

## Related

- Transit T-1635 (consolidates T-1637)
- T-1091 — same bug shape in the graph renderers
- T-1239 / T-1522 — transformation application in nested regular sections (CSV / table)
- T-1448 — table fallback rendered pre-transform content
- T-1472 — nil-entry guards in collapsible loops (preserved by this fix)
- T-1601 — `applyContentTransformations` now errors on nil operation results (error path handled here)
- T-2031 — planned unification of the table renderer dispatch switches
