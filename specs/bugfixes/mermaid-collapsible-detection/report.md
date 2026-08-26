# Bugfix Report: HTML Renderer Omits Mermaid Script for Charts Inside Collapsible Sections

**Ticket:** T-1593
**Date:** 2026-08-22
**Status:** Fixed

## Description of the Issue

When a document's only Mermaid charts live inside a `DefaultCollapsibleSection`, the HTML renderer emits the chart as `<pre class="mermaid">` but omits the Mermaid.js import/initializer script. Browsers therefore display the raw Mermaid source text instead of the rendered chart.

**Reproduction steps:**
1. Build a document containing a collapsible section that wraps a chart:
   ```go
   section := NewCollapsibleSection("Charts", []Content{
       NewPieChart("Status", []PieSlice{{Label: "ok", Value: 1}}, false),
   })
   doc := New().AddContent(section).Build()
   got, _ := HTML().Renderer.Render(context.Background(), doc)
   ```
2. Inspect the output: it contains `<pre class="mermaid">` but no `<script type="module">` block importing mermaid.
3. Open the HTML in a browser — the Mermaid source is shown as preformatted text instead of a chart.

**Impact:** Any HTML output where charts appear only inside collapsible sections (directly, or nested at any depth under a collapsible section) renders unusable chart markup. Both `Render` and the streaming `RenderTo` path are affected. Documents with at least one chart at the top level or inside a plain `SectionContent` are unaffected because detection succeeds via another path.

## Investigation Summary

Systematic inspection (Fagan-style walkthrough) of `v2/html_renderer.go` comparing the render traversal with the mermaid-detection traversal.

- **Symptoms examined:** Rendered output for a chart inside `NewCollapsibleSection` contains the mermaid pre block but no script; the same chart at the top level or in a `SectionContent` gets the script.
- **Code inspected:** `documentContainsMermaidCharts`, `sectionContainsMermaidCharts`, `renderContent` dispatch, `renderCollapsibleSection`, `renderSectionContentHTML`, `Render`/`RenderTo` script-injection call sites, and `GraphContent` handling.
- **Hypotheses tested:**
  - Script injection broken generally — ruled out; top-level charts get the script (existing `TestHTMLRenderer_MermaidScriptInjection` passes).
  - Collapsible renderer not emitting mermaid markup — ruled out; `renderCollapsibleSection` renders nested content through `renderContent`, which produces `<pre class="mermaid">` for `*ChartContent`.
  - `GraphContent` also needing the script — ruled out; the HTML `renderContent` switch has no `*GraphContent` case, so graphs fall through to the escaped `<pre>` fallback without the mermaid class and do not require the script.

## Discovered Root Cause

`documentContainsMermaidCharts` only recognised `*ChartContent` at the top level and recursed exclusively into `*SectionContent` (via `sectionContainsMermaidCharts`). Neither helper inspected `*DefaultCollapsibleSection`, while `renderCollapsibleSection` renders nested charts all the same. The detection traversal had diverged from the render traversal.

**Defect type:** Logic error — missing case in a duplicated traversal.

**Why it occurred:** Collapsible sections were added as a second nested-container type with their own render path. Mermaid detection was written as two near-duplicate, type-specific walkers (document-level and `SectionContent`-level) and neither was updated when collapsible sections learned to render charts, so containers could be rendered that detection never visited.

**Contributing factors:** No shared traversal helper mirroring `renderContent`'s dispatch; each container type re-implements its own recursion, inviting divergence.

## Resolution for the Issue

**Changes made:**
- `v2/html_renderer.go` - Replaced `sectionContainsMermaidCharts` with a shared recursive helper `contentsContainMermaidCharts([]Content) bool` that detects `*ChartContent` and recurses into both `*SectionContent` (`Contents()`) and `*DefaultCollapsibleSection` (`Content()`), in any nesting combination. `documentContainsMermaidCharts` now delegates to it. Nil entries are skipped defensively, consistent with the nil-skip guards in the collapsible render path (T-1472).

**Approach rationale:** A single helper over `[]Content` mirrors the renderer's own dispatch, so every container the renderer descends into is also visited by detection. This removes the duplication that caused the divergence, fixing both `Render` and `RenderTo` (both call `documentContainsMermaidCharts`).

**Alternatives considered:**
- Adding a `*DefaultCollapsibleSection` case to both existing walkers - keeps the duplicated traversals that caused the bug; the next container type would need edits in multiple places.
- Injecting the script whenever the rendered output contains `<pre class="mermaid">` - string-matching rendered output is fragile (false positives from raw/escaped content) and does not work for the streaming `RenderTo` path, which writes content before deciding on the script.

## Regression Test

**Test file:** `v2/mermaid_html_markdown_test.go`
**Test name:** `TestHTMLRenderer_MermaidScriptInjectionCollapsibleSections`

**What it verifies:** For both `Render` and `RenderTo`, the mermaid script is present when a chart is (a) directly inside a collapsible section, (b) inside a `SectionContent` nested in a collapsible section, (c) inside a collapsible section nested in a `SectionContent`, and (d) inside a collapsible section nested in another collapsible section; and the script is absent for a collapsible section without charts.

**Run command:** `cd v2 && go test -run TestHTMLRenderer_MermaidScriptInjectionCollapsibleSections ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/html_renderer.go` | Shared recursive mermaid-chart detection covering collapsible sections |
| `v2/mermaid_html_markdown_test.go` | Regression tests for chart detection in collapsible sections |
| `CHANGELOG.md` | Changelog entry under Unreleased/Fixed |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Rendered output for the repro document (asserted directly in the regression test) now contains both `<pre class="mermaid">` and the mermaid import/initialize script; before the fix the test failure output showed the pre block without the script.

## Prevention

**Recommendations to avoid similar bugs:**
- When a renderer gains a new nested-container content type, audit every document-level scan (feature detection, script injection, validation) to ensure it recurses into the new container.
- Prefer one shared content-tree traversal per concern over per-container duplicates so a missing case fails everywhere at once instead of silently in one path.

## Related

- Transit ticket T-1593 (this bug)
- T-1472 — nil-skip guards in collapsible render loops (preserved; detection also skips nils)
- T-1615 — chart rendering ignores caller context (separate open ticket, intentionally not addressed here)
