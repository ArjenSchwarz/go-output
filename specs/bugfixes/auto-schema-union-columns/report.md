# Bugfix Report: Auto Schema Omits Later-Only Table Columns

**Ticket:** T-1576
**Date:** 2026-08-03
**Status:** In Progress

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

_To be filled in after the fix is implemented._

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

_To be filled in after the fix is implemented._

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

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
