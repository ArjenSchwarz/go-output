# Bugfix Report: Graph and Draw.io Content Expose Mutable Caller-Owned Data

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

Several v2 graph/Draw.io content constructors and accessors kept or returned
caller-owned mutable data instead of defensive copies. This violated the
document/content immutability model: a caller could change later render output
by mutating data it had passed in, or by mutating data returned from a getter,
long after the content had been created.

**Reproduction steps:**
1. Create graph content with `NewGraphContent(title, edges)` (or Draw.io content
   with `NewDrawIOContent(title, records, header)` /
   `NewDrawIOContentFromTable(table, header)`).
2. Mutate the slice/maps that were passed in, the source table's records, or the
   slice/maps returned by `GetEdges()` / `GetRecords()`.
3. Observe that the content's stored data (and therefore its rendered output)
   reflects the mutation.

**Impact:** Medium. Any caller that retains a reference to an input slice/map,
the source table, or a getter result could silently corrupt graph/Draw.io
content after construction. The corruption is non-obvious because it happens
through aliasing rather than an explicit API call.

## Investigation Summary

- **Symptoms examined:** Aliasing of caller-owned slices and `Record` maps into
  `GraphContent`/`DrawIOContent` internals, and exposure of those internals to
  callers through getters.
- **Code inspected:** `v2/graph_content.go` (`NewGraphContent`, `GetEdges`,
  `NewDrawIOContent`, `NewDrawIOContentFromTable`, `GetRecords`, and the existing
  `Clone` methods which already performed correct deep copies).
- **Hypotheses tested:** Confirmed each leak point individually with targeted unit
  tests that mutate caller-owned data and assert the content's internal state is
  unaffected. Confirmed the in-package precedent (T-1086 schema fix and the
  existing `Clone` methods) establishes defensive copying as the convention.

## Discovered Root Cause

Slices and maps in Go are reference types. The constructors assigned the input
slices directly, and the getters returned the internal slices directly, so the
underlying backing array - and, for Draw.io, the inner `Record` maps - was shared
with the caller. `NewDrawIOContentFromTable` additionally aliased the source
table's `records` slice and maps.

**Defect type:** Missing defensive copy / reference aliasing (immutability
violation).

**Why it occurred:** The constructors and accessors were written to assign and
return slices/maps directly without copying. The existing `Clone` methods already
contained the correct copying logic, but that logic was not applied at the
construction and accessor boundaries.

**Contributing factors:** `Edge` is a value struct with no reference-typed fields,
so a shallow slice copy is sufficient for graph edges. `Record` is a map, so the
Draw.io path requires copying both the slice and each inner map - making the
Draw.io leak more impactful and easier to overlook.

## Resolution for the Issue

**Changes made:**
- `v2/graph_content.go` `NewGraphContent` - clones the input `edges` slice.
- `v2/graph_content.go` `GetEdges` - returns a copy of the internal `edges` slice.
- `v2/graph_content.go` `NewDrawIOContent` - deep-copies the input `records`
  (slice and inner maps).
- `v2/graph_content.go` `NewDrawIOContentFromTable` - deep-copies the source
  table's `records` (slice and inner maps) instead of aliasing them.
- `v2/graph_content.go` `GetRecords` - returns a deep copy of the internal
  `records` (slice and inner maps).
- Added private helpers `cloneEdges` and `cloneRecords` and reused them from the
  constructors, getters, and the existing `Clone` methods to keep the copy logic
  in one place.

**Approach rationale:** Defensive copying on both input and output is the minimal,
localized fix that restores the documented immutability without changing any
public signatures. Reusing shared helpers removes the duplicated copy logic that
already lived in the `Clone` methods.

**Alternatives considered:**
- Documenting the slices/maps as "do not mutate" instead of copying - Rejected
  because it does not enforce immutability and contradicts the existing `Clone`
  behaviour and the T-1086 precedent.
- Copying only on construction (not in getters) - Rejected because getters still
  leaked internal state, allowing post-construction mutation.

## Regression Test

**Test file:** `v2/graph_content_mutation_test.go`
**Test names:**
- `TestGraphContent_MutateInputEdgesAfterCreate`
- `TestGraphContent_MutateEdgesThroughGetter`
- `TestDrawIOContent_MutateInputRecordsAfterCreate`
- `TestDrawIOContent_MutateRecordsThroughGetter`
- `TestDrawIOContentFromTable_MutateSourceTableRecords`

**What it verifies:** Mutating caller-owned data after construction, mutating the
source table's records, and mutating data returned by `GetEdges`/`GetRecords` do
not change the content's stored data.

**Run command:**
`go test ./v2/ -run 'TestGraphContent_Mutate|TestDrawIOContent_Mutate|TestDrawIOContentFromTable_Mutate' -v`

## Affected Files

| File | Change |
|------|--------|
| `v2/graph_content.go` | Defensive copies in constructors/getters; shared clone helpers |
| `v2/graph_content_mutation_test.go` | New regression tests (added) |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed tests fail before the fix and pass after it.

## Prevention

**Recommendations to avoid similar bugs:**
- Treat slices and maps crossing a content/value boundary as requiring a
  defensive copy on both store and return.
- Reuse a single clone helper per reference-typed field so constructors,
  getters, and `Clone` cannot diverge.

## Related

- T-1086 (Schema Key Order Leaks Mutable Slices) - same defect class, schema layer.
- T-1304 (Draw.io from-table nil-guard) - separate ticket, out of scope here.
