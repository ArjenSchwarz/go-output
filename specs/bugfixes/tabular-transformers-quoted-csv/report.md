# Bugfix Report: Tabular Transformers Misparse Quoted CSV

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

`SortTransformer` and `LineSplitTransformer` in `v2/transformers.go` advertise CSV
support (`CanTransform` returns true for `FormatCSV`), but both parse the rendered
CSV bytes with `strings.Split` and `detectSeparator` instead of `encoding/csv`.
This naive splitting ignores RFC 4180 quoting:

- A quoted field containing a comma, such as `"Smith, Bob",NY`, is split into three
  cells (`Smith`, ` Bob"`, `NY`). Sorting by a later column then compares the wrong
  value, and a re-emitted row has the wrong shape.
- Quoted fields containing embedded newlines are torn apart because the whole
  payload is split on `\n`, so one logical record becomes several broken rows.

**Reproduction steps:**
1. Render a table to CSV that contains a quoted field with a comma (e.g.
   `"Smith, Bob",NY`) or an embedded newline.
2. Apply `SortTransformer` (sort by a column after the quoted field) or
   `LineSplitTransformer`.
3. Observe rows split into the wrong number of cells, sorted by the wrong value,
   or multiline records broken across rows.

**Impact:** Medium. Any CSV output that uses quoted fields (commas, separators, or
embedded newlines inside values) is corrupted when these transformers run.

## Investigation Summary

- **Symptoms examined:** Sorting by a column produced the wrong order when an
  earlier field was quoted; multiline quoted records were broken into multiple
  rows.
- **Code inspected:** `v2/transformers.go` — `SortTransformer.Transform`,
  `LineSplitTransformer.Transform`, and the shared `detectSeparator` helper.
- **Hypotheses tested:** Confirmed via failing tests that `strings.Split(content,
  "\n")` breaks embedded newlines and `strings.Split(line, separator)` breaks
  quoted commas. Ruled out the table/markdown (tab/pipe) paths as affected — those
  formats do not use RFC 4180 quoting.

## Discovered Root Cause

Both transformers operate on already-rendered bytes using single-character string
splitting that assumes a field separator never appears inside a value and that a
newline always ends a record. CSV (RFC 4180) allows commas, the separator
character, and newlines inside double-quoted fields, so byte-level splitting
mis-parses them.

**Defect type:** Logic error — incorrect parsing (ignoring quoting rules).

**Why it occurred:** The transformers were written to handle several text formats
with one generic separator-splitting routine, which happens to be wrong for CSV.

**Contributing factors:** `CanTransform` returns true for `FormatCSV`, so the
incorrect path is reached whenever CSV output is transformed.

## Resolution for the Issue

**Changes made:**
- `v2/transformers.go` — `SortTransformer.Transform`: when the format is
  `FormatCSV`, parse the input with `encoding/csv` (variable fields per record,
  embedded newlines allowed), sort the records by the requested column, and
  re-emit with `encoding/csv`. Non-CSV formats keep the existing string-split
  logic unchanged.
- `v2/transformers.go` — `LineSplitTransformer.Transform`: when the format is
  `FormatCSV`, parse records with `encoding/csv`, split the targeted cell, and
  re-emit with `encoding/csv`. Non-CSV formats keep the existing logic.

**Approach rationale:** Parsing and re-emitting with `encoding/csv` is the robust
option recommended by the ticket. It correctly handles quoted commas, embedded
separators, and embedded newlines, and produces canonical, valid CSV. Scoping the
change to the `FormatCSV` branch leaves the table/markdown behaviour (validated by
existing tests, including the T-1221 Markdown separator handling) untouched.

**Alternatives considered:**
- Removing CSV support from these transformers — rejected because it is a
  user-facing capability reduction; correct parsing is preferable.
- Patching the string-split logic to skip quotes — rejected as a fragile
  reimplementation of a standard-library parser.

## Regression Test

**Test file:** `v2/transformers_test.go`
**Test names:** `TestSortTransformer_QuotedCSV`, `TestLineSplitTransformer_QuotedCSV`

**What it verifies:** Quoted CSV fields containing commas and embedded newlines are
parsed as whole values; sorting uses the correct column value; line splitting only
affects the targeted cell; and the CSV is re-emitted with quoting preserved.

**Run command:** `go test -run 'TestSortTransformer_QuotedCSV|TestLineSplitTransformer_QuotedCSV' ./v2/...`

## Affected Files

| File | Change |
|------|--------|
| `v2/transformers.go` | CSV-format paths of `SortTransformer` and `LineSplitTransformer` now parse/re-emit with `encoding/csv`. |
| `v2/transformers_test.go` | Added quoted-CSV regression tests for both transformers. |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the regression tests fail before the fix and pass after.

## Prevention

**Recommendations to avoid similar bugs:**
- Use `encoding/csv` for any CSV parsing rather than `strings.Split`.
- When a transformer claims support for a structured format, parse with that
  format's canonical reader instead of generic separator detection.

## Related

- Transit ticket T-1269
- Related transformer fixes in the same file: T-1267 (emoji word boundaries),
  T-1221 (Markdown separator row).
