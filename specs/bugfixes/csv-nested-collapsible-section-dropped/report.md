# Bugfix Report: CSV Nested Collapsible Section Dropped

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-2283

## Description of the Issue

The CSV renderer produced no output for a `DefaultCollapsibleSection` placed
inside a regular `SectionContent`: neither the `# Section:` metadata row nor any
of its nested table rows. The same document renders correctly in the table,
markdown and HTML renderers.

**Reproduction steps:**
1. Create a `SectionContent` and add a `DefaultCollapsibleSection` holding a table to it.
2. Add the section to a document and render it with `NewCSVRendererWithCollapsible`.
3. Observe empty output, with no error.

**Impact:** High. Silent, total data loss for a supported composition
(`SectionContent.AddContent` accepts any `Content`).

## Investigation Summary

- **Symptoms examined:** Rendering the nested composition to CSV returns an empty byte slice and a nil error.
- **Code inspected:** `v2/csv_renderer.go` (`renderDocumentCSVTo`, `renderSectionTablesCSV`, `renderCollapsibleSectionCSV`), and `v2/table_renderer.go` (`renderSectionTable`) for the reference behaviour.
- **Hypotheses tested:** Transformations removing the content (ruled out: content reaches the type switch untouched); the collapsible renderer failing (ruled out: it is never called).

## Discovered Root Cause

`renderSectionTablesCSV` switched on the transformed content type with cases for
`*TableContent` and `*SectionContent` only, and no default. A nested
`*DefaultCollapsibleSection` matched nothing and was skipped.

**Defect type:** Missing case in type dispatch.

**Why it occurred:** The top-level loop in `renderDocumentCSVTo` gained a
`*DefaultCollapsibleSection` case, but the section-flattening helper (added in
T-1239) only mirrored the table and section cases. The table renderer's
equivalent `renderSectionTable` handles the nesting, so the gap was CSV-only.

**Contributing factors:** Dispatch logic is duplicated between the top-level
loop and the section helper (tracked separately as T-2031).

## Resolution for the Issue

**Changes made:**
- `v2/csv_renderer.go` (`renderSectionTablesCSV`) - added a
  `*DefaultCollapsibleSection` case delegating to `renderCollapsibleSectionCSV`
  with the shared `lastKeyOrder` and document-level `flushCSV`, with the same
  error wrapping as the top-level case.

**Approach rationale:** Reuses the existing collapsible renderer, so nested
collapsible sections behave exactly like top-level ones (metadata row,
transformations, header re-writes on schema change, separators).

**Alternatives considered:**
- Adding a `default` branch to render other content types - not done. Text and
  raw content inside regular sections are currently skipped by CSV by design of
  the flattening path; changing that alters existing output and is outside this
  ticket's scope.
- Unifying dispatch between the top-level loop and the section helper - deferred to T-2031.

## Regression Test

**Test file:** `v2/renderer_collapsible_transform_test.go`
**Test name:** `TestCSVRenderer_NestedCollapsibleSectionRendered`

**What it verifies:** A collapsible section nested in a regular section renders
its metadata row and only the rows surviving its table's filter+limit
transformations (mirrors `TestTableRenderer_NestedCollapsibleSectionAppliesTransformations`).

**Run command:** `cd v2 && go test -run TestCSVRenderer_NestedCollapsibleSectionRendered .`

## Affected Files

| File | Change |
|------|--------|
| `v2/csv_renderer.go` | Dispatch nested `*DefaultCollapsibleSection` in `renderSectionTablesCSV` |
| `v2/renderer_collapsible_transform_test.go` | Regression test |

## Verification

**Automated:**
- [x] Regression test passes (failed before the fix with empty output)
- [x] Full test suite passes, except two pre-existing failures unrelated to this change (see below)
- [x] Linters/validators pass (`make lint`: 0 issues; `go vet ./...`; `make doc-examples`)

`TestFileWriter_AppendErrorHandling/file_read_error_includes_operation_and_path`
and `TestFileWriter_AppendIOErrorsIncludeContext/read_error_includes_'read'_operation`
also fail on unmodified main in this environment: they rely on `chmod` making a
file unreadable, which has no effect when tests run as root.

## Prevention

**Recommendations to avoid similar bugs:**
- When adding a content-type case to one renderer dispatch, check every other dispatch site in the same renderer.
- T-2031 (dispatch unification) would remove the duplicated switch.

## Related

- T-2283, T-1635 (PR #123), T-1239, T-2031
