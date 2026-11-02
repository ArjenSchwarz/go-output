# Bugfix Report: Renderer Config Transformers

**Date:** 2026-11-02
**Status:** Fixed
**Tickets:** T-1718, T-2322 (RendererConfig part only; `NewSortOp`/`NewGroupByOp` belong to T-1467)

## Description of the Issue

`RendererConfig` carries `DataTransformers` and `ByteTransformers`, and the
`New*RendererWithCollapsible` constructors copy it into `baseRenderer.config`
(T-1223). Only `baseRenderer.renderDocumentWithFormat` read those fields, and
only buffered HTML (`htmlRenderer.Render`) used that path:

- `markdownRenderer.Render` (and `RenderTo`, which delegates to the same
  buffered code) called `renderDocumentMarkdown` directly, so neither kind of
  transformer ran for Markdown.
- `htmlRenderer.RenderTo` used `baseRenderer.renderDocumentTo`, which applied
  per-content operations but ignored the config, so streaming HTML differed
  from buffered HTML for the same renderer.

Separately (T-2322), the RendererConfig-taking constructors stored the config
by value without cloning its reference-typed fields. `HTMLCSSClasses` (map),
`DataTransformers` (slice) and `ByteTransformers` (`*TransformPipeline`)
stayed aliased to the caller, so changing or reusing a config after
construction changed existing renderers. `DefaultRendererConfig` itself holds
one shared map and one shared pipeline, so `cfg := DefaultRendererConfig;
cfg.ByteTransformers.Add(x)` affected every existing renderer built from the
default. (The fix insulates constructed renderers; `DefaultRendererConfig`
still holds one map and pipeline, so copies of it must assign fresh values
before mutating them.)

**Reproduction steps:**
1. Build a `RendererConfig` with a filtering data transformer and a suffix byte
   transformer limited to `markdown` (or `html`).
2. Call `NewMarkdownRendererWithCollapsible(cfg).Render(...)`, or
   `NewHTMLRendererWithCollapsible(cfg).RenderTo(...)`.
3. Observe the unfiltered rows and no suffix.

**Impact:** High for T-1718: configured filtering or value transformations were
silently omitted depending on format or on Render vs RenderTo. Medium for
T-2322: reusing a config could change already-constructed renderers.

## Investigation Summary

- **Symptoms examined:** transformer call counts and rendered output on the
  Markdown `Render`/`RenderTo` and HTML `RenderTo` paths.
- **Code inspected:** `v2/base_renderer.go` (`renderDocumentWithFormat`,
  `renderDocumentTo`, constructors), `v2/markdown_renderer.go`,
  `v2/html_renderer.go`, `v2/transformer.go` (`TransformPipeline`),
  `v2/csv_renderer.go`, `v2/table_renderer.go`.
- **Hypotheses tested:** whether another layer (Output-level transformers)
  applied the config for these paths — it does not; Output-level transformers
  are separate and unaffected.

## Discovered Root Cause

**Defect type:** Missing integration / aliasing of reference-typed fields.

**Why it occurred:** The dual transformer system was implemented only in the
buffered helper. Markdown has its own document renderer (front matter, ToC)
and never routed through it; the streaming helper `renderDocumentTo` was
written without the transformer steps (it took an RLock on config but never
used it). T-1223 fixed the constructors to populate `baseRenderer.config` but
only verified buffered HTML. The constructors copy a struct value, which does
not copy maps, slices or pointed-to pipelines.

## Resolution for the Issue

**Changes made:**
- `v2/base_renderer.go` — `renderDocumentTo` now delegates to the new
  `renderDocumentToWithFormat`, which applies `DataTransformers` before
  streaming. If `ByteTransformers` has transformers, the content is buffered,
  passed through the pipeline and then written; otherwise it streams directly
  as before. Content streaming moved into `streamContents`.
- `v2/html_renderer.go` — `RenderTo` uses `renderDocumentToWithFormat(...,
  FormatHTML)`. Byte transformers apply to the rendered content only, before
  the Mermaid script and template footer, matching `Render`.
- `v2/markdown_renderer.go` — `Render` applies data transformers (format
  `markdown`), renders, then applies byte transformers to the full output;
  `RenderTo` delegates to `Render`.
- `v2/base_renderer.go` — new `cloneRendererConfig` clones `HTMLCSSClasses`,
  `DataTransformers` and `ByteTransformers`; all four
  `New*RendererWithCollapsible` constructors use it.
- `v2/transformer.go` — unexported `TransformPipeline.clone()` (snapshot of the
  transformer list; transformers themselves are shared).

**Approach rationale:** Mirrors the buffered path's three steps and error
wrapping (`data transformation failed` / `byte transformation failed`) so all
paths that accept a `RendererConfig` behave the same. Buffering only when byte
transformers exist keeps true streaming for the common case.

**Alternatives considered:**
- Rejecting byte transformers on `RenderTo` — would contradict the advertised
  `RendererConfig` contract while the buffered path supports them.
- Cloning `DataTransformers` but leaving `ByteTransformers` shared — leaves the
  same aliasing through the shared `DefaultRendererConfig` pipeline.

**Out of scope (unchanged):** table and CSV renderers still do not apply
`RendererConfig` transformers; this is documented on their constructors and
was not broadened. `NewStreamingValueProcessor`/`NewMemoryOptimizedProcessor`
only read scalar fields and were left alone.

## Regression Test

**Test file:** `v2/renderer_config_transformer_test.go`
**Test names:**
- `TestNewMarkdownRendererWithCollapsible_AppliesTransformers` (Render, RenderTo)
- `TestNewHTMLRendererWithCollapsible_RenderToAppliesTransformers` (also checks RenderTo == Render)
- `TestHTMLRenderTo_DataTransformerOnly`
- `TestRendererConstructors_DoNotAliasCallerConfig` (markdown, html)
- `TestNewHTMLRendererWithCollapsible_DoesNotAliasCSSClasses`
- `TestCloneRendererConfig_NilFields`

**What it verifies:** transformers configured via `RendererConfig` run on the
Markdown and streaming HTML paths, and post-construction mutation of the
caller's config does not affect the renderer.

**Run command:** `cd v2 && go test -run 'AppliesTransformers|RenderToAppliesTransformers|DataTransformerOnly|DoNotAlias|CloneRendererConfig' .`

## Affected Files

| File | Change |
|------|--------|
| `v2/base_renderer.go` | Streaming transformer support; config cloning in constructors |
| `v2/html_renderer.go` | `RenderTo` uses the format-aware streaming helper |
| `v2/markdown_renderer.go` | `Render` applies data/byte transformers |
| `v2/transformer.go` | `TransformPipeline.clone()` |
| `v2/renderer_config_transformer_test.go` | Regression tests |
| `CHANGELOG.md` | Entry |

## Verification

**Automated:**
- [x] Regression tests failed before the fix and pass after
- [x] Full test suite passes, except two pre-existing failures that also fail on
  main in this environment (`TestFileWriter_AppendErrorHandling`,
  `TestFileWriter_AppendIOErrorsIncludeContext` rely on file permissions and the
  container runs as root)
- [x] `make lint` — 0 issues; `make doc-examples` passes; targeted `-race` run passes

## Prevention

- New render paths that accept `RendererConfig` should go through
  `renderDocumentWithFormat` / `renderDocumentToWithFormat`.
- Constructors taking structs with maps, slices or mutable pointers should
  clone them at the boundary (see `cloneDrawIOHeader`, `cloneRendererConfig`).

## Related

- T-1223 (constructors populate `baseRenderer.config`), T-1438 (nil adapters),
  T-1371 (same aliasing shape for `DrawIOHeader`), T-1467 (operation constructors)
