# Bugfix Report: pretty-progress-signal-race

**Date:** 2026-07-10
**Status:** Fixed
**Ticket:** T-1429

## Description of the Issue

`v2/progress_pretty.go` has an unsynchronized access to `p.ctx`: the signal-handling
goroutine started by `NewPrettyProgress` calls `handleSignals()`, which evaluates
`p.contextDone()` on every select iteration. `contextDone()` reads the shared `p.ctx`
field without holding `p.mu`, while `SetContext()` writes `p.ctx` under the mutex.
Calling `SetContext()` on a TTY-backed `prettyProgress` while the signal goroutine is
running is a data race.

**Reproduction steps:**
1. Construct a `prettyProgress` (TTY path) so `handleSignals()` is running.
2. Concurrently call `SetContext()` from another goroutine while signals drive the
   loop (must be a different goroutine than the one feeding `p.signals`, otherwise
   the channel send/receive creates a happens-before edge that hides the race).
3. Run under `go test -race`: the detector reports a write at
   `progress_pretty.go:324` (`SetContext`) racing with a read at
   `progress_pretty.go:100` (`contextDone`).

**Impact:** Data race in any program that uses `NewPrettyProgress` on a real terminal
and calls `SetContext` after construction. Undefined behaviour per the Go memory
model; the signal goroutine can observe stale or inconsistent context state. Fails
`go test -race` in consuming projects.

## Investigation Summary

- **Symptoms examined:** Race-detector report between `handleSignals` -> `contextDone`
  (unsynchronized read of `p.ctx`) and `SetContext` (locked write of `p.ctx`).
- **Code inspected:** `v2/progress_pretty.go` (all reads/writes of `p.ctx`),
  `v2/progress_core_test.go` (existing prettyProgress test helpers).
- **Hypotheses tested:** Audited every access to `p.ctx`:
  - `NewPrettyProgress` write — safe (happens-before the `go` statements).
  - `SetContext` write and its watcher goroutine's read — safe (both under `p.mu`).
  - `contextDone()` read from the signal goroutine — unsynchronized. Confirmed as the
    only defect by a focused failing regression test.

## Discovered Root Cause

`contextDone()` reads `p.ctx` without acquiring `p.mu`, while `SetContext()` mutates
`p.ctx` under the lock.

**Defect type:** Race condition (unsynchronized read of a mutex-guarded field).

**Why it occurred:** `contextDone()` was introduced as a nil-safety accessor in the
earlier `prettyprogress-handle-signals-nil-context` fix, at a time when the code was
written as if `p.ctx` were set once at construction. The v1-compatibility
`SetContext()` makes `p.ctx` mutable at runtime, and the reader was never revisited.
The related `progress-setcontext-replacement-fails` fix (T-1358) synchronized the
watcher goroutine's read of `p.ctx` but not the signal goroutine's.

**Contributing factors:** `NewPrettyProgress` falls back to `textProgress` when stderr
is not a TTY, so CI tests never exercised the signal goroutine concurrently with
`SetContext`.

## Resolution for the Issue

**Changes made:**
- `v2/progress_pretty.go:99-109` - `contextDone()` now acquires `p.mu.RLock()` before
  reading `p.ctx`, and documents why the lock is needed.

**Approach rationale:** `p.mu` is already an `sync.RWMutex` guarding every other
access to `p.ctx`; taking a read lock in the accessor is the minimal change that
makes all reads and writes of the field synchronized. The lock is held only for the
duration of the channel lookup (never while blocked in select), so there is no
deadlock or contention concern. It matches the pattern the T-1358 fix established
for the SetContext watcher goroutine.

**Alternatives considered:**
- Capture a stable done channel once at goroutine start (or pass it as a parameter
  to `handleSignals`) - also race-free, but changes the function signature and the
  goroutine's observable lifetime semantics for no additional benefit; the locked
  accessor is smaller and keeps existing behaviour.
- Store `p.ctx` in an `atomic.Pointer[context.Context]` - avoids the mutex but adds
  a second synchronization mechanism for a field the mutex already guards elsewhere,
  making the invariants harder to follow.

## Regression Test

**Test file:** `v2/progress_core_test.go`
**Test name:** `TestPrettyProgress_SetContext_SignalGoroutine_NoRace`

**What it verifies:** Driving the signal loop (so `contextDone()` is re-evaluated on
each iteration) while `SetContext` concurrently replaces the context is race-free.
The `SetContext` calls run in a separate goroutine from the signal feeder so no
accidental happens-before edge masks the race.

**Run command:** `cd v2 && go test -race -run TestPrettyProgress_SetContext_SignalGoroutine_NoRace -count=1 .`

## Affected Files

| File | Change |
|------|--------|
| `v2/progress_core_test.go` | Added race regression test |
| `v2/progress_pretty.go` | `contextDone()` reads `p.ctx` under `p.mu.RLock()` |

## Verification

**Automated:**
- [x] Regression test fails before the fix (race detected at `progress_pretty.go:100` vs `:324`)
- [x] Regression test passes (`go test -race -run TestPrettyProgress_SetContext_SignalGoroutine_NoRace -count=5 .`)
- [x] Full test suite passes with the race detector (`go test -race ./...`: ok for v2 and v2/icons)
- [x] Linters/validators pass (`golangci-lint run`: 0 issues; `go vet`, `gofmt` clean)

## Prevention

**Recommendations to avoid similar bugs:**
- When adding an accessor for a mutex-guarded field, take the lock inside the
  accessor rather than assuming callers hold it or that the field is immutable.
- Run the package tests with `-race` whenever goroutines are added or modified.

## Related

- Transit ticket T-1429
- Prior fixes: `specs/bugfixes/prettyprogress-handle-signals-nil-context/`,
  `specs/bugfixes/progress-setcontext-replacement-fails/` (T-1358)
