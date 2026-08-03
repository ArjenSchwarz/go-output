# Bugfix Report: HTML Data Transformer Path Panics On Nil Adapter Results

**Ticket:** T-1438
**Date:** 2026-08-03
**Status:** In Progress

## Description of the Issue

`baseRenderer.applyDataTransformers` in `v2/base_renderer.go` panicked in two
ways on the data transformer path:

1. It iterated `RendererConfig.DataTransformers` and immediately called
   `adapter.IsDataTransformer()` without checking whether the
   `*TransformerAdapter` entry was nil. `IsDataTransformer` dereferences the
   receiver's `transformer` field, so a nil entry caused a nil pointer
   dereference panic.
2. It appended the result of `DataTransformer.TransformData` without checking
   whether the returned `Content` was nil. A custom transformer returning
   `(nil, nil)` put a nil `Content` into the transformed `Document`, and the
   next render stage panicked in `applyContentTransformations` when calling
   `content.GetTransformations()` on the nil interface.

**Reproduction steps:**

1. Build a document with a table (`New().Table(...).Build()`).
2. Create a renderer via the public constructor
   `NewHTMLRendererWithCollapsible(RendererConfig{DataTransformers: []*TransformerAdapter{nil}})`.
3. Call `renderer.Render(ctx, doc)` — panic at `v2/base_renderer.go:135`
   (`IsDataTransformer` on nil adapter).
4. Alternatively, register a transformer whose `TransformData` returns
   `(nil, nil)` — panic at `v2/renderer.go:180`
   (`applyContentTransformations` on nil content).

**Impact:** Any renderer that embeds `baseRenderer` and routes through
`renderDocumentWithFormat` (HTML, Markdown with collapsible config, JSON, YAML)
crashes the calling process instead of returning an error. `RendererConfig` is
a public struct, so callers can construct the nil-adapter case directly without
going through `NewTransformerAdapter`.

## Investigation Summary

- **Symptoms examined:** Nil pointer dereference panics reproduced with
  regression tests: `IsDataTransformer` panic at `v2/transform_data.go:63` via
  `v2/base_renderer.go:135`, and nil-content panic at `v2/renderer.go:180` via
  `v2/base_renderer.go:106`.
- **Code inspected:** `v2/base_renderer.go` (`renderDocumentWithFormat`,
  `applyDataTransformers`, `renderTransformedDocument`),
  `v2/transform_data.go` (`TransformerAdapter`), `v2/renderer.go`
  (`applyContentTransformations`), `v2/html_renderer.go` (`Render`).
- **Hypotheses tested:** Confirmed both panic paths with failing tests before
  changing any code. Verified `applyByteTransformers` already guards against a
  nil pipeline, and `applyContentTransformations` already skips nil operations
  (T-1142) — the data transformer path was the remaining unguarded entry.

## Discovered Root Cause

**Defect type:** Missing validation (nil checks) on externally supplied data.

**Why it occurred:** The transformer loop assumed adapters are only created via
`NewTransformerAdapter` (never nil) and that any `DataTransformer` returning a
nil error also returns non-nil content. Neither assumption is enforced:
`RendererConfig` is a public struct whose `DataTransformers` slice callers
populate directly, and `DataTransformer` is a public interface implemented by
user code.

**Contributing factors:** Sibling code paths already handled the equivalent
cases (`applyByteTransformers` guards nil pipelines,
`applyContentTransformations` skips nil operations, `AsDataTransformer` nil
result is skipped), which made the two remaining gaps easy to miss.

## Resolution for the Issue

_To be filled in after the fix is implemented._

## Regression Test

**Test file:** `v2/base_renderer_nil_transformer_test.go`
**Test names:** `TestApplyDataTransformersNilAdapterEntry`,
`TestApplyDataTransformersNilTransformedContent`

**What it verifies:**
- A `DataTransformers` slice containing only a nil adapter renders
  successfully, and a nil entry alongside a valid transformer still applies the
  valid transformer.
- A transformer returning `(nil, nil)` produces a render error that names the
  transformer instead of panicking.

**Run command:** `go test -run 'TestApplyDataTransformersNil' ./v2/...`

## Affected Files

| File | Change |
|------|--------|
| `v2/base_renderer.go` | Nil checks in `applyDataTransformers` for adapter entries and transformed content |
| `v2/base_renderer_nil_transformer_test.go` | New regression tests |
| `CHANGELOG.md` | Fixed entry under Unreleased |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- Reproduced both panics with the regression tests before the fix (red phase),
  confirming the exact stack traces from the ticket.

## Prevention

**Recommendations to avoid similar bugs:**
- When iterating externally supplied slices of pointers or interface values,
  nil-check each element before invoking methods on it.
- When a pipeline stage produces a value consumed by later stages, validate the
  produced value (non-nil) at the boundary instead of trusting the producer.

## Related

- T-1142 — nil content into table operations (nil `Operation` skip in
  `applyContentTransformations`)
- T-1363 — nil nested content in collapsible sections
- T-1223 — collapsible constructors populating `baseRenderer.config`, which
  made this path reachable from public constructors
