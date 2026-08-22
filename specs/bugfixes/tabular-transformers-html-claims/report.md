# Bugfix Report: Tabular Byte Transformers Falsely Advertise HTML Support

**Ticket:** T-1510
**Date:** 2026-08-22
**Status:** Fixed

## Description of the Issue

`SortTransformer.CanTransform`, `LineSplitTransformer.CanTransform`, and the
`FormatDetector` predicates `IsTabularFormat`/`SupportsSorting`/
`SupportsLineSplitting` (which also drive `FormatAwareTransformer` and
`EnhancedSortTransformer`) all reported `FormatHTML` as supported. The actual
`Transform` implementations in `v2/transformers.go` are byte-level: CSV is
parsed with `encoding/csv`, and every other tabular format is handled by
splitting lines on `\n` and detecting a tab/comma/pipe column separator.
Rendered HTML tables (`<table><tr><td>...`) use none of those separators, so:

- **Sorting was a no-op on HTML.** Header detection either found no separator
  line or latched onto a comma inside prose, and the sort key never matched a
  fragment of HTML markup, so the input was returned unchanged despite the
  advertised support.
- **Line splitting could corrupt raw HTML.** When cell content contained a
  comma (or the configured separator), `detectSeparator` mistook the comma for
  a column separator and the row was split, trimmed, and reassembled across
  multiple mangled lines.

**Reproduction steps:**
1. Build a `TransformPipeline` containing `NewLineSplitTransformer(";")`.
2. Run `pipeline.Transform(ctx, input, output.FormatHTML)` with rendered HTML
   such as `<tr><td>Alice</td><td>Java;Go, senior</td></tr>` inside a table.
3. Observe the output: the row is rewritten as
   `<tr><td>Alice</td><td>Java, senior</td></tr>` followed by a stray `Go,`
   line — corrupted HTML. Equivalently, observe that
   `NewSortTransformer("Name", true).CanTransform("html")` returns `true`
   while the transform never sorts HTML output.

**Impact:** Any pipeline that registers a sort or line-split transformer and
renders HTML. Sorting silently did nothing (misleading, but harmless bytes);
line splitting actively corrupted HTML output whenever cell text contained the
separator. Both transformers also wasted a full copy-and-scan pass over HTML
output they could never handle.

## Investigation Summary

- **Symptoms examined:** Advertised `CanTransform("html") == true` on both
  byte-level tabular transformers and across the `FormatDetector` tabular
  predicate family; confirmed via a pipeline-level reproduction that line
  splitting mangles HTML rows and sorting no-ops.
- **Code inspected:** `v2/transformers.go` (`SortTransformer.CanTransform`,
  `LineSplitTransformer.CanTransform`, `detectSeparator`, both `Transform`
  paths), `v2/format_aware.go` (`IsTabularFormat`, `SupportsSorting`,
  `SupportsLineSplitting`, `FormatAwareTransformer.CanTransform`,
  `EnhancedSortTransformer`), `v2/transformer.go` (`TransformPipeline.Transform`
  gates on `CanTransform`), `v2/output.go:382` (same gating in the render
  path).
- **Hypotheses tested:** Checked whether any non-test consumer relies on
  `IsTabularFormat` including HTML — the only consumers are
  `SupportsSorting`/`SupportsLineSplitting` and the wrappers above, so the
  predicate family can be fixed consistently. Checked whether HTML-aware
  transformation belongs in this fix — rejected as a feature, not a bugfix
  (per ticket guidance).

## Discovered Root Cause

The tabular format predicates were written as a conceptual "format family"
check (HTML can represent tables) rather than reflecting what the byte-level
implementations can actually parse (tab/comma/pipe-separated lines or CSV).
Nothing ever implemented HTML parsing, so the advertisement was false from the
start.

**Defect type:** Logic error — capability predicate out of sync with
implementation.

**Why it occurred:** `CanTransform`/`IsTabularFormat` were populated with every
format that *renders* tabular data instead of every format the transformers
can *parse*. The existing tests asserted the advertised values
(`{"html", true}`) rather than the transform behaviour on HTML, baking the
false claim into the test suite.

**Contributing factors:** `detectSeparator` treats any comma as a column
separator, which makes line-based transforms actively dangerous on markup;
the pipeline trusts `CanTransform` completely, so a wrong predicate is the
only gate between a transformer and formats it cannot handle.

## Resolution for the Issue

_To be completed after the fix is implemented._

## Regression Test

**Test file:** `v2/transformers_html_test.go`
**Test names:** `TestTabularTransformersDoNotAdvertiseHTMLSupport`,
`TestTransformPipeline_LineSplitLeavesHTMLIntact`,
`TestTransformPipeline_SortLeavesHTMLIntact`,
`TestEnhancedSortTransformer_Transform_HTMLPassthrough`

**What it verifies:** Every predicate in the tabular family
(`SortTransformer`, `LineSplitTransformer`, `FormatDetector.IsTabularFormat`/
`SupportsSorting`/`SupportsLineSplitting`, `EnhancedSortTransformer`, and the
`FormatAwareTransformer` wrappers for sort and linesplit) reports HTML as
unsupported, and pipeline-level runs with sort and line-split transformers
leave HTML bytes byte-for-byte intact (the line-split case reproduces real
corruption before the fix).

**Run command:** `cd v2 && go test -run 'TestTabularTransformersDoNotAdvertiseHTMLSupport|TestTransformPipeline_LineSplitLeavesHTMLIntact|TestTransformPipeline_SortLeavesHTMLIntact|TestEnhancedSortTransformer_Transform_HTMLPassthrough' ./...`

## Affected Files

_To be completed after the fix is implemented._

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass (`make check`)

## Prevention

**Recommendations to avoid similar bugs:**
- When writing a `CanTransform`/capability predicate, derive the supported set
  from what the implementation parses, not from which formats conceptually fit
  the category; add a behaviour-level test per advertised format rather than
  only asserting the predicate's return values.
- Treat `detectSeparator`-style heuristics as unsafe outside the formats they
  were designed for; gate them behind precise predicates.

## Related

- Transit ticket T-1510
- T-1518 (ColorTransformer scheme fix — same file, predicate/behaviour
  mismatch theme)
- T-1509 (EnhancedEmojiTransformer word-boundary fix in `format_aware.go`)
- T-1269 (CSV-aware parsing for these same transformers — established the
  "parse what you can actually parse" precedent)
