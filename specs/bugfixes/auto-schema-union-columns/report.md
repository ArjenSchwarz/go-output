# Bugfix Report: Auto Schema Omits Later-Only Table Columns

**Ticket:** T-1576
**Date:** 2026-08-03
**Status:** Fixed

## Description of the Issue

v2 `DetectSchemaFromData` only inspected the first row for `[]Record`,
`[]map[string]any`, and `[]any` map inputs. When later rows contained
additional columns, `NewTableContent`'s default auto schema excluded those
keys, and renderers such as `AppendText` silently dropped the later-only
values.

**Reproduction steps:**
1. Create a table with sparse rows where a column first appears after row 0:
   ```go
   table, _ := NewTableContent("Sparse", []map[string]any{
       {"name": "Alice"},
       {"name": "Bob", "email": "bob@example.com"},
   })
   out, _ := table.AppendText(nil)
   ```
2. Observe that the output contains only the `name` header.
3. `bob@example.com` is silently absent from the output, even though the
   value is stored in the table's records.

**Impact:** Silent data loss in every output format for any auto-schema table
whose rows do not all share the first row's key set. No error or warning is
produced, so callers have no signal that values were dropped.

## Investigation Summary

Systematic inspection (Fagan-style walkthrough) of the schema detection path.

- **Symptoms examined:** Rendered output missing columns that exist in the
  input data; records verified to contain the full data (`convertToRecords`
  copies every key), confirming the loss happens at schema detection, not at
  record conversion.
- **Code inspected:** `v2/table_options.go` (`DetectSchemaFromData`,
  `DetectSchemaFromMap`), `v2/content.go` (`newTableContent`,
  `convertToRecords`, `TableContent.AppendText`).
- **Hypotheses tested:** Renderer-level filtering was ruled out — renderers
  correctly iterate the schema's key order; the schema itself was missing the
  keys. Record conversion was ruled out — all keys survive into records.

## Discovered Root Cause

`DetectSchemaFromData` sampled only the first element of slice inputs:
`case []Record` and `case []map[string]any` returned
`DetectSchemaFromMap(v[0])`, and `case []any` inspected only `v[0]`. The
implementation assumed all rows are homogeneous (share the first row's keys)
and never validated that assumption.

**Defect type:** Logic error (single-row sampling where a union over all rows
is required).

**Why it occurred:** Sampling `v[0]` is the cheapest implementation and
typical tabular data is homogeneous, so the sparse-row case was never
exercised. Because records retain all keys and renderers filter by schema
keys without reporting unknown record keys, the mismatch stayed silent.

**Contributing factors:** No cross-check between detected schema keys and
actual record keys anywhere in the pipeline.

## Resolution for the Issue

**Changes made:**
- `v2/table_options.go` - `DetectSchemaFromData` now scans slice input in
  full: the `[]Record` and `[]map[string]any` cases delegate to a new
  `detectSchemaFromMaps` helper (generic over `~map[string]any`) that builds
  the union of keys across all rows, and the `[]any` case collects every
  `map[string]any` element before doing the same. Keys are sorted
  alphabetically (the T-1692 convention for map input) and each column's type
  is detected from its value in the first row containing the key.
  `DetectSchemaFromMap` now delegates to the same helper with a single row,
  keeping one implementation. Doc comments updated.
- `v2/content.go` - `NewTableContent` doc comment updated to state that
  auto-detection covers the union of keys across all rows.
- `CHANGELOG.md` - entry under Unreleased/Fixed.

**Approach rationale:** The union keeps all data (the ticket's expected
behavior) while staying consistent with the deterministic-alphabetical
ordering T-1692 just established for map-derived schemas. Rows missing a
column simply render it as empty, which renderers already handle. No change
was needed to `newTableContent`: its existing `ErrTableKeyOrderGuessed` logic
keys off the detected key order, so the warning now correctly fires when the
multi-column union made the order a guess.

**Alternatives considered:**
- Return a validation error for sparse rows - rejected: sparse rows are
  legitimate data; erroring would break working callers while the union loses
  nothing.
- Preserve first-row order and append later-only keys - rejected:
  inconsistent with the deterministic alphabetical convention established by
  T-1692 for map input, and first-row map order is unrecoverable anyway.

## Regression Test

**Test file:** `v2/table_sparse_columns_test.go`
**Test names:**
- `TestDetectSchemaFromData_UnionOfColumns` — union of keys across rows for
  `[]map[string]any`, `[]Record`, and `[]any` inputs, alphabetical ordering,
  and column type taken from the first row containing the key.
- `TestNewTableContent_SparseRowsRenderAllColumns` — the ticket's repro:
  `AppendText` output must include the later-only column and value.
- `TestBuilderTable_SparseRowsKeyOrderWarning` — `ErrTableKeyOrderGuessed`
  fires when the multi-column union makes the alphabetical order a guess,
  even though the first row alone has a single column.

**What it verifies:** Automatic schema detection covers the union of columns
across all rows with deterministic (alphabetical) ordering, and no data is
silently dropped.

**Run command:** `cd v2 && go test -run 'TestDetectSchemaFromData_UnionOfColumns|TestNewTableContent_SparseRowsRenderAllColumns|TestBuilderTable_SparseRowsKeyOrderWarning' ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/table_options.go` | Union-of-columns schema detection via new `detectSchemaFromMaps` helper; doc comments |
| `v2/content.go` | `NewTableContent` doc comment |
| `v2/table_sparse_columns_test.go` | New regression tests |
| `CHANGELOG.md` | Unreleased/Fixed entry |
| `specs/bugfixes/auto-schema-union-columns/report.md` | This report |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`)
- [x] Linters/validators pass (`make lint`, 0 issues; `gofmt` clean)

**Manual verification:**
- Reproduction case output inspected before/after the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- When deriving metadata (schemas, headers) from collections, derive from the
  whole collection or explicitly document and validate the sampling
  assumption.
- Prefer surfacing a warning/error over silently dropping data when derived
  structure and actual data disagree.

## Related

- Transit ticket T-1576
- T-1692 — established the deterministic alphabetical ordering convention for
  map-derived schemas and the `ErrTableKeyOrderGuessed` warning; this fix
  keeps that convention for the union of columns.
- T-1451 (separate, open) — `WithAutoSchemaOrdered` skips schema detection;
  out of scope here.
