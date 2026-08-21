# Bugfix Report: CSV Renderer Panics When Operation Returns Nil Content

**Ticket:** T-1601
**Date:** 2026-08-17
**Status:** Fixed

## Description of the Issue

A custom per-content `Operation` attached via `WithTransformations()` can
return `(nil, nil)` from `Apply` — nil content with a nil error.
`applyContentTransformations` in `v2/renderer.go` accepted that result and
assigned it to the running content without validation, so the CSV renderer
received a nil `Content`. The nil interface matched no concrete case in the
renderer's type switch, fell to the default branch, and
`content.AppendText(nil)` panicked with a nil pointer dereference
(`v2/csv_renderer.go:141`). In the nested-section path
(`renderSectionTablesCSV`) the nil was silently dropped instead of erroring.

**Reproduction steps:**
1. Build a document with a table carrying a transformation whose `Apply` returns `(nil, nil)`:
   `New().Table("t", records, WithKeys(...), WithTransformations(op)).Build()`
2. Render it with the CSV renderer: `(&csvRenderer{}).Render(ctx, doc)`
3. Observe a runtime panic: `invalid memory address or nil pointer dereference`

**Impact:** Any application using a custom `Operation` that can return a nil
result crashes at render time instead of receiving an error. The defect lives
in the shared helper `applyContentTransformations`, so every renderer (CSV,
JSON, YAML, HTML, Markdown, Table, DOT, Mermaid, Draw.io) was exposed to nil
Content propagation; CSV was the observed panic site.

## Investigation Summary

Systematic inspection (Fagan-style walkthrough) of the transformation and CSV
render paths.

- **Symptoms examined:** nil pointer dereference panic during CSV rendering of content with a `(nil, nil)`-returning operation; silent content drop in the nested-section path.
- **Code inspected:** `v2/renderer.go` (`applyContentTransformations`), `v2/csv_renderer.go` (top-level type switch and `renderSectionTablesCSV`), `v2/pipeline.go` (`Operation` interface), `v2/base_renderer.go` (T-1438 precedent in `applyDataTransformers`).
- **Hypotheses tested:** confirmed the nil originates from the unvalidated `op.Apply` result at `renderer.go:210-215`; ruled out the nil-operation case (already skipped since T-1208) and the `DataTransformers` path (already guarded since T-1438).

## Discovered Root Cause

`applyContentTransformations` assigned the result of `op.Apply(ctx, current)`
to `current` after checking only the error. A contract-breaking operation
returning `(nil, nil)` therefore produced a `(nil, nil)` return from the
helper, and renderer-specific code dereferenced the nil Content.

**Defect type:** Missing validation at a trust boundary.

**Why it occurred:** The `Operation.Apply` contract (`v2/pipeline.go`) never
documented that a nil error implies non-nil Content, and the helper implicitly
trusted implementers. `Operation` is a public interface implementable by user
code, so the library must validate the invariant itself.

**Contributing factors:** The parallel `DataTransformer` path had the same
defect, fixed in T-1438; the per-content `Operation` path was not covered by
that fix.

## Resolution for the Issue

**Changes made:**
- `v2/renderer.go` (`applyContentTransformations`) - reject a nil result from `op.Apply` when the error is nil, returning a transformation error that names the content and operation: `content %s transformation %d (%s) returned nil content`
- `v2/pipeline.go` (`Operation` interface) - document the contract that `Apply` must return non-nil Content when the error is nil

**Approach rationale:** The ticket calls for central rejection in
`applyContentTransformations` so every renderer is protected by one guard.
The error message follows the existing error style in the same function
(`content %s transformation %d (%s) ...`) and mirrors the T-1438 guard's
"returned nil content" phrasing for consistency.

**Alternatives considered:**
- Guarding in each renderer's type switch (e.g., a `case nil` branch) - rejected: duplicates the check across nine renderers and leaves future renderers exposed; the ticket explicitly asks for a central fix.
- Skipping the offending operation and continuing with the previous content - rejected: silently ignoring a broken operation hides bugs in user code; an explicit error matches the T-1438 precedent.

## Regression Test

**Test file:** `v2/csv_renderer_nil_operation_test.go`
**Test names:**
- `TestApplyContentTransformationsNilOperationResult` - central guard returns an error naming the operation
- `TestApplyContentTransformationsNilResultStopsChain` - nil result is not passed to subsequent operations
- `TestCSVRendererNilOperationResult` - CSV render returns an error instead of panicking (reproduces the original panic before the fix)
- `TestCSVRendererNilOperationResultInSection` - nested-section path surfaces the error instead of silently dropping the content

**What it verifies:** an `Operation` returning `(nil, nil)` produces a normal
transformation error from `applyContentTransformations` before any
renderer-specific code sees the nil Content.

**Run command:** `cd v2 && go test -run 'TestApplyContentTransformationsNilOperationResult|TestApplyContentTransformationsNilResultStopsChain|TestCSVRendererNilOperationResult|TestCSVRendererNilOperationResultInSection' ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/renderer.go` | Nil-result guard in `applyContentTransformations` |
| `v2/pipeline.go` | Document `Operation.Apply` non-nil contract |
| `v2/csv_renderer_nil_operation_test.go` | New regression tests |
| `CHANGELOG.md` | Entry under Unreleased / Fixed |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the pre-fix panic via `TestCSVRendererNilOperationResult` (nil pointer dereference at the type switch default branch) before implementing the guard.

## Prevention

**Recommendations to avoid similar bugs:**
- When a public interface's return value feeds internal dereferences, validate the nil-error-implies-non-nil-result invariant at the boundary rather than trusting implementers.
- Document such invariants on the interface method itself (done here for `Operation.Apply`, previously for `DataTransformer.TransformData` in T-1438).

## Related

- T-1438 (merged, PR #111): same defect class in the `RendererConfig.DataTransformers` path; error style mirrored here.
- T-1208: nil `Operation` entries skipped defensively in the same loop.
- T-1448 (parallel): table renderer fallback rendering pre-transform content; this central guard makes the table renderer's nil-fallback unreachable for the nil case. `v2/table_renderer.go` deliberately left untouched.
