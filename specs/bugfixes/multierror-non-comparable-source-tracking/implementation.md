# Implementation Explanation: MultiError Non-Comparable Source Tracking (T-1508)

Branch: `T-1508/bugfix-multierror-non-comparable` vs `origin/main`.

## Beginner Level

### What Changed
`MultiError` is the library's way of bundling several errors from one operation (for example, rendering a document to JSON, CSV, and HTML at once) into a single error. To remember *where* each error came from (renderer? writer?), it stored that information in a Go map that used the error itself as the lookup key.

Go maps have a rule: a key must be "hashable" — the language must be able to compare two keys for equality. Most error values are, but some are not: an error built from a struct containing a slice or map field cannot be hashed. Using one as a map key does not fail gracefully — the whole program crashes (a "panic").

The fix stops using errors as map keys internally. Instead, source information is stored by position: "the source for error number 2 is the renderer". A number is always hashable, so the crash is impossible.

### Why It Matters
Any application using this library could crash simply because one of its components returned a perfectly legal Go error whose internal shape happened to include a slice. Worse, the crash happened while *reporting* errors — exactly when you most need the program to stay up and tell you what went wrong.

### Key Concepts
- **Panic**: Go's program-crashing failure, like an uncaught exception.
- **Comparable/hashable**: a value Go can check for equality; required for map keys. Slices, maps, and functions are not comparable.
- **Interface value**: Go's `error` is an interface — a box holding any concrete type. The box's hashability depends on what is inside, which can only be known at runtime.

---

## Intermediate Level

### Changes Overview
- `v2/errors.go`: `MultiError` gains an unexported `sources map[int]ErrorSource` keyed by index into `Errors`. `AddWithSource` records the source positionally and additionally mirrors it into the exported `SourceMap` only when the error value is hashable. `Error()` resolves sources via a new `sourceOf(i, err)` helper: positional entry first, then a guarded `SourceMap` fallback for entries callers placed there directly. A new `isHashableError` helper wraps `reflect.ValueOf(err).IsValid() && v.Comparable()`.
- `v2/multierror_source_tracking_test.go`: regression tests for both panic sites, SourceMap compatibility, interleaved `Add`/`AddWithSource` alignment, and duplicate-error source retention.
- `CHANGELOG.md` and `specs/bugfixes/multierror-non-comparable-source-tracking/report.md`: documentation.

### Implementation Approach
Two panic sites existed, not one:
1. **Insert**: `AddWithSource` executed `e.SourceMap[err] = ...` — hashing the key.
2. **Lookup**: `Error()` executed `e.SourceMap[err]` — since Go 1.12, map lookups hash the key *even on an empty map*, and `NewMultiError` always allocates `SourceMap`. So a plain `Add()` followed by `Error()` also panicked.

The write path the library controls (via `AddWithSource`) never touches the error-keyed map for unhashable values; the read path (`Error()` → `sourceOf`) short-circuits with `len(e.SourceMap) == 0 || !isHashableError(err)` before any hash can occur.

The dynamic check `reflect.Value.Comparable()` (Go 1.20+) is deliberate: `reflect.Type.Comparable()` is a static check that reports `true` for a struct whose only field is an interface — but hashing such a value still panics if the interface holds a slice. The test type `dynamicPayloadError` exercises exactly this distinction.

### Trade-offs
- **Keep `SourceMap` exported and populated for hashable errors** vs. replacing it with a slice: preserves the public API surface at the cost of dual bookkeeping. External readers of `SourceMap` keep working, but see an incomplete view when non-comparable errors are present (documented on the field).
- **Positional map (`map[int]`) vs. index-aligned slice**: `Add()` appends to `Errors` without recording a source, so a parallel slice would force `Add` to insert placeholders. The sparse map keeps `Add` untouched.
- **Behaviour change as side effect**: the same error value added twice via `AddWithSource` now keeps both source entries in `Error()` output (positional entries are independent), where the old map key overwrote the first. Documented in the CHANGELOG and locked down by a test.

---

## Expert Level

### Technical Deep Dive
`sourceOf` resolution order matters. Positional entries win over `SourceMap`, so for errors added through `AddWithSource` the exported map is never consulted — reflection cost is confined to (a) one `reflect.ValueOf` per `AddWithSource` call and (b) the fallback path for directly-constructed `MultiError` literals. Both are error-path only; the single production call site is `Output.Render`'s aggregation loop (`v2/output.go:345`), which runs single-threaded after `wg.Wait()`, so no concurrency semantics change.

The `len(e.SourceMap) == 0` guard is not merely an optimisation: it is what makes the `NewMultiError` + `Add(nonComparable)` + `Error()` sequence safe without paying reflection on the common path — although `!isHashableError(err)` alone would also prevent the panic. `isHashableError`'s `IsValid()` guard handles a nil error placed by hand into `Errors` (`reflect.ValueOf(nil)` is the zero Value).

Rejected alternatives (from the bugfix report): type-level `reflect.Type.Comparable()` (unsound for dynamic payloads); wrapping non-comparable errors in a comparable pointer (changes identity observable via `errors.As`); exporting an index-aligned slice (breaking API change).

### Architecture Impact
The exported mutable surface (`Errors`, `SourceMap` as public fields) is the root constraint here — the fix works around it rather than breaking compatibility. Two consequences are consciously accepted and documented:
1. `SourceMap` is now a *partial* view; consumers iterating it under-report sources for non-comparable errors.
2. Positional tracking assumes `Errors` is append-only. A caller that splices or reorders `e.Errors` directly after `AddWithSource` gets silently misattributed sources (the old value-keyed map tolerated reordering). Documented on the `sources` field.

A future v3 could replace both exported fields with accessor methods and a single positional store.

### Potential Issues
- Callers mutating `Errors` directly (see above) — wrong output, not a panic; judged low likelihood.
- `AddWithSource` overwrites a caller-pre-populated `SourceMap` entry for the same hashable error value (last-wins in the map, while `Error()` renders positional entries). Corner case; `Error()` output is correct.
- `MultiError` remains non-goroutine-safe, unchanged from before; concurrent `AddWithSource` was already a data race on `SourceMap`.

---

## Completeness Assessment

**Fully implemented:**
- Both panic sites closed (insert in `AddWithSource`, lookup in `Error()`), verified by red→green regression tests.
- Exported `SourceMap` behaviour preserved for hashable errors, including directly-populated entries.
- Positional alignment under interleaved `Add`/`AddWithSource`, duplicate-error source retention — both tested.
- Documentation: CHANGELOG entry, doc comments on `SourceMap`/`sources`/`Add`/`AddWithSource`, bugfix report with accurate pre-fix line references.

**Partially implemented / accepted gaps:**
- `AddWithSource(nil, ...)` no-op is implemented but has no dedicated assertion (nit; the pre-existing `TestMultiError` covers `Add(nil)`).

**Missing:** nothing required by the ticket. T-1515 (AsError/IsCancelled semantics) is explicitly out of scope.
