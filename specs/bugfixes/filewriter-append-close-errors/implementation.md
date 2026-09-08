# Implementation Explanation: FileWriter Append Paths Report Close Errors

**Ticket:** T-1399
**PR:** #130 (branch `T-1399/bugfix-filewriter-append-close-errors`)
**Generated:** 2026-09-07 by the pre-push review

---

## Beginner Level

### What Changed

When a program saves a file, "closing" the file is the last step. On ordinary local disks that step almost never fails, but on some storage (network drives, certain cloud or FUSE filesystems, disks that report "full" late) closing is the moment the system finally says whether everything was really written. `FileWriter` is the part of go-output that writes rendered reports (JSON, CSV, HTML, Markdown, and so on) to disk. When it *overwrote* a file it already listened to that final answer. When it was in **append mode** — adding to the end of an existing file — it threw the answer away and reported success regardless.

This change makes the append paths listen too. If closing the file fails, `Write` now returns an error, exactly as the overwrite path always did. HTML files are appended differently: a complete new copy is written to a temporary file and then swapped into place. For that path the close is checked *before* the swap, so a temp file that failed to close never replaces the original.

### Why It Matters

Any program that checks the error from `Write` to decide whether its output was safely stored can now trust that answer in append mode. On healthy local filesystems nothing observable changes; the fix matters where errors are reported late.

### Key Concepts

- **Append mode**: adding new content to the end of an existing file instead of replacing it, like adding lines to a log.
- **Close error**: the operating system can report a failure when a file handle is closed, not only when data is written — the receipt printer jamming after the card was already handed over.
- **Temp-file-and-rename**: writing the new version to a scratch file and renaming it over the original in one step, so readers never see a half-written file.
- **Test seam**: a small hook that lets a test swap in a fake file whose `Close` fails on purpose, because real disks do not fail on command.

---

## Intermediate Level

### Changes Overview

- `v2/file_writer.go`
  - `appendByteLevel` (JSON/YAML/Markdown/table, and CSV via `appendCSVWithoutHeaders`): now has a named return value and a deferred close that promotes a `Close` error to `failed to close file: ...` when no earlier error exists. Previously `defer func() { _ = file.Close() }()` with a comment claiming the data was already safe.
  - `appendHTMLWithMarker`: the close that precedes `os.Rename` is checked and returns `failed to close temp file: ...`. The two closes on the write-error and sync-error paths became explicit `_ = tempFile.Close()` so the earlier error keeps precedence. The pre-existing deferred `os.Remove(tempPath)` cleans up the temp file when the close fails.
  - New unexported `writableFile` interface (`io.Writer` + `Sync() error` + `Close() error`), an unexported `wrapFile func(*os.File) writableFile` field on `FileWriter`, and an `openedFile` helper applied at all three write-side open sites (`Write`, `appendByteLevel`, `appendHTMLWithMarker`). With `wrapFile == nil` the `*os.File` is used directly.
- `v2/file_writer_close_errors_test.go`: `faultyFile` embeds `*os.File` with injectable Write and Close errors and always really closes the file so `t.TempDir` cleanup works. `TestFileWriterReportsCloseErrors` is a map-based table test with seven subtests: a no-fault control, close-error reporting on all four write paths, HTML original-left-intact with no temp-file leak, and write-vs-close precedence on the overwrite and byte-level paths.
- `CHANGELOG.md`: Unreleased → Fixed entry.
- `specs/bugfixes/filewriter-append-close-errors/report.md`: investigation report with root cause, alternatives, and verification.

### Implementation Approach

The overwrite path already used the idiomatic pattern — named return plus a deferred close that only overwrites a nil return — so the byte-level path adopts the identical block rather than a new mechanism. The HTML path cannot use a deferred check because the close must be verified before `os.Rename`; it uses an explicit check. Fault injection uses a per-instance function field rather than a package-level variable, matching v2's no-global-state design; the field is unexported and no option exposes it, so the public API is unchanged.

### Trade-offs

- **Shared `writeSyncClose` helper vs. mirroring the pattern**: mirroring keeps the diff small and lets each path keep its own error wording, at the cost of three near-identical close blocks. The report names the helper as a follow-up if a fourth path appears.
- **Function-field seam vs. `var openFile = os.OpenFile`**: the field avoids mutable global state and covers both `os.OpenFile` and `os.CreateTemp` with one hook, at the cost of a test-only field on a production struct.
- **Setting an unexported field from the test vs. adding an option**: keeps the exported API unchanged; acceptable because the test is in the same package.

---

## Expert Level

### Technical Deep Dive

- **Precedence.** In `appendByteLevel` the deferred check runs after `return fmt.Errorf(...)` has assigned `returnErr`, so a write or sync error always wins; a close error surfaces only on the otherwise-successful path. This matches `Write`. `Sync` still precedes `Close`, so a reported close error is a genuinely late condition (NFS writeback, `ENOSPC` under delayed allocation, `EIO`).
- **HTML close-once invariant.** Write-error path: close (ignored), return. Sync-error path: close (ignored), return. Success path: close (checked), then rename. Every path closes exactly once; no double close. The deferred `os.Remove(tempPath)` runs on every path and is a harmless `ENOENT` after a successful rename. Pre-fix, all three closes were unchecked, and the pre-rename one was the dangerous one.
- **Mutex discipline unchanged.** In `Write`, `fw.mu.Unlock` is deferred before the file-close defer, so LIFO ordering closes the file before unlocking; the append helpers run with the lock held.
- **Interface boxing.** `openedFile` returns `*os.File` inside `writableFile`; the dynamic dispatch on Write/Sync/Close is negligible next to the syscalls. `FileWriter` was already non-comparable (map field), so the func field changes nothing there.
- **Error-wrapping asymmetry (pre-existing, unchanged).** `Write` and `appendHTMLWithMarker` wrap with `fw.wrapError` (`*WriteError`); `appendByteLevel`'s errors — including the new close error — are returned by `appendToFile` unwrapped. `errors.As(err, &we)` for `*WriteError` succeeds on the overwrite and HTML paths but not on the byte-level or CSV append paths. The report documents this as out of scope.

### Architecture Impact

Contained to `FileWriter`. No exported API change. The seam sets a precedent for function-typed injection fields in v2 (existing seams are interface-typed setters such as `StdoutWriter.SetWriter`). Any future write path in this file should route its handle through `openedFile` to stay fault-testable.

### Potential Issues

- The seam covers only handles opened for writing; `fileNeedsRowSeparator`'s read-only handle still ignores its close error, which is correct — closing a read-only descriptor cannot lose data.
- `faultyFile` overrides only Write and Close, so sync-then-close precedence is untested on every path and write-then-close precedence is untested on the HTML path. Low risk given the code shape; cheap to add.
- `errcheck` remains disabled in `v2/.golangci.yml`, so this class of bug can recur silently. The report recommends re-enabling it with exclusions for the fluent API.

---

## Completeness Assessment

**Fully implemented**
- Byte-level append (and CSV via delegation) reports close errors with write/sync precedence.
- HTML temp-file close is checked before rename; the original is preserved and the temp file removed on failure.
- Regression test exercises all four paths through the public `Write` API; passes now and, per the checkpoint commit `410f96a` and the report, the three append cases failed before the fix.
- CHANGELOG entry and bugfix report are present and match the code.

**Partially covered**
- Precedence tests exist only for write-vs-close on the overwrite and byte-level paths; HTML write-vs-close and sync-vs-close on any path are not asserted.

**Out of scope (documented)**
- `*WriteError` wrapping for `appendByteLevel` errors — pre-existing inconsistency, called out in the report.
- Re-enabling `errcheck` — recommendation only.
