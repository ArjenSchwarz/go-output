package output

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

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
	tests := map[string]struct {
		err      error
		expected bool
	}{"CancelledError": {

		err:      NewCancelledError("test", context.Canceled),
		expected: true,
	}, "context.Canceled": {

		err:      context.Canceled,
		expected: true,
	}, "context.DeadlineExceeded": {

		err:      context.DeadlineExceeded,
		expected: true,
	}, "nil error": {

		err:      nil,
		expected: false,
	}, "regular error": {

		err:      errors.New("regular error"),
		expected: false,
	}, "wrapped CancelledError": {

		err:      fmt.Errorf("wrapper: %w", NewCancelledError("test", context.Canceled)),
		expected: true,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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

// TestEnhancedRenderError tests the enhanced RenderError with detailed context
func TestMultiErrorWithSourceTracking(t *testing.T) {
	multiErr := NewMultiError("render")
	multiErr.AddContext("total_formats", 3)
	multiErr.AddContext("document_contents", 5)

	// Add errors with source tracking
	renderErr := NewRenderError("json", &TableContent{id: "table-1"}, errors.New("serialization failed"))
	multiErr.AddWithSource(renderErr, "renderer", map[string]any{
		"format":   "json",
		"renderer": "JSONRenderer",
	})

	writerErr := NewWriterError("FileWriter", "html", errors.New("permission denied"))
	multiErr.AddWithSource(writerErr, "writer", map[string]any{
		"format": "html",
		"writer": "FileWriter",
	})

	// Test error message includes source information
	errorStr := multiErr.Error()

	// Should contain context information
	if !strings.Contains(errorStr, "total_formats=3") || !strings.Contains(errorStr, "document_contents=5") {
		t.Errorf("MultiError should include context information")
	}

	// Should contain source information for each error
	if !strings.Contains(errorStr, "component=renderer") || !strings.Contains(errorStr, "format=json") || !strings.Contains(errorStr, "renderer=JSONRenderer") {
		t.Errorf("MultiError should include source information for renderer error")
	}

	if !strings.Contains(errorStr, "component=writer") || !strings.Contains(errorStr, "format=html") || !strings.Contains(errorStr, "writer=FileWriter") {
		t.Errorf("MultiError should include source information for writer error")
	}

	// Test error count
	if len(multiErr.Errors) != 2 {
		t.Errorf("MultiError should have 2 errors, got %d", len(multiErr.Errors))
	}

	// Test source map
	if len(multiErr.SourceMap) != 2 {
		t.Errorf("MultiError should have 2 source mappings, got %d", len(multiErr.SourceMap))
	}
}

// TestStructuredError tests the new StructuredError type
func TestStructuredError(t *testing.T) {
	tests := map[string]struct {
		code        string
		component   string
		operation   string
		message     string
		context     map[string]any
		details     map[string]any
		cause       error
		expectParts []string
	}{"complete structured error": {

		code:      "RENDER_001",
		component: "renderer",
		operation: "encode",
		message:   "JSON encoding failed",
		context:   map[string]any{"format": "json", "content_type": "table"},
		details:   map[string]any{"input_size": 1024},
		cause:     errors.New("invalid character"),
		expectParts: []string{
			"code=RENDER_001",
			"component=renderer",
			"operation=encode",
			"message=JSON encoding failed",
			"context=[content_type=table, format=json]", // Alphabetically sorted keys
			"cause: invalid character",
		},
	}, "minimal structured error": {

		code:      "GENERIC_001",
		component: "system",
		operation: "process",
		message:   "Operation failed",
		expectParts: []string{
			"code=GENERIC_001",
			"component=system",
			"operation=process",
			"message=Operation failed",
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var err *StructuredError
			if tt.cause != nil {
				err = NewStructuredErrorWithCause(tt.code, tt.component, tt.operation, tt.message, tt.cause)
			} else {
				err = NewStructuredError(tt.code, tt.component, tt.operation, tt.message)
			}

			// Add context and details
			for k, v := range tt.context {
				err.AddContext(k, v)
			}
			for k, v := range tt.details {
				err.AddDetail(k, v)
			}

			errorStr := err.Error()
			for _, part := range tt.expectParts {
				if !strings.Contains(errorStr, part) {
					t.Errorf("StructuredError.Error() should contain %q, got: %s", part, errorStr)
				}
			}

			// Test fields
			if err.Code != tt.code {
				t.Errorf("StructuredError.Code = %q, want %q", err.Code, tt.code)
			}

			if err.Component != tt.component {
				t.Errorf("StructuredError.Component = %q, want %q", err.Component, tt.component)
			}

			// Test unwrapping
			if tt.cause != nil && !errors.Is(err, tt.cause) {
				t.Errorf("StructuredError should wrap the cause error")
			}
		})
	}
}

// TestToStructuredError tests conversion of various error types to StructuredError
func TestToStructuredError(t *testing.T) {
	tests := map[string]struct {
		inputError     error
		defaultCode    string
		defaultComp    string
		defaultOp      string
		expectedCode   string
		expectedComp   string
		expectedFields map[string]any
	}{"nil error": {

		inputError:   nil,
		defaultCode:  "DEFAULT",
		defaultComp:  "default",
		defaultOp:    "default",
		expectedCode: "",
		expectedComp: "",
	}, "render error conversion": {

		inputError:   NewRenderError("json", &TableContent{id: "test"}, errors.New("failed")),
		defaultCode:  "DEFAULT",
		defaultComp:  "default",
		defaultOp:    "default",
		expectedCode: "RENDER_ERROR",
		expectedComp: "renderer",
		expectedFields: map[string]any{
			"format":       "json",
			"content_type": "table",
			"content_id":   "test",
		},
	}, "transform error conversion": {

		inputError:   NewTransformError("emoji", "html", []byte("test"), errors.New("failed")),
		defaultCode:  "DEFAULT",
		defaultComp:  "default",
		defaultOp:    "default",
		expectedCode: "TRANSFORM_ERROR",
		expectedComp: "transformer",
		expectedFields: map[string]any{
			"transformer": "emoji",
			"format":      "html",
			"input_size":  4,
		},
	}, "unknown error conversion": {

		inputError:   errors.New("unknown error"),
		defaultCode:  "UNKNOWN_001",
		defaultComp:  "system",
		defaultOp:    "process",
		expectedCode: "UNKNOWN_001",
		expectedComp: "system",
	}, "writer error conversion": {

		inputError:   NewWriterError("FileWriter", "csv", errors.New("failed")),
		defaultCode:  "DEFAULT",
		defaultComp:  "default",
		defaultOp:    "default",
		expectedCode: "WRITER_ERROR",
		expectedComp: "writer",
		expectedFields: map[string]any{
			"writer": "FileWriter",
			"format": "csv",
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := ToStructuredError(tt.inputError, tt.defaultCode, tt.defaultComp, tt.defaultOp)

			if tt.inputError == nil {
				if result != nil {
					t.Errorf("ToStructuredError(nil) should return nil")
				}
				return
			}

			if result == nil {
				t.Fatalf("ToStructuredError() should not return nil for non-nil input")
			}

			if result.Code != tt.expectedCode {
				t.Errorf("StructuredError.Code = %q, want %q", result.Code, tt.expectedCode)
			}

			if result.Component != tt.expectedComp {
				t.Errorf("StructuredError.Component = %q, want %q", result.Component, tt.expectedComp)
			}

			// Check expected fields in context
			for key, expectedValue := range tt.expectedFields {
				if value, exists := result.Context[key]; !exists || value != expectedValue {
					t.Errorf("StructuredError.Context[%q] = %v, want %v", key, value, expectedValue)
				}
			}

			// Test that it wraps the original error
			if !errors.Is(result, tt.inputError) {
				t.Errorf("StructuredError should wrap the original error")
			}
		})
	}
}

// TestPipelineError tests the new PipelineError type for pipeline operation failures
