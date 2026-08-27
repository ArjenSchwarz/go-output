# Implementation Explanation: Tabular Transformers Stop Advertising HTML Support (T-1510)

Explanation of the changes on branch `T-1510/bugfix-tabular-transformer-html-claims` at three expertise levels, generated as part of the pre-push review.

## Beginner Level

### What Changed

The go-output library can transform its rendered output before it is written — for example, sorting table rows or splitting long cells across multiple lines. Each transformer has a `CanTransform(format)` method that answers "can I handle this output format?". The sorting and line-splitting transformers used to answer "yes" for HTML. That was wrong: they can only read plain-text tables where columns are separated by tabs, commas, or pipes (or proper CSV). An HTML table looks like `<tr><td>Alice</td></tr>` — there are no separator characters to find. This change makes those transformers answer "no" for HTML.

### Why It Matters

Two bad things happened because of the wrong "yes":

1. **Sorting did nothing on HTML** — the transformer ran, scanned the whole output, found nothing it understood, and returned the input unchanged. Users who asked for sorted HTML tables silently got unsorted ones.
2. **Line splitting could corrupt HTML** — worse than doing nothing. If a table cell contained a comma (very common in prose), the transformer mistook the comma for a column separator and rewrote the row, splitting it across lines and destroying the markup.

After this change, the transformation pipeline simply skips these two transformers for HTML output, so HTML passes through byte-for-byte intact.

### Key Concepts

- **Transformer**: A component that rewrites rendered output bytes (sort rows, convert words to emoji, add colors). Think of it as a filter in a pipeline.
- **`CanTransform` predicate**: A yes/no gate the pipeline consults before running each transformer on a given format. If the gate is wrong, the pipeline runs a transformer on data it cannot understand.
- **Byte-level parsing**: These transformers work on the final rendered text, not the original structured data. They can only "see" what the text encodes with separators — markup like HTML is opaque to them.

## Intermediate Level

### Changes Overview

Production changes are three predicate edits (net −3 formats advertised):

- `v2/transformers.go` — `SortTransformer.CanTransform` and `LineSplitTransformer.CanTransform` drop `FormatHTML`; both now return true only for `table`, `csv`, `markdown`. Godoc on each explains the exclusion.
- `v2/format_aware.go` — `FormatDetector.IsTabularFormat` drops `FormatHTML`. This single edit cascades through `SupportsSorting` and `SupportsLineSplitting` (both delegate to it), which in turn drive `FormatAwareTransformer.CanTransform`'s `"sort"`/`"linesplit"` switch cases and `EnhancedSortTransformer` (both its `CanTransform` and the internal gate in its `Transform`).

Test changes: `v2/transformers_test.go` and `v2/format_aware_test.go` flip five `{html, true}` expectations to `false`; `v2/transformers_html_test.go` is new and adds eight predicate assertions plus three behavior-level passthrough tests. `CHANGELOG.md` documents the behavior change; `specs/bugfixes/tabular-transformers-html-claims/report.md` is the investigation report.

### Implementation Approach

The fix follows the "fix the predicate at its source" pattern. `IsTabularFormat` was verified (by grep) to have exactly two consumers — `SupportsSorting` and `SupportsLineSplitting` — so removing HTML there fixes the entire Enhanced/FormatAware layer consistently in one place, rather than special-casing each wrapper. The pipeline (`TransformPipeline.Transform` in `v2/transformer.go`) already gates every transformer on `CanTransform`, so no pipeline changes were needed: correcting the predicates is sufficient to keep the transformers off HTML output entirely.

The red-phase commit (5d9211c) added failing regression tests that reproduce genuine corruption — `Java;Go, senior` inside a `<td>` was mangled into two broken rows by `LineSplitTransformer` — before the green-phase commit (3cdbe9f) applied the predicate fix.

### Trade-offs

- **Drop the claim vs. implement HTML support**: Parsing `<tr>/<td>` structure to genuinely sort or split HTML tables would be a new feature with real parsing complexity (entities, nested markup, attributes). The ticket explicitly endorsed removing the false advertisement instead. Nothing that worked is lost — sorting on HTML never did anything.
- **Behavior change accepted**: Pipelines registering these transformers no longer run them on HTML. For sort this is byte-identical output (the pass was a no-op); for line-split the output *changes* — from corrupted to intact — which is the point of the fix. The CHANGELOG flags this explicitly.
- **Markdown remains advertised**: Markdown tables use pipe separators, which the byte-level parser genuinely handles, so it stays in the set.

## Expert Level

### Technical Deep Dive

The defect class is a capability predicate out of sync with its implementation. `SortTransformer.Transform` and `LineSplitTransformer.Transform` share a parsing model: CSV via `encoding/csv` when `format == FormatCSV`, otherwise line-based splitting with `detectSeparator` choosing among `\t`, `,`, `|`. `detectSeparator` treats *any* occurrence of a candidate separator as structural — so on HTML, a comma inside cell prose is indistinguishable from a column boundary. That makes the line-based path actively unsafe on markup, not merely useless: `LineSplitTransformer` would split a "cell" at the configured separator, trim whitespace, and reassemble rows, destroying tag structure. The sort path failed closed only by accident (sort keys never match HTML fragments), which is why it presented as a silent no-op rather than corruption.

The fix's correctness argument rests on the consumer graph: `IsTabularFormat` → {`SupportsSorting`, `SupportsLineSplitting`} → {`FormatAwareTransformer` switch cases `"sort"`/`"linesplit"`, `EnhancedSortTransformer.CanTransform` + internal `Transform` gate}. All were verified as the sole consumers; no non-test code reads `IsTabularFormat` directly. `FormatAwareTransformer.CanTransform` also short-circuits on the underlying transformer's own `CanTransform` first, so the wrapper and the wrapped transformer can never disagree into a true.

The regression tests pin both layers: eight predicate assertions cover every entry point in the family (including both wrapper paths), and three behavior tests assert byte-for-byte passthrough at pipeline level (`TransformPipeline.Transform`) and at the `EnhancedSortTransformer.Transform` internal gate. The line-split passthrough test input (`Java;Go, senior`) is constructed to hit both the configured separator (`;`) and the comma heuristic, so it genuinely failed pre-fix.

### Architecture Impact

- **Public API surface**: `CanTransform`/`IsTabularFormat`/`SupportsSorting`/`SupportsLineSplitting` return values change for `"html"`. This is a documented behavior change, not an API signature change. Any downstream caller branching on these predicates for HTML gets the corrected answer.
- **Pipeline trust model unchanged**: `TransformPipeline` still trusts `CanTransform` completely — the predicate is the *only* gate between a transformer and a format it cannot parse. This fix restores the invariant but does not add defense in depth to the line-based `Transform` paths themselves (see below).
- **Format taxonomy clarified**: `IsTabularFormat` now means "byte-level parseable tabular text", documented in its godoc. `IsTextBasedFormat` (which still includes HTML, feeding `SupportsEmoji`) is unaffected — emoji transformation on HTML remains supported and is regex-based, not separator-based, so its claim is genuine.

### Potential Issues

- **Downstream reliance on the no-op**: A consumer that registered a sort transformer for mixed-format output and *relied* on `CanTransform("html")` being true (e.g., for capability display) will see changed booleans. Sorting behavior cannot regress — it never worked.
- **`detectSeparator` remains a footgun**: Any future format added to the tabular set gets the comma heuristic with it. The report's prevention section recommends behavior-level tests per advertised format; the new test file establishes that pattern for HTML.
- **HTML-aware transformation** remains unimplemented and out of scope; if ever built, it should be a distinct transformer with its own predicate, not a widening of these byte-level ones.

## Completeness Assessment

- **Fully implemented**: Predicate corrections across the entire tabular family (both base transformers, the detector, both wrapper layers); regression coverage at predicate and behavior level; CHANGELOG and investigation report.
- **Partially implemented**: None.
- **Missing / explicitly out of scope**: HTML-aware table transformation (feature, per ticket guidance); hardening of `detectSeparator` itself (pre-existing heuristic, unchanged); consolidated typed-nil validation referenced from adjacent work (T-1649, unrelated).

All report claims were verified against the code during review: the consumer graph of `IsTabularFormat`, the test names, and the stated run command all check out.
