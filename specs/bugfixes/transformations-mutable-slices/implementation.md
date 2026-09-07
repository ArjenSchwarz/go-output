# Implementation Explanation: GetTransformations Exposes Mutable Operation Slices (T-1378)

Branch: `T-1378/bugfix-transformations-mutable-slices` (PR #129), two commits on top of `main` at `b20e9e0`.
Spec: `report.md` in this directory.

## Beginner Level

### What Changed

A go-output "document" is built once and then rendered as many times as you like (JSON, a terminal table, Markdown, and so on). Some pieces of a document carry a list of *transformations* — small operations such as "only keep active rows" or "sort by name" — that run every time the piece is rendered.

Before this change, asking a piece of content for its transformation list (`GetTransformations()`) handed you the library's own internal list, not a copy. If you replaced an entry in that list, you replaced it inside the document too. Three of the four ways to attach transformations when building content (`WithTextTransformations`, `WithRawTransformations`, `WithSectionTransformations`) had the mirror-image problem: they kept hold of the list you passed in, so changing your list afterwards changed the content.

Now every list that goes into a content item is copied on the way in, and every list handed back out is a copy on the way out. Empty slots (`nil` operations) are dropped during the copy, which the table option already did and the other three now match.

### Why It Matters

The library promises that a built document cannot change. That promise is what makes it safe to render the same document from several goroutines at once, or to render it twice and get the same bytes. The bug meant that promise could be broken silently — no method on the document was called, only a write into a list the library had handed out — and, if a render was in progress at the same time, it was also a data race.

### Key Concepts

- **Slice**: Go's list type. A slice is a small header pointing at a shared block of memory (the *backing array*). Handing someone a slice hands them a window onto the same memory, so writes through their window are visible through yours.
- **Defensive copy**: making a fresh block of memory with the same contents before handing a list out or storing one you received, so the two sides can no longer see each other's writes.
- **Immutable after `Build()`**: once `Build()` returns, nothing about the document should be changeable from outside.
- **Variadic argument (`ops ...Operation`)**: a function that takes "any number of operations". When you call it with `myList...`, Go passes `myList` itself, not a copy.

---

## Intermediate Level

### Changes Overview

- `v2/pipeline.go`: new unexported helper `cloneOperations(ops []Operation) []Operation`. It returns a fresh, non-nil slice holding the non-nil entries of its input.
- `v2/content.go`: `TableContent`, `TextContent`, `RawContent`, and `SectionContent.GetTransformations()` each become `return cloneOperations(x.transformations)`. The previous `nil -> []Operation{}` normalisation is preserved because the helper never returns nil. The `Content` interface godoc now states that implementations must not expose internal state through this method.
- `v2/text_options.go`, `v2/raw_options.go`, `v2/section_options.go`: the three option functions store `cloneOperations(ops)` instead of `ops`.
- `v2/table_options.go`: `WithTransformations` replaces its inline nil-filter loop (from T-1208) with the same helper; behaviour is unchanged.
- `v2/content_transformations_immutability_test.go`: five regression tests. Four iterate over a map of constructors, one per content type, so each assertion runs against all four types.
- `v2/docs/API.md`, `CHANGELOG.md`: copy semantics documented.

### Implementation Approach

The fix follows the copy-on-input / copy-on-output convention the package already uses for schema and key-order slices (T-1086, via `slices.Clone`). It could not reuse `slices.Clone` directly because the nil-filtering rule from T-1208 has to be applied at the same time, and `slices.DeleteFunc(slices.Clone(nil), ...)` would return nil, breaking the documented empty-non-nil contract. One helper on both boundaries means the four content types behave identically and there is a single place where the nil rule lives.

The only production consumer of `GetTransformations` is `applyContentTransformations` in `v2/renderer.go`, which calls it once, checks `len(...) == 0`, and iterates. It depends on neither slice identity nor nil-ness, so rendered output is unchanged.

The graph, chart, Draw.io, and collapsible-section types hold no transformation storage and return a `nil` literal; they are untouched, and the interface godoc allows nil for such types.

### Trade-offs

- **Allocation per render**: contents that carry transformations now pay one extra small allocation (16 bytes per operation) per render. Against the `Clone()` and `Apply` work in the same function this is negligible, and the zero-transformation case allocates nothing before or after (`make([]T, 0, 0)` returns the runtime's zero-base pointer).
- **Copy on input as well as output**: copying only in the accessor would have fixed the reported reproduction, but a caller-owned slice spread with `...` is a realistic way to build option lists dynamically, so the ticket named both paths.
- **Empty vs nil**: returning nil when absent would have been cheaper to express but changes the public contract for no benefit; existing table tests already lock in the empty-non-nil behaviour.

---

## Expert Level

### Technical Deep Dive

`cloneOperations` is a plain filter-append loop with `make([]Operation, 0, len(ops))`. It is called at seven sites: four accessors (egress) and four options (ingress; the table option already had the loop inline). The constructors assign the config field directly and every `Clone()` method already `make`s and `copy`s its own slice, so the total copy count along option -> config -> content -> Clone -> render is exactly the intended ingress/egress pair plus the pre-existing Clone copy; there is no accidental triple copy.

Pre-fix aliasing was real on both paths. Go passes a `...`-spread slice header unchanged, so text/raw/section content shared the caller's backing array; and every accessor returned the field's header. The tests prove this: `ReturnsDefensiveCopy` and `TransformationsCannotBeSwapped` were red for all four types, `CopyCallerSlice` and `DropNilOperations` red for text/raw/section and green for table (which already copied), `AbsentReturnsEmptyNonNil` green throughout.

### Architecture Impact

The `Content` interface contract is now explicit: `GetTransformations` must return a copy or nil. Third-party `Content` implementations are not forced to comply, and `applyContentTransformations` keeps its own `op == nil` guard for that reason. The four built-in stubs (`DefaultCollapsibleSection`, `GraphContent`, `ChartContent`, `DrawIOContent`) return nil while the four storage types return empty-non-nil; the godoc states both honestly. A caller comparing to `nil` or JSON-encoding the result sees the difference; the render path does not.

### Potential Issues

- **Typed nils are not filtered.** `cloneOperations` and `applyContentTransformations` both test `op == nil`. The package's convention for rejecting typed nils is `isNilValue` (`v2/errors.go:505`, eleven production callers, established by T-1649 for `TransformPipeline.Add`). A `(*FilterOp)(nil)` passed to any of the four options therefore survives the filter and panics on `Validate`/`Apply` during render. This gap predates the PR (the T-1208 loop had it too) but the PR makes `cloneOperations` the single choke point, so it is the natural place to close it.
- **API.md wording** says `GetTransformations()` "returns a copy (an empty, non-nil slice when no transformations are attached)"; this is true for the four storage types but not for the nil-returning stubs.
- **`WithTransformations` godoc** does not carry the copy/nil sentence its three siblings now have; the contract is only stated in a body comment godoc readers never see.
- **Clone independence** is correct by construction (make+copy) but no test can observe it any more through the accessor, since the accessor now copies. A test would have to compare `&slice[0]` on the unexported field.

---

## Completeness Assessment

**Fully implemented**
- Both aliasing paths named by the ticket are closed: accessor egress for all four storage types, option ingress for text/raw/section (table already copied).
- Nil filtering is uniform across the four options.
- Empty-non-nil contract preserved and locked by a test.
- Rendered output unchanged; only production consumer verified.
- Report, CHANGELOG, API.md, and interface godoc updated. Every factual claim in `report.md` was checked against the code and found accurate.
- Full unit suite (2617 tests) green; `golangci-lint` 0 issues; `go vet` and `gofmt` clean.

**Partially implemented**
- Nil filtering matches the T-1208 rule (`== nil`) rather than the later T-1649 convention (`isNilValue`), so typed nils still pass. Within the ticket's stated scope, but a one-line consistency gap now that filtering is centralised.
- Doc precision: API.md and the `WithTransformations` godoc as described above.

**Missing**
- Nothing required by the ticket is missing.
