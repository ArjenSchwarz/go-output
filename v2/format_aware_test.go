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
		{"table", true},
		{"markdown", true},
		{"html", true},
		{"csv", true},
		{"yaml", true},
		{"json", false},
		{"dot", false},
		{"mermaid", false},
		{"drawio", false},
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
		{"json", true},
		{"yaml", true},
		{"table", false},
		{"html", false},
		{"csv", false},
		{"markdown", false},
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
		{"table", true},
		{"csv", true},
		{"html", true},
		{"markdown", true},
		{"json", false},
		{"yaml", false},
		{"dot", false},
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
		{"dot", true},
		{"mermaid", true},
		{"drawio", true},
		{"table", false},
		{"json", false},
		{"html", false},
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
		{"table", true},
		{"html", false},
		{"markdown", false},
		{"csv", false},
		{"json", false},
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
		{"table", true},
		{"html", true},
		{"markdown", true},
		{"csv", true},
		{"yaml", false}, // Structured format
		{"json", false}, // Structured format
		{"dot", false},  // Not text-based
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
		{"html", true},
		{"markdown", true},
		{"csv", true},
		{"json", true},
		{"yaml", true},
		{"table", false},
		{"dot", false},
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
		{"table", true},    // Text-based and supports emoji
		{"html", true},     // Text-based and supports emoji
		{"markdown", true}, // Text-based and supports emoji
		{"csv", true},      // Text-based and supports emoji
		{"yaml", false},    // Structured format, no emoji
		{"json", false},    // Structured format, no emoji
		{"dot", false},     // Not text-based
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
		{"table", true},     // Supports colors
		{"html", false},     // No color support
		{"markdown", false}, // No color support
		{"csv", false},      // No color support
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
		{"table", true},
		{"csv", true},
		{"html", true},
		{"markdown", true},
		{"json", false}, // Not tabular
		{"yaml", false}, // Not tabular
		{"dot", false},  // Not tabular
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
	result, err := wrapper.Transform(ctx, originalData, "table")
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
		{"table", true},
		{"html", true},
		{"markdown", true},
		{"csv", true},
		{"yaml", false}, // Structured format
		{"json", false}, // Structured format
		{"dot", false},  // Graph format
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

	tests := []struct {
		name     string
		format   string
		input    string
		expected string
	}{
		{
			name:     "markdown format conservative emoji",
			format:   "markdown",
			input:    "!! Warning OK",
			expected: "⚠️ Warning ✅",
		},
		{
			name:     "html format with HTML entities",
			format:   "html",
			input:    "!! Warning OK Yes No",
			expected: "&#x1F6A8; Warning &#x2705; &#x2705; &#x274C;",
		},
		{
			name:     "table format default behavior",
			format:   "table",
			input:    "!! Warning OK",
			expected: "🚨 Warning ✅",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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
		{"table", true}, // Only format that supports colors
		{"html", false},
		{"markdown", false},
		{"csv", false},
		{"json", false},
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
	result, err := transformer.Transform(ctx, input, "html")
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
		{"table", true},
		{"csv", true},
		{"html", true},
		{"markdown", true},
		{"json", false}, // Not tabular
		{"yaml", false}, // Not tabular
		{"dot", false},  // Not tabular
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
	result, err := transformer.Transform(ctx, input, "json")
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
	result1, err := emojiTransformer.Transform(ctx, input, "table")
	if err != nil {
		t.Fatalf("Enhanced emoji transform error: %v", err)
	}

	// Verify original input is unchanged
	if string(input) != "Status: OK, Error: !!" {
		t.Error("Original input was modified during emoji transformation")
	}

	// Apply color transformation
	result2, err := colorTransformer.Transform(ctx, result1, "table")
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
	if emojiTransformer.CanTransform("json") {
		t.Error("Enhanced emoji transformer should not support JSON format")
	}

	// Transformation should not be attempted
	result, err := emojiTransformer.Transform(ctx, input, "json")
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.SupportsEmoji("table")
		detector.SupportsEmoji("html")
		detector.SupportsEmoji("json")
		detector.SupportsEmoji("dot")
	}
}

func BenchmarkEnhancedEmojiTransformer_Transform(b *testing.B) {
	transformer := NewEnhancedEmojiTransformer()
	ctx := context.Background()
	input := []byte("Status: OK, Warning: !!, Active: Yes, Inactive: No")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(ctx, input, "table")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataIntegrityValidator_ValidateIntegrity(b *testing.B) {
	originalData := []byte("This is some test data that we want to validate for integrity")
	validator := NewDataIntegrityValidator(originalData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateIntegrity(originalData)
	}
}
