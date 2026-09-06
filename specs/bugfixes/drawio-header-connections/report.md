# Bugfix Report: DrawIOContent Exposes Mutable Header Connections

**Date:** 2026-09-06
**Status:** Fixed
**Ticket:** T-1371

## Description of the Issue

`DrawIOHeader` carries a `Connections []DrawIOConnection` slice. `NewDrawIOContent`
and `NewDrawIOContentFromTable` stored the header struct by value, and `GetHeader`
returned it by value. Go struct assignment copies the slice header (pointer,
length, capacity) but not its elements, so the caller's `Connections` and the
content's stored `Connections` shared one backing array. A caller could therefore
change later Draw.io output by mutating the header it had passed in, or by
mutating the header returned from `GetHeader()`, long after the content had been
created. The same aliased header is embedded in JSON and YAML output.

T-1295 fixed the equivalent leak for records and edges (`cloneRecords`,
`cloneEdges`) but left the header's nested slice untouched.

**Reproduction steps:**
1. Build a header with a connection: `header := DefaultDrawIOHeader(); header.Connections = []DrawIOConnection{{From: "Name", To: "Upstream"}}`
2. Create content: `content := NewDrawIOContent("infra", records, header)` (or `NewDrawIOContentFromTable(table, header)`).
3. Mutate the caller-owned header (`header.Connections[0].To = "mutated"`) or the getter result (`content.GetHeader().Connections[0].To = "mutated"`).
4. Render with the Draw.io renderer: the `# connect:` line now reflects the mutation.

**Impact:** Medium. Any caller that retains the header it passed in, or the value
returned by `GetHeader()`, can silently corrupt Draw.io connection directives
(and the JSON/YAML `header` field) after construction. Concurrent mutation
during rendering is additionally a data race, since the renderer reads the same
backing array. The corruption is non-obvious because it happens through aliasing
rather than an explicit API call.

## Investigation Summary

Followed the systematic debugging methodology (Fagan inspection phases).

- **Symptoms examined:** Mutating `header.Connections` after construction, or the
  slice returned by `GetHeader()`, changes the connection directive in
  subsequent Draw.io renders.
- **Code inspected:** `v2/graph_content.go` (`DrawIOHeader`, `DrawIOConnection`,
  `NewDrawIOContent`, `NewDrawIOContentFromTable`, `GetHeader`, `Clone`, and the
  existing `cloneRecords`/`cloneEdges` helpers from T-1295); the consumers of
  `GetHeader()` in `v2/graph_renderers.go` (`renderDrawIOContent` →
  `writeDrawIOHeader`) and `v2/json_yaml_renderer.go` (header embedded in the
  Draw.io output map on both the multi-content and streaming paths).
- **Hypotheses tested:**
  - `DrawIOConnection` might hold nested reference types requiring a deeper copy —
    ruled out: it holds only strings and a bool, and `Connections` is the only
    reference-typed field in `DrawIOHeader`, so a `slices.Clone` of `Connections`
    is a full deep copy.
  - A consumer might depend on receiving the internal slice (e.g. mutate it in
    place) — ruled out: all three `GetHeader()` call sites only read the header.
    `renderGraphAsDrawIO`/`renderTableAsDrawIO` assign a fresh `Connections`
    slice to their own local header and never touch content state.
  - Copying might change JSON/YAML output for headers without connections —
    ruled out: `slices.Clone` preserves nil (marshals as `null`) versus empty
    (marshals as `[]`).

## Discovered Root Cause

The defensive-copy convention established by T-1086 and T-1295 was applied per
parameter type rather than per reference-typed field. Slice parameters
(`records`, `edges`) were cloned, but `DrawIOHeader` is a struct parameter and
was treated as a plain value even though it wraps a slice. The correct copying
logic existed only inline in `Clone`, so there was no shared helper to apply at
the constructor and getter boundaries.

**Defect type:** Missing defensive copy / reference aliasing (immutability
violation).

**Why it occurred:** Five Whys — the renderer reads a backing array the caller
still holds; because the constructors assign the struct by value; because
T-1295 cloned the slice-typed parameters and did not look inside struct-typed
ones; because no `cloneDrawIOHeader` helper existed to make the header boundary
explicit; because the header-copy logic lived only inside `Clone`.

**Contributing factors:** `DrawIOHeader` is a public struct that users build
directly and commonly keep a reference to (see `examples/charts/main.go`),
making post-construction mutation an easy mistake.

## Resolution for the Issue

_To be filled in after the fix is implemented._

## Regression Test

**Test file:** `v2/graph_content_mutation_test.go`
**Test names:**
- `TestDrawIOContent_HeaderConnectionsDefensivelyCopied` (map-based table test with
  four cases: input header mutated after `NewDrawIOContent`, input header mutated
  after `NewDrawIOContentFromTable`, `GetHeader` result mutated, clone header
  mutated)
- `TestDrawIOContent_NilHeaderConnectionsStayNil`

**What it verifies:** After each mutation vector, `GetHeader().Connections` still
holds the original connection and the Draw.io render output is byte-identical to
the pre-mutation baseline. The nil test guards that copying preserves a nil
`Connections` slice so JSON/YAML output is unchanged.

**Pre-fix results (red):** the first three subtests fail (stored connection and
render output both show the mutation). The "clone header mutated" subtest and the
nil test pass before the fix; they guard the refactor of `Clone` onto the shared
helper.

**Run command:**
`cd v2 && go test -run 'TestDrawIOContent_HeaderConnectionsDefensivelyCopied|TestDrawIOContent_NilHeaderConnectionsStayNil' -v .`

## Affected Files

| File | Change |
|------|--------|
| `v2/graph_content.go` | _pending_ |
| `v2/graph_content_mutation_test.go` | New regression tests (added) |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- Confirmed the three aliasing subtests fail before the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Treat every reference-typed field crossing a content boundary as requiring a
  defensive copy, including slices nested inside struct-typed parameters — not
  just top-level slice parameters.
- Keep one clone helper per reference-carrying type and reuse it from
  constructors, getters, and `Clone` so the three cannot diverge.

## Related

- T-1295 (Graph and Draw.io content expose mutable caller-owned data) - same
  defect class; fixed records and edges but not the header.
- T-1086 (Schema Key Order Leaks Mutable Slices) - same defect class, schema layer.
