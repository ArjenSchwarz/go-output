# Implementation Explanation: Auto Schema Covers Union of Columns (T-1576)

Branch: `T-1576/bugfix-auto-schema-union-columns`
Commits reviewed: `553d01c`, `a3bc436`, `e1589b3`, `eab0123` (base `ce3c190`)

---

## Beginner Level

### What Changed

The go-output library turns lists of data (like a list of people with names
and emails) into tables. When you don't tell it which columns to use, it
looks at your data and figures out the columns by itself — this is called
"automatic schema detection".

Before this fix, the library only looked at the **first row** of your data to
decide which columns exist. If a later row had an extra column — say the first
person has only a `name`, but the second person also has an `email` — that
extra column was invisible. The table printed without the email column, and
the email value simply disappeared from the output.

Now the library looks at **every row** and combines all the column names it
finds (the "union" of the columns). Nothing gets dropped: a row that doesn't
have a value for some column just shows an empty cell there.

### Why It Matters

This was silent data loss. There was no error and no warning — data you put
in just never came out. Anyone building tables from "sparse" data (rows that
don't all have the same fields, which is very common with JSON or API
responses) could ship reports with missing values and never know.

### Key Concepts

- **Schema**: the list of columns a table has, in order. Think of it as the
  header row of a spreadsheet.
- **Auto-detection**: when you don't specify columns yourself, the library
  inspects your data and builds the schema for you.
- **Union of keys**: combining the column names from every row, so a column
  that appears in *any* row makes it into the table.
- **Alphabetical ordering**: Go's maps (the data structure holding each row)
  don't remember the order you wrote the keys in, so the library sorts column
  names alphabetically to give the same order every time. If you care about
  a specific column order, you pass `WithKeys(...)` or `WithSchema(...)`.
- **Warning, not error**: when the library had to guess the column order, it
  records a retrievable warning (`ErrTableKeyOrderGuessed`) instead of
  failing — your table still renders.

---

## Intermediate Level

### Changes Overview

- `v2/table_options.go` — the core fix. `DetectSchemaFromData` previously
  did `DetectSchemaFromMap(v[0])` for `[]Record` and `[]map[string]any`, and
  inspected only `v[0]` for `[]any`. All three slice cases now scan every
  row. The scanning logic lives in one new unexported helper:

  ```go
  func detectSchemaFromMaps[M ~map[string]any](rows []M) *Schema
  ```

  It is generic over `~map[string]any`, so `[]Record` (a named map type) and
  `[]map[string]any` share one implementation without conversion loops.
  `DetectSchemaFromMap` now delegates to the same helper with a single-row
  slice, so there is exactly one place that builds map-derived schemas.
- `v2/content.go` — `NewTableContent` doc comment updated to state that
  auto-detection covers the union of keys across all rows.
- `v2/table_sparse_columns_test.go` — new regression tests (map-based table
  tests per project convention): union across `[]map[string]any`, `[]Record`,
  and `[]any`; disjoint rows; non-map elements skipped in `[]any`;
  type-from-first-occurrence; the ticket's end-to-end repro via `AppendText`;
  and the `ErrTableKeyOrderGuessed` warning firing for sparse rows.
- `CHANGELOG.md` — Unreleased/Fixed entry.

### Implementation Approach

`detectSchemaFromMaps` makes a single pass over all rows, filling a
`types map[string]string` keyed by column name. The first row containing a
key determines that column's type (`DetectType(val)` runs only on first
sight — the `if _, seen := types[k]` check makes the map double as the
seen-set). Keys are then sorted alphabetically and turned into `Field`
entries.

The `[]any` case collects the `map[string]any` elements into a slice first
and passes them to the same helper. Non-map elements are skipped during
detection — deliberately, with a comment explaining why: `convertToRecords`
rejects `[]any` input containing non-map elements with an error, so a schema
detected from such input is discarded before anything renders. Detection
just has to not panic or bail out early.

No change was needed in `newTableContent`. Its existing T-1692 logic —
`keyOrderGuessed := tc.schema == nil && len(tc.keys) == 0 && len(table.schema.GetKeyOrder()) > 1`
— keys off the *detected* order length, so the warning now correctly fires
when the multi-row union produced multiple columns even though row 0 alone
had one column (previously no warning fired for that shape, because
detection saw only one key).

### Trade-offs

- **Union vs. error on sparse rows**: erroring would break legitimate
  callers — sparse rows are normal data. The union loses nothing; missing
  cells render empty, which renderers already handle.
- **Alphabetical vs. first-row order + appended later keys**: first-row map
  order is unrecoverable in Go anyway, and T-1692 just established the
  deterministic-alphabetical convention for map-derived schemas. Preserving a
  fake "first-row order" would be inconsistent and no more meaningful.
- **Full scan vs. sampling**: detection is now O(total cells) instead of
  O(first-row cells). This runs once per table build, and the constant work
  per cell is a map lookup; type detection still runs only once per column.

---

## Expert Level

### Technical Deep Dive

The generic constraint `~map[string]any` is the load-bearing detail: `Record`
is `map[string]any` under a named type, and without the tilde the `[]Record`
case would need an element-wise copy into `[]map[string]any` just to call the
helper. With it, both slice cases dispatch directly and the `[]any` case pays
one filtered copy (unavoidable — the elements must be type-asserted anyway).

Type detection semantics changed subtly and the tests pin them: a column's
`Field.Type` comes from the first row *containing that key*, not from row 0.
For heterogeneous values across rows (e.g. `score` is `3.5` in one row and
`"high"` in another) the first occurrence wins (`float`). This is
first-occurrence-in-slice-order, not alphabetical-first — iteration over rows
is ordered even though iteration within a row's map is not, so the result is
deterministic.

The review round (commits `e1589b3`, `eab0123`) tightened two things: a local
variable that shadowed an outer identifier was renamed, and an initial
two-map implementation (separate seen-set + types map) was collapsed into the
single `types` map. The exported docs (`DetectSchemaFromData`,
`WithAutoSchema`) now state the union behaviour and the []any non-map skip,
so the contract is visible without reading the implementation.

### Architecture Impact

- **API surface**: no signatures changed. `DetectSchemaFromData` and
  `DetectSchemaFromMap` keep their contracts; behaviour is a strict widening
  (more columns detected, never fewer). `DetectSchemaFromMap` becoming a
  delegation means any future change to map-schema construction happens in
  one place.
- **Empty-input handling**: the explicit `len(v) == 0` early-returns were
  removed; `detectSchemaFromMaps` of an empty slice naturally yields
  `&Schema{Fields: []Field{}, keyOrder: []string{}}` — same result, less
  branching. The trailing `return` for unrecognized input types is unchanged.
- **Warning coupling**: `keyOrderGuessed` in `newTableContent` now fires for
  a strictly larger set of inputs (sparse single-key-first-row data). That is
  correct — the order genuinely was guessed — but callers that treat
  `Builder.Errors()` as must-be-empty will start seeing the non-fatal
  `ErrTableKeyOrderGuessed` for data that previously produced none. The
  sentinel is documented as filterable with `errors.Is`.

### Potential Issues

- **Behavioural change for existing sparse-data callers**: output now
  includes columns that were previously (wrongly) dropped, and the column
  set/order can differ from before. This is the point of the fix, but it is
  a visible output change, correctly flagged in the CHANGELOG.
- **First-occurrence type for mixed-type columns**: reasonable and
  deterministic, but a column whose first value is atypical (e.g. a null-ish
  placeholder) gets that type. Pre-existing behaviour for homogeneous
  detection; now merely extended to later rows.
- **T-1451 (open, out of scope)**: `WithAutoSchemaOrdered` skips schema
  detection entirely; sparse data through that path is not covered by this
  fix.

---

## Completeness Assessment

**Fully implemented:**
- Union-of-keys detection for all three slice input shapes (`[]Record`,
  `[]map[string]any`, `[]any`), with alphabetical ordering per T-1692.
- Type detection from first row containing each key.
- Non-map `[]any` elements skipped during detection, documented in both the
  exported doc comment and an inline comment, with the `convertToRecords`
  rejection rationale.
- `ErrTableKeyOrderGuessed` fires for sparse data where the union made the
  order a guess.
- Regression tests covering every case above plus the ticket's end-to-end
  repro; CHANGELOG entry; doc comments on all touched exported symbols.

**Partially implemented / missing:** nothing within the ticket's scope.
T-1451 (`WithAutoSchemaOrdered` skips detection) is a separate open ticket
explicitly excluded in the report.

**Divergences from design:** none found — the implementation matches the
resolution described in `report.md`, including both rejected alternatives.
