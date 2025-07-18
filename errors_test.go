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

func TestErrorMode_String(t *testing.T) {
	tests := []struct {
		name string
		mode ErrorMode
		want string
	}{
		{"Strict mode", ErrorModeStrict, "strict"},
		{"Lenient mode", ErrorModeLenient, "lenient"},
		{"Interactive mode", ErrorModeInteractive, "interactive"},
		{"Unknown mode", ErrorMode(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("ErrorMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDefaultErrorHandler(t *testing.T) {
	handler := NewDefaultErrorHandler()

	if handler.GetMode() != ErrorModeStrict {
		t.Errorf("Expected default mode to be strict, got %v", handler.GetMode())
	}

	if handler.HasErrors() {
		t.Error("Expected no errors initially")
	}

	if len(handler.GetCollectedErrors()) != 0 {
		t.Errorf("Expected no collected errors initially, got %d", len(handler.GetCollectedErrors()))
	}
}

func TestNewErrorHandlerWithMode(t *testing.T) {
	tests := []struct {
		name string
		mode ErrorMode
	}{
		{"Strict mode", ErrorModeStrict},
		{"Lenient mode", ErrorModeLenient},
		{"Interactive mode", ErrorModeInteractive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewErrorHandlerWithMode(tt.mode)

			if handler.GetMode() != tt.mode {
				t.Errorf("Expected mode %v, got %v", tt.mode, handler.GetMode())
			}
		})
	}
}

func TestDefaultErrorHandler_SetMode(t *testing.T) {
	handler := NewDefaultErrorHandler()

	// Test changing modes
	handler.SetMode(ErrorModeLenient)
	if handler.GetMode() != ErrorModeLenient {
		t.Errorf("Expected mode to be lenient, got %v", handler.GetMode())
	}

	handler.SetMode(ErrorModeInteractive)
	if handler.GetMode() != ErrorModeInteractive {
		t.Errorf("Expected mode to be interactive, got %v", handler.GetMode())
	}
}

func TestDefaultErrorHandler_HandleError_Nil(t *testing.T) {
	handler := NewDefaultErrorHandler()

	err := handler.HandleError(nil)
	if err != nil {
		t.Errorf("Expected nil error to return nil, got %v", err)
	}
}

func TestDefaultErrorHandler_StrictMode(t *testing.T) {
	handler := NewErrorHandlerWithMode(ErrorModeStrict)

	tests := []struct {
		name        string
		inputError  error
		expectError bool
	}{
		{
			"Info severity - should not fail",
			NewErrorBuilder(ErrInvalidFormat, "info message").WithSeverity(SeverityInfo).Build(),
			false,
		},
		{
			"Warning severity - should not fail",
			NewErrorBuilder(ErrInvalidFormat, "warning message").WithSeverity(SeverityWarning).Build(),
			false,
		},
		{
			"Error severity - should fail",
			NewErrorBuilder(ErrInvalidFormat, "error message").WithSeverity(SeverityError).Build(),
			true,
		},
		{
			"Fatal severity - should fail",
			NewErrorBuilder(ErrInvalidFormat, "fatal message").WithSeverity(SeverityFatal).Build(),
			true,
		},
		{
			"Standard error - should fail",
			errors.New("standard error"),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.HandleError(tt.inputError)

			if tt.expectError && err == nil {
				t.Error("Expected error to be returned, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error to be returned, got %v", err)
			}

			// Strict mode should not collect errors
			if handler.HasErrors() {
				t.Error("Expected strict mode to not collect errors")
			}
		})
	}
}

func TestDefaultErrorHandler_LenientMode(t *testing.T) {
	handler := NewErrorHandlerWithMode(ErrorModeLenient)

	tests := []struct {
		name        string
		inputError  error
		expectError bool
	}{
		{
			"Info severity - should not fail but collect",
			NewErrorBuilder(ErrInvalidFormat, "info message").WithSeverity(SeverityInfo).Build(),
			false,
		},
		{
			"Warning severity - should not fail but collect",
			NewErrorBuilder(ErrInvalidFormat, "warning message").WithSeverity(SeverityWarning).Build(),
			false,
		},
		{
			"Error severity - should not fail but collect",
			NewErrorBuilder(ErrInvalidFormat, "error message").WithSeverity(SeverityError).Build(),
			false,
		},
		{
			"Fatal severity - should fail and collect",
			NewErrorBuilder(ErrInvalidFormat, "fatal message").WithSeverity(SeverityFatal).Build(),
			true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.HandleError(tt.inputError)

			if tt.expectError && err == nil {
				t.Error("Expected error to be returned, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error to be returned, got %v", err)
			}

			// Lenient mode should collect all errors
			expectedCount := i + 1
			if len(handler.GetCollectedErrors()) != expectedCount {
				t.Errorf("Expected %d collected errors, got %d", expectedCount, len(handler.GetCollectedErrors()))
			}

			if !handler.HasErrors() {
				t.Error("Expected handler to have errors")
			}
		})
	}
}

func TestDefaultErrorHandler_InteractiveMode(t *testing.T) {
	handler := NewErrorHandlerWithMode(ErrorModeInteractive)

	// Interactive mode currently behaves like strict mode
	err := NewErrorBuilder(ErrInvalidFormat, "error message").WithSeverity(SeverityError).Build()
	result := handler.HandleError(err)

	if result == nil {
		t.Error("Expected error to be returned in interactive mode")
	}

	// Should not collect errors in current implementation
	if handler.HasErrors() {
		t.Error("Expected interactive mode to not collect errors (current implementation)")
	}
}

func TestDefaultErrorHandler_Clear(t *testing.T) {
	handler := NewErrorHandlerWithMode(ErrorModeLenient)

	// Add some errors
	err1 := NewErrorBuilder(ErrInvalidFormat, "error 1").Build()
	err2 := NewErrorBuilder(ErrMissingColumn, "error 2").Build()

	handler.HandleError(err1)
	handler.HandleError(err2)

	if len(handler.GetCollectedErrors()) != 2 {
		t.Errorf("Expected 2 collected errors, got %d", len(handler.GetCollectedErrors()))
	}

	// Clear errors
	handler.Clear()

	if handler.HasErrors() {
		t.Error("Expected no errors after clear")
	}

	if len(handler.GetCollectedErrors()) != 0 {
		t.Errorf("Expected 0 collected errors after clear, got %d", len(handler.GetCollectedErrors()))
	}
}

func TestDefaultErrorHandler_SetWarningHandler(t *testing.T) {
	handler := NewErrorHandlerWithMode(ErrorModeStrict)

	var capturedWarning error
	handler.SetWarningHandler(func(err error) {
		capturedWarning = err
	})

	warningErr := NewErrorBuilder(ErrInvalidFormat, "warning message").WithSeverity(SeverityWarning).Build()
	result := handler.HandleError(warningErr)

	if result != nil {
		t.Errorf("Expected no error to be returned for warning, got %v", result)
	}

	if capturedWarning == nil {
		t.Error("Expected warning to be captured by warning handler")
	}

	if capturedWarning != warningErr {
		t.Error("Expected captured warning to match input warning")
	}
}

func TestDefaultErrorHandler_Summary(t *testing.T) {
	handler := NewErrorHandlerWithMode(ErrorModeLenient)

	// Add various errors
	err1 := NewErrorBuilder(ErrInvalidFormat, "error 1").WithSeverity(SeverityError).WithSuggestions("Fix format").Build()
	err2 := NewErrorBuilder(ErrMissingColumn, "error 2").WithSeverity(SeverityWarning).WithSuggestions("Add column", "Check schema").Build()
	err3 := NewErrorBuilder(ErrInvalidFormat, "error 3").WithSeverity(SeverityInfo).WithSuggestions("Fix format").Build() // Duplicate suggestion
	err4 := NewErrorBuilder(ErrFileWrite, "error 4").WithSeverity(SeverityFatal).Build()

	handler.HandleError(err1)
	handler.HandleError(err2)
	handler.HandleError(err3)
	handler.HandleError(err4)

	summary := handler.Summary()

	// Test total errors
	if summary.TotalErrors != 4 {
		t.Errorf("Expected 4 total errors, got %d", summary.TotalErrors)
	}

	// Test by category
	if summary.ByCategory[ErrInvalidFormat] != 2 {
		t.Errorf("Expected 2 ErrInvalidFormat errors, got %d", summary.ByCategory[ErrInvalidFormat])
	}

	if summary.ByCategory[ErrMissingColumn] != 1 {
		t.Errorf("Expected 1 ErrMissingColumn error, got %d", summary.ByCategory[ErrMissingColumn])
	}

	if summary.ByCategory[ErrFileWrite] != 1 {
		t.Errorf("Expected 1 ErrFileWrite error, got %d", summary.ByCategory[ErrFileWrite])
	}

	// Test by severity
	if summary.BySeverity[SeverityError] != 1 {
		t.Errorf("Expected 1 error severity, got %d", summary.BySeverity[SeverityError])
	}

	if summary.BySeverity[SeverityWarning] != 1 {
		t.Errorf("Expected 1 warning severity, got %d", summary.BySeverity[SeverityWarning])
	}

	if summary.BySeverity[SeverityInfo] != 1 {
		t.Errorf("Expected 1 info severity, got %d", summary.BySeverity[SeverityInfo])
	}

	if summary.BySeverity[SeverityFatal] != 1 {
		t.Errorf("Expected 1 fatal severity, got %d", summary.BySeverity[SeverityFatal])
	}

	// Test unique suggestions (should deduplicate "Fix format")
	expectedSuggestions := 3 // "Fix format", "Add column", "Check schema"
	if len(summary.Suggestions) != expectedSuggestions {
		t.Errorf("Expected %d unique suggestions, got %d: %v", expectedSuggestions, len(summary.Suggestions), summary.Suggestions)
	}

	// Test fixable errors (warnings and info)
	if summary.FixableErrors != 2 {
		t.Errorf("Expected 2 fixable errors, got %d", summary.FixableErrors)
	}
}

func TestDefaultErrorHandler_HasErrorsWithSeverity(t *testing.T) {
	handler := NewErrorHandlerWithMode(ErrorModeLenient)

	// Add errors with different severities
	infoErr := NewErrorBuilder(ErrInvalidFormat, "info").WithSeverity(SeverityInfo).Build()
	warningErr := NewErrorBuilder(ErrMissingColumn, "warning").WithSeverity(SeverityWarning).Build()
	errorErr := NewErrorBuilder(ErrFileWrite, "error").WithSeverity(SeverityError).Build()

	handler.HandleError(infoErr)
	handler.HandleError(warningErr)
	handler.HandleError(errorErr)

	tests := []struct {
		name     string
		severity ErrorSeverity
		expected bool
	}{
		{"Has info or higher", SeverityInfo, true},
		{"Has warning or higher", SeverityWarning, true},
		{"Has error or higher", SeverityError, true},
		{"Has fatal or higher", SeverityFatal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.HasErrorsWithSeverity(tt.severity)
			if result != tt.expected {
				t.Errorf("HasErrorsWithSeverity(%v) = %v, want %v", tt.severity, result, tt.expected)
			}
		})
	}
}

func TestErrorSummary_JSONMarshaling(t *testing.T) {
	summary := ErrorSummary{
		TotalErrors: 3,
		ByCategory: map[ErrorCode]int{
			ErrInvalidFormat: 2,
			ErrMissingColumn: 1,
		},
		BySeverity: map[ErrorSeverity]int{
			SeverityError:   2,
			SeverityWarning: 1,
		},
		Suggestions:   []string{"Fix format", "Add column"},
		FixableErrors: 1,
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorSummary: %v", err)
	}

	var result ErrorSummary
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal ErrorSummary: %v", err)
	}

	if result.TotalErrors != summary.TotalErrors {
		t.Errorf("Expected TotalErrors %d, got %d", summary.TotalErrors, result.TotalErrors)
	}

	if len(result.Suggestions) != len(summary.Suggestions) {
		t.Errorf("Expected %d suggestions, got %d", len(summary.Suggestions), len(result.Suggestions))
	}
}

func TestDefaultErrorHandler_WrapStandardError(t *testing.T) {
	handler := NewDefaultErrorHandler()

	standardErr := errors.New("standard error message")
	result := handler.HandleError(standardErr)

	if result == nil {
		t.Error("Expected wrapped standard error to be returned")
	}

	outputErr, ok := result.(OutputError)
	if !ok {
		t.Error("Expected result to be an OutputError")
	}

	if outputErr.Code() != ErrFormatGeneration {
		t.Errorf("Expected error code %v, got %v", ErrFormatGeneration, outputErr.Code())
	}

	if !strings.Contains(outputErr.Error(), "unexpected error occurred") {
		t.Errorf("Expected wrapped error message, got: %v", outputErr.Error())
	}

	if !strings.Contains(outputErr.Error(), "standard error message") {
		t.Errorf("Expected original error in cause, got: %v", outputErr.Error())
	}
}
