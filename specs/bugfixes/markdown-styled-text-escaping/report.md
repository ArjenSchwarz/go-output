# Bugfix Report: Markdown Styled Text Content Not HTML-Escaped

**Ticket:** T-2030
**Date:** 2026-11-02
**Status:** Fixed

## Description of the Issue

When a `TextContent` has `Color` or `Size` set, the Markdown renderer wraps the
text in a raw `<span style="...">`. T-1625 (PR #104) HTML-escaped the color
value inside the `style` attribute, but the text placed between the tags was
only Markdown-escaped. `escapeMarkdown` does not touch `<`, `>`, `&` or quotes,
so text containing HTML became live markup in any Markdown consumer that
passes inline HTML through.

**Reproduction steps:**
1. `doc := output.New().Text("<script>alert(1)</script>", output.WithTextStyle(output.TextStyle{Color: "red"})).Build()`
2. Render with `output.Markdown()`.
3. Observe `<span style="color: red"><script>alert(1)</script></span>` in the output.

**Impact:** High. Ordinary text containing `<`, `>` or `&` (e.g. `a < b`) is
misrendered, and untrusted text can inject elements/scripts into rendered
Markdown. Only the styled (Color/Size) path is affected.

## Investigation Summary

- **Symptoms examined:** Styled text emitted verbatim inside a raw `<span>`.
- **Code inspected:** `renderTextContentMarkdown` (`v2/markdown_renderer.go`),
  `escapeMarkdown`, and the HTML renderer's text handling
  (`v2/html_renderer.go`, which calls `html.EscapeString` on text).
- **Hypotheses tested:** Whether anything upstream sanitises text — it does
  not; `TextContent` stores the string as given.

## Discovered Root Cause

**Defect type:** Missing output encoding.

**Why it occurred:** The span path was added to approximate styling that
Markdown cannot express, reusing the Markdown-escaped string. Markdown escaping
and HTML escaping protect against different syntaxes; once the text is inside
a raw HTML element both are needed. T-1625 fixed only the attribute value.

## Resolution for the Issue

**Changes made:**
- `v2/markdown_renderer.go` (`renderTextContentMarkdown`) - wrap the span's
  inner content in `html.EscapeString`.

**Approach rationale:** HTML escaping is applied *after* Markdown escaping and
the bold/italic wrapping. The Markdown escapes (`\*`, `\_`, …) and emphasis
markers contain no HTML-special characters, so they are unchanged, and the
entities produced by `html.EscapeString` (e.g. `&#39;`) are never re-processed
by `escapeMarkdown` (which would turn `#` into `\#` and break them).

**Alternatives considered:**
- HTML-escape before Markdown-escape - rejected; `escapeMarkdown` would break
  numeric entities like `&#39;` / `&#34;`.
- HTML-escape all text, including the unstyled path - out of scope; plain
  Markdown text is not inside a renderer-emitted HTML element and changing it
  would alter documented output for every text block.

## Regression Test

**Test file:** `v2/renderer_markdown_regression_test.go`
**Test name:** `TestMarkdownRenderer_StyledTextContentEscaping`

**What it verifies:** `<script>`/element injection is escaped inside the span
for both Color and Size; bold emphasis is preserved alongside escaped `<`/`&`;
Markdown escapes survive and apostrophes become `&#39;` (not `&\#39;`).

**Run command:** `cd v2 && go test -run TestMarkdownRenderer_StyledTextContentEscaping .`

## Affected Files

| File | Change |
|------|--------|
| `v2/markdown_renderer.go` | HTML-escape text inside the styled span |
| `v2/renderer_markdown_regression_test.go` | Regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

- Any time a renderer emits raw HTML, every interpolated value (attributes and
  text) needs HTML encoding, regardless of other escaping already applied.

## Related

- T-1625 / PR #104 (color attribute on the same span)
