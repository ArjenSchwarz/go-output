package errors

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestErrorCode(t *testing.T) {
	// Test error code constants
	if ErrInvalidFormat != "OUT-1001" {
		t.Errorf("Expected ErrInvalidFormat to be 'OUT-1001', got %s", ErrInvalidFormat)
	}
	if ErrMissingRequired != "OUT-1002" {
		t.Errorf("Expected ErrMissingRequired to be 'OUT-1002', got %s", ErrMissingRequired)
	}
}

func TestErrorSeverity(t *testing.T) {
	// Test severity constants
	if SeverityFatal != 0 {
		t.Errorf("Expected SeverityFatal to be 0, got %d", SeverityFatal)
	}
	if SeverityError != 1 {
		t.Errorf("Expected SeverityError to be 1, got %d", SeverityError)
	}
	if SeverityWarning != 2 {
		t.Errorf("Expected SeverityWarning to be 2, got %d", SeverityWarning)
	}
	if SeverityInfo != 3 {
		t.Errorf("Expected SeverityInfo to be 3, got %d", SeverityInfo)
	}
}

func TestErrorContext(t *testing.T) {
	ctx := ErrorContext{
		Operation: "validation",
		Field:     "name",
		Value:     "test",
		Index:     0,
		Metadata:  map[string]interface{}{"key": "value"},
	}

	if ctx.Operation != "validation" {
		t.Errorf("Expected Operation to be 'validation', got %s", ctx.Operation)
	}
	if ctx.Field != "name" {
		t.Errorf("Expected Field to be 'name', got %s", ctx.Field)
	}
	if ctx.Value != "test" {
		t.Errorf("Expected Value to be 'test', got %v", ctx.Value)
	}
	if ctx.Index != 0 {
		t.Errorf("Expected Index to be 0, got %d", ctx.Index)
	}
	if ctx.Metadata["key"] != "value" {
		t.Errorf("Expected Metadata key to be 'value', got %v", ctx.Metadata["key"])
	}
}

func TestOutputErrorInterface(t *testing.T) {
	// Test that baseError implements OutputError interface
	err := NewError(ErrInvalidFormat, "test error")

	var outputErr OutputError = err
	if outputErr == nil {
		t.Error("Expected baseError to implement OutputError interface")
	}

	// Test interface methods
	if outputErr.Code() != ErrInvalidFormat {
		t.Errorf("Expected Code to be %s, got %s", ErrInvalidFormat, outputErr.Code())
	}

	if outputErr.Severity() != SeverityError {
		t.Errorf("Expected Severity to be %d, got %d", SeverityError, outputErr.Severity())
	}

	if outputErr.Error() == "" {
		t.Error("Expected Error() to return non-empty string")
	}

	if len(outputErr.Suggestions()) != 0 {
		t.Errorf("Expected no suggestions initially, got %d", len(outputErr.Suggestions()))
	}

	ctx := outputErr.Context()
	if ctx.Operation != "" {
		t.Errorf("Expected empty context initially, got Operation: %s", ctx.Operation)
	}
}

func TestBaseErrorCreation(t *testing.T) {
	err := NewError(ErrInvalidFormat, "invalid format specified")

	if err.Code() != ErrInvalidFormat {
		t.Errorf("Expected Code to be %s, got %s", ErrInvalidFormat, err.Code())
	}

	if err.Severity() != SeverityError {
		t.Errorf("Expected default Severity to be %d, got %d", SeverityError, err.Severity())
	}

	expectedMsg := "[OUT-1001] invalid format specified"
	if err.Error() != expectedMsg {
		t.Errorf("Expected Error message to be '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestBaseErrorWithContext(t *testing.T) {
	ctx := ErrorContext{
		Operation: "validation",
		Field:     "outputFormat",
		Value:     "invalid",
	}

	err := NewError(ErrInvalidFormat, "invalid format").WithContext(ctx)

	if err.Context().Field != "outputFormat" {
		t.Errorf("Expected Field to be 'outputFormat', got %s", err.Context().Field)
	}

	expectedMsg := "[OUT-1001] invalid format (field: outputFormat)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected Error message to be '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestBaseErrorWithSuggestions(t *testing.T) {
	suggestions := []string{"use 'json' format", "use 'csv' format"}
	err := NewError(ErrInvalidFormat, "invalid format").WithSuggestions(suggestions...)

	if len(err.Suggestions()) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(err.Suggestions()))
	}

	expectedMsg := "[OUT-1001] invalid format\nSuggestions:\n  - use 'json' format\n  - use 'csv' format\n"
	if err.Error() != expectedMsg {
		t.Errorf("Expected Error message to be '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestBaseErrorWithSeverity(t *testing.T) {
	err := NewError(ErrInvalidFormat, "test error").WithSeverity(SeverityWarning)

	if err.Severity() != SeverityWarning {
		t.Errorf("Expected Severity to be %d, got %d", SeverityWarning, err.Severity())
	}
}

func TestBaseErrorWrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := NewError(ErrInvalidFormat, "wrapper error").Wrap(originalErr)

	expectedMsg := "[OUT-1001] wrapper error\nCaused by: original error"
	if err.Error() != expectedMsg {
		t.Errorf("Expected Error message to be '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestBaseErrorBuilderPattern(t *testing.T) {
	ctx := ErrorContext{Field: "name", Value: "test"}
	suggestions := []string{"check the value"}
	originalErr := errors.New("underlying issue")

	err := NewError(ErrMissingRequired, "field is required").
		WithContext(ctx).
		WithSuggestions(suggestions...).
		WithSeverity(SeverityFatal).
		Wrap(originalErr)

	if err.Code() != ErrMissingRequired {
		t.Errorf("Expected Code to be %s, got %s", ErrMissingRequired, err.Code())
	}
	if err.Severity() != SeverityFatal {
		t.Errorf("Expected Severity to be %d, got %d", SeverityFatal, err.Severity())
	}
	if err.Context().Field != "name" {
		t.Errorf("Expected Field to be 'name', got %s", err.Context().Field)
	}
	if len(err.Suggestions()) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(err.Suggestions()))
	}

	errorStr := err.Error()
	if !contains(errorStr, "[OUT-1002]") || !contains(errorStr, "field is required") {
		t.Errorf("Error message should contain code and message: %s", errorStr)
	}
	if !contains(errorStr, "field: name") {
		t.Errorf("Error message should contain field context: %s", errorStr)
	}
	if !contains(errorStr, "check the value") {
		t.Errorf("Error message should contain suggestions: %s", errorStr)
	}
	if !contains(errorStr, "underlying issue") {
		t.Errorf("Error message should contain wrapped error: %s", errorStr)
	}
}

func TestBaseErrorJSONMarshaling(t *testing.T) {
	ctx := ErrorContext{Field: "test", Value: "value"}
	err := NewError(ErrInvalidFormat, "test error").
		WithContext(ctx).
		WithSuggestions("try this").
		WithSeverity(SeverityWarning)

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal error to JSON: %v", marshalErr)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(data, &result); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	if result["code"] != string(ErrInvalidFormat) {
		t.Errorf("Expected code to be %s, got %v", ErrInvalidFormat, result["code"])
	}
	if result["message"] != "test error" {
		t.Errorf("Expected message to be 'test error', got %v", result["message"])
	}
	if result["severity"] != "Warning" {
		t.Errorf("Expected severity to be 'Warning', got %v", result["severity"])
	}
}

func TestBaseErrorImmutability(t *testing.T) {
	original := NewError(ErrInvalidFormat, "original")

	// Creating new instances with modifications should not affect original
	withContext := original.WithContext(ErrorContext{Field: "test"})
	withSuggestions := original.WithSuggestions("suggestion")
	withSeverity := original.WithSeverity(SeverityFatal)

	// Original should remain unchanged
	if original.Context().Field != "" {
		t.Errorf("Original error context should be empty, got Field: %s", original.Context().Field)
	}
	if len(original.Suggestions()) != 0 {
		t.Errorf("Original error should have no suggestions, got %d", len(original.Suggestions()))
	}
	if original.Severity() != SeverityError {
		t.Errorf("Original error severity should be default, got %d", original.Severity())
	}

	// New instances should have the modifications
	if withContext.Context().Field != "test" {
		t.Errorf("WithContext should set field, got %s", withContext.Context().Field)
	}
	if len(withSuggestions.Suggestions()) != 1 {
		t.Errorf("WithSuggestions should add suggestion, got %d", len(withSuggestions.Suggestions()))
	}
	if withSeverity.Severity() != SeverityFatal {
		t.Errorf("WithSeverity should change severity, got %d", withSeverity.Severity())
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrInvalidFormat, "OUT-1001"},
		{ErrMissingRequired, "OUT-1002"},
		{ErrIncompatibleConfig, "OUT-1003"},
		{ErrInvalidFilePath, "OUT-1004"},
	}

	for _, test := range tests {
		if string(test.code) != test.expected {
			t.Errorf("Expected error code %s to be %s, got %s", test.code, test.expected, string(test.code))
		}
	}
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		expected string
	}{
		{SeverityFatal, "Fatal"},
		{SeverityError, "Error"},
		{SeverityWarning, "Warning"},
		{SeverityInfo, "Info"},
	}

	for _, test := range tests {
		if test.severity.String() != test.expected {
			t.Errorf("Expected severity %d to string as %s, got %s", test.severity, test.expected, test.severity.String())
		}
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
