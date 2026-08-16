# Bugfix Report: Typed Nil Options Bypass Output Validation

**Ticket:** T-1649
**Date:** 2026-08-17
**Status:** Fixed

## Description of the Issue

Output's option validation compares interface values with `== nil`. A Go
interface holding a nil concrete pointer (a "typed nil") is itself non-nil,
so typed-nil transformers, progress indicators, renderers, and writers passed
every validation check and were dereferenced later, panicking during render
(surfaced as a recovered `PanicError` inside a `MultiError`) instead of
returning the documented validation error.

This consolidates T-1649 (transformers), T-1654 (progress), and T-1710
(renderers/writers), and picks up the stdio `SetWriter` typed-nil gap that
T-1387 explicitly deferred to this ticket.

**Reproduction steps:**
1. Build an Output with a typed-nil option, e.g. the ticket reproduction:
   ```go
   out := output.NewOutput(
       output.WithFormat(output.JSON()),
       output.WithWriter(output.WriterFunc(func(context.Context, string, []byte) error { return nil })),
       output.WithTransformer(output.NewFormatAwareTransformer(nil)), // (*FormatAwareTransformer)(nil)
   )
   ```
2. Call `out.Render(ctx, doc)`.
3. Observe a nil-pointer panic at `Priority()` inside the render goroutine,
   surfaced as `panic in operation "render-json": runtime error: invalid
   memory address or nil pointer dereference` rather than a validation error.

Equivalent failures existed for each option type:
- **Progress:** `NewOutput` only replaced progress when `output.progress ==
  nil`, so a typed-nil `Progress` survived and panicked at
  `progress.SetTotal(totalWork)` in `renderWithConfig`.
- **Renderer/Writer:** `validateConfigEntries` missed a typed-nil `Renderer`
  inside a `Format` and a typed-nil `Writer`; the panic happened at
  `f.Renderer.Render(...)` / `writer.Write(...)`.
- **TransformPipeline.Add:** the `transformer == nil` guard missed typed
  nils; `sortTransformers`/`Transform` then dereferenced them.
- **Stdio writers:** `StdoutWriter.SetWriter`/`StderrWriter.SetWriter`
  ignored only untyped nil, so a typed-nil `io.Writer` was stored and
  panicked on the next `Write` (deferred from T-1387).
- **Document:** `ValidateNonNil("document", doc)` boxed the `*Document`
  parameter into `any`, so `Render(ctx, nil)` passed validation and panicked
  at `doc.GetContents()` (`d.mu.RLock()` on a nil receiver).

**Impact:** Any consumer passing a typed-nil option — easy to do since
constructors like `NewFormatAwareTransformer(nil)` legitimately return nil
concrete pointers that box silently at interface call sites — got a runtime
panic (recovered into an opaque `PanicError`) during render instead of the
documented validation error. No data corruption; failure mode only.

## Investigation Summary

Followed the systematic (Fagan-style) inspection of every validation point in
the Output option pipeline.

- **Symptoms examined:** Nil-pointer panics during `Render` recovered by
  `SafeExecuteWithTracer` into `PanicError`, confirmed by the red run of the
  regression tests (panic at `format_aware.go` `CanTransform`, panic at
  `Priority()` in `transformFormatData`, `MultiError` wrapping
  `panic in operation "render-json"`).
- **Code inspected:** `v2/output.go` (`NewOutput`, `Render`,
  `validateConfigEntries`, `renderWithConfig`, `transformFormatData`,
  `writeFormatData`), `v2/transformer.go` (`TransformPipeline`),
  `v2/format_aware.go` (`NewFormatAwareTransformer`), `v2/stdout_writer.go` /
  `v2/stderr_writer.go` (`SetWriter`), `v2/errors.go` (`ValidateNonNil`).
- **Hypotheses tested:** Confirmed every guard uses interface `== nil`
  comparison; no reflection-based typed-nil detection existed anywhere in the
  package (the only related prior art is `isHashableError` in `errors.go` and
  the typed-nil-aware `sealContents` walk in `transform_data.go`, both
  narrowly scoped).

## Discovered Root Cause

**Defect type:** Missing validation (interface `== nil` cannot detect typed
nils).

**Why it occurred:** Five Whys — Render panics with typed-nil options →
because typed nils pass validation → because interface `== nil` only detects
untyped nil (an interface with a set type word and nil data word is non-nil)
→ because constructors returning concrete nil pointers box silently at
interface call sites → because prior nil-hardening (T-1131, T-1387, T-1444)
added untyped-nil guards per-site and deliberately deferred reflection-based
detection to a consolidated fix. Root cause: **no shared typed-nil detection
helper existed, so every validation point relied on interface equality, which
cannot see typed nils.**

**Contributing factors:** Go's interface semantics make this class of bug
invisible at compile time; the option pattern (`WithX(iface)`) maximises the
number of boxing sites.

## Resolution for the Issue

**Changes made:**
- `v2/errors.go` — added the shared reflection-based helper
  `isNilValue(value any) bool` next to the validation helpers. It reports
  true for untyped nil and for typed nils of pointer, map, slice, func,
  chan, interface, and unsafe-pointer kinds. `ValidateNonNil` now uses it, so
  the documented validation error path rejects typed nils (including a nil
  `*Document` boxed into `any`).
- `v2/output.go` — `validateConfigEntries` uses `isNilValue` for the
  renderer, transformer, and writer checks; `NewOutput` uses it for the
  progress fallback so a typed-nil `Progress` is treated as absent and
  replaced with the no-op progress.
- `v2/transformer.go` — `TransformPipeline.Add` ignores typed-nil
  transformers via `isNilValue`, matching its documented nil handling.
- `v2/format_aware.go` — `NewFormatAwareTransformer` returns nil for a
  typed-nil inner transformer instead of building a wrapper that panics on
  `Name()`/`Priority()`.
- `v2/stdout_writer.go`, `v2/stderr_writer.go` — `SetWriter` ignores
  typed-nil `io.Writer` values (the T-1387 deferral), keeping the previous
  writer.

**Approach rationale:** One shared helper applied at the existing validation
points is the smallest change that covers every option type with identical
semantics, exactly as the consolidated ticket prescribes. Rejection happens
on the documented validation error path (`*ValidationError`); progress falls
back to no-op because absence is its documented default.

**Alternatives considered:**
- Reject typed nils inside each `WithX` option (fail at construction) — not
  chosen: options have no error return, so rejection would mean silently
  dropping values or panicking in the constructor; validating at `Render`
  keeps the documented error path and reports the exact slice index.
- Skip typed-nil entries during rendering (treat as absent everywhere) — not
  chosen for transformers/renderers/writers: silently ignoring a configured
  renderer or writer hides misconfiguration; the ticket asks for the
  validation error path. Only progress, whose absence has a documented no-op
  default, is treated as absent.

## Regression Test

**Test file:** `v2/typed_nil_validation_test.go`

**Test names:**
- `TestRenderRejectsTypedNilTransformer` (subtests for `WithTransformer` and
  `WithTransformers`, using the ticket reproduction
  `NewFormatAwareTransformer(nil)`)
- `TestTransformPipelineAddIgnoresTypedNil`
- `TestNewFormatAwareTransformerRejectsTypedNil`
- `TestRenderTypedNilProgressFallsBackToNoOp`
- `TestRenderRejectsTypedNilRenderer`
- `TestRenderRejectsTypedNilWriter`
- `TestRenderRejectsTypedNilDocument`
- `TestStdioSetWriterIgnoresTypedNil` (subtests for stdout and stderr)

**What it verifies:** Each option type's typed-nil value is rejected via
`*ValidationError` (transformer, renderer, writer, document), treated as
absent (progress → no-op, pipeline `Add`, stdio `SetWriter`), or refused at
construction (`NewFormatAwareTransformer`), and never panics during render.

**Run command:** `go test -run TypedNil ./...` (from `v2/`)

## Affected Files

| File | Change |
|------|--------|
| `v2/errors.go` | Add shared `isNilValue` helper; make `ValidateNonNil` typed-nil aware |
| `v2/output.go` | Use `isNilValue` in `validateConfigEntries` and the `NewOutput` progress fallback |
| `v2/transformer.go` | `TransformPipeline.Add` ignores typed-nil transformers |
| `v2/format_aware.go` | `NewFormatAwareTransformer` returns nil for typed-nil input |
| `v2/stdout_writer.go` | `SetWriter` ignores typed-nil writers |
| `v2/stderr_writer.go` | `SetWriter` ignores typed-nil writers |
| `v2/typed_nil_validation_test.go` | Regression tests (new) |
| `CHANGELOG.md` | Fixed entry under Unreleased |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Linters/validators pass (`make lint`)

**Manual verification:**
- Confirmed the red run before the fix: `MultiError` wrapping
  `panic in operation "render-json"` for the ticket reproduction, SIGSEGV in
  `TransformPipeline.Transform`, typed nil stored by `Add`.

## Prevention

**Recommendations to avoid similar bugs:**
- Use `isNilValue` (not interface `== nil`) whenever validating an
  interface-typed configuration value that a caller could construct from a
  concrete nil pointer.
- Constructors that wrap an interface (like `NewFormatAwareTransformer`)
  should validate with `isNilValue` so nil-ness is decided once, at the
  boundary.
- When passing a concrete pointer to an `any`/interface parameter for
  validation, remember the boxing hides nil-ness — validate before or with
  reflection.

## Related

- T-1649 (consolidates T-1654 progress and T-1710 renderers/writers)
- T-1387 / `specs/bugfixes/stdio-writer-nil-setwriter` — explicitly deferred
  the stdio `SetWriter` typed-nil gap to this ticket
- T-1131 / `specs/bugfixes/transformpipeline-nil-transformer` — untyped-nil
  transformer hardening this fix extends
- T-1444 — nil functional options hardening in constructors
- T-1524 — two-phase render split that reworked `output.go` (ticket line
  numbers predate it)
