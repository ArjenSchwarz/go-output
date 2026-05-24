# Bugfix Report: Schema Key Order Leaks Mutable Slices

**Date:** 2026-05-24
**Status:** Fixed

## Description of the Issue

The v2 schema/key-order APIs exposed and stored mutable caller-owned slices. This
broke the documented immutability of `Document`/`TableContent` and allowed the
rendered column order of a table to change after the table had been created
(and after `Build()`).

**Reproduction steps:**
1. Create a table with `output.WithKeys("a", "b", "c")` (or build a `Schema` via
   `NewSchemaFromKeys`/`NewSchemaFromFields`).
2. Either mutate the slice that was passed in, mutate the slice returned by
   `Schema().GetKeyOrder()`, or call `table.Schema().SetKeyOrder(...)` on a built
   table.
3. Observe that subsequent JSON/CSV/table/markdown/HTML renders use the changed
   key order, violating the immutability/key-order guarantees.

**Impact:** Medium. Any caller that retains a reference to an input slice, the
returned key-order slice, or the schema pointer could silently corrupt the column
order of an already-constructed table. The corruption is non-obvious because it
happens through aliasing rather than an explicit API call.

## Investigation Summary

- **Symptoms examined:** Aliasing of caller-owned slices into schema internals and
  exposure of schema internals to callers.
- **Code inspected:** `v2/schema.go` (`GetKeyOrder`, `SetKeyOrder`,
  `NewSchemaFromKeys`, `NewSchemaFromFields`), `v2/table_options.go` (`WithKeys`,
  `WithSchema`), `v2/content.go` (`NewTableContent`, `TableContent.Schema()`), and
  all read-side callers in the renderers/operations.
- **Hypotheses tested:** Confirmed each leak point individually with targeted unit
  tests that mutate a caller-owned slice and assert the schema's internal state is
  unaffected. Verified that all renderer/operation call sites only read from the
  schema (via `GetKeyOrder()`/`Fields`), so returning a copy from `Schema()` does
  not break internal behaviour.

## Discovered Root Cause

The schema type stored slices by reference and returned them by reference without
defensive copying. Go variadic spreads (`WithKeys(keys...)`, `WithSchema(fields...)`)
pass the caller's backing array rather than a fresh copy, so even the options API
leaked caller state.

**Defect type:** Missing defensive copy / reference aliasing (immutability violation).

**Why it occurred:** The schema constructors and accessors were written to assign
and return slices directly. Slices in Go are reference types, so storing/returning
them without copying shares the underlying backing array with the caller.

**Contributing factors:** `keyOrder` is computed independently of `Fields`, which
masked the `Fields` leak in any test that only checked key order — so the `Fields`
aliasing was easy to miss.

## Resolution for the Issue

**Changes made:**
- `v2/schema.go` `GetKeyOrder()` - returns a copy of the key-order slice (and of the
  extracted key order when `keyOrder` is nil).
- `v2/schema.go` `SetKeyOrder()` - clones the caller slice before storing.
- `v2/schema.go` `NewSchemaFromKeys()` - clones the caller `keys` slice.
- `v2/schema.go` `NewSchemaFromFields()` - clones the caller `fields` slice.
- `v2/table_options.go` `WithSchema()` - clones the variadic `fields` slice.
- `v2/table_options.go` `WithKeys()` - clones the variadic `keys` slice.
- `v2/content.go` `TableContent.Schema()` - returns a defensive copy of the schema so
  callers cannot mutate the built table's schema internals.

**Approach rationale:** Defensive copying on both input and output is the minimal,
localized fix that restores the documented immutability without changing any public
signatures. Returning a schema copy from `Schema()` prevents post-build mutation
while keeping all internal read-only callers working unchanged.

**Alternatives considered:**
- Returning an unexported read-only schema view - More invasive; would require a new
  type and touch every read call site. Rejected for being heavier than necessary.
- Documenting the slices as "do not mutate" instead of copying - Rejected because it
  does not actually enforce immutability and contradicts the existing `Records()`
  pattern which already returns copies.

## Regression Test

**Test files:** `v2/schema_test.go`, `v2/table_options_test.go`
**Test names:** `TestSchemaKeyOrderDefensiveCopy`, `TestTableSchemaDefensiveCopy`

**What it verifies:** That mutating an input slice (`NewSchemaFromKeys`,
`SetKeyOrder`, `NewSchemaFromFields`, `WithKeys`, `WithSchema`), mutating the slice
returned from `GetKeyOrder()`, or calling `SetKeyOrder()` on a built table's schema
does not alter the table's internal key order or fields.

**Run command:**
`cd v2 && go test . -run 'TestSchemaKeyOrderDefensiveCopy|TestTableSchemaDefensiveCopy'`

## Affected Files

| File | Change |
|------|--------|
| `v2/schema.go` | Clone slices in `GetKeyOrder`, `SetKeyOrder`, `NewSchemaFromKeys`, `NewSchemaFromFields` |
| `v2/table_options.go` | Clone variadic slices in `WithKeys` and `WithSchema` |
| `v2/content.go` | Return a defensive schema copy from `TableContent.Schema()` |
| `v2/schema_test.go` | Add `TestSchemaKeyOrderDefensiveCopy` regression test |
| `v2/table_options_test.go` | Add `TestTableSchemaDefensiveCopy` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed all renderer/operation call sites only read from the schema, so the
  `Schema()` copy does not change rendered output.

## Prevention

**Recommendations to avoid similar bugs:**
- Treat all slice/map fields on "immutable" types as copy-on-input and copy-on-output,
  matching the existing `Records()` pattern.
- When testing immutability, mutate the underlying field directly (e.g. `schema.Fields`)
  rather than only checking derived values that may be computed independently.

## Related

- Transit ticket T-1086
