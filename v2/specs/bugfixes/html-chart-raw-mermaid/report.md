# Bugfix Report: HTML Chart Rendering Writes Raw Mermaid Text Into Pre Tags

**Date:** 2026-05-25
**Status:** Fixed
**Ticket:** T-1293

## Description of the Issue

When rendering a document to HTML, chart content (`ChartContent`) is converted
to Mermaid syntax and embedded inside a `<pre class="mermaid">` block so
Mermaid.js can draw it client-side. The Mermaid renderer emits user-controlled
fields — chart titles, gantt task names, gantt section names, and pie slice
labels — as raw text. `htmlRenderer.renderChartContentHTML` wrote those raw
Mermaid bytes directly into the `<pre>` block without HTML escaping.

If any of those fields contained HTML metacharacters (`<`, `>`, `&`), the output
was malformed. Worse, a value such as `</pre><script>...</script>` could break
out of the pre block and inject arbitrary HTML/script into the page — a stored
XSS injection path.

**Reproduction steps:**
1. Build a document with a pie or gantt chart whose title/label/task/section
   contains `<script>alert(1)</script>` or `</pre><script>...`.
2. Render it with the HTML renderer.
3. Observe the literal `<script>`/`</pre>` tags inside the `<pre class="mermaid">`
   block, unescaped.

**Impact:** Security-relevant (XSS). Any caller embedding untrusted text in chart
fields and serving the resulting HTML is exposed to script injection and
malformed output. Affects all chart types rendered via the HTML renderer.

## Investigation Summary

- **Symptoms examined:** Raw HTML chart output containing unescaped user text.
- **Code inspected:**
  - `v2/html_renderer.go` `renderChartContentHTML` (~lines 301-323) — the
    embedding point.
  - `v2/graph_renderers.go` `mermaidRenderer` (`renderGanttChart`,
    `renderPieChart`, `renderFlowchartContent`, `renderGraphContent`) — confirmed
    titles, task titles, section names, and pie labels are written verbatim.
- **Hypotheses tested:**
  - Could escaping happen in the Mermaid renderer instead? Rejected — the Mermaid
    renderer also feeds Markdown output (code fences) and standalone `.mmd`
    files, where HTML escaping would corrupt the syntax. Escaping belongs at the
    HTML embedding boundary.

## Discovered Root Cause

`renderChartContentHTML` wrote `mermaidData` straight into the HTML stream with
`result.Write(mermaidData)`, with no HTML escaping, even though every other text
path in `html_renderer.go` routes user data through `html.EscapeString`.

**Defect type:** Missing output encoding / missing validation (XSS).

**Why it occurred:** The Mermaid bytes were treated as already-safe markup rather
than as text that needs encoding for the HTML context it is placed in. The
Mermaid renderer's job is Mermaid syntax, not HTML safety.

**Contributing factors:** Mermaid.js reads the *text content* of the pre block,
so HTML-escaping is invisible to it (the browser un-escapes text content before
Mermaid parses it). This made the missing escape easy to overlook because the
chart still renders correctly for benign input.

## Resolution for the Issue

**Changes made:**
- `v2/html_renderer.go:renderChartContentHTML` — escape the Mermaid output with
  `html.EscapeString` before writing it inside `<pre class="mermaid">`.

**Approach rationale:** Escaping at the HTML embedding boundary is the correct
layer: it is the point where Mermaid text crosses into an HTML context. It
matches how the rest of `html_renderer.go` handles user data, is a one-line
change, and preserves valid Mermaid syntax because Mermaid.js reads the
DOM text content (which the browser has already un-escaped).

**Alternatives considered:**
- **Escape inside the Mermaid renderer** — rejected; it would corrupt Markdown
  code-fence and standalone Mermaid output, which must not be HTML-escaped.
- **Sanitize/strip the user fields** — rejected; lossy and changes the rendered
  diagram text. Escaping preserves the intended content.

## Regression Test

**Test file:** `v2/mermaid_html_markdown_test.go`
**Test name:** `TestHTMLRenderer_ChartContent_EscapesUserText`

**What it verifies:** Pie and gantt charts whose title, labels, task names, and
section names contain `<script>` and `</pre>` payloads produce HTML where those
metacharacters are escaped (`&lt;`, `&gt;`, `&#39;`) and no literal `<script>` or
`</pre><...>` markup appears inside the mermaid `<pre>` block.

**Run command:** `cd v2 && go test . -run TestHTMLRenderer_ChartContent_EscapesUserText -v`

## Affected Files

| File | Change |
|------|--------|
| `v2/html_renderer.go` | Escape Mermaid output before embedding in `<pre class="mermaid">` |
| `v2/mermaid_html_markdown_test.go` | Add XSS/escaping regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Inspected rendered HTML for both pie and gantt charts to confirm escaped
  entities appear and Mermaid syntax (titles, labels, task lines) is otherwise
  intact.

## Prevention

**Recommendations to avoid similar bugs:**
- Treat any bytes written into an HTML context as text that needs encoding,
  unless they are known-trusted markup produced by the renderer itself.
- When a renderer's output is reused across formats (HTML, Markdown, raw files),
  apply context-specific encoding at each embedding boundary rather than in the
  shared producer.

## Related

- Transit ticket T-1293
