# Bugfix Report: Operation Constructor Input Ownership

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1467 (operation-constructor portion of T-2322)

## Description of the Issue

`NewSortOp`, `NewGroupByOp` and `NewAddColumnOp` stored caller-owned slices, maps and pointers directly. Operations run later, during rendering, so changing those inputs after construction changed the operation's behaviour, and changing them during a render was a data race.

**Reproduction steps:**
1. `keys := []SortKey{{Column: "id", Direction: Ascending}}; op := NewSortOp(keys...)`
2. `keys[0].Direction = Descending`
3. Apply `op`: rows come out descending.

Same shape for the `groupBy` slice and `aggregates` map of `NewGroupByOp` and the `position` pointer of `NewAddColumnOp`.

**Impact:** Medium. Reusing a configuration slice/map/int to build several reports silently altered earlier operations.

## Investigation Summary

- **Code inspected:** constructors and `Apply`/`Validate`/`evolveSchema` in `v2/operations.go`.
- **Findings:** All later reads go through `o.keys`, `o.groupBy`, `o.aggregates`, `o.position`; none of them were owned by the operation. `FilterOp`/`LimitOp` hold only a function/int and are unaffected.

## Discovered Root Cause

**Defect type:** Aliasing of caller-owned mutable state.

**Why it occurred:** Constructors predate the defensive-copy convention the codebase later adopted for options and content (T-1208, T-1378).

## Resolution for the Issue

**Changes made:**
- `v2/operations.go` `NewSortOp` — `slices.Clone(keys)`.
- `v2/operations.go` `NewGroupByOp` — `slices.Clone(groupBy)`, `maps.Clone(aggregates)`.
- `v2/operations.go` `NewAddColumnOp` — copies `*position` into operation-owned storage when non-nil.
- Godoc on each constructor states the inputs are copied.

**Approach rationale:** Smallest change; nil inputs stay nil (`slices.Clone`/`maps.Clone` preserve nil), so `Validate` errors are unchanged. Aggregate functions and the add-column function are shared, as functions are immutable values.

## Regression Test

**Test file:** `v2/operations_constructor_ownership_test.go`
**Test names:** `TestNewSortOpCopiesKeys`, `TestNewGroupByOpCopiesInputs`, `TestNewAddColumnOpCopiesPosition`

**What it verifies:** Mutating the original `[]SortKey`, `[]string`, aggregates map and position int after construction does not change the applied result.

**Run command:** `cd v2 && go test -run 'TestNew(SortOp|GroupByOp|AddColumnOp)Copies' .`

## Affected Files

| File | Change |
|------|--------|
| `v2/operations.go` | Defensive copies in constructors |
| `v2/operations_constructor_ownership_test.go` | Regression tests |
| `CHANGELOG.md` | Entry |

## Verification

**Automated:**
- [x] Regression tests pass (all three failed before the fix)
- [x] Full test suite passes, except two pre-existing `FileWriter` permission tests that also fail on main when run as root
- [x] Linters/validators pass (`make lint`: 0 issues)

## Prevention

- Constructors and options that retain slices, maps or pointers should clone them.

## Related

- T-2322 (broader input-ownership ticket; RendererConfig portion owned by another group)
