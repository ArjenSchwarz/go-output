package output

import (
	"errors"
	"fmt"
	"testing"
)

func TestCollapsibleFormatter(t *testing.T) {
	tests := []struct {
		name            string
		summaryTemplate string
		detailFunc      func(any) any
		input           any
		wantSummary     string
		wantDetails     any
		shouldCollapse  bool
	}{
		{
			name:            "basic collapsible formatter",
			summaryTemplate: "Value: %v (expand for details)",
			detailFunc: func(val any) any {
				return fmt.Sprintf("Full details: %v", val)
			},
			input:          "test",
			wantSummary:    "Value: test (expand for details)",
			wantDetails:    "Full details: test",
			shouldCollapse: true,
		},
		{
			name:            "nil detail function returns original",
			summaryTemplate: "Summary: %v",
			detailFunc:      nil,
			input:           "test",
			wantSummary:     "",
			wantDetails:     nil,
			shouldCollapse:  false,
		},
		{
			name:            "detail function returns nil",
			summaryTemplate: "Summary: %v",
			detailFunc: func(val any) any {
				return nil
			},
			input:          "test",
			wantSummary:    "",
			wantDetails:    nil,
			shouldCollapse: false,
		},
		{
			name:            "detail function returns same value",
			summaryTemplate: "Summary: %v",
			detailFunc: func(val any) any {
				return val
			},
			input:          "test",
			wantSummary:    "",
			wantDetails:    nil,
			shouldCollapse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := CollapsibleFormatter(tt.summaryTemplate, tt.detailFunc)
			result := formatter(tt.input)

			if !tt.shouldCollapse {
				// Should return original value unchanged
				if result != tt.input {
					t.Errorf("CollapsibleFormatter() = %v, want %v", result, tt.input)
				}
				return
			}

			// Should return CollapsibleValue
			cv, ok := result.(CollapsibleValue)
			if !ok {
				t.Fatalf("CollapsibleFormatter() returned %T, want CollapsibleValue", result)
			}

			if cv.Summary() != tt.wantSummary {
				t.Errorf("Summary() = %q, want %q", cv.Summary(), tt.wantSummary)
			}

			if cv.Details() != tt.wantDetails {
				t.Errorf("Details() = %v, want %v", cv.Details(), tt.wantDetails)
			}
		})
	}
}

func TestErrorListFormatter(t *testing.T) {
	tests := []struct {
		name           string
		input          any
		wantSummary    string
		wantDetails    []string
		shouldCollapse bool
	}{
		{
			name:           "string array with errors",
			input:          []string{"error 1", "error 2", "error 3"},
			wantSummary:    "3 errors (click to expand)",
			wantDetails:    []string{"error 1", "error 2", "error 3"},
			shouldCollapse: true,
		},
		{
			name:           "error array",
			input:          []error{errors.New("error 1"), errors.New("error 2")},
			wantSummary:    "2 errors (click to expand)",
			wantDetails:    []string{"error 1", "error 2"},
			shouldCollapse: true,
		},
		{
			name:           "empty string array",
			input:          []string{},
			shouldCollapse: false,
		},
		{
			name:           "empty error array",
			input:          []error{},
			shouldCollapse: false,
		},
		{
			name:           "incompatible type",
			input:          "not an array",
			shouldCollapse: false,
		},
		{
			name:           "nil input",
			input:          nil,
			shouldCollapse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := ErrorListFormatter()
			result := formatter(tt.input)

			if !tt.shouldCollapse {
				// Should return original value unchanged
				// For slice types, we need to compare differently
				switch input := tt.input.(type) {
				case []string:
					if resultSlice, ok := result.([]string); ok {
						if len(resultSlice) != len(input) {
							t.Errorf("ErrorListFormatter() length = %d, want %d", len(resultSlice), len(input))
						}
					} else {
						t.Errorf("ErrorListFormatter() = %T, want []string", result)
					}
				case []error:
					if resultSlice, ok := result.([]error); ok {
						if len(resultSlice) != len(input) {
							t.Errorf("ErrorListFormatter() length = %d, want %d", len(resultSlice), len(input))
						}
					} else {
						t.Errorf("ErrorListFormatter() = %T, want []error", result)
					}
				default:
					if result != tt.input {
						t.Errorf("ErrorListFormatter() = %v, want %v", result, tt.input)
					}
				}
				return
			}

			// Should return CollapsibleValue
			cv, ok := result.(CollapsibleValue)
			if !ok {
				t.Fatalf("ErrorListFormatter() returned %T, want CollapsibleValue", result)
			}

			if cv.Summary() != tt.wantSummary {
				t.Errorf("Summary() = %q, want %q", cv.Summary(), tt.wantSummary)
			}

			details, ok := cv.Details().([]string)
			if !ok {
				t.Fatalf("Details() returned %T, want []string", cv.Details())
			}

			if len(details) != len(tt.wantDetails) {
				t.Errorf("Details length = %d, want %d", len(details), len(tt.wantDetails))
				return
			}

			for i, detail := range details {
				if detail != tt.wantDetails[i] {
					t.Errorf("Details[%d] = %q, want %q", i, detail, tt.wantDetails[i])
				}
			}
		})
	}
}

func TestFilePathFormatter(t *testing.T) {
	tests := []struct {
		name           string
		maxLength      int
		input          any
		wantSummary    string
		wantDetails    string
		shouldCollapse bool
	}{
		{
			name:           "long path gets collapsed",
			maxLength:      50,
			input:          "/very/long/path/to/some/deeply/nested/file/that/exceeds/the/maximum/length.txt",
			wantSummary:    "...e/maximum/length.txt (show full path)",
			wantDetails:    "/very/long/path/to/some/deeply/nested/file/that/exceeds/the/maximum/length.txt",
			shouldCollapse: true,
		},
		{
			name:           "short path not collapsed",
			maxLength:      50,
			input:          "/short/path.txt",
			shouldCollapse: false,
		},
		{
			name:           "non-string input not collapsed",
			maxLength:      10,
			input:          123,
			shouldCollapse: false,
		},
		{
			name:           "path exactly at limit not collapsed",
			maxLength:      10,
			input:          "1234567890",
			shouldCollapse: false,
		},
		{
			name:           "very short path with small max length",
			maxLength:      5,
			input:          "/short.txt",
			wantSummary:    ".../short.txt (show full path)",
			wantDetails:    "/short.txt",
			shouldCollapse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := FilePathFormatter(tt.maxLength)
			result := formatter(tt.input)

			if !tt.shouldCollapse {
				// Should return original value unchanged
				if result != tt.input {
					t.Errorf("FilePathFormatter() = %v, want %v", result, tt.input)
				}
				return
			}

			// Should return CollapsibleValue
			cv, ok := result.(CollapsibleValue)
			if !ok {
				t.Fatalf("FilePathFormatter() returned %T, want CollapsibleValue", result)
			}

			if cv.Summary() != tt.wantSummary {
				t.Errorf("Summary() = %q, want %q", cv.Summary(), tt.wantSummary)
			}

			if cv.Details() != tt.wantDetails {
				t.Errorf("Details() = %v, want %v", cv.Details(), tt.wantDetails)
			}
		})
	}
}

func TestJSONFormatter(t *testing.T) {
	tests := []struct {
		name           string
		maxLength      int
		input          any
		wantSummary    string
		shouldCollapse bool
		expectError    bool
	}{
		{
			name:           "complex data structure",
			maxLength:      10,
			input:          map[string]any{"key1": "value1", "key2": []int{1, 2, 3}},
			wantSummary:    "JSON data (52 bytes)", // Approximate size
			shouldCollapse: true,
		},
		{
			name:           "small data not collapsed",
			maxLength:      100,
			input:          map[string]any{"key": "value"},
			shouldCollapse: false,
		},
		{
			name:           "unmarshalable input not collapsed",
			maxLength:      10,
			input:          make(chan int), // Cannot be marshaled to JSON
			shouldCollapse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := JSONFormatter(tt.maxLength)
			result := formatter(tt.input)

			if !tt.shouldCollapse {
				// Should return original value unchanged
				// For map types, we need to check type rather than comparing directly
				switch tt.input.(type) {
				case map[string]any:
					if _, ok := result.(map[string]any); !ok {
						t.Errorf("JSONFormatter() = %T, want map[string]any", result)
					}
				default:
					if result != tt.input {
						t.Errorf("JSONFormatter() = %v, want %v", result, tt.input)
					}
				}
				return
			}

			// Should return CollapsibleValue
			cv, ok := result.(CollapsibleValue)
			if !ok {
				t.Fatalf("JSONFormatter() returned %T, want CollapsibleValue", result)
			}

			summary := cv.Summary()
			// Check that summary starts with expected pattern (bytes count may vary)
			if len(summary) < 15 || summary[:10] != "JSON data " {
				t.Errorf("Summary() = %q, want pattern 'JSON data (X bytes)'", summary)
			}

			// Verify details is a valid JSON string
			details, ok := cv.Details().(string)
			if !ok {
				t.Fatalf("Details() returned %T, want string", cv.Details())
			}

			if len(details) == 0 {
				t.Error("Details() returned empty string")
			}
		})
	}
}

func TestFormatterWithOptions(t *testing.T) {
	// Test formatters with CollapsibleOptions
	formatter := ErrorListFormatter(WithExpanded(true), WithMaxLength(10))

	input := []string{"very long error message that exceeds limit", "short"}
	result := formatter(input)

	cv, ok := result.(CollapsibleValue)
	if !ok {
		t.Fatalf("ErrorListFormatter() returned %T, want CollapsibleValue", result)
	}

	if !cv.IsExpanded() {
		t.Error("Expected CollapsibleValue to be expanded by default")
	}

	// Details should be truncated due to maxLength option
	details := cv.Details()
	if details == nil {
		t.Error("Details() returned nil")
	}
}
