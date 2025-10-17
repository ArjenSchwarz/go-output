package output

import (
	"strings"
	"testing"
)

func TestStyleWarning(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string // We'll check for ANSI codes presence, not exact match
	}{
		"simple text": {
			input: "Error occurred",
			want:  "Error occurred",
		},
		"empty string": {
			input: "",
			want:  "",
		},
		"with newlines": {
			input: "Line 1\nLine 2",
			want:  "Line 1\nLine 2",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := StyleWarning(tc.input)

			// Check that output contains the original text
			if !strings.Contains(got, tc.want) {
				t.Errorf("StyleWarning(%q) should contain %q, got %q", tc.input, tc.want, got)
			}

			// Check that ANSI codes are present (unless input is empty)
			if tc.input != "" && !strings.Contains(got, "\x1b[") {
				t.Errorf("StyleWarning(%q) should contain ANSI codes, got %q", tc.input, got)
			}
		})
	}
}

func TestStylePositive(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"success message": {
			input: "Success",
			want:  "Success",
		},
		"empty string": {
			input: "",
			want:  "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := StylePositive(tc.input)

			if !strings.Contains(got, tc.want) {
				t.Errorf("StylePositive(%q) should contain %q, got %q", tc.input, tc.want, got)
			}

			if tc.input != "" && !strings.Contains(got, "\x1b[") {
				t.Errorf("StylePositive(%q) should contain ANSI codes, got %q", tc.input, got)
			}
		})
	}
}

func TestStyleNegative(t *testing.T) {
	input := "Failed"
	got := StyleNegative(input)
	expected := StyleWarning(input)

	// StyleNegative should be an alias for StyleWarning
	if got != expected {
		t.Errorf("StyleNegative should be alias for StyleWarning, got different outputs")
	}
}

func TestStyleInfo(t *testing.T) {
	input := "Information"
	got := StyleInfo(input)

	if !strings.Contains(got, input) {
		t.Errorf("StyleInfo(%q) should contain %q, got %q", input, input, got)
	}

	if !strings.Contains(got, "\x1b[") {
		t.Errorf("StyleInfo(%q) should contain ANSI codes, got %q", input, got)
	}
}

func TestStyleBold(t *testing.T) {
	input := "Bold text"
	got := StyleBold(input)

	if !strings.Contains(got, input) {
		t.Errorf("StyleBold(%q) should contain %q, got %q", input, input, got)
	}

	if !strings.Contains(got, "\x1b[") {
		t.Errorf("StyleBold(%q) should contain ANSI codes, got %q", input, got)
	}
}

func TestStyleWarningIf(t *testing.T) {
	tests := map[string]struct {
		input    string
		useColor bool
		wantANSI bool
	}{
		"with color enabled": {
			input:    "Warning",
			useColor: true,
			wantANSI: true,
		},
		"with color disabled": {
			input:    "Warning",
			useColor: false,
			wantANSI: false,
		},
		"empty with color disabled": {
			input:    "",
			useColor: false,
			wantANSI: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := StyleWarningIf(tc.input, tc.useColor)

			hasANSI := strings.Contains(got, "\x1b[")

			if tc.wantANSI && !hasANSI {
				t.Errorf("StyleWarningIf(%q, %v) should contain ANSI codes, got %q", tc.input, tc.useColor, got)
			}

			if !tc.wantANSI && hasANSI {
				t.Errorf("StyleWarningIf(%q, %v) should not contain ANSI codes, got %q", tc.input, tc.useColor, got)
			}

			if !tc.useColor && got != tc.input {
				t.Errorf("StyleWarningIf(%q, false) should return original text, got %q", tc.input, got)
			}
		})
	}
}

func TestStylePositiveIf(t *testing.T) {
	input := "Success"

	// With color
	gotWithColor := StylePositiveIf(input, true)
	if !strings.Contains(gotWithColor, "\x1b[") {
		t.Errorf("StylePositiveIf with color=true should contain ANSI codes")
	}

	// Without color
	gotWithoutColor := StylePositiveIf(input, false)
	if gotWithoutColor != input {
		t.Errorf("StylePositiveIf with color=false should return original, got %q", gotWithoutColor)
	}
}

func TestStyleNegativeIf(t *testing.T) {
	input := "Error"

	gotWithColor := StyleNegativeIf(input, true)
	expectedWithColor := StyleWarningIf(input, true)

	if gotWithColor != expectedWithColor {
		t.Errorf("StyleNegativeIf should match StyleWarningIf behavior")
	}

	gotWithoutColor := StyleNegativeIf(input, false)
	if gotWithoutColor != input {
		t.Errorf("StyleNegativeIf with color=false should return original, got %q", gotWithoutColor)
	}
}

func TestStyleInfoIf(t *testing.T) {
	input := "Info"

	gotWithColor := StyleInfoIf(input, true)
	if !strings.Contains(gotWithColor, "\x1b[") {
		t.Errorf("StyleInfoIf with color=true should contain ANSI codes")
	}

	gotWithoutColor := StyleInfoIf(input, false)
	if gotWithoutColor != input {
		t.Errorf("StyleInfoIf with color=false should return original, got %q", gotWithoutColor)
	}
}

func TestStyleBoldIf(t *testing.T) {
	input := "Bold"

	gotWithBold := StyleBoldIf(input, true)
	if !strings.Contains(gotWithBold, "\x1b[") {
		t.Errorf("StyleBoldIf with useBold=true should contain ANSI codes")
	}

	gotWithoutBold := StyleBoldIf(input, false)
	if gotWithoutBold != input {
		t.Errorf("StyleBoldIf with useBold=false should return original, got %q", gotWithoutBold)
	}
}

// TestInlineStyleUsage demonstrates how these functions would be used in practice
func TestInlineStyleUsage(t *testing.T) {
	// Simulating fog's use case: inline styling within table data
	changetype := "DELETED"
	styledChangetype := StyleWarning(changetype)

	if !strings.Contains(styledChangetype, changetype) {
		t.Errorf("Styled changetype should contain original text")
	}

	// Building table data with inline styling
	tableData := []map[string]any{
		{
			"Resource":   "aws_instance.example",
			"ChangeType": StyleWarning("DELETED"),
			"Status":     StylePositive("SUCCESS"),
		},
	}

	if len(tableData) != 1 {
		t.Errorf("Expected 1 row, got %d", len(tableData))
	}

	row := tableData[0]
	if !strings.Contains(row["ChangeType"].(string), "\x1b[") {
		t.Errorf("ChangeType should be styled with ANSI codes")
	}
}
