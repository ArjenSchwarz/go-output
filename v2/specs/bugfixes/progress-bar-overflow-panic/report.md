# Bugfix Report: progress-bar-overflow-panic

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

`textProgress.renderDefault` rendered the progress bar by computing
`filled := int(progress * float64(barWidth))` and then
`strings.Repeat(" ", barWidth-filled)`. When the current count exceeded the
total, `filled` exceeded `barWidth`, making `barWidth-filled` negative.
`strings.Repeat` panics on a negative count, crashing any program that drove
the progress past its total.

A separate trigger of the same panic: configuring a negative bar width via
`WithWidth(-1)` or `WithTrackerLength(-1)` made `barWidth` itself negative, so
even `filled = 0` produced `strings.Repeat(" ", -1)`.

**Reproduction steps:**
1. `p := NewProgress(WithWidth(10))`
2. `p.SetTotal(10)` then `p.SetCurrent(11)` (or `p.Increment(11)`)
3. Observe `panic: strings: negative Repeat count` inside `renderDefault`.

Alternatively:
1. `p := NewProgress(WithWidth(-1))` (or `WithTrackerLength(-1)`)
2. `p.SetTotal(10)`; `p.SetCurrent(5)`
3. Observe the same panic.

**Impact:** Runtime panic (crash) in any caller using the text progress
indicator that overruns the configured total or supplies a non-positive width.
Scope is limited to `textProgress`; the no-op and pretty progress backends are
not affected.

## Investigation Summary

- **Symptoms examined:** `panic: strings: negative Repeat count` originating
  from `progress_text.go` during a `draw()` call.
- **Code inspected:** `v2/progress_text.go` (`renderDefault` bar math) and
  `v2/progress.go` (`WithWidth`, `WithTrackerLength`, `ProgressConfig.Width`).
- **Hypotheses tested:** Confirmed two independent inputs reach the unguarded
  subtraction — `current > total` (filled > barWidth) and a negative configured
  `Width` (barWidth < 0). Both were reproduced with failing tests.

## Discovered Root Cause

The bar math performed `strings.Repeat(" ", barWidth-filled)` without bounding
either operand. Two unbounded inputs could drive the count negative:
`filled` could exceed `barWidth` (overrun), and `barWidth` could be negative
(bad config).

**Defect type:** Missing input validation / unbounded arithmetic.

**Why it occurred:** The renderer assumed `0 <= current <= total` and a
positive `Width`, but the public API (`SetCurrent`, `Increment`, `WithWidth`,
`WithTrackerLength`) does not enforce those invariants.

**Contributing factors:** Existing tests covered only in-range values, so the
overrun and negative-width paths were never exercised.

## Resolution for the Issue

**Changes made:**
- `v2/progress_text.go:137-150` - In `renderDefault`, clamp `barWidth` to a
  minimum of 0 and clamp `filled` to the `[0, barWidth]` range before calling
  `strings.Repeat`. On overrun the bar renders full; a non-positive width
  renders an empty `[]` bar without panicking.

**Approach rationale:** Clamping at the render site is the minimal, localized
fix. It keeps the public API permissive (callers may still pass any value) while
guaranteeing the renderer never produces a negative repeat count. Showing a full
bar on overrun matches user expectations (100% is the visual ceiling).

**Alternatives considered:**
- Clamp `current`/`total` in the setters (`SetCurrent`, `Increment`) - Rejected:
  more invasive, changes observable state (the `(11/10)` count display), and
  would not cover the negative-width trigger.
- Validate `Width` in `WithWidth`/`WithTrackerLength` - Rejected: only addresses
  one of the two triggers and silently alters caller-supplied config; the render
  guard is needed regardless.

## Regression Test

**Test file:** `v2/progress_core_test.go`
**Test names:** `TestTextProgress_OverrunDoesNotPanic`,
`TestTextProgress_NegativeWidthDoesNotPanic`

**What it verifies:**
- `SetCurrent` and `Increment` beyond the total do not panic and render a full
  bar (`[==========]`).
- `WithWidth(-1)`, `WithTrackerLength(-1)`, and `WithWidth(0)` do not panic
  during draw/complete.

**Run command:**
`go test -run 'TestTextProgress_OverrunDoesNotPanic|TestTextProgress_NegativeWidthDoesNotPanic' .`

## Affected Files

| File | Change |
|------|--------|
| `v2/progress_text.go` | Clamp `barWidth` (>= 0) and `filled` ([0, barWidth]) in `renderDefault` |
| `v2/progress_core_test.go` | Added overrun and negative-width regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Linters/validators pass (`make lint`, 0 issues)

**Manual verification:**
- Confirmed the new tests panic on the pre-fix code (red) and pass after the
  fix (green).

## Prevention

**Recommendations to avoid similar bugs:**
- Guard any `strings.Repeat`/slice arithmetic that derives a count from external
  inputs; clamp before use.
- When adding configuration options that feed into sizing math, cover boundary
  and invalid values (negative, zero, over-max) in tests.

## Related

- Transit ticket T-1346
- Note: `progress_text.go` was also touched by T-1254 (SetContext); this fix is
  isolated to the bar math in `renderDefault`.
