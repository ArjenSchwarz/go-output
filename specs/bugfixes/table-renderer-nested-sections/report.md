# Bugfix Report: Table Renderer Silently Drops Content in Sections Nested 3+ Levels Deep

**Date:** 2026-07-12
**Status:** Fixed
**Ticket:** T-1522

## Description of the Issue

The table (console) renderer's `renderDocumentTable` handled nested sections by manually unrolling the hierarchy instead of recursing. Level 1 (contents of a top-level section) handled tables, text, and nested sections; level 2 (contents of a nested section) handled only tables. As a result:

- `TextContent` inside a section nested within another section was silently dropped.
- Any `SectionContent` nested three or more levels deep vanished entirely — title and all contents.

No error was reported; the content was simply absent from the output. The CSV renderer had the identical flaw, fixed with recursion in T-1239 (`renderSectionTablesCSV`).

**Reproduction steps:**
1. Build a document with sections nested three deep, with a table in the innermost section: `outer{middle{inner{table}}}`.
2. Render it with the table renderer (`format: table`).
3. Observe the output contains `=== outer ===` and `=== middle ===` but neither `=== inner ===` nor the table. Similarly, text placed inside a depth-2 section never renders.

**Impact:** Medium. Any consumer rendering documents with sections nested 2+ deep to console table format loses text (depth 2) or entire subtrees (depth 3+) with no error signal. Found in code audit.

## Investigation Summary

Fagan-style inspection of `v2/table_renderer.go` against the T-1239 CSV fix as reference.

- **Symptoms examined:** Regression tests confirmed depth-1/2 tables render, depth-3+ tables and headers are absent, and depth-2 text is absent (with a stray blank line where the separator counted the dropped item).
- **Code inspected:** `renderDocumentTable` `case *SectionContent:` (v2/table_renderer.go:112-162); `csv_renderer.go` `renderSectionTablesCSV` (the T-1239 recursive reference); `SectionContent.Contents()` accessor (read-only, unaffected by the T-1543/T-1677 freeze/seal changes).
- **Hypotheses tested:** Confirmed the drop happens in the renderer, not the builder — the document holds the full hierarchy; the inner type switch simply has no branch for text or sections at depth 2, so unmatched content falls through silently.

## Discovered Root Cause

The nested-section branch is a hand-unrolled copy of the outer loop rather than a recursive call, despite a comment claiming "Handle nested sections recursively". `SectionContent` is a recursive data structure (`[]Content` that may contain further `SectionContent`); any fixed unroll depth necessarily loses content beyond it. The unrolled inner loop also handled a narrower type set (tables only) than level 1.

**Defect type:** Logic error (control flow) — loop unrolling standing in for recursion, plus silent fall-through for unmatched content types.

**Why it occurred:** The original implementation targeted the shapes exercised by tests at the time (one nested level, tables only). The mislabelled comment hid the limitation. T-1239 fixed the identical pattern in the CSV renderer but its scope did not include auditing the table renderer for the same duplicated pattern.

**Contributing factors:** No table-renderer test covered text inside nested sections or nesting depth ≥ 3; deep-nesting regression tests existed only for CSV.

## Resolution for the Issue

**Changes made:**
- `v2/table_renderer.go` - Replaced the hand-unrolled `case *SectionContent:` body in `renderDocumentTable` with a call to a new recursive helper `renderSectionTable(ctx, section, result)`, mirroring the T-1239 CSV approach. The helper writes the `=== Title ===` header, applies per-content transformations at every level, renders every content type the top-level loop does (tables, text, raw content, collapsible sections, and an `AppendText` fallback for unknown types), and recurses into nested sections to any depth. Handling the full content-type set closes a second instance of the same defect: `RawContent` and `DefaultCollapsibleSection` nested inside a section were also silently dropped (and left a stray blank line), because the original inner loop matched only tables, text, and sections. The `case *SectionContent:` in `renderDocumentTable` now also wraps recursion errors with the section title for context, matching the collapsible-section case.

**Approach rationale:** Recursion matches the recursive shape of the data, so no fixed depth exists to fall off. The helper preserves the existing per-item rendering and separator logic, so output for previously-working documents (depth 1-2 tables, depth-1 text) is unchanged; only previously-dropped content now appears.

**Alternatives considered:**
- Deepening the manual unroll by one more level - Rejected: same defect class, just moved one level down.
- Passing a `depth` parameter (as sketched in the ticket) - Omitted: headers do not vary by depth in the existing output (`=== Title ===` at every level), so the parameter would be unused; matches the parameterless-depth signature of the T-1239 CSV helper.

## Regression Test

**Test file:** `v2/renderer_table_test.go`
**Test names:** `TestTableRenderer_DeeplyNestedSections`, `TestTableRenderer_NestedSectionTextContent`, `TestTableRenderer_MultipleTablesAcrossNestingLevels`, `TestTableRenderer_NestedRawAndCollapsibleContent`

**What it verifies:** Tables and innermost section headers render at nesting depths 1, 2, 3, and 5; text content inside a nested section renders; tables at different depths within one hierarchy all render; and `RawContent` plus an expanded `DefaultCollapsibleSection` nested inside a section both render alongside a following table. Mirrors the T-1239 CSV regression tests.

**Run command:** `cd v2 && go test -run 'TestTableRenderer_DeeplyNestedSections|TestTableRenderer_NestedSectionTextContent|TestTableRenderer_MultipleTablesAcrossNestingLevels|TestTableRenderer_NestedRawAndCollapsibleContent' .`

## Affected Files

| File | Change |
|------|--------|
| `v2/table_renderer.go` | Recursive `renderSectionTable` helper replaces hand-unrolled nested-section loop |
| `v2/renderer_table_test.go` | Regression tests for nested-section rendering |
| `CHANGELOG.md` | Fixed entry noting the output change for deeply nested documents |

## Verification

**Automated:**
- [x] Regression test passes (`go test -run 'TestTableRenderer' .`)
- [x] Full test suite passes (`make test` and `make test-integration`)
- [x] Linters/validators pass (`make lint` — 0 issues; `gofmt -l` clean)

**Manual verification:**
- Confirmed the failing-test output before the fix showed exactly the reported drops (depth-3+ subtree and nested text absent).

## Prevention

**Recommendations to avoid similar bugs:**
- When handling recursive structures (`SectionContent`), always recurse; never unroll a fixed number of levels.
- When fixing a defect in one renderer, grep the sibling renderers for the same duplicated pattern (this bug is the table-renderer twin of T-1239).
- Prefer a shared section-walking helper over per-renderer copies where output formats allow.

## Related

- T-1239 — identical flaw in the CSV renderer, fixed with `renderSectionTablesCSV` (reference implementation)
- T-1635 — collapsible sections skip nested transformations in csv+table renderers (separate, out of scope here)
