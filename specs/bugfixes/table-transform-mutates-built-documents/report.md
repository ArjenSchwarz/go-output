# Bugfix Report: TableContent.Transform Mutates Built Documents

**Date:** 2026-07-10
**Status:** Fixed
**Ticket:** T-1677

## Description of the Issue

`(*TableContent).Transform` in `v2/transform_data.go` mutated `tc.records` in place. Documents are documented as immutable after `Build()`, but `Document.GetContents()` returns the same `Content` interface values the document holds (only the slice header is copied). A caller could therefore obtain a `*TableContent` from a built document and call `Transform` to change what later renders produce. The same path could race with concurrent rendering, because renderers read table records while `Transform` writes them without synchronization.

**Reproduction steps:**
1. Build a document: `doc := New().Table("t", records).Build()`
2. Grab the table: `tbl := doc.GetContents()[0].(*TableContent)`
3. Call `tbl.Transform(fn)` with a function that rewrites the records
4. Render the document — the output reflects the mutated records, violating the immutability contract

**Impact:** Any consumer relying on the documented post-`Build()` immutability guarantee (`v2/docs/GETTING-STARTED.md`, `v2/docs/API.md`). Documents shared across goroutines could observe changed output or a data race if `Transform` ran concurrently with rendering.

## Investigation Summary

Structured inspection of the mutation and render paths.

- **Symptoms examined:** Post-`Build()` mutation of rendered table data via the public `Transform` method; potential data race between `Transform` writes and renderer reads of `tc.records`.
- **Code inspected:** `v2/transform_data.go` (`Transform`, `TransformableContent`), `v2/document.go` (`GetContents`, `Builder.AddContent`, `Build`), `v2/content.go` (`TableContent`, `Clone`, `Records`, `Schema`), `v2/renderer.go` (`applyContentTransformations`), `v2/operations.go` (operation `Apply` implementations), `v2/collapsible_section.go`.
- **Hypotheses tested:**
  - *Internal pipeline relies on in-place Transform mutation* — ruled out: the render pipeline is clone-based (`applyContentTransformations` clones before applying operations; every operation `Apply` clones the table). No non-test code calls `(*TableContent).Transform`.
  - *Other TableContent mutation vectors* — ruled out: `Records()` returns deep copies, `Schema()` returns a defensive clone (fixed previously). `Transform` was the remaining public mutation path.

## Discovered Root Cause

`TableContent` has no notion of being attached to a document, so its public `Transform` method happily mutates the receiver even when that receiver is shared with a built (documented-immutable) document. Two concrete defects in `Transform`:

1. It passed the live `tc.records` slice to the caller's function, so the function could mutate the shared record maps in place — even when it subsequently returned an error.
2. It assigned `tc.records = records` on a receiver that `Document.GetContents()` shares with the built document, changing later renders and racing concurrent readers.

**Defect type:** Missing immutability enforcement (logic error) with a resulting data-race exposure.

**Why it occurred:** v2's immutability design enforced one-shot building at the `Builder` level (T-1689) and defensive copies on accessors (`Records()`, `Schema()`), assuming content is never mutated post-`Build()` because internal code always clones. The public `TransformableContent.Transform` escape hatch — intended for the clone-then-transform pipeline pattern — was overlooked.

**Contributing factors:** `GetContents()` intentionally shares content pointers for performance; the `Transform(fn) error` signature bakes in receiver mutation, so it cannot be made non-mutating without breaking the exported interface.

## Resolution for the Issue

**Changes made:**
- `v2/content.go` - Added an unexported `sealed atomic.Bool` field to `TableContent` with a `seal()` method; `Clone()` intentionally returns an unsealed copy (the sanctioned escape hatch for transforming document data).
- `v2/transform_data.go` - `Transform` now returns an error when the table is sealed (attached to a document), and hands the transformation function a deep copy of the records (via `Records()`) so a failed transform cannot corrupt the table and the function cannot alias internal record maps. Added `sealContents` helper that recursively seals tables nested in sections and collapsible sections; it guards against typed-nil pointers, which pass `AddContent`'s untyped-nil interface check.
- `v2/document.go` - `Builder.AddContent` calls `sealContents(content)`; every builder path (including `Section`/`CollapsibleSection` sub-builders and `AddCollapsibleTable`) routes through it, so all tables reachable from a document get sealed.

**Approach rationale:** The ticket allows either making built contents non-mutable through public APIs or scoping `Transform` so it cannot mutate attached content. Sealing at attach time keeps the exported `TransformableContent` interface intact (no breaking signature change), matches the existing enforcement pattern (T-1689 records errors for post-`Build()` builder mutations), and closes the race: a sealed `Transform` returns before touching `tc.records`. Sealing at `AddContent` rather than `Build()` is strictly safer (the builder shares the pointer from that moment) and avoids restructuring `Build()`/`GetContents()`, which T-1543 is modifying in parallel.

**Alternatives considered:**
- Deep-clone contents in `GetContents()` — rejected: overlaps the parallel T-1543 fix, changes pointer-identity semantics for all callers, and adds cost to every render.
- Change `Transform` to return a new `Content` — rejected: breaking change to the exported `TransformableContent` interface within v2.
- Seal during `Build()` — rejected: requires walking contents in `Build()` (high conflict risk with T-1543) and leaves a mutation window between `AddContent` and `Build()`.

## Regression Test

**Test file:** `v2/transform_data_immutability_test.go`
**Test name:** `TestTransformCannotMutateBuiltDocument`

**What it verifies:**
- `Transform` on a table obtained from a built document returns an error and leaves the document's data unchanged (top-level, nested in `Section`, and inside a collapsible section).
- `Clone()` of a document table remains transformable without affecting the document.
- Standalone (unattached) tables remain transformable.
- A transformation function that mutates its input and then fails cannot leak partial mutations into the table.
- `TestAddContentTypedNilDoesNotPanic`: `Builder.AddContent` with a typed-nil `*TableContent`/`*SectionContent`/`*DefaultCollapsibleSection` (including one nested in a collapsible section) does not panic in `sealContents` — matching pre-fix behaviour, where the typed nil was stored untouched.

**Run command:** `go test -run 'TestTransformCannotMutateBuiltDocument|TestAddContentTypedNilDoesNotPanic' ./...` (from `v2/`)

## Affected Files

| File | Change |
|------|--------|
| `v2/content.go` | `sealed` flag + `seal()` on `TableContent`; `Clone()` documents unsealed copies |
| `v2/transform_data.go` | Sealed check and records deep-copy in `Transform`; `sealContents` helper |
| `v2/document.go` | `Builder.AddContent` seals attached content |
| `v2/transform_data_immutability_test.go` | Regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed no non-test callers of `(*TableContent).Transform` exist, so sealing introduces no internal behaviour change.

## Prevention

**Recommendations to avoid similar bugs:**
- When adding public mutating methods to content types, gate them on attachment state or require operating on clones — documents share content pointers.
- Prefer the established clone-then-transform pattern (`Clone()` + operations) over in-place mutation for any pipeline work.
- When documenting an immutability guarantee, add a regression test that attempts every public mutation path against a built document.

## Related

- Transit T-1677 (this fix)
- Transit T-1543 — built documents expose mutable section contents (parallel fix, section path)
- Transit T-1689 — builder silently dropped content / post-Build mutations (same enforcement pattern)
- `v2/docs/GETTING-STARTED.md`, `v2/docs/API.md` — immutability contract
