# Bugfix Report: DefaultCollapsibleValue Aliases Caller-Owned Hint Map

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

`DefaultCollapsibleValue` in `v2/collapsible.go` aliased caller-owned mutable
data through its format-hints map. The `WithFormatHint` option stored the
caller's `hints` map directly, and the `FormatHint` accessor returned the
internal map directly. As a result, external mutation of the map passed to
`WithFormatHint`, or of the map returned by `FormatHint`, changed renderer
behavior after the value had been constructed. This violates the v2 design
principle that values become effectively immutable after construction.

**Reproduction steps:**
1. Build a value: `cv := NewCollapsibleValue("s", "d", WithFormatHint("html", hints))`.
2. Mutate the caller map after construction (`hints["class"] = "mutated"`), or
   mutate the map returned by `cv.FormatHint("html")`.
3. Observe that a subsequent `cv.FormatHint("html")` reflects the external
   mutation instead of the originally configured hints.

**Impact:** Medium. Renderer hints could silently change after construction,
producing non-deterministic output. Scope limited to consumers that use
collapsible-value format hints.

## Investigation Summary

- **Symptoms examined:** Stored hints reflecting external mutation of both the
  input map and the returned map.
- **Code inspected:** `v2/collapsible.go` (`WithFormatHint`, `FormatHint`),
  compared against the already-fixed `v2/collapsible_section.go` (T-1317).
- **Hypotheses tested:** Confirmed T-1233 only added `sync.Once` for the details
  lazy cache and did not touch hint-map aliasing; confirmed T-1317 fixed the
  equivalent issue in `DefaultCollapsibleSection` but not in
  `DefaultCollapsibleValue`.

## Discovered Root Cause

`WithFormatHint` assigned the caller's map by reference
(`cv.formatHints[format] = hints`) and `FormatHint` returned the stored map by
reference. Maps in Go are reference types, so no defensive copy meant the
internal state and external callers shared the same backing map.

**Defect type:** Mutable shared state / missing defensive copy (immutability violation).

**Why it occurred:** The collapsible value implementation predates the T-1317
fix for the section type, and the defensive-copy pattern was not back-applied to
the value type.

**Contributing factors:** Two near-identical types (`DefaultCollapsibleValue`
and `DefaultCollapsibleSection`) with separate hint-handling code; fixing one
did not automatically fix the other.

## Resolution for the Issue

**Changes made:**
- `v2/collapsible.go` - Added `maps` import.
- `v2/collapsible.go` `WithFormatHint` - Store a defensive copy via `maps.Clone(hints)`.
- `v2/collapsible.go` `FormatHint` - Return a defensive copy via `maps.Clone(hints)`.

**Approach rationale:** `maps.Clone` matches the established T-1317 pattern in
`collapsible_section.go`, keeping the two types consistent. `maps.Clone(nil)`
returns `nil`, so the no-hints path is unchanged and existing behavior for
unset formats is preserved.

**Alternatives considered:**
- Manual `make` + `maps.Copy` loop - More verbose with no benefit over
  `maps.Clone`, and inconsistent with the section fix.

## Regression Test

**Test file:** `v2/collapsible_test.go`
**Test names:** `TestCollapsibleValueFormatHintInputImmutability`,
`TestCollapsibleValueFormatHintOutputImmutability`

**What it verifies:** Mutating the caller-provided hints map after construction,
and mutating the map returned by `FormatHint`, both leave the value's stored
hints unchanged.

**Run command:**
`go test -run 'TestCollapsibleValueFormatHint(Input|Output)Immutability' ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/collapsible.go` | Defensive copy of hints map on input and output via `maps.Clone` |
| `v2/collapsible_test.go` | Added input/output immutability regression tests |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`make test`)
- [x] Linters pass (`make lint`, 0 issues)

**Manual verification:**
- Confirmed the new tests fail before the fix and pass after.

## Prevention

**Recommendations to avoid similar bugs:**
- When introducing reference-type fields (maps, slices) configured by callers,
  copy defensively on both input and output to preserve immutability.
- When fixing an aliasing bug in one type, check for sibling types with the same
  pattern (here, `DefaultCollapsibleValue` vs `DefaultCollapsibleSection`).

## Related

- T-1359 (this fix)
- T-1317 - Equivalent fix for `DefaultCollapsibleSection` (the pattern matched here)
- T-1233 - Added `sync.Once` for the details lazy cache (did not address aliasing)
