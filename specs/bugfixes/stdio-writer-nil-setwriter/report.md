# Bugfix Report: Stdout and Stderr Writers Panic After SetWriter Nil

**Ticket:** T-1387
**Date:** 2026-08-03
**Status:** Fixed

## Description of the Issue

`StdoutWriter.SetWriter` and `StderrWriter.SetWriter` accept an `io.Writer`
for injection (primarily used in tests) but store the value without any
validation. When `SetWriter(nil)` is called, the nil writer replaces the
default `os.Stdout`/`os.Stderr`. A later `Write` call passes context and
input validation, acquires the mutex, and then calls `sw.writer.Write(data)`
on the nil interface, panicking with a nil pointer dereference.

**Reproduction steps:**
1. `sw := output.NewStdoutWriter()` (or `NewStderrWriter()`)
2. `sw.SetWriter(nil)`
3. `sw.Write(ctx, output.FormatText, []byte("test"))` — panics with
   `runtime error: invalid memory address or nil pointer dereference`

**Impact:** Any caller passing a nil writer (directly, or via a variable that
happens to be nil) crashes the process instead of receiving an error. Both
public writers are affected. Prior nil-writer hardening covered renderers
(`RenderTo`), progress (`WithProgressWriter`), and `MultiWriter.AddWriter`,
but missed these two setters.

## Investigation Summary

- **Symptoms examined:** Nil pointer dereference panic in `Write` after
  `SetWriter(nil)`, reproduced by the new regression tests before the fix.
- **Code inspected:** `v2/stdout_writer.go`, `v2/stderr_writer.go`,
  `v2/multi_writer.go` (`AddWriter`), `v2/progress.go`
  (`WithProgressWriter`), `v2/base_renderer.go` (nil writer checks) to
  identify the established nil-handling pattern.
- **Hypotheses tested:** Confirmed `validateInput` only checks format and
  data, never the writer field, so nothing between `SetWriter(nil)` and the
  dereference can catch the nil. Also confirmed a zero-value
  `StdoutWriter{}`/`StderrWriter{}` (constructed without the `New*`
  constructor) has a nil internal writer and panics the same way.

Root cause chain (Five Whys):
1. Why does `Write` panic? `sw.writer` is nil when `sw.writer.Write(data)`
   runs.
2. Why is `sw.writer` nil? `SetWriter(nil)` stored the nil without
   validation.
3. Why did `SetWriter` accept nil? It was written as a bare test-injection
   setter with no guard.
4. Why did nothing downstream catch it? `validateInput` validates format and
   data only; the writer field is trusted implicitly.
5. Why was this missed? Earlier nil-writer fixes targeted renderers,
   progress, and `MultiWriter`; the stdio setters were overlooked.

## Discovered Root Cause

`SetWriter` stores its argument unconditionally, and `Write` dereferences
the stored writer without a nil check. Once nil is stored there is no code
path that can turn the eventual dereference into an error.

**Defect type:** Missing validation

**Why it occurred:** The setters were added as simple test hooks before the
codebase-wide nil-writer hardening effort, and that effort did not audit
them.

**Contributing factors:** The writer field is only ever non-nil when the
`New*` constructors are used and `SetWriter` receives valid input, so the
gap never surfaced in normal use.

## Resolution for the Issue

**Changes made:**
- `v2/stdout_writer.go` — `SetWriter` ignores a nil writer, keeping the
  existing writer; `Write` returns a `*WriteError` if the internal writer is
  nil (zero-value construction) instead of panicking.
- `v2/stderr_writer.go` — same two changes.

**Approach rationale:** Ignoring nil in the setter matches the established
pattern in `MultiWriter.AddWriter` and `WithProgressWriter` ("a nil writer
is ignored so the existing/default writer is kept"). The defensive check in
`Write` covers the remaining path to a nil writer — a zero-value struct that
bypassed the constructor — and converts it into the documented `WriteError`
path.

**Alternatives considered:**
- Make `Write` alone return an error for a nil writer (no setter guard) —
  rejected: silently keeping the default writer on `SetWriter(nil)` matches
  the library's existing convention and keeps the writer usable.
- Add reflection-based typed-nil detection (e.g. `SetWriter((*bytes.Buffer)(nil))`)
  — deferred: no such helper exists in the codebase yet, and open ticket
  T-1649 plans one consolidated reflection-based fix covering transformers,
  progress, renderers, and writers. A one-off helper here would fragment
  that work. Note that the most likely typed nil, `(*os.File)(nil)`, does
  not panic: `os.File.Write` returns `ErrInvalid` for a nil receiver, which
  already flows through the existing `WriteError` path.

## Regression Test

**Test file:** `v2/stdout_writer_test.go`, `v2/stderr_writer_test.go`
**Test name:** `TestStdoutWriterSetWriterNil`, `TestStderrWriterSetWriterNil`

**What it verifies:**
- `SetWriter(nil)` is ignored: a subsequent `Write` does not panic and the
  previously configured writer still receives the data.
- A zero-value writer (nil internal writer) returns a `*WriteError` from
  `Write` instead of panicking.

**Run command:** `go test -run 'TestStdoutWriterSetWriterNil|TestStderrWriterSetWriterNil' ./v2/...`

## Affected Files

| File | Change |
|------|--------|
| `v2/stdout_writer.go` | Guard `SetWriter` against nil; error instead of panic in `Write` |
| `v2/stderr_writer.go` | Guard `SetWriter` against nil; error instead of panic in `Write` |
| `v2/stdout_writer_test.go` | Regression tests |
| `v2/stderr_writer_test.go` | Regression tests |
| `CHANGELOG.md` | Changelog entry |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Linters/validators pass (`make lint`)

**Manual verification:**
- Confirmed the regression tests fail with the original panic before the
  fix and pass after.

## Prevention

**Recommendations to avoid similar bugs:**
- Any public setter that stores an interface used later without a nil check
  should either reject nil or the consumer must handle nil.
- T-1649 tracks the remaining typed-nil gap across all option types; when
  that lands, these setters should adopt the shared helper. (Landed: both
  setters now use the shared `isNilValue` helper — see
  `specs/bugfixes/typed-nil-output-validation`.)

## Related

- T-1387 (this fix)
- T-1649 — consolidated typed-nil validation (fixed in
  `specs/bugfixes/typed-nil-output-validation`; covers the typed-nil
  variant of this class of bug)
- specs/bugfixes/renderto-nil-writer — nil writer hardening for renderers
