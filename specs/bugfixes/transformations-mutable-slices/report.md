# Bugfix Report: GetTransformations Exposes Mutable Operation Slices

**Date:** 2026-09-06
**Status:** In Progress
**Ticket:** T-1378

## Description of the Issue

v2 documents are documented as immutable after `Build()`, but the per-content
transformation slices could be changed from outside the library through two
aliasing paths:

1. `TableContent.GetTransformations`, `TextContent.GetTransformations`,
   `RawContent.GetTransformations`, and `SectionContent.GetTransformations`
   (all in `v2/content.go`) returned the internal `[]Operation` slice header.
   A caller that obtained content through `Document.GetContents()` could
   assign into the returned slice and change which operations run on every
   later render.
2. `WithTextTransformations` (`v2/text_options.go`), `WithRawTransformations`
   (`v2/raw_options.go`), and `WithSectionTransformations`
   (`v2/section_options.go`) stored the variadic `ops` slice directly. When a
   caller spreads its own slice (`WithTextTransformations(ops...)`) the content
   shares the caller's backing array, so later writes to that slice change the
   content. They also kept nil entries, unlike `WithTransformations` for tables,
   which has filtered nils since T-1208.

**Reproduction steps:**
1. `doc := New().Table("users", data, WithKeys("name", "active"), WithTransformations(NewFilterOp(onlyActive))).Build()`
2. Render as JSON and note that inactive rows are filtered out.
3. `ops := doc.GetContents()[0].GetTransformations(); ops[0] = NewFilterOp(func(Record) bool { return true })`
4. Render again — the inactive rows now appear; the built document's output changed.

**Impact:** Medium. Any consumer relying on the post-`Build()` immutability
guarantee (`v2/docs/API.md`, `v2/docs/GETTING-STARTED.md`). The mutation is
silent — no API call on the document is needed, only a write into a slice the
library handed out — and, because renders read the slice without
synchronisation, a concurrent write is also a data race.

## Investigation Summary

- **Symptoms examined:** Post-build replacement of an operation via the slice
  returned by `GetTransformations` changed the rendered JSON of a built
  document; mutation of a caller-owned slice passed to the text/raw/section
  options changed the content's transformations; nil operations passed to those
  options were stored and surfaced from `GetTransformations`.
- **Code inspected:** `v2/content.go` (the four `GetTransformations`
  implementations, constructors, `Clone` methods), `v2/table_options.go`
  (`WithTransformations`, which already filters and copies), `v2/text_options.go`,
  `v2/raw_options.go`, `v2/section_options.go`, `v2/renderer.go`
  (`applyContentTransformations`, the only production consumer of
  `GetTransformations`), `v2/collapsible_section.go` and `v2/graph_content.go`
  (`GetTransformations` stubs that return a nil literal and hold no storage, so
  they cannot alias anything), and the sibling fixes T-1086
  (`specs/bugfixes/schema-key-order-leaks-slices`), T-1208
  (`specs/bugfixes/table-transformations-nil-op`), T-1543
  (`specs/bugfixes/mutable-section-contents`), and T-1677
  (`specs/bugfixes/table-transform-mutates-built-documents`).
- **Hypotheses tested:**
  - *Internal code depends on slice identity or nil-ness from
    `GetTransformations`* — ruled out: `applyContentTransformations` only checks
    `len(...) == 0` and iterates; every `Clone` copies the field directly, not
    through the accessor; test doubles implement the interface themselves.
  - *The table option path is also affected* — ruled out for the option:
    `WithTransformations` builds a fresh filtered slice (T-1208), so the caller's
    slice is never stored. The table accessor still aliased its internal slice.

## Discovered Root Cause

The accessors were written as plain getters (with a nil-to-empty
normalisation) and three of the four option functions assigned the variadic
slice directly. Go slices share their backing array, so both paths handed out
or retained a view onto the content's internal state.

**Defect type:** Missing defensive copy / reference aliasing (immutability
violation).

**Why it occurred:** The per-content transformations feature relied on `Clone()`
in the render path to preserve immutability while applying operations, and
treated `GetTransformations` as an internal read path. The defensive-copy
convention established later for schema slices (T-1086) and the nil filter
added to `WithTransformations` (T-1208) were applied to the table path only,
not to the sibling text/raw/section option families.

**Contributing factors:** The existing tests checked counts and order of the
returned operations, which aliasing does not affect, so no test exercised
mutation through the returned or caller-owned slice.

## Resolution for the Issue

_To be completed after the fix is implemented._

## Regression Test

**Test file:** `v2/content_transformations_immutability_test.go`

**Tests:**
- `TestGetTransformations_ReturnsDefensiveCopy` — replacing an entry in the
  slice returned by `GetTransformations` leaves the content unchanged (table,
  text, raw, section). Red before the fix for all four types.
- `TestTransformationOptions_CopyCallerSlice` — mutating the caller's slice
  after passing it with `...` leaves the content unchanged. Red before the fix
  for text, raw, and section; the table option already copied.
- `TestTransformationOptions_DropNilOperations` — nil operations are dropped by
  all four options. Red before the fix for text, raw, and section.
- `TestBuiltDocument_TransformationsCannotBeSwapped` — end-to-end: swapping the
  filter operation obtained from a built document does not change the rendered
  JSON. Red before the fix.
- `TestGetTransformations_AbsentReturnsEmptyNonNil` — content without
  transformations yields an empty, non-nil slice (contract lock; passes before
  and after).

**Run command:**
`cd v2 && go test -run 'TestGetTransformations_|TestTransformationOptions_|TestBuiltDocument_TransformationsCannotBeSwapped' .`

## Affected Files

_To be completed after the fix is implemented._

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

## Prevention

_To be completed after the fix is implemented._

## Related

- Transit ticket T-1378 (this fix)
- T-1086 — schema key order leaked mutable slices (defensive-copy convention)
- T-1208 — `WithTransformations` nil operation filtering
- T-1543 — built documents expose mutable section contents
- T-1677 — `TableContent.Transform` mutates built documents
