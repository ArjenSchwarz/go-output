# Bugfix Report: prettyprogress-handle-signals-nil-context

**Date:** 2026-02-28
**Status:** Fixed

## Description of the Issue

`NewPrettyProgress` starts `handleSignals` immediately, but `handleSignals` selected on `p.ctx.Done()` even when `SetContext` had not been called yet. That caused a nil pointer dereference panic at startup.

**Reproduction steps:**
1. Create a `prettyProgress` instance with `ctx == nil`.
2. Start `handleSignals`.
3. Observe panic: `invalid memory address or nil pointer dereference`.

**Impact:** Pretty progress startup could panic before any work is done when context was not initialized yet.

## Investigation Summary

Systematic inspection of pretty progress startup and signal handling found an unconditional dereference in the signal loop.

- **Symptoms examined:** Panic in `handleSignals` before `SetContext`.
- **Code inspected:** `v2/progress_pretty.go`, `v2/progress_core_test.go`.
- **Hypotheses tested:** Uninitialized `p.ctx` used in `select`; confirmed by a focused failing regression test.

## Discovered Root Cause

`handleSignals` always evaluated `p.ctx.Done()` without checking whether `p.ctx` was nil.

**Defect type:** Missing nil guard / nil pointer dereference.

**Why it occurred:** Signal handling goroutine starts during constructor setup, but context initialization was deferred to optional `SetContext`.

**Contributing factors:** Existing pretty progress tests often run with non-TTY fallback, so this startup path was not explicitly regression-tested.

## Resolution for the Issue

**Changes made:**
- `v2/progress_pretty.go:48-56` - Initialize a default cancelable background context in `NewPrettyProgress`.
- `v2/progress_pretty.go:85-104` - Handle closed signal channels and use `contextDone()` to safely return nil channel when context is nil.
- `v2/progress_core_test.go:793-820` - Added regression test for nil-context `handleSignals` startup behavior.

**Approach rationale:** Initialize context at construction and make signal handling nil-safe so startup cannot panic even if context is absent.

**Alternatives considered:**
- Guarding only constructor context initialization - not chosen alone because direct nil-context handling remained brittle without explicit guard.

## Regression Test

**Test file:** `v2/progress_core_test.go`
**Test name:** `TestPrettyProgress_HandleSignals_NilContext_DoesNotPanic`

**What it verifies:** `handleSignals` does not panic when `p.ctx` is nil and exits cleanly when signal channel closes.

**Run command:** `cd v2 && go test ./... -run TestPrettyProgress_HandleSignals_NilContext_DoesNotPanic -count=1`

## Affected Files

| File | Change |
|------|--------|
| `v2/progress_pretty.go` | Added safe default context initialization and nil-safe signal/context handling |
| `v2/progress_core_test.go` | Added regression test reproducing the nil-context panic scenario |
| `specs/bugfixes/prettyprogress-handle-signals-nil-context/report.md` | Added bugfix investigation and verification report |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed regression test failed before fix with nil pointer panic.
- Confirmed regression test passes after fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Initialize lifecycle contexts at constructor boundaries when goroutines depend on them.
- Use helper methods that safely return nil channels for optional contexts in `select` statements.
- Add targeted tests for startup goroutines that run before optional configuration methods are called.

## Related

- Transit ticket: `T-251`
