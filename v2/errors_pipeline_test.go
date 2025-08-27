package output

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestPipelineError(t *testing.T) {
	tests := map[string]struct {
		operation         string
		stage             int
		input             any
		context           map[string]any
		pipelineContext   map[string]any
		cause             error
		expectParts       []string
		expectNotContains []string
	}{"aggregation error with type mismatch":

	// No stage

	{

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
	}, "filter operation error": {

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
	}, "minimal pipeline error": {

		stage: -1,
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
	}, "sort operation error": {

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
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
func TestPipelineErrorTypeMismatch(t *testing.T) {
	tests := map[string]struct {
		operation    string
		expectedType string
		actualType   string
		field        string
		value        any
	}{"filter requires table content": {

		operation:    "Filter",
		expectedType: "table",
		actualType:   "text",
		field:        "content_type",
		value:        "text",
	}, "groupBy requires table content": {

		operation:    "GroupBy",
		expectedType: "table",
		actualType:   "section",
		field:        "content_type",
		value:        "section",
	}, "numeric comparison with string value": {

		operation:    "Sort",
		expectedType: "numeric",
		actualType:   "string",
		field:        "field_value",
		value:        "not-a-number",
	}, "sort requires table content": {

		operation:    "Sort",
		expectedType: "table",
		actualType:   "raw",
		field:        "content_type",
		value:        "raw",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
