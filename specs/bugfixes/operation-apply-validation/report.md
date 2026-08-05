# Bugfix Report: Direct Table Operation Apply Can Panic on Invalid Config

**Ticket:** T-1502
**Date:** 2026-08-03
**Status:** Fixed

## Description of the Issue

The five table operations in `v2/operations.go` (`FilterOp`, `SortOp`, `LimitOp`, `GroupByOp`, `AddColumnOp`) each expose a public `Apply` (and `ApplyWithFormat`) method. The renderer path validates operations before applying them (`applyContentTransformations` in `v2/renderer.go:204`), but the `Apply` methods are public API and callable directly. None of them checked their own `Validate` result, so direct calls with invalid configuration panicked or silently misbehaved:

- `NewFilterOp(nil).Apply(ctx, table)` — nil pointer panic at `o.predicate(record)`
- `NewLimitOp(-1).Apply(ctx, table)` — `slice bounds out of range [:-1]` panic
- `NewAddColumnOp("x", nil, nil).Apply(ctx, table)` — nil pointer panic at `o.fn(record)`
- `NewGroupByOp(...)` with a nil aggregate entry — nil pointer panic at `aggFunc(groupRecords, field)`
- `NewSortOp(SortKey{Direction: SortDirection(99)})` — silently sorts ascending instead of erroring
- `NewSortOp()` (no keys, no comparator), empty column names, empty groupBy columns, empty/negative addColumn config — silently accepted

**Reproduction steps:**
1. Build a document with a table containing at least one record
2. Call e.g. `NewFilterOp(nil).Apply(context.Background(), tableContent)` directly
3. Observe a runtime panic instead of a validation error

**Impact:** Any library consumer using operations directly (outside the renderer's transformation pipeline) gets runtime panics from invalid configuration instead of the structured validation errors the operations already know how to produce. Severity is moderate: no data corruption, but panics can crash consumer applications.

## Investigation Summary

- **Symptoms examined:** Panics (nil dereference, negative slice bound) and silent misbehaviour when calling `Apply` directly with invalid config; correct structured errors when the same operations run through the renderer pipeline.
- **Code inspected:** `v2/operations.go` (all five `Apply`/`ApplyWithFormat`/`Validate` implementations), `v2/renderer.go` (`applyContentTransformations`), existing regression tests from T-1142 (`v2/operations_nil_content_test.go`).
- **Hypotheses tested:** Confirmed via failing regression tests that every case in the ticket reproduces: four panic paths and the silent sort-direction path. Confirmed `ApplyWithFormat` delegates to `Apply` in all five operations, so wiring validation into `Apply` covers both entry points.

## Discovered Root Cause

Validation and application are decoupled: each operation implements `Validate()` with complete config checks, but only the renderer pipeline calls it. The public `Apply` methods assume configuration was validated by the caller and dereference it unconditionally.

**Defect type:** Missing validation (validate-before-apply wiring gap in public API)

**Why it occurred:** The `Operation` interface was designed for the renderer pipeline, where `applyContentTransformations` validates each operation before applying it. When the same methods were exposed as public API, the implicit "caller validates first" contract was not enforced inside `Apply`.

**Contributing factors:** T-1142 fixed the sibling issue (nil `Content` argument panics) by adding guards inside `Apply`, but invalid operation *configuration* was left to the pipeline's external `Validate` call.

## Resolution for the Issue

**Changes made:**
- `v2/operations.go` — each of the five `Apply` methods (`FilterOp.Apply`, `SortOp.Apply`, `LimitOp.Apply`, `GroupByOp.Apply`, `AddColumnOp.Apply`) now calls `o.Validate()` first and returns the structured validation error before touching content or configuration. `ApplyWithFormat` delegates to `Apply` in all five operations, so it inherits the check.

**Approach rationale:** The `Validate` methods already exist and return the exact structured errors the ticket expects; the fix is pure wiring. Placing the check at the top of `Apply` guarantees every entry point (direct `Apply`, direct `ApplyWithFormat`, and the renderer pipeline) enforces the same contract. The pipeline's pre-validation in `renderer.go` remains: its error wrapping adds content ID and operation index context that callers rely on, and the duplicate check is a few cheap field comparisons.

**Alternatives considered:**
- Ad-hoc guards inside each `Apply` (e.g. `if o.predicate == nil`) — rejected: duplicates logic already in `Validate`, would drift over time, and would not fix the silent invalid-sort-direction case without re-implementing that check as well.
- Removing the pipeline's `Validate` call now that `Apply` validates — rejected: the pipeline's wrapping (content ID, operation index) would be lost for validation failures, changing existing error messages for no benefit.

## Regression Test

**Test file:** `v2/operations_apply_validation_test.go`
**Test names:** `TestOperationsApplyInvalidConfig`, `TestOperationsApplyWithFormatInvalidConfig`, `TestOperationsApplyValidConfigStillWorks`

**What it verifies:** Direct `Apply` and `ApplyWithFormat` calls with invalid configuration (nil predicate, negative limit, nil calculation function, empty/negative addColumn config, nil aggregate function, empty groupBy columns, invalid sort direction, empty sort keys/columns) return the operation's structured validation error instead of panicking or silently succeeding. A companion test confirms valid configuration still applies successfully.

**Run command:** `cd v2 && go test -run 'TestOperationsApply(InvalidConfig|WithFormatInvalidConfig|ValidConfigStillWorks)' ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/operations.go` | Call `Validate()` at the top of each operation's `Apply` |
| `v2/operations_apply_validation_test.go` | New regression tests for direct Apply with invalid config |
| `CHANGELOG.md` | Fixed entry under Unreleased |

## Verification

**Automated:**
- [x] Regression test passes (`TestOperationsApplyInvalidConfig`, `TestOperationsApplyWithFormatInvalidConfig`, `TestOperationsApplyValidConfigStillWorks`)
- [x] Full test suite passes (`make test`)
- [x] Linters/validators pass (`make lint` — 0 issues; `gofmt` clean)

**Manual verification:**
- Confirmed pre-fix panics and silent behaviour via the failing regression tests (red phase): nil-pointer panics for filter/addColumn/groupBy, `slice bounds out of range [:-1]` for limit, silent ascending sort for the invalid direction

## Prevention

**Recommendations to avoid similar bugs:**
- When a type implements both `Validate()` and an action method that is public API, the action method should enforce `Validate()` itself rather than relying on callers.
- New `Operation` implementations should start `Apply` with a `Validate()` call; the regression test's map-based structure makes it easy to add cases for new operations.

## Related

- Transit ticket T-1502
- T-1142 — sibling fix for nil `Content` passed into `Apply` (`v2/operations_nil_content_test.go`)
- T-1465 — separate open ticket about SortOp and sparse tables (intentionally not addressed here)
