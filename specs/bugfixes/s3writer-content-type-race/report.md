# Bugfix Report: S3Writer Content Type Map Races With Writes

**Date:** 2026-07-10
**Status:** Fixed
**Ticket:** T-1583

## Description of the Issue

`S3Writer.SetContentType` mutated the shared `contentTypes` map without any
synchronization while `Write` read the same map through `getContentType`.
Running configuration updates concurrently with writes triggered a Go data
race and risked a `concurrent map read and map write` runtime panic.

**Reproduction steps:**
1. Create an `S3Writer` with a mock (or real) S3 client.
2. From one goroutine, repeatedly call `writer.SetContentType(output.FormatJSON, "application/json")`.
3. From another goroutine, repeatedly call `writer.Write(context.Background(), output.FormatJSON, []byte("{}"))`.
4. Run under `go test -race`: the detector reports races between
   `(*S3Writer).SetContentType` (`v2/s3_writer.go:314`) and
   `(*S3Writer).getContentType` (`v2/s3_writer.go:341`), plus write/write
   races between concurrent `SetContentType` calls.

**Impact:** Any application sharing an `S3Writer` across goroutines while
adjusting content types could hit undefined behaviour or crash with a map
concurrency panic. The v2 library explicitly promises thread-safe operations,
so this violated the documented contract.

## Investigation Summary

Systematic inspection (Fagan-style walkthrough) of `v2/s3_writer.go`.

- **Symptoms examined:** Race detector reports referenced in T-1583
  (reproduced 2026-06-19), pointing at the map write in `SetContentType` and
  the map read in `getContentType`.
- **Code inspected:** All accesses to `S3Writer.contentTypes`
  (`NewS3Writer`, `SetContentType`, `getContentType`, `WithContentTypes`),
  all `Write` code paths (`putS3Object`, `appendToS3Object`), and the other
  mutable `S3Writer` fields. Also reviewed sibling writers for conventions —
  `FileWriter` already guards mutable state with `sync.Mutex`
  (`v2/file_writer.go:30`).
- **Hypotheses tested:** Verified that no other post-construction setters
  exist (`appendMode`, `maxAppendSize` are only set via options at
  construction) and that `defaultContentTypes()` returns a fresh map per
  instance, so the race is confined to `contentTypes` on a single writer.
  Confirmed the existing `TestS3WriterConcurrency` never caught this because
  it only runs concurrent `Write` calls (pure map reads).

## Discovered Root Cause

Mutable shared state (the `contentTypes` map) is exposed through an
unsynchronized public setter on a type whose contract is thread safety.

**Defect type:** Race condition (unsynchronized concurrent map access).

**Why it occurred:** `S3Writer` was designed assuming configuration completes
before any writes. `SetContentType` was added as a post-construction setter
without revisiting that assumption, and nothing documents that it must not be
called after writes begin.

**Contributing factors:**
- `WithContentTypes` also mutates the map via `maps.Copy`; it is normally
  applied pre-publication in `NewS3WriterWithOptions`, but `S3WriterOption`
  is an exported func type, so it can be applied to a live writer too.
- The existing concurrency test only exercised concurrent reads, so
  `go test -race` in CI never saw the write path race.

## Resolution for the Issue

_To be filled in after the fix is implemented._

## Regression Test

**Test file:** `v2/s3_writer_test.go`
**Test name:** `TestS3WriterSetContentTypeConcurrentWithWrite`

**What it verifies:** Two goroutines repeatedly calling `SetContentType`
while another goroutine repeatedly calls `Write` produce no data race under
the race detector, and all writes still succeed with the expected number of
`PutObject` calls.

**Run command:** `go test -race -run TestS3WriterSetContentTypeConcurrentWithWrite ./...` (from `v2/`)

## Affected Files

| File | Change |
|------|--------|
| `v2/s3_writer.go` | Synchronize access to `contentTypes` (fix pending) |
| `v2/s3_writer_test.go` | Added regression test `TestS3WriterSetContentTypeConcurrentWithWrite` |

## Verification

**Automated:**
- [ ] Regression test passes under `go test -race`
- [ ] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- Confirmed the regression test fails with the documented race reports before
  the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Any exported setter on a type documented as thread-safe must synchronize
  with all readers of the state it mutates.
- When adding concurrency tests, include configuration mutation alongside
  reads, not just concurrent read-only operations.
- Prefer running `go test -race` for the whole package when touching shared
  state in writers.

## Related

- Transit ticket T-1583
- Similar prior fix pattern: `FileWriter` mutex (`v2/file_writer.go`)
