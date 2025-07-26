package output

import (
	"reflect"
	"testing"
)

// TestCollapsibleValue_Interface tests that DefaultCollapsibleValue implements CollapsibleValue
func TestCollapsibleValue_Interface(t *testing.T) {
	var _ CollapsibleValue = (*DefaultCollapsibleValue)(nil)
}

// TestNewCollapsibleValue tests the basic constructor functionality
func TestNewCollapsibleValue(t *testing.T) {
	cv := NewCollapsibleValue("test summary", "test details")

	if cv.Summary() != "test summary" {
		t.Errorf("Summary() = %q, want %q", cv.Summary(), "test summary")
	}

	if cv.Details() != "test details" {
		t.Errorf("Details() = %q, want %q", cv.Details(), "test details")
	}

	if cv.IsExpanded() != false {
		t.Errorf("IsExpanded() = %t, want %t", cv.IsExpanded(), false)
	}

	if cv.FormatHint("json") != nil {
		t.Errorf("FormatHint(\"json\") = %v, want nil", cv.FormatHint("json"))
	}
}

// TestNewCollapsibleValue_WithOptions tests constructor with functional options
func TestNewCollapsibleValue_WithOptions(t *testing.T) {
	formatHints := map[string]any{"class": "test-class"}
	cv := NewCollapsibleValue("summary", "details",
		WithExpanded(true),
		WithMaxLength(10),
		WithTruncateIndicator("..."),
		WithFormatHint("html", formatHints),
	)

	if !cv.IsExpanded() {
		t.Errorf("IsExpanded() = %t, want true", cv.IsExpanded())
	}

	if cv.maxDetailLength != 10 {
		t.Errorf("maxDetailLength = %d, want 10", cv.maxDetailLength)
	}

	if cv.truncateIndicator != "..." {
		t.Errorf("truncateIndicator = %q, want %q", cv.truncateIndicator, "...")
	}

	hints := cv.FormatHint("html")
	if !reflect.DeepEqual(hints, formatHints) {
		t.Errorf("FormatHint(\"html\") = %v, want %v", hints, formatHints)
	}
}

// TestCollapsibleValue_Summary tests Summary method with edge cases
func TestCollapsibleValue_Summary(t *testing.T) {
	tests := []struct {
		name     string
		summary  string
		expected string
	}{
		{"Normal summary", "test summary", "test summary"},
		{"Empty summary", "", "[no summary]"},
		{"Whitespace summary", "   ", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCollapsibleValue(tt.summary, "details")
			if got := cv.Summary(); got != tt.expected {
				t.Errorf("Summary() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestCollapsibleValue_Details tests Details method with various data types
func TestCollapsibleValue_Details(t *testing.T) {
	tests := []struct {
		name     string
		details  any
		expected any
	}{
		{"String details", "test details", "test details"},
		{"Array details", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"Map details", map[string]any{"key": "value"}, map[string]any{"key": "value"}},
		{"Number details", 42, 42},
		{"Bool details", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCollapsibleValue("summary", tt.details)
			if got := cv.Details(); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Details() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestCollapsibleValue_NilDetails tests handling of nil details
func TestCollapsibleValue_NilDetails(t *testing.T) {
	cv := NewCollapsibleValue("test summary", nil)

	// Should fallback to summary when details are nil
	if got := cv.Details(); got != "test summary" {
		t.Errorf("Details() with nil = %v, want %q", got, "test summary")
	}
}

// TestCollapsibleValue_CharacterTruncation tests character limit functionality
func TestCollapsibleValue_CharacterTruncation(t *testing.T) {
	longDetails := "This is a very long string that should be truncated when the character limit is exceeded"

	tests := []struct {
		name              string
		maxLength         int
		truncateIndicator string
		details           any
		expectedPrefix    string
		shouldTruncate    bool
	}{
		{
			name:              "Truncate long string",
			maxLength:         20,
			truncateIndicator: "[...truncated]",
			details:           longDetails,
			expectedPrefix:    "This is a very long ",
			shouldTruncate:    true,
		},
		{
			name:              "No truncation for short string",
			maxLength:         200,
			truncateIndicator: "[...truncated]",
			details:           "Short string",
			expectedPrefix:    "Short string",
			shouldTruncate:    false,
		},
		{
			name:              "Zero length disables truncation",
			maxLength:         0,
			truncateIndicator: "[...truncated]",
			details:           longDetails,
			expectedPrefix:    longDetails,
			shouldTruncate:    false,
		},
		{
			name:              "Non-string details not truncated",
			maxLength:         5,
			truncateIndicator: "[...truncated]",
			details:           []string{"a", "b", "c"},
			expectedPrefix:    "",
			shouldTruncate:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCollapsibleValue("summary", tt.details,
				WithMaxLength(tt.maxLength),
				WithTruncateIndicator(tt.truncateIndicator),
			)

			result := cv.Details()

			if tt.shouldTruncate {
				resultStr, ok := result.(string)
				if !ok {
					t.Errorf("Expected string result for truncation test, got %T", result)
					return
				}

				expectedResult := tt.expectedPrefix + tt.truncateIndicator
				if resultStr != expectedResult {
					t.Errorf("Details() = %q, want %q", resultStr, expectedResult)
				}
			} else {
				if tt.name == "Non-string details not truncated" {
					// Non-string details should be returned as-is
					if !reflect.DeepEqual(result, tt.details) {
						t.Errorf("Details() = %v, want %v", result, tt.details)
					}
				} else {
					// String details that don't need truncation
					if result != tt.details {
						t.Errorf("Details() = %v, want %v", result, tt.details)
					}
				}
			}
		})
	}
}

// TestCollapsibleValue_IsExpanded tests expansion state
func TestCollapsibleValue_IsExpanded(t *testing.T) {
	tests := []struct {
		name     string
		expanded bool
	}{
		{"Default collapsed", false},
		{"Explicitly expanded", true},
		{"Explicitly collapsed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cv *DefaultCollapsibleValue
			if tt.name == "Default collapsed" {
				cv = NewCollapsibleValue("summary", "details")
			} else {
				cv = NewCollapsibleValue("summary", "details", WithExpanded(tt.expanded))
			}

			if got := cv.IsExpanded(); got != tt.expanded {
				t.Errorf("IsExpanded() = %t, want %t", got, tt.expanded)
			}
		})
	}
}

// TestCollapsibleValue_FormatHint tests format-specific hints
func TestCollapsibleValue_FormatHint(t *testing.T) {
	htmlHints := map[string]any{"class": "collapsible", "style": "color: blue"}
	jsonHints := map[string]any{"array_format": "compact"}

	cv := NewCollapsibleValue("summary", "details",
		WithFormatHint("html", htmlHints),
		WithFormatHint("json", jsonHints),
	)

	// Test existing format hints
	if got := cv.FormatHint("html"); !reflect.DeepEqual(got, htmlHints) {
		t.Errorf("FormatHint(\"html\") = %v, want %v", got, htmlHints)
	}

	if got := cv.FormatHint("json"); !reflect.DeepEqual(got, jsonHints) {
		t.Errorf("FormatHint(\"json\") = %v, want %v", got, jsonHints)
	}

	// Test non-existing format hint
	if got := cv.FormatHint("yaml"); got != nil {
		t.Errorf("FormatHint(\"yaml\") = %v, want nil", got)
	}
}

// TestCollapsibleValue_DefaultConfiguration tests default values
func TestCollapsibleValue_DefaultConfiguration(t *testing.T) {
	cv := NewCollapsibleValue("summary", "details")

	// Test default values
	if cv.maxDetailLength != 500 {
		t.Errorf("Default maxDetailLength = %d, want 500", cv.maxDetailLength)
	}

	if cv.truncateIndicator != "[...truncated]" {
		t.Errorf("Default truncateIndicator = %q, want %q", cv.truncateIndicator, "[...truncated]")
	}

	if cv.defaultExpanded != false {
		t.Errorf("Default defaultExpanded = %t, want false", cv.defaultExpanded)
	}

	if cv.formatHints == nil {
		t.Error("formatHints should be initialized, got nil")
	}
}

// TestCollapsibleValue_String tests the String method for debugging
func TestCollapsibleValue_String(t *testing.T) {
	tests := []struct {
		name     string
		summary  string
		expanded bool
		expected string
	}{
		{
			name:     "Collapsed value",
			summary:  "test summary",
			expanded: false,
			expected: `CollapsibleValue{summary: "test summary", expanded: false}`,
		},
		{
			name:     "Expanded value",
			summary:  "another summary",
			expanded: true,
			expected: `CollapsibleValue{summary: "another summary", expanded: true}`,
		},
		{
			name:     "Empty summary",
			summary:  "",
			expanded: false,
			expected: `CollapsibleValue{summary: "", expanded: false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCollapsibleValue(tt.summary, "details", WithExpanded(tt.expanded))
			if got := cv.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestCollapsibleValue_MultipleFormatHints tests multiple format hints
func TestCollapsibleValue_MultipleFormatHints(t *testing.T) {
	cv := NewCollapsibleValue("summary", "details")

	// Add multiple format hints
	htmlHints := map[string]any{"class": "html-class"}
	jsonHints := map[string]any{"compact": true}
	yamlHints := map[string]any{"indent": 2}

	cv = NewCollapsibleValue("summary", "details",
		WithFormatHint("html", htmlHints),
		WithFormatHint("json", jsonHints),
		WithFormatHint("yaml", yamlHints),
	)

	// Verify all hints are stored correctly
	if got := cv.FormatHint("html"); !reflect.DeepEqual(got, htmlHints) {
		t.Errorf("FormatHint(\"html\") = %v, want %v", got, htmlHints)
	}

	if got := cv.FormatHint("json"); !reflect.DeepEqual(got, jsonHints) {
		t.Errorf("FormatHint(\"json\") = %v, want %v", got, jsonHints)
	}

	if got := cv.FormatHint("yaml"); !reflect.DeepEqual(got, yamlHints) {
		t.Errorf("FormatHint(\"yaml\") = %v, want %v", got, yamlHints)
	}
}

// TestCollapsibleValue_EdgeCases tests various edge cases
func TestCollapsibleValue_EdgeCases(t *testing.T) {
	t.Run("Very long summary", func(t *testing.T) {
		longSummary := string(make([]byte, 1000))
		for i := range longSummary {
			longSummary = longSummary[:i] + "a" + longSummary[i+1:]
		}

		cv := NewCollapsibleValue(longSummary, "details")
		if cv.Summary() != longSummary {
			t.Error("Long summary should be preserved exactly")
		}
	})

	t.Run("Complex nested details", func(t *testing.T) {
		complexDetails := map[string]any{
			"level1": map[string]any{
				"level2": []any{
					map[string]any{"key": "value"},
					[]string{"a", "b", "c"},
					42,
				},
			},
		}

		cv := NewCollapsibleValue("summary", complexDetails)
		if !reflect.DeepEqual(cv.Details(), complexDetails) {
			t.Error("Complex nested details should be preserved exactly")
		}
	})

	t.Run("Negative max length", func(t *testing.T) {
		cv := NewCollapsibleValue("summary", "long details", WithMaxLength(-1))
		// Negative length should disable truncation
		if cv.Details() != "long details" {
			t.Error("Negative max length should disable truncation")
		}
	})
}
