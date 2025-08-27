package output

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

// TestCollapsibleValue_ErrorHandling tests error handling and edge cases in CollapsibleValue
func TestCollapsibleValue_ErrorHandling(t *testing.T) {
	tests := map[string]struct {
		summary  string
		details  any
		expected struct {
			summary string
			details any
		}
	}{"character limit truncation":

	// Should fallback to summary

	{

		summary: "test",
		details: strings.Repeat("a", 600), // Longer than default 500 limit
		expected: struct {
			summary string
			details any
		}{
			summary: "test",
			details: strings.Repeat("a", 500) + "[...truncated]",
		},
	}, "empty summary fallback": {

		summary: "",
		details: "some details",
		expected: struct {
			summary string
			details any
		}{
			summary: "[no summary]",
			details: "some details",
		},
	}, "nil details fallback": {

		summary: "test summary",
		details: nil,
		expected: struct {
			summary string
			details any
		}{
			summary: "test summary",
			details: "test summary",
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cv := NewCollapsibleValue(tt.summary, tt.details)

			if cv.Summary() != tt.expected.summary {
				t.Errorf("Summary() = %q, want %q", cv.Summary(), tt.expected.summary)
			}

			if cv.Details() != tt.expected.details {
				t.Errorf("Details() = %q, want %q", cv.Details(), tt.expected.details)
			}
		})
	}
}

// TestNestedCollapsibleValue_Prevention tests prevention of nested CollapsibleValues
func TestNestedCollapsibleValue_Prevention(t *testing.T) {
	// Create a base CollapsibleValue
	innerCV := NewCollapsibleValue("inner summary", "inner details")

	tests := map[string]struct {
		formatter func(any) any
		input     any
		expected  any
	}{"CollapsibleFormatter with CollapsibleValue details":

	// Should return original CollapsibleValue

	// Should return original CollapsibleValue

	// Should return original CollapsibleValue

	// Should return original CollapsibleValue

	{

		formatter: CollapsibleFormatter("template %v", func(v any) any {
			return innerCV // Return CollapsibleValue as details
		}),
		input:    "test",
		expected: "test", // Should return original value to prevent nesting
	}, "CollapsibleFormatter with CollapsibleValue input": {

		formatter: CollapsibleFormatter("template %v", func(v any) any { return "details" }),
		input:     innerCV,
		expected:  innerCV,
	}, "ErrorListFormatter with CollapsibleValue input": {

		formatter: ErrorListFormatter(),
		input:     innerCV,
		expected:  innerCV,
	}, "FilePathFormatter with CollapsibleValue input": {

		formatter: FilePathFormatter(10),
		input:     innerCV,
		expected:  innerCV,
	}, "JSONFormatter with CollapsibleValue input": {

		formatter: JSONFormatter(10),
		input:     innerCV,
		expected:  innerCV,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := tt.formatter(tt.input)
			if result != tt.expected {
				t.Errorf("formatter result = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMarkdownRenderer_ErrorRecovery tests error recovery in markdown renderer
func TestMarkdownRenderer_ErrorRecovery(t *testing.T) {
	renderer := NewMarkdownRendererWithCollapsible(DefaultRendererConfig)

	tests := map[string]struct {
		cv       CollapsibleValue
		expected string
	}{"empty summary": {

		cv:       NewCollapsibleValue("", "details"),
		expected: "<details><summary>[no summary]</summary><br/>details</details>",
	}, "nested CollapsibleValue in details": {

		cv:       NewCollapsibleValue("outer", NewCollapsibleValue("inner", "inner details")),
		expected: "<details><summary>outer</summary><br/>[nested collapsible: inner]</details>",
	}, "nil CollapsibleValue": {

		cv:       nil,
		expected: "[invalid collapsible value]",
	}, "nil details": {

		cv:       NewCollapsibleValue("summary", nil),
		expected: "<details><summary>summary</summary><br/>summary</details>", // Nil details fallback to summary
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var result string
			if tt.cv == nil {
				// Test nil handling directly
				result = renderer.(*markdownRenderer).renderCollapsibleValue(nil)
			} else {
				result = renderer.(*markdownRenderer).renderCollapsibleValue(tt.cv)
			}

			if result != tt.expected {
				t.Errorf("renderCollapsibleValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestTableRenderer_ErrorRecovery tests error recovery in table renderer
func TestTableRenderer_ErrorRecovery(t *testing.T) {
	renderer := NewTableRendererWithCollapsible("default", DefaultRendererConfig)
	tableRenderer := renderer.(*tableRenderer)

	tests := map[string]struct {
		cv       CollapsibleValue
		expected string
	}{"empty summary": {

		cv:       NewCollapsibleValue("", "details"),
		expected: "[no summary] [details hidden - use --expand for full view]",
	}, "nested CollapsibleValue in details": {

		cv:       NewCollapsibleValue("outer", NewCollapsibleValue("inner", "inner details")),
		expected: "outer [details hidden - use --expand for full view]",
	}, "nil CollapsibleValue": {

		cv:       nil,
		expected: "[invalid collapsible value]",
	}, "nil details": {

		cv:       NewCollapsibleValue("summary", nil),
		expected: "summary [details hidden - use --expand for full view]",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var result string
			if tt.cv == nil {
				// Test nil handling directly
				result = tableRenderer.renderCollapsibleValueSafe(nil)
			} else {
				result = tableRenderer.renderCollapsibleValueSafe(tt.cv)
			}

			if result != tt.expected {
				t.Errorf("renderCollapsibleValueSafe() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestCharacterLimitEnforcement tests character limit enforcement through public API
func TestCharacterLimitEnforcement(t *testing.T) {
	config := RendererConfig{
		MaxDetailLength:   20,
		TruncateIndicator: "...",
	}

	longDetails := "This is a very long string that exceeds the 20 character limit"

	// Test through public API - create a table with collapsible content
	data := []map[string]any{
		{"summary": "test", "details": NewCollapsibleValue("summary", longDetails)},
	}

	// Test markdown renderer
	markdownRenderer := NewMarkdownRendererWithCollapsible(config)
	table, err := NewTableContent("Test", data, WithKeys("summary", "details"))
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	doc := New().AddContent(table).Build()
	_, err = markdownRenderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Should contain truncated content - the CollapsibleValue itself should handle truncation
	cv := NewCollapsibleValue("summary", longDetails, WithMaxLength(20), WithTruncateIndicator("..."))
	truncatedDetails := cv.Details().(string)

	if !strings.Contains(truncatedDetails, "...") {
		t.Errorf("Expected truncated details to contain '...', got: %s", truncatedDetails)
	}

	if len(truncatedDetails) > 23 { // 20 chars + "..." = 23
		t.Errorf("Expected details length <= 23, got %d: %s", len(truncatedDetails), truncatedDetails)
	}
}

// TestComplexDataErrorHandling tests error handling with complex data structures
func TestComplexDataErrorHandling(t *testing.T) {
	tests := map[string]struct {
		details any
		expect  string
	}{"empty map": {

		details: map[string]any{},
		expect:  "[empty map]",
	}, "empty string array": {

		details: []string{},
		expect:  "[empty list]",
	}, "map with nil values": {

		details: map[string]any{"key1": nil, "key2": "value"},
		expect:  "<strong>key1:</strong> [nil]", // Should handle nil values with HTML formatting
	}, "nil interface": {

		details: nil,
		expect:  "test", // Nil details fallback to summary
	}}

	// Test through public API by creating tables with complex data
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cv := NewCollapsibleValue("test", tt.details)
			data := []map[string]any{
				{"item": cv},
			}

			table, err := NewTableContent("Test", data, WithKeys("item"))
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			markdownRenderer := NewMarkdownRendererWithCollapsible(DefaultRendererConfig)
			doc := New().AddContent(table).Build()
			result, err := markdownRenderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render: %v", err)
			}

			if !strings.Contains(string(result), tt.expect) {
				t.Errorf("Expected result to contain %q, got %q", tt.expect, string(result))
			}
		})
	}
}

// TestPanicRecovery tests that panic recovery works correctly through public API
func TestPanicRecovery(t *testing.T) {
	// Capture stderr to verify error logging
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	defer func() {
		w.Close()
		os.Stderr = oldStderr
	}()

	// Create a custom CollapsibleValue that panics
	panicCV := &panicCollapsibleValue{}

	// Test through public API
	data := []map[string]any{
		{"item": panicCV},
	}

	table, err := NewTableContent("Test", data, WithKeys("item"))
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	markdownRenderer := NewMarkdownRendererWithCollapsible(DefaultRendererConfig)
	doc := New().AddContent(table).Build()

	// This should not panic, but should log an error
	result, err := markdownRenderer.Render(context.Background(), doc)

	// Close write end to read from pipe
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderrOutput := buf.String()

	// Verify that error was logged and function returned without panicking
	if !strings.Contains(stderrOutput, "Error") {
		t.Error("Expected error to be logged to stderr")
	}

	// Should still produce some output even with panicking CollapsibleValue
	if err != nil {
		t.Errorf("Expected no error from Render, got: %v", err)
	}

	if len(result) == 0 {
		t.Error("Expected non-empty result after panic recovery")
	}
}

// panicCollapsibleValue is a test helper that panics on method calls
type panicCollapsibleValue struct{}

func (p *panicCollapsibleValue) Summary() string {
	panic("test panic in Summary()")
}

func (p *panicCollapsibleValue) Details() any {
	panic("test panic in Details()")
}

func (p *panicCollapsibleValue) IsExpanded() bool {
	panic("test panic in IsExpanded()")
}

func (p *panicCollapsibleValue) FormatHint(format string) map[string]any {
	panic("test panic in FormatHint()")
}

// TestFieldFormatterErrorHandling tests error handling in field formatters
func TestFieldFormatterErrorHandling(t *testing.T) {
	tests := map[string]struct {
		formatter func(any) any
		input     any
		expected  any
	}{"CollapsibleFormatter with nil detail function":

	// Not a string array or error array
	// Should return original value

	// Not a string
	// Should return original value

	// Cannot be marshaled to JSON
	// Should return original value

	{

		formatter: CollapsibleFormatter("template", nil),
		input:     "test",
		expected:  "test", // Should return original value
	}, "ErrorListFormatter with invalid type": {

		formatter: ErrorListFormatter(),
		input:     123,
		expected:  123,
	}, "FilePathFormatter with non-string": {

		formatter: FilePathFormatter(10),
		input:     123,
		expected:  123,
	}, "JSONFormatter with unmarshalable data": {

		formatter: JSONFormatter(10),
		input:     make(chan int),
		expected:  make(chan int),
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := tt.formatter(tt.input)

			// For channel comparison, we need to handle it specially
			if name == "JSONFormatter with unmarshalable data" {
				if result == nil {
					t.Error("Expected non-nil result for unmarshalable data")
				}
			} else {
				if result != tt.expected {
					t.Errorf("formatter result = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}
