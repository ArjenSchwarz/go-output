# Bugfix Report: Output Render Accepts Nil Configuration Entries

**Date:** 2026-05-25
**Status:** Fixed
**Ticket:** T-1184

## Description of the Issue

`Output.Render` validated only that the `formats` and `writers` slices were
non-empty. It did not validate the individual entries. As a result, a `Format`
with a nil `Renderer`, a nil transformer, or a nil writer was carried through to
the rendering path and dereferenced, where it triggered a nil pointer panic.
The panic was caught by the `SafeExecuteWithTracer` recovery wrapper and
surfaced as a recovered `PanicError` instead of a normal up-front validation
error.

**Reproduction steps:**
1. Build an `Output` with one of the following misconfigurations:
   - `WithFormat(Format{Name: "json"})` (nil `Renderer` field)
   - `WithTransformer(nil)`
   - `WithWriter(nil)`
2. Call `output.Render(ctx, doc)`.
3. Observe that the returned error is a recovered `PanicError`
   (`panic in operation "render-json": runtime error: invalid memory address or
   nil pointer dereference`) rather than a `ValidationError`.

**Impact:** Low/medium severity. Misconfiguration produced confusing panic
errors with stack traces instead of clear validation messages. No data
corruption, but poor caller ergonomics and harder debugging.

## Investigation Summary

- **Symptoms examined:** Recovered `PanicError` returned from `Render` for nil
  configuration entries.
- **Code inspected:** `v2/output.go` `Render` (validation block) and the
  rendering path: `f.Renderer.Render`, `transformer.CanTransform`, and
  `writer.Write`. Validation helpers in `v2/errors.go` (`ValidateNonNil`,
  `ValidateSliceNonEmpty`, `FailFast`, `NewValidationError`).
- **Hypotheses tested:** Confirmed the existing validation block only checks
  slice non-emptiness, never entry validity. Ruled out the `TransformPipeline`
  (T-1131) and `MultiWriter` (T-1168) paths — those are separate APIs in other
  files; this defect is in the top-level `Output` configuration path.

## Discovered Root Cause

The `Render` validation block checked slice length but never inspected slice
entries, so nil renderers/transformers/writers passed validation and were
dereferenced during rendering.

**Defect type:** Missing input validation.

**Why it occurred:** `ValidateSliceNonEmpty` only guards against empty slices.
There was no per-entry nil check, and the panic-recovery wrapper masked the
underlying nil dereference as a generic `PanicError`, hiding the real cause.

**Contributing factors:** Interface-typed slice entries (`Renderer` field,
`Transformer`, `Writer`) are nilable, and the functional-options API
(`WithFormat`, `WithTransformer`, `WithWriter`) accepts any value without
validation.

## Resolution for the Issue

**Changes made:**
- `v2/output.go` (`Render`) - After the existing non-empty slice checks, call a
  new `validateConfigEntries` helper and return its error before rendering.
- `v2/output.go` (`validateConfigEntries`) - New helper that iterates formats,
  transformers, and writers and returns a `ValidationError` ("cannot be nil")
  for the first nil entry (nil `Format.Renderer`, nil transformer, nil writer).

**Approach rationale:** Fail-fast validation at the top of `Render` matches the
existing pattern (the empty-slice checks live there) and keeps the fix contained
to `output.go`. Reusing `NewValidationError` produces errors consistent with the
rest of the API and the existing empty-slice errors.

**Alternatives considered:**
- Validate at configuration time in the `WithX` option functions - Rejected
  because options are applied without an error channel, and `Render` is the
  natural validation boundary where the config snapshot is taken.
- Return the panic-recovered error as-is - Rejected because a `PanicError` with a
  stack trace is poor ergonomics for a simple misconfiguration.

## Regression Test

**Test file:** `v2/output_test.go`
**Test name:** `TestOutput_Render_NilConfigurationEntries`

**What it verifies:** For each of nil renderer, nil transformer, and nil writer,
`Render` returns a `*ValidationError` mentioning the relevant field and "cannot
be nil", and does NOT return a `*PanicError`.

**Run command:** `go test -run TestOutput_Render_NilConfigurationEntries ./...`
(from `v2/`)

## Affected Files

| File | Change |
|------|--------|
| `v2/output.go` | Added `validateConfigEntries` helper and a call to it in `Render` after the non-empty slice checks |
| `v2/output_test.go` | Added `TestOutput_Render_NilConfigurationEntries` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Linters/validators pass (`make lint`, 0 issues)

**Manual verification:**
- Confirmed the regression test fails before the fix (recovered `PanicError`)
  and passes after.

## Prevention

**Recommendations to avoid similar bugs:**
- Validate the contents of nilable interface slices, not just their length,
  before dereferencing entries.
- Be aware that panic-recovery wrappers can mask missing validation; prefer
  explicit up-front checks for configuration that is known at call time.

## Related

- Separate but related nil-entry tickets in other files: T-1131
  (`TransformPipeline`) and T-1168 (`MultiWriter`). Out of scope here.
