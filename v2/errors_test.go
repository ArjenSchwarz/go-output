package output

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRenderError(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		content  Content
		cause    error
		expected string
	}{
		{
			name:     "with table content",
			format:   "json",
			content:  &TableContent{id: "test-123"},
			cause:    errors.New("serialization failed"),
			expected: "render json for table content test-123: serialization failed",
		},
		{
			name:     "with text content",
			format:   "html",
			content:  &TextContent{id: "text-456"},
			cause:    errors.New("encoding error"),
			expected: "render html for text content text-456: encoding error",
		},
		{
			name:     "with nil content",
			format:   "csv",
			content:  nil,
			cause:    errors.New("content missing"),
			expected: "render csv for unknown content unknown: content missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewRenderError(tt.format, tt.content, tt.cause)

			if err.Error() != tt.expected {
				t.Errorf("RenderError.Error() = %q, want %q", err.Error(), tt.expected)
			}

			if err.Format != tt.format {
				t.Errorf("RenderError.Format = %q, want %q", err.Format, tt.format)
			}

			if !errors.Is(err, tt.cause) {
				t.Errorf("RenderError should wrap the cause error")
			}
		})
	}
}

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

func TestContextError(t *testing.T) {
	cause := errors.New("underlying error")

	t.Run("basic context error", func(t *testing.T) {
		err := NewContextError("render", cause)
		expected := `operation "render" failed; cause: underlying error`

		if err.Error() != expected {
			t.Errorf("ContextError.Error() = %q, want %q", err.Error(), expected)
		}

		if !errors.Is(err, cause) {
			t.Errorf("ContextError should wrap the cause error")
		}
	})

	t.Run("context error with context", func(t *testing.T) {
		err := NewContextError("transform", cause)
		err.AddContext("format", "json")
		err.AddContext("transformer", "emoji")

		result := err.Error()
		expectedParts := []string{
			`operation "transform" failed`,
			"context:",
			"format=json",
			"transformer=emoji",
			"cause: underlying error",
		}

		for _, part := range expectedParts {
			if !strings.Contains(result, part) {
				t.Errorf("ContextError.Error() = %q, should contain %q", result, part)
			}
		}
	})

	t.Run("ErrorWithContext helper", func(t *testing.T) {
		err := ErrorWithContext("write", cause, "destination", "file", "format", "csv")

		result := err.Error()
		expectedParts := []string{
			`operation "write" failed`,
			"context:",
			"destination=file",
			"format=csv",
			"cause: underlying error",
		}

		for _, part := range expectedParts {
			if !strings.Contains(result, part) {
				t.Errorf("ErrorWithContext result = %q, should contain %q", result, part)
			}
		}
	})
}

func TestMultiError(t *testing.T) {
	t.Run("empty multi-error", func(t *testing.T) {
		err := NewMultiError("test")

		if err.HasErrors() {
			t.Errorf("Empty MultiError should not have errors")
		}

		if err.ErrorOrNil() != nil {
			t.Errorf("Empty MultiError.ErrorOrNil() should return nil")
		}
	})

	t.Run("single error", func(t *testing.T) {
		err := NewMultiError("test")
		err.Add(errors.New("single error"))

		if !err.HasErrors() {
			t.Errorf("MultiError should have errors")
		}

		expected := "test: single error"
		if err.Error() != expected {
			t.Errorf("MultiError.Error() = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		err := NewMultiError("validation")
		err.Add(errors.New("error 1"))
		err.Add(errors.New("error 2"))
		err.Add(errors.New("error 3"))

		result := err.Error()
		expected := "validation failed with 3 errors:"

		if !strings.HasPrefix(result, expected) {
			t.Errorf("MultiError.Error() should start with %q, got %q", expected, result)
		}

		if !strings.Contains(result, "1. error 1") {
			t.Errorf("MultiError.Error() should contain numbered errors")
		}

		if !strings.Contains(result, "2. error 2") {
			t.Errorf("MultiError.Error() should contain numbered errors")
		}

		if !strings.Contains(result, "3. error 3") {
			t.Errorf("MultiError.Error() should contain numbered errors")
		}
	})

	t.Run("nil errors are ignored", func(t *testing.T) {
		err := NewMultiError("test")
		err.Add(nil)
		err.Add(errors.New("real error"))
		err.Add(nil)

		if len(err.Errors) != 1 {
			t.Errorf("MultiError should only contain non-nil errors, got %d", len(err.Errors))
		}
	})
}

func TestCancelledError(t *testing.T) {
	t.Run("context.Canceled", func(t *testing.T) {
		err := NewCancelledError("render", context.Canceled)
		expected := `operation "render" was cancelled: context canceled`

		if err.Error() != expected {
			t.Errorf("CancelledError.Error() = %q, want %q", err.Error(), expected)
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("CancelledError should wrap context.Canceled")
		}
	})

	t.Run("context.DeadlineExceeded", func(t *testing.T) {
		err := NewCancelledError("transform", context.DeadlineExceeded)
		expected := `operation "transform" was cancelled: context deadline exceeded`

		if err.Error() != expected {
			t.Errorf("CancelledError.Error() = %q, want %q", err.Error(), expected)
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("CancelledError should wrap context.DeadlineExceeded")
		}
	})
}

func TestIsCancelled(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context.Canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "context.DeadlineExceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "CancelledError",
			err:      NewCancelledError("test", context.Canceled),
			expected: true,
		},
		{
			name:     "wrapped CancelledError",
			err:      fmt.Errorf("wrapper: %w", NewCancelledError("test", context.Canceled)),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCancelled(tt.err)
			if result != tt.expected {
				t.Errorf("IsCancelled(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestAsError(t *testing.T) {
	t.Run("direct type match", func(t *testing.T) {
		originalErr := NewValidationError("test", "value", "message")

		var validationErr *ValidationError
		if !AsError(originalErr, &validationErr) {
			t.Errorf("AsError should match ValidationError directly")
		}

		if validationErr != originalErr {
			t.Errorf("AsError should return the same instance")
		}
	})

	t.Run("wrapped error", func(t *testing.T) {
		originalErr := NewRenderError("json", nil, errors.New("cause"))
		wrappedErr := fmt.Errorf("wrapper: %w", originalErr)

		var renderErr *RenderError
		if !AsError(wrappedErr, &renderErr) {
			t.Errorf("AsError should match wrapped RenderError")
		}

		if renderErr != originalErr {
			t.Errorf("AsError should return the original error instance")
		}
	})

	t.Run("no match", func(t *testing.T) {
		err := errors.New("regular error")

		var validationErr *ValidationError
		if AsError(err, &validationErr) {
			t.Errorf("AsError should not match different error types")
		}

		if validationErr != nil {
			t.Errorf("AsError should not modify target when no match")
		}
	})
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

func TestErrorHelpers(t *testing.T) {
	t.Run("FailFast", func(t *testing.T) {
		err1 := errors.New("first error")
		err2 := errors.New("second error")

		result := FailFast(nil, nil, err1, err2)
		if result != err1 {
			t.Errorf("FailFast should return first non-nil error")
		}

		result = FailFast(nil, nil, nil)
		if result != nil {
			t.Errorf("FailFast should return nil when all errors are nil")
		}
	})

	t.Run("CollectErrors", func(t *testing.T) {
		err1 := errors.New("first error")
		err2 := errors.New("second error")

		result := CollectErrors("test", nil, err1, nil, err2)
		if result == nil {
			t.Errorf("CollectErrors should return non-nil when there are errors")
		}

		var multiErr *MultiError
		if !AsError(result, &multiErr) {
			t.Errorf("CollectErrors should return MultiError")
		}

		if len(multiErr.Errors) != 2 {
			t.Errorf("CollectErrors should collect only non-nil errors, got %d", len(multiErr.Errors))
		}

		result = CollectErrors("test", nil, nil)
		if result != nil {
			t.Errorf("CollectErrors should return nil when all errors are nil")
		}
	})
}
