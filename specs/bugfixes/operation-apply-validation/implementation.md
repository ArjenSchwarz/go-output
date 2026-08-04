# Implementation Explanation: Operation Apply Validation (T-1502)

Branch: `T-1502/bugfix-operation-apply-validation` — explains the change at three expertise levels, generated as part of the pre-push review.

## Beginner Level

### What Changed
The go-output library lets you run "operations" on tables: filter rows, sort them, limit how many you keep, group them, or add a calculated column. Each operation already knew how to check its own settings for mistakes (a `Validate()` method), but that check only ran when the library itself applied operations during rendering. If you called an operation's `Apply` method directly with broken settings — say, a filter with no filter function, or a limit of -1 — the program crashed (a panic) or quietly did the wrong thing.

This fix makes every operation check its own settings first, at the top of `Apply`. Broken settings now come back as a normal error you can handle, instead of a crash.

### Why It Matters
A library that panics crashes the whole application using it. Returning an error instead lets the caller decide what to do — log it, show a message, fall back. And the silent case was arguably worse: asking for a sort in an invalid direction just sorted ascending without telling you, so you could ship wrong output without noticing.

### Key Concepts
- **Panic vs error**: Go programs signal problems two ways. An *error* is a value handed back for the caller to inspect. A *panic* is an emergency stop that unwinds the program. Libraries should return errors for bad input, not panic.
- **Validation**: checking configuration is sensible before using it — like checking a form is filled in before submitting it.
- **Public API**: methods outside callers are allowed to use. Once `Apply` is public, you cannot assume "someone else validated first".

---

## Intermediate Level

### Changes Overview
- `v2/operations.go` — each of the five operations (`FilterOp`, `SortOp`, `LimitOp`, `GroupByOp`, `AddColumnOp`) gained the same guard at the top of `Apply`: call `o.Validate()`, return its error before touching content or config. `ApplyWithFormat` is a one-line delegation to `Apply` in all five, so it inherits the check.
- `v2/operations_apply_validation_test.go` — new map-based table tests: ten invalid-config cases for `Apply`, five for `ApplyWithFormat`, plus a valid-config sanity test proving the wiring rejects nothing legitimate.
- `CHANGELOG.md` — Fixed entry under Unreleased.
- `specs/bugfixes/operation-apply-validation/report.md` — bugfix report with investigation and root cause.
- Doc polish from this review: the five `Apply` godocs now state the validation contract, and `v2/docs/BEST_PRACTICES.md` no longer implies validation only happens during rendering.

### Implementation Approach
The `Validate()` methods already existed with complete checks and structured `*ValidationError` returns; the renderer pipeline (`applyContentTransformations` in `v2/renderer.go`) was the only caller. The fix is pure wiring: enforce the existing contract inside the entry point itself. This mirrors T-1142, which added per-operation nil-content guards inside `Apply` for the same reason — public entry points cannot rely on caller discipline.

### Trade-offs
- **Double validation on the render path**: the pipeline still validates before calling `Apply`, so pipeline-applied operations validate twice. Kept deliberately — the pipeline's wrapping adds content-ID and operation-index context to its error messages, and all five `Validate` implementations are a handful of nil/len comparisons with no success-path allocations. Removing the pipeline call would change existing error text for no measurable gain.
- **Ad-hoc guards rejected**: checking individual fields inline (`if o.predicate == nil`) would duplicate `Validate` logic and drift; it also would not fix the silent invalid-sort-direction case without re-implementing that check.
- **Validate-before-nil-content ordering**: an invalid op applied to nil content now reports the config error rather than the content error. Config errors are the more fundamental complaint; both are structured `*ValidationError`s.

---

## Expert Level

### Technical Deep Dive
The defect class is a decoupled validate/apply contract: `Operation.Validate()` existed for the pipeline's benefit, and `Apply` assumed post-validation state, dereferencing config unconditionally. Failure modes before the fix: nil function pointer panics (`FilterOp.predicate`, `AddColumnOp.fn`, nil `AggregateFunc` values in `GroupByOp`), a negative slice bound (`LimitOp` `records[:o.count]`), and a semantic silent failure — `SortOp` treating any unknown `SortDirection` as ascending. The last is the interesting one: it is unreachable via the pipeline (validation rejects it there), so only direct `Apply` callers ever hit it, which is precisely why it survived until now.

The guard is placed before the nil-content check and the `*TableContent` type assertion, so `Validate` is now the first observable effect of `Apply` in all five operations. `Validate` implementations are pure field reads over construction-time config — no mutation, no allocation on success — so the guard adds no race surface (operations remain safe for concurrent use across renders) and no measurable cost against the mandatory full-table `Clone` each `Apply` already performs.

### Architecture Impact
Every entry point — direct `Apply`, direct `ApplyWithFormat`, pipeline — now enforces one contract, and new `Operation` implementations have an established pattern to follow (start `Apply` with `Validate()`). Error surface is unchanged in kind: callers already had to handle `*ValidationError` from the pipeline; direct callers now get the same type instead of a panic. Pipeline error text is byte-for-byte unchanged since its pre-validation fires first.

### Potential Issues
- The validate-before-nil-content precedence is intentional but not pinned by a test; a future reorder would pass the suite silently.
- Tests assert error-message substrings rather than `errors.As(*ValidationError)`; rewording a validation message breaks tests without a behaviour change.
- Three `GroupByOp.Validate` branches (empty column name in a non-empty list, empty aggregates map, empty aggregate-function name) are enforced but untested.
- T-1465 (SortOp on sparse tables) is adjacent and deliberately untouched; no comparison logic changed here.

---

## Completeness Assessment

**Fully implemented**: all six defect cases from the ticket (four panic paths, silent sort direction, silently-accepted empty configs) are fixed via the `Validate` guard in all five operations; `ApplyWithFormat` covered by delegation; regression tests red-before/green-after; CHANGELOG and bugfix report accurate against the code; godocs and BEST_PRACTICES updated in this review.

**Partially implemented**: test coverage of the validation surface — ten of the thirteen reachable `Validate` failure branches are asserted; the three `GroupByOp` branches above are not, and the structured-error type is asserted only via message text.

**Missing**: nothing required by the ticket. Follow-up candidates (out of scope, test-file changes): pin the validate/nil-content precedence, use the exported `FormatAwareOperation` interface in the `ApplyWithFormat` test, `errors.As` assertions, and the pre-existing non-compiling examples in `v2/docs/API.md` (`NewGroupByOp`/`NewAddColumnOp`).
