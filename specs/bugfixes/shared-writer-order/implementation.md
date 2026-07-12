# Implementation Explanation: Shared Writer Order (T-1524)

Branch: `T-1524/bugfix-shared-writer-order` — diff against `origin/main`, including pre-push review fixes.

## Beginner Level

### What Changed

This library can turn one document into several output formats at once — for example JSON and CSV — and send them all to the same destination, such as the terminal. Before this fix, the order in which the formats appeared was random: sometimes JSON came first, sometimes CSV. Nothing crashed, the output was just shuffled differently on every run.

The fix keeps the fast part (producing each format) running in parallel, but makes the writing part take turns: format outputs are now written in the order you asked for them.

### Why It Matters

Command-line tools are often used in scripts that read their output. If the output layout changes from run to run, those scripts break unpredictably. With this fix, asking for "JSON, then CSV" always gives you JSON first, then CSV — every run.

### Key Concepts

- **Goroutine**: a lightweight worker Go can run in parallel with others. Think of several chefs cooking dishes at the same time.
- **Race condition (ordering)**: when the final result depends on which worker happens to finish first. Here: whichever format finished rendering first got written first.
- **Writer**: the destination the output goes to (terminal, file, S3). A "shared" writer means several formats send output to the same one.
- **The fix in one sentence**: let all chefs cook in parallel, but plate the dishes in menu order instead of whoever-finishes-first order.

---

## Intermediate Level

### Changes Overview

- `v2/output.go` — `renderWithConfig` restructured from "one goroutine per format renders AND writes" into a two-phase pipeline. The old `processFormatData` is split into `transformFormatData` (concurrent phase) and `writeFormatData` (sequential phase).
- `v2/output_write_order_test.go` — new regression test file: test doubles and six tests covering shared-writer order, multi-writer order, failed-format skipping, progress timing, MultiError ordering, and write-failure semantics.
- `v2/docs/API.md`, `CHANGELOG.md` — the ordering guarantee and its two side effects documented.
- `specs/bugfixes/shared-writer-order/report.md` — full investigation and resolution report.

### Implementation Approach

Phase 1 fans out one goroutine per format. Each goroutine renders and transforms, then stores its result in `results[idx]` and any error in `renderErrs[idx]` — its own slot, so no locks are needed; `wg.Wait()` is the happens-before edge that makes those slots safe to read afterwards. Phase 2 walks `formats` in declared order and writes each result to all writers sequentially. Because the write loop is single-threaded, the old `workMu` mutex and error channel are deleted; errors accumulate in a plain slice, which is what makes `MultiError` ordering deterministic.

Error semantics are preserved: a format that fails to render is recorded but doesn't block later formats; a write failure stops only that format's remaining writers. Each phase-2 write pass gets its own `SafeExecuteWithTracer` wrapper, so writer panics are recovered under a `write-<format>` label (an improvement — previously they were recovered under the render label).

A review fix also releases each `results[i]` slot (`results[i] = nil`) as it is handed to the write phase, so earlier formats' buffers can be collected while later formats are still being written to slow writers.

### Trade-offs

- **Buffer-then-write vs streaming order**: the first byte now lands only after the slowest render finishes. Renders stay concurrent, so wall time is essentially unchanged; renderer output was already fully buffered (`[]byte`), so peak memory matches the old design's worst case.
- **Alternatives rejected**: fully sequential pipeline (gives up render concurrency for nothing); documenting the nondeterminism (users of a CLI output library expect stable order); per-writer ordering locks (more machinery, same result).
- **Progress side effect**: `SetCurrent` stays at 0 until all renders complete, then advances once per write. Totals and status messages are unchanged. This is documented and pinned by a test.

---

## Expert Level

### Technical Deep Dive

The pre-fix design conflated the parallelizable stage (render/transform, independent per format) with the order-sensitive stage (writes to shared writers). The `Writer` contract only guarantees per-call atomicity — `StdoutWriter`'s mutex makes one `Write` atomic but cannot sequence calls across goroutines — so cross-format ordering was left entirely to the scheduler.

Notable details of the restructure:

- **Memory model**: per-slot writes into `results`/`renderErrs` with `wg.Done()`/`wg.Wait()` as the synchronization point is the minimal correct pattern; verified clean under `-race`. No `errgroup` dependency exists in the module, and none is needed since partial failure must not cancel siblings.
- **Error context fix**: `multiErr.AddContext("total_formats", len(formats))` now uses the snapshot copied under `o.mu.RLock()` in `Render` rather than the previously unsynchronized read of `o.formats`.
- **Cancellation**: there is no explicit check between phases, but `writeFormatData` checks `IsCancelled` before every write, so cancellation mid-render yields one `CancelledError` per format — the same aggregation shape as before.
- **`workDone` stays a pointer**: reviewed and deliberately kept. `SafeExecuteWithTracer` recovers panics; with pointer semantics the per-write increments survive a mid-format panic, whereas a value-plus-return signature would lose the completed-write count in that path.

### Architecture Impact

The transform/write split creates a clean seam: `transformFormatData` is a pure `[]byte -> []byte` stage safe for any concurrency structure, and `writeFormatData` is the only place ordering matters. A future relaxation (e.g. per-format done channels letting format i write as soon as formats 0..i have rendered, improving first-byte latency while preserving order) would touch only phase 2 — but it would change the documented progress behaviour that `TestOutput_Render_ProgressAdvancesOnlyDuringWritePhase` pins, so it is a deliberate API decision, not a refactor.

### Potential Issues

- **Latency-sensitive consumers**: anything expecting the fastest format's bytes early now waits for the slowest render. Documented in CHANGELOG.md.
- **Progress observers**: progress bars sit at 0 during the render phase, then advance quickly. Cosmetic, documented, and tested.
- **Peak memory**: all format outputs are held simultaneously between `wg.Wait()` and their write. Same as the old design's worst case; the slot-release fix bounds retention during slow writes.
- **Error-order consumers**: code that (incorrectly) relied on scheduler-order `MultiError.Errors` will now see declared format order. This is strictly more predictable.

---

## Completeness Assessment

**Fully implemented:**
- Deterministic write order in declared format order across shared and multiple writers (tested, race-clean).
- Concurrent render/transform preserved; per-format panic recovery in both phases.
- Preserved error semantics: failed render skips only that format's writes; failed write stops only that format's remaining writers (both tested).
- Deterministic `MultiError` ordering and documented progress timing (both tested).
- Documentation: godoc on `Render`/`renderWithConfig`, API.md, CHANGELOG.md, bugfix report.

**Partially implemented:**
- None identified.

**Missing:**
- None identified against the report's scope. The optional first-byte-latency relaxation is out of scope by design.
