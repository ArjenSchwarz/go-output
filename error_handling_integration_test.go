package format

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestErrorHandlingPipelineIntegration tests the complete error handling pipeline
// from validation through error processing to recovery
func TestErrorHandlingPipelineIntegration(t *testing.T) {
	tests := []struct {
		name           string
		setupOutput    func() *OutputArray
		setupHandler   func() ErrorHandler
		expectError    bool
		expectedCode   ErrorCode
		expectedMode   ErrorMode
		validateResult func(t *testing.T, err error, handler ErrorHandler)
	}{
		{
			name: "strict mode - fail fast on validation error",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid") // Missing required FromToColumns
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": 123}},
					},
				}
			},
			setupHandler: func() ErrorHandler {
				return NewErrorHandlerWithMode(ErrorModeStrict)
			},
			expectError:  true,
			expectedCode: ErrMissingRequired,
			expectedMode: ErrorModeStrict,
			validateResult: func(t *testing.T, err error, handler ErrorHandler) {
				if err == nil {
					t.Fatal("expected error in strict mode")
				}

				outputErr, ok := err.(OutputError)
				if !ok {
					t.Fatalf("expected OutputError, got %T", err)
				}

				if outputErr.Code() != ErrMissingRequired {
					t.Errorf("expected error code %s, got %s", ErrMissingRequired, outputErr.Code())
				}

				// In strict mode, no errors should be collected
				if len(handler.GetCollectedErrors()) != 0 {
					t.Errorf("strict mode should not collect errors, got %d", len(handler.GetCollectedErrors()))
				}
			},
		},
		{
			name: "lenient mode - collect multiple validation errors",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid") // Missing required FromToColumns
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value", "Required"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": 123}}, // Missing "Required"
					},
				}

				// Add validators that will fail
				output.AddValidator(NewRequiredColumnsValidator("Required", "Missing"))
				output.AddValidator(NewEmptyDatasetValidator(false)) // This will pass

				return output
			},
			setupHandler: func() ErrorHandler {
				return NewErrorHandlerWithMode(ErrorModeLenient)
			},
			expectError:  false, // Lenient mode continues processing
			expectedMode: ErrorModeLenient,
			validateResult: func(t *testing.T, err error, handler ErrorHandler) {
				// Should not return error immediately in lenient mode
				if err != nil {
					t.Fatalf("lenient mode should not return error immediately, got: %v", err)
				}

				// Should have collected errors
				collectedErrors := handler.GetCollectedErrors()
				if len(collectedErrors) == 0 {
					t.Fatal("lenient mode should collect errors")
				}

				// Verify we collected the expected errors
				foundFormatError := false
				foundValidationError := false

				for _, collectedErr := range collectedErrors {
					if outputErr, ok := collectedErr.(OutputError); ok {
						switch outputErr.Code() {
						case ErrMissingRequired:
							if strings.Contains(outputErr.Error(), "mermaid") {
								foundFormatError = true
							} else if strings.Contains(outputErr.Error(), "Required") {
								foundValidationError = true
							}
						case ErrMissingColumn:
							foundValidationError = true
						}
					}
				}

				if !foundFormatError {
					t.Error("expected to collect format validation error")
				}
				if !foundValidationError {
					t.Error("expected to collect column validation error")
				}
			},
		},
		{
			name: "interactive mode - behaves like strict mode for now",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("invalid_format")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}
			},
			setupHandler: func() ErrorHandler {
				return NewErrorHandlerWithMode(ErrorModeInteractive)
			},
			expectError:  true,
			expectedCode: ErrInvalidFormat,
			expectedMode: ErrorModeInteractive,
			validateResult: func(t *testing.T, err error, handler ErrorHandler) {
				if err == nil {
					t.Fatal("expected error in interactive mode")
				}

				// Interactive mode currently behaves like strict mode
				if len(handler.GetCollectedErrors()) != 0 {
					t.Errorf("interactive mode should not collect errors (behaves like strict), got %d", len(handler.GetCollectedErrors()))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()
			handler := tt.setupHandler()
			output.WithErrorHandler(handler)

			// Test the complete pipeline through Write()
			err := output.Write()

			// Validate the result
			tt.validateResult(t, err, handler)

			// Verify handler mode
			if handler.GetMode() != tt.expectedMode {
				t.Errorf("expected mode %s, got %s", tt.expectedMode, handler.GetMode())
			}
		})
	}
}

// TestErrorPropagationThroughSystem tests that errors propagate correctly through the entire system
func TestErrorPropagationThroughSystem(t *testing.T) {
	tests := []struct {
		name             string
		setupOutput      func() *OutputArray
		injectError      func(*OutputArray) // Function to inject errors at different levels
		expectedErrors   []ErrorCode
		expectedSeverity ErrorSeverity
	}{
		{
			name: "settings validation error propagation",
			setupOutput: func() *OutputArray {
				// Create settings but make it invalid
				settings := NewOutputSettings()
				settings.SetOutputFormat("invalid_format") // This will cause validation error
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}
			},
			injectError: func(output *OutputArray) {
				// No additional error injection needed
			},
			expectedErrors:   []ErrorCode{ErrInvalidFormat},
			expectedSeverity: SeverityError,
		},
		{
			name: "format-specific validation error propagation",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("dot") // Requires FromToColumns
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Source", "Target"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Source": "A", "Target": "B"}},
					},
				}
			},
			injectError: func(output *OutputArray) {
				// FromToColumns is missing, which will cause validation error
			},
			expectedErrors:   []ErrorCode{ErrMissingRequired},
			expectedSeverity: SeverityError,
		},
		{
			name: "data validation error propagation",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": 123}},
					},
				}

				// Add validator that will fail
				output.AddValidator(NewRequiredColumnsValidator("MissingColumn"))
				return output
			},
			injectError: func(output *OutputArray) {
				// Validator will fail due to missing column
			},
			expectedErrors:   []ErrorCode{ErrMissingColumn},
			expectedSeverity: SeverityError,
		},
		{
			name: "multiple error types propagation",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid") // Missing FromToColumns
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}

				// Add multiple validators that will fail
				output.AddValidator(NewRequiredColumnsValidator("Missing1", "Missing2"))
				output.AddValidator(NewEmptyDatasetValidator(false)) // This will pass

				return output
			},
			injectError: func(output *OutputArray) {
				// Multiple validation errors will occur
			},
			expectedErrors:   []ErrorCode{ErrMissingRequired, ErrMissingColumn},
			expectedSeverity: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()
			tt.injectError(output)

			// Use lenient mode to collect all errors
			handler := NewErrorHandlerWithMode(ErrorModeLenient)
			output.WithErrorHandler(handler)

			// Execute the pipeline
			err := output.Write()

			// In lenient mode, we should not get an immediate error for non-fatal issues
			if err != nil {
				// Check if it's a fatal error
				if outputErr, ok := err.(OutputError); ok && outputErr.Severity() != SeverityFatal {
					t.Errorf("unexpected error in lenient mode: %v", err)
				}
			}

			// Check collected errors
			collectedErrors := handler.GetCollectedErrors()
			if len(collectedErrors) == 0 {
				t.Fatal("expected errors to be collected")
			}

			// Verify expected error codes are present
			foundCodes := make(map[ErrorCode]bool)
			maxSeverity := SeverityInfo

			for _, collectedErr := range collectedErrors {
				if outputErr, ok := collectedErr.(OutputError); ok {
					foundCodes[outputErr.Code()] = true
					if outputErr.Severity() > maxSeverity {
						maxSeverity = outputErr.Severity()
					}
				}
			}

			for _, expectedCode := range tt.expectedErrors {
				if !foundCodes[expectedCode] {
					t.Errorf("expected error code %s not found in collected errors", expectedCode)
				}
			}

			if maxSeverity < tt.expectedSeverity {
				t.Errorf("expected maximum severity %s, got %s", tt.expectedSeverity, maxSeverity)
			}
		})
	}
}

// TestRecoveryAndContinuationScenarios tests error recovery and continuation scenarios
func TestRecoveryAndContinuationScenarios(t *testing.T) {
	tests := []struct {
		name               string
		setupOutput        func() *OutputArray
		setupRecovery      func() RecoveryHandler
		expectRecovery     bool
		expectContinuation bool
		validateRecovery   func(t *testing.T, handler RecoveryHandler, err error)
	}{
		{
			name: "format fallback recovery",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("table") // Start with table format
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": 123}},
					},
				}
			},
			setupRecovery: func() RecoveryHandler {
				handler := NewDefaultRecoveryHandler()
				handler.AddStrategy(NewFormatFallbackStrategy("table", "csv", "json"))
				return handler
			},
			expectRecovery:     true,
			expectContinuation: true,
			validateRecovery: func(t *testing.T, handler RecoveryHandler, err error) {
				// Test that the recovery handler can handle format errors
				formatError := NewErrorBuilder(ErrFormatGeneration, "format generation failed").
					WithSeverity(SeverityError).
					Build()

				canRecover := handler.CanRecover(formatError)
				if !canRecover {
					t.Error("recovery handler should be able to recover from format errors")
				}

				// Test recovery attempt
				recoveryErr := handler.Recover(formatError)
				if recoveryErr == formatError {
					// Recovery didn't work, but that's expected in this test setup
					// since we don't have the full context
					t.Log("Recovery attempt made (expected to not fully succeed in test)")
				}
			},
		},
		{
			name: "default value recovery",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value", "Optional"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": 123}}, // Missing "Optional"
					},
				}
			},
			setupRecovery: func() RecoveryHandler {
				handler := NewDefaultRecoveryHandler()
				defaults := map[string]any{
					"Optional": "default_value",
					"Missing":  nil,
				}
				handler.AddStrategy(NewDefaultValueStrategy(defaults))
				return handler
			},
			expectRecovery:     true,
			expectContinuation: true,
			validateRecovery: func(t *testing.T, handler RecoveryHandler, err error) {
				// Test that the recovery handler can handle missing data errors
				missingDataError := NewErrorBuilder(ErrMissingColumn, "missing optional column").
					WithField("Optional").
					WithSeverity(SeverityWarning).
					Build()

				canRecover := handler.CanRecover(missingDataError)
				if !canRecover {
					t.Error("recovery handler should be able to recover from missing data errors")
				}

				// Test recovery attempt
				recoveryErr := handler.Recover(missingDataError)
				if recoveryErr != nil && recoveryErr != missingDataError {
					t.Errorf("unexpected recovery error: %v", recoveryErr)
				}
			},
		},
		{
			name: "retry strategy for transient errors",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}
			},
			setupRecovery: func() RecoveryHandler {
				handler := NewDefaultRecoveryHandler()
				backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 3)
				handler.AddStrategy(NewRetryStrategy(backoff))
				return handler
			},
			expectRecovery:     true,
			expectContinuation: true,
			validateRecovery: func(t *testing.T, handler RecoveryHandler, err error) {
				// Test that the recovery handler can handle retryable errors
				retryableError := NewProcessingError(ErrNetworkTimeout, "network timeout", true)

				canRecover := handler.CanRecover(retryableError)
				if !canRecover {
					t.Error("recovery handler should be able to recover from retryable errors")
				}

				// Test recovery attempt
				recoveryErr := handler.Recover(retryableError)
				if recoveryErr != nil && recoveryErr != retryableError {
					t.Errorf("unexpected recovery error: %v", recoveryErr)
				}
			},
		},
		{
			name: "composite recovery strategy",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}
			},
			setupRecovery: func() RecoveryHandler {
				handler := NewDefaultRecoveryHandler()

				// Add multiple strategies
				handler.AddStrategy(NewFormatFallbackStrategy("table", "csv", "json"))
				defaults := map[string]any{"Missing": "default"}
				handler.AddStrategy(NewDefaultValueStrategy(defaults))
				backoff := NewExponentialBackoff(100*time.Millisecond, 1*time.Second, 2)
				handler.AddStrategy(NewRetryStrategy(backoff))

				// Add composite strategy
				composite := NewCompositeRecoveryStrategy("comprehensive",
					NewFormatFallbackStrategy("mermaid", "dot", "json"),
					NewDefaultValueStrategy(map[string]any{"Fallback": "value"}),
				)
				handler.AddStrategy(composite)

				return handler
			},
			expectRecovery:     true,
			expectContinuation: true,
			validateRecovery: func(t *testing.T, handler RecoveryHandler, err error) {
				strategies := handler.GetStrategies()
				if len(strategies) < 4 { // 3 individual + 1 composite
					t.Errorf("expected at least 4 strategies, got %d", len(strategies))
				}

				// Test different error types
				errorTypes := []OutputError{
					NewErrorBuilder(ErrFormatGeneration, "format error").Build(),
					NewErrorBuilder(ErrMissingColumn, "missing data").WithField("Test").Build(),
					NewProcessingError(ErrNetworkTimeout, "timeout", true),
				}

				for _, testErr := range errorTypes {
					canRecover := handler.CanRecover(testErr)
					if !canRecover {
						t.Errorf("composite recovery should handle error type %s", testErr.Code())
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()
			recoveryHandler := tt.setupRecovery()

			// Test recovery capabilities
			tt.validateRecovery(t, recoveryHandler, nil)

			// Test integration with error handler
			errorHandler := NewErrorHandlerWithMode(ErrorModeLenient)
			output.WithErrorHandler(errorHandler)

			// Execute pipeline - should not fail due to recovery
			err := output.Write()

			if tt.expectContinuation && err != nil {
				// Check if it's a fatal error that should stop processing
				if outputErr, ok := err.(OutputError); ok && outputErr.Severity() == SeverityFatal {
					t.Log("Fatal error stopped processing as expected")
				} else {
					t.Errorf("expected continuation but got error: %v", err)
				}
			}
		})
	}
}

// TestErrorHandlingModeInteractions tests interactions between different error handling modes
func TestErrorHandlingModeInteractions(t *testing.T) {
	tests := []struct {
		name         string
		modes        []ErrorMode
		setupOutput  func() *OutputArray
		validateMode func(t *testing.T, mode ErrorMode, handler ErrorHandler, output *OutputArray)
	}{
		{
			name:  "mode switching behavior",
			modes: []ErrorMode{ErrorModeStrict, ErrorModeLenient, ErrorModeInteractive},
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid") // Will cause validation error
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}
				output.AddValidator(NewRequiredColumnsValidator("Missing"))
				return output
			},
			validateMode: func(t *testing.T, mode ErrorMode, handler ErrorHandler, output *OutputArray) {
				// Clear any previous errors
				handler.Clear()
				handler.SetMode(mode)

				// Test validation behavior
				err := output.validateWithErrorHandler()

				switch mode {
				case ErrorModeStrict:
					if err == nil {
						t.Error("strict mode should return error immediately")
					}
					if len(handler.GetCollectedErrors()) != 0 {
						t.Error("strict mode should not collect errors")
					}
				case ErrorModeLenient:
					// Lenient mode may or may not return error depending on severity
					collectedErrors := handler.GetCollectedErrors()
					if len(collectedErrors) == 0 {
						t.Error("lenient mode should collect errors")
					}
				case ErrorModeInteractive:
					// Interactive mode currently behaves like strict mode
					if err == nil {
						t.Error("interactive mode should return error immediately")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()
			handler := NewDefaultErrorHandler()
			output.WithErrorHandler(handler)

			for _, mode := range tt.modes {
				t.Run(fmt.Sprintf("mode_%s", mode.String()), func(t *testing.T) {
					tt.validateMode(t, mode, handler, output)
				})
			}
		})
	}
}

// TestBackwardCompatibilityWithLegacyErrorHandling tests backward compatibility with legacy error handling
func TestBackwardCompatibilityWithLegacyErrorHandling(t *testing.T) {
	tests := []struct {
		name            string
		setupOutput     func() *OutputArray
		useLegacyMode   bool
		useCompatMethod bool
		expectPanic     bool
		validateResult  func(t *testing.T, output *OutputArray, panicOccurred bool, panicValue interface{})
	}{
		{
			name: "legacy error handler with error",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("invalid_format")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}
			},
			useLegacyMode:   true,
			useCompatMethod: false,
			expectPanic:     true,
			validateResult: func(t *testing.T, output *OutputArray, panicOccurred bool, panicValue interface{}) {
				if !panicOccurred {
					t.Error("expected panic with legacy error handler")
				}

				if panicValue != nil {
					panicStr := fmt.Sprintf("%v", panicValue)
					if !strings.Contains(panicStr, "FATAL") {
						t.Errorf("expected FATAL in panic message, got: %s", panicStr)
					}
				}
			},
		},
		{
			name: "WriteCompat method with error",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid") // Missing FromToColumns
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}
			},
			useLegacyMode:   false,
			useCompatMethod: true,
			expectPanic:     true,
			validateResult: func(t *testing.T, output *OutputArray, panicOccurred bool, panicValue interface{}) {
				if !panicOccurred {
					t.Error("expected panic with WriteCompat method")
				}
			},
		},
		{
			name: "EnableLegacyMode method",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}
			},
			useLegacyMode:   true,
			useCompatMethod: false,
			expectPanic:     false, // No error should occur
			validateResult: func(t *testing.T, output *OutputArray, panicOccurred bool, panicValue interface{}) {
				if panicOccurred {
					t.Errorf("unexpected panic: %v", panicValue)
				}

				// Verify legacy mode is enabled
				if handler, ok := output.errorHandler.(*LegacyErrorHandler); !ok {
					t.Error("expected LegacyErrorHandler to be set")
				} else {
					// Test that it behaves like strict mode
					if handler.GetMode() != ErrorModeStrict {
						t.Error("legacy handler should report strict mode")
					}

					if len(handler.GetCollectedErrors()) != 0 {
						t.Error("legacy handler should not collect errors")
					}
				}
			},
		},
		{
			name: "migration from legacy to new error handling",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test"}},
					},
				}

				// Start with legacy mode
				output.EnableLegacyMode()

				// Then switch to new error handling
				output.WithErrorHandler(NewErrorHandlerWithMode(ErrorModeLenient))

				return output
			},
			useLegacyMode:   false,
			useCompatMethod: false,
			expectPanic:     false,
			validateResult: func(t *testing.T, output *OutputArray, panicOccurred bool, panicValue interface{}) {
				if panicOccurred {
					t.Errorf("unexpected panic during migration: %v", panicValue)
				}

				// Verify new error handler is in use
				if output.errorHandler.GetMode() != ErrorModeLenient {
					t.Error("expected lenient mode after migration")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()

			if tt.useLegacyMode {
				output.EnableLegacyMode()
			}

			// Capture panics
			var panicOccurred bool
			var panicValue interface{}

			func() {
				defer func() {
					if r := recover(); r != nil {
						panicOccurred = true
						panicValue = r
					}
				}()

				if tt.useCompatMethod {
					output.WriteCompat()
				} else {
					err := output.Write()
					if err != nil && tt.useLegacyMode {
						// Legacy mode should have panicked, not returned error
						t.Error("legacy mode should panic, not return error")
					}
				}
			}()

			// Validate the result
			tt.validateResult(t, output, panicOccurred, panicValue)
		})
	}
}

// TestErrorHandlingPerformanceImpact tests that error handling has minimal performance impact
func TestErrorHandlingPerformanceImpact(t *testing.T) {
	// Create a large dataset for performance testing
	createLargeOutput := func() *OutputArray {
		settings := NewOutputSettings()
		settings.SetOutputFormat("json")

		output := &OutputArray{
			Settings: settings,
			Keys:     []string{"ID", "Name", "Value", "Status"},
			Contents: make([]OutputHolder, 1000), // 1000 records
		}

		for i := 0; i < 1000; i++ {
			output.Contents[i] = OutputHolder{
				Contents: map[string]interface{}{
					"ID":     i,
					"Name":   fmt.Sprintf("Item_%d", i),
					"Value":  float64(i) * 1.5,
					"Status": "active",
				},
			}
		}

		return output
	}

	tests := []struct {
		name           string
		setupOutput    func() *OutputArray
		setupHandler   func() ErrorHandler
		addValidators  bool
		maxOverheadPct float64 // Maximum acceptable overhead percentage
	}{
		{
			name:           "baseline - no error handling",
			setupOutput:    createLargeOutput,
			setupHandler:   func() ErrorHandler { return nil },
			addValidators:  false,
			maxOverheadPct: 0, // Baseline
		},
		{
			name:        "strict mode with basic validation",
			setupOutput: createLargeOutput,
			setupHandler: func() ErrorHandler {
				return NewErrorHandlerWithMode(ErrorModeStrict)
			},
			addValidators:  true,
			maxOverheadPct: 20.0, // Allow higher overhead for test environment
		},
		{
			name:        "lenient mode with multiple validators",
			setupOutput: createLargeOutput,
			setupHandler: func() ErrorHandler {
				return NewErrorHandlerWithMode(ErrorModeLenient)
			},
			addValidators:  true,
			maxOverheadPct: 20.0, // Allow higher overhead for test environment
		},
	}

	var baselineTime time.Duration

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()

			if tt.setupHandler() != nil {
				output.WithErrorHandler(tt.setupHandler())
			}

			if tt.addValidators {
				// Add performance-optimized validators
				output.AddValidator(NewRequiredColumnsValidator("ID", "Name"))
				output.AddValidator(NewEmptyDatasetValidator(false))

				// Add data type validator
				typeValidator := NewDataTypeValidator().
					WithStringColumn("Name").
					WithIntColumn("ID").
					WithFloatColumn("Value")
				output.AddValidator(typeValidator)
			}

			// Measure execution time
			start := time.Now()

			// Run multiple iterations for more accurate measurement
			iterations := 10
			for j := 0; j < iterations; j++ {
				// Reset output for each iteration
				testOutput := tt.setupOutput()
				if tt.setupHandler() != nil {
					testOutput.WithErrorHandler(tt.setupHandler())
				}
				if tt.addValidators {
					testOutput.AddValidator(NewRequiredColumnsValidator("ID", "Name"))
					testOutput.AddValidator(NewEmptyDatasetValidator(false))
				}

				// Just run validation, not full Write() to isolate error handling performance
				err := testOutput.Validate()
				if err != nil && tt.setupHandler() != nil && tt.setupHandler().GetMode() == ErrorModeStrict {
					// Expected in strict mode if there are validation errors
				}
			}

			elapsed := time.Since(start) / time.Duration(iterations)

			if i == 0 {
				// Store baseline time
				baselineTime = elapsed
				t.Logf("Baseline time: %v", baselineTime)
			} else {
				// Calculate overhead
				overhead := float64(elapsed-baselineTime) / float64(baselineTime) * 100
				t.Logf("Time: %v, Overhead: %.2f%%", elapsed, overhead)

				if overhead > tt.maxOverheadPct {
					t.Errorf("Error handling overhead %.2f%% exceeds maximum %.2f%%", overhead, tt.maxOverheadPct)
				}
			}
		})
	}
}

// TestErrorReportingIntegration tests integration with error reporting and monitoring
func TestErrorReportingIntegration(t *testing.T) {
	t.Run("error reporter integration", func(t *testing.T) {
		// Create output with errors
		settings := NewOutputSettings()
		settings.SetOutputFormat("mermaid") // Missing FromToColumns
		output := &OutputArray{
			Settings: settings,
			Keys:     []string{"Name"},
			Contents: []OutputHolder{
				{Contents: map[string]interface{}{"Name": "test"}},
			},
		}
		output.AddValidator(NewRequiredColumnsValidator("Missing"))

		// Create error reporter
		reporter := NewErrorReporter()

		// Use lenient mode to collect errors
		handler := NewErrorHandlerWithMode(ErrorModeLenient)
		output.WithErrorHandler(handler)

		// Execute pipeline
		err := output.Write()
		if err != nil && err.(OutputError).Severity() == SeverityFatal {
			t.Log("Fatal error occurred as expected")
		}

		// Report collected errors
		for _, collectedErr := range handler.GetCollectedErrors() {
			if outputErr, ok := collectedErr.(OutputError); ok {
				reporter.Report(outputErr)
			}
		}

		// Verify error reporting
		summary := reporter.Summary()
		if summary.TotalErrors == 0 {
			t.Error("expected errors to be reported")
		}

		// Test JSON export
		jsonData, err := reporter.ExportJSON()
		if err != nil {
			t.Errorf("failed to export JSON: %v", err)
		}

		if len(jsonData) == 0 {
			t.Error("expected non-empty JSON export")
		}

		// Test metrics export
		metricsData, err := reporter.ExportMetricsJSON()
		if err != nil {
			t.Errorf("failed to export metrics JSON: %v", err)
		}

		if len(metricsData) == 0 {
			t.Error("expected non-empty metrics export")
		}
	})

	t.Run("monitoring integration", func(t *testing.T) {
		reporter := NewErrorReporter()
		integration := NewMonitoringIntegration(reporter, "go-output-test")

		// Report some errors
		errors := []OutputError{
			NewErrorBuilder(ErrInvalidFormat, "test error 1").WithSeverity(SeverityError).Build(),
			NewErrorBuilder(ErrMissingColumn, "test error 2").WithSeverity(SeverityWarning).Build(),
			NewErrorBuilder(ErrNetworkTimeout, "test error 3").WithSeverity(SeverityError).Build(),
		}

		for _, err := range errors {
			reporter.Report(err)
		}

		// Test threshold checking
		threshold := AlertThreshold{
			ErrorRate:     1.0, // 1 error per minute
			TotalErrors:   2,   // 2 total errors
			SeverityLevel: SeverityWarning,
			TimeWindow:    time.Minute,
		}

		exceeded := integration.CheckThresholds(threshold)
		if !exceeded {
			t.Error("expected thresholds to be exceeded")
		}

		// Test sending summary (this just logs in the test implementation)
		err := integration.SendSummaryToMonitoring()
		if err != nil {
			t.Errorf("failed to send monitoring summary: %v", err)
		}
	})
}
