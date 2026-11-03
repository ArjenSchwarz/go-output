# Bugfix Report: Markdown Front Matter Constructors Alias Caller Map

**Ticket:** T-1407
**Date:** 2026-11-02
**Status:** Fixed

## Description of the Issue

`NewMarkdownRendererWithFrontMatter` and `NewMarkdownRendererWithOptions`
stored the caller's `map[string]string` directly. The public format helpers
`MarkdownWithFrontMatter` and `MarkdownWithOptions` use these constructors, so
any later change to the caller's map changed the YAML front matter of every
subsequent render from that renderer/format.

**Reproduction steps:**
1. `fm := map[string]string{"title": "First Report"}`
2. `f := output.MarkdownWithFrontMatter(fm)`
3. `fm["title"] = "Second Report"` (reusing the map for another report)
4. Render a document with `f` — the front matter says `Second Report`.

**Impact:** Medium. Reusing a metadata map silently changes earlier
renderers. Mutating the map while a render iterates it is also a data race.

## Investigation Summary

- **Code inspected:** both constructors in `v2/markdown_renderer.go`,
  `withCompatOptions` (copies only when merging Output-level entries), and
  `Output.WithFrontMatter` in `v2/output.go` (already copies via `maps.Copy`).
- **Hypotheses tested:** `withCompatOptions` returns `m.frontMatter` unchanged
  when no Output-level front matter is set, so the alias also reaches renders
  done through `Output`.

## Discovered Root Cause

**Defect type:** Shared mutable state (missing defensive copy).

**Why it occurred:** The constructors assign the parameter straight into the
struct; `Output.WithFrontMatter` was written with a copy but the renderer
constructors were not.

## Resolution for the Issue

**Changes made:**
- `v2/markdown_renderer.go` - both constructors store `maps.Clone(frontMatter)`.

**Approach rationale:** Copying at construction matches `Output.WithFrontMatter`
and makes the renderer's front matter immutable afterwards (nothing else
writes to it). `maps.Clone(nil)` returns nil, so the no-front-matter case is
unchanged.

**Alternatives considered:**
- Copy at render time - rejected; still races with concurrent caller writes
  and does not give construction-time snapshot semantics.

## Regression Test

**Test file:** `v2/renderer_markdown_regression_test.go`
**Test name:** `TestMarkdownRenderer_FrontMatterNotAliased`

**What it verifies:** For both constructors and both format helpers, mutating
(changing, adding, deleting keys in) the original map after construction does
not change the rendered front matter.

**Run command:** `cd v2 && go test -run TestMarkdownRenderer_FrontMatterNotAliased .`

## Affected Files

| File | Change |
|------|--------|
| `v2/markdown_renderer.go` | Clone front matter in constructors |
| `v2/renderer_markdown_regression_test.go` | Regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

- Constructors that accept maps or slices should clone them unless sharing is
  documented.

## Related

- T-1338 (deterministic front matter key order, done)
