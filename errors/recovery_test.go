package errors

import (
	"fmt"
	"testing"
	"time"
)

// Test RecoveryHandler interface
func TestRecoveryHandler_Interface(t *testing.T) {
	var handler RecoveryHandler

	// Test that we can assign a DefaultRecoveryHandler
	handler = NewDefaultRecoveryHandler()
	if handler == nil {
		t.Fatal("NewDefaultRecoveryHandler should not return nil")
	}

	// Test interface methods exist
	err := NewError(ErrFileWrite, "test error")
	if canRecover := handler.CanRecover(err); canRecover {
		// This error shouldn't be recoverable by default
		t.Error("Default handler should not be able to recover ErrFileWrite")
	}

	if recoveryErr := handler.Recover(err); recoveryErr != nil {
		// Recovery should fail for non-recoverable errors
		if recoveryErr != err {
			t.Error("Recovery should return original error when it cannot recover")
		}
	}
}

// Test RecoveryStrategy interface
func TestRecoveryStrategy_Interface(t *testing.T) {
	var strategy RecoveryStrategy

	// Test FormatFallbackStrategy
	strategy = NewFormatFallbackStrategy([]string{"table", "csv", "json"})
	if strategy == nil {
		t.Fatal("NewFormatFallbackStrategy should not return nil")
	}

	err := NewError(ErrInvalidFormat, "unsupported format")
	if !strategy.ApplicableFor(err) {
		t.Error("FormatFallbackStrategy should be applicable for format errors")
	}

	// Test with mock context
	mockSettings := map[string]interface{}{
		"OutputFormat": "table",
	}

	result, applyErr := strategy.Apply(err, mockSettings)
	if applyErr != nil {
		t.Errorf("Apply should not return error: %v", applyErr)
	}
	if result == nil {
		t.Error("Apply should return non-nil result")
	}
}

// Test FormatFallbackStrategy
func TestFormatFallbackStrategy(t *testing.T) {
	testCases := []struct {
		name           string
		fallbackChain  []string
		currentFormat  string
		expectedFormat string
		shouldFail     bool
	}{
		{
			name:           "fallback from table to csv",
			fallbackChain:  []string{"table", "csv", "json"},
			currentFormat:  "table",
			expectedFormat: "csv",
			shouldFail:     false,
		},
		{
			name:           "fallback from csv to json",
			fallbackChain:  []string{"table", "csv", "json"},
			currentFormat:  "csv",
			expectedFormat: "json",
			shouldFail:     false,
		},
		{
			name:          "no fallback for last format",
			fallbackChain: []string{"table", "csv", "json"},
			currentFormat: "json",
			shouldFail:    true,
		},
		{
			name:          "unknown format",
			fallbackChain: []string{"table", "csv", "json"},
			currentFormat: "unknown",
			shouldFail:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strategy := NewFormatFallbackStrategy(tc.fallbackChain)

			mockSettings := map[string]interface{}{
				"OutputFormat": tc.currentFormat,
			}

			err := NewError(ErrInvalidFormat, "format error")
			result, applyErr := strategy.Apply(err, mockSettings)

			if tc.shouldFail {
				if applyErr == nil {
					t.Error("Expected Apply to fail")
				}
				return
			}

			if applyErr != nil {
				t.Errorf("Apply should not fail: %v", applyErr)
				return
			}

			updatedSettings, ok := result.(map[string]interface{})
			if !ok {
				t.Error("Result should be map[string]interface{}")
				return
			}

			if updatedSettings["OutputFormat"] != tc.expectedFormat {
				t.Errorf("Expected format %s, got %s", tc.expectedFormat, updatedSettings["OutputFormat"])
			}
		})
	}
}

// Test DefaultValueStrategy
func TestDefaultValueStrategy(t *testing.T) {
	defaults := map[string]interface{}{
		"Status": "Unknown",
		"Count":  0,
		"Active": false,
	}

	strategy := NewDefaultValueStrategy(defaults)

	testCases := []struct {
		name        string
		errorCode   ErrorCode
		context     interface{}
		expectedKey string
		expectedVal interface{}
		shouldFail  bool
	}{
		{
			name:      "missing field error",
			errorCode: ErrMissingRequired,
			context: map[string]interface{}{
				"MissingField": "Status",
				"Data":         map[string]interface{}{},
			},
			expectedKey: "Status",
			expectedVal: "Unknown",
			shouldFail:  false,
		},
		{
			name:       "not applicable error",
			errorCode:  ErrFileWrite,
			context:    map[string]interface{}{},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := NewError(tc.errorCode, "test error")

			if !tc.shouldFail && !strategy.ApplicableFor(err) {
				t.Error("Strategy should be applicable for missing field errors")
			}

			result, applyErr := strategy.Apply(err, tc.context)

			if tc.shouldFail {
				if applyErr == nil {
					t.Error("Expected Apply to fail")
				}
				return
			}

			if applyErr != nil {
				t.Errorf("Apply should not fail: %v", applyErr)
				return
			}

			updatedData, ok := result.(map[string]interface{})
			if !ok {
				t.Error("Result should be map[string]interface{}")
				return
			}

			if updatedData[tc.expectedKey] != tc.expectedVal {
				t.Errorf("Expected %s=%v, got %v", tc.expectedKey, tc.expectedVal, updatedData[tc.expectedKey])
			}
		})
	}
}

// Test RetryStrategy
func TestRetryStrategy(t *testing.T) {
	strategy := NewRetryStrategy(3, 100*time.Millisecond, IsTransient)

	testCases := []struct {
		name        string
		errorCode   ErrorCode
		retryable   bool
		shouldApply bool
	}{
		{
			name:        "retryable error",
			errorCode:   ErrS3Upload,
			retryable:   true,
			shouldApply: true,
		},
		{
			name:        "non-retryable error",
			errorCode:   ErrFileWrite,
			retryable:   false,
			shouldApply: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err OutputError
			if tc.retryable {
				err = NewProcessingError(tc.errorCode, "test error").WithRetryable(true)
			} else {
				err = NewError(tc.errorCode, "test error")
			}

			applicable := strategy.ApplicableFor(err)
			if applicable != tc.shouldApply {
				t.Errorf("Expected ApplicableFor=%v, got %v", tc.shouldApply, applicable)
			}

			if tc.shouldApply {
				result, applyErr := strategy.Apply(err, nil)
				if applyErr != nil {
					t.Errorf("Apply should not fail for retryable errors: %v", applyErr)
				}

				retryInfo, ok := result.(RetryInfo)
				if !ok {
					t.Error("Result should be RetryInfo")
				} else {
					if retryInfo.MaxAttempts != 3 {
						t.Errorf("Expected MaxAttempts=3, got %d", retryInfo.MaxAttempts)
					}
					if retryInfo.InitialDelay != 100*time.Millisecond {
						t.Errorf("Expected InitialDelay=100ms, got %v", retryInfo.InitialDelay)
					}
				}
			}
		})
	}
}

// Test DefaultRecoveryHandler
func TestDefaultRecoveryHandler(t *testing.T) {
	handler := NewDefaultRecoveryHandler()

	// Add strategies
	formatStrategy := NewFormatFallbackStrategy([]string{"table", "csv", "json"})
	defaultStrategy := NewDefaultValueStrategy(map[string]interface{}{
		"Status": "Unknown",
	})
	retryStrategy := NewRetryStrategy(3, 100*time.Millisecond, IsTransient)

	handler.AddStrategy(formatStrategy)
	handler.AddStrategy(defaultStrategy)
	handler.AddStrategy(retryStrategy)

	testCases := []struct {
		name        string
		error       OutputError
		context     interface{}
		canRecover  bool
		shouldApply bool
	}{
		{
			name:  "format error recovery",
			error: NewError(ErrInvalidFormat, "unsupported format"),
			context: map[string]interface{}{
				"OutputFormat": "table",
			},
			canRecover:  true,
			shouldApply: true,
		},
		{
			name:        "missing field recovery",
			error:       NewError(ErrMissingRequired, "missing field"),
			context:     map[string]interface{}{"MissingField": "Status", "Data": map[string]interface{}{}},
			canRecover:  true,
			shouldApply: true,
		},
		{
			name:        "retryable error recovery",
			error:       NewProcessingError(ErrS3Upload, "upload failed").WithRetryable(true),
			context:     nil,
			canRecover:  true,
			shouldApply: true,
		},
		{
			name:       "unrecoverable error",
			error:      NewError(ErrMemoryExhausted, "out of memory").WithSeverity(SeverityFatal),
			context:    nil,
			canRecover: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			canRecover := handler.CanRecover(tc.error)
			if canRecover != tc.canRecover {
				t.Errorf("Expected CanRecover=%v, got %v", tc.canRecover, canRecover)
			}

			if tc.canRecover {
				recoveryErr := handler.RecoverWithContext(tc.error, tc.context)
				if tc.shouldApply && recoveryErr != nil {
					t.Errorf("Recovery should succeed: %v", recoveryErr)
				}
				if !tc.shouldApply && recoveryErr == nil {
					t.Error("Recovery should fail")
				}
			}
		})
	}
}

// Test ChainedRecoveryHandler
func TestChainedRecoveryHandler(t *testing.T) {
	handler1 := NewDefaultRecoveryHandler()
	handler1.AddStrategy(NewFormatFallbackStrategy([]string{"table", "csv"}))

	handler2 := NewDefaultRecoveryHandler()
	handler2.AddStrategy(NewDefaultValueStrategy(map[string]interface{}{"Status": "Unknown"}))

	chained := NewChainedRecoveryHandler(handler1, handler2)

	// Test that it tries handler1 first, then handler2
	formatErr := NewError(ErrInvalidFormat, "format error")
	context := map[string]interface{}{"OutputFormat": "table"}

	if !chained.CanRecover(formatErr) {
		t.Error("Chained handler should be able to recover format errors")
	}

	recoveryErr := chained.RecoverWithContext(formatErr, context)
	if recoveryErr != nil {
		t.Errorf("Chained recovery should succeed: %v", recoveryErr)
	}

	// Test fallback to second handler
	missingErr := NewError(ErrMissingRequired, "missing field")
	missingContext := map[string]interface{}{"MissingField": "Status", "Data": map[string]interface{}{}}

	if !chained.CanRecover(missingErr) {
		t.Error("Chained handler should be able to recover missing field errors")
	}

	recoveryErr = chained.RecoverWithContext(missingErr, missingContext)
	if recoveryErr != nil {
		t.Errorf("Chained recovery should succeed for missing fields: %v", recoveryErr)
	}
}

// Test BackoffCalculator
func TestBackoffCalculator(t *testing.T) {
	calc := NewExponentialBackoffCalculator(100*time.Millisecond, 2.0, 5*time.Second)

	testCases := []struct {
		attempt  int
		expected time.Duration
		maxDelay time.Duration
	}{
		{1, 100 * time.Millisecond, 5 * time.Second},
		{2, 200 * time.Millisecond, 5 * time.Second},
		{3, 400 * time.Millisecond, 5 * time.Second},
		{4, 800 * time.Millisecond, 5 * time.Second},
		{10, 5 * time.Second, 5 * time.Second}, // Should cap at max
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("attempt_%d", tc.attempt), func(t *testing.T) {
			delay := calc.Calculate(tc.attempt)

			// Allow some tolerance for timing
			if tc.attempt <= 4 {
				if delay != tc.expected {
					t.Errorf("Expected delay %v for attempt %d, got %v", tc.expected, tc.attempt, delay)
				}
			} else {
				// For high attempts, just check it doesn't exceed max
				if delay > tc.maxDelay {
					t.Errorf("Delay %v exceeds max %v for attempt %d", delay, tc.maxDelay, tc.attempt)
				}
			}
		})
	}
}

// Test integration with error handler
func TestRecoveryWithErrorHandler(t *testing.T) {
	// Create error handler with recovery
	handler := NewDefaultErrorHandler()
	handler.SetMode(ErrorModeLenient)

	recovery := NewDefaultRecoveryHandler()
	recovery.AddStrategy(NewFormatFallbackStrategy([]string{"table", "csv", "json"}))

	// Add recovery handler to error handler (we'll need to implement this integration)
	// This test verifies the interface exists for future integration

	formatErr := NewError(ErrInvalidFormat, "unsupported format")

	// Test that recovery handler can work with the error
	if !recovery.CanRecover(formatErr) {
		t.Error("Recovery handler should be able to recover format errors")
	}

	// Test error handler can collect the error
	handledErr := handler.HandleError(formatErr)
	if handledErr != nil {
		t.Error("Lenient handler should not return error immediately")
	}

	summary := handler.GetSummary()
	if summary.TotalErrors != 1 {
		t.Errorf("Expected 1 error collected, got %d", summary.TotalErrors)
	}
}
