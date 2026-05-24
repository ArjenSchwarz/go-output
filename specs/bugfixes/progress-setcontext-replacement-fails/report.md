# Bugfix Report: progress-setcontext-replacement-fails

**Date:** 2026-05-24
**Status:** Fixed

## Description of the Issue

`textProgress.SetContext` installs a cancellation watcher goroutine each time it
is called. The watcher read `p.ctx` (the shared struct field) instead of the
context it was created for. When `SetContext` was called a second time, the old
watcher was released by `p.cancel()`, and after the new context was installed
the released watcher could mark the progress as failed and set `p.err` from the
new `p.ctx.Err()`.

**Reproduction steps:**
1. Create a text progress: `p := NewProgress(...)`.
2. Install a live context: `p.SetContext(ctx1)`.
3. Replace it with another live context: `p.SetContext(ctx2)`.
4. Observe (under `go test -race`) a data race on `p.ctx` between the watcher
   goroutine and `SetContext`, and a possible spurious failure of the progress.

**Impact:** Replacing a progress context (a supported v1-compatibility
operation) could mark a healthy progress as failed and corrupt `p.err`, plus a
data race on the `p.ctx` field. Medium severity; affects any caller that calls
`SetContext` more than once on a `textProgress`.

## Investigation Summary

- **Symptoms examined:** Spurious `failed=true`/`err` after a second
  `SetContext`; data race reported by `go test -race`.
- **Code inspected:** `v2/progress_text.go` (`SetContext` and the watcher
  goroutine), `v2/progress.go` (`NewProgress`), existing tests in
  `v2/progress_core_test.go`. The same pattern exists in
  `v2/progress_pretty.go` but is out of scope for this ticket (T-1254 scopes the
  fix to `textProgress`).
- **Hypotheses tested:** Confirmed via `-race` that the watcher reads `p.ctx`
  (progress_text.go:243) concurrently with the reassignment in `SetContext`
  (progress_text.go:239). Confirmed the watcher's `Err()` was read from the
  shared field rather than its own context.

## Discovered Root Cause

The watcher goroutine closed over the receiver `p` and dereferenced the shared
field `p.ctx` in both `<-p.ctx.Done()` and `p.ctx.Err()`, assuming it would not
change. A subsequent `SetContext` reassigns `p.ctx`/`p.cancel` and cancels the
prior derived context, releasing the old watcher. The released watcher then
operated against whatever `p.ctx` had become, so a context replacement was
treated like a real cancellation/failure.

**Defect type:** Concurrency bug — data race plus stale-closure / incorrect
shared-state read.

**Why it occurred:** The goroutine read mutable shared state (`p.ctx`) instead
of capturing the specific derived context and cancel token it was created to
watch.

**Contributing factors:** No existing test exercised replacing the context, and
CI does not run with `-race`, so the race went undetected.

## Resolution for the Issue

**Changes made:**
- `v2/progress_text.go` (`SetContext`) - The watcher now captures its own
  derived context (`watchedCtx`) as a local variable rather than reading
  `p.ctx`. When it fires, it ignores the completion if `p.ctx != watchedCtx`
  (i.e. it has been superseded by a later `SetContext`). Also tidied the
  nil-context path to clear `p.ctx`/`p.cancel` instead of leaving a derived
  context installed.

**Approach rationale:** Capturing the context/cancel token per watcher removes
the data race entirely (the goroutine never reads the shared `p.ctx` for its
wait) and makes stale completions explicitly ignorable via the identity check
under the lock. This matches the ticket's expectation that each watcher capture
its own derived context.

**Alternatives considered:**
- Using a generation counter (e.g. an int incremented per `SetContext`) checked
  by the watcher - rejected because the captured-context identity check is
  simpler and needs no extra field.
- Not deriving a child context and watching the caller's context directly -
  rejected because the derived context/cancel is needed for `Close()` to stop
  the watcher.

## Regression Test

**Test file:** `v2/progress_core_test.go`
**Test name:** `TestProgress_SetContext_Replacement_DoesNotFail`

**What it verifies:** After replacing the progress context with a second live
context, the progress remains active and is not marked failed (no spurious
`failed`/`err`), and the current (live) context can still fail the progress when
cancelled. Run under `-race`, it also proves the `p.ctx` data race is gone.

**Run command:**
`cd v2 && go test -race -run TestProgress_SetContext_Replacement_DoesNotFail -count=1 .`

## Affected Files

| File | Change |
|------|--------|
| `v2/progress_text.go` | Watcher captures its own derived context/cancel and ignores stale (replaced) completions; nil-context path clears state |
| `v2/progress_core_test.go` | Added regression test for context replacement not failing the progress |
| `specs/bugfixes/progress-setcontext-replacement-fails/report.md` | This report |

## Verification

**Automated:**
- [x] Regression test passes (and fails under `-race` before the fix)
- [x] Full test suite passes (`make test`, and `go test -race .`)
- [x] Linters/validators pass for changed files (pre-existing unrelated
  `goconst` warnings in `s3_writer.go`/`transformer*.go` remain)

**Manual verification:**
- Confirmed the regression test produced a data race (progress_text.go:239 vs
  :243) under `go test -race` before the fix.
- Confirmed it passes deterministically under `-race` (5x) after the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Goroutines should capture the specific values they operate on as locals rather
  than reading mutable receiver fields after they may have changed.
- When a long-lived goroutine acts on shared state, verify identity (or a
  generation token) under the lock before acting, to ignore stale work.
- Consider running CI with `-race`; this class of bug is invisible without it.

## Related

- Transit ticket: `T-1254`
- Same watcher pattern exists in `v2/progress_pretty.go` (`prettyProgress.SetContext`)
  and may warrant a follow-up ticket; out of scope for T-1254.
