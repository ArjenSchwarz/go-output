# Bugfix Report: FileWriter Append Paths Ignore Close Errors

**Date:** 2026-09-06
**Status:** Fixed
**Ticket:** T-1399

## Description of the Issue

`FileWriter.Write` reports `Close` failures on its normal overwrite path — it names its return value and, in a deferred function, promotes a `Close` error to the return value when nothing earlier failed. The append-mode helpers do not. `appendByteLevel` defers `_ = file.Close()` with a comment claiming close errors "are intentionally ignored as we already have the data", and `appendHTMLWithMarker` calls `tempFile.Close()` three times without looking at the result — including the close that precedes the atomic rename of the temp file over the original.

On filesystems where `close(2)` is where delayed writeback or metadata errors surface (NFS, some FUSE and network filesystems, disk-full conditions reported late), an append-mode `Write` therefore returns `nil` even though the data may not have been persisted. For HTML, the temp file is renamed over the original after a close whose failure was never observed.

**Reproduction steps:**
1. Create a `FileWriter` with `WithAppendMode()` targeting an existing file.
2. Call `Write` for a byte-level format (JSON, Markdown, etc.), for CSV (which delegates to the byte-level helper), or for HTML (temp-file-and-rename path).
3. Arrange for `Close` on the written file to fail (the regression test injects the failure through a small file abstraction; in production it depends on the filesystem).
4. Observe that `Write` returns `nil` on all three append paths, while the overwrite path returns `failed to close file: ...`.

**Impact:** Medium. Any caller relying on `Write`'s error return to decide whether an append was durable can be misled on filesystems that report errors at close time. The overwrite path already behaved correctly, so the inconsistency is confined to append mode; there is no data corruption on healthy filesystems.

## Investigation Summary

Traced every file handle opened for writing in `v2/file_writer.go` and compared how each path handles the `Close` result.

- **Symptoms examined:** Append-mode writes returning `nil` regardless of `Close` outcome; the three unchecked `tempFile.Close()` calls in the HTML path; the `_ = file.Close()` defer in the byte-level path.
- **Code inspected:** `v2/file_writer.go` — `Write` (overwrite path, lines ~130–150), `appendToFile`, `appendByteLevel`, `appendHTMLWithMarker`, `appendCSVWithoutHeaders`, `fileNeedsRowSeparator`; `v2/.golangci.yml` (linter configuration); `v2/writer.go` (`WriteError.Unwrap`).
- **Hypotheses tested:**
  - CSV append needs its own fix — ruled out: `appendCSVWithoutHeaders` performs no writes itself; it strips the header, decides whether a row separator is needed, and delegates to `appendByteLevel`. Fixing the byte-level helper fixes CSV.
  - The `defer func() { _ = file.Close() }()` in `fileNeedsRowSeparator` is part of the bug — ruled out: that handle is opened read-only with `os.Open` for a single `ReadAt`; closing a read-only descriptor cannot lose data, so ignoring its close error is correct.
  - The linter should have caught the unchecked `tempFile.Close()` calls — confirmed why it did not: `errcheck` is disabled in `v2/.golangci.yml` because fluent-API methods intentionally discard return values, so unchecked `Close` calls are invisible to `make lint`.
  - A deferred close-check in the HTML path would suffice — rejected: the temp file must be closed *before* `os.Rename`, and a deferred check would only run after the rename had already replaced the original, defeating the purpose.

## Discovered Root Cause

The append helpers were written as a separate feature (PR #33) and did not adopt the close-error discipline that the overwrite path already had. `appendByteLevel` codified the wrong assumption in a comment ("we already have the data"): the data has been handed to the kernel, but on some filesystems `close` is the call that reports whether it reached stable storage. `appendHTMLWithMarker` simply never inspected the `Close` result on any of its three call sites.

**Defect type:** Missing error handling (ignored `Close` return value) on three call sites.

**Why it occurred:**
1. Why do append writes return `nil` after a failed close? — Because the append helpers discard the `Close` result.
2. Why do they discard it? — `appendByteLevel` was written on the assumption that a successful `Write` + `Sync` makes `Close` errors irrelevant; the HTML path's bare `tempFile.Close()` calls look like cleanup rather than a durability step.
3. Why was the assumption not challenged? — The overwrite path's correct pattern lives in `Write`, while the helpers are further down the file; nothing forced them to share code.
4. Why did tooling not catch it? — `errcheck` is disabled in the lint configuration, and no test exercised a failing `Close`, because there was no seam through which a test could inject one.
5. Why was there no seam? — The helpers call `os.OpenFile`/`os.CreateTemp` directly and operate on `*os.File`, so a close failure could only come from the real filesystem.

**Contributing factors:** `errcheck` disabled repo-wide; no shared helper for write-sync-close; no file abstraction in the writer for fault injection.

## Resolution for the Issue

**Changes made:**
- `v2/file_writer.go` (`appendByteLevel`) - Named the return value and replaced the deferred `_ = file.Close()` with the same check the overwrite path uses: a `Close` error is returned as `failed to close file: ...` when no earlier error exists. `appendCSVWithoutHeaders` delegates its write to this helper, so CSV append is fixed by the same change.
- `v2/file_writer.go` (`appendHTMLWithMarker`) - The close that precedes `os.Rename` now returns `failed to close temp file: ...` on failure, so a temp file whose close failed never replaces the original (the existing deferred `os.Remove` cleans it up). The two closes on the write/sync error paths are now explicit `_ = tempFile.Close()`, documenting that the earlier error takes precedence.
- `v2/file_writer.go` (`writableFile`, `FileWriter.wrapFile`, `openedFile`) - Small test seam: an interface covering the `Write`/`Sync`/`Close` subset of `*os.File`, and an unexported per-instance hook that wraps each file opened for writing. Production behaviour is unchanged (a nil hook returns the `*os.File` as-is); tests inject a handle whose `Close` fails.

**Approach rationale:** The overwrite path already had the right pattern, so the append helpers now mirror it instead of introducing a new mechanism. The HTML path needs an explicit (not deferred) check because the close must be verified *before* the rename. The seam is the smallest change that makes close failures injectable through the public `Write` API, which is what lets the regression test fail before the fix and pass after it without depending on OS-specific filesystem behaviour.

**Alternatives considered:**
- Extract a shared `writeSyncClose(f, data)` helper used by all three paths and unit-test it with a fake - Not chosen: the tests would cover the helper but not that each path routes through it, and they could not exist before the fix. Reasonable follow-up if a fourth write path appears.
- Package-level `var openFile = os.OpenFile` override - Rejected: reintroduces mutable global state, which the v2 design deliberately avoids, and would need a second override for `os.CreateTemp`.
- Deferred close check in the HTML path - Rejected: it would run after `os.Rename`, so a failed close would be reported only after the unverified temp file had already replaced the original.

**Observation (out of scope):** errors from `appendByteLevel` — including the new close error — are returned by `appendToFile` without the `*WriteError` wrapping that the overwrite and HTML paths apply. This inconsistency predates this fix and is left unchanged here.

## Regression Test

**Test file:** `v2/file_writer_close_errors_test.go`
**Test name:** `TestFileWriterReportsCloseErrors`

**What it verifies:** Through the public `Write` API, with a `faultyFile` wrapper injected via the unexported `wrapFile` seam:
- the overwrite, byte-level append, CSV append, and HTML append paths all return an error wrapping the injected close error (`errors.Is`) with a `failed to close file` / `failed to close temp file` message;
- the HTML path leaves the original file untouched and removes its temp file when the close fails;
- a close error never masks an earlier write error on the overwrite and byte-level paths;
- a wrapped file with no injected faults appends normally (the seam itself is behaviour-neutral).

**Run command:** `cd v2 && go test -run TestFileWriterReportsCloseErrors ./...`

Before the fix, the three append cases fail (`Write() error = nil, want ...`; the HTML case additionally shows the original file replaced). The overwrite and precedence cases pass before and after, documenting the discipline the append paths now share.

## Affected Files

| File | Change |
|------|--------|
| `v2/file_writer.go` | Report close errors on the byte-level and HTML append paths; add the `writableFile` seam |
| `v2/file_writer_close_errors_test.go` | New regression test with a close-failing file fake |
| `CHANGELOG.md` | Unreleased → Fixed entry |

## Verification

**Automated:**
- [x] Regression test passes (`go test -run TestFileWriterReportsCloseErrors ./...`)
- [x] Full test suite passes (`make test`, plus `go test -race -run TestFileWriter`)
- [x] Linters/validators pass (`make lint`, `gofmt`)

**Manual verification:**
- Confirmed pre-fix (checkpoint commit `410f96a`) that the three append subtests fail with `Write() error = nil` and the overwrite/precedence/no-fault subtests pass; all seven pass after the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Treat `Close` on a file opened for writing as part of the write, not as cleanup: check its error when no earlier error exists (the named-return + deferred-check pattern already used in `Write`).
- When an atomic write-temp-and-rename sequence is used, check the temp file's `Close` *before* the rename — a deferred check runs too late.
- `errcheck` is disabled repo-wide for fluent-API reasons; consider re-enabling it with an exclusion list for the fluent methods so unchecked `Close` calls are flagged again.
- Keep the `wrapFile` seam in mind when adding new file-writing paths so close-failure behaviour stays testable.

## Related

- Transit ticket T-1399
- PR #33 — introduced the append-mode helpers without the close-error discipline
- T-1629 / PR #100 — previous `FileWriter` fix (extension map race)
- T-1109 / PR #82 — `appendCSVWithoutHeaders` row-separator fix (same code path)
