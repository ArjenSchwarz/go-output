# Bugfix Report: Shared Writer Order

**Date:** 2026-07-10
**Status:** Fixed
**Ticket:** T-1524

## Description of the Issue

When an `Output` is configured with two or more formats and a shared writer
(for example one `StdoutWriter`), the order in which each format's output
appears in the writer varies from run to run. Nothing errors — the output is
simply shuffled at format granularity, which is a real defect class for a CLI
output library. Nothing documented the ordering as undefined either.

**Reproduction steps:**
1. Create an `Output` with `WithFormats(JSON(), CSV())` and a single `WithWriter(NewStdoutWriter())`.
2. Call `Render` and observe which format's output appears first.
3. Repeat — the order changes between runs depending on goroutine scheduling.

**Impact:** Medium severity (from code audit). Affects every consumer that
renders multiple formats to a shared writer; downstream tooling that parses
CLI output cannot rely on a stable layout.

## Investigation Summary

Systematic (Fagan-style) inspection of `v2/output.go`.

- **Symptoms examined:** cross-format write order follows goroutine completion
  order; regression tests confirmed `[third second first]` for declared order
  `[first second third]` when the first format renders slowest.
- **Code inspected:** `renderWithConfig` and `processFormatData` in
  `v2/output.go`; `StdoutWriter` mutex semantics in `v2/writer.go`; progress
  accounting expectations in `v2/output_test.go`.
- **Hypotheses tested:** the writer's internal mutex was confirmed to only
  make a single `Write` call atomic — it cannot sequence calls across
  goroutines. Renderer output is already fully buffered (`[]byte`), so
  buffering results before writing adds no meaningful memory overhead versus
  the existing concurrent design.

## Discovered Root Cause

`renderWithConfig` spawned one goroutine per format, and each goroutine both
rendered and wrote via `processFormatData`. The pipeline conflated the
parallelizable stage (render/transform — independent per format) with the
order-sensitive stage (writes to shared writers), leaving cross-format write
ordering entirely to the Go scheduler.

**Defect type:** Race condition (nondeterministic ordering, not data race).

**Why it occurred:** concurrency was introduced for render throughput and the
write step was carried along inside the same goroutine. The `Writer` contract
only guarantees per-call atomicity, and no component owned cross-format
sequencing.

**Contributing factors:** the fused render+write function made the ordering
gap invisible; no documentation stated ordering was undefined; existing tests
only checked output content and progress totals, never cross-format order.

## Resolution for the Issue

**Changes made:**
- `v2/output.go` (`renderWithConfig`) — split into two phases. Phase 1:
  renders and transforms still run concurrently (one goroutine per format,
  each storing its transformed bytes and error into its own slot of a results
  slice — no shared-state races). Phase 2: after `wg.Wait()`, writes are
  performed sequentially in declared format order, each format's write pass
  wrapped in its own `SafeExecuteWithTracer` for panic recovery.
- `v2/output.go` — `processFormatData` split into `transformFormatData`
  (concurrent phase) and `writeFormatData` (sequential phase). The `workMu`
  mutex and the error channel were removed since the write phase is
  single-threaded; `workDone` is now a plain counter threaded through the
  sequential write loop, and errors are collected into a slice in
  deterministic order.
- `v2/output.go` — doc comments on `Render` and `renderWithConfig` now state
  the ordering guarantee.
- `v2/docs/API.md` — performance section documents that writes are serialized
  in declared format order.

**Approach rationale:** this is the API-compatible fix proposed in the ticket:
rendering stays concurrent (no throughput loss for the expensive stage), while
writes become strictly ordered. Error-aggregation semantics are preserved: a
format that fails to render is reported but does not block other formats, and
a write failure stops only that format's remaining writers.

**Alternatives considered:**
- Fully sequential pipeline (render and write one format at a time) — simpler,
  but gives up render concurrency for no benefit.
- Documenting the nondeterminism instead of fixing it — rejected; stable output
  order is the behaviour users of a CLI output library expect.
- Per-writer ordering locks — more machinery, same result, harder to reason
  about.

**Behavioural notes for reviewers:**
- Progress: `SetCurrent` now remains at 0 until all renders complete, because
  writes happen after the concurrent phase. Total work units
  (formats × writers), the increment-per-write behaviour, and all status
  messages are unchanged, so existing progress expectations hold.
- Error aggregation: `MultiError` membership is unchanged, but error ordering
  is now deterministic (declared format order) instead of scheduler order.
- Latency: the first byte is not written until the slowest format has finished
  rendering. Renders remain concurrent, so total wall time is essentially
  unchanged.

## Regression Test

**Test file:** `v2/output_write_order_test.go`
**Test names:**
- `TestOutput_Render_WriteOrderMatchesFormatOrder` — with a deliberately slow
  first-declared format, writes to a shared writer must still occur in
  declared format order across repeated renders (both call order and byte
  order are asserted).
- `TestOutput_Render_WriteOrderWithMultipleWriters` — every writer receives
  formats in declared order.
- `TestOutput_Render_WriteOrderSkipsFailedFormats` — a failing format is
  reported as an error but does not prevent the remaining formats from being
  written in order.
- `TestOutput_Render_ProgressAdvancesOnlyDuringWritePhase` — added after
  review: a spy `Progress` asserts `SetCurrent` is never called while any
  format is still rendering and then advances once per write, in write order.
- `TestOutput_Render_MultiErrorOrderMatchesFormatOrder` — added after review:
  with two failing formats (slowest declared first), `MultiError.Errors`
  follows declared format order rather than goroutine completion order.
- `TestOutput_Render_WriteOrderContinuesAfterWriteFailure` — added in pre-push
  review: a writer that fails for one format stops only that format's
  remaining writers; later formats are still written to every writer in
  declared order, and the failure surfaces as a `WriterError`. This locks in
  the preserved write-failure semantics rather than reproducing the original
  bug.

**Run command:**
`go test -run 'TestOutput_Render_WriteOrder|TestOutput_Render_ProgressAdvancesOnlyDuringWritePhase|TestOutput_Render_MultiErrorOrderMatchesFormatOrder' -count=1`
(in `v2/`)

The first five tests all fail before the fix — the write-order tests observe
goroutine-completion order (`[third second first]`, `[second first]`,
`[third first]`), and the progress and error-ordering tests catch writes and
error aggregation happening in scheduler order — and all pass after it. The
write-failure test asserts semantics preserved by the fix rather than a
pre-fix failure.

## Affected Files

| File | Change |
|------|--------|
| `v2/output.go` | Two-phase pipeline: concurrent render/transform, sequential ordered writes |
| `v2/output_write_order_test.go` | New regression tests for write ordering |
| `v2/docs/API.md` | Documented the ordered-write guarantee |
| `specs/bugfixes/shared-writer-order/report.md` | This report |

## Verification

**Automated:**
- [x] Regression tests pass (`go test -run 'TestOutput_Render_WriteOrder' -count=3`)
- [x] Full unit test suite passes (`make test`)
- [x] Integration tests pass (`make test-integration`)
- [x] Race detector clean (`go test -race -count=1` in `v2/`)
- [x] Linter passes (`golangci-lint run` — 0 issues); `gofmt -l` clean; `go vet` clean

**Manual verification:**
- Confirmed pre-fix failure output shows goroutine-completion ordering
  (`[third second first]`, `[second first]`, `[third first]`).

## Prevention

**Recommendations to avoid similar bugs:**
- Keep order-sensitive side effects (writes to shared destinations) out of
  fan-out goroutines; fan out pure computation, join, then apply effects in a
  defined order.
- When adding concurrency, document the ordering guarantees (or explicitly the
  lack of them) in the public API's doc comments.
- Test ordering explicitly with artificial delays that would flip the order
  under scheduler-dependent behaviour.

## Related

- Transit ticket T-1524
- Found in code audit (medium severity)
