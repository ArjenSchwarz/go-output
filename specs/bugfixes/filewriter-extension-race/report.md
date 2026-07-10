# Bugfix Report: FileWriter Extension Map Races with Writes

**Date:** 2026-07-10
**Status:** Fixed

## Description of the Issue

`FileWriter.Write` calls `generateFilename(format)` before acquiring `fw.mu`, and `generateFilename` reads the `fw.extensions` map. `SetExtension` mutates that same map while holding `fw.mu`. Because `Write` does not hold the mutex during the read, a caller that changes extensions while another goroutine writes hits a Go map read/write data race, which can crash the process with a fatal `concurrent map read and map write` error.

**Reproduction steps:**
1. Create a `FileWriter` with `NewFileWriter(dir, "race-{format}.{ext}")`.
2. Start one goroutine looping `fw.SetExtension("json", ...)`.
3. Concurrently, start another goroutine looping `fw.Write(ctx, "json", data)`.
4. Run under `go test -race`: the detector reports a data race on `fw.extensions` (read at `v2/file_writer.go:169`, write at `v2/file_writer.go:158`). Without `-race`, the runtime can fault fatally on concurrent map access.

**Impact:** Any application that reconfigures extensions on a shared `FileWriter` while writes are in flight can crash. The `FileWriter` documentation promises thread-safety via `sync.Mutex`, so callers reasonably assume this is safe.

## Investigation Summary

Inspected the locking discipline of every `fw.extensions` access in `v2/file_writer.go`.

- **Symptoms examined:** Data race reported by `go test -race` between `SetExtension` and `Write`; potential fatal runtime error on concurrent map access.
- **Code inspected:** `v2/file_writer.go` — `Write`, `SetExtension`, `generateFilename`, `validateFormatMatch`, `WithExtensions`.
- **Hypotheses tested:**
  - `generateFilename` read is unsynchronized — confirmed: called at `Write` line 88, before `fw.mu.Lock()` at line 99.
  - `validateFormatMatch` (also reads the map) is unsynchronized — ruled out: it is only reached via `appendToFile`, which `Write` calls after acquiring the lock.
  - `WithExtensions` option mutation is racy — ruled out: options run at construction time, before the writer is shared.

## Discovered Root Cause

`Write` performs filename generation (a read of the mutable `fw.extensions` map) outside the critical section, while `SetExtension` mutates the map inside it. One unlocked reader plus one locked writer is still a data race — the lock only works when all accesses share it.

**Defect type:** Race condition (unsynchronized map read concurrent with locked map write).

**Why it occurred:** The mutex in `Write` was placed to guard file I/O ("Write the file with proper locking"), treating filename generation as pure computation over immutable configuration. `SetExtension` later made that configuration mutable after construction without auditing existing unlocked readers.

**Contributing factors:** The map read is hidden two calls away from the lock site, so the gap is easy to miss in review; the default test suite runs without `-race`, so the race was never surfaced.

## Resolution for the Issue

**Changes made:**
- `v2/file_writer.go` (`Write`) - Moved `fw.mu.Lock()` to before the `generateFilename` call, so the `fw.extensions` read shares the critical section with `SetExtension`'s write. Filename validation and file I/O remain under the same lock as before.
- `v2/file_writer.go` (`FileWriter.mu`) - Updated the field comment to state that it guards the extensions map as well as file operations.

**Approach rationale:** Acquiring the existing mutex slightly earlier is the minimal change that restores the invariant that all `fw.extensions` accesses happen under `fw.mu`. Filename generation is trivial string work, so holding the lock for it has no measurable cost.

**Alternatives considered:**
- Split configuration behind a `sync.RWMutex` shared by `generateFilename`/`validateFormatMatch` readers and `SetExtension` writer - Rejected: more moving parts and a second locking discipline to maintain, with no practical concurrency benefit since writes are serialized by `fw.mu` anyway.
- Copy-on-write extensions map (replace the map on `SetExtension`, read atomically) - Rejected: over-engineering for a configuration setter that is called rarely.

## Regression Test

**Test file:** `v2/file_writer_extension_race_test.go`
**Test name:** `TestFileWriterSetExtensionConcurrentWithWrite`

**What it verifies:** `SetExtension` can run concurrently with `Write` on a shared `FileWriter` without a data race. One goroutine mutates the extension mapping for the full duration of another goroutine performing writes into a temporary directory.

**Run command:** `cd v2 && go test -race -run TestFileWriterSetExtensionConcurrentWithWrite ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/file_writer.go` | Acquire `fw.mu` before generating the filename in `Write`; document that `mu` guards `extensions` |
| `v2/file_writer_extension_race_test.go` | New race regression test |

## Verification

**Automated:**
- [x] Regression test passes (`go test -race -run TestFileWriterSetExtensionConcurrentWithWrite`)
- [x] Full test suite passes (`make test`, plus `go test -race ./...`)
- [x] Linters/validators pass (`make lint`)

**Manual verification:**
- Confirmed pre-fix that the regression test fails under `-race` with the exact read/write pair described in the ticket (read `file_writer.go:169`, write `file_writer.go:158`).

## Prevention

**Recommendations to avoid similar bugs:**
- When a struct mixes a mutex with mutable maps, every access to the guarded fields must occur under the lock — including reads reached indirectly through helper methods.
- When adding a setter that makes previously construction-only state mutable, audit all existing readers for lock coverage.
- Consider running the test suite with `-race` in CI so unsynchronized access is caught automatically.

## Related

- Transit ticket T-1629
