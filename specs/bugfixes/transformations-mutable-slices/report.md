# Bugfix Report: GetTransformations Exposes Mutable Operation Slices

**Date:** 2026-09-06
**Status:** Fixed
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

**Changes made:**
- `v2/pipeline.go` - Added the unexported `cloneOperations` helper next to the
  `Operation` interface. It returns a fresh, non-nil slice holding the non-nil
  entries of its input and is the single place operation slices cross the API
  boundary.
- `v2/content.go` - `TableContent`, `TextContent`, `RawContent`, and
  `SectionContent.GetTransformations` now `return cloneOperations(x.transformations)`,
  which preserves the existing "empty non-nil slice when absent" behaviour. The
  `Content` interface godoc now states that implementations must not expose
  internal state through `GetTransformations`.
- `v2/text_options.go`, `v2/raw_options.go`, `v2/section_options.go` - The
  three option functions store `cloneOperations(ops)` instead of `ops`, so the
  caller's backing array is never retained and nil entries are dropped.
- `v2/table_options.go` - `WithTransformations` reuses `cloneOperations`
  instead of its inline nil-filter loop; behaviour is unchanged.
- `v2/docs/API.md` - Documented the copy semantics under "Content-Specific
  Transformation Options".
- `CHANGELOG.md` - Entry under Unreleased -> Fixed.

**Approach rationale:** Copy-on-input and copy-on-output is the convention the
library already uses for its other "immutable" slices (`Records()`,
`Schema()`, `GetKeyOrder()`, `WithKeys`, `WithSchema` — T-1086), and
`WithTransformations` already did the input half for tables (T-1208). Routing
all seven sites through one helper makes the four content types behave
identically and keeps the nil-filtering rule in one place. The only
production consumer of `GetTransformations`, `applyContentTransformations` in
`v2/renderer.go`, checks the length and iterates, so an extra small
allocation per render of transformed content is the entire cost.

**Alternatives considered:**
- Freezing or sealing the slice (as T-1543/T-1677 did for `AddContent` and
  `Transform`) - Not applicable: the leak is through a returned slice header,
  not a mutating method; there is nothing to gate.
- Returning `nil` instead of an empty slice when no transformations are set -
  Rejected: the existing table tests and the ticket require an empty non-nil
  slice, and changing it would alter the public contract for no benefit.
- Copying only in `GetTransformations` and leaving the options aliasing the
  caller's slice - Rejected: the ticket names both paths, and a caller-owned
  slice spread with `...` is a realistic way to build option lists dynamically.
- Making the graph/chart/Draw.io/collapsible-section stubs return an empty
  slice instead of nil - Out of scope: they hold no transformations, so nothing
  can be aliased; the interface godoc allows nil for such types.

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

| File | Change |
|------|--------|
| `v2/pipeline.go` | New `cloneOperations` helper |
| `v2/content.go` | Four `GetTransformations` implementations return a copy; `Content` interface godoc |
| `v2/text_options.go` | `WithTextTransformations` copies and nil-filters |
| `v2/raw_options.go` | `WithRawTransformations` copies and nil-filters |
| `v2/section_options.go` | `WithSectionTransformations` copies and nil-filters |
| `v2/table_options.go` | `WithTransformations` reuses the helper |
| `v2/content_transformations_immutability_test.go` | Regression tests |
| `v2/docs/API.md` | Copy semantics documented |
| `CHANGELOG.md` | Unreleased -> Fixed entry |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes — `go test ./...` in `v2` green
- [x] Linters/validators pass — `golangci-lint run`: 0 issues; `gofmt -l`: clean

**Manual verification:**
- Confirmed `applyContentTransformations` is the only production caller of
  `GetTransformations` and depends on neither slice identity nor nil-ness, so
  returning a copy changes no rendered output.

## Prevention

**Recommendations to avoid similar bugs:**
- Treat every slice or map field on a content type as copy-on-input and
  copy-on-output; a getter that returns the field directly is an aliasing bug
  on an immutable type.
- When a fix establishes a convention for one option family (as T-1208 did for
  `WithTransformations`), apply it to the sibling families in the same change.
- Immutability tests should mutate through the returned value and through the
  original input, not only assert counts and order.

## Related

- Transit ticket T-1378 (this fix)
- T-1086 — schema key order leaked mutable slices (defensive-copy convention)
- T-1208 — `WithTransformations` nil operation filtering
- T-1543 — built documents expose mutable section contents
- T-1677 — `TableContent.Transform` mutates built documents
