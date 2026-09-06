# Implementation Explanation: WithAutoSchemaOrdered Runs Schema Detection (T-1451)

Branch: `T-1451/bugfix-auto-schema-ordered` vs `origin/main`

## Beginner Level

### What Changed
The go-output library turns lists of data (like rows of users) into tables in
many formats — JSON, CSV, Markdown, and more. When you build a table you can
tell the library which columns to show and in what order, or you can let it
figure the columns out from the data ("auto-detection").

One option, `WithAutoSchemaOrdered`, promised the best of both: "detect all
the columns from my data, but put the ones I name first." Due to a bug, it
never actually detected anything — it behaved exactly like the plain "only
show these columns" option. If your data had a column you didn't name, that
column silently vanished from the output.

The fix makes the option do what it always said: every column found in the
data appears in the table, with the ones you named first, and the rest after
them in alphabetical order.

### Why It Matters
Silently losing data columns is the worst kind of bug — nothing crashes, no
warning appears, the output just quietly misses information. Anyone who used
this option with a partial list of column names was shipping incomplete
tables without knowing it.

### Key Concepts
- **Schema**: the table's blueprint — which columns exist, their order, and
  their types. Think of it as the header row plus formatting rules.
- **Auto-detection**: scanning the data to discover the columns, like reading
  every row of a spreadsheet to compile the full set of column headings.
- **Option (functional option)**: a small configuration knob passed when
  creating a table, e.g. `WithKeys("name", "age")`. Several knobs can be
  combined, and the library decides what the combination means.

---

## Intermediate Level

### Changes Overview
- `v2/content.go` — `newTableContent`'s option-precedence switch gains an arm
  `case tc.autoSchema && len(tc.keys) > 0:` ahead of the plain-keys case; it
  runs `DetectSchemaFromData(data)` and merges with the explicit keys. The
  old `case tc.autoSchema:` block (which contained an unreachable
  `SetKeyOrder` sub-branch) is folded into `default`.
- `v2/schema.go` — new unexported helper `newSchemaWithKeyOrder(detected
  *Schema, keys []string)`: explicit keys first (deduplicated), detected
  field definitions reused for listed keys, untyped `Field{Name: key}` for
  keys absent from the data, then the detected remainder appended in
  detection order (alphabetical).
- `v2/table_options.go` — `WithAutoSchemaOrdered` now clones the caller's
  slice (`slices.Clone`) and carries a full godoc contract, including the
  zero-keys degradation.
- Tests: `v2/table_auto_schema_ordered_test.go` (5 functions) and two new
  cases in `v2/table_key_order_warning_test.go`.
- Docs: `v2/docs/API.md`, `v2/docs/DOCUMENTATION.md`, `v2/doc.go`,
  `v2/CLAUDE.md` (`AGENTS.md` symlink), `CHANGELOG.md`, and the bugfix
  report.

### Implementation Approach
The root cause was case ordering in a switch: `WithAutoSchemaOrdered` sets
both `autoSchema=true` and `keys`, but the switch tested `len(tc.keys) > 0`
before `tc.autoSchema`, so the keys-only branch always won. The fix treats
the final config state declaratively — the `(autoSchema, keys)` combination
gets its own arm — rather than adding new config fields. The merge lives in
a dedicated schema constructor beside its siblings (`NewSchemaFromKeys`,
`NewSchemaFromFields`).

Warning semantics were decided deliberately: `ErrTableKeyOrderGuessed`
exists for silent guesses (no ordering input at all). With this option the
caller opted into detection under a documented ordering contract, so no
warning is recorded for the alphabetical remainder — but the zero-keys form
degrades to plain `WithAutoSchema` and does warn.

### Trade-offs
- **Declarative state vs. positional options**: `WithKeys("a")` followed by
  `WithAutoSchema()` now means "detect, a first" instead of silently
  ignoring the second option. Cheap, predictable, but not strictly
  last-option-wins: `WithSchema` is sticky (nothing clears `tc.schema`), so
  an earlier `WithSchema` beats a later `WithAutoSchemaOrdered`. This is
  test-locked in `TestTableOptionPrecedence`.
- **No warning for partial keys**: warning would make the option's primary
  use case permanently noisy; the alternative (warn on any appended
  remainder) was rejected and the choice is locked by a test.
- **Rejected**: reusing the dead `SetKeyOrder` sub-branch (would drop
  unlisted columns from the key order — same bug elsewhere); a separate
  `orderedKeys` config field (redundant state, the same trap that produced
  the dead `detectOrder` field removed in T-1692).

---

## Expert Level

### Technical Deep Dive
`newSchemaWithKeyOrder` is a stable merge: one pass over `keys` with a
`listed` set (dedup after first occurrence), `detected.FindField` to reuse
the detected `Field` (type) by value-copy, `Field{Name: key}` for absent
keys — bit-for-bit the `NewSchemaFromKeys` degradation — then one pass over
`detected.Fields` appending unlisted fields in detection order, which
`detectSchemaFromMaps` guarantees is alphabetical (T-1692). `keyOrder` is
derived from the merged fields via `extractKeyOrder`, so Fields/keyOrder
cannot skew. Detection itself is the T-1576 union across all rows, so
columns first appearing in later rows are included.

Edge cases verified: duplicate listed keys (deduped); empty/unsupported data
(`DetectSchemaFromData` returns an empty non-nil schema — listed keys become
untyped, remainder loop no-ops); zero keys (falls through to `default`,
alphabetized, warns); `[]Record` and `[]map[string]any` inputs; detected
fields can never carry `Formatter` or `Hidden`, so the value-copy shares no
mutable state and `extractKeyOrder`'s hidden-skip can't diverge.

The `keyOrderGuessed` predicate (`tc.schema == nil && len(tc.keys) == 0 &&
len(keyOrder) > 1`) is unchanged in shape but its comment now carries the
contract: keys present suppress the warning even though detection ran.

### Architecture Impact
No API surface change; one unexported helper. The switch remains the single
decision point for schema construction, now with precedence explicit schema
> detection-with-keys > keys-only > detection. The behavioral side effect —
`WithKeys(...)` + `WithAutoSchema()` ≡ `WithAutoSchemaOrdered(...)` — is a
contract change for anyone who relied on the trailing option being ignored;
it is CHANGELOG-documented and test-pinned.

### Potential Issues
- `WithSchema` stickiness: auto-schema options do not reset `tc.schema`, so
  option order is immaterial once a schema is set. Deterministic and now
  test-locked, but worth remembering when adding future schema options.
- The option-level `slices.Clone` is convention-consistency, not the actual
  independence guarantee (schema construction builds fresh slices in the
  same call stack); the test comment now says so.
- Regression tests assert at the schema level (`Schema().GetKeyOrder()`,
  `FindField().Type`), not rendered output. Schema→render propagation is
  covered broadly elsewhere in the suite, so this is an accepted proxy.

## Completeness Assessment

**Fully implemented:** detection merge with explicit ordering (all contract
points tested: dedup, detected types, untyped absent keys, alphabetical
remainder, row-union); no-warning-for-partial-keys and warn-on-zero-keys
semantics; defensive copy; precedence matrix; documentation across all
sibling-option lists; CHANGELOG and bugfix report aligned with the code
after review fixes.

**Partially implemented:** none.

**Missing:** a render-level end-to-end assertion for the recovered column
(accepted gap — schema-level assertions plus existing schema→render coverage
lock the fix; the symptom test would add little).
