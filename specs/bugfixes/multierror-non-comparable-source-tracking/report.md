# Bugfix Report: MultiError Source Tracking Panics on Non-Comparable Errors

**Date:** 2026-08-03
**Status:** Fixed
**Ticket:** T-1508

## Description of the Issue

`MultiError` tracks per-error source metadata in the exported field `SourceMap map[error]ErrorSource`. Go panics with `hash of unhashable type` when an interface value whose dynamic type is not comparable is used as a map key. Any error implemented as a value-receiver struct containing a slice, map, or function field triggers this.

Two code paths panic:

1. `AddWithSource` (`v2/errors.go:327`) inserts the error as a map key.
2. `Error()` (`v2/errors.go:286`) looks the error up in `SourceMap`. Since Go 1.12 map lookups hash the key even when the map is empty, and `NewMultiError` always initializes `SourceMap`, formatting panics even for errors added via plain `Add()`.

This is public API surface (`NewMultiError(...).AddWithSource(...)`) and also affects render error aggregation: `Output.Render` calls `AddWithSource` (`v2/output.go:345`) with whatever error a custom renderer, transformer, or writer returned.

**Reproduction steps:**
1. Define a value-receiver error type with a slice field, e.g. `type sliceFieldError struct { parts []string }` with `func (e sliceFieldError) Error() string`.
2. `multiErr := NewMultiError("render"); multiErr.AddWithSource(sliceFieldError{parts: []string{"boom"}}, "renderer", nil)` — panics.
3. Alternatively `multiErr.Add(sliceFieldError{...}); multiErr.Add(errors.New("x")); _ = multiErr.Error()` — panics in `Error()`.

**Impact:** Library consumers aggregating errors, and any `Output.Render` call where a custom component returns a non-comparable error — a valid `error` value crashes the process instead of being reported.

## Investigation Summary

Followed the systematic debugging methodology (Fagan inspection phases).

- **Symptoms examined:** `panic: hash of unhashable type` from both `AddWithSource` (insert) and `Error()` (lookup), reproduced by the regression tests before the fix.
- **Code inspected:** `v2/errors.go` (`MultiError`, `AddWithSource`, `Error`, `NewMultiError`, `CollectErrors`), all `SourceMap`/`AddWithSource` usages: one production call site (`v2/output.go:345`) and one test reading `SourceMap` (`v2/errors_core_test.go:326`). No external-facing iteration of `SourceMap` exists in the repo.
- **Hypotheses tested:**
  - Guard with `reflect.TypeOf(err).Comparable()` only — ruled out as incomplete: a statically comparable struct with an interface field holding a slice still panics when hashed. `reflect.Value.Comparable()` (Go 1.20+) checks the dynamic value and is the precise guard.
  - Only the insert path panics — ruled out: the `Error()` lookup at line 286 hashes the key too, so plain `Add()` followed by `Error()` also panics (second defect, confirmed by a dedicated regression test).

## Discovered Root Cause

Source metadata is keyed by error identity in a `map[error]ErrorSource`. The design assumed all error values are comparable — true for the library's own pointer-receiver error types, but not guaranteed by the language for arbitrary user errors. Using an interface value with a non-comparable dynamic type as a map key is a runtime panic in Go.

**Defect type:** Missing validation / unsound data-structure key choice (interface map key without comparability guarantee).

**Why it occurred:** The library's own error types (`*RenderError`, `*WriterError`, etc.) are pointers and therefore always comparable, so the panic never surfaced in internal usage. The public API accepts any `error`, where comparability is not guaranteed.

**Contributing factors:** Map lookups hashing keys even on empty maps (Go 1.12+) widened the blast radius from `AddWithSource` to any `Error()` call on a `MultiError` with a non-nil `SourceMap`, which `NewMultiError` always creates.

## Resolution for the Issue

**Changes made:**
- `v2/errors.go` — `MultiError` gains an unexported `sources map[int]ErrorSource` keyed by index into `Errors`. `AddWithSource` records the source there (never panics) and additionally populates the exported `SourceMap` only when the error value is hashable. `Error()` resolves sources through a new `sourceOf` helper: positional entry first, then a hashability-guarded `SourceMap` lookup as fallback for entries callers placed into the field directly. Hashability is checked with `reflect.Value.Comparable()`, which inspects the dynamic value and therefore also rejects statically comparable structs holding non-comparable interface payloads.
- `v2/CHANGELOG.md` — entry under Unreleased/Fixed.

**Approach rationale:** Positional tracking removes error values from the map-key position entirely on the write path the library controls, fixing both panic sites. Keeping `SourceMap` populated for hashable errors preserves the exported surface — the in-repo reader (`errors_core_test.go`) and any external readers keep working — without a breaking change. As a side effect, duplicate error values added twice now retain their individual sources (previously the map overwrote the first).

**Alternatives considered:**
- Change `SourceMap` to an index-aligned exported slice — rejected: breaking change to a public field, not justified when the surface can be preserved.
- Guard with `reflect.TypeOf(err).Comparable()` and drop source info for non-comparable errors — rejected: type-level check misses dynamically unhashable values, and silently losing source metadata contradicts the point of `AddWithSource`.
- Wrap non-comparable errors in a comparable pointer wrapper before storing — rejected: changes the identity of values in `Errors`, observable via `errors.As`/type assertions.

## Regression Test

**Test file:** `v2/multierror_source_tracking_test.go`
**Test names:** `TestMultiErrorAddWithSourceNonComparable`, `TestMultiErrorErrorFormattingNonComparableViaAdd`, `TestMultiErrorSourceMapCompatibility`

**What it verifies:** `AddWithSource` accepts errors with slice fields, map fields, and statically-comparable-but-dynamically-unhashable payloads without panicking, and their source info appears in `Error()` output; `Error()` does not panic for non-comparable errors added via plain `Add()`; the exported `SourceMap` still receives entries for comparable errors and directly populated entries are still rendered.

**Run command:** `cd v2 && go test -run 'TestMultiError' .`

## Affected Files

| File | Change |
|------|--------|
| `v2/errors.go` | Positional source tracking; hashability guard on `SourceMap` insert and lookup |
| `v2/multierror_source_tracking_test.go` | New regression tests |
| `v2/CHANGELOG.md` | Unreleased/Fixed entry |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed both panic sites (insert at `errors.go:327`, lookup at `errors.go:286`) reproduce before the fix via the regression tests (red), and pass after (green).

## Prevention

**Recommendations to avoid similar bugs:**
- Never key a map by an interface type unless the API guarantees comparability; prefer positional or ID-based keys for user-supplied values.
- When a comparability guard is unavoidable, use `reflect.Value.Comparable()` (dynamic check), not `reflect.Type.Comparable()` (static check).

## Related

- Transit ticket T-1508
- Related but separate: T-1515 (AsError/IsCancelled semantics) — explicitly out of scope here.
