# Implementation Explanation: error-semantics-stdlib (T-1515)

Branch: `T-1515/bugfix-error-semantics-stdlib` vs `origin/main`

## Beginner Level

### What Changed

The go-output library has helper functions for asking questions about errors:
"was this operation cancelled?" (`IsCancelled`) and "is there an error of type X
hiding inside this error?" (`AsError`). Go's standard library already ships
battle-tested functions for exactly these questions (`errors.Is` and
`errors.As`), but the library had written its own simpler versions — and the
home-grown versions missed cases the standard ones handle.

Think of an error in Go as a set of nested envelopes: the outer envelope says
"rendering failed", inside it another says "writing the file failed", and the
innermost holds the real cause, e.g. "operation cancelled". The old
`IsCancelled` only looked at the outermost envelope, so a cancellation wrapped
inside another message was reported as "not cancelled". The old `AsError`
could open envelopes one at a time in a straight line, but some envelopes
contain *several* envelopes at once (a multi-error) — it never looked inside
those. Both helpers now hand the work to the standard library, which opens
every envelope, including branching ones.

A second fix: several error types print extra "context" (key=value details)
in their messages. Go's maps return keys in a random order on purpose, so the
same error printed twice could show its details in a different order each
time. Seven copies of that printing code were replaced by one shared helper
(`formatContext`) that sorts the keys first, so messages are now stable.

### Why It Matters

Code that checks "did the user cancel this?" now gets the right answer even
when the cancellation is buried inside other error wrapping — which is how
errors actually travel through this library. And because error messages are
now stable, searching logs, de-duplicating alerts, and asserting on messages
in tests all work reliably.

### Key Concepts

- **Error wrapping (`%w`)**: attaching an error inside another so the cause
  is preserved — the envelope analogy above.
- **`errors.Is` / `errors.As`**: standard-library tools that walk the whole
  chain (and trees) of wrapped errors to find a sentinel value or a type.
- **Map iteration order**: Go deliberately randomizes it; anything printed
  straight from a map loop changes between runs unless you sort first.

## Intermediate Level

### Changes Overview

One production file changed — `v2/errors.go` — plus a new regression-test
file `v2/errors_semantics_test.go`, the bugfix report, and a CHANGELOG entry.

1. `AsError[T error](err, *T)` — body replaced by a one-line delegation to
   `errors.As(err, target)`. This gains `Unwrap() []error` tree traversal
   (Go 1.20 multi-error support, used by `MultiWriteError`) and support for
   custom `As(any) bool` conversion hooks. Signature and public API unchanged.
2. `IsCancelled(err)` — `==` sentinel comparisons replaced with
   `errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)`,
   plus `errors.As` for `*CancelledError`. Wrapped cancellations
   (`fmt.Errorf("op: %w", ctx.Err())`) now classify correctly.
3. New unexported helper `formatContext(map[string]any) string` renders
   sorted `key=value` pairs; it replaced eight duplicated unsorted loops in
   the `Error()` methods of `RenderError`, `ContextError`, `MultiError`
   (operation context in both its single- and multi-error branches, plus
   per-error source details), `WriterError`, `PipelineError` (pipeline and
   operation context), and `StructuredError` (which already sorted, but
   inline — now shares the helper). Import of `sort` dropped in favour of
   `slices.Sort`.

### Implementation Approach

Delegate-to-stdlib rather than extend the bespoke loop: `errors.As`/`errors.Is`
are strict supersets of the old logic for every existing caller (verified
against all call sites in `v2/output.go`, `v2/debug.go`, `v2/errors.go`), so
no call-site changes were needed. The deliberate behavioural decision —
`context.DeadlineExceeded` continues to count as "cancelled" — is documented
in the `IsCancelled` godoc and the report: all three render-path callers pass
`ctx.Err()` and must abort on either form of context termination, and the
pre-existing `TestIsCancelled` codifies it.

### Trade-offs

- Extending the manual loop with `Unwrap() []error` + `As` support was
  rejected: it re-implements stdlib a second time and will drift again.
- Restricting `IsCancelled` to `context.Canceled` only was rejected: callers
  and existing tests require deadline expiry to abort too.
- `ContextError` is not special-cased as a cancellation; it is classified by
  its cause chain, which `errors.Is`/`errors.As` traverse naturally.
- `IsCancelled` may now walk the chain up to three times, but every render-path
  call receives either `nil` (early return) or a bare sentinel (first
  `errors.Is` hits immediately), and it only runs on the failure path.

## Expert Level

### Technical Deep Dive

- `AsError` keeps its generic `[T error]` constraint, which preserves the
  compile-time guarantee the wrapper exists for: `errors.As` panics at runtime
  on a non-pointer/non-error target, while `AsError` makes those mistakes
  unrepresentable. The delegation also changes `nil`-error handling from an
  explicit guard to `errors.As`'s own `err == nil` short-circuit — same
  observable result (returns false).
- The old loop had a subtle extra defect the new code fixes for free: it did
  a direct type assertion `err.(T)` without honouring `As` hooks and stopped
  at the first non-`Unwrap() error` node, so any error joined via
  `errors.Join` or `MultiWriteError.Unwrap() []error` was unreachable.
  `ToStructuredError` builds on `AsError` for every error type, so multi-error
  aggregates were previously invisible to structured-error extraction.
- `IsCancelled` semantics: `errors.Is` also honours custom `Is(error) bool`
  methods, so third-party errors that declare equivalence to
  `context.Canceled` now classify as cancelled — a strictly broader, and
  intended, contract.
- `formatContext` output is byte-identical to the old code for a single
  ordering of keys; determinism is the only observable change.
  `MultiError.Error()` source details prepend `component=` before appending
  the formatted details string, producing one nested join — cosmetically
  identical output, negligible extra allocation on a cold path.

### Architecture Impact

Public API surface unchanged; no call-site edits anywhere. The library now
tracks stdlib error semantics automatically as they evolve (as they did in
Go 1.20), instead of lagging behind in bespoke copies. `formatContext` gives
future error types a single place to route context rendering through.
T-1508's positional `Sources` map handling and T-1649's planned typed-nil
validation are untouched.

### Potential Issues

- Any external caller relying on the *old, narrower* semantics (e.g.
  treating a wrapped `context.Canceled` as non-cancellation) sees a
  behaviour change — this is the bug being fixed, and the CHANGELOG entry
  spells it out.
- `errors.Is` on large multi-error trees is O(nodes); irrelevant here since
  classification runs on failure only.
- The 8-key determinism tests have a 1/8! (~0.0025%) chance of passing by
  luck per type under unsorted iteration — negligible, and the suite covers
  seven call sites, compounding the odds.

## Completeness Assessment

**Fully implemented:**
- `AsError` delegation to `errors.As` (multi-error trees + `As` hooks) —
  covered by `TestAsErrorMultiErrorTree` and `TestAsErrorAsMethodHook`.
- `IsCancelled` via `errors.Is`/`errors.As`, wrapped and nested cases —
  covered by `TestIsCancelledWrappedContextErrors` (6 cases including a
  multi-error tree and a negative `ContextError` case).
- Deterministic context formatting across all six error types and
  `MultiError` per-error source details — covered by
  `TestErrorMessagesDeterministicContext` (7 cases) and
  `TestMultiErrorSourceDetailsDeterministic`.

**Partially implemented:** none identified.

**Missing:** nothing against the report's scope. Out-of-scope by design:
T-1508 positional sources rework, T-1649 typed-nil validation, and the
pre-existing `v2/examples` go.mod drift.
