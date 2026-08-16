# Bugfix Report: Table Renderer Fallback Ignores Transformed Content

**Ticket:** T-1448
**Date:** 2026-08-17
**Status:** Fixed

## Description of the Issue

`v2/table_renderer.go` applies per-content transformations before rendering each
content item, but the default branch of the top-level content switch in
`renderDocumentTable` rendered the original `content` variable instead of the
`transformed` result (`contentBytes, err := content.AppendText(nil)`). When a
transformation returned a content type not explicitly handled by the switch, the
transformation was silently ignored and the pre-transform content was emitted.

**Reproduction steps:**
1. Create a `Content` implementation whose concrete type is not one of the
   types handled by the table renderer's switch (`*TableContent`,
   `*TextContent`, `*SectionContent`, `*RawContent`,
   `*DefaultCollapsibleSection`).
2. Attach a transformation (`Operation`) to it that changes its output.
3. Add it to a document at the top level and render with the table renderer.
4. Observe that the output contains the original, pre-transform text — the
   transformation result is discarded.

**Impact:** Any custom/unknown content type with transformations attached
renders stale, pre-transform output in table format. Silent data divergence:
no error is raised, so callers cannot tell the transformation was dropped. The
built-in content types were unaffected.

## Investigation Summary

- **Symptoms examined:** Transformed output missing for unknown content types
  at the document top level, while nested (section-level) unknown content was
  transformed correctly.
- **Code inspected:** `v2/table_renderer.go` (`renderDocumentTable`,
  `renderSectionTable`), `v2/renderer.go` (`applyContentTransformations`).
- **Hypotheses tested:** Confirmed the transformation pipeline itself works
  (`applyContentTransformations` returns the transformed content); the defect
  is purely in which variable the fallback branch reads. Audited every branch
  of both switches: all typed branches use the type-switched transformed value
  (`c` / `sub`); only the top-level default branch referenced the loop
  variable `content`.

## Discovered Root Cause

In `renderDocumentTable`, the loop computes
`transformed, err := applyContentTransformations(ctx, content)` and switches on
`transformed`, but the `default` branch called `content.AppendText(nil)` and
`content.ID()` — the pre-transform loop variable — instead of `transformed`.

**Defect type:** Logic error (wrong variable referenced in fallback branch).

**Why it occurred:** The transformation step was added in front of an existing
switch; every typed case naturally picked up the transformed value through the
type switch binding (`switch c := transformed.(type)`), but the default branch
had no binding for the concrete type and kept its original reference to
`content`. The T-1522 rewrite introduced `renderSectionTable` with a correct
default branch (using the type-switched `sub`), leaving only the top-level
branch wrong.

**Contributing factors:** The default branch is only reachable with content
types outside the library's own set, so no existing test exercised it with
transformations attached.

## Resolution for the Issue

_To be completed after the fix is implemented._

## Regression Test

**Test file:** `v2/renderer_table_test.go`
**Test names:** `TestTableRenderer_FallbackRendersTransformedContent`,
`TestTableRenderer_NestedFallbackRendersTransformedContent`

**What it verifies:** An unknown content type with a text-replacing
transformation renders the transformed text (never the pre-transform text) both
at the document top level (the bug) and nested inside a section (consistency
guard for the already-correct helper).

**Run command:** `go test -run 'TestTableRenderer_FallbackRendersTransformedContent|TestTableRenderer_NestedFallbackRendersTransformedContent' ./v2/...`

## Affected Files

| File | Change |
|------|--------|
| `v2/table_renderer.go` | Default branch renders `transformed` instead of `content` |
| `v2/renderer_table_test.go` | Regression tests for top-level and nested fallback |
| `CHANGELOG.md` | Fixed entry under Unreleased |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- Audited both content switches in `v2/table_renderer.go` for other branches
  reading the pre-transform variable; none remain.

## Prevention

**Recommendations to avoid similar bugs:**
- When adding a pre-processing step in front of a `switch x := y.(type)`
  block, audit the `default` branch — it is the only branch without a typed
  binding and can silently keep referencing the unprocessed value.
- Cover fallback/default branches with a test double when they are unreachable
  with the library's own types.

## Related

- T-1522 (PR #106) — rewrote section rendering; its `renderSectionTable`
  default branch was already correct and served as the reference behaviour.
- T-1601 — centralises nil-transformed-content handling in
  `applyContentTransformations` (`v2/renderer.go`).
