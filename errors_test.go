package format

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		name string
		code ErrorCode
		want string
	}{
		{"Configuration error", ErrInvalidFormat, "OUT-1001"},
		{"Validation error", ErrMissingColumn, "OUT-2001"},
		{"Processing error", ErrFileWrite, "OUT-3001"},
		{"Runtime error", ErrNetworkTimeout, "OUT-4001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.code) != tt.want {
				t.Errorf("ErrorCode = %v, want %v", string(tt.code), tt.want)
			}
		})
	}
}

func TestErrorSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		want     string
	}{
		{"Fatal", SeverityFatal, "fatal"},
		{"Error", SeverityError, "error"},
		{"Warning", SeverityWarning, "warning"},
		{"Info", SeverityInfo, "info"},
		{"Unknown", ErrorSeverity(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.want {
				t.Errorf("ErrorSeverity.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBaseError_Creation(t *testing.T) {
	err := NewOutputError(ErrInvalidFormat, SeverityError, "test message")

	if err.Code() != ErrInvalidFormat {
		t.Errorf("Expected code %v, got %v", ErrInvalidFormat, err.Code())
	}

	if err.Severity() != SeverityError {
		t.Errorf("Expected severity %v, got %v", SeverityError, err.Severity())
	}

	if !strings.Contains(err.Error(), "test message") {
		t.Errorf("Expected error message to contain 'test message', got: %v", err.Error())
	}

	if !strings.Contains(err.Error(), string(ErrInvalidFormat)) {
		t.Errorf("Expected error message to contain error code, got: %v", err.Error())
	}
}

func TestBaseError_WithContext(t *testing.T) {
	context := ErrorContext{
		Operation: "validation",
		Field:     "OutputFormat",
		Value:     "invalid",
		Index:     0,
		Metadata:  map[string]interface{}{"key": "value"},
	}

	err := NewErrorBuilder(ErrInvalidFormat, "test message").
		WithContext(context).
		Build()

	if err.Context().Operation != "validation" {
		t.Errorf("Expected operation 'validation', got %v", err.Context().Operation)
	}

	if err.Context().Field != "OutputFormat" {
		t.Errorf("Expected field 'OutputFormat', got %v", err.Context().Field)
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "(field: OutputFormat)") {
		t.Errorf("Expected error message to contain field info, got: %v", errorMsg)
	}

	if !strings.Contains(errorMsg, "(operation: validation)") {
		t.Errorf("Expected error message to contain operation info, got: %v", errorMsg)
	}
}

func TestBaseError_WithSuggestions(t *testing.T) {
	suggestions := []string{
		"Use 'json' format instead",
		"Check available formats with --help",
	}

	err := NewErrorBuilder(ErrInvalidFormat, "invalid format").
		WithSuggestions(suggestions...).
		Build()

	if len(err.Suggestions()) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(err.Suggestions()))
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "Suggestions:") {
		t.Errorf("Expected error message to contain suggestions section, got: %v", errorMsg)
	}

	for _, suggestion := range suggestions {
		if !strings.Contains(errorMsg, suggestion) {
			t.Errorf("Expected error message to contain suggestion '%s', got: %v", suggestion, errorMsg)
		}
	}
}

func TestBaseError_WithCause(t *testing.T) {
	originalErr := errors.New("original error")

	err := NewErrorBuilder(ErrFileWrite, "failed to write file").
		WithCause(originalErr).
		Build()

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "Caused by: original error") {
		t.Errorf("Expected error message to contain cause, got: %v", errorMsg)
	}
}

func TestBaseError_Wrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := NewOutputError(ErrFileWrite, SeverityError, "failed to write file")

	wrappedErr := err.Wrap(originalErr)

	errorMsg := wrappedErr.Error()
	if !strings.Contains(errorMsg, "Caused by: original error") {
		t.Errorf("Expected wrapped error message to contain cause, got: %v", errorMsg)
	}
}

func TestValidationError_Creation(t *testing.T) {
	err := NewValidationError(ErrMissingColumn, "missing required column")

	if err.Code() != ErrMissingColumn {
		t.Errorf("Expected code %v, got %v", ErrMissingColumn, err.Code())
	}

	if err.Severity() != SeverityError {
		t.Errorf("Expected severity %v, got %v", SeverityError, err.Severity())
	}

	if len(err.Violations()) != 0 {
		t.Errorf("Expected no violations initially, got %d", len(err.Violations()))
	}

	if err.IsComposite() {
		t.Error("Expected single error, not composite")
	}
}

func TestValidationError_WithViolations(t *testing.T) {
	violations := []Violation{
		{
			Field:      "Name",
			Value:      nil,
			Constraint: "required",
			Message:    "Name is required",
		},
		{
			Field:      "Age",
			Value:      -1,
			Constraint: "positive",
			Message:    "Age must be positive",
		},
	}

	err := NewValidationErrorBuilder(ErrConstraintViolation, "validation failed").
		WithViolations(violations...).
		Build()

	if len(err.Violations()) != 2 {
		t.Errorf("Expected 2 violations, got %d", len(err.Violations()))
	}

	if !err.IsComposite() {
		t.Error("Expected composite error with multiple violations")
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "Validation violations:") {
		t.Errorf("Expected error message to contain violations section, got: %v", errorMsg)
	}

	if !strings.Contains(errorMsg, "Name: Name is required") {
		t.Errorf("Expected error message to contain Name violation, got: %v", errorMsg)
	}

	if !strings.Contains(errorMsg, "Age: Age must be positive") {
		t.Errorf("Expected error message to contain Age violation, got: %v", errorMsg)
	}
}

func TestProcessingError_Creation(t *testing.T) {
	err := NewProcessingError(ErrS3Upload, "failed to upload to S3", true)

	if err.Code() != ErrS3Upload {
		t.Errorf("Expected code %v, got %v", ErrS3Upload, err.Code())
	}

	if !err.Retryable() {
		t.Error("Expected error to be retryable")
	}

	if err.PartialResult() != nil {
		t.Error("Expected no partial result initially")
	}
}

func TestProcessingError_WithPartialResult(t *testing.T) {
	partialData := map[string]interface{}{"processed": 50, "total": 100}

	err := &processingError{
		baseError: baseError{
			code:     ErrMemoryExhausted,
			severity: SeverityError,
			message:  "out of memory",
		},
		retryable:     false,
		partialResult: partialData,
	}

	if err.PartialResult() == nil {
		t.Error("Expected partial result to be set")
	}

	result, ok := err.PartialResult().(map[string]interface{})
	if !ok {
		t.Error("Expected partial result to be a map")
	}

	if result["processed"] != 50 {
		t.Errorf("Expected processed count 50, got %v", result["processed"])
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected bool
	}{
		{"Nil error", nil, true},
		{"Standard error", errors.New("standard error"), false},
		{"OutputError", NewOutputError(ErrInvalidFormat, SeverityError, "test"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.input)

			if tt.expected && result != nil {
				t.Errorf("Expected nil result for nil input, got %v", result)
			}

			if !tt.expected && result == nil {
				t.Error("Expected non-nil result for non-nil input")
			}

			if !tt.expected && result != nil {
				if _, ok := result.(OutputError); !ok {
					t.Error("Expected result to implement OutputError interface")
				}
			}
		})
	}
}

func TestErrorBuilder_FluentInterface(t *testing.T) {
	err := NewErrorBuilder(ErrInvalidFormat, "test message").
		WithSeverity(SeverityWarning).
		WithField("OutputFormat").
		WithOperation("validation").
		WithValue("invalid").
		WithSuggestions("Use 'json' instead").
		Build()

	if err.Code() != ErrInvalidFormat {
		t.Errorf("Expected code %v, got %v", ErrInvalidFormat, err.Code())
	}

	if err.Severity() != SeverityWarning {
		t.Errorf("Expected severity %v, got %v", SeverityWarning, err.Severity())
	}

	if err.Context().Field != "OutputFormat" {
		t.Errorf("Expected field 'OutputFormat', got %v", err.Context().Field)
	}

	if len(err.Suggestions()) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(err.Suggestions()))
	}
}

func TestValidationErrorBuilder_FluentInterface(t *testing.T) {
	err := NewValidationErrorBuilder(ErrConstraintViolation, "validation failed").
		WithViolation("Name", "required", "Name is required", nil).
		WithViolation("Age", "positive", "Age must be positive", -1).
		WithSeverity(SeverityWarning).
		WithSuggestions("Check input data").
		Build()

	if err.Code() != ErrConstraintViolation {
		t.Errorf("Expected code %v, got %v", ErrConstraintViolation, err.Code())
	}

	if err.Severity() != SeverityWarning {
		t.Errorf("Expected severity %v, got %v", SeverityWarning, err.Severity())
	}

	if len(err.Violations()) != 2 {
		t.Errorf("Expected 2 violations, got %d", len(err.Violations()))
	}

	if !err.IsComposite() {
		t.Error("Expected composite error")
	}
}

func TestBaseError_JSONMarshaling(t *testing.T) {
	err := NewErrorBuilder(ErrInvalidFormat, "test message").
		WithField("OutputFormat").
		WithSuggestions("Use 'json' instead").
		Build()

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal error: %v", marshalErr)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(data, &result); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal error: %v", unmarshalErr)
	}

	if result["code"] != string(ErrInvalidFormat) {
		t.Errorf("Expected code %v, got %v", ErrInvalidFormat, result["code"])
	}

	if result["severity"] != "error" {
		t.Errorf("Expected severity 'error', got %v", result["severity"])
	}

	if result["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", result["message"])
	}

	suggestions, ok := result["suggestions"].([]interface{})
	if !ok || len(suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %v", result["suggestions"])
	}
}

func TestErrorContext_JSONMarshaling(t *testing.T) {
	context := ErrorContext{
		Operation: "validation",
		Field:     "OutputFormat",
		Value:     "invalid",
		Index:     0,
		Metadata:  map[string]interface{}{"key": "value"},
	}

	data, err := json.Marshal(context)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorContext: %v", err)
	}

	var result ErrorContext
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal ErrorContext: %v", err)
	}

	if result.Operation != context.Operation {
		t.Errorf("Expected operation %v, got %v", context.Operation, result.Operation)
	}

	if result.Field != context.Field {
		t.Errorf("Expected field %v, got %v", context.Field, result.Field)
	}
}

func TestViolation_JSONMarshaling(t *testing.T) {
	violation := Violation{
		Field:      "Name",
		Value:      nil,
		Constraint: "required",
		Message:    "Name is required",
	}

	data, err := json.Marshal(violation)
	if err != nil {
		t.Fatalf("Failed to marshal Violation: %v", err)
	}

	var result Violation
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal Violation: %v", err)
	}

	if result.Field != violation.Field {
		t.Errorf("Expected field %v, got %v", violation.Field, result.Field)
	}

	if result.Constraint != violation.Constraint {
		t.Errorf("Expected constraint %v, got %v", violation.Constraint, result.Constraint)
	}

	if result.Message != violation.Message {
		t.Errorf("Expected message %v, got %v", violation.Message, result.Message)
	}
}
