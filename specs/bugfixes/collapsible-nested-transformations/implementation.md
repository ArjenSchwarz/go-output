# Implementation Explanation: Collapsible Nested Transformations (T-1635)

Generated during pre-push review of branch `T-1635/bugfix-collapsible-nested-transformations` (PR #123).

## Beginner Level

### What Changed

The go-output library turns structured data (tables, text, sections) into different output formats: CSV files, terminal tables, Markdown, HTML, and so on. Tables can carry "transformations" — instructions like "only show rows where keep is true" or "show at most 2 rows" — that run when the document is rendered.

The library also supports "collapsible sections": groups of content that can be expanded or collapsed, like a `<details>` block on a web page. The bug: when a table with transformations was placed *inside* a collapsible section, the CSV and terminal-table renderers ignored the transformations and printed every row. The same table outside a collapsible section rendered correctly.

The fix teaches those two renderers to apply the transformations to content inside collapsible sections, exactly like they already do everywhere else.

### Why It Matters

Filtering is often used to *remove* data — for example, hiding internal rows before sharing a CSV export. Because the bug silently skipped the filter, rows the user intended to exclude could leak into the output. There was no error or warning; the output just quietly contained too much.

### Key Concepts

- **Renderer**: code that converts the document into one output format (CSV renderer, table renderer, etc.).
- **Per-content transformation**: a filter/sort/limit operation attached to a specific piece of content, applied at render time. Think of it as a saved search applied just before printing.
- **Collapsible section**: a titled wrapper around other content that can be expanded or collapsed.
- **The fix in one sentence**: before rendering each item inside a collapsible section, run its transformations and render the *result*, not the original.

---

## Intermediate Level

### Changes Overview

- `v2/csv_renderer.go` — `renderCollapsibleSectionCSV` now takes `ctx context.Context` and the document-level `flushCSV func() error`. Each non-nil nested item passes through `applyContentTransformations(ctx, content)` before the type switch; the switch (including the non-table fallback) operates on the transformed value. Content-numbering for `# Content N` metadata rows now uses a rendered-entries counter instead of the raw slice index.
- `v2/table_renderer.go` — `renderCollapsibleSection` now takes `ctx`; same transform-then-switch pattern; the `AppendText` fallback renders the transformed value. Both call sites (`renderDocumentTable`, `renderSectionTable`) pass `ctx` through.
- Review follow-up (commit e9c8e2c): per-item `ctx.Done()` checks added to the collapsible loops in both renderers and to `renderSectionTablesCSV`, mirroring the existing guards in `renderSectionTable` and `renderDocumentCSVTo`; CSV cancellation paths flush buffered rows first (T-1186 pattern).
- Tests: `v2/renderer_collapsible_transform_test.go` (three regression tests: CSV, table, and the second table call site via a nested regular section) and `TestCSVCollapsibleContentNumberingSkipsNilEntries` in `v2/collapsible_nil_content_test.go`.

### Implementation Approach

The two renderers each have three dispatch sites (top level, regular section, collapsible section). The first two already followed the contract: call `applyContentTransformations`, switch on the transformed value. The collapsible helpers predated per-content transformations and never gained a `ctx` parameter, so they couldn't call the helper. The fix threads `ctx` (and, for CSV, `flushCSV`) into the helpers and replicates the exact pattern used by the sibling dispatch sites — including the CSV error-path rule from T-1186: on a transformation error, flush buffered rows first and prefer a surfaced write error, so a failing destination isn't masked.

Two prior-bug patterns are deliberately respected:

- **T-1448**: after transforming, the fallback branches render the transformed value (`contentItem`/`c`), never the original `content`. Falling back to the raw value silently discards transformations.
- **T-1472**: nil entries in a malformed section are skipped, and separators/numbering count *rendered* entries so a skipped nil can't produce a leading blank line or a numbering gap (`Content 1, Content 3`).

### Trade-offs

- **Fix at the dispatch sites vs. inside the leaf helpers** (`renderTable`, `renderTableContentCSV`): the leaf-helper option was rejected because paths that already transform would double-apply, and non-table nested content would remain unfixed.
- **Render-time transformation vs. pre-transforming at Build()**: rejected because transformations are defined to run at render time on an immutable document; materialising them early changes observable semantics.
- **Passing `flushCSV` as a function parameter** continues the existing shape used by `renderSectionTablesCSV`; bundling the CSV render state into a struct would be tidier but is a refactor for T-2031, not this fix.

---

## Expert Level

### Technical Deep Dive

`applyContentTransformations` is allocation-free when a content item carries no transformations (`len == 0` short-circuits, no clone), so the new per-item calls cost nothing on the common path; the `Clone()` only runs when operations are actually attached. The call sits per nested content item, never per row — the row-writing leaves are untouched and operate on the already-transformed `*TableContent`.

The CSV error path is subtle: `csv.Writer` buffers, so an underlying writer failure may surface only at `Flush()`. Both the transformation-error and (new) cancellation paths flush first and return the flush error in preference, matching `renderDocumentCSVTo`. Without this, a transformation error could mask the more actionable destination-write failure.

The cancellation follow-up closes a real gap the ctx-threading exposed: `applyContentTransformations` only observes `ctx.Err()` when the item has transformations, so a transformation-free collapsible section would previously render to completion regardless of cancellation. The per-item `select` guard mirrors `renderSectionTable`'s documented pattern.

`contentNum` (CSV) increments before the transformation-error return, so an aborted render could have "counted" an item that produced no metadata row — unobservable, because the error return means no further `# Content N` row is written in that call.

### Architecture Impact

Both renderers' collapsible dispatch switches now have the same shape as their top-level and regular-section siblings: cancellation guard → nil guard → transform → switch on transformed value → fallback renders transformed value. That uniformity is deliberate groundwork for T-2031, which plans to unify the three dispatch switches per renderer so cross-cutting behaviour (transformations, nil guards, cancellation) is written once.

### Potential Issues

- **Nested collapsible sections are not recursed into** by either fixed helper — a `DefaultCollapsibleSection` inside a collapsible section hits the `AppendText` fallback. Known deferral, tracked on T-2031; now noted in code comments on both fallback branches.
- **CSV drops collapsible sections nested in regular sections**: `renderSectionTablesCSV` switches only on `*TableContent` and `*SectionContent`, so a `DefaultCollapsibleSection` inside a regular `SectionContent` is silently skipped in CSV output. Pre-existing, filed as T-2283.
- **Typed-nil transformation results** slip past the T-1601 nil guard; part of the consolidated typed-nil validation sweep (T-2234/T-1649).
- **Type-changing transformations** are handled correctly: the switch runs on the transformed value, so an operation that returns a different content type dispatches to the right branch.

---

## Completeness Assessment

**Fully implemented:**
- Transformations applied to nested content in both renderers' collapsible paths, at both table-renderer call sites (all three covered by regression tests).
- CSV error path flushes first, write error preferred (T-1186 contract).
- T-1472 nil guards and rendered-entry separators/numbering preserved; CSV numbering-gap bug fixed and tested.
- T-1448 pattern respected in both fallbacks.
- Per-item cancellation guards (review follow-up e9c8e2c).

**Partially implemented / deferred by design:**
- Nested-collapsible recursion in the table and CSV collapsible paths — deferred to T-2031 (dispatch unification), noted in code comments.

**Missing (pre-existing, ticketed separately):**
- CSV rendering of collapsible sections nested inside regular sections (T-2283).
- Typed-nil result detection (T-2234/T-1649).

No spec requirement was found that could not be explained; the implementation matches the bugfix report's stated approach and alternatives.
