# Bugfix Report: Escape Graph Labels in DOT and Mermaid Renderers

**Date:** 2026-05-25
**Status:** Fixed
**Ticket:** T-1292

## Description of the Issue

The DOT (Graphviz) and Mermaid renderers in `v2/graph_renderers.go` emitted
user-controlled label text directly into the output without escaping
format-special characters. Titles, edge labels, and node text containing double
quotes, backslashes, newlines, or Mermaid delimiter characters broke the
generated graph syntax or could inject unintended markup.

**Reproduction steps:**

1. Build a graph with an edge whose label contains a double quote, e.g.
   `Graph("", []Edge{{From: "A", To: "B", Label: `+"`weight \"5\"`"+`}})`.
2. Render it with the DOT renderer.
3. Observe the output `A -> B [label="weight "5""];` — the embedded quotes
   terminate the DOT string early, producing invalid Graphviz.

The same class of failure occurs in Mermaid: an edge label containing `|`
produced `A -->|a|b| B`, where the embedded pipe prematurely closes the
edge-label delimiter, and node text with a quote produced `[node "x"]`.

**Impact:** Medium. Any caller rendering graphs from data containing quotes,
backslashes, pipes, or newlines in labels would get malformed DOT/Mermaid output
that fails to parse or renders incorrectly. Affects DOT and Mermaid output
formats only (the Draw.io CSV path already escaped via `writeCSVRow`).

## Investigation Summary

Applied a structured root-cause analysis to the rendering paths.

- **Symptoms examined:** Invalid DOT/Mermaid produced for labels with special
  characters.
- **Code inspected:** `v2/graph_renderers.go` —
  `dotRenderer.renderGraphContent`, `mermaidRenderer.renderGraphContent`,
  `sanitizeMermaidID`, `sanitizeDOTID`.
- **Hypotheses tested:** Whether `sanitizeDOTID` (used for node IDs) covered
  labels — it does not; labels are emitted via raw `fmt.Fprintf`. `sanitizeDOTID`
  also escapes only `"`, missing `\` and newlines, so it is insufficient even
  for labels.

## Discovered Root Cause

Label text was written into the output via `fmt.Fprintf` using `label="%s"`
(DOT title and edge), `-->|%s|` (Mermaid edge), and `[%s]` (Mermaid node via
`sanitizeMermaidID`) without any escaping. No escaping helper for label strings
existed; `sanitizeDOTID` handled identifiers only and escaped just the double
quote.

**Defect type:** Missing output escaping (injection / malformed output).

**Why it occurred:** The renderers were written assuming simple alphanumeric
labels. The ID-sanitisation helper escaped quotes but was never applied to
labels, and there was no equivalent for backslashes, newlines, or Mermaid
delimiters.

**Contributing factors:** DOT and Mermaid have different escaping conventions,
so a single shared helper would not suffice — each format needs its own.

## Resolution for the Issue

**Changes made:**

- `v2/graph_renderers.go` — added `escapeDOTLabel` (escapes `\`, `"`, newline,
  carriage-return per DOT string rules) and `escapeMermaidLabel` (encodes `"` as
  `&quot;`, newline as `<br/>`, strips carriage-return), plus
  `mermaidLabelNeedsQuoting` to decide when the quoted/escaped form is required.
- `dotRenderer.renderGraphContent` — title and edge labels now pass through
  `escapeDOTLabel`.
- `mermaidRenderer.renderGraphContent` — edge labels containing delimiter-
  breaking characters are wrapped in quotes and passed through
  `escapeMermaidLabel`; plain labels keep the bare `|label|` form.
- `sanitizeMermaidID` — node text containing characters that break the `[..]`
  syntax (`"[]|<>` or newlines) is wrapped in quotes and escaped; ordinary
  special characters (spaces, dashes, colons) keep the simpler `[text]` form.

**Approach rationale:** Format-specific escaping is applied at every label
emission point. Escaping is conditional for Mermaid so that existing valid
output for ordinary special characters (e.g. `[Node A]`, `|HTTP|`) is unchanged,
keeping the diff and behavioural change minimal while closing the syntax-breaking
cases.

**Alternatives considered:**

- **Always quote-wrap every Mermaid label/node** — Rejected: it churns existing
  valid output and the golden tests for no functional benefit.
- **Reuse/extend `sanitizeDOTID` for labels** — Rejected: ID sanitisation also
  decides whether to quote at all, which is the wrong contract for labels (DOT
  labels are always quoted), and it lacked backslash/newline handling.

## Regression Test

**Test file:** `v2/graph_renderers_test.go`
**Test names:** `TestDOTRenderer_EscapesLabels`, `TestMermaidRenderer_EscapesLabels`
(and an updated case in `TestSanitizeMermaidID`).

**What it verifies:** DOT titles/edge labels escape `"`, `\`, and newlines;
Mermaid edge labels and node text with `"`, `|`, or newlines are quoted and
escaped (`&quot;`, `<br/>`).

**Run command:**
`go test -run 'TestDOTRenderer_EscapesLabels|TestMermaidRenderer_EscapesLabels' ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/graph_renderers.go` | Added DOT/Mermaid label escaping helpers and applied them at all label emission points |
| `v2/graph_renderers_test.go` | Added escaping regression tests; updated one `sanitizeMermaidID` expectation |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`make test`)
- [x] Linter passes (`make lint`, 0 issues)

**Manual verification:**
- Confirmed the new tests fail against the pre-fix code (the "red" stage) before
  implementing the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Route all user-controlled text destined for a structured output format through
  a format-specific escaping helper rather than raw `fmt.Fprintf`.
