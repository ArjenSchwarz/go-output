# Bugfix Report: Output Ignores Transformer Priority Ordering

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

`Output` accepts transformers via `WithTransformer` / `WithTransformers`, but
`processFormatData` (`v2/output.go`) applied the copied `[]Transformer` slice in
caller-argument (insertion) order rather than sorting by `Transformer.Priority()`.
This contradicts the transformer pipeline contract — `TransformPipeline` sorts
lower priority first — and the v2 design requirement for priority-based execution.

**Reproduction steps:**
1. Construct `NewOutput(WithFormat(...), WithTransformers(high, low), WithWriter(...))`
   where `high.Priority() == 100` and `low.Priority() == 10`, and both mutate the
   rendered bytes.
2. Call `Render`.
3. Observe that the high-priority transformer runs before the low-priority one —
   render order follows the argument order, not `Priority()`.

**Impact:** Any consumer relying on transformer priority (the documented contract)
gets incorrect, order-dependent output. Severity is correctness: results depend on
the order transformers happen to be registered rather than their declared priority.

## Investigation Summary

- **Symptoms examined:** With two byte-mutating transformers registered in reverse
  priority order, the output reflected argument order (`[high][low]`) instead of
  priority order (`[low][high]`).
- **Code inspected:** `v2/output.go` (`Render`, `renderWithConfig`,
  `processFormatData`) and `v2/transformer.go` (`TransformPipeline`,
  `sortTransformers`, `Transform`).
- **Hypotheses tested:** Confirmed the `Transformer` interface exposes `Priority()`
  (lower = earlier) and that `TransformPipeline.sortTransformers` sorts ascending by
  priority. Confirmed `Output` never routes through `TransformPipeline` and instead
  iterates its own copied slice directly, skipping the sort step.

## Discovered Root Cause

`processFormatData` iterated the transformer slice in the order it was built by
`WithTransformer` / `WithTransformers`, never applying the priority sort that the
pipeline contract requires.

**Defect type:** Logic error — missing ordering step.

**Why it occurred:** `Output` reimplemented the transformer-application loop instead
of reusing `TransformPipeline`. The reimplementation omitted the priority-sort step
that lives in `TransformPipeline.sortTransformers`.

**Five Whys:**
1. Why does render order follow argument order? `processFormatData` iterates the
   transformer slice directly.
2. Why is the slice in argument order? `WithTransformer`/`WithTransformers` append
   in call order and `Render` copies the slice verbatim.
3. Why isn't it sorted? The sort logic only exists in
   `TransformPipeline.sortTransformers`, which `Output` never invokes.
4. Why doesn't `Output` use `TransformPipeline`? `Output` was implemented with its
   own ad-hoc transformer loop that bypasses the pipeline.
5. Root cause: the transformer-application path in `Output` duplicates pipeline
   behaviour but omits the priority-sort the contract requires.

## Resolution for the Issue

**Changes made:**
- `v2/output.go` — `processFormatData` now sorts a local copy of the transformer
  slice by `Priority()` (ascending, stable) before applying them, so execution order
  always honours the declared priority regardless of registration order.

**Approach rationale:** Per the ticket scope constraint, the change is confined to
`v2/output.go`. A local stable sort by `Priority()` is the minimal change: it does
not mutate the caller's slice (a fresh copy is sorted) and uses `sort.SliceStable`
so transformers with equal priority keep their insertion order, matching the
pipeline's behaviour.

**Alternatives considered:**
- **Route through `TransformPipeline`** — Would also fix the bug and reuse existing
  logic, but it is heavier (constructing a pipeline per render) and a parallel ticket
  (T-1131) owns changes to `transformer.go`; routing through it risks coupling and
  merge conflicts. Rejected to keep the change isolated to `output.go`.
- **Sort the shared slice once in `Render`/`renderWithConfig`** — Would work but
  `processFormatData` is the documented locus of the bug and is called per format;
  sorting a local copy there keeps the fix localized and avoids mutating the shared
  copied slice that other code paths may rely on.

## Regression Test

**Test file:** `v2/output_test.go`
**Test name:** `TestOutput_Render_TransformerPriorityOrder`

**What it verifies:** Two byte-mutating transformers are registered in reverse
priority order (`high` before `low`); the test asserts the lower-priority
transformer's marker appears earlier in the output, proving priority order is
honoured regardless of argument order.

**Run command:** `go test -run TestOutput_Render_TransformerPriorityOrder ./v2/...`

## Affected Files

| File | Change |
|------|--------|
| `v2/output.go` | Sort transformers by `Priority()` in `processFormatData` |
| `v2/output_test.go` | Add `TestOutput_Render_TransformerPriorityOrder` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the regression test fails before the fix (output `[high][low]`) and
  passes after (output `[low][high]`).

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer routing through `TransformPipeline` rather than reimplementing the
  transformer-application loop, so the priority contract lives in one place.
- When duplicating behaviour from an existing abstraction, add a test that asserts
  the contract (here, priority ordering) holds.

## Related

- Transit ticket T-1183
- Related ticket T-1131 (owns `transformer.go` nil-handling changes)
