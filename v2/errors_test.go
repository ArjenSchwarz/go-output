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
			expected: "render failed; format=json; content_type=table; content_id=test-123; cause: serialization failed",
		},
		{
			name:     "with text content",
			format:   "html",
			content:  &TextContent{id: "text-456"},
			cause:    errors.New("encoding error"),
			expected: "render failed; format=html; content_type=text; content_id=text-456; cause: encoding error",
		},
		{
			name:     "with nil content",
			format:   "csv",
			content:  nil,
			cause:    errors.New("content missing"),
			expected: "render failed; format=csv; cause: content missing",
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

// TestEnhancedRenderError tests the enhanced RenderError with detailed context
func TestEnhancedRenderError(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		renderer    string
		operation   string
		content     Content
		context     map[string]any
		cause       error
		expectParts []string
	}{
		{
			name:      "detailed render error",
			format:    "json",
			renderer:  "JSONRenderer",
			operation: "encode",
			content:   &TableContent{id: "test-123"},
			context:   map[string]any{"data_size": 1024, "encoding": "utf-8"},
			cause:     errors.New("json encoding failed"),
			expectParts: []string{
				"operation \"encode\" failed",
				"format=json",
				"renderer=JSONRenderer",
				"content_type=table",
				"content_id=test-123",
				"data_size=1024",
				"encoding=utf-8",
				"cause: json encoding failed",
			},
		},
		{
			name:    "minimal render error",
			format:  "csv",
			content: nil,
			cause:   errors.New("content missing"),
			expectParts: []string{
				"render failed",
				"format=csv",
				"cause: content missing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err *RenderError
			if tt.renderer != "" && tt.operation != "" {
				err = NewRenderErrorWithDetails(tt.format, tt.renderer, tt.operation, tt.content, tt.cause)
			} else {
				err = NewRenderError(tt.format, tt.content, tt.cause)
			}

			// Add context if provided
			for k, v := range tt.context {
				err.AddContext(k, v)
			}

			errorStr := err.Error()
			for _, part := range tt.expectParts {
				if !strings.Contains(errorStr, part) {
					t.Errorf("RenderError.Error() should contain %q, got: %s", part, errorStr)
				}
			}

			// Test unwrapping
			if !errors.Is(err, tt.cause) {
				t.Errorf("RenderError should wrap the cause error")
			}
		})
	}
}

// TestWriterError tests the new WriterError type
func TestWriterError(t *testing.T) {
	tests := []struct {
		name        string
		writer      string
		format      string
		operation   string
		context     map[string]any
		cause       error
		expectParts []string
	}{
		{
			name:      "detailed writer error",
			writer:    "FileWriter",
			format:    "html",
			operation: "write",
			context:   map[string]any{"file_path": "/tmp/output.html", "data_size": 2048},
			cause:     errors.New("permission denied"),
			expectParts: []string{
				"operation \"write\" failed",
				"format=html",
				"writer=FileWriter",
				"file_path=/tmp/output.html",
				"data_size=2048",
				"cause: permission denied",
			},
		},
		{
			name:   "minimal writer error",
			writer: "S3Writer",
			format: "json",
			cause:  errors.New("network timeout"),
			expectParts: []string{
				"write failed",
				"format=json",
				"writer=S3Writer",
				"cause: network timeout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err *WriterError
			if tt.operation != "" {
				err = NewWriterErrorWithDetails(tt.writer, tt.format, tt.operation, tt.cause)
			} else {
				err = NewWriterError(tt.writer, tt.format, tt.cause)
			}

			// Add context if provided
			for k, v := range tt.context {
				err.AddContext(k, v)
			}

			errorStr := err.Error()
			for _, part := range tt.expectParts {
				if !strings.Contains(errorStr, part) {
					t.Errorf("WriterError.Error() should contain %q, got: %s", part, errorStr)
				}
			}

			// Test unwrapping
			if !errors.Is(err, tt.cause) {
				t.Errorf("WriterError should wrap the cause error")
			}
		})
	}
}

// TestMultiErrorWithSourceTracking tests enhanced MultiError with source tracking
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
	tests := []struct {
		name        string
		code        string
		component   string
		operation   string
		message     string
		context     map[string]any
		details     map[string]any
		cause       error
		expectParts []string
	}{
		{
			name:      "complete structured error",
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
		},
		{
			name:      "minimal structured error",
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	tests := []struct {
		name           string
		inputError     error
		defaultCode    string
		defaultComp    string
		defaultOp      string
		expectedCode   string
		expectedComp   string
		expectedFields map[string]any
	}{
		{
			name:         "render error conversion",
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
		},
		{
			name:         "transform error conversion",
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
		},
		{
			name:         "writer error conversion",
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
		},
		{
			name:         "unknown error conversion",
			inputError:   errors.New("unknown error"),
			defaultCode:  "UNKNOWN_001",
			defaultComp:  "system",
			defaultOp:    "process",
			expectedCode: "UNKNOWN_001",
			expectedComp: "system",
		},
		{
			name:         "nil error",
			inputError:   nil,
			defaultCode:  "DEFAULT",
			defaultComp:  "default",
			defaultOp:    "default",
			expectedCode: "",
			expectedComp: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
func TestPipelineError(t *testing.T) {
	tests := []struct {
		name              string
		operation         string
		stage             int
		input             any
		context           map[string]any
		pipelineContext   map[string]any
		cause             error
		expectParts       []string
		expectNotContains []string
	}{
		{
			name:      "filter operation error",
			operation: "Filter",
			stage:     0,
			input:     map[string]any{"id": 1, "status": "active"},
			context:   map[string]any{"predicate": "status == 'invalid'", "record_count": 100},
			pipelineContext: map[string]any{
				"total_operations": 3,
				"document_id":      "doc-123",
			},
			cause: errors.New("predicate evaluation failed"),
			expectParts: []string{
				"pipeline operation \"Filter\" failed",
				"stage=0",
				"pipeline_context=[",
				"document_id=doc-123",
				"total_operations=3",
				"operation_context=[",
				"predicate=status == 'invalid'",
				"record_count=100",
				"input_sample=map[id:1 status:active]",
				"cause: predicate evaluation failed",
			},
		},
		{
			name:      "sort operation error",
			operation: "Sort",
			stage:     2,
			context:   map[string]any{"column": "timestamp", "direction": "asc"},
			pipelineContext: map[string]any{
				"total_operations": 5,
			},
			cause: errors.New("column 'timestamp' not found"),
			expectParts: []string{
				"pipeline operation \"Sort\" failed",
				"stage=2",
				"pipeline_context=[total_operations=5]",
				"operation_context=[",
				"column=timestamp",
				"direction=asc",
				"cause: column 'timestamp' not found",
			},
			expectNotContains: []string{
				"input_sample=",
			},
		},
		{
			name:  "minimal pipeline error",
			stage: -1, // No stage
			cause: errors.New("generic pipeline failure"),
			expectParts: []string{
				"pipeline failed",
				"cause: generic pipeline failure",
			},
			expectNotContains: []string{
				"stage=",
				"pipeline_context=",
				"operation_context=",
				"input_sample=",
			},
		},
		{
			name:      "aggregation error with type mismatch",
			operation: "GroupBy",
			stage:     1,
			input:     []map[string]any{{"id": 1, "amount": "not-a-number"}},
			context: map[string]any{
				"aggregate_function": "Sum",
				"field":              "amount",
				"group_columns":      []string{"department"},
			},
			cause: errors.New("cannot sum non-numeric values"),
			expectParts: []string{
				"pipeline operation \"GroupBy\" failed",
				"stage=1",
				"aggregate_function=Sum",
				"field=amount",
				"cause: cannot sum non-numeric values",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err *PipelineError
			if tt.input != nil {
				err = NewPipelineErrorWithInput(tt.operation, tt.stage, tt.input, tt.cause)
			} else {
				err = NewPipelineError(tt.operation, tt.stage, tt.cause)
			}

			// Add context if provided
			for k, v := range tt.context {
				err.AddContext(k, v)
			}
			for k, v := range tt.pipelineContext {
				err.AddPipelineContext(k, v)
			}

			errorStr := err.Error()

			// Check expected parts are present
			for _, part := range tt.expectParts {
				if !strings.Contains(errorStr, part) {
					t.Errorf("PipelineError.Error() should contain %q, got: %s", part, errorStr)
				}
			}

			// Check parts that should not be present
			for _, part := range tt.expectNotContains {
				if strings.Contains(errorStr, part) {
					t.Errorf("PipelineError.Error() should not contain %q, got: %s", part, errorStr)
				}
			}

			// Test unwrapping
			if !errors.Is(err, tt.cause) {
				t.Errorf("PipelineError should wrap the cause error")
			}

			// Test field access
			if err.Operation != tt.operation {
				t.Errorf("PipelineError.Operation = %q, want %q", err.Operation, tt.operation)
			}

			if err.Stage != tt.stage {
				t.Errorf("PipelineError.Stage = %d, want %d", err.Stage, tt.stage)
			}
		})
	}
}

// TestPipelineErrorContextMethods tests context manipulation methods
func TestPipelineErrorContextMethods(t *testing.T) {
	cause := errors.New("test error")
	err := NewPipelineError("TestOp", 1, cause)

	t.Run("AddContext", func(t *testing.T) {
		result := err.AddContext("key1", "value1").AddContext("key2", 42)

		if result != err {
			t.Errorf("AddContext should return the same instance for chaining")
		}

		if err.Context["key1"] != "value1" {
			t.Errorf("AddContext should set key1 to value1")
		}

		if err.Context["key2"] != 42 {
			t.Errorf("AddContext should set key2 to 42")
		}
	})

	t.Run("AddPipelineContext", func(t *testing.T) {
		result := err.AddPipelineContext("pipeline_key", "pipeline_value")

		if result != err {
			t.Errorf("AddPipelineContext should return the same instance for chaining")
		}

		if err.PipelineCtx["pipeline_key"] != "pipeline_value" {
			t.Errorf("AddPipelineContext should set pipeline_key to pipeline_value")
		}
	})
}

// TestFailFastErrorHandling tests fail-fast behavior in error scenarios
func TestFailFastErrorHandling(t *testing.T) {
	t.Run("FailFast with pipeline errors", func(t *testing.T) {
		err1 := NewPipelineError("Filter", 0, errors.New("filter failed"))
		err2 := NewPipelineError("Sort", 1, errors.New("sort failed"))
		err3 := errors.New("generic error")

		result := FailFast(nil, err1, err2, err3)

		// Should return the first non-nil error
		var pipelineErr *PipelineError
		if !AsError(result, &pipelineErr) {
			t.Errorf("FailFast should return the first PipelineError")
		}

		if pipelineErr.Operation != "Filter" {
			t.Errorf("FailFast should return the first error (Filter), got %s", pipelineErr.Operation)
		}
	})

	t.Run("FailFast with all nil errors", func(t *testing.T) {
		result := FailFast(nil, nil, nil)
		if result != nil {
			t.Errorf("FailFast should return nil when all errors are nil")
		}
	})
}

// TestValidationErrorsForPipelineOperations tests validation errors for pipeline operations
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
func TestPipelineErrorTypeMismatch(t *testing.T) {
	tests := []struct {
		name         string
		operation    string
		expectedType string
		actualType   string
		field        string
		value        any
	}{
		{
			name:         "filter requires table content",
			operation:    "Filter",
			expectedType: "table",
			actualType:   "text",
			field:        "content_type",
			value:        "text",
		},
		{
			name:         "sort requires table content",
			operation:    "Sort",
			expectedType: "table",
			actualType:   "raw",
			field:        "content_type",
			value:        "raw",
		},
		{
			name:         "groupBy requires table content",
			operation:    "GroupBy",
			expectedType: "table",
			actualType:   "section",
			field:        "content_type",
			value:        "section",
		},
		{
			name:         "numeric comparison with string value",
			operation:    "Sort",
			expectedType: "numeric",
			actualType:   "string",
			field:        "field_value",
			value:        "not-a-number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a type mismatch error
			message := fmt.Sprintf("%s requires %s type, got %s", tt.operation, tt.expectedType, tt.actualType)
			validationErr := NewValidationError(tt.field, tt.value, message)

			// Wrap in pipeline error
			pipelineErr := NewPipelineError(tt.operation, 0, validationErr)
			pipelineErr.AddContext("expected_type", tt.expectedType)
			pipelineErr.AddContext("actual_type", tt.actualType)

			errorStr := pipelineErr.Error()

			// Should contain operation info
			if !strings.Contains(errorStr, tt.operation) {
				t.Errorf("Error should contain operation %q", tt.operation)
			}

			// Should contain type information
			if !strings.Contains(errorStr, tt.expectedType) {
				t.Errorf("Error should contain expected type %q", tt.expectedType)
			}

			if !strings.Contains(errorStr, tt.actualType) {
				t.Errorf("Error should contain actual type %q", tt.actualType)
			}

			// Should be able to unwrap to validation error
			var unwrappedValidation *ValidationError
			if !AsError(pipelineErr, &unwrappedValidation) {
				t.Errorf("PipelineError should wrap ValidationError")
			}

			if unwrappedValidation.Field != tt.field {
				t.Errorf("Unwrapped ValidationError.Field = %q, want %q",
					unwrappedValidation.Field, tt.field)
			}
		})
	}
}

// TestToStructuredErrorPipeline tests conversion of PipelineError to StructuredError
func TestToStructuredErrorPipeline(t *testing.T) {
	originalErr := errors.New("operation failed")
	pipelineErr := NewPipelineErrorWithInput("Filter", 2,
		map[string]any{"id": 123}, originalErr)
	pipelineErr.AddContext("predicate", "status == 'active'")
	pipelineErr.AddContext("record_count", 50)
	pipelineErr.AddPipelineContext("total_operations", 5)
	pipelineErr.AddPipelineContext("document_id", "doc-456")

	structuredErr := ToStructuredError(pipelineErr, "DEFAULT", "default", "default")

	// Check that it's converted to structured error
	if structuredErr == nil {
		t.Fatalf("ToStructuredError should not return nil")
	}

	// Check code and component
	if structuredErr.Code != "PIPELINE_ERROR" {
		t.Errorf("StructuredError.Code = %q, want PIPELINE_ERROR", structuredErr.Code)
	}

	if structuredErr.Component != "pipeline" {
		t.Errorf("StructuredError.Component = %q, want pipeline", structuredErr.Component)
	}

	// Check context fields
	expectedContext := map[string]any{
		"operation":    "Filter",
		"stage":        2,
		"predicate":    "status == 'active'",
		"record_count": 50,
	}

	for key, expectedValue := range expectedContext {
		if value, exists := structuredErr.Context[key]; !exists || value != expectedValue {
			t.Errorf("StructuredError.Context[%q] = %v, want %v", key, value, expectedValue)
		}
	}

	// Check details fields (pipeline context goes to details)
	expectedDetails := map[string]any{
		"total_operations": 5,
		"document_id":      "doc-456",
		"input_sample":     map[string]any{"id": 123},
	}

	for key, expectedValue := range expectedDetails {
		if value, exists := structuredErr.Details[key]; !exists {
			t.Errorf("StructuredError.Details should contain key %q", key)
		} else {
			// Special handling for input_sample which is a complex type
			if key == "input_sample" {
				continue // Skip detailed comparison for complex input sample
			}
			if value != expectedValue {
				t.Errorf("StructuredError.Details[%q] = %v, want %v", key, value, expectedValue)
			}
		}
	}

	// Check that it wraps the original error
	if !errors.Is(structuredErr, originalErr) {
		t.Errorf("StructuredError should wrap the original error")
	}
}
