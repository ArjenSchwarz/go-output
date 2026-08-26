# Implementation Explanation: Mermaid Script Injection for Collapsible Sections (T-1593)

Branch: `T-1593/bugfix-mermaid-collapsible-detection` — explanation of the changes relative to `origin/main`.

## Beginner Level

### What Changed

The go-output library can turn data into HTML pages. One of the things it can put on a page is a chart (for example a pie chart), which it writes using a text-based chart language called Mermaid. A browser cannot draw Mermaid charts by itself — the page must also include a small JavaScript snippet that loads the Mermaid library and tells it to convert the chart text into a picture.

The HTML renderer only adds that snippet when it believes the page contains a chart. To decide, it walks through the document's contents looking for charts. The bug: the walk knew how to look inside ordinary sections, but not inside "collapsible sections" (the fold-out `<details>` blocks you can click to expand). So if every chart on a page lived inside a collapsible section, the renderer concluded "no charts here", skipped the snippet, and the browser showed the raw chart text instead of a drawing.

The fix teaches the chart-finding walk to look inside collapsible sections too — at any nesting depth, in any combination of sections and collapsible sections.

### Why It Matters

Anyone using this library to publish reports with charts tucked inside fold-out sections got broken pages: instead of a pie chart, readers saw code-like text such as `pie title Status ...`. After the fix, those charts render as actual diagrams.

### Key Concepts

- **Renderer**: the part of the library that converts a document into a specific output format, here HTML.
- **Mermaid**: a mini-language for describing charts as text; a JavaScript library turns the text into graphics in the browser.
- **Collapsible section**: a fold-out container (`<details>`/`<summary>` in HTML) holding other content.
- **Recursion**: a function that calls itself to handle nested structures — needed here because sections can contain sections can contain charts, like boxes inside boxes.

---

## Intermediate Level

### Changes Overview

- `v2/html_renderer.go`: `documentContainsMermaidCharts` previously had an inline loop over top-level contents plus a near-duplicate helper `sectionContainsMermaidCharts` that recursed only into `*SectionContent`. Neither visited `*DefaultCollapsibleSection`, while `renderCollapsibleSection` happily rendered nested charts as `<pre class="mermaid">`. Both walkers are replaced by one recursive helper, `contentsContainMermaidCharts([]Content) bool`, which detects `*ChartContent` and descends into `*SectionContent` (via `Contents()`) and `*DefaultCollapsibleSection` (via `Content()`). `documentContainsMermaidCharts` is now a one-line delegate.
- `v2/mermaid_html_markdown_test.go`: new table-driven regression test `TestHTMLRenderer_MermaidScriptInjectionCollapsibleSections` covering chart-in-collapsible, section-in-collapsible, collapsible-in-section, collapsible-in-collapsible, and a no-chart negative case, asserted on both `Render` and `RenderTo`.
- `CHANGELOG.md`: entry under Unreleased/Fixed.
- `specs/bugfixes/mermaid-collapsible-detection/report.md`: investigation report (root cause, alternatives, prevention).

### Implementation Approach

The guiding principle is: detection must mirror the render dispatch. `renderContent`'s type switch recurses into exactly two container types (`*SectionContent`, `*DefaultCollapsibleSection`), and `renderChartContentHTML` is the only producer of `<pre class="mermaid">`. The new helper's case set is deliberately identical to the recursion cases of that switch, so every container the renderer can descend into is also visited by detection. Nil entries are skipped, mirroring the T-1472 nil-skip guards in the collapsible render loop. Both `Render` and `RenderTo` call `documentContainsMermaidCharts`, so one fix covers both the buffered and streaming paths.

### Trade-offs

- **Patch the two existing walkers instead**: rejected — it keeps the duplicated traversal that caused the divergence; the next container type would need edits in multiple places.
- **String-match the rendered output for `<pre class="mermaid">`**: rejected — fragile (false positives from raw/escaped content) and incompatible with `RenderTo`, which streams content before deciding on the script.
- **Detection is a separate pre-render tree walk** rather than a flag set during rendering. It costs one extra O(n) pass, acceptable for a boolean check and consistent with the existing structure; folding it into the render pass would be a larger refactor than a bugfix warrants.

---

## Expert Level

### Technical Deep Dive

`contentsContainMermaidCharts` is depth-first with early return: the first `*ChartContent` found propagates `true` up the stack without visiting remaining siblings. It uses the concrete `*DefaultCollapsibleSection` type rather than the `CollapsibleSection` interface — intentionally, because `renderContent` also dispatches on the concrete type. A third-party `CollapsibleSection` implementation falls through `renderContent`'s `default` branch and renders as an escaped `<pre>` without the mermaid class, so it needs no script; detection skipping it is consistent, not a gap. The same reasoning covers `*GraphContent`: the HTML switch has no case for it, so graphs take the escaped fallback and never produce mermaid markup. `CollapsibleValue` (table-cell expandables) carries `any` details, never `[]Content`, and cannot route back into `renderContent`, so it is correctly outside the traversal.

The walk goes through the public accessors (`GetContents()`, `Contents()`, `Content()`), which return defensive copies — a handful of shallow slice allocations per document, immaterial for a one-shot boolean walk. Recursion depth equals content-tree depth; a cyclic graph is not constructible through the public API and would already hang `freezeSectionContents` at `Build()` before detection ever ran.

### Architecture Impact

The change removes a per-container duplicated traversal in favour of a single walker aligned with the render dispatch, which is the pattern the codebase should follow for any future document-level scan (feature detection, validation, asset injection). The prevention note in the report captures the rule: when a renderer gains a new nested-container type, every document-level scan must gain the corresponding case — and having exactly one walker per concern makes a missing case fail loudly everywhere rather than silently in one path.

### Potential Issues

- If a future container type is added to `renderContent`'s dispatch without updating `contentsContainMermaidCharts`, the same class of bug returns. The helper's doc comment states the mirroring contract explicitly to make that coupling discoverable.
- The nil-skip branch in detection is defence-in-depth only: `NewCollapsibleSection` and `SectionContent.AddContent` already drop untyped nils, so the branch is unreachable via the public API and has no direct test coverage (typed nils remain undetectable pending T-1649).
- Chart rendering still ignores caller context (T-1615, separate open ticket) — intentionally untouched here.

## Completeness Assessment

- **Fully implemented**: recursive chart detection across sections and collapsible sections in any nesting combination, on both `Render` and `RenderTo`; regression tests for all four positive nesting shapes plus a negative case; changelog entry; investigation report.
- **Partially implemented**: nothing — the fix's scope is complete for T-1593.
- **Missing / intentionally out of scope**: unifying detection with the render pass (flag-during-render refactor); typed-nil validation (T-1649); context propagation in chart rendering (T-1615).
