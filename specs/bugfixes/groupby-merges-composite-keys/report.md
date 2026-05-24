# Bugfix Report: GroupBy Merges Distinct Composite Keys

**Date:** 2026-05-24
**Status:** Fixed

## Description of the Issue

`GroupByOp` builds a composite group key by concatenating the grouped column
values with the literal separator `||`. Because the separator can also appear
inside a value, distinct value tuples can serialize to the same key string and
get incorrectly merged into a single group. The aggregated output then reports
one group instead of two, and the group's sample column values come from
whichever tuple was seen first.

**Reproduction steps:**
1. Group a table by columns `["a", "b"]`.
2. Provide two records whose values collide under `||` joining:
   - `{a: "x||y", b: "z"}`  -> key `x||y||z`
   - `{a: "x", b: "y||z"}`  -> key `x||y||z`
3. Observe that the result has 1 group instead of 2.

A related collision exists for missing values: a missing column serialized to
the literal string `<nil>`, colliding with a record whose value is literally
the string `"<nil>"`.

**Impact:** Silent data corruption in aggregation. Any caller grouping over
string columns whose values can contain `||` (or the literal `<nil>`) could get
under-counted groups and incorrect aggregate results. Severity is moderate:
correctness bug, no crash, limited to specific data shapes.

## Investigation Summary

- **Symptoms examined:** Two distinct records merged into one group.
- **Code inspected:** `v2/operations.go` — `GroupByOp.Apply` (grouping loop) and
  `GroupByOp.createGroupKey` (key construction).
- **Hypotheses tested:** Ruled out the grouping loop and aggregate functions —
  they operate correctly on whatever keys `createGroupKey` returns. The defect
  is isolated to the key construction.

## Discovered Root Cause

`createGroupKey` produced a non-injective encoding of the value tuple. Joining
values with a fixed `||` delimiter means the boundary between values is
indistinguishable from a `||` that occurs inside a value, so different tuples
can map to the same string. The missing-value sentinel `<nil>` had the same
flaw against the literal string `"<nil>"`.

**Defect type:** Logic error — ambiguous (non-injective) serialization.

**Why it occurred:** The original implementation assumed `||` would never occur
in data, which is not guaranteed for arbitrary string columns.

**Contributing factors:** No test covered values containing the separator.

## Resolution for the Issue

**Changes made:**
- `v2/operations.go` — `GroupByOp.createGroupKey` now builds a length-prefixed
  encoding. Each value is serialized with `fmt.Sprintf("%v", val)`, then written
  to a `strings.Builder` as `<byteLen>:<value>`. Missing values are encoded with
  a distinct `nil:` marker rather than the ambiguous `<nil>` string. Because each
  segment carries its own length, the concatenation is unambiguous regardless of
  value content — no value can be confused with a delimiter.

**Approach rationale:** Length-prefixing makes the encoding injective with no
escaping rules to get wrong, and it stays a plain `string` key so the existing
`map[string][]Record` grouping is unchanged. It is also cheap (no hashing, no
allocation per value beyond the builder).

**Alternatives considered:**
- **Escaping `||` and the escape char in values** — Works but is fiddlier and
  easier to get subtly wrong than length-prefixing.
- **Hashing the joined values (e.g. SHA-256)** — Adds cost and a (tiny) collision
  risk; unnecessary when an exact injective encoding is available.
- **Using a struct/slice key** — Slices aren't comparable so can't be map keys
  directly; would require an array or a separate index structure, more invasive
  than needed.

## Regression Test

**Test file:** `v2/operations_aggregate_test.go`
**Test name:** `TestGroupByCompositeKeyCollision`

**What it verifies:**
- Two records whose values contain `||` and would collide under naive joining
  produce two distinct groups, each with count 1.
- A literal `"<nil>"` string value is not merged with a record that is missing
  the grouped column.

**Run command:** `cd v2 && go test -run TestGroupByCompositeKeyCollision`

## Affected Files

| File | Change |
|------|--------|
| `v2/operations.go` | Length-prefixed, collision-safe composite key encoding in `createGroupKey` |
| `v2/operations_aggregate_test.go` | Added `TestGroupByCompositeKeyCollision` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the regression test fails before the fix and passes after.

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer injective (length-prefixed or escaped) encodings whenever composing a
  composite key from arbitrary user data — never assume a delimiter is absent.
- Use a distinct sentinel for "missing" that cannot be produced by a real value.

## Related

- Transit ticket: T-1100
