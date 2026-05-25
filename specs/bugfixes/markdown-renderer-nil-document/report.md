# Bugfix Report: Markdown Renderer Panics On Nil Documents

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

The Markdown renderer panicked with a nil pointer dereference when given a nil
document, instead of returning a clean error like the other renderers do.

**Reproduction steps:**
1. Construct the Markdown renderer: `output.Markdown().Renderer`
2. Call `Render(ctx, nil)` (or `RenderTo(ctx, nil, w)`)
3. Observe a `runtime error: invalid memory address or nil pointer dereference`
   panic instead of a returned error.

**Impact:** Any caller passing a nil document to the Markdown renderer crashes
the process. Every other renderer (JSON/YAML, table, CSV, base) returns
`document cannot be nil`, so this was an inconsistency that turned a recoverable
error into a panic.

## Investigation Summary

- **Symptoms examined:** Panic stack trace pointing at `document.go:18`
  (`(*Document).GetContents`) called from `markdown_renderer.go:66`.
- **Code inspected:** `v2/markdown_renderer.go` (`Render`, `RenderTo`,
  `renderDocumentMarkdown`, `generateTableOfContents`); compared against the nil
  guards in `json_yaml_renderer.go`, `table_renderer.go`, `csv_renderer.go`, and
  `base_renderer.go`.
- **Hypotheses tested:** Confirmed the dereference happens in
  `renderDocumentMarkdown` — both via `doc.GetContents()` in the render loop and
  via `generateTableOfContents(doc)` on the ToC path, which can run earlier when
  `includeToC` is set.

## Discovered Root Cause

`renderDocumentMarkdown` dereferenced `doc` (through `doc.GetContents()` and
`m.generateTableOfContents(doc)`) without first checking for nil. Both the
`Render` and `RenderTo` entry points funnel through this single helper, so
neither path was guarded.

**Defect type:** Missing input validation (nil check).

**Why it occurred:** The other renderers guard nil at the top of their
render-document helper; the Markdown renderer was written without the equivalent
guard, so the established convention was not applied consistently.

**Contributing factors:** The dereference is reachable from two code paths (the
ToC generation and the main content loop), which made the missing guard easy to
overlook.

## Resolution for the Issue

**Changes made:**
- `v2/markdown_renderer.go:44` - Added `if doc == nil { return nil,
  fmt.Errorf("document cannot be nil") }` at the start of
  `renderDocumentMarkdown`, before any `doc` dereference.

**Approach rationale:** Placing the guard at the start of the shared helper
covers both `Render` and `RenderTo` (which both delegate to it) and both
dereference paths (ToC and main loop) with a single check. The error message
matches the wording used by every other renderer for consistency.

**Alternatives considered:**
- Guarding in `Render` and `RenderTo` separately - Rejected: duplicates the
  check and risks future entry points missing it; the shared helper is the
  single choke point.

## Regression Test

**Test file:** `v2/renderer_markdown_test.go`
**Test names:** `TestMarkdownRenderer_NilDocument`,
`TestMarkdownRenderer_NilDocumentWithToC`,
`TestMarkdownRenderer_RenderToNilDocument`

**What it verifies:** `Render` and `RenderTo` return a `document cannot be nil`
error (and produce no output) instead of panicking when passed a nil document,
including the ToC-enabled path.

**Run command:**
`go test -run 'TestMarkdownRenderer_NilDocument|TestMarkdownRenderer_RenderToNilDocument' ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/markdown_renderer.go` | Added nil-document guard in `renderDocumentMarkdown` |
| `v2/renderer_markdown_test.go` | Added three nil-document regression tests |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed tests panic (red) before the fix and pass (green) after.

## Prevention

**Recommendations to avoid similar bugs:**
- New renderers should add the `document cannot be nil` guard at the start of
  their render-document helper, matching the existing renderers.
- Consider a shared helper or interface-level contract test that asserts every
  renderer returns an error (not a panic) for a nil document.

## Related

- Transit ticket T-1119
- Similar prior fix: T-1184 (validate nil configuration entries in
  `Output.Render`)
