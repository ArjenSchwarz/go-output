# Bugfix Report: RenderTo Implementations Panic On Nil Writers

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

Several renderer `RenderTo` methods write to the `io.Writer` argument without
first checking whether it is nil. After rendering succeeds, the final
`w.Write(data)` (or, for HTML with templating, an earlier header write) panics
with a nil pointer dereference instead of returning an error.

This was inconsistent with `baseRenderer.renderDocumentTo` and the CSV streaming
renderer, both of which explicitly return `writer cannot be nil`.

**Reproduction steps:**
1. Build any document.
2. Call a renderer's `RenderTo(ctx, doc, nil)` (e.g. JSON, YAML, Markdown, Table, DOT, Mermaid, Draw.io, or HTML with a template).
3. Observe a `runtime error: invalid memory address or nil pointer dereference` panic.

**Impact:** Medium. A nil writer is a caller programming error, but the library
should fail gracefully with a clear error rather than panicking. The behaviour
was inconsistent across renderers (CSV returned an error; others panicked).

## Investigation Summary

- **Symptoms examined:** Nil pointer panic from `w.Write` after a successful render.
- **Code inspected:** Every public `RenderTo` in `v2/json_yaml_renderer.go`, `v2/markdown_renderer.go`, `v2/table_renderer.go`, `v2/graph_renderers.go`, `v2/html_renderer.go`, plus the reference guards in `v2/base_renderer.go` and `v2/csv_renderer.go`.
- **Hypotheses tested:** Confirmed via a regression test that JSON, YAML, Markdown, Table, DOT, Mermaid, Draw.io, and HTML-with-template panic, while CSV and HTML-without-template already return the error (CSV has its own guard; HTML-without-template delegates straight to `renderDocumentTo`, which guards).

## Discovered Root Cause

The affected `RenderTo` methods render to a byte slice and then unconditionally
call `w.Write(...)`. None of them check `w == nil` first. `htmlRenderer.RenderTo`
additionally writes a template header to `w` before delegating to the guarded
`renderDocumentTo`, so it panics earlier when template output is enabled.

**Defect type:** Missing input validation (nil guard).

**Why it occurred:** The nil-writer guard lived only in `renderDocumentTo` and
the CSV renderer. Renderers that build their output independently and write it in
one shot never inherited that guard.

**Contributing factors:** Each renderer implements `RenderTo` separately, so the
guard was easy to omit in the renderers that do not route through
`renderDocumentTo`.

## Resolution for the Issue

**Changes made:**
- `v2/json_yaml_renderer.go` - Added `if w == nil` guard at the top of `jsonRenderer.RenderTo` and `yamlRenderer.RenderTo`.
- `v2/markdown_renderer.go` - Added the guard at the top of `markdownRenderer.RenderTo`.
- `v2/table_renderer.go` - Added the guard at the top of `tableRenderer.RenderTo`.
- `v2/graph_renderers.go` - Added the guard at the top of `dotRenderer.RenderTo`, `mermaidRenderer.RenderTo`, and `drawioRenderer.RenderTo`.
- `v2/html_renderer.go` - Added the guard at the top of `htmlRenderer.RenderTo`, before the template header write.

**Approach rationale:** Match the existing pattern. Each guard returns
`fmt.Errorf("writer cannot be nil")`, identical to `renderDocumentTo` and the CSV
renderer, and runs before any write so no partial output is produced.

**Alternatives considered:**
- Wrap all writes in a shared helper - Rejected as larger than necessary; the renderers build output differently and a one-line guard per method is clearer and lower risk.

## Regression Test

**Test file:** `v2/renderer_nil_writer_test.go`
**Test name:** `TestRenderTo_NilWriter`, `TestHTMLRenderer_NilWriterWithTemplate`

**What it verifies:** Every built-in renderer's `RenderTo` returns an error
containing `writer cannot be nil` (and does not panic) when passed a nil writer.
The HTML-with-template case is covered separately because it panics on an earlier
write than the other renderers.

**Run command:** `cd v2 && go test . -run 'TestRenderTo_NilWriter|TestHTMLRenderer_NilWriterWithTemplate' -v`

## Affected Files

| File | Change |
|------|--------|
| `v2/json_yaml_renderer.go` | Nil-writer guard in JSON and YAML `RenderTo` |
| `v2/markdown_renderer.go` | Nil-writer guard in Markdown `RenderTo` |
| `v2/table_renderer.go` | Nil-writer guard in Table `RenderTo` |
| `v2/graph_renderers.go` | Nil-writer guard in DOT, Mermaid, Draw.io `RenderTo` |
| `v2/html_renderer.go` | Nil-writer guard in HTML `RenderTo` before header write |
| `v2/renderer_nil_writer_test.go` | New regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the regression tests fail (panic) before the fix and pass after.

## Prevention

**Recommendations to avoid similar bugs:**
- Validate `io.Writer` arguments at the top of every public method that writes to them.
- Keep nil-argument behaviour consistent across renderers (return an error, never panic).

## Related

- Transit ticket T-1120
