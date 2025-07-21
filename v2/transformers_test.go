package output

import (
	"context"
	"strings"
	"testing"
)

// Test EmojiTransformer

func TestEmojiTransformer_Name(t *testing.T) {
	transformer := &EmojiTransformer{}
	if transformer.Name() != "emoji" {
		t.Errorf("EmojiTransformer.Name() = %s, want emoji", transformer.Name())
	}
}

func TestEmojiTransformer_Priority(t *testing.T) {
	transformer := &EmojiTransformer{}
	if transformer.Priority() != 100 {
		t.Errorf("EmojiTransformer.Priority() = %d, want 100", transformer.Priority())
	}
}

func TestEmojiTransformer_CanTransform(t *testing.T) {
	transformer := &EmojiTransformer{}

	tests := []struct {
		format   string
		expected bool
	}{
		{"table", true},
		{"markdown", true},
		{"html", true},
		{"csv", true},
		{"json", false},
		{"yaml", false},
		{"dot", false},
	}

	for _, test := range tests {
		result := transformer.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("EmojiTransformer.CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestEmojiTransformer_Transform(t *testing.T) {
	transformer := &EmojiTransformer{}
	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "warning indicators",
			input:    "!! Critical error !!",
			expected: "🚨 Critical error 🚨",
		},
		{
			name:     "success indicators",
			input:    "Status: OK",
			expected: "Status: ✅",
		},
		{
			name:     "yes/no indicators",
			input:    "Active: Yes, Enabled: No",
			expected: "Active: ✅, Enabled: ❌",
		},
		{
			name:     "boolean values",
			input:    "Running: true, Stopped: false",
			expected: "Running: ✅, Stopped: ❌",
		},
		{
			name:     "mixed content",
			input:    "!! Error: task failed\nResult: OK\nEnabled: true",
			expected: "🚨 Error: task failed\nResult: ✅\nEnabled: ✅",
		},
		{
			name:     "no changes needed",
			input:    "This is normal text with no indicators",
			expected: "This is normal text with no indicators",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := transformer.Transform(ctx, []byte(test.input), "table")
			if err != nil {
				t.Fatalf("EmojiTransformer.Transform() error = %v", err)
			}

			if string(result) != test.expected {
				t.Errorf("EmojiTransformer.Transform() = %s, want %s", string(result), test.expected)
			}
		})
	}
}

// Test ColorTransformer

func TestColorTransformer_Name(t *testing.T) {
	transformer := NewColorTransformer()
	if transformer.Name() != "color" {
		t.Errorf("ColorTransformer.Name() = %s, want color", transformer.Name())
	}
}

func TestColorTransformer_Priority(t *testing.T) {
	transformer := NewColorTransformer()
	if transformer.Priority() != 200 {
		t.Errorf("ColorTransformer.Priority() = %d, want 200", transformer.Priority())
	}
}

func TestColorTransformer_CanTransform(t *testing.T) {
	transformer := NewColorTransformer()

	tests := []struct {
		format   string
		expected bool
	}{
		{"table", true},
		{"markdown", false},
		{"html", false},
		{"csv", false},
		{"json", false},
		{"yaml", false},
	}

	for _, test := range tests {
		result := transformer.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("ColorTransformer.CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestColorTransformer_Transform(t *testing.T) {
	transformer := NewColorTransformer()
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "success content",
			input: "Status: ✅",
		},
		{
			name:  "error content",
			input: "Status: ❌",
		},
		{
			name:  "warning content",
			input: "Alert: 🚨",
		},
		{
			name:  "info content",
			input: "Info: ℹ️",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := transformer.Transform(ctx, []byte(test.input), "table")
			if err != nil {
				t.Fatalf("ColorTransformer.Transform() error = %v", err)
			}

			// Just verify that the result is not empty and has been processed
			// (Color codes are complex to test exactly)
			if len(result) == 0 {
				t.Error("ColorTransformer.Transform() returned empty result")
			}
		})
	}
}

func TestColorScheme(t *testing.T) {
	scheme := DefaultColorScheme()

	if scheme.Success != "green" {
		t.Errorf("DefaultColorScheme().Success = %s, want green", scheme.Success)
	}

	customScheme := ColorScheme{
		Success: "blue",
		Warning: "purple",
		Error:   "red",
		Info:    "cyan",
	}

	transformer := NewColorTransformerWithScheme(customScheme)
	if transformer.scheme.Success != "blue" {
		t.Errorf("Custom color scheme not applied correctly")
	}
}

// Test SortTransformer

func TestSortTransformer_Name(t *testing.T) {
	transformer := NewSortTransformer("name", true)
	if transformer.Name() != "sort" {
		t.Errorf("SortTransformer.Name() = %s, want sort", transformer.Name())
	}
}

func TestSortTransformer_Priority(t *testing.T) {
	transformer := NewSortTransformer("name", true)
	if transformer.Priority() != 50 {
		t.Errorf("SortTransformer.Priority() = %d, want 50", transformer.Priority())
	}
}

func TestSortTransformer_CanTransform(t *testing.T) {
	transformer := NewSortTransformer("name", true)

	tests := []struct {
		format   string
		expected bool
	}{
		{"table", true},
		{"csv", true},
		{"html", true},
		{"markdown", true},
		{"json", false},
		{"yaml", false},
		{"dot", false},
	}

	for _, test := range tests {
		result := transformer.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("SortTransformer.CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestSortTransformer_Transform(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		transformer *SortTransformer
		input       string
		expected    string
	}{
		{
			name:        "tab-separated ascending sort",
			transformer: NewSortTransformerAscending("Name"),
			input:       "Name\tAge\tCity\nCharlie\t30\tChicago\nAlice\t25\tNew York\nBob\t35\tLos Angeles",
			expected:    "Name\tAge\tCity\nAlice\t25\tNew York\nBob\t35\tLos Angeles\nCharlie\t30\tChicago",
		},
		{
			name:        "csv format ascending sort",
			transformer: NewSortTransformerAscending("Age"),
			input:       "Name,Age,City\nCharlie,30,Chicago\nAlice,25,New York\nBob,35,Los Angeles",
			expected:    "Name,Age,City\nAlice,25,New York\nCharlie,30,Chicago\nBob,35,Los Angeles",
		},
		{
			name:        "descending sort",
			transformer: NewSortTransformer("Age", false),
			input:       "Name\tAge\tCity\nCharlie\t30\tChicago\nAlice\t25\tNew York\nBob\t35\tLos Angeles",
			expected:    "Name\tAge\tCity\nBob\t35\tLos Angeles\nCharlie\t30\tChicago\nAlice\t25\tNew York",
		},
		{
			name:        "string sort",
			transformer: NewSortTransformerAscending("City"),
			input:       "Name\tAge\tCity\nCharlie\t30\tChicago\nAlice\t25\tNew York\nBob\t35\tLos Angeles",
			expected:    "Name\tAge\tCity\nCharlie\t30\tChicago\nBob\t35\tLos Angeles\nAlice\t25\tNew York",
		},
		{
			name:        "no sort key",
			transformer: NewSortTransformer("", true),
			input:       "Name\tAge\tCity\nCharlie\t30\tChicago\nAlice\t25\tNew York\nBob\t35\tLos Angeles",
			expected:    "Name\tAge\tCity\nCharlie\t30\tChicago\nAlice\t25\tNew York\nBob\t35\tLos Angeles",
		},
		{
			name:        "sort key not found",
			transformer: NewSortTransformerAscending("NonExistent"),
			input:       "Name\tAge\tCity\nCharlie\t30\tChicago\nAlice\t25\tNew York\nBob\t35\tLos Angeles",
			expected:    "Name\tAge\tCity\nCharlie\t30\tChicago\nAlice\t25\tNew York\nBob\t35\tLos Angeles",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.transformer.Transform(ctx, []byte(test.input), "table")
			if err != nil {
				t.Fatalf("SortTransformer.Transform() error = %v", err)
			}

			if string(result) != test.expected {
				t.Errorf("SortTransformer.Transform() = %s, want %s", string(result), test.expected)
			}
		})
	}
}

// Test LineSplitTransformer

func TestLineSplitTransformer_Name(t *testing.T) {
	transformer := NewLineSplitTransformerDefault()
	if transformer.Name() != "linesplit" {
		t.Errorf("LineSplitTransformer.Name() = %s, want linesplit", transformer.Name())
	}
}

func TestLineSplitTransformer_Priority(t *testing.T) {
	transformer := NewLineSplitTransformerDefault()
	if transformer.Priority() != 150 {
		t.Errorf("LineSplitTransformer.Priority() = %d, want 150", transformer.Priority())
	}
}

func TestLineSplitTransformer_CanTransform(t *testing.T) {
	transformer := NewLineSplitTransformerDefault()

	tests := []struct {
		format   string
		expected bool
	}{
		{"table", true},
		{"csv", true},
		{"html", true},
		{"markdown", true},
		{"json", false},
		{"yaml", false},
		{"dot", false},
	}

	for _, test := range tests {
		result := transformer.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("LineSplitTransformer.CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestLineSplitTransformer_Transform(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		transformer *LineSplitTransformer
		input       string
		expected    string
	}{
		{
			name:        "pipe split tab-separated",
			transformer: NewLineSplitTransformer("|"),
			input:       "Name\tTags\tStatus\nAlice\ttag1|tag2\tActive\nBob\tsingle\tInactive",
			expected:    "Name\tTags\tStatus\nAlice\ttag1\tActive\n\ttag2\t\nBob\tsingle\tInactive",
		},
		{
			name:        "comma split within CSV field",
			transformer: NewLineSplitTransformer(";"),
			input:       "Name,Skills,Level\nAlice,Java;Go,Expert\nBob,Python,Beginner",
			expected:    "Name,Skills,Level\nAlice,Java,Expert\n,Go,\nBob,Python,Beginner",
		},
		{
			name:        "semicolon split",
			transformer: NewLineSplitTransformer(";"),
			input:       "Name\tRoles\tDept\nAlice\tDev;Lead\tEng\nBob\tDev\tEng",
			expected:    "Name\tRoles\tDept\nAlice\tDev\tEng\n\tLead\t\nBob\tDev\tEng",
		},
		{
			name:        "no splits needed",
			transformer: NewLineSplitTransformerDefault(),
			input:       "Name\tAge\tCity\nAlice\t25\tNew York\nBob\t30\tChicago",
			expected:    "Name\tAge\tCity\nAlice\t25\tNew York\nBob\t30\tChicago",
		},
		{
			name:        "empty input",
			transformer: NewLineSplitTransformerDefault(),
			input:       "",
			expected:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.transformer.Transform(ctx, []byte(test.input), "table")
			if err != nil {
				t.Fatalf("LineSplitTransformer.Transform() error = %v", err)
			}

			if string(result) != test.expected {
				t.Errorf("LineSplitTransformer.Transform() = %q, want %q", string(result), test.expected)
			}
		})
	}
}

// Test RemoveColorsTransformer

func TestRemoveColorsTransformer_Name(t *testing.T) {
	transformer := NewRemoveColorsTransformer()
	if transformer.Name() != "remove-colors" {
		t.Errorf("RemoveColorsTransformer.Name() = %s, want remove-colors", transformer.Name())
	}
}

func TestRemoveColorsTransformer_Priority(t *testing.T) {
	transformer := NewRemoveColorsTransformer()
	if transformer.Priority() != 1000 {
		t.Errorf("RemoveColorsTransformer.Priority() = %d, want 1000", transformer.Priority())
	}
}

func TestRemoveColorsTransformer_CanTransform(t *testing.T) {
	transformer := NewRemoveColorsTransformer()

	// Should work with all formats
	formats := []string{"table", "csv", "html", "markdown", "json", "yaml", "dot"}
	for _, format := range formats {
		if !transformer.CanTransform(format) {
			t.Errorf("RemoveColorsTransformer.CanTransform(%s) = false, want true", format)
		}
	}
}

func TestRemoveColorsTransformer_Transform(t *testing.T) {
	transformer := NewRemoveColorsTransformer()
	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove basic color codes",
			input:    "\x1B[31mRed text\x1B[0m and \x1B[32mGreen text\x1B[0m",
			expected: "Red text and Green text",
		},
		{
			name:     "remove complex color codes",
			input:    "\x1B[1;31;40mBold red on black\x1B[0m",
			expected: "Bold red on black",
		},
		{
			name:     "no color codes",
			input:    "Plain text without colors",
			expected: "Plain text without colors",
		},
		{
			name:     "mixed content",
			input:    "Normal text \x1B[33mwarning\x1B[0m more text",
			expected: "Normal text warning more text",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := transformer.Transform(ctx, []byte(test.input), "table")
			if err != nil {
				t.Fatalf("RemoveColorsTransformer.Transform() error = %v", err)
			}

			if string(result) != test.expected {
				t.Errorf("RemoveColorsTransformer.Transform() = %q, want %q", string(result), test.expected)
			}
		})
	}
}

// Integration tests

func TestTransformers_Integration(t *testing.T) {
	ctx := context.Background()

	// Test emoji then color transformation
	emojiTransformer := &EmojiTransformer{}
	colorTransformer := NewColorTransformer()

	input := "Status: OK, Error: !!"

	// Apply emoji transformation first
	result1, err := emojiTransformer.Transform(ctx, []byte(input), "table")
	if err != nil {
		t.Fatalf("Emoji transform error: %v", err)
	}

	expected1 := "Status: ✅, Error: 🚨"
	if string(result1) != expected1 {
		t.Errorf("Emoji transform result = %s, want %s", string(result1), expected1)
	}

	// Apply color transformation
	result2, err := colorTransformer.Transform(ctx, result1, "table")
	if err != nil {
		t.Fatalf("Color transform error: %v", err)
	}

	// Color transformation may or may not add codes depending on terminal
	// Just verify it doesn't error and produces some output
	if len(result2) == 0 {
		t.Error("Color transformation should produce some output")
	}
}

func TestTransformers_PipelineOrder(t *testing.T) {
	// Test that transformers have correct priorities for proper ordering
	sort := NewSortTransformer("name", true)
	emoji := &EmojiTransformer{}
	color := NewColorTransformer()
	linesplit := NewLineSplitTransformerDefault()
	removeColors := NewRemoveColorsTransformer()

	// Verify priority order: sort < emoji < linesplit < color < removeColors
	if sort.Priority() >= emoji.Priority() {
		t.Error("Sort should have lower priority than emoji")
	}

	if emoji.Priority() >= linesplit.Priority() {
		t.Error("Emoji should have lower priority than linesplit")
	}

	if linesplit.Priority() >= color.Priority() {
		t.Error("LineSplit should have lower priority than color")
	}

	if color.Priority() >= removeColors.Priority() {
		t.Error("Color should have lower priority than removeColors")
	}
}

// Benchmark tests

func BenchmarkEmojiTransformer_Transform(b *testing.B) {
	transformer := &EmojiTransformer{}
	ctx := context.Background()
	input := []byte("Status: OK, Warning: !!, Active: Yes, Inactive: No, Running: true, Stopped: false")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(ctx, input, "table")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSortTransformer_Transform(b *testing.B) {
	transformer := NewSortTransformerAscending("Name")
	ctx := context.Background()

	// Create a larger dataset for benchmarking
	var lines []string
	lines = append(lines, "Name\tAge\tCity")
	for i := 0; i < 100; i++ {
		lines = append(lines, "Person"+string(rune(90-i%26))+"\t"+string(rune(20+i%50))+"\tCity"+string(rune(65+i%26)))
	}
	input := []byte(strings.Join(lines, "\n"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(ctx, input, "table")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLineSplitTransformer_Transform(b *testing.B) {
	transformer := NewLineSplitTransformerDefault()
	ctx := context.Background()
	input := []byte("Name\tTags\tStatus\nAlice\ttag1\ntag2\ntag3\tActive\nBob\ttag4\ntag5\tInactive")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(ctx, input, "table")
		if err != nil {
			b.Fatal(err)
		}
	}
}
