# Bugfix Report: Optionless Table() Silently Alphabetizes Columns

**Date:** 2026-07-12
**Status:** In Progress
**Ticket:** T-1692

## Description of the Issue

v2's flagship guarantee (doc.go Key Order Preservation section, CLAUDE.md, API.md "Critical Implementation Rules", requirements 5.1-5.7) is that table key order is never alphabetized or reordered. In reality, every successful optionless `builder.Table("x", data)` call alphabetizes the columns:

- `ApplyTableOptions` defaults `autoSchema: true` (table_options.go:183)
- `NewTableContent`'s autoSchema branch (content.go:111-114) calls `DetectSchemaFromData` → `DetectSchemaFromMap`
- `DetectSchemaFromMap` does `sort.Strings(keys)` (table_options.go:144)
- `convertToRecords` (content.go:291-320) accepts only map-shaped data, so every optionless call that succeeds takes this path

Because the result is deterministically alphabetical rather than random, it looks deliberate, and users may build on the ordering without realizing the library reordered their columns.

Documentation contradicts the actual behavior in several places:

- doc.go:199 — "Keys appear in specified order, never alphabetized"
- doc.go:206 — claims `WithAutoSchema()` uses "map iteration order" (actually alphabetical)
- docs/API.md:73 — "No WithKeys - order will be random!" (actually deterministic alphabetical)
- `WithAutoSchema` godoc (table_options.go:42) — "the system will preserve the order keys appear in the source data" (impossible for Go maps)
- `DetectSchemaFromData` / `DetectSchemaFromMap` godocs — claim "preserving key order" / "preserving insertion order"

Additionally, `tableConfig.detectOrder` (table_options.go:13) is dead code: written by `WithAutoSchema`/`WithAutoSchemaOrdered`, never read anywhere.

**Reproduction steps:**
1. `builder := output.New()`
2. `builder.Table("users", []map[string]any{{"zebra": 1, "apple": 2, "mango": 3}})` — no options
3. Render: columns appear as apple, mango, zebra (alphabetized). `builder.HasErrors()` is `false`; no signal of any kind.

**Impact:** Medium severity (multi-agent code audit 2026-07-07, verified against code). The most natural first call of the API silently violates its central documented guarantee. Deterministic alphabetical output masks the reordering as intentional.

## Investigation Summary

Systematic (Fagan-style) inspection of `v2/table_options.go`, `v2/content.go`, `v2/document.go`, `v2/doc.go`, and `v2/docs/API.md`, confirming every audit claim in code.

- **Symptoms examined:** optionless table columns render alphabetically; no builder error or warning recorded; docs promise the opposite in five places.
- **Code inspected:** `ApplyTableOptions` default (table_options.go:181-189); `NewTableContent` schema branch (content.go:106-118); `DetectSchemaFromMap` sort (table_options.go:132-158); `convertToRecords` accepted types (content.go:291-320); builder error mechanism `HasErrors`/`Errors`/`addError` (document.go:148-174); `Builder.Table` (document.go:229-238).
- **Hypotheses tested:**
  - Whether any input path avoids `DetectSchemaFromMap` — no: `convertToRecords` accepts only `[]Record`, `[]map[string]any`, `map[string]any`, and `[]any` of maps, so all successful optionless calls hit the alphabetical fallback.
  - Whether `detectOrder` is read anywhere — no: only written (table_options.go:46, :55) and asserted directly in `table_options_test.go`; never consulted by production code.
  - Whether existing tests rely on `HasErrors() == false` after optionless `Table` calls — the "no errors" tests (e.g. `TestBuilder_ErrorHandling_NoErrors`, document_test.go:420) use `WithKeys`, so a recorded warning is compatible; verified by full-suite run.
  - Whether encounter order of map keys is recoverable — no: Go randomizes map iteration order per range; preserving source order for map input is impossible at runtime (per ticket, explicitly out of scope; requiring explicit order would be a v3 change).

## Discovered Root Cause

**Defect type:** Silent failure / documentation-behavior mismatch / dead code.

**Why it occurred:** The original design intended auto-detection to preserve the order keys appear in the source data — the `detectOrder` flag was scaffolded for it and the docs advertise it. That intent is unimplementable for Go maps, whose iteration order is deliberately randomized. The implementation quietly settled on `sort.Strings` as a deterministic fallback (the code comment at table_options.go:133-140 even acknowledges the limitation), but the public docs and the dead flag continued to advertise the original aspiration, and no warning channel was wired to flag the guess.

**Contributing factors:** `NewTableContent` returns only `(*TableContent, error)` with no channel for non-fatal conditions; `ApplyTableOptions` makes guessing the default rather than an opt-in; the deterministic (rather than random) result hides the problem.

## Resolution for the Issue

_To be completed after the fix is implemented._

## Regression Test

**Test file:** `v2/table_key_order_warning_test.go`
**Test names:** `TestBuilderTable_KeyOrderGuessWarning`, `TestBuilderTable_KeyOrderWarningNamesTable`

**What it verifies:**
- Optionless `Builder.Table` with multi-column map data (all accepted map shapes) records a non-fatal key-order warning retrievable via `Errors()`; the table is still added to the document.
- Explicit `WithAutoSchema()` also warns (it guesses the same way).
- `WithKeys`, `WithSchema`, and `WithAutoSchemaOrdered` do not warn.
- Single-column and empty data do not warn (no order to guess).
- The warning names the offending table.

**Run command:** `go test -run "TestBuilderTable_KeyOrder" ./...` (in `v2/`)

Confirmed failing (red) before the fix: the four warning-expected cases fail with `Errors() = []`.

## Affected Files

_To be completed after the fix is implemented._

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

**Manual verification:**
- _To be completed._

## Prevention

_To be completed._

## Related

- Transit ticket T-1692
- Transit ticket T-1451 (WithAutoSchemaOrdered skips schema detection — separate fix; `detectOrder` removal here overlaps minimally and is noted on that ticket)
- Multi-agent code audit 2026-07-07
