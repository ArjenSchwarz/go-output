# Bugfix Report: Table Transformations Panic On Nil Operations

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

`WithTransformations` stored the provided `Operation` values without filtering or
validation. During rendering, `applyContentTransformations` iterated the slice and
called `op.Validate()`, `op.Name()`, and `op.Apply(...)` without checking whether
`op` was nil. A caller passing `WithTransformations(nil)` therefore caused direct
renderer use to panic with a nil-pointer dereference instead of returning normally
or producing a recoverable error.

**Reproduction steps:**
1. `doc := output.New().Table("t", data, output.WithTransformations(nil)).Build()`
2. `renderer := &jsonRenderer{}` (or any renderer that runs transformations)
3. `renderer.Render(ctx, doc)` — panics with SIGSEGV at `renderer.go:197`.

**Impact:** A nil operation passed by a caller crashes the whole process instead of
being handled gracefully. This is inconsistent with the rest of the transformation
API, which skips nil inputs (e.g. `TransformPipeline.Add` after T-1131) rather than
crashing.

## Investigation Summary

- **Symptoms examined:** Nil-pointer panic in `applyContentTransformations` at the
  `op.Validate()` call (`renderer.go:197`), reached via `renderDocumentGeneric`.
- **Code inspected:** `v2/renderer.go` (`applyContentTransformations`),
  `v2/table_options.go` (`WithTransformations`), `v2/pipeline.go` (`Operation`
  interface), and the sibling fix for T-1131 (`v2/transformer.go`) for the
  established "skip nil consistently" convention.
- **Hypotheses tested:** Confirmed the panic originates from storing a nil
  `Operation` interface value, not from any data-dependent path. The loop assumed
  every stored element was non-nil.

## Discovered Root Cause

Missing input validation. `WithTransformations` appended the operations verbatim,
and `applyContentTransformations` assumed every stored element was a usable
non-nil `Operation`. Calling any interface method on a nil interface value
dereferences a nil pointer and panics.

**Defect type:** Missing validation (nil dereference).

**Why it occurred:** `WithTransformations` is a functional option with a void
effect, so it had no channel to report invalid input — the nil case was never
handled. `applyContentTransformations` likewise trusted its input.

**Contributing factors:** The `Operation` interface is consumed via multiple method
calls per element (`Validate`, `Name`, `Apply`); none guarded against nil.

## Resolution for the Issue

**Changes made:**
- `v2/table_options.go` — `WithTransformations` filters out nil operations before
  storing them, so a nil never reaches the content's transformation slice.
- `v2/renderer.go` — `applyContentTransformations` defensively skips any nil
  operation in the loop, so the renderer cannot panic even if a nil entry was set
  on the content directly (bypassing the option).

**Approach rationale:** Skipping nil at both the configuration boundary and the
iteration point matches the convention established by T-1131 (`TransformPipeline.Add`
ignores nil; iteration methods skip nil defensively). It keeps `WithTransformations`'
void functional-option signature stable while guaranteeing nothing nil is ever
applied, and the renderer guard is belt-and-suspenders for content constructed by
other paths.

**Alternatives considered:**
- Return a validation/render error from `applyContentTransformations` on nil —
  rejected: the rest of the transformation API skips nil rather than erroring, so
  erroring here would be inconsistent and would surface a "failure" for what is
  effectively a no-op.
- Panic with a clearer message — rejected: the API avoids panics for invalid input.
- Guard only in the renderer — rejected: leaves nil values stored on the content,
  which other consumers of `GetTransformations()` would still have to defend against.

## Regression Test

**Test file:** `v2/renderer_json_test.go`
**Test name:** `TestJSONRenderer_NilTransformation`

**What it verifies:** `WithTransformations(nil)` (single nil, nil mixed with a valid
op, and multiple nils) rendered through the JSON renderer returns normally without
panicking and produces valid JSON output.

**Run command:**
`go test -run 'TestJSONRenderer_NilTransformation' ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/table_options.go` | `WithTransformations` filters out nil operations |
| `v2/renderer.go` | `applyContentTransformations` skips nil operations defensively |
| `v2/renderer_json_test.go` | Add `TestJSONRenderer_NilTransformation` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed red (panicking) state before the fix at `renderer.go:197`.

## Prevention

**Recommendations to avoid similar bugs:**
- Validate/skip nil interface inputs at the boundary where they enter a collection,
  especially for functional options with void effects.
- When iterating a slice of interface values, guard against nil entries before
  calling interface methods.

## Related

- Transit ticket T-1208
- Related nil-handling fixes: T-1131 (TransformPipeline), T-1120 (RenderTo nil writers)
