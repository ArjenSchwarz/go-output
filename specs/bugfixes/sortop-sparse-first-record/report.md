# Bugfix Report: SortOp Sparse First Record

**Date:** 2026-11-02
**Status:** Fixed
**Ticket:** T-1465

## Description of the Issue

`SortOp.Apply` validated keyed sort columns by looking only at the first record. A sparse table whose first row omits the sort column failed even when the schema and later rows contain it.

**Reproduction steps:**
1. `NewTableContent("t", []map[string]any{{"id": 1}, {"id": 2, "score": 10}}, WithKeys("id", "score"))`
2. `NewSortOp(SortKey{Column: "score", Direction: Ascending}).Apply(ctx, table)`
3. Observe `sort column 'score' does not exist in table data. Available columns: [id]`

**Impact:** Medium. Any sparse table (optional fields, merged sources) could not be sorted by a field missing from its first row, which also failed the render.

## Investigation Summary

- **Symptoms examined:** Validation error naming a column that exists in the schema.
- **Code inspected:** `SortOp.Apply`, `compareRecords`/`compareValues` in `v2/operations.go`; `Schema` in `v2/schema.go`.
- **Hypotheses tested:** The comparator already reads `a[key.Column]`, which yields nil for missing keys, and `compareValues` orders nil first. Only the pre-sort validation rejected the table.

## Discovered Root Cause

**Defect type:** Missing validation scope (validation used a sample instead of the table's column set).

**Why it occurred:** The check was written before the schema-driven model and sparse-data support (T-1576); its comment already noted missing values elsewhere are treated as nil, but the first record was treated as authoritative.

## Resolution for the Issue

**Changes made:**
- `v2/operations.go` `SortOp.Apply` — validate each sort key with a new `TableContent.hasColumn` helper: the column is known if the schema defines it (`HasField`, so hidden fields count, or it is in the key order) or any record contains it.
- `v2/operations.go` — new `TableContent.availableColumns` helper for the error message: schema key order, then extra record columns sorted (deterministic, instead of first-record map order).

**Approach rationale:** Accepting schema columns matches the ticket and the table model; also accepting record-only columns keeps tables with an incomplete `WithKeys` list sortable by data that the old check would have accepted when present in the first row. Truly unknown columns are still rejected, so typos keep failing loudly.

**Alternatives considered:**
- Schema only — would regress tables whose `WithKeys` omits a column that appears in the data (previously accepted if present in row 0).
- Remove validation entirely — would hide typos in sort columns.

## Regression Test

**Test file:** `v2/operations_sort_sparse_test.go`
**Test names:** `TestSortOpSparseFirstRecord`, `TestSortOpUnknownColumnStillRejected`

**What it verifies:** Sparse tables sort without error with nil values ordered first ascending / last descending, for schema keys, hidden schema fields and record-only columns; unknown columns still error.

**Run command:** `cd v2 && go test -run 'TestSortOpSparseFirstRecord|TestSortOpUnknownColumnStillRejected' .`

## Affected Files

| File | Change |
|------|--------|
| `v2/operations.go` | Schema/all-record sort column validation; helpers |
| `v2/operations_sort_sparse_test.go` | Regression tests |
| `CHANGELOG.md` | Entry |

## Verification

**Automated:**
- [x] Regression test passes (failed before the fix)
- [x] Full test suite passes, except two pre-existing `FileWriter` permission tests that also fail on main when run as root
- [x] Linters/validators pass (`make lint`: 0 issues)

## Prevention

- Validate against the schema rather than sampling records; records may be sparse.

## Related

- T-1576 (auto-schema union of columns across rows)
