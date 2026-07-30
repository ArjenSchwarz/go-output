# Bugfix Report: EnhancedEmojiTransformer Corrupts Embedded Indicator Words

**Date:** 2026-07-12
**Status:** Fixed
**Ticket:** T-1509

## Description of the Issue

`EnhancedEmojiTransformer.Transform` in `v2/format_aware.go` has format-specific
branches for Markdown and HTML that replace the word indicators `OK`, `Yes`, and
`No` using raw `strings.ReplaceAll`. This performs unanchored substring
replacement, so ordinary words that merely contain an indicator are corrupted.
The base `EmojiTransformer` was fixed in T-1267 to use word-boundary matching,
but the enhanced transformer's format-specific branches were not updated and
diverged from the fixed base behaviour.

**Reproduction steps:**
1. Create an `EnhancedEmojiTransformer` via `NewEnhancedEmojiTransformer()`.
2. Transform text containing words that embed an indicator, e.g. "Notes",
   "Nobody", "Yesterday", or "BROKEN", with format `html` or `markdown`.
3. Observe the corruption: in HTML, "Notes" -> "&#x274C;tes", "Nobody" ->
   "&#x274C;body", "Yesterday" -> "&#x2705;terday", "BROKEN" -> "BR&#x2705;EN";
   in Markdown, "BROKEN" -> "BR✅EN".

**Impact:** Medium severity, low exposure. Any text passed through the enhanced
transformer for HTML or Markdown output is silently corrupted when it contains
embedded indicator substrings, compromising rendered data integrity. Exposure is
limited because the "Enhanced" layer in `format_aware.go` currently has no
production callers inside the library (tests and migration examples only), but
it is exported API that external consumers can use directly.

## Investigation Summary

Structured root cause analysis performed against the T-1509 ticket description.

- **Symptoms examined:** Words embedding `OK`/`Yes`/`No` partially replaced by
  emoji (Markdown branch) or HTML entities (HTML branch); reproduced exactly as
  described in the ticket via regression tests.
- **Code inspected:** `v2/format_aware.go` (`EnhancedEmojiTransformer.Transform`,
  format-specific branches), `v2/transformers.go` (base
  `EmojiTransformer.Transform` and the boundary-aware
  `emojiIndicatorReplacements` table from T-1267),
  `v2/format_aware_test.go` (existing expectations for the enhanced
  transformer).
- **Hypotheses tested:** Confirmed the default branch (table/CSV) already
  delegates to the fixed base transformer and is unaffected. Only the
  Markdown and HTML branches use unanchored `strings.ReplaceAll` for word
  indicators. The `!!` indicator is punctuation, so word boundaries do not
  apply to it and plain substring replacement is correct there.

## Discovered Root Cause

The indicator-replacement logic was duplicated in
`EnhancedEmojiTransformer.Transform` rather than shared with the base
transformer. When T-1267 fixed the base `EmojiTransformer` to use word-boundary
regexes, the duplicated format-specific branches in `format_aware.go` kept the
old unanchored `strings.ReplaceAll` calls and silently diverged.

**Defect type:** Logic error (unanchored substring replacement) caused by
duplicated logic diverging from its fixed counterpart.

**Why it occurred:** The enhanced transformer implements format-specific output
(HTML entities, a deliberately conservative Markdown set) inline instead of
delegating the matching semantics to a shared definition, so the T-1267 fix did
not reach it.

**Contributing factors:** The "Enhanced" layer has no production callers inside
the library, so the divergence was never observed in rendered output and no
existing test covered embedded indicator words for this type.

## Resolution for the Issue

**Changes made:**
- `v2/format_aware.go` - Replaced the unanchored `strings.ReplaceAll` calls for
  word indicators in the Markdown and HTML branches with word-boundary regex
  replacements, defined in a package-level `enhancedEmojiReplacements` table
  (compiled once at package load, mirroring the base transformer's
  `emojiIndicatorReplacements`). Each format keeps its deliberate replacement
  set: Markdown converts only standalone `OK`; HTML converts standalone `OK`,
  `Yes`, and `No` to HTML entities. The `!!` indicator remains a plain
  substring replacement because word boundaries do not apply to punctuation.
  The `case "markdown"` literal was also changed to the `FormatMarkdown`
  constant for consistency.

**Approach rationale:** Matches the standalone-indicator semantics established
by T-1267 for the base transformer while preserving the enhanced transformer's
format-specific behaviour (conservative Markdown set, HTML entity encoding).
Minimal, isolated change in a single file.

**Alternatives considered:**
- Delegate to the base `EmojiTransformer.Transform` and post-process emoji into
  HTML entities - Rejected: it would silently change the Markdown branch's
  behaviour (the base transformer also converts `Yes`/`No`/`true`/`false`,
  while the Markdown branch is deliberately conservative) and would need an
  emoji-to-entity mapping pass for HTML, adding complexity without benefit.
- Delete the "Enhanced" layer since it has no production callers - Rejected:
  removing exported types is a breaking change reserved for a future major
  version (v3 decision), out of scope for this bugfix.

## Regression Test

**Test file:** `v2/format_aware_test.go`
**Test name:** `TestEnhancedEmojiTransformer_PreservesEmbeddedIndicatorWords`

**What it verifies:** Words embedding indicators ("Notes", "Nobody",
"Yesterday", "BROKEN", "LOOKUP") pass through unchanged in both the Markdown
and HTML branches, while standalone indicators are still converted (`OK` ->
emoji in Markdown; `OK`/`Yes`/`No` -> HTML entities in HTML).

**Run command:** `cd v2 && go test -run TestEnhancedEmojiTransformer_PreservesEmbeddedIndicatorWords ./...`

## Affected Files

| File | Change |
|------|--------|
| `v2/format_aware.go` | Word-boundary regex replacement for word indicators in Markdown/HTML branches |
| `v2/format_aware_test.go` | Added regression test for embedded indicator words |
| `CHANGELOG.md` | Added Unreleased/Fixed entry; removed a stray committed `<<<<<<< HEAD` merge marker adjacent to it |

## Verification

**Automated:**
- [x] Regression test passes (all 8 subtests)
- [x] Full test suite passes (unit and `INTEGRATION=1` runs)
- [x] Linters/validators pass (`golangci-lint run`: 0 issues; `gofmt`: clean)

**Manual verification:**
- Confirmed the regression tests fail before the fix with exactly the
  corruption described in the ticket ("Notes" -> "&#x274C;tes",
  "BROKEN" -> "BR✅EN").

## Prevention

**Recommendations to avoid similar bugs:**
- When fixing a defect in shared logic, grep for duplicated copies of the same
  pattern (`strings.ReplaceAll` on the same indicator strings) across the
  package and fix or consolidate them in the same change.
- Prefer a single package-level replacement table consumed by all transformer
  variants over per-branch inline replacement calls.
- The `format_aware.go` "Enhanced" layer has zero production callers; consider
  removing it in v3 to eliminate the duplication entirely.

## Related

- T-1267: base `EmojiTransformer` word-boundary fix
  (`specs/bugfixes/emoji-transformer-corrupts-words/report.md`)
- T-1514: duplicate ticket, abandoned in favour of T-1509
