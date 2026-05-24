# Bugfix Report: Emoji Transformer Corrupts Words Containing Indicators

**Date:** 2026-05-25
**Status:** Fixed

## Description of the Issue

`EmojiTransformer.Transform` converts text-based status indicators (`OK`, `Yes`,
`No`, `!!`) into emoji. It runs on table, markdown, HTML, and CSV output, so it
processes ordinary cell text in addition to standalone status values. The string
indicators were applied with `strings.ReplaceAll`, which performs unanchored
substring replacement. Any cell text that merely contained one of the indicators
as a substring was corrupted.

**Reproduction steps:**
1. Render a table/markdown/HTML/CSV document containing the cell value "Notes"
   (or "Nobody", "Nope", "Yesterday", etc.).
2. Run the output through the emoji transformer.
3. Observe the corrupted output: "Notes" -> "❌tes", "Nobody" -> "❌body",
   "Yesterday" -> "✅terday".

**Impact:** Medium. Any consumer using the emoji transformer on tabular output
silently corrupts cell text containing the substrings "No" or "Yes" (and "OK" in
upper-case forms). Data integrity in rendered output is compromised.

## Investigation Summary

- **Symptoms examined:** Cell text containing "No"/"Yes"/"OK" substrings being
  partially replaced by emoji.
- **Code inspected:** `v2/transformers.go`, `EmojiTransformer.Transform`.
- **Hypotheses tested:** Confirmed the boolean replacements (`true`/`false`)
  already use word-boundary regexes (`\btrue\b`, `\bfalse\b`) and are unaffected,
  while the string indicators use unanchored `strings.ReplaceAll`. This
  inconsistency is the defect.

## Discovered Root Cause

The string indicator replacements used `strings.ReplaceAll`, which replaces every
substring occurrence regardless of word boundaries. The boolean replacements in
the same function already used word-boundary regexes, but the string indicators
did not follow the same pattern.

**Defect type:** Logic error — unanchored substring replacement where
word-boundary matching was required.

**Why it occurred:** The replacements were stored in a map and applied with
`strings.ReplaceAll` in a loop. The map ordering is also non-deterministic, which
is unsuitable for ordered/anchored replacement.

**Contributing factors:** Inconsistent replacement strategy within a single
function (substring vs word-boundary).

## Resolution for the Issue

**Changes made:**
- `v2/transformers.go` — `EmojiTransformer.Transform` now converts the word-based
  string indicators (`OK`, `Yes`, `No`) using word-boundary regexes
  (`\bOK\b`, `\bYes\b`, `\bNo\b`), matching the existing approach for `true`/`false`.
  The non-word `!!` indicator continues to use `strings.ReplaceAll` because `\b`
  word boundaries do not apply to punctuation. Replacements are applied in a fixed
  order rather than via non-deterministic map iteration.

**Approach rationale:** Word-boundary regexes are the same mechanism already used
for the boolean indicators in this function, keeping the implementation
consistent and minimal. Only standalone indicators (bounded by non-word
characters or string edges) are converted; indicators embedded in larger words
are left untouched.

**Alternatives considered:**
- Full cell-aware parsing of the tabular structure — Rejected as significantly
  more complex and risky for what is fundamentally a token-matching problem.
  Word boundaries solve the reported corruption directly.

## Regression Test

**Test file:** `v2/transformers_test.go`
**Test name:** `TestEmojiTransformer_Transform_WordBoundaries`

**What it verifies:** Words that merely contain an indicator as a substring
("Notes", "Nobody", "Nope", "Yesterday", "ABCNoDEF") are left untouched, while
standalone indicators ("No", "Yes", "OK", and indicators surrounded by
punctuation/whitespace) are still converted to emoji.

**Run command:** `go test ./v2/ -run TestEmojiTransformer_Transform_WordBoundaries`
(from the v2 module directory) or `make test` from the repo root.

## Affected Files

| File | Change |
|------|--------|
| `v2/transformers.go` | Use word-boundary regexes for string indicators |
| `v2/transformers_test.go` | Add `TestEmojiTransformer_Transform_WordBoundaries` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the failing tests reproduce the corruption before the fix (red), then
  pass after the fix (green).

## Prevention

**Recommendations to avoid similar bugs:**
- Use word-boundary matching consistently for word-based token replacement.
- Avoid `strings.ReplaceAll` for replacing whole words/tokens in free text.
- Prefer deterministic, ordered replacement over map iteration when order matters.

## Related

- Transit ticket T-1267
