# Bugfix Report: AsError and IsCancelled Have Weaker Semantics Than Stdlib errors.As/errors.Is

**Ticket:** T-1515
**Date:** 2026-08-22
**Status:** Fixed

## Description of the Issue

Two helpers in `v2/errors.go` reimplemented stdlib error-inspection semantics
incompletely:

- `AsError` reimplemented `errors.As` with a manual unwrap loop that only
  follows `Unwrap() error`. It therefore missed matches inside
  `Unwrap() []error` multi-error trees — which `MultiWriteError` in
  `v2/multi_writer.go` actually implements — and ignored the `As(any) bool`
  conversion hook that `errors.As` honors.
- `IsCancelled` compared `err == context.Canceled || err ==
  context.DeadlineExceeded` with identity comparison, so any wrapped
  cancellation (`fmt.Errorf("op: %w", ctx.Err())`) was misclassified as a
  non-cancellation.

A related defect from the same audit: `RenderError`, `WriterError`,
`ContextError`, `PipelineError`, and `MultiError` each duplicated the same
context-formatting snippet using unsorted Go map iteration, producing
non-deterministic error messages (only `StructuredError` sorted its keys).
`MultiError`'s per-error source details had the same problem.

**Reproduction steps:**
1. `IsCancelled(fmt.Errorf("render json: %w", context.Canceled))` returns
   `false` (should be `true`).
2. `AsError` on a `*MultiWriteError` containing a `*WriterError` returns
   `false` (its `Unwrap() []error` branch is never traversed; the old loop
   only recognised `Unwrap() error`).
3. Call `(&RenderError{Context: map[string]any{...8 keys...}}).Error()`
   repeatedly — the `context=[...]` section changes order between calls.

**Impact:** Wrapped context cancellations flowing through the render path are
not recognised as cancellations, so callers checking `IsCancelled` on wrapped
errors get the wrong classification. Errors aggregated by `MultiWriter` are
invisible to `AsError`-based type extraction (including `ToStructuredError`,
which uses `AsError` for every error type). Non-deterministic error messages
break log grepping, deduplication, and message-based test assertions.

## Investigation Summary

- **Symptoms examined:** `AsError`'s manual loop (v2/errors.go:459-486) and
  `IsCancelled`'s `==` comparisons (v2/errors.go:443-456); the
  `Error()` methods of the five error types iterating context maps directly.
- **Code inspected:** `v2/errors.go`, `v2/multi_writer.go`
  (`MultiWriteError.Unwrap() []error`), `v2/debug.go` and `v2/output.go`
  (all `AsError`/`IsCancelled` callers), `v2/errors_core_test.go`
  (existing expectations).
- **Callers checked (to preserve intended behaviour):** All three render-path
  `IsCancelled` call sites (`v2/output.go:232,258,411`) pass `ctx.Err()` and
  intend to abort on *any* context termination — cancellation or deadline.
  The existing `TestIsCancelled` also expects `context.DeadlineExceeded` to
  report `true`. So `DeadlineExceeded` must continue to count as cancellation.
- **Hypotheses tested:** Whether delegating to stdlib could change behaviour
  for existing callers — no: `errors.As` is a strict superset of the manual
  loop for every caller (`*PanicError`, `*RenderError`, etc. all use plain
  `Unwrap() error` chains), and `errors.Is` is a strict superset of `==`.

## Discovered Root Cause

`AsError` and `IsCancelled` hand-rolled traversal/comparison logic that the
standard library already provides, and the hand-rolled versions lagged behind
stdlib semantics: no `Unwrap() []error` support (added to stdlib in Go 1.20),
no `As(any) bool` hook support, and sentinel identity comparison instead of
`errors.Is` chain traversal.

**Defect type:** Logic error (incomplete reimplementation of stdlib
semantics); non-determinism from unsorted map iteration.

**Why it occurred:** The helpers were written as bespoke loops rather than
delegating to `errors.As`/`errors.Is`; when the package later gained a
multi-error type (`MultiWriteError`) and errors started being wrapped with
`%w`, the helpers were never revisited. The context-formatting duplication
spread by copy-paste; only the newest type (`StructuredError`) sorted keys.

**Contributing factors:** Six near-identical `Error()` implementations made
it easy for a fix (sorting) to land in one place and not the others.

## Resolution for the Issue

**Changes made:**
- `v2/errors.go` — `AsError` now delegates directly to `errors.As`, gaining
  `Unwrap() []error` tree traversal and `As(any) bool` hook support while
  keeping the same generic signature.
- `v2/errors.go` — `IsCancelled` now uses `errors.Is(err, context.Canceled)
  || errors.Is(err, context.DeadlineExceeded)` plus `errors.As` for
  `*CancelledError`. Documented decision: `DeadlineExceeded` continues to
  count as cancellation (all render-path callers pass `ctx.Err()` and must
  abort on either form of termination; the existing test suite also expects
  it). `ContextError` is *not* inherently a cancellation — it is classified
  by its cause chain, which `errors.Is`/`errors.As` traverse naturally.
- `v2/errors.go` — extracted a shared `formatContext` helper that renders a
  `map[string]any` as `key=value` pairs with sorted keys, and replaced the
  eight duplicated unsorted-iteration snippets in `RenderError.Error`,
  `ContextError.Error`, `MultiError.Error` (operation context in both its
  single- and multi-error branches, plus per-error source details),
  `WriterError.Error`, `PipelineError.Error` (operation and pipeline
  context), and `StructuredError.Error` (inline sort replaced by the
  helper). All error messages with context are now deterministic.

**Approach rationale:** Delegating to stdlib is the minimal fix that is
correct by construction — `errors.As`/`errors.Is` are strict supersets of
the old logic for every existing caller, so no call-site changes are needed
and the public API is unchanged.

**Alternatives considered:**
- Extending the manual loop with `Unwrap() []error` and `As` support —
  rejected: reimplements stdlib a second time and will drift again.
- Restricting `IsCancelled` to `context.Canceled` only — rejected: all
  render-path callers pass `ctx.Err()` and intend to abort on deadline
  expiry too; existing tests codify `DeadlineExceeded == true`.
- Special-casing `ContextError` in `IsCancelled` — rejected: a
  `ContextError` is a generic wrapper; whether it represents a cancellation
  depends solely on its cause, which chain traversal already handles.

## Regression Test

**Test file:** `v2/errors_semantics_test.go`

**Test names:**
- `TestIsCancelledWrappedContextErrors` — wrapped `context.Canceled` /
  `context.DeadlineExceeded` (single and nested `%w`), `ContextError`
  wrapping a cancellation vs a non-cancellation, and a multi-error tree
  containing `context.Canceled`.
- `TestAsErrorMultiErrorTree` — finds a `*WriterError` inside a
  `*MultiWriteError` (`Unwrap() []error`), bare and wrapped in `%w`.
- `TestAsErrorAsMethodHook` — honors a custom `As(any) bool` hook.
- `TestErrorMessagesDeterministicContext` — asserts sorted-key context
  output for all six error types (8-key map; unsorted iteration has a
  1/8! chance of passing accidentally).
- `TestMultiErrorSourceDetailsDeterministic` — sorted source details in
  `MultiError.Error()`.

**Run command:** `go test -run 'TestIsCancelledWrappedContextErrors|TestAsErrorMultiErrorTree|TestAsErrorAsMethodHook|TestErrorMessagesDeterministicContext|TestMultiErrorSourceDetailsDeterministic' ./...` (from `v2/`)

## Affected Files

| File | Change |
|------|--------|
| `v2/errors.go` | `AsError` delegates to `errors.As`; `IsCancelled` uses `errors.Is`/`errors.As`; shared sorted `formatContext` helper replaces eight unsorted map-iteration snippets |
| `v2/errors_semantics_test.go` | New regression tests |
| `CHANGELOG.md` | Entry under Unreleased → Fixed |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Linters/validators pass (`make lint`)

**Manual verification:**
- Confirmed all `AsError`/`IsCancelled` call sites (`v2/output.go`,
  `v2/debug.go`, `v2/errors.go`) rely only on semantics that stdlib
  delegation preserves or strengthens.

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer delegating to stdlib `errors.Is`/`errors.As` over hand-rolled
  unwrap loops or sentinel `==` comparisons; stdlib semantics evolve
  (e.g. Go 1.20 multi-error support) and bespoke copies silently lag.
- When formatting maps into user-visible strings, always sort keys —
  route through a shared helper instead of copy-pasting iteration.

## Related

- T-1508 — MultiError source tracking rework (positional sources map);
  this fix builds on it and keeps its behaviour, only sorting the
  rendered detail order.
- T-1524 — deterministic multi-format write ordering (same determinism
  theme in the render path).
