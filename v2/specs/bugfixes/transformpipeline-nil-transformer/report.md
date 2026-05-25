# Bugfix Report: TransformPipeline Panics On Nil Transformers

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

`TransformPipeline` accepted nil transformers and stored them, and
`NewFormatAwareTransformer` wrapped nil transformers. Any later call that invoked
an interface method on the stored/wrapped transformer dereferenced the nil
interface value and panicked with a nil-pointer segfault.

**Reproduction steps:**
1. `pipeline := output.NewTransformPipeline()`
2. `pipeline.Add(nil)`
3. Call `pipeline.Info()` (or `Transform`/`Has`/`Get`/`Remove`) — panics.
4. Similarly, `output.NewFormatAwareTransformer(nil)` returns a wrapper whose
   `Name()`/`Priority()`/`CanTransform()`/`Transform()` panic.

**Impact:** A nil transformer passed by a caller crashes the whole process
instead of producing a recoverable error. This is inconsistent with the rest of
the transformation API, which validates invalid inputs (nil filter predicates,
nil aggregate functions, nil config transformers/writers) rather than crashing.

## Investigation Summary

- **Symptoms examined:** Nil-pointer panic in `sortTransformers` (`Priority()`)
  and in the `Name()`/`CanTransform()`/`Transform()` calls of the pipeline
  methods.
- **Code inspected:** `v2/transformer.go` (`Add`, `Remove`, `Has`, `Get`,
  `Transform`, `Info`, `sortTransformers`), `v2/format_aware.go`
  (`NewFormatAwareTransformer` and the `FormatAwareTransformer` methods),
  `v2/output.go` (`validateConfigEntries`), `v2/operations.go`
  (`FilterOp.Validate`, `GroupByOp.Validate`) for the existing nil-validation
  convention.
- **Hypotheses tested:** Confirmed the panic originates from storing/wrapping a
  nil `Transformer` interface value, not from any data-dependent path.

## Discovered Root Cause

Missing input validation. `TransformPipeline.Add` appended the transformer
without a nil check, and `NewFormatAwareTransformer` wrapped without one. The
pipeline's read methods then assumed every stored element was non-nil.

**Defect type:** Missing validation (nil dereference).

**Why it occurred:** `Add` has a void signature, so unlike the validating
operations (`FilterOp.Validate`, `GroupByOp.Validate`) and `validateConfigEntries`
it had no channel to report invalid input — the nil case was never handled.

**Contributing factors:** The `Transformer` interface is consumed in several
methods, each of which independently trusted that stored values were non-nil.

## Resolution for the Issue

**Changes made:**
- `v2/transformer.go` — `Add` now ignores nil transformers (never stores them).
  `sortTransformers`, `Remove`, `Has`, `Get`, `Transform`, and `Info` skip any
  nil entries defensively so the pipeline cannot panic even if a nil slips in.
- `v2/format_aware.go` — `NewFormatAwareTransformer(nil)` returns nil instead of
  wrapping a nil transformer.

**Approach rationale:** Rejecting/skipping nil at insertion and wrapping time is
the cleanest fix and matches the "skip them consistently" option from the ticket.
`Add` is a void method used by many call sites, so silently skipping nil keeps the
public signature stable while guaranteeing nothing nil is ever stored. The
defensive skips in the iteration methods are belt-and-suspenders.

**Alternatives considered:**
- Change `Add` to return an error — rejected: breaks the existing void signature
  and every current call site, and T-1131 explicitly allows the skip-consistently
  option.
- Panic with a clearer message — rejected: the rest of the API avoids panics for
  invalid input.

## Regression Test

**Test files:** `v2/transformer_test.go`, `v2/format_aware_test.go`
**Test names:** `TestTransformPipeline_AddNil`, `TestNewFormatAwareTransformer_Nil`

**What it verifies:** Adding nil transformers does not store them and does not
panic across `Info`/`Transform`/`Has`/`Get`/`Remove`; wrapping a nil transformer
with `NewFormatAwareTransformer` returns nil.

**Run command:**
`go test -run 'TestTransformPipeline_AddNil|TestNewFormatAwareTransformer_Nil' ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/transformer.go` | Skip nil transformers in `Add` and all read/iterate methods |
| `v2/format_aware.go` | `NewFormatAwareTransformer(nil)` returns nil |
| `v2/transformer_test.go` | Add `TestTransformPipeline_AddNil` regression test |
| `v2/format_aware_test.go` | Add `TestNewFormatAwareTransformer_Nil` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed red (failing/panicking) state before the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Validate nil interface inputs at the boundary where they enter a collection.
- When a setter cannot return an error, decide explicitly between skipping and
  documenting the behaviour rather than storing invalid values.

## Related

- Transit ticket T-1131
