package output

import (
	"errors"
	"strings"
	"testing"
)

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    any
		message  string
		cause    error
		expected string
	}{
		{
			name:     "simple validation error",
			field:    "name",
			value:    "",
			message:  "cannot be empty",
			expected: `field "name": value : cannot be empty`,
		},
		{
			name:     "validation error with cause",
			field:    "age",
			value:    -1,
			message:  "must be positive",
			cause:    errors.New("range error"),
			expected: `field "age": value -1: must be positive: range error`,
		},
		{
			name:     "validation error without message",
			field:    "data",
			value:    nil,
			expected: `field "data": value <nil>: validation failed`,
		},
		{
			name:     "validation error without field",
			value:    "invalid",
			message:  "format not supported",
			expected: `value invalid: format not supported`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err *ValidationError
			if tt.cause != nil {
				err = NewValidationErrorWithCause(tt.field, tt.value, tt.message, tt.cause)
			} else {
				err = NewValidationError(tt.field, tt.value, tt.message)
			}

			if err.Error() != tt.expected {
				t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), tt.expected)
			}

			if tt.cause != nil && !errors.Is(err, tt.cause) {
				t.Errorf("ValidationError should wrap the cause error")
			}
		})
	}
}

func TestValidationHelpers(t *testing.T) {
	t.Run("ValidateNonEmpty", func(t *testing.T) {
		err := ValidateNonEmpty("name", "")
		if err == nil {
			t.Errorf("ValidateNonEmpty should fail for empty string")
		}

		var validationErr *ValidationError
		if !AsError(err, &validationErr) {
			t.Errorf("ValidateNonEmpty should return ValidationError")
		}

		if validationErr.Field != "name" || validationErr.Message != "cannot be empty" {
			t.Errorf("ValidateNonEmpty should set correct field and message")
		}

		err = ValidateNonEmpty("name", "valid")
		if err != nil {
			t.Errorf("ValidateNonEmpty should pass for non-empty string")
		}
	})

	t.Run("ValidateNonNil", func(t *testing.T) {
		err := ValidateNonNil("data", nil)
		if err == nil {
			t.Errorf("ValidateNonNil should fail for nil value")
		}

		err = ValidateNonNil("data", "not nil")
		if err != nil {
			t.Errorf("ValidateNonNil should pass for non-nil value")
		}
	})

	t.Run("ValidateSliceNonEmpty", func(t *testing.T) {
		err := ValidateSliceNonEmpty("items", []string{})
		if err == nil {
			t.Errorf("ValidateSliceNonEmpty should fail for empty slice")
		}

		err = ValidateSliceNonEmpty("items", []string{"item"})
		if err != nil {
			t.Errorf("ValidateSliceNonEmpty should pass for non-empty slice")
		}
	})

	t.Run("ValidateMapNonEmpty", func(t *testing.T) {
		err := ValidateMapNonEmpty("data", map[string]int{})
		if err == nil {
			t.Errorf("ValidateMapNonEmpty should fail for empty map")
		}

		err = ValidateMapNonEmpty("data", map[string]int{"key": 1})
		if err != nil {
			t.Errorf("ValidateMapNonEmpty should pass for non-empty map")
		}
	})
}

func TestValidationErrorsForPipelineOperations(t *testing.T) {
	tests := []struct {
		name          string
		field         string
		value         any
		message       string
		expectedField string
		expectedValue any
		expectedMsg   string
	}{
		{
			name:          "invalid predicate function",
			field:         "predicate",
			value:         nil,
			message:       "filter predicate cannot be nil",
			expectedField: "predicate",
			expectedValue: nil,
			expectedMsg:   "filter predicate cannot be nil",
		},
		{
			name:          "negative limit count",
			field:         "count",
			value:         -5,
			message:       "limit count must be non-negative",
			expectedField: "count",
			expectedValue: -5,
			expectedMsg:   "limit count must be non-negative",
		},
		{
			name:          "empty sort keys",
			field:         "sort_keys",
			value:         "[]",
			message:       "sort requires at least one key",
			expectedField: "sort_keys",
			expectedValue: "[]",
			expectedMsg:   "sort requires at least one key",
		},
		{
			name:          "invalid aggregate function",
			field:         "aggregate_function",
			value:         "unknown_function",
			message:       "unsupported aggregate function",
			expectedField: "aggregate_function",
			expectedValue: "unknown_function",
			expectedMsg:   "unsupported aggregate function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.field, tt.value, tt.message)

			if err.Field != tt.expectedField {
				t.Errorf("ValidationError.Field = %q, want %q", err.Field, tt.expectedField)
			}

			if err.Value != tt.expectedValue {
				t.Errorf("ValidationError.Value = %v, want %v", err.Value, tt.expectedValue)
			}

			if err.Message != tt.expectedMsg {
				t.Errorf("ValidationError.Message = %q, want %q", err.Message, tt.expectedMsg)
			}

			// Test error message format
			errorStr := err.Error()
			if !strings.Contains(errorStr, tt.expectedField) {
				t.Errorf("ValidationError.Error() should contain field name %q", tt.expectedField)
			}
			if !strings.Contains(errorStr, tt.expectedMsg) {
				t.Errorf("ValidationError.Error() should contain message %q", tt.expectedMsg)
			}
		})
	}
}

// TestPipelineErrorTypeMismatch tests errors caused by type mismatches in operations
