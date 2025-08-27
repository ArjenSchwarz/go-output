package output

import (
	"context"
	"testing"
)

// Test FormatDetector

func TestFormatDetector_IsTextBasedFormat(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true},
		{FormatMarkdown, true},
		{FormatHTML, true},
		{FormatCSV, true},
		{FormatYAML, true},
		{FormatJSON, false},
		{FormatDOT, false},
		{FormatMermaid, false},
		{FormatDrawIO, false},
	}

	for _, test := range tests {
		result := detector.IsTextBasedFormat(test.format)
		if result != test.expected {
			t.Errorf("IsTextBasedFormat(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestFormatDetector_IsStructuredFormat(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatJSON, true},
		{FormatYAML, true},
		{FormatTable, false},
		{FormatHTML, false},
		{FormatCSV, false},
		{FormatMarkdown, false},
	}

	for _, test := range tests {
		result := detector.IsStructuredFormat(test.format)
		if result != test.expected {
			t.Errorf("IsStructuredFormat(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestFormatDetector_IsTabularFormat(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true},
		{FormatCSV, true},
		{FormatHTML, true},
		{FormatMarkdown, true},
		{FormatJSON, false},
		{FormatYAML, false},
		{FormatDOT, false},
	}

	for _, test := range tests {
		result := detector.IsTabularFormat(test.format)
		if result != test.expected {
			t.Errorf("IsTabularFormat(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestFormatDetector_IsGraphFormat(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatDOT, true},
		{FormatMermaid, true},
		{FormatDrawIO, true},
		{FormatTable, false},
		{FormatJSON, false},
		{FormatHTML, false},
	}

	for _, test := range tests {
		result := detector.IsGraphFormat(test.format)
		if result != test.expected {
			t.Errorf("IsGraphFormat(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestFormatDetector_SupportsColors(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true},
		{FormatHTML, false},
		{FormatMarkdown, false},
		{FormatCSV, false},
		{FormatJSON, false},
	}

	for _, test := range tests {
		result := detector.SupportsColors(test.format)
		if result != test.expected {
			t.Errorf("SupportsColors(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestFormatDetector_SupportsEmoji(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true},
		{FormatHTML, true},
		{FormatMarkdown, true},
		{FormatCSV, true},
		{FormatYAML, false}, // Structured format
		{FormatJSON, false}, // Structured format
		{FormatDOT, false},  // Not text-based
	}

	for _, test := range tests {
		result := detector.SupportsEmoji(test.format)
		if result != test.expected {
			t.Errorf("SupportsEmoji(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestFormatDetector_RequiresEscaping(t *testing.T) {
	detector := NewFormatDetector()

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatHTML, true},
		{FormatMarkdown, true},
		{FormatCSV, true},
		{FormatJSON, true},
		{FormatYAML, true},
		{FormatTable, false},
		{FormatDOT, false},
	}

	for _, test := range tests {
		result := detector.RequiresEscaping(test.format)
		if result != test.expected {
			t.Errorf("RequiresEscaping(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

// Test FormatAwareTransformer

func TestFormatAwareTransformer_Wrapping(t *testing.T) {
	originalTransformer := &EmojiTransformer{}
	wrapper := NewFormatAwareTransformer(originalTransformer)

	if wrapper.Name() != originalTransformer.Name() {
		t.Errorf("FormatAwareTransformer.Name() = %s, want %s", wrapper.Name(), originalTransformer.Name())
	}

	if wrapper.Priority() != originalTransformer.Priority() {
		t.Errorf("FormatAwareTransformer.Priority() = %d, want %d", wrapper.Priority(), originalTransformer.Priority())
	}
}

func TestFormatAwareTransformer_EmojiCanTransform(t *testing.T) {
	emojiTransformer := &EmojiTransformer{}
	wrapper := NewFormatAwareTransformer(emojiTransformer)

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true},    // Text-based and supports emoji
		{FormatHTML, true},     // Text-based and supports emoji
		{FormatMarkdown, true}, // Text-based and supports emoji
		{FormatCSV, true},      // Text-based and supports emoji
		{FormatYAML, false},    // Structured format, no emoji
		{FormatJSON, false},    // Structured format, no emoji
		{FormatDOT, false},     // Not text-based
	}

	for _, test := range tests {
		result := wrapper.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("FormatAwareTransformer(emoji).CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestFormatAwareTransformer_ColorCanTransform(t *testing.T) {
	colorTransformer := NewColorTransformer()
	wrapper := NewFormatAwareTransformer(colorTransformer)

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true},     // Supports colors
		{FormatHTML, false},     // No color support
		{FormatMarkdown, false}, // No color support
		{FormatCSV, false},      // No color support
	}

	for _, test := range tests {
		result := wrapper.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("FormatAwareTransformer(color).CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestFormatAwareTransformer_SortCanTransform(t *testing.T) {
	sortTransformer := NewSortTransformer("name", true)
	wrapper := NewFormatAwareTransformer(sortTransformer)

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true},
		{FormatCSV, true},
		{FormatHTML, true},
		{FormatMarkdown, true},
		{FormatJSON, false}, // Not tabular
		{FormatYAML, false}, // Not tabular
		{FormatDOT, false},  // Not tabular
	}

	for _, test := range tests {
		result := wrapper.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("FormatAwareTransformer(sort).CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestFormatAwareTransformer_DataIntegrity(t *testing.T) {
	originalData := []byte("Status: OK, Error: !!")
	emojiTransformer := &EmojiTransformer{}
	wrapper := NewFormatAwareTransformer(emojiTransformer)

	ctx := context.Background()
	result, err := wrapper.Transform(ctx, originalData, FormatTable)
	if err != nil {
		t.Fatalf("FormatAwareTransformer.Transform() error = %v", err)
	}

	// Verify original data is unchanged
	expectedOriginal := "Status: OK, Error: !!"
	if string(originalData) != expectedOriginal {
		t.Errorf("Original data was modified: got %s, want %s", string(originalData), expectedOriginal)
	}

	// Verify transformation was applied to result
	expectedResult := "Status: ✅, Error: 🚨"
	if string(result) != expectedResult {
		t.Errorf("Transform result = %s, want %s", string(result), expectedResult)
	}
}

// Test EnhancedEmojiTransformer

func TestEnhancedEmojiTransformer_CanTransform(t *testing.T) {
	transformer := NewEnhancedEmojiTransformer()

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true},
		{FormatHTML, true},
		{FormatMarkdown, true},
		{FormatCSV, true},
		{FormatYAML, false}, // Structured format
		{FormatJSON, false}, // Structured format
		{FormatDOT, false},  // Graph format
	}

	for _, test := range tests {
		result := transformer.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("EnhancedEmojiTransformer.CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestEnhancedEmojiTransformer_FormatSpecificTransform(t *testing.T) {
	transformer := NewEnhancedEmojiTransformer()
	ctx := context.Background()

	tests := map[string]struct {
		format   string
		input    string
		expected string
	}{"html format with HTML entities": {

		format:   FormatHTML,
		input:    "!! Warning OK Yes No",
		expected: "&#x1F6A8; Warning &#x2705; &#x2705; &#x274C;",
	}, "markdown format conservative emoji": {

		format:   FormatMarkdown,
		input:    "!! Warning OK",
		expected: "⚠️ Warning ✅",
	}, "table format default behavior": {

		format:   FormatTable,
		input:    "!! Warning OK",
		expected: "🚨 Warning ✅",
	}}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := transformer.Transform(ctx, []byte(test.input), test.format)
			if err != nil {
				t.Fatalf("EnhancedEmojiTransformer.Transform() error = %v", err)
			}

			if string(result) != test.expected {
				t.Errorf("EnhancedEmojiTransformer.Transform() = %s, want %s", string(result), test.expected)
			}
		})
	}
}

// Test EnhancedColorTransformer

func TestEnhancedColorTransformer_CanTransform(t *testing.T) {
	transformer := NewEnhancedColorTransformer()

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true}, // Only format that supports colors
		{FormatHTML, false},
		{FormatMarkdown, false},
		{FormatCSV, false},
		{FormatJSON, false},
	}

	for _, test := range tests {
		result := transformer.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("EnhancedColorTransformer.CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestEnhancedColorTransformer_NonColorFormat(t *testing.T) {
	transformer := NewEnhancedColorTransformer()
	ctx := context.Background()

	input := []byte("Status: ✅")
	result, err := transformer.Transform(ctx, input, FormatHTML)
	if err != nil {
		t.Fatalf("EnhancedColorTransformer.Transform() error = %v", err)
	}

	// Should return unchanged input for non-color formats
	if string(result) != string(input) {
		t.Errorf("EnhancedColorTransformer should not modify non-color formats")
	}
}

// Test EnhancedSortTransformer

func TestEnhancedSortTransformer_CanTransform(t *testing.T) {
	transformer := NewEnhancedSortTransformer("name", true)

	tests := []struct {
		format   string
		expected bool
	}{
		{FormatTable, true},
		{FormatCSV, true},
		{FormatHTML, true},
		{FormatMarkdown, true},
		{FormatJSON, false}, // Not tabular
		{FormatYAML, false}, // Not tabular
		{FormatDOT, false},  // Not tabular
	}

	for _, test := range tests {
		result := transformer.CanTransform(test.format)
		if result != test.expected {
			t.Errorf("EnhancedSortTransformer.CanTransform(%s) = %t, want %t", test.format, result, test.expected)
		}
	}
}

func TestEnhancedSortTransformer_NonTabularFormat(t *testing.T) {
	transformer := NewEnhancedSortTransformer("name", true)
	ctx := context.Background()

	input := []byte(`{"name": "Alice", "age": 30}`)
	result, err := transformer.Transform(ctx, input, FormatJSON)
	if err != nil {
		t.Fatalf("EnhancedSortTransformer.Transform() error = %v", err)
	}

	// Should return unchanged input for non-tabular formats
	if string(result) != string(input) {
		t.Errorf("EnhancedSortTransformer should not modify non-tabular formats")
	}
}

// Test DataIntegrityValidator

func TestDataIntegrityValidator_ValidateIntegrity(t *testing.T) {
	originalData := []byte("Original data content")
	validator := NewDataIntegrityValidator(originalData)

	// Test with unchanged data
	if !validator.ValidateIntegrity(originalData) {
		t.Error("ValidateIntegrity should return true for unchanged data")
	}

	// Test with modified data
	modifiedData := []byte("Modified data content")
	if validator.ValidateIntegrity(modifiedData) {
		t.Error("ValidateIntegrity should return false for modified data")
	}

	// Test with different length data
	shorterData := []byte("Short")
	if validator.ValidateIntegrity(shorterData) {
		t.Error("ValidateIntegrity should return false for different length data")
	}
}

func TestDataIntegrityValidator_GetOriginalData(t *testing.T) {
	originalData := []byte("Original data content")
	validator := NewDataIntegrityValidator(originalData)

	// Get a copy of original data
	copy := validator.GetOriginalData()

	// Verify it's the same content
	if string(copy) != string(originalData) {
		t.Errorf("GetOriginalData() returned different content")
	}

	// Modify the copy and verify original is unchanged
	copy[0] = 'X'
	if string(validator.GetOriginalData()) != string(originalData) {
		t.Error("Modifying copy affected original data")
	}
}

// Integration tests

func TestFormatAware_Integration(t *testing.T) {
	ctx := context.Background()

	// Test that enhanced transformers work together properly
	emojiTransformer := NewEnhancedEmojiTransformer()
	colorTransformer := NewEnhancedColorTransformer()

	input := []byte("Status: OK, Error: !!")

	// Apply emoji transformation first
	result1, err := emojiTransformer.Transform(ctx, input, FormatTable)
	if err != nil {
		t.Fatalf("Enhanced emoji transform error: %v", err)
	}

	// Verify original input is unchanged
	if string(input) != "Status: OK, Error: !!" {
		t.Error("Original input was modified during emoji transformation")
	}

	// Apply color transformation
	result2, err := colorTransformer.Transform(ctx, result1, FormatTable)
	if err != nil {
		t.Fatalf("Enhanced color transform error: %v", err)
	}

	// Verify result1 is unchanged
	expectedAfterEmoji := "Status: ✅, Error: 🚨"
	if string(result1) != expectedAfterEmoji {
		t.Error("First transformation result was modified during second transformation")
	}

	// Final result should have colors applied (though exact content depends on terminal)
	if len(result2) == 0 {
		t.Error("Color transformation should produce some output")
	}
}

func TestFormatAware_UnsupportedFormat(t *testing.T) {
	ctx := context.Background()

	// Test that transformers properly handle unsupported formats
	emojiTransformer := NewEnhancedEmojiTransformer()
	input := []byte(`{"status": "OK", "error": "!!"}`)

	// JSON doesn't support emoji
	if emojiTransformer.CanTransform(FormatJSON) {
		t.Error("Enhanced emoji transformer should not support JSON format")
	}

	// Transformation should not be attempted
	result, err := emojiTransformer.Transform(ctx, input, FormatJSON)
	if err != nil {
		t.Fatalf("Transform should not error on unsupported format: %v", err)
	}

	// Result should be unchanged for unsupported formats
	if string(result) != string(input) {
		t.Error("Transform should not modify content for unsupported formats")
	}
}

// Benchmark tests

func BenchmarkFormatDetector_SupportsEmoji(b *testing.B) {
	detector := NewFormatDetector()

	for b.Loop() {
		detector.SupportsEmoji(FormatTable)
		detector.SupportsEmoji(FormatHTML)
		detector.SupportsEmoji(FormatJSON)
		detector.SupportsEmoji(FormatDOT)
	}
}

func BenchmarkEnhancedEmojiTransformer_Transform(b *testing.B) {
	transformer := NewEnhancedEmojiTransformer()
	ctx := context.Background()
	input := []byte("Status: OK, Warning: !!, Active: Yes, Inactive: No")

	for b.Loop() {
		_, err := transformer.Transform(ctx, input, FormatTable)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataIntegrityValidator_ValidateIntegrity(b *testing.B) {
	originalData := []byte("This is some test data that we want to validate for integrity")
	validator := NewDataIntegrityValidator(originalData)

	for b.Loop() {
		validator.ValidateIntegrity(originalData)
	}
}
