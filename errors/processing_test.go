package errors

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestProcessingErrorInterface(t *testing.T) {
	// Test that processingError implements ProcessingError interface
	err := NewProcessingError(ErrFileWrite, "file write failed")

	var processingErr ProcessingError = err
	if processingErr == nil {
		t.Error("Expected processingError to implement ProcessingError interface")
	}

	// Test ProcessingError interface methods
	if processingErr.Retryable() {
		t.Error("Expected processing error to not be retryable by default")
	}

	if processingErr.PartialResult() != nil {
		t.Error("Expected no partial result by default")
	}

	// Test that it also implements OutputError
	var outputErr OutputError = err
	if outputErr == nil {
		t.Error("Expected processingError to implement OutputError interface")
	}
}

func TestProcessingErrorCreation(t *testing.T) {
	err := NewProcessingError(ErrS3Upload, "S3 upload failed")

	if err.Code() != ErrS3Upload {
		t.Errorf("Expected Code to be %s, got %s", ErrS3Upload, err.Code())
	}

	if err.Severity() != SeverityError {
		t.Errorf("Expected default Severity to be %d, got %d", SeverityError, err.Severity())
	}

	expectedMsg := "[OUT-3002] S3 upload failed"
	if err.Error() != expectedMsg {
		t.Errorf("Expected Error message to be '%s', got '%s'", expectedMsg, err.Error())
	}

	if err.Retryable() {
		t.Error("Expected processing error to not be retryable by default")
	}

	if err.PartialResult() != nil {
		t.Error("Expected no partial result by default")
	}
}

func TestProcessingErrorWithRetryable(t *testing.T) {
	err := NewProcessingError(ErrS3Upload, "temporary S3 failure").
		WithRetryable(true)

	if !err.Retryable() {
		t.Error("Expected error to be retryable")
	}

	errorStr := err.Error()
	if !contains(errorStr, "retryable") {
		t.Errorf("Error message should indicate retryable: %s", errorStr)
	}
}

func TestProcessingErrorWithPartialResult(t *testing.T) {
	partialData := map[string]interface{}{
		"processed_rows": 150,
		"total_rows":     200,
	}

	err := NewProcessingError(ErrTemplateRender, "template rendering failed").
		WithPartialResult(partialData)

	if err.PartialResult() == nil {
		t.Error("Expected partial result to be set")
	}

	result, ok := err.PartialResult().(map[string]interface{})
	if !ok {
		t.Error("Expected partial result to be a map")
	}

	if result["processed_rows"] != 150 {
		t.Errorf("Expected processed_rows to be 150, got %v", result["processed_rows"])
	}

	errorStr := err.Error()
	if !contains(errorStr, "Partial result available") {
		t.Errorf("Error message should mention partial result: %s", errorStr)
	}
}

func TestProcessingErrorBuilderPattern(t *testing.T) {
	partialData := []string{"item1", "item2"}

	err := NewProcessingError(ErrFileWrite, "file write failed").
		WithRetryable(true).
		WithPartialResult(partialData)

	// Need to type assert to ProcessingError to access processing-specific methods
	processingErr, ok := err.(ProcessingError)
	if !ok {
		t.Fatal("Expected ProcessingError type")
	}

	// Continue building with other methods
	finalErr := processingErr.WithContext(ErrorContext{Operation: "batch_write", Field: "output_file"}).
		WithSuggestions("check disk space", "verify permissions").
		WithSeverity(SeverityWarning)

	if finalErr.Code() != ErrFileWrite {
		t.Errorf("Expected Code to be %s, got %s", ErrFileWrite, finalErr.Code())
	}
	if finalErr.Severity() != SeverityWarning {
		t.Errorf("Expected Severity to be %d, got %d", SeverityWarning, finalErr.Severity())
	}
	if !processingErr.Retryable() {
		t.Error("Expected error to be retryable")
	}
	if processingErr.PartialResult() == nil {
		t.Error("Expected partial result to be set")
	}
	if finalErr.Context().Operation != "batch_write" {
		t.Errorf("Expected Operation to be 'batch_write', got %s", finalErr.Context().Operation)
	}
	if len(finalErr.Suggestions()) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(finalErr.Suggestions()))
	}

	errorStr := processingErr.Error()
	if !contains(errorStr, "[OUT-3001]") || !contains(errorStr, "file write failed") {
		t.Errorf("Error message should contain code and message: %s", errorStr)
	}
	if !contains(errorStr, "retryable") {
		t.Errorf("Error message should mention retryable: %s", errorStr)
	}
	if !contains(errorStr, "Partial result available") {
		t.Errorf("Error message should mention partial result: %s", errorStr)
	}
}

func TestRetryableErrorCreation(t *testing.T) {
	originalErr := fmt.Errorf("network timeout")
	retryableErr := NewRetryableError(originalErr, "upload failed due to timeout")

	if !retryableErr.Retryable() {
		t.Error("Expected retryable error to be retryable")
	}

	if retryableErr.Code() != ErrRetryable {
		t.Errorf("Expected Code to be %s, got %s", ErrRetryable, retryableErr.Code())
	}

	errorStr := retryableErr.Error()
	if !contains(errorStr, "retryable") {
		t.Errorf("Error message should mention retryable: %s", errorStr)
	}
	if !contains(errorStr, "network timeout") {
		t.Errorf("Error message should contain wrapped error: %s", errorStr)
	}
}

func TestRetryableErrorWrapper(t *testing.T) {
	// Test wrapping an existing ProcessingError
	originalErr := NewProcessingError(ErrS3Upload, "S3 upload failed")
	retryableErr := WrapAsRetryable(originalErr)

	if !retryableErr.Retryable() {
		t.Error("Expected wrapped error to be retryable")
	}

	if retryableErr.Code() != originalErr.Code() {
		t.Errorf("Expected Code to be preserved: %s, got %s", originalErr.Code(), retryableErr.Code())
	}

	// Test wrapping a regular error
	regularErr := fmt.Errorf("regular error")
	wrappedErr := WrapAsRetryable(regularErr)

	if !wrappedErr.Retryable() {
		t.Error("Expected wrapped regular error to be retryable")
	}

	if wrappedErr.Code() != ErrRetryable {
		t.Errorf("Expected Code to be %s for wrapped regular error, got %s", ErrRetryable, wrappedErr.Code())
	}
}

func TestProcessingErrorCodes(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrFileWrite, "OUT-3001"},
		{ErrS3Upload, "OUT-3002"},
		{ErrTemplateRender, "OUT-3003"},
		{ErrMemoryExhausted, "OUT-3004"},
		{ErrRetryable, "OUT-3005"},
	}

	for _, test := range tests {
		if string(test.code) != test.expected {
			t.Errorf("Expected error code %s to be %s, got %s", test.code, test.expected, string(test.code))
		}
	}
}

func TestProcessingErrorJSONMarshaling(t *testing.T) {
	partialData := map[string]int{"completed": 5, "total": 10}
	err := NewProcessingError(ErrTemplateRender, "rendering failed").
		WithRetryable(true).
		WithPartialResult(partialData).
		WithSeverity(SeverityWarning)

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal processing error to JSON: %v", marshalErr)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(data, &result); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	if result["code"] != string(ErrTemplateRender) {
		t.Errorf("Expected code to be %s, got %v", ErrTemplateRender, result["code"])
	}

	if result["retryable"] != true {
		t.Errorf("Expected retryable to be true, got %v", result["retryable"])
	}

	if result["partial_result"] == nil {
		t.Error("Expected partial_result to be present in JSON")
	}
}

func TestProcessingErrorImmutability(t *testing.T) {
	original := NewProcessingError(ErrFileWrite, "original")

	// Creating new instances with modifications should not affect original
	withRetryable := original.WithRetryable(true)
	withPartialResult := original.WithPartialResult("data")

	// Original should remain unchanged
	if original.Retryable() {
		t.Error("Original error should not be retryable")
	}
	if original.PartialResult() != nil {
		t.Error("Original error should have no partial result")
	}

	// New instances should have the modifications
	if !withRetryable.Retryable() {
		t.Error("WithRetryable should make error retryable")
	}
	if withPartialResult.PartialResult() == nil {
		t.Error("WithPartialResult should set partial result")
	}
}

func TestNewProcessingErrorWithOptions(t *testing.T) {
	partialData := "some data"
	err := NewProcessingErrorWithOptions(
		ErrS3Upload,
		"upload failed",
		true, // retryable
		partialData,
	)

	if err.Code() != ErrS3Upload {
		t.Errorf("Expected Code to be %s, got %s", ErrS3Upload, err.Code())
	}
	if !err.Retryable() {
		t.Error("Expected error to be retryable")
	}
	if err.PartialResult() == nil {
		t.Error("Expected partial result to be set")
	}
	if err.PartialResult() != partialData {
		t.Errorf("Expected partial result to be %v, got %v", partialData, err.PartialResult())
	}
}

func TestProcessingErrorSeverityHandling(t *testing.T) {
	// Test that memory exhausted errors default to Fatal
	memErr := NewProcessingError(ErrMemoryExhausted, "out of memory")
	if memErr.Severity() != SeverityFatal {
		t.Errorf("Expected memory exhausted error to be Fatal, got %d", memErr.Severity())
	}

	// Test that other processing errors default to Error
	fileErr := NewProcessingError(ErrFileWrite, "write failed")
	if fileErr.Severity() != SeverityError {
		t.Errorf("Expected file write error to be Error severity, got %d", fileErr.Severity())
	}
}

func TestRetryableErrorWithBackoff(t *testing.T) {
	err := NewRetryableErrorWithBackoff(
		fmt.Errorf("connection failed"),
		"network error",
		3,    // max attempts
		1000, // initial delay ms
	)

	if !err.Retryable() {
		t.Error("Expected error to be retryable")
	}

	// Check for retry configuration in error details
	if retryableErr, ok := err.(*retryableError); ok {
		if retryableErr.maxAttempts != 3 {
			t.Errorf("Expected max attempts to be 3, got %d", retryableErr.maxAttempts)
		}
		if retryableErr.initialDelayMs != 1000 {
			t.Errorf("Expected initial delay to be 1000ms, got %d", retryableErr.initialDelayMs)
		}
	} else {
		t.Error("Expected retryableError type")
	}
}

func TestTransientErrorDetection(t *testing.T) {
	tests := []struct {
		code      ErrorCode
		transient bool
	}{
		{ErrS3Upload, true},         // Network operations are typically transient
		{ErrFileWrite, false},       // File operations are typically not transient
		{ErrTemplateRender, false},  // Template errors are typically not transient
		{ErrMemoryExhausted, false}, // Memory issues are typically not transient
	}

	for _, test := range tests {
		err := NewProcessingError(test.code, "test error")
		if IsTransient(err) != test.transient {
			t.Errorf("Expected %s to have transient=%v, got %v", test.code, test.transient, IsTransient(err))
		}
	}
}
