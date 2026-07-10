# Bugfix Report: Shared Writer Order

**Date:** 2026-07-10
**Status:** In Progress
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

_To be completed once the fix is implemented._

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

**Run command:** `go test -run 'TestOutput_Render_WriteOrder' -count=1` (in `v2/`)

All three tests fail before the fix (observed order `[third second first]`).

## Affected Files

| File | Change |
|------|--------|
| `v2/output.go` | Two-phase pipeline: concurrent render/transform, sequential ordered writes (pending) |
| `v2/output_write_order_test.go` | New regression tests for write ordering |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes (including `-race`)
- [ ] Linters/validators pass (`make lint`)

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
