# Bugfix Report: Markdown Text Color Is Not Escaped in Style Attribute

**Ticket:** T-1625
**Date:** 2026-07-12
**Status:** Fixed

## Description of the Issue

The Markdown renderer emits raw HTML when `TextStyle.Color` or `TextStyle.Size` is set, because color and size have no native Markdown equivalent. The color value was interpolated directly into the `style` attribute of the generated `<span>` without any HTML attribute escaping or validation.

**Reproduction steps:**
1. Build a document with `Text("Styled text", WithTextStyle(TextStyle{Color: "red\" onclick=\"alert(1)"}))`
2. Render the document with the Markdown renderer
3. Observe the output: `<span style="color: red" onclick="alert(1)">Styled text</span>` — the hostile color value closed the `style` attribute and injected an `onclick` attribute

**Impact:** Any consumer that renders the Markdown output as HTML (GitHub, documentation sites, wikis) can end up with injected attributes or elements if the color value comes from untrusted input. Severity is moderated by the fact that the color value is supplied by the calling application, but the library must not be an injection vector when that value is attacker-influenced. The HTML renderer already escaped this value; only the Markdown renderer was vulnerable.

## Investigation Summary

The ticket identified the exact defect site, and inspection confirmed it and ruled out sibling occurrences.

- **Symptoms examined:** Rendered Markdown output containing a live `onclick` attribute and a live `<script>` element when hostile color values were supplied via `WithTextStyle`/`WithColor`.
- **Code inspected:** `v2/markdown_renderer.go` (`renderTextContentMarkdown`), `v2/html_renderer.go` (`renderTextContentHTML`, which escapes with `html.EscapeString`), and all other raw-HTML emission sites in the Markdown renderer (`<details>`/`<summary>` handling).
- **Hypotheses tested:** Checked whether other Markdown renderer sites interpolate user data into HTML attributes — they do not. The `<details%s>` sites only interpolate a fixed `openAttr` constant; summary/detail *content* goes through `escapeHTMLContent`/`escapeMarkdown`. `TextStyle.Size` is an `int` formatted with `%d`, so it cannot carry an injection payload. The `style` attribute in `renderTextContentMarkdown` was the only injection point.

## Discovered Root Cause

`renderTextContentMarkdown` built the style attribute with `fmt.Sprintf("color: %s", style.Color)` and embedded it via `<span style="%s">` without escaping. A `"` in the color value therefore terminated the attribute, allowing arbitrary attribute or element injection.

**Defect type:** Missing output encoding (HTML attribute escaping) of user-controlled data.

**Why it occurred:** The Markdown renderer's escaping helpers focus on Markdown metacharacters. When the color/size feature required dropping down to raw HTML, the value was embedded without applying HTML escaping. The equivalent code path in the HTML renderer was hardened with `html.EscapeString`, but the duplicated logic in the Markdown renderer was overlooked.

**Contributing factors:** The styled-text-to-HTML logic is duplicated between the HTML and Markdown renderers rather than shared, so a fix in one did not propagate to the other.

## Resolution for the Issue

_Pending — filled in after the fix is implemented._

## Regression Test

**Test file:** `v2/renderer_markdown_test.go`
**Test name:** `TestMarkdownRenderer_TextStyleColorEscaping`

**What it verifies:** Hostile color values cannot escape the `style` attribute. Covers a double-quote attribute breakout (`red" onclick="alert(1)`) and an element injection (`red"><script>alert(1)</script>`), asserting the escaped form is emitted and the live attribute/element is absent.

**Run command:** `cd v2 && go test -run TestMarkdownRenderer_TextStyleColorEscaping ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/markdown_renderer.go` | Escape `style.Color` before embedding in the style attribute |
| `v2/renderer_markdown_test.go` | Add `TestMarkdownRenderer_TextStyleColorEscaping` regression test |
| `CHANGELOG.md` | Add entry under Unreleased → Fixed |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- Confirmed the existing `TestMarkdownRenderer_TextStyling` expectation (`<span style="color: red; font-size: 14px">`) still holds, since benign color values are unchanged by escaping.

## Prevention

**Recommendations to avoid similar bugs:**
- Whenever a renderer emits raw HTML, every interpolated dynamic value must be HTML-escaped (attribute and text contexts alike) — mirror the existing `html.EscapeString` usage in `v2/html_renderer.go`.
- When hardening one renderer against an injection, grep the other renderers for the same pattern; the styled-text logic exists in both the HTML and Markdown renderers.

## Related

- Transit ticket T-1625
- Similar prior fix: `specs/bugfixes/escape-graph-labels/` (escaping of user data in graph output)
