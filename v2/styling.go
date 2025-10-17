package output

import (
	"github.com/fatih/color"
)

// Inline styling functions provide stateless helpers for applying ANSI colors to strings.
// These functions are designed for inline use within table cells and other content,
// maintaining compatibility with v1's StringWarningInline, StringPositiveInline, etc.
//
// Unlike v1, these are package-level functions with no global state dependency.
// They always return styled strings using fatih/color with ForceColor enabled.

func init() {
	// Force colors to always be applied, matching v1 behavior
	// This ensures colors work even when output isn't detected as a TTY
	color.NoColor = false
}

// StyleWarning applies bold red styling to text (for warnings/errors).
// Equivalent to v1's StringWarningInline with colors enabled.
func StyleWarning(text string) string {
	red := color.New(color.FgRed, color.Bold)
	red.EnableColor()
	return red.Sprint(text)
}

// StylePositive applies bold green styling to text (for success/positive states).
// Equivalent to v1's StringPositiveInline with colors enabled.
func StylePositive(text string) string {
	green := color.New(color.FgGreen, color.Bold)
	green.EnableColor()
	return green.Sprint(text)
}

// StyleNegative applies bold red styling to text (for failure/negative states).
// Alias for StyleWarning to match v1 naming patterns.
func StyleNegative(text string) string {
	return StyleWarning(text)
}

// StyleInfo applies blue styling to text (for informational content).
func StyleInfo(text string) string {
	blue := color.New(color.FgBlue)
	blue.EnableColor()
	return blue.Sprint(text)
}

// StyleBold applies bold styling to text without color.
func StyleBold(text string) string {
	bold := color.New(color.Bold)
	bold.EnableColor()
	return bold.Sprint(text)
}

// StyleWarningIf conditionally applies warning styling based on useColor flag.
// When useColor is false, returns the original text unchanged.
// This is useful for respecting terminal capabilities or user preferences.
func StyleWarningIf(text string, useColor bool) string {
	if !useColor {
		return text
	}
	return StyleWarning(text)
}

// StylePositiveIf conditionally applies positive styling based on useColor flag.
// When useColor is false, returns the original text unchanged.
func StylePositiveIf(text string, useColor bool) string {
	if !useColor {
		return text
	}
	return StylePositive(text)
}

// StyleNegativeIf conditionally applies negative styling based on useColor flag.
// When useColor is false, returns the original text unchanged.
func StyleNegativeIf(text string, useColor bool) string {
	if !useColor {
		return text
	}
	return StyleNegative(text)
}

// StyleInfoIf conditionally applies info styling based on useColor flag.
// When useColor is false, returns the original text unchanged.
func StyleInfoIf(text string, useColor bool) string {
	if !useColor {
		return text
	}
	return StyleInfo(text)
}

// StyleBoldIf conditionally applies bold styling based on useBold flag.
// When useBold is false, returns the original text unchanged.
func StyleBoldIf(text string, useBold bool) string {
	if !useBold {
		return text
	}
	return StyleBold(text)
}
