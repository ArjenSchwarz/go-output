package output

import (
	"context"
	"strings"
	"testing"
)

// Regression tests for T-1518: ColorTransformer ignored its ColorScheme (so
// NewColorTransformerWithScheme had no effect) and wrapped the ENTIRE rendered
// document in a single ANSI color based on the first matching indicator found
// anywhere in the output.

// ANSI foreground escape prefixes used to detect which color was applied.
const (
	ansiEscape  = "\x1b["
	ansiRed     = "\x1b[31"
	ansiGreen   = "\x1b[32"
	ansiYellow  = "\x1b[33"
	ansiBlue    = "\x1b[34"
	ansiMagenta = "\x1b[35"
	ansiCyan    = "\x1b[36"
)

// TestColorTransformerHonorsScheme verifies that a custom ColorScheme passed
// to NewColorTransformerWithScheme controls the colors that are applied.
// Before the T-1518 fix, Transform hard-coded green/red/blue and never read
// the configured scheme.
func TestColorTransformerHonorsScheme(t *testing.T) {
	transformer := NewColorTransformerWithScheme(ColorScheme{
		Success: "cyan",
		Warning: "magenta",
		Error:   "yellow",
		Info:    "green",
	})
	ctx := context.Background()

	tests := map[string]struct {
		input       string
		wantColor   string
		avoidColors []string
	}{
		"success uses scheme success color": {
			input:       "✅ done",
			wantColor:   ansiCyan,
			avoidColors: []string{ansiGreen},
		},
		"error uses scheme error color": {
			input:       "❌ failed",
			wantColor:   ansiYellow,
			avoidColors: []string{ansiRed},
		},
		"warning uses scheme warning color": {
			input:       "🚨 alert",
			wantColor:   ansiMagenta,
			avoidColors: []string{ansiRed},
		},
		"info uses scheme info color": {
			input:       "ℹ️ note",
			wantColor:   ansiGreen,
			avoidColors: []string{ansiBlue},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := transformer.Transform(ctx, []byte(test.input), FormatTable)
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}
			got := string(result)
			if !strings.Contains(got, test.wantColor) {
				t.Errorf("Transform(%q) = %q, want it to contain scheme color code %q", test.input, got, test.wantColor)
			}
			for _, avoid := range test.avoidColors {
				if strings.Contains(got, avoid) {
					t.Errorf("Transform(%q) = %q, contains hard-coded color code %q instead of scheme color", test.input, got, avoid)
				}
			}
		})
	}
}

// TestColorTransformerDefaultSchemeWarning verifies that the default scheme's
// Warning color (yellow) is applied to warning indicators. Before the T-1518
// fix, warnings were hard-coded to red regardless of the scheme.
func TestColorTransformerDefaultSchemeWarning(t *testing.T) {
	transformer := NewColorTransformer()

	result, err := transformer.Transform(context.Background(), []byte("🚨 alert"), FormatTable)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	got := string(result)
	if !strings.Contains(got, ansiYellow) {
		t.Errorf("Transform(warning) = %q, want default scheme warning color %q", got, ansiYellow)
	}
	if strings.Contains(got, ansiRed) {
		t.Errorf("Transform(warning) = %q, contains hard-coded red instead of scheme warning color", got)
	}
}

// TestColorTransformerColorsPerLine verifies that coloring is applied per
// line rather than wrapping the whole document in one color. Before the
// T-1518 fix, a single ✅ anywhere in the output tinted the entire document
// green, including lines with ❌ failures.
func TestColorTransformerColorsPerLine(t *testing.T) {
	transformer := NewColorTransformer()
	input := "✅ passed\nplain middle line\n❌ failed"

	result, err := transformer.Transform(context.Background(), []byte(input), FormatTable)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	lines := strings.Split(string(result), "\n")
	if len(lines) != 3 {
		t.Fatalf("Transform() produced %d lines, want 3: %q", len(lines), string(result))
	}

	if !strings.Contains(lines[0], ansiGreen) {
		t.Errorf("success line = %q, want it colored green (%q)", lines[0], ansiGreen)
	}
	if lines[1] != "plain middle line" {
		t.Errorf("line without indicators = %q, want it unchanged", lines[1])
	}
	if !strings.Contains(lines[2], ansiRed) {
		t.Errorf("error line = %q, want it colored red (%q); whole-document tinting would leave it in the success color", lines[2], ansiRed)
	}
}

// TestColorTransformerWordBoundaries verifies that word indicators such as
// "No" and "Yes" only match as standalone words. Before the T-1518 fix,
// substring matching meant ordinary text like "Notes" tinted the whole
// document red (same defect class as T-1267 in the emoji transformer).
func TestColorTransformerWordBoundaries(t *testing.T) {
	transformer := NewColorTransformer()

	tests := map[string]string{
		"No embedded in word":  "Notes from Nobody",
		"Yes embedded in word": "Yesterday's Eyestrain",
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := transformer.Transform(context.Background(), []byte(input), FormatTable)
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}
			if strings.Contains(string(result), ansiEscape) {
				t.Errorf("Transform(%q) = %q, want no coloring for embedded substrings", input, string(result))
			}
		})
	}
}

// TestColorTransformerUnknownColorName verifies that a scheme entry naming an
// unsupported color leaves matching text unstyled instead of silently falling
// back to a hard-coded color.
func TestColorTransformerUnknownColorName(t *testing.T) {
	transformer := NewColorTransformerWithScheme(ColorScheme{
		Success: "sparkly",
	})

	result, err := transformer.Transform(context.Background(), []byte("✅ done"), FormatTable)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if strings.Contains(string(result), ansiEscape) {
		t.Errorf("Transform() = %q, want unknown scheme color to leave text unstyled", string(result))
	}
}
