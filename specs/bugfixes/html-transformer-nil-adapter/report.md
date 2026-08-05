# Bugfix Report: HTML Data Transformer Path Panics On Nil Adapter Results

**Ticket:** T-1438
**Date:** 2026-08-03
**Status:** Fixed

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

**Impact:** Any renderer that routes through `renderDocumentWithFormat` — the
HTML renderer, and the Mermaid/Draw.io table-content fallback paths — crashes
the calling process instead of returning an error. `RendererConfig` is
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
  (T-1208) — the data transformer path was the remaining unguarded entry.

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

**Changes made:**
- `v2/base_renderer.go` (`applyDataTransformers`, adapter filter loop) — skip
  nil `*TransformerAdapter` entries before calling `IsDataTransformer`,
  matching the existing skip of adapters that are not data transformers.
- `v2/base_renderer.go` (`applyDataTransformers`, transform loop) — after a
  successful `TransformData` call, return
  `transformer %s returned nil content for content %s` when the returned
  `Content` is nil, before the transformed document is constructed. The error
  surfaces through the existing `data transformation failed: %w` wrapper in
  `renderDocumentWithFormat`.

**Approach rationale:** Skipping nil adapter entries is consistent with the
established defensive pattern in this codebase (`applyContentTransformations`
skips nil operations, `AsDataTransformer() == nil` entries are skipped one line
below the panic site) — a nil adapter transforms nothing, so it is inert
configuration. A nil transformed content, by contrast, signals a broken
transformer contract, so it becomes a normal error rather than being silently
dropped or passed through.

**Alternatives considered:**
- Returning a validation error for nil adapter entries — rejected for
  consistency with how the surrounding code treats inert entries.
- Validating `DataTransformers` in the renderer constructors — rejected because
  constructors like `NewHTMLRendererWithCollapsible` return the `Renderer`
  interface with no error channel, and config can also be set after
  construction.
- Passing nil content through unchanged (treat as "no transformation") —
  rejected because it would mask transformer bugs.

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

**Run command:** `cd v2 && go test -run TestApplyDataTransformersNil ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/base_renderer.go` | Nil checks in `applyDataTransformers` for adapter entries and transformed content |
| `v2/base_renderer_nil_transformer_test.go` | New regression tests |
| `CHANGELOG.md` | Fixed entry under Unreleased |

## Verification

**Automated:**
- [x] Regression test passes (`cd v2 && go test -run TestApplyDataTransformersNil ./...`)
- [x] Full test suite passes (`go test ./...` in v2; the sandboxed
  `TestDocumentationExamplesCompile` failure is a known sandbox artifact and
  passes unsandboxed)
- [x] Linters/validators pass (`make lint` — 0 issues; `gofmt -l` clean)

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

- T-1208 — nil `Operation` skip in `applyContentTransformations`
- T-1142 — nil content guards in table operations
- T-1363 — nil nested content in collapsible sections
- T-1223 — collapsible constructors populating `baseRenderer.config`, which
  made this path reachable from public constructors
