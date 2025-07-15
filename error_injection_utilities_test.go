package format

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestErrorInjector(t *testing.T) {
	t.Run("basic error injection", func(t *testing.T) {
		injector := NewErrorInjector()
		testErr := fmt.Errorf("test error")

		// Inject error
		injector.InjectError("test_operation", testErr)

		// Retrieve error
		retrievedErr := injector.GetError("test_operation")
		if retrievedErr == nil {
			t.Fatal("expected error to be retrieved")
		}

		if retrievedErr.Error() != testErr.Error() {
			t.Errorf("expected error message %q, got %q", testErr.Error(), retrievedErr.Error())
		}

		// Check error count
		if count := injector.GetErrorCount("test_operation"); count != 1 {
			t.Errorf("expected error count 1, got %d", count)
		}
	})

	t.Run("validation error injection", func(t *testing.T) {
		injector := NewErrorInjector()
		validationErr := NewValidationErrorBuilder(ErrMissingColumn, "test validation error").
			WithViolation("TestField", "required", "field is required", nil).
			Build()

		injector.InjectValidationError("validation_test", validationErr)

		retrievedErr := injector.GetError("validation_test")
		if retrievedErr == nil {
			t.Fatal("expected validation error to be retrieved")
		}

		if valErr, ok := retrievedErr.(ValidationError); ok {
			if len(valErr.Violations()) != 1 {
				t.Errorf("expected 1 violation, got %d", len(valErr.Violations()))
			}
		} else {
			t.Error("expected ValidationError type")
		}
	})

	t.Run("processing error injection", func(t *testing.T) {
		injector := NewErrorInjector()
		processingErr := NewProcessingError(ErrFileWrite, "test processing error", true)

		injector.InjectProcessingError("processing_test", processingErr)

		retrievedErr := injector.GetError("processing_test")
		if retrievedErr == nil {
			t.Fatal("expected processing error to be retrieved")
		}

		if procErr, ok := retrievedErr.(ProcessingError); ok {
			if !procErr.Retryable() {
				t.Error("expected processing error to be retryable")
			}
		} else {
			t.Error("expected ProcessingError type")
		}
	})

	t.Run("conditional error injection", func(t *testing.T) {
		injector := NewErrorInjector()
		testErr := fmt.Errorf("conditional error")
		conditionMet := false

		injector.InjectErrorWithCondition("conditional_test", testErr, func() bool {
			return conditionMet
		})

		// Should not return error when condition is false
		if err := injector.GetError("conditional_test"); err != nil {
			t.Error("expected no error when condition is false")
		}

		// Should return error when condition is true
		conditionMet = true
		if err := injector.GetError("conditional_test"); err == nil {
			t.Error("expected error when condition is true")
		}
	})

	t.Run("disable/enable functionality", func(t *testing.T) {
		injector := NewErrorInjector()
		testErr := fmt.Errorf("test error")

		injector.InjectError("test_operation", testErr)

		// Disable injection
		injector.Disable()
		if injector.IsEnabled() {
			t.Error("expected injector to be disabled")
		}

		// Should not return error when disabled
		if err := injector.GetError("test_operation"); err != nil {
			t.Error("expected no error when injector is disabled")
		}

		// Enable injection
		injector.Enable()
		if !injector.IsEnabled() {
			t.Error("expected injector to be enabled")
		}

		// Should return error when enabled
		if err := injector.GetError("test_operation"); err == nil {
			t.Error("expected error when injector is enabled")
		}
	})

	t.Run("clear functionality", func(t *testing.T) {
		injector := NewErrorInjector()
		injector.InjectError("test1", fmt.Errorf("error1"))
		injector.InjectError("test2", fmt.Errorf("error2"))

		// Verify errors exist
		if len(injector.GetAllOperations()) != 2 {
			t.Error("expected 2 operations before clear")
		}

		// Clear all errors
		injector.Clear()

		// Verify errors are cleared
		if len(injector.GetAllOperations()) != 0 {
			t.Error("expected 0 operations after clear")
		}

		if err := injector.GetError("test1"); err != nil {
			t.Error("expected no error after clear")
		}
	})

	t.Run("error creation helpers", func(t *testing.T) {
		injector := NewErrorInjector()

		// Test configuration error creation
		configErr := injector.CreateConfigurationError(ErrInvalidFormat, "test config error")
		if configErr.Code() != ErrInvalidFormat {
			t.Errorf("expected error code %s, got %s", ErrInvalidFormat, configErr.Code())
		}

		// Test validation error creation
		violations := []Violation{
			{Field: "test", Constraint: "required", Message: "field required", Value: nil},
		}
		valErr := injector.CreateValidationError(ErrMissingColumn, "test validation error", violations...)
		if valErr.Code() != ErrMissingColumn {
			t.Errorf("expected error code %s, got %s", ErrMissingColumn, valErr.Code())
		}
		if len(valErr.Violations()) != 1 {
			t.Errorf("expected 1 violation, got %d", len(valErr.Violations()))
		}

		// Test processing error creation
		procErr := injector.CreateProcessingError(ErrFileWrite, "test processing error", true)
		if procErr.Code() != ErrFileWrite {
			t.Errorf("expected error code %s, got %s", ErrFileWrite, procErr.Code())
		}
		if !procErr.Retryable() {
			t.Error("expected processing error to be retryable")
		}
	})
}

func TestMockValidator(t *testing.T) {
	t.Run("basic mock validator", func(t *testing.T) {
		validator := NewMockValidator("test_validator")

		if validator.Name() != "test_validator" {
			t.Errorf("expected name 'test_validator', got %q", validator.Name())
		}

		// Test successful validation
		validator.WithSuccess()
		if err := validator.Validate("test_data"); err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if validator.GetCallCount() != 1 {
			t.Errorf("expected call count 1, got %d", validator.GetCallCount())
		}
	})

	t.Run("mock validator with failure", func(t *testing.T) {
		validator := NewMockValidator("failing_validator")
		testErr := fmt.Errorf("validation failed")

		validator.WithFailure(testErr)

		if err := validator.Validate("test_data"); err == nil {
			t.Error("expected validation to fail")
		} else if err.Error() != testErr.Error() {
			t.Errorf("expected error %q, got %q", testErr.Error(), err.Error())
		}
	})

	t.Run("mock validator with custom validation", func(t *testing.T) {
		validator := NewMockValidator("custom_validator")

		validator.WithCustomValidation(func(subject any) error {
			if str, ok := subject.(string); ok && str == "invalid" {
				return fmt.Errorf("custom validation failed")
			}
			return nil
		})

		// Should pass for valid data
		if err := validator.Validate("valid"); err != nil {
			t.Errorf("expected no error for valid data, got %v", err)
		}

		// Should fail for invalid data
		if err := validator.Validate("invalid"); err == nil {
			t.Error("expected error for invalid data")
		}
	})

	t.Run("performance aware validator interface", func(t *testing.T) {
		validator := NewMockValidator("perf_validator").
			WithCost(15).
			WithFailFast(true)

		if validator.EstimatedCost() != 15 {
			t.Errorf("expected cost 15, got %d", validator.EstimatedCost())
		}

		if !validator.IsFailFast() {
			t.Error("expected validator to be fail-fast")
		}
	})

	t.Run("validator state tracking", func(t *testing.T) {
		validator := NewMockValidator("state_validator")
		testData := "test_subject"

		validator.Validate(testData)

		if validator.GetLastSubject() != testData {
			t.Errorf("expected last subject %q, got %v", testData, validator.GetLastSubject())
		}

		// Reset and verify
		validator.Reset()
		if validator.GetCallCount() != 0 {
			t.Errorf("expected call count 0 after reset, got %d", validator.GetCallCount())
		}
		if validator.GetLastSubject() != nil {
			t.Errorf("expected nil last subject after reset, got %v", validator.GetLastSubject())
		}
	})
}

func TestMockErrorHandler(t *testing.T) {
	t.Run("basic error handling", func(t *testing.T) {
		handler := NewMockErrorHandler()
		testErr := fmt.Errorf("test error")

		if err := handler.HandleError(testErr); err != nil {
			t.Errorf("expected no error returned, got %v", err)
		}

		if handler.GetCallCount() != 1 {
			t.Errorf("expected call count 1, got %d", handler.GetCallCount())
		}

		if handler.GetLastError() != testErr {
			t.Errorf("expected last error %v, got %v", testErr, handler.GetLastError())
		}
	})

	t.Run("error collection", func(t *testing.T) {
		handler := NewMockErrorHandler()
		err1 := fmt.Errorf("error 1")
		err2 := fmt.Errorf("error 2")

		handler.HandleError(err1)
		handler.HandleError(err2)

		collectedErrors := handler.GetCollectedErrors()
		if len(collectedErrors) != 2 {
			t.Errorf("expected 2 collected errors, got %d", len(collectedErrors))
		}

		// Clear and verify
		handler.Clear()
		if len(handler.GetCollectedErrors()) != 0 {
			t.Error("expected no collected errors after clear")
		}
	})

	t.Run("custom error handler", func(t *testing.T) {
		handler := NewMockErrorHandler()
		customErr := fmt.Errorf("custom handled error")

		handler.WithCustomHandler(func(err error) error {
			return customErr
		})

		result := handler.HandleError(fmt.Errorf("original error"))
		if result != customErr {
			t.Errorf("expected custom error %v, got %v", customErr, result)
		}
	})

	t.Run("error return configuration", func(t *testing.T) {
		handler := NewMockErrorHandler()
		returnErr := fmt.Errorf("return this error")

		handler.WithErrorReturn(returnErr)

		result := handler.HandleError(fmt.Errorf("input error"))
		if result != returnErr {
			t.Errorf("expected return error %v, got %v", returnErr, result)
		}
	})

	t.Run("mode handling", func(t *testing.T) {
		handler := NewMockErrorHandler()

		// Test default mode
		if handler.GetMode() != ErrorModeStrict {
			t.Errorf("expected default mode %v, got %v", ErrorModeStrict, handler.GetMode())
		}

		// Test mode setting
		handler.SetMode(ErrorModeLenient)
		if handler.GetMode() != ErrorModeLenient {
			t.Errorf("expected mode %v, got %v", ErrorModeLenient, handler.GetMode())
		}

		// Test with mode configuration
		handler2 := NewMockErrorHandler().WithMode(ErrorModeInteractive)
		if handler2.GetMode() != ErrorModeInteractive {
			t.Errorf("expected mode %v, got %v", ErrorModeInteractive, handler2.GetMode())
		}
	})
}

func TestMockRecoveryStrategy(t *testing.T) {
	t.Run("basic recovery strategy", func(t *testing.T) {
		strategy := NewMockRecoveryStrategy("test_strategy")
		testErr := NewConfigError(ErrInvalidFormat, "test error")

		if strategy.Name() != "test_strategy" {
			t.Errorf("expected name 'test_strategy', got %q", strategy.Name())
		}

		if strategy.Priority() != 10 {
			t.Errorf("expected default priority 10, got %d", strategy.Priority())
		}

		if !strategy.ApplicableFor(testErr) {
			t.Error("expected strategy to be applicable by default")
		}
	})

	t.Run("successful recovery", func(t *testing.T) {
		strategy := NewMockRecoveryStrategy("success_strategy").WithSuccess()
		testErr := NewConfigError(ErrInvalidFormat, "test error")

		result, err := strategy.Apply(testErr, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if result != "recovery_successful" {
			t.Errorf("expected 'recovery_successful', got %v", result)
		}

		if strategy.GetCallCount() != 1 {
			t.Errorf("expected call count 1, got %d", strategy.GetCallCount())
		}
	})

	t.Run("failed recovery", func(t *testing.T) {
		failureErr := fmt.Errorf("recovery failed")
		strategy := NewMockRecoveryStrategy("failure_strategy").WithFailure(failureErr)
		testErr := NewConfigError(ErrInvalidFormat, "test error")

		result, err := strategy.Apply(testErr, nil)
		if err == nil {
			t.Error("expected recovery to fail")
		}

		if err.Error() != failureErr.Error() {
			t.Errorf("expected error %q, got %q", failureErr.Error(), err.Error())
		}

		if result != nil {
			t.Errorf("expected nil result on failure, got %v", result)
		}
	})

	t.Run("custom apply function", func(t *testing.T) {
		strategy := NewMockRecoveryStrategy("custom_strategy")
		testContext := "test_context"

		strategy.WithApply(func(err OutputError, context any) (any, error) {
			return fmt.Sprintf("recovered_%s_%s", err.Code(), context), nil
		})

		testErr := NewConfigError(ErrInvalidFormat, "test error")
		result, err := strategy.Apply(testErr, testContext)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		expected := fmt.Sprintf("recovered_%s_%s", ErrInvalidFormat, testContext)
		if result != expected {
			t.Errorf("expected %q, got %v", expected, result)
		}
	})

	t.Run("custom applicable function", func(t *testing.T) {
		strategy := NewMockRecoveryStrategy("selective_strategy")

		strategy.WithApplicableFor(func(err OutputError) bool {
			return err.Code() == ErrInvalidFormat
		})

		// Should be applicable for ErrInvalidFormat
		formatErr := NewConfigError(ErrInvalidFormat, "format error")
		if !strategy.ApplicableFor(formatErr) {
			t.Error("expected strategy to be applicable for format error")
		}

		// Should not be applicable for other errors
		otherErr := NewConfigError(ErrMissingRequired, "other error")
		if strategy.ApplicableFor(otherErr) {
			t.Error("expected strategy not to be applicable for other error")
		}
	})

	t.Run("priority configuration", func(t *testing.T) {
		strategy := NewMockRecoveryStrategy("priority_strategy").WithPriority(5)

		if strategy.Priority() != 5 {
			t.Errorf("expected priority 5, got %d", strategy.Priority())
		}
	})

	t.Run("state tracking", func(t *testing.T) {
		strategy := NewMockRecoveryStrategy("tracking_strategy")
		testErr := NewConfigError(ErrInvalidFormat, "test error")
		testContext := "test_context"

		strategy.Apply(testErr, testContext)

		if strategy.GetLastError() != testErr {
			t.Errorf("expected last error %v, got %v", testErr, strategy.GetLastError())
		}

		if strategy.GetLastContext() != testContext {
			t.Errorf("expected last context %v, got %v", testContext, strategy.GetLastContext())
		}

		// Reset and verify
		strategy.Reset()
		if strategy.GetCallCount() != 0 {
			t.Errorf("expected call count 0 after reset, got %d", strategy.GetCallCount())
		}
		if strategy.GetLastError() != nil {
			t.Errorf("expected nil last error after reset, got %v", strategy.GetLastError())
		}
	})
}

func TestErrorScenarioRunner(t *testing.T) {
	t.Run("successful scenario", func(t *testing.T) {
		runner := NewErrorScenarioRunner()

		scenario := ErrorScenario{
			Name:        "success_scenario",
			Description: "A scenario that should succeed",
			Setup: func(injector *ErrorInjector) {
				// No error injection for success scenario
			},
			Validators: []Validator{
				NewMockValidator("success_validator").WithSuccess(),
			},
			Expected: ExpectedResults{
				ShouldFail: false,
			},
		}

		runner.AddScenario(scenario)
		result := runner.RunScenario(scenario)

		if !result.Success {
			t.Errorf("expected scenario to succeed, got failure: %s", result.ErrorMessage)
		}

		if result.ActualError != nil {
			t.Errorf("expected no error, got %v", result.ActualError)
		}
	})

	t.Run("failure scenario", func(t *testing.T) {
		runner := NewErrorScenarioRunner()
		testErr := NewValidationError(ErrMissingColumn, "test validation failure")

		scenario := ErrorScenario{
			Name:        "failure_scenario",
			Description: "A scenario that should fail",
			Validators: []Validator{
				NewMockValidator("failing_validator").WithFailure(testErr),
			},
			Expected: ExpectedResults{
				ShouldFail:        true,
				ExpectedErrorCode: ErrMissingColumn,
				ExpectedSeverity:  SeverityError,
			},
		}

		result := runner.RunScenario(scenario)

		if !result.Success {
			t.Errorf("expected scenario evaluation to succeed, got failure: %s", result.ErrorMessage)
		}

		if result.ActualError == nil {
			t.Error("expected actual error to be present")
		}
	})

	t.Run("recovery scenario", func(t *testing.T) {
		runner := NewErrorScenarioRunner()
		testErr := NewConfigError(ErrInvalidFormat, "format error")

		scenario := ErrorScenario{
			Name:        "recovery_scenario",
			Description: "A scenario that should recover",
			Validators: []Validator{
				NewMockValidator("failing_validator").WithFailure(testErr),
			},
			Recovery: []RecoveryStrategy{
				NewMockRecoveryStrategy("test_recovery").WithSuccess(),
			},
			Expected: ExpectedResults{
				ShouldFail:       false, // Should not fail due to recovery
				ShouldRecover:    true,
				RecoveryStrategy: "test_recovery",
			},
		}

		result := runner.RunScenario(scenario)

		if !result.Success {
			t.Errorf("expected scenario to succeed, got failure: %s", result.ErrorMessage)
		}

		if result.RecoveryUsed != "test_recovery" {
			t.Errorf("expected recovery strategy 'test_recovery', got %q", result.RecoveryUsed)
		}
	})

	t.Run("multiple scenarios", func(t *testing.T) {
		runner := NewErrorScenarioRunner()

		scenarios := []ErrorScenario{
			{
				Name: "scenario1",
				Validators: []Validator{
					NewMockValidator("validator1").WithSuccess(),
				},
				Expected: ExpectedResults{ShouldFail: false},
			},
			{
				Name: "scenario2",
				Validators: []Validator{
					NewMockValidator("validator2").WithFailure(fmt.Errorf("test error")),
				},
				Expected: ExpectedResults{ShouldFail: true},
			},
		}

		for _, scenario := range scenarios {
			runner.AddScenario(scenario)
		}

		results := runner.RunAllScenarios()

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		successful := runner.GetSuccessfulScenarios()
		failed := runner.GetFailedScenarios()

		if len(successful) != 2 {
			t.Errorf("expected 2 successful scenarios, got %d", len(successful))
		}

		if len(failed) != 0 {
			t.Errorf("expected 0 failed scenarios, got %d", len(failed))
		}
	})
}

func TestTestDataBuilder(t *testing.T) {
	t.Run("basic data building", func(t *testing.T) {
		builder := NewTestDataBuilder()

		output := builder.
			WithKeys("Name", "Age", "City").
			WithRow(map[string]any{"Name": "John", "Age": 30, "City": "NYC"}).
			WithRow(map[string]any{"Name": "Jane", "Age": 25, "City": "LA"}).
			WithFormat("json").
			Build()

		if len(output.Keys) != 3 {
			t.Errorf("expected 3 keys, got %d", len(output.Keys))
		}

		if len(output.Contents) != 2 {
			t.Errorf("expected 2 rows, got %d", len(output.Contents))
		}

		if output.Settings.OutputFormat != "json" {
			t.Errorf("expected format 'json', got %q", output.Settings.OutputFormat)
		}
	})

	t.Run("empty and nil rows", func(t *testing.T) {
		builder := NewTestDataBuilder()

		output := builder.
			WithKeys("Name").
			WithEmptyRow().
			WithNilRow().
			Build()

		if len(output.Contents) != 2 {
			t.Errorf("expected 2 rows, got %d", len(output.Contents))
		}

		// First row should have empty contents
		if output.Contents[0].Contents == nil {
			t.Error("expected empty row to have non-nil contents")
		}
		if len(output.Contents[0].Contents) != 0 {
			t.Errorf("expected empty row to have 0 contents, got %d", len(output.Contents[0].Contents))
		}

		// Second row should have nil contents
		if output.Contents[1].Contents != nil {
			t.Error("expected nil row to have nil contents")
		}
	})
}

func TestErrorTypeHelper(t *testing.T) {
	helper := NewErrorTypeHelper()

	t.Run("configuration errors", func(t *testing.T) {
		errors := helper.CreateConfigurationErrors()

		if len(errors) == 0 {
			t.Error("expected configuration errors to be created")
		}

		// Verify all are configuration errors (1xxx codes)
		for _, err := range errors {
			code := string(err.Code())
			if !strings.HasPrefix(code, "OUT-1") {
				t.Errorf("expected configuration error code (OUT-1xxx), got %s", code)
			}
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		errors := helper.CreateValidationErrors()

		if len(errors) == 0 {
			t.Error("expected validation errors to be created")
		}

		// Verify all are validation errors (2xxx codes)
		for _, err := range errors {
			code := string(err.Code())
			if !strings.HasPrefix(code, "OUT-2") {
				t.Errorf("expected validation error code (OUT-2xxx), got %s", code)
			}
		}
	})

	t.Run("processing errors", func(t *testing.T) {
		errors := helper.CreateProcessingErrors()

		if len(errors) == 0 {
			t.Error("expected processing errors to be created")
		}

		// Verify all are processing errors (3xxx codes)
		for _, err := range errors {
			code := string(err.Code())
			if !strings.HasPrefix(code, "OUT-3") {
				t.Errorf("expected processing error code (OUT-3xxx), got %s", code)
			}
		}
	})

	t.Run("runtime errors", func(t *testing.T) {
		errors := helper.CreateRuntimeErrors()

		if len(errors) == 0 {
			t.Error("expected runtime errors to be created")
		}

		// Verify all are runtime errors (4xxx codes)
		for _, err := range errors {
			code := string(err.Code())
			if !strings.HasPrefix(code, "OUT-4") {
				t.Errorf("expected runtime error code (OUT-4xxx), got %s", code)
			}
		}
	})

	t.Run("errors with severity", func(t *testing.T) {
		errorsBySeverity := helper.CreateErrorsWithSeverity()

		for severity, errors := range errorsBySeverity {
			if len(errors) == 0 {
				t.Errorf("expected errors for severity %v", severity)
				continue
			}

			for _, err := range errors {
				if err.Severity() != severity {
					t.Errorf("expected severity %v, got %v", severity, err.Severity())
				}
			}
		}
	})
}

func TestPerformanceTestHelper(t *testing.T) {
	t.Run("operation measurement", func(t *testing.T) {
		helper := NewPerformanceTestHelper()

		err := helper.MeasureOperation("test_operation", func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		duration := helper.GetMeasurement("test_operation")
		if duration < 10*time.Millisecond {
			t.Errorf("expected duration >= 10ms, got %v", duration)
		}
	})

	t.Run("error measurement", func(t *testing.T) {
		helper := NewPerformanceTestHelper()
		testErr := fmt.Errorf("test error")

		err := helper.MeasureOperation("error_operation", func() error {
			time.Sleep(5 * time.Millisecond)
			return testErr
		})

		if err != testErr {
			t.Errorf("expected test error, got %v", err)
		}

		duration := helper.GetMeasurement("error_operation")
		if duration < 5*time.Millisecond {
			t.Errorf("expected duration >= 5ms, got %v", duration)
		}
	})

	t.Run("performance overhead verification", func(t *testing.T) {
		helper := NewPerformanceTestHelper()

		// Measure baseline operation
		helper.MeasureOperation("baseline", func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})

		// Measure operation with overhead
		helper.MeasureOperation("with_overhead", func() error {
			time.Sleep(11 * time.Millisecond) // 10% overhead
			return nil
		})

		// Should pass with 20% max overhead
		if !helper.VerifyPerformanceOverhead("baseline", "with_overhead", 20.0) {
			t.Error("expected performance overhead verification to pass")
		}

		// Should fail with 5% max overhead
		if helper.VerifyPerformanceOverhead("baseline", "with_overhead", 5.0) {
			t.Error("expected performance overhead verification to fail")
		}
	})

	t.Run("all measurements", func(t *testing.T) {
		helper := NewPerformanceTestHelper()

		helper.MeasureOperation("op1", func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})

		helper.MeasureOperation("op2", func() error {
			time.Sleep(2 * time.Millisecond)
			return nil
		})

		measurements := helper.GetAllMeasurements()
		if len(measurements) != 2 {
			t.Errorf("expected 2 measurements, got %d", len(measurements))
		}

		if _, exists := measurements["op1"]; !exists {
			t.Error("expected op1 measurement to exist")
		}

		if _, exists := measurements["op2"]; !exists {
			t.Error("expected op2 measurement to exist")
		}
	})

	t.Run("reset functionality", func(t *testing.T) {
		helper := NewPerformanceTestHelper()

		helper.MeasureOperation("test_op", func() error {
			return nil
		})

		if len(helper.GetAllMeasurements()) != 1 {
			t.Error("expected 1 measurement before reset")
		}

		helper.Reset()

		if len(helper.GetAllMeasurements()) != 0 {
			t.Error("expected 0 measurements after reset")
		}
	})
}

func TestIntegrationScenarios(t *testing.T) {
	t.Run("complete error handling pipeline", func(t *testing.T) {
		// Create error injection setup
		injector := NewErrorInjector()
		validationErr := NewValidationError(ErrMissingColumn, "missing required column")
		injector.InjectValidationError("validation", validationErr)

		// Create mock components
		validator := NewMockValidator("pipeline_validator").
			WithFailure(validationErr)

		handler := NewMockErrorHandler().
			WithMode(ErrorModeLenient)

		recovery := NewMockRecoveryStrategy("pipeline_recovery").
			WithSuccess()

		// Create scenario
		scenario := ErrorScenario{
			Name:        "complete_pipeline",
			Description: "Test complete error handling pipeline",
			Validators:  []Validator{validator},
			Handler:     handler,
			Recovery:    []RecoveryStrategy{recovery},
			Expected: ExpectedResults{
				ShouldFail:    false, // Should not fail due to recovery
				ShouldRecover: true,
			},
		}

		// Run scenario
		runner := NewErrorScenarioRunner()
		result := runner.RunScenario(scenario)

		if !result.Success {
			t.Errorf("expected pipeline scenario to succeed, got: %s", result.ErrorMessage)
		}

		// Verify all components were called
		if validator.GetCallCount() != 1 {
			t.Errorf("expected validator to be called once, got %d", validator.GetCallCount())
		}

		if handler.GetCallCount() != 1 {
			t.Errorf("expected handler to be called once, got %d", handler.GetCallCount())
		}

		if recovery.GetCallCount() != 1 {
			t.Errorf("expected recovery to be called once, got %d", recovery.GetCallCount())
		}
	})

	t.Run("performance impact testing", func(t *testing.T) {
		helper := NewPerformanceTestHelper()

		// Measure baseline operation (no error handling)
		helper.MeasureOperation("baseline", func() error {
			// Simulate some work
			for i := 0; i < 1000; i++ {
				_ = fmt.Sprintf("test_%d", i)
			}
			return nil
		})

		// Measure operation with error handling
		helper.MeasureOperation("with_error_handling", func() error {
			// Simulate same work plus error handling overhead
			for i := 0; i < 1000; i++ {
				_ = fmt.Sprintf("test_%d", i)
			}

			// Simulate error handling overhead
			validator := NewMockValidator("perf_validator").WithSuccess()
			validator.Validate("test_data")

			handler := NewMockErrorHandler()
			handler.HandleError(nil)

			return nil
		})

		// Verify error handling overhead is within acceptable limits (< 1% as per requirements)
		if !helper.VerifyPerformanceOverhead("baseline", "with_error_handling", 1.0) {
			baselineDuration := helper.GetMeasurement("baseline")
			errorHandlingDuration := helper.GetMeasurement("with_error_handling")
			overhead := float64(errorHandlingDuration-baselineDuration) / float64(baselineDuration) * 100

			t.Errorf("error handling overhead %.2f%% exceeds 1%% requirement", overhead)
		}
	})
}
