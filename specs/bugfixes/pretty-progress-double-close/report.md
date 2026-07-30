# Bugfix Report: prettyProgress Close Panics When Called Twice

**Date:** 2026-07-12
**Status:** Fixed
**Ticket:** T-1427

## Description of the Issue

`prettyProgress.Close()` in `v2/progress_pretty.go` calls `close(p.signals)` unconditionally on every invocation. Closing an already-closed channel panics in Go, so the second `Close()` on the same TTY-backed `prettyProgress` crashes the program with `close of closed channel` instead of returning normally.

**Reproduction steps:**
1. Create a TTY-backed pretty progress indicator (stderr attached to a terminal), e.g. `p := NewPrettyProgress()`.
2. Call `p.Close()` — returns nil.
3. Call `p.Close()` again — panics with `close of closed channel`.

The double call happens naturally in real code: `Output.Close()` forwards to the same `Progress` instance (`v2/output.go:477`), so the idiomatic `defer out.Close()` combined with an explicit `out.Close()` — or any code path that shares one progress instance across two cleanup routes — triggers the panic.

**Impact:** Runtime panic (process crash) during cleanup for any consumer running in a terminal who closes an `Output` or `Progress` more than once. Non-TTY environments are unaffected because `NewPrettyProgress` falls back to `textProgress` there — which is also why CI never caught it.

## Investigation Summary

Followed the systematic debugging methodology (Fagan inspection phases).

- **Symptoms examined:** Second `Close()` panics at `v2/progress_pretty.go:225` (`close(p.signals)`); confirmed with a regression test that reproduces the panic without a TTY by constructing `prettyProgress` directly.
- **Code inspected:** `v2/progress_pretty.go` (`Close`, `handleSignals`, `contextDone`, `NewPrettyProgress`), `v2/progress_pretty_unix.go` / `v2/progress_pretty_windows.go` (`setupSignals`), `v2/progress_text.go` (`Close`), `v2/progress_noop.go` (`Close`), `v2/output.go` (`Output.Close` → `o.progress.Close()`).
- **Hypotheses tested:** Every statement in `Close()` was checked for repeat-safety. `p.cancel()` is idempotent (stdlib contract), `MarkAsDone` is guarded by `p.active`, `signal.Stop` is a documented no-op when already stopped, and `p.writer.Stop()` is unreachable on a second call once an idempotency guard exists. Only `close(p.signals)` panics on repetition. `textProgress.Close` and `noOpProgress.Close` were inspected and test-verified to be repeat-safe already — no equivalent panic path.

## Discovered Root Cause

`Close()` performs one-shot cleanup (closing the signal channel that terminates the `handleSignals` goroutine) but has no record of whether that cleanup already ran, so a second call re-executes it and panics.

**Defect type:** Missing idempotency guard on one-shot resource cleanup (state-management error).

**Why it occurred:** The struct tracks `active` ("progress finished"), but `Close()` must run cleanup even when `active` is already false (Close after `Complete()`/`Fail()`), so `active` cannot double as the guard. The code conflates the "progress finished" lifecycle with the "resources released" lifecycle and never introduced a separate closed state.

**Contributing factors:** The pretty code path requires a TTY, so CI and casual local testing exercise the `textProgress` fallback instead; no regression test covered repeated `Close()` for any implementation.

## Resolution for the Issue

**Changes made:**
- `v2/progress_pretty.go` - Added a `closed bool` field to `prettyProgress`, guarded by the existing `p.mu`. `Close()` now returns nil immediately when the flag is already set, and sets it before running cleanup, making the method idempotent. First-call behaviour is unchanged.

**Approach rationale:** The struct already manages its lifecycle state (`active`, `completed`, `failed`) as fields under `p.mu`; a `closed` flag under the same mutex is the smallest change consistent with that style, and keeps all state transitions visible to any lock-holder.

**Alternatives considered:**
- `sync.Once` around the cleanup - Adds a second synchronization primitive to a struct that already serializes everything through `p.mu`, and hides the closed state from other methods; rejected as less consistent for no benefit.
- Guarding only the `close(p.signals)` call (e.g. nil-ing the channel) - Leaves `p.writer.Stop()` and the rest of the body re-running on every call; a full early return is simpler and makes the whole method idempotent.

## Regression Test

**Test file:** `v2/progress_core_test.go`
**Test names:** `TestPrettyProgress_Close_Idempotent` (reproduces the panic pre-fix), `TestTextProgress_Close_Idempotent` and `TestNoOpProgress_Close_Idempotent` (pin the already-correct behaviour of the other two implementations)

**What it verifies:** Calling `Close()` twice on each `Progress` implementation returns nil both times and never panics. The pretty test constructs `prettyProgress` directly (via the existing `newTestPrettyProgress` helper) so it exercises the TTY code path without needing a terminal.

**Run command:** `go test -race -run 'Close_Idempotent' ./...` (from `v2/`)

## Affected Files

| File | Change |
|------|--------|
| `v2/progress_pretty.go` | Add `closed` field; make `Close()` idempotent via early return |
| `v2/progress_core_test.go` | Add double-close regression tests for pretty, text, and noop implementations |
| `CHANGELOG.md` | Entry under Unreleased → Fixed; also removes a stray committed `<<<<<<< HEAD` conflict marker in that section |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test -race ./...`)
- [x] Linters/validators pass (`make lint`)

**Manual verification:**
- Confirmed the regression test panics at `progress_pretty.go:225` before the fix (red) and passes after (green).

## Prevention

**Recommendations to avoid similar bugs:**
- Treat every `Close()`/cleanup method as callable multiple times; guard one-shot operations (channel close, resource release) with a closed flag or early return.
- When a code path is unreachable in CI (here: TTY-only), test it by constructing the type directly, as `newTestPrettyProgress` does.

## Related

- T-1429 — data race between `SetContext` and the signal goroutine in this same file (fixed previously; the `contextDone` locking is untouched by this change).
- T-1730 — `SetContext` permanently terminating signal handling (open, deliberately out of scope here).
