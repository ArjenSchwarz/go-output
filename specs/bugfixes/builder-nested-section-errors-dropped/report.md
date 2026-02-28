# Bugfix Report: Builder Nested Section Errors Are Dropped

**Date:** 2026-02-28
**Status:** Fixed

## Description of the Issue

`Builder.Section` and `Builder.CollapsibleSection` executed nested builder callbacks but did not propagate nested builder errors to the parent builder.

**Reproduction steps:**
1. Create a parent builder with `New()`.
2. Add a `Section` or `CollapsibleSection` and inside the callback call `Table` with invalid data and `Raw` with invalid format.
3. Observe `builder.HasErrors()` returns `false` and `builder.Errors()` is empty.

**Impact:** Error reporting was incomplete for nested builder operations, so callers could miss table/raw creation failures inside sections.

## Investigation Summary

A structured inspection of builder flow showed parent and nested builders used separate `errors` slices.

- **Symptoms examined:** Parent builder reported no errors after invalid nested table/raw creation.
- **Code inspected:** `v2/document.go` (`Section`, `CollapsibleSection`, `Table`, `Raw`), plus section builder tests.
- **Hypotheses tested:**
  - Nested operations were not erroring (ruled out by existing `Table`/`Raw` behavior).
  - Parent builder was overwriting errors (ruled out).
  - Nested errors were never merged into parent (confirmed).

## Discovered Root Cause

Nested builders collected errors locally, but `Section` and `CollapsibleSection` only transferred nested content and ignored `subBuilder.Errors()`.

**Defect type:** Error handling omission.

**Why it occurred:**
- `Table`/`Raw` append errors to the current builder.
- Section methods instantiate a new sub-builder.
- Section methods call `Build()` and copy contents only.
- No merge step exists for sub-builder errors.
- Parent `HasErrors()` remains false even when nested failures occur.

**Contributing factors:** Sub-builder pattern separated content assembly from error aggregation without a shared helper for recursive error propagation.

## Resolution for the Issue

**Changes made:**
- `v2/document.go:151` - In `Section`, merged `subBuilder.Errors()` into parent `b.errors` under lock before returning.
- `v2/document.go:218` - In `CollapsibleSection`, merged `subBuilder.Errors()` into parent `b.errors` under lock before returning.

**Approach rationale:** Minimal fix in the two affected entry points preserves existing API and behavior while making nested errors visible to callers.

**Alternatives considered:**
- Shared recursive error tree/state across builders - not chosen due to broader API/design impact.

## Regression Test

**Test file:** `v2/builder_section_test.go`
**Test name:** `TestBuilder_NestedSectionErrorsArePropagated`

**What it verifies:** Both `Section` and `CollapsibleSection` propagate nested `Table`/`Raw` creation errors to the parent builder.

**Run command:** `cd v2 && go test ./... -run TestBuilder_NestedSectionErrorsArePropagated`

## Affected Files

| File | Change |
|------|--------|
| `v2/document.go` | Propagate nested sub-builder errors to parent in section builders. |
| `v2/builder_section_test.go` | Added regression test covering both section builder variants. |
| `specs/bugfixes/builder-nested-section-errors-dropped/report.md` | Added bugfix analysis and verification report. |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- Confirmed regression test failed before fix and passed after fix.

## Prevention

**Recommendations to avoid similar bugs:**
- When introducing sub-builders, explicitly merge both content and error state.
- Add regression tests for parent-observable behavior of nested builder callbacks.
- Prefer shared helper(s) for sub-builder finalization to avoid inconsistent propagation logic.

## Related

- Transit ticket `T-269`
