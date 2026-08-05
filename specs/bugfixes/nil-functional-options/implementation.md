# Implementation Explanation: Nil Functional Options (T-1444)

Branch `T-1444/bugfix-nil-functional-options`, diff `ce3c190..HEAD`.

## Beginner Level

### What Changed

The go-output library lets you configure things by passing small "option"
functions to constructors, like `NewTableContent("x", data, WithKeys("a", "b"))`.
Before this fix, if one of those options was `nil` — an empty placeholder
instead of a real function — the library tried to call it anyway and the whole
program crashed with a panic.

This change teaches every place that applies options to simply skip `nil`
entries. Fourteen loops across twelve source files got the same three-line
guard: `if opt == nil { continue }`. Each affected function's documentation now
says "Nil options are ignored." A new test file proves that every constructor
accepts nil options without crashing.

### Why It Matters

A common Go pattern is building a list of options step by step: start with an
empty list and append options only when certain conditions hold. A bug in that
caller code (or an uninitialized variable) can easily put a `nil` into the
list. Before, that mistake crashed the program at a distance from the actual
error, which is confusing to debug. Now the nil is harmlessly ignored.

### Key Concepts

- **Functional options**: a Go idiom where configuration is passed as functions
  that mutate a config struct. Think of them as order forms handed to a
  factory — each form tweaks one setting.
- **nil function**: in Go, a variable of function type can be "nil" (points to
  nothing). Calling it is like dialing a disconnected number — the program
  panics.
- **Panic**: an abrupt crash, unlike an error value which the caller can
  inspect and handle.

## Intermediate Level

### Changes Overview

- 12 production files in `v2/` gained the nil guard inside their option
  application loops: `table_options.go`, `text_options.go`, `raw_options.go`,
  `section_options.go`, `output.go`, `collapsible.go`, `collapsible_section.go`,
  `progress.go` (`NewProgress`, `NewAutoProgress`), `progress_pretty.go`,
  `file_writer.go`, `s3_writer.go`, `graph_content.go` (`NewDrawIOContent`,
  `NewDrawIOContentFromTable`) — 14 loops total, covering all 11 public option
  families.
- `v2/nil_options_test.go` (new): a map-based table test with one subtest per
  public constructor/applicator (20 direct cases), plus an interleaving test
  verifying that real options around a nil are still applied.
- Godoc on each application site now states the contract: "Nil options are
  ignored."
- `CHANGELOG.md` and `specs/bugfixes/nil-functional-options/report.md` document
  the fix.

### Implementation Approach

The guard is applied at the *application sites* rather than the many wrapper
functions (Builder methods, `NewProgressForFormat*`, etc.), because every
wrapper forwards its variadic slice to one of these fourteen loops. Guarding
the sink covers all entry points with no duplication at the call graph's leaves.

The regression test uses a shared `expectNoPanicWithNilOption(t, fn)` helper
with `recover()`, matching the repo's map-based table test convention and no
`t.Parallel()` (per v2/CLAUDE.md). Progress subtests `Close()` the indicators
they construct so the TTY-path goroutines and SIGWINCH handler registered by
`NewPrettyProgress` do not leak into other tests.

### Trade-offs

- **Silent skip vs. error return**: only three of the affected constructors
  return errors; making some families error on nil and others skip would be
  inconsistent. A nil option carries no intent, so skipping loses nothing.
- **Silent skip vs. descriptive panic**: the ticket's goal was removing the
  crash surface; a nicer panic message still crashes.
- **No behavior change for non-nil options**: the guard is purely additive, so
  no existing caller can be affected (nil previously always panicked).

## Expert Level

### Technical Deep Dive

Function types in Go are nilable, and the functional-options pattern's
implicit "options come from our `With*` constructors" contract is invisible to
the type system. The fix makes the nil case part of the documented contract at
each of the 14 `for _, opt := range opts` sinks. The guard is O(1) per option
in already-cold constructor paths — no performance consideration applies.

Coverage topology: the 20 direct regression subtests exercise every family;
`NewProgressForFormat`/`ForFormatName`/`ForFormats` are wrappers over
`NewProgress`/`NewPrettyProgress`, so their subtests verify forwarding rather
than distinct loops. The interleaving test pins the semantics that matter
beyond "no panic": a nil must not truncate application of subsequent options
(guard-with-`continue`, not `break`), verified via table key order and text
style.

### Architecture Impact

The contract "Nil options are ignored" is now uniform across all 11 option
families, which is the property that keeps the API learnable. New option
families should copy an existing loop (all now carry the guard) and add a case
to `nil_options_test.go` — the report's Prevention section records this.

### Potential Issues

- The 14 identical guards are deliberate repetition. A generic
  `applyOptions[T]` helper could unify them, but each family's applicator also
  owns family-specific defaulting, and the repo's convention is per-family
  applicators; the abstraction would save 2 lines per site at the cost of
  indirection.
- Pre-existing (not this PR): `NewAutoProgress` applies `opts` into a local
  `ProgressConfig` that is never read before it forwards the raw `opts` to
  `NewPrettyProgress`/`NewProgress`, and `NewPrettyProgress`'s non-TTY fallback
  similarly builds-then-discards a config before `NewProgress` re-applies the
  options (options applied twice). Dead computation worth a follow-up ticket.

## Completeness Assessment

- **Fully implemented**: nil guards on all 14 application loops (verified by
  grep — no unguarded `range opts` loop over option types remains in v2
  production code); godoc contract on every site; regression coverage for all
  11 families; CHANGELOG and bugfix report.
- **Partially implemented**: nothing identified.
- **Missing**: nothing required by the ticket. Follow-up candidate: the
  pre-existing dead `ProgressConfig` computation in `NewAutoProgress` /
  `NewPrettyProgress`'s non-TTY fallback.
