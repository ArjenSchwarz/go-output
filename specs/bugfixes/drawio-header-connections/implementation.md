# Implementation Explanation: DrawIOContent Clones Header Connections (T-1371)

Branch: `T-1371/bugfix-drawio-header-connections` vs `origin/main`

## Beginner Level

### What Changed

The go-output library can describe a diagram for Draw.io: a list of records
(the nodes) plus a `DrawIOHeader` that tells Draw.io how to draw them. One
part of that header, `Connections`, is a list of "draw an arrow from column A
to column B" rules.

When you handed a header to `NewDrawIOContent` or `NewDrawIOContentFromTable`,
the content kept the header — but it kept the *same* connection list you still
held, not a copy. The same was true the other way round: `GetHeader()` handed
back the content's own list. So if you changed a connection afterwards, even
by accident, the diagram rendered later changed too, with no warning.

The fix adds one small helper, `cloneDrawIOHeader`, that copies the connection
list, and uses it everywhere a header crosses into or out of the content:
both constructors, `GetHeader`, and `Clone`. Two regression tests lock the
behaviour in.

### Why It Matters

A `DrawIOContent` is supposed to be frozen once it is built; that is what
lets the library render it safely, including from several goroutines at once.
Sharing the list broke that promise quietly: nothing crashed, the `# connect:`
line in the Draw.io output (and the `header` field in JSON/YAML output) just
silently followed whatever the caller did to its own variable. An earlier fix
(T-1295) had already copied the records and edges for exactly this reason but
missed the list hiding inside the header.

### Key Concepts

- **Slice**: Go's list type. A slice value is a small pointer-plus-length
  handle to an underlying array. Copying the handle does not copy the array,
  so two slices can point at the same elements.
- **Struct copy**: assigning a struct copies every field, including slice
  handles — so a struct that contains a slice is only "half copied".
- **Defensive copy**: making your own copy of data at the boundary where it
  enters or leaves an object, so nobody outside can change it behind your
  back.

---

## Intermediate Level

### Changes Overview

- `v2/graph_content.go` — new unexported
  `cloneDrawIOHeader(header DrawIOHeader) DrawIOHeader`: takes the header by
  value, replaces `Connections` with `slices.Clone(header.Connections)`, and
  returns it. Its godoc records why a shallow slice clone is a full deep copy.
- `NewDrawIOContent` / `NewDrawIOContentFromTable` — store
  `cloneDrawIOHeader(header)` instead of the caller's header.
- `GetHeader` — returns `cloneDrawIOHeader(d.header)`.
- `Clone` — the inline copy (`slices.Clone` plus reassign) that previously
  lived only there is replaced by the helper.
- Godoc on both constructors and `GetHeader` now states the copying contract.
- `v2/graph_content_mutation_test.go` — two new tests appended (see below).
- `CHANGELOG.md` (Unreleased → Fixed) and `docs/agent-notes/drawio-csv.md`
  (one-line contract note).

No public signatures change.

### Implementation Approach

The correct copy already existed — inside `Clone`. The fix hoists it into a
helper and applies it at the three other boundaries so the four sites cannot
drift apart, mirroring how `cloneRecords`/`cloneEdges` from T-1295 are shared
between constructors, getters, and `Clone`.

The helper is deliberately shallow: `DrawIOConnection` is four strings and a
bool, and `Connections` is the only reference-typed field among
`DrawIOHeader`'s eighteen (the rest are `string` and `int`), so cloning the
one slice is a complete deep copy. `slices.Clone` returns nil for nil input,
so headers built without connections still marshal as `null` rather than
`[]` in JSON/YAML.

Tests: `TestDrawIOContent_HeaderConnectionsDefensivelyCopied` is a map-based
table test with four cases — input header mutated after `NewDrawIOContent`,
after `NewDrawIOContentFromTable`, `GetHeader()` result mutated, and the
clone's header mutated. Each case renders a baseline through
`drawioRenderer.renderDrawIOContent`, applies the mutation, then asserts both
that `GetHeader().Connections` still equals the original connection and that
a second render is byte-identical to the baseline.
`TestDrawIOContent_NilHeaderConnectionsStayNil` asserts nil survives both the
constructor and `Clone`. Before the fix the first three subtests fail; the
clone and nil cases pass and exist to guard the `Clone` refactor.

### Trade-offs

- **Copy at both boundaries vs. constructors only**: copying only on the way
  in would leave `GetHeader()` handing out the internal slice. Both sides are
  copied; the cost is one small slice allocation per constructor or
  `GetHeader` call, and the renderers call `GetHeader` once per content.
- **Copy vs. change the return type**: returning `*DrawIOHeader` or an
  unexported type from `GetHeader` would be a breaking API change for a
  problem a copy solves.
- **Copy vs. document "do not mutate"**: a doc comment enforces nothing and
  would contradict `Clone`'s existing behaviour and the T-1086/T-1295
  precedent.

---

## Expert Level

### Technical Deep Dive

`cloneDrawIOHeader` works as both an input and an output boundary because of
Go's by-value parameter semantics: the function receives its own copy of the
18-field struct, reassigns `Connections` on that copy, and returns it. The
caller's struct — whether that is user code passing a header in or `d.header`
being read out — is never written to. `slices.Clone` returns nil for a nil
input (the nil test pins this) and otherwise `append(S{}, s...)`, a fresh
backing array with no shared capacity, so appends on either side cannot
interfere.

The three `GetHeader()` consumers were checked: `renderDrawIOContent` →
`writeDrawIOHeader` in `v2/graph_renderers.go`, and the two sites in
`v2/json_yaml_renderer.go` that embed the header in the Draw.io output map
(multi-content and streaming paths). All read only, so the extra copy changes
no behaviour beyond severing the alias. `renderGraphAsDrawIO` and
`renderTableAsDrawIO` build their own local headers with fresh `Connections`
and never touch content state.

Test mechanics worth noting: the mutations run through a `*DrawIOHeader`
pointer to the test's local `input`, so the input-header cases mutate the
caller's variable after construction, exactly as user code would. The "clone
header mutated" case reaches into `clone.header.Connections` directly rather
than via `GetHeader()` — necessarily, since `GetHeader()` on the clone now
returns yet another copy and would exercise the getter, not `Clone`.
Assertions are at the rendered-bytes level (`renderDrawIOContent` on a bare
`drawioRenderer`), so the test observes the actual symptom, not just internal
state.

The race described in the report (caller mutating during render) is closed as
a consequence of the copy: the renderer's `GetHeader()` result no longer shares
a backing array with anything the caller holds.

### Architecture Impact

Public API surface unchanged; one unexported helper. `DrawIOContent` now
applies the same copy-in / copy-out discipline to every reference-carrying
field it stores (`records`, `columns`, `header.Connections`), completing the
T-1295 work. The helper sits directly under the `DrawIOConnection` definition
with a comment stating the invariant it depends on.

### Potential Issues

- The helper's correctness rests on an invariant nothing enforces: if a
  slice, map, or pointer field is ever added to `DrawIOHeader` or
  `DrawIOConnection`, the shallow clone silently reopens the leak. The godoc
  on `cloneDrawIOHeader` records the invariant, but no reflective test would
  fail when it breaks.
- `GetHeader()` now allocates on every call. Negligible for the renderers
  (once per content), but a caller polling it in a tight loop pays for a
  copy each time.
- The regression test drives `drawioRenderer.renderDrawIOContent` directly
  rather than `Output.Render`; the JSON/YAML header embedding is not asserted
  by a render test and relies on the same helper being correct.

---

## Completeness Assessment

**Fully implemented:** shared `cloneDrawIOHeader` helper applied at both
constructors, `GetHeader`, and `Clone`; nil preservation; godoc contract on
the three public entry points; regression tests covering all three previously
open mutation vectors plus the `Clone` refactor; CHANGELOG and agent-notes
entries; bugfix report aligned with the code.

**Partially implemented:** none.

**Missing:** nothing required by the ticket. Accepted gaps: no reflective
field guard for the shallow-clone invariant, and no JSON/YAML-level render
assertion for the header (the Draw.io render assertion plus the shared helper
cover the same code path).
