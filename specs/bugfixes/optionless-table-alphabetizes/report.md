# Bugfix Report: Optionless Table() Silently Alphabetizes Columns

**Date:** 2026-07-12
**Status:** Fixed
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

The alphabetical fallback itself is intentionally unchanged — Go map key order is unrecoverable, and per the ticket, requiring explicit order for map data is only viable as a v3 change. The fix makes the guess visible and the documentation honest, and removes the dead scaffold:

**Changes made:**
- `v2/content.go` - Added the `ErrTableKeyOrderGuessed` sentinel (works with `errors.Is`). Refactored the constructor into an internal `newTableContent` that additionally reports whether auto-detection had to guess (alphabetize) the key order — true when neither `WithSchema` nor `WithKeys` was supplied and the data yielded more than one column. Public `NewTableContent` signature is unchanged; its godoc now states the alphabetical fallback.
- `v2/document.go` - `Builder.Table` now records `table %q: ErrTableKeyOrderGuessed` as a non-fatal builder error when the order was guessed. The table is still added and renders normally. Godoc documents the warning and how to filter it.
- `v2/table_options.go` - Removed the dead `tableConfig.detectOrder` field and its writes in `WithAutoSchema`/`WithAutoSchemaOrdered`. Corrected the godocs of `WithAutoSchema` (no longer claims source-order preservation), `DetectSchemaFromData`, and `DetectSchemaFromMap` (both now state the alphabetical fallback).
- `v2/doc.go` - Key Order Preservation section now distinguishes explicit order (never reordered) from auto-detection (alphabetical fallback plus builder warning); no longer claims "map iteration order".
- `v2/docs/API.md` - The "INCORRECT" example no longer claims "order will be random!"; it states columns are alphabetized and a warning is recorded. The Key Order Preservation bullets qualify the "never alphabetized" guarantee as applying to explicitly specified order.
- `v2/CLAUDE.md` - Same qualification of the "never alphabetized" guarantee.
- `v2/table_options_test.go` - Removed assertions on the deleted `detectOrder` field.
- `CHANGELOG.md` - Entry under Unreleased/Fixed; also removed a stray `<<<<<<< HEAD` conflict marker committed on main at the top of that section.

**Approach rationale:** The ticket prescribes this exact fix: correct the docs, signal the guess through the existing builder error mechanism (`Errors()`), and remove the dead flag. An internal three-value constructor keeps the public `NewTableContent` API unchanged while giving `Builder.Table` the signal it needs. A sentinel error keeps the warning non-fatal in practice: callers who accept alphabetical order can filter it with `errors.Is` without string matching. Zero- and one-column tables do not warn because there is no order to guess.

**Alternatives considered:**
- Preserve source key order for map input - Impossible: Go randomizes map iteration and the encounter order is unrecoverable at runtime (explicitly out of scope per ticket).
- Make optionless map input an error (require explicit order) - Breaking change to the most common call; only viable for v3 (per ticket).
- Store a `keyOrderGuessed` field on `TableContent` instead of a second constructor - Adds state that `Clone`/sealing would have to reason about; the constructor-return approach keeps the flag transient.
- Separate warnings channel on the Builder - New API surface for one warning; the ticket explicitly asks for the existing `Errors()` mechanism.

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

| File | Change |
|------|--------|
| `v2/content.go` | `ErrTableKeyOrderGuessed` sentinel; internal `newTableContent` reporting the guess; corrected `NewTableContent` godoc |
| `v2/document.go` | `Builder.Table` records the non-fatal warning; godoc documents it |
| `v2/table_options.go` | Removed dead `detectOrder` field; corrected `WithAutoSchema`/`DetectSchemaFromData`/`DetectSchemaFromMap` godocs |
| `v2/doc.go` | Corrected Key Order Preservation section |
| `v2/docs/API.md` | Corrected "order will be random!" example and "never alphabetized" bullet |
| `v2/CLAUDE.md` | Qualified the "never alphabetized" guarantee |
| `v2/table_options_test.go` | Removed `detectOrder` assertions |
| `v2/table_key_order_warning_test.go` | New regression tests |
| `CHANGELOG.md` | Entry under Unreleased/Fixed; removed stray conflict marker |

## Verification

**Automated:**
- [x] Regression test passes (`go test -run "TestBuilderTable_KeyOrder" ./...` — red before fix, green after)
- [x] Full test suite passes (`make test` and `make test-integration`)
- [x] Linters/validators pass (`golangci-lint run`: 0 issues; `go fmt ./...`: clean)

Note: `make fmt` fails on the example modules with a pre-existing `go mod tidy` complaint unrelated to this change (fails identically on a clean tree), so formatting and lint were run directly on `v2/`.

**Manual verification:**
- Confirmed the warning message names the table and reads actionably: `table "x": key order auto-detected from map data and alphabetized; pass WithKeys or WithSchema to control column order`.
- Confirmed `WithAutoSchemaOrdered` (keys supplied) and explicit `WithKeys`/`WithSchema` never warn, and zero/one-column data never warns.

## Prevention

**Recommendations to avoid similar bugs:**
- When a documented guarantee cannot be implemented (here: preserving Go map key order), change the docs and add a runtime signal instead of silently substituting different behavior.
- Avoid speculative config fields (`detectOrder` was written but never read for years); wire a flag to behavior in the same change that introduces it, or don't add it.
- Non-fatal conditions surfaced through `Builder.Errors()` should wrap a sentinel so callers can filter with `errors.Is` rather than string matching.

## Related

- Transit ticket T-1692
- Transit ticket T-1451 (WithAutoSchemaOrdered skips schema detection — separate fix; `detectOrder` removal here overlaps minimally and is noted on that ticket)
- Multi-agent code audit 2026-07-07
