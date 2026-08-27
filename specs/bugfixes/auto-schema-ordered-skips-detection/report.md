# Bugfix Report: WithAutoSchemaOrdered Skips Schema Detection

**Date:** 2026-08-22
**Status:** Fixed
**Ticket:** T-1451

## Description of the Issue

`WithAutoSchemaOrdered` is documented as enabling automatic schema detection
with a custom key order. In reality it never ran detection: any data field not
listed in the custom order was silently dropped from the schema and from the
rendered output.

`WithAutoSchemaOrdered(keys...)` sets both `autoSchema = true` and
`tc.keys`, but the option-precedence switch in `newTableContent`
(content.go) checked `len(tc.keys) > 0` before `tc.autoSchema`. The keys
branch therefore always won, building the schema with
`NewSchemaFromKeys(tc.keys)` — identical to `WithKeys` — and
`DetectSchemaFromData` was never called.

**Reproduction steps:**
1. `table, _ := NewTableContent("users", []map[string]any{{"name": "Alice", "age": 30}}, WithAutoSchemaOrdered("name"))`
2. `table.Schema().GetKeyOrder()` returns `[name]`
3. Expected for an auto-schema option: all detected fields with the custom
   ordering applied, i.e. `[name age]`. The `age` column is silently missing
   from the schema and from every rendered format.

**Impact:** Medium. Any caller using `WithAutoSchemaOrdered` with a partial
key list loses data columns silently — the option behaves exactly like
`WithKeys`, contradicting its documented contract. No error or warning is
recorded.

## Investigation Summary

- **Symptoms examined:** schema built from `WithAutoSchemaOrdered("name")`
  contains only the listed keys; unlisted data fields absent from schema and
  output; no builder error recorded.
- **Code inspected:** option-precedence switch in `newTableContent`
  (v2/content.go:144-156); `WithAutoSchemaOrdered`, `WithKeys`,
  `WithAutoSchema`, `ApplyTableOptions` (v2/table_options.go);
  `DetectSchemaFromData`/`detectSchemaFromMaps` union-and-alphabetize
  behaviour (v2/table_options.go, T-1576); `NewSchemaFromKeys` and
  `Schema.SetKeyOrder` (v2/schema.go); `Builder.Table` warning wiring for
  `ErrTableKeyOrderGuessed` (v2/document.go:237-251, T-1692).
- **Hypotheses tested:** confirmed the `case tc.autoSchema:` branch contained
  a dead sub-branch `if len(tc.keys) > 0 { table.schema.SetKeyOrder(tc.keys) }`
  — unreachable because the `len(tc.keys) > 0` case is evaluated first. The
  intended merge behaviour was partially written but could never execute.
  Also confirmed existing tests asserted only the config flags
  (`tc.autoSchema`, `tc.keys`), never the resulting schema, which is why the
  dead branch went unnoticed.

## Discovered Root Cause

**Defect type:** Logic error — wrong case ordering in the option-precedence
switch, making the auto-schema-with-keys combination unreachable.

**Why it occurred:** `tableConfig` encodes `WithKeys` (keys, autoSchema=false)
and `WithAutoSchemaOrdered` (keys, autoSchema=true) with the same `keys`
field, distinguished only by `autoSchema`. The switch in `newTableContent`
dispatched on `len(tc.keys) > 0` before consulting `autoSchema`, collapsing
both options into the `NewSchemaFromKeys` path. The unreachable
`SetKeyOrder` sub-branch inside the `autoSchema` case shows the merge was
intended but was placed behind a case that could never be reached with keys
set. (Even had it been reachable, `SetKeyOrder(tc.keys)` would have replaced
the detected key order entirely rather than merging.)

**Contributing factors:**
- Tests for the option asserted only config-flag state, not final schema
  behaviour.
- The formerly-associated `detectOrder` field was dead code (removed by
  T-1692), which had made this branch easy to miss.

## Resolution for the Issue

**Changes made:**
- `v2/content.go` — added a `case tc.autoSchema && len(tc.keys) > 0:` arm
  before the plain keys case. It runs `DetectSchemaFromData(data)` and merges
  the result with the explicit keys via `newSchemaWithKeyOrder`. Removed the
  now-redundant `case tc.autoSchema:` block entirely (including its dead
  `SetKeyOrder` sub-branch); the keyless auto-schema path falls through to
  the `default` case, which performs the identical `DetectSchemaFromData`
  call. Updated the `newTableContent` godoc.
- `v2/schema.go` — new unexported helper `newSchemaWithKeyOrder(detected
  *Schema, keys []string) *Schema`: explicit keys first in the given order
  (duplicates deduplicated; keys present in the data keep their detected
  field type, keys absent from the data become untyped fields exactly like
  `WithKeys`), then all remaining detected fields appended in detection
  order — which is alphabetical (T-1692's deterministic convention).
- `v2/table_options.go` — `WithAutoSchemaOrdered` now clones the caller's
  key slice (matching the `WithKeys`/`WithSchema` defensive-copy convention,
  T-1086) and carries a full godoc contract: detection runs over the data
  (union of columns across all rows, T-1576), listed keys come first,
  remaining detected columns are appended alphabetically, and no
  `ErrTableKeyOrderGuessed` warning is recorded.
- `v2/docs/API.md` — documented `WithAutoSchemaOrdered` alongside the other
  table options. Also added to the sibling-option lists in
  `v2/docs/DOCUMENTATION.md`, the package godoc (`v2/doc.go`), and the
  contributor snippet in `v2/CLAUDE.md`/`v2/AGENTS.md`.

**Warning semantics decision (T-1692 interaction):** `WithAutoSchemaOrdered`
does NOT record `ErrTableKeyOrderGuessed`, even when unlisted remainder
columns are appended alphabetically. Rationale: the warning exists for silent
guesses — cases where the caller gave no ordering input and may not realize
an order was invented. With `WithAutoSchemaOrdered` the caller explicitly
opted into detection with a partial order, and the alphabetical remainder is
the option's documented contract, not a guess. Warning here would make the
option's primary use case (partial ordering) permanently noisy. Locked in by
the "WithAutoSchemaOrdered with partial keys does not warn" test case.

**Behaviour note:** the config state `autoSchema && len(keys) > 0` is now
interpreted declaratively from the final option state, so the combination
`WithKeys("a"), WithAutoSchema()` behaves identically to
`WithAutoSchemaOrdered("a")` (detection with "a" first) instead of silently
ignoring the trailing `WithAutoSchema()`. Precedence is not strictly
last-option-wins: the switch resolves explicit schema > detection-with-keys
> keys-only > detection, and because the auto-schema options do not clear a
previously set schema, an earlier `WithSchema` still beats a later
`WithAutoSchemaOrdered` (locked in by `TestTableOptionPrecedence`).

**Zero-keys degradation:** `WithAutoSchemaOrdered()` called with no keys
supplies no ordering contract, so it deliberately degrades to plain
`WithAutoSchema` behaviour — all columns alphabetized and the
`ErrTableKeyOrderGuessed` warning recorded as usual. Documented in the
option's godoc and locked in by `TestWithAutoSchemaOrdered_ZeroKeys` plus
the "zero keys warns" warning-test case.

**Approach rationale:** the switch is the single decision point for schema
construction; adding an explicit arm for the auto+keys combination fixes the
precedence at its root with no API changes. The merge helper lives next to
the other schema constructors in schema.go.

**Alternatives considered:**
- Reusing the dead `SetKeyOrder(tc.keys)` sub-branch by reordering cases —
  rejected: `SetKeyOrder` replaces the key order outright, so unlisted
  detected columns would be dropped from the order (the same bug in a
  different spot) while their fields lingered in `Schema.Fields`.
- Warning (`ErrTableKeyOrderGuessed`) when remainder columns are appended —
  rejected: see warning semantics decision above.
- A separate `orderedKeys` field on `tableConfig` to distinguish the options
  — rejected: the `autoSchema` flag already disambiguates, and a second field
  reintroduces the kind of redundant state that made `detectOrder` dead code.

## Regression Test

**Test file:** `v2/table_auto_schema_ordered_test.go`
**Test names:** `TestWithAutoSchemaOrdered_SchemaDetection`,
`TestWithAutoSchemaOrdered_PreservesDetectedTypes`,
`TestWithAutoSchemaOrdered_ZeroKeys`,
`TestWithAutoSchemaOrdered_DoesNotRetainCallerSlice`,
`TestTableOptionPrecedence`

**What they verify:** the ticket reproduction (unlisted field detected);
explicit keys first with alphabetical remainder; full explicit list keeps
given order; missing explicit keys kept untyped like `WithKeys`; column
union across rows (T-1576); duplicate explicit keys deduplicated; `[]Record`
input; detected field types preserved for listed keys; zero-keys degradation
to plain `WithAutoSchema` behaviour; caller's key slice not retained; and
the option-precedence switch (explicit schema wins regardless of position,
detection-with-keys next, keys-only last — including the documented
`WithKeys` + `WithAutoSchema` equivalence with `WithAutoSchemaOrdered`).

Additionally `v2/table_key_order_warning_test.go` gained two cases:
"WithAutoSchemaOrdered with partial keys does not warn" (locking the warning
semantics decision) and "WithAutoSchemaOrdered with zero keys warns like
WithAutoSchema" (locking the degradation semantics).

**Run command:** `go test -run "TestWithAutoSchemaOrdered_|TestTableOptionPrecedence|TestBuilderTable_KeyOrderGuessWarning" ./...` (in `v2/`)

## Affected Files

| File | Change |
|------|--------|
| `v2/content.go` | New switch arm for autoSchema+keys; dead sub-branch removed; godoc updated |
| `v2/schema.go` | New `newSchemaWithKeyOrder` merge helper |
| `v2/table_options.go` | `WithAutoSchemaOrdered` clones keys; full godoc contract |
| `v2/table_auto_schema_ordered_test.go` | New regression tests |
| `v2/table_key_order_warning_test.go` | Warning-semantics cases for partial and zero keys |
| `v2/docs/API.md` | Document `WithAutoSchemaOrdered` |
| `v2/docs/DOCUMENTATION.md` | Add `WithAutoSchemaOrdered` to the Table Options reference |
| `v2/doc.go` | Package godoc: third way to specify key order |
| `v2/CLAUDE.md` (+ `v2/AGENTS.md` symlink) | Add option to the functional-options snippet |
| `CHANGELOG.md` | Fixed entry under Unreleased |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`make test`)
- [x] Linters pass (`make lint`)

**Manual verification:**
- Ticket reproduction sketch re-run: `WithAutoSchemaOrdered("name")` on
  `{name, age}` data now yields key order `[name age]`.

## Prevention

**Recommendations to avoid similar bugs:**
- When an option sets multiple config fields, test the resulting artifact
  (schema/output), not just the config flags — the original tests asserted
  flags only and missed that the combination was never honoured.
- Treat unreachable branches as red flags: the dead `SetKeyOrder` sub-branch
  encoded the intended behaviour but could never run. Linters do not catch
  semantically dead switch sub-branches; a covering test would have.

## Related

- Transit ticket T-1451 (this fix)
- T-1692 — `ErrTableKeyOrderGuessed` warning and alphabetical convention
  (`specs/bugfixes/optionless-table-alphabetizes/report.md`)
- T-1576 — detection scans all rows, union of columns
  (`specs/bugfixes/auto-schema-union-columns/report.md`)
- T-1086 — defensive-copy convention for option slices
