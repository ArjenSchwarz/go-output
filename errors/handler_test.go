package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorMode(t *testing.T) {
	tests := []struct {
		mode     ErrorMode
		expected string
	}{
		{ErrorModeStrict, "Strict"},
		{ErrorModeLenient, "Lenient"},
		{ErrorModeInteractive, "Interactive"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.mode.String() != tt.expected {
				t.Errorf("Expected ErrorMode.String() to return %s, got %s", tt.expected, tt.mode.String())
			}
		})
	}
}

func TestDefaultErrorHandlerCreation(t *testing.T) {
	handler := NewDefaultErrorHandler()

	if handler == nil {
		t.Fatal("Expected NewDefaultErrorHandler to return non-nil handler")
	}

	// Should default to strict mode
	if handler.Mode() != ErrorModeStrict {
		t.Errorf("Expected default mode to be Strict, got %s", handler.Mode().String())
	}

	// Should be empty initially
	if len(handler.Errors()) != 0 {
		t.Errorf("Expected no errors initially, got %d", len(handler.Errors()))
	}
}

func TestDefaultErrorHandlerSetMode(t *testing.T) {
	handler := NewDefaultErrorHandler()

	// Test setting different modes
	modes := []ErrorMode{ErrorModeStrict, ErrorModeLenient, ErrorModeInteractive}
	for _, mode := range modes {
		handler.SetMode(mode)
		if handler.Mode() != mode {
			t.Errorf("Expected mode to be %s, got %s", mode.String(), handler.Mode().String())
		}
	}
}

func TestDefaultErrorHandlerStrictMode(t *testing.T) {
	handler := NewDefaultErrorHandler()
	handler.SetMode(ErrorModeStrict)

	tests := []struct {
		name          string
		inputError    error
		shouldReturn  bool
		shouldCollect bool
		description   string
	}{
		{
			name:          "Fatal error returns immediately",
			inputError:    NewError(ErrMemoryExhausted, "out of memory").WithSeverity(SeverityFatal),
			shouldReturn:  true,
			shouldCollect: false,
			description:   "Fatal errors should be returned immediately in strict mode",
		},
		{
			name:          "Error returns immediately",
			inputError:    NewError(ErrInvalidFormat, "invalid format").WithSeverity(SeverityError),
			shouldReturn:  true,
			shouldCollect: false,
			description:   "Errors should be returned immediately in strict mode",
		},
		{
			name:          "Warning passes through",
			inputError:    NewError(ErrInvalidFormat, "minor issue").WithSeverity(SeverityWarning),
			shouldReturn:  false,
			shouldCollect: false,
			description:   "Warnings should not stop execution in strict mode",
		},
		{
			name:          "Info passes through",
			inputError:    NewError(ErrInvalidFormat, "info message").WithSeverity(SeverityInfo),
			shouldReturn:  false,
			shouldCollect: false,
			description:   "Info messages should not stop execution in strict mode",
		},
		{
			name:          "Regular error gets wrapped",
			inputError:    errors.New("regular error"),
			shouldReturn:  true,
			shouldCollect: false,
			description:   "Regular errors should be wrapped and returned in strict mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset handler state
			handler = NewDefaultErrorHandler()
			handler.SetMode(ErrorModeStrict)

			result := handler.HandleError(tt.inputError)

			if tt.shouldReturn && result == nil {
				t.Errorf("Expected error to be returned, got nil. %s", tt.description)
			}

			if !tt.shouldReturn && result != nil {
				t.Errorf("Expected no error to be returned, got %v. %s", result, tt.description)
			}

			collectedErrors := handler.Errors()
			if tt.shouldCollect && len(collectedErrors) == 0 {
				t.Errorf("Expected error to be collected, but none were. %s", tt.description)
			}

			if !tt.shouldCollect && len(collectedErrors) > 0 {
				t.Errorf("Expected no errors to be collected, but got %d. %s", len(collectedErrors), tt.description)
			}
		})
	}
}

func TestDefaultErrorHandlerLenientMode(t *testing.T) {
	handler := NewDefaultErrorHandler()
	handler.SetMode(ErrorModeLenient)

	tests := []struct {
		name          string
		inputError    error
		shouldReturn  bool
		shouldCollect bool
		description   string
	}{
		{
			name:          "Fatal error returns immediately",
			inputError:    NewError(ErrMemoryExhausted, "out of memory").WithSeverity(SeverityFatal),
			shouldReturn:  true,
			shouldCollect: true,
			description:   "Fatal errors should be returned and collected in lenient mode",
		},
		{
			name:          "Error gets collected",
			inputError:    NewError(ErrInvalidFormat, "invalid format").WithSeverity(SeverityError),
			shouldReturn:  false,
			shouldCollect: true,
			description:   "Errors should be collected but not returned in lenient mode",
		},
		{
			name:          "Warning gets collected",
			inputError:    NewError(ErrInvalidFormat, "minor issue").WithSeverity(SeverityWarning),
			shouldReturn:  false,
			shouldCollect: true,
			description:   "Warnings should be collected in lenient mode",
		},
		{
			name:          "Info gets collected",
			inputError:    NewError(ErrInvalidFormat, "info message").WithSeverity(SeverityInfo),
			shouldReturn:  false,
			shouldCollect: true,
			description:   "Info messages should be collected in lenient mode",
		},
		{
			name:          "Regular error gets wrapped and collected",
			inputError:    errors.New("regular error"),
			shouldReturn:  false,
			shouldCollect: true,
			description:   "Regular errors should be wrapped and collected in lenient mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset handler state
			handler = NewDefaultErrorHandler()
			handler.SetMode(ErrorModeLenient)

			result := handler.HandleError(tt.inputError)

			if tt.shouldReturn && result == nil {
				t.Errorf("Expected error to be returned, got nil. %s", tt.description)
			}

			if !tt.shouldReturn && result != nil {
				t.Errorf("Expected no error to be returned, got %v. %s", result, tt.description)
			}

			collectedErrors := handler.Errors()
			if tt.shouldCollect && len(collectedErrors) == 0 {
				t.Errorf("Expected error to be collected, but none were. %s", tt.description)
			}

			if !tt.shouldCollect && len(collectedErrors) > 0 {
				t.Errorf("Expected no errors to be collected, but got %d. %s", len(collectedErrors), tt.description)
			}
		})
	}
}

func TestDefaultErrorHandlerErrorCollection(t *testing.T) {
	handler := NewDefaultErrorHandler()
	handler.SetMode(ErrorModeLenient)

	// Add multiple errors
	errors := []error{
		NewError(ErrInvalidFormat, "error 1").WithSeverity(SeverityError),
		NewError(ErrMissingRequired, "error 2").WithSeverity(SeverityWarning),
		NewError(ErrInvalidDataType, "error 3").WithSeverity(SeverityInfo),
	}

	for _, err := range errors {
		handler.HandleError(err)
	}

	collected := handler.Errors()
	if len(collected) != len(errors) {
		t.Errorf("Expected %d collected errors, got %d", len(errors), len(collected))
	}

	// Test GetSummary
	summary := handler.GetSummary()
	if summary.TotalErrors != len(errors) {
		t.Errorf("Expected summary to show %d total errors, got %d", len(errors), summary.TotalErrors)
	}

	// Check severity breakdown
	expectedSeverities := map[ErrorSeverity]int{
		SeverityError:   1,
		SeverityWarning: 1,
		SeverityInfo:    1,
	}

	for severity, expectedCount := range expectedSeverities {
		if summary.BySeverity[severity] != expectedCount {
			t.Errorf("Expected %d errors of severity %s, got %d", expectedCount, severity.String(), summary.BySeverity[severity])
		}
	}
}

func TestDefaultErrorHandlerClear(t *testing.T) {
	handler := NewDefaultErrorHandler()
	handler.SetMode(ErrorModeLenient)

	// Add some errors
	handler.HandleError(NewError(ErrInvalidFormat, "test error"))
	handler.HandleError(NewError(ErrMissingRequired, "another error"))

	if len(handler.Errors()) != 2 {
		t.Errorf("Expected 2 errors before clear, got %d", len(handler.Errors()))
	}

	// Clear errors
	handler.Clear()

	if len(handler.Errors()) != 0 {
		t.Errorf("Expected 0 errors after clear, got %d", len(handler.Errors()))
	}

	summary := handler.GetSummary()
	if summary.TotalErrors != 0 {
		t.Errorf("Expected summary to show 0 total errors after clear, got %d", summary.TotalErrors)
	}
}

func TestDefaultErrorHandlerWithCallbacks(t *testing.T) {
	handler := NewDefaultErrorHandler()

	// Test warning handler callback
	var capturedWarnings []error
	handler.SetWarningHandler(func(err error) {
		capturedWarnings = append(capturedWarnings, err)
	})

	warningError := NewError(ErrInvalidFormat, "warning message").WithSeverity(SeverityWarning)
	handler.HandleError(warningError)

	if len(capturedWarnings) != 1 {
		t.Errorf("Expected 1 captured warning, got %d", len(capturedWarnings))
	}

	// Test info handler callback
	var capturedInfo []error
	handler.SetInfoHandler(func(err error) {
		capturedInfo = append(capturedInfo, err)
	})

	infoError := NewError(ErrInvalidFormat, "info message").WithSeverity(SeverityInfo)
	handler.HandleError(infoError)

	if len(capturedInfo) != 1 {
		t.Errorf("Expected 1 captured info message, got %d", len(capturedInfo))
	}
}

func TestLegacyErrorHandler(t *testing.T) {
	handler := NewLegacyErrorHandler()

	// Test that it implements ErrorHandler interface
	var _ ErrorHandler = handler

	if handler.Mode() != ErrorModeStrict {
		t.Errorf("Expected LegacyErrorHandler to be in strict mode, got %s", handler.Mode().String())
	}

	// Test that it always returns the error (simulating log.Fatal behavior)
	testError := NewError(ErrInvalidFormat, "test error")
	result := handler.HandleError(testError)

	if result != testError {
		t.Errorf("Expected LegacyErrorHandler to return the original error")
	}

	// Test with regular error
	regularError := errors.New("regular error")
	result = handler.HandleError(regularError)

	if result != regularError {
		t.Errorf("Expected LegacyErrorHandler to return the original regular error")
	}

	// Test with nil error
	result = handler.HandleError(nil)
	if result != nil {
		t.Errorf("Expected LegacyErrorHandler to return nil for nil input")
	}
}

func TestLegacyErrorHandlerSetMode(t *testing.T) {
	handler := NewLegacyErrorHandler()

	// Mode setting should have no effect on legacy handler
	handler.SetMode(ErrorModeLenient)
	if handler.Mode() != ErrorModeStrict {
		t.Errorf("Expected LegacyErrorHandler to remain in strict mode regardless of SetMode call")
	}
}

func TestErrorSummary(t *testing.T) {
	summary := ErrorSummary{
		TotalErrors: 5,
		BySeverity: map[ErrorSeverity]int{
			SeverityFatal:   1,
			SeverityError:   2,
			SeverityWarning: 1,
			SeverityInfo:    1,
		},
		ByCategory: map[ErrorCode]int{
			ErrInvalidFormat:   2,
			ErrMissingRequired: 2,
			ErrMemoryExhausted: 1,
		},
		Suggestions: []string{
			"Check your configuration",
			"Verify input data format",
		},
	}

	// Test HasErrors
	if !summary.HasErrors() {
		t.Error("Expected HasErrors to return true when there are errors")
	}

	// Test HasFatalErrors
	if !summary.HasFatalErrors() {
		t.Error("Expected HasFatalErrors to return true when there are fatal errors")
	}

	// Test GetHighestSeverity
	if summary.GetHighestSeverity() != SeverityFatal {
		t.Errorf("Expected highest severity to be Fatal, got %s", summary.GetHighestSeverity().String())
	}

	// Test empty summary
	emptySummary := ErrorSummary{}
	if emptySummary.HasErrors() {
		t.Error("Expected HasErrors to return false for empty summary")
	}

	if emptySummary.HasFatalErrors() {
		t.Error("Expected HasFatalErrors to return false for empty summary")
	}

	if emptySummary.GetHighestSeverity() != SeverityInfo {
		t.Errorf("Expected highest severity to be Info for empty summary, got %s", emptySummary.GetHighestSeverity().String())
	}
}

func TestErrorHandlerInterface(t *testing.T) {
	// Test that DefaultErrorHandler implements ErrorHandler interface
	var _ ErrorHandler = NewDefaultErrorHandler()

	// Test that LegacyErrorHandler implements ErrorHandler interface
	var _ ErrorHandler = NewLegacyErrorHandler()
}

func TestDefaultErrorHandlerConcurrency(t *testing.T) {
	handler := NewDefaultErrorHandler()
	handler.SetMode(ErrorModeLenient)

	// Test concurrent access to error handler
	numGoroutines := 10
	errorsPerGoroutine := 5
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < errorsPerGoroutine; j++ {
				err := NewError(ErrInvalidFormat, fmt.Sprintf("error %d-%d", goroutineID, j))
				handler.HandleError(err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	expectedTotal := numGoroutines * errorsPerGoroutine
	actualTotal := len(handler.Errors())

	if actualTotal != expectedTotal {
		t.Errorf("Expected %d total errors from concurrent access, got %d", expectedTotal, actualTotal)
	}

	summary := handler.GetSummary()
	if summary.TotalErrors != expectedTotal {
		t.Errorf("Expected summary to show %d total errors, got %d", expectedTotal, summary.TotalErrors)
	}
}