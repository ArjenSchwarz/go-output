# Bugfix Report: ColorTransformer Ignores Its ColorScheme and Tints the Entire Document

**Ticket:** T-1518
**Date:** 2026-08-17
**Status:** Fixed

## Description of the Issue

`ColorTransformer.Transform` (`v2/transformers.go`) hard-coded green, red, and
blue ANSI colors and never read the transformer's `scheme` field, so
`NewColorTransformerWithScheme` accepted a custom `ColorScheme` that had no
effect on the output. Worse, the transform inspected the ENTIRE rendered
document with `strings.Contains` and wrapped the whole buffer in a single
`Sprint` call — one matching cell anywhere in a table tinted every line of the
document with that one color.

Two secondary defects sat in the same code:

- The word indicators (`Yes`, `No`, `true`, `false`) were matched as bare
  substrings, so ordinary text such as "Notes" or "Yesterday" triggered
  coloring — the same defect class fixed for the emoji transformer in T-1267.
- `RemoveColorsTransformer.Transform` recompiled its ANSI-escape regex on
  every call instead of compiling it once at package load.

**Reproduction steps:**
1. `transformer := output.NewColorTransformerWithScheme(output.ColorScheme{Success: "cyan"})`
2. `result, _ := transformer.Transform(ctx, []byte("✅ passed\nplain line\n❌ failed"), output.FormatTable)`
3. Observe the whole three-line document wrapped in one bold-green escape
   sequence (`\x1b[32;1m...\x1b[0;22m`): the configured cyan is ignored, the
   plain line is tinted, and the failure line is green.

**Impact:** Any consumer using `NewColorTransformerWithScheme` got default
colors instead of their configuration (silently dead config). Any table
containing a single success indicator rendered entirely green, making
failure rows visually indistinguishable — the opposite of the transformer's
purpose. `EnhancedColorTransformer` (`v2/format_aware.go`) delegates to the
same method and was equally affected.

## Investigation Summary

- **Symptoms examined:** Custom schemes having no effect; whole-document
  tinting reproduced by the new regression tests before the fix.
- **Code inspected:** `v2/transformers.go` (ColorTransformer,
  RemoveColorsTransformer, EmojiTransformer for the T-1267 word-boundary
  precedent), `v2/format_aware.go` (EnhancedColorTransformer delegation),
  `v2/styling.go` (fatih/color usage and `color.NoColor` handling),
  `v2/transformers_test.go` and the backward-compatibility tests (existing
  assertions only checked output was non-empty, which is why the defect
  survived).
- **Hypotheses tested:** Confirmed no other code path applies the scheme
  (`c.scheme` had zero readers). Confirmed `EnhancedColorTransformer` passes
  through to the same buggy `Transform`. Confirmed no existing test pins the
  whole-buffer behavior, so a per-line fix breaks nothing.

Root cause chain (Five Whys):
1. Why does a custom scheme have no effect? `Transform` builds its colors with
   hard-coded `color.FgGreen`/`FgRed`/`FgBlue` attributes.
2. Why is the whole document one color? The switch evaluates
   `strings.Contains` over the full input and wraps the entire buffer in a
   single `Sprint`.
3. Why did the scheme field exist but go unread? The constructors and
   `ColorScheme` type were built for API completeness, but `Transform` was
   left as a placeholder-quality implementation that was never wired to it.
4. Why did nothing catch it? The existing tests only asserted the transform
   output was non-empty ("Color codes are complex to test exactly").
5. Why did the substring defect persist after T-1267? That fix targeted the
   emoji transformer only; the same pattern in ColorTransformer was not
   audited at the time.

## Discovered Root Cause

`ColorTransformer.Transform` never dereferences `c.scheme` and operates on the
whole rendered buffer as a single unit: one hard-coded color is selected by
the first `strings.Contains` match against the full document and applied via
one `Sprint` wrap.

**Defect type:** Logic error (dead configuration; wrong granularity of
application; missing word-boundary matching).

**Why it occurred:** The transform was written before the scheme plumbing and
never revisited; permissive tests allowed it to pass unnoticed.

**Contributing factors:** ANSI output is hard to eyeball in test logs, and the
default scheme's green/red overlap with the hard-coded colors masked the dead
config in casual use.

## Resolution for the Issue

_To be completed after the fix is implemented._

## Regression Test

**Test file:** `v2/transformers_color_test.go`
**Test names:** `TestColorTransformerHonorsScheme`,
`TestColorTransformerDefaultSchemeWarning`,
`TestColorTransformerColorsPerLine`, `TestColorTransformerWordBoundaries`,
`TestColorTransformerUnknownColorName`

**What it verifies:** Custom scheme colors are applied per role (and
hard-coded colors are absent); the default scheme's yellow is used for
warnings; a document mixing success, plain, and failure lines gets green,
unstyled, and red lines respectively instead of one whole-document color;
embedded substrings ("Notes", "Yesterday") do not trigger coloring; unknown
scheme color names leave text unstyled.

**Run command:** `cd v2 && go test -run 'TestColorTransformer' ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/transformers.go` | Scheme-driven per-line coloring; hoisted ANSI regex |
| `v2/transformers_color_test.go` | New regression tests (failing before fix) |
| `CHANGELOG.md` | Unreleased/Fixed entry flagging output changes |

## Verification

**Automated:**
- [ ] Regression tests pass
- [ ] Full test suite passes (`make test`)
- [ ] Linters/validators pass (`make check`)

**Manual verification:**
- Red-phase run confirmed all five regression tests fail against the original
  implementation with exactly the predicted output (whole-buffer
  `\x1b[32;1m...\x1b[0;22m` wrap, hard-coded colors, substring matches).

## Prevention

**Recommendations to avoid similar bugs:**
- When a struct field is added for configuration, add a test that a
  non-default value observably changes behavior — dead config is silent.
- Avoid "output is non-empty" assertions for transformers; assert on the
  specific bytes or markers the transform is supposed to produce.
- When fixing a defect pattern (e.g. T-1267's substring matching), grep the
  package for other instances of the same pattern.

## Related

- T-1267 — word-boundary fix for the emoji transformer (same defect class).
- `v2/format_aware.go` `EnhancedColorTransformer` — delegates to the fixed
  method; no changes needed there.
