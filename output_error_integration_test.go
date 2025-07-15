package format

import (
	"testing"

	"github.com/ArjenSchwarz/go-output/errors"
	"github.com/ArjenSchwarz/go-output/validators"
)

// Test OutputArray error integration interfaces
func TestOutputArrayErrorIntegration_Interface(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")

	output := &OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Value"},
	}

	// Test that OutputArray has validator fields and methods
	if output.validators == nil {
		output.validators = make([]validators.Validator, 0)
	}
	if output.errorHandler == nil {
		output.errorHandler = errors.NewDefaultErrorHandler()
	}

	// Test AddValidator method exists
	validator := validators.NewNotEmptyValidator()
	output.AddValidator(validator)

	// Test WithErrorHandler method exists
	handler := errors.NewDefaultErrorHandler()
	handler.SetMode(errors.ErrorModeLenient)
	result := output.WithErrorHandler(handler)
	if result != output {
		t.Error("WithErrorHandler should return the same OutputArray instance")
	}
}

// Test Validate method
func TestOutputArray_Validate(t *testing.T) {
	testCases := []struct {
		name           string
		setupOutput    func() *OutputArray
		shouldFail     bool
		expectedErrors int
	}{
		{
			name: "valid output with data",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings:     settings,
					Keys:         []string{"Name", "Value"},
					Contents:     []OutputHolder{{Contents: map[string]interface{}{"Name": "test", "Value": 123}}},
					validators:   make([]validators.Validator, 0),
					errorHandler: errors.NewDefaultErrorHandler(),
				}
				output.AddValidator(validators.NewNotEmptyValidator())
				return output
			},
			shouldFail:     false,
			expectedErrors: 0,
		},
		{
			name: "empty dataset with NotEmptyValidator",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings:     settings,
					Keys:         []string{"Name", "Value"},
					Contents:     []OutputHolder{},
					validators:   make([]validators.Validator, 0),
					errorHandler: errors.NewDefaultErrorHandler(),
				}
				output.AddValidator(validators.NewNotEmptyValidator())
				return output
			},
			shouldFail:     true,
			expectedErrors: 1,
		},
		{
			name: "missing required columns",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings:     settings,
					Keys:         []string{"Name"}, // Missing "Value" column
					Contents:     []OutputHolder{{Contents: map[string]interface{}{"Name": "test"}}},
					validators:   make([]validators.Validator, 0),
					errorHandler: errors.NewDefaultErrorHandler(),
				}
				output.AddValidator(validators.NewRequiredColumnsValidator("Name", "Value"))
				return output
			},
			shouldFail:     true,
			expectedErrors: 1,
		},
		{
			name: "invalid settings format",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("invalid-format")
				output := &OutputArray{
					Settings:     settings,
					Keys:         []string{"Name", "Value"},
					Contents:     []OutputHolder{{Contents: map[string]interface{}{"Name": "test", "Value": 123}}},
					validators:   make([]validators.Validator, 0),
					errorHandler: errors.NewDefaultErrorHandler(),
				}
				return output
			},
			shouldFail:     true,
			expectedErrors: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.setupOutput()

			err := output.Validate()

			if tc.shouldFail {
				if err == nil {
					t.Error("Expected validation to fail")
					return
				}

				// Check error type and count
				if outputErr, ok := err.(errors.OutputError); ok {
					if compositeErr, ok := outputErr.(errors.CompositeError); ok {
						// For composite errors, check if it has errors
						if !compositeErr.HasErrors() {
							t.Errorf("Expected validation errors but composite error is empty")
						}
					} else {
						// For single validation errors, just check that we have an error (which we do)
						if tc.expectedErrors > 1 {
							t.Errorf("Expected composite error with %d errors, got single error", tc.expectedErrors)
						}
					}
				} else {
					t.Errorf("Expected OutputError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass, got error: %v", err)
				}
			}
		})
	}
}

// Test WriteWithValidation method (new error-returning API)
func TestOutputArray_WriteWithValidation(t *testing.T) {
	testCases := []struct {
		name        string
		setupOutput func() *OutputArray
		shouldFail  bool
	}{
		{
			name: "successful write with validation",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings:     settings,
					Keys:         []string{"Name", "Value"},
					Contents:     []OutputHolder{{Contents: map[string]interface{}{"Name": "test", "Value": 123}}},
					validators:   make([]validators.Validator, 0),
					errorHandler: errors.NewDefaultErrorHandler(),
				}
				output.AddValidator(validators.NewNotEmptyValidator())
				return output
			},
			shouldFail: false,
		},
		{
			name: "write fails on validation error",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings:     settings,
					Keys:         []string{"Name", "Value"},
					Contents:     []OutputHolder{}, // Empty dataset
					validators:   make([]validators.Validator, 0),
					errorHandler: errors.NewDefaultErrorHandler(),
				}
				output.AddValidator(validators.NewNotEmptyValidator())
				return output
			},
			shouldFail: true,
		},
		{
			name: "lenient mode collects errors but continues",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				handler := errors.NewDefaultErrorHandler()
				handler.SetMode(errors.ErrorModeLenient)
				output := &OutputArray{
					Settings:     settings,
					Keys:         []string{"Name"},
					Contents:     []OutputHolder{{Contents: map[string]interface{}{"Name": "test"}}},
					validators:   make([]validators.Validator, 0),
					errorHandler: handler,
				}
				output.AddValidator(validators.NewRequiredColumnsValidator("Name", "MissingColumn"))
				return output
			},
			shouldFail: false, // Lenient mode should not fail immediately
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.setupOutput()

			err := output.WriteWithValidation()

			if tc.shouldFail {
				if err == nil {
					t.Error("Expected WriteWithValidation to fail")
				}
			} else {
				if err != nil {
					t.Errorf("Expected WriteWithValidation to succeed, got error: %v", err)
				}
			}
		})
	}
}

// Test WriteCompat method (backward compatibility)
func TestOutputArray_WriteCompat(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")
	output := &OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Value"},
		Contents: []OutputHolder{{Contents: map[string]interface{}{"Name": "test", "Value": 123}}},
	}

	// Test that WriteCompat doesn't panic and behaves like old Write
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WriteCompat should not panic: %v", r)
		}
	}()

	output.WriteCompat()
}

// Test error handler integration
func TestOutputArray_ErrorHandlerIntegration(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")

	// Create handler with lenient mode
	handler := errors.NewDefaultErrorHandler()
	handler.SetMode(errors.ErrorModeLenient)

	output := &OutputArray{
		Settings:     settings,
		Keys:         []string{"Name"},
		Contents:     []OutputHolder{{Contents: map[string]interface{}{"Name": "test"}}},
		validators:   make([]validators.Validator, 0),
		errorHandler: handler,
	}

	// Add validator that will fail
	output.AddValidator(validators.NewRequiredColumnsValidator("Name", "MissingColumn"))

	// Validate should collect error but not fail immediately
	err := output.Validate()
	if err != nil {
		t.Errorf("Lenient mode should not return error immediately: %v", err)
	}

	// Check that error was collected
	summary := handler.GetSummary()
	if summary.TotalErrors == 0 {
		t.Error("Expected errors to be collected in lenient mode")
	}
}

// Test validator chain with error recovery
func TestOutputArray_ValidatorChainWithRecovery(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("table") // Valid format that can fallback

	// Create recovery handler with format fallback
	recovery := errors.NewDefaultRecoveryHandler()
	recovery.AddStrategy(errors.NewFormatFallbackStrategy([]string{"table", "csv", "json"}))

	// Create error handler with recovery
	handler := errors.NewDefaultErrorHandler()
	handler.SetMode(errors.ErrorModeLenient)

	output := &OutputArray{
		Settings:        settings,
		Keys:            []string{"Name", "Value"},
		Contents:        []OutputHolder{{Contents: map[string]interface{}{"Name": "test", "Value": 123}}},
		validators:      make([]validators.Validator, 0),
		errorHandler:    handler,
		recoveryHandler: recovery,
	}

	// Add validators
	output.AddValidator(validators.NewNotEmptyValidator())
	output.AddValidator(validators.NewRequiredColumnsValidator("Name", "Value"))

	// Test validation passes
	err := output.Validate()
	if err != nil {
		t.Errorf("Validation should pass: %v", err)
	}

	// Test write succeeds
	err = output.WriteWithValidation()
	if err != nil {
		t.Errorf("Write should succeed: %v", err)
	}
}

// Test validation context integration
func TestOutputArray_ValidationContext(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")

	output := &OutputArray{
		Settings:     settings,
		Keys:         []string{"Name"},
		Contents:     []OutputHolder{}, // Empty to trigger validation error
		validators:   make([]validators.Validator, 0),
		errorHandler: errors.NewDefaultErrorHandler(),
	}

	output.AddValidator(validators.NewNotEmptyValidator())

	err := output.Validate()
	if err == nil {
		t.Error("Expected validation to fail for empty dataset")
		return
	}

	// Check that error contains context
	if outputErr, ok := err.(errors.OutputError); ok {
		context := outputErr.Context()
		if context.Operation == "" {
			t.Error("Expected error to contain operation context")
		}
	}
}

// Test multiple validator integration
func TestOutputArray_MultipleValidators(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")

	// Create output with data that will trigger multiple validation errors
	output := &OutputArray{
		Settings:     settings,
		Keys:         []string{"Name"}, // Missing required column
		Contents:     []OutputHolder{}, // Empty dataset
		validators:   make([]validators.Validator, 0),
		errorHandler: errors.NewDefaultErrorHandler(),
	}

	// Add multiple validators
	output.AddValidator(validators.NewNotEmptyValidator())
	output.AddValidator(validators.NewRequiredColumnsValidator("Name", "Value"))

	err := output.Validate()
	if err == nil {
		t.Error("Expected validation to fail")
		return
	}

	// In strict mode, should get first error
	if outputErr, ok := err.(errors.OutputError); ok {
		if outputErr.Code() != errors.ErrEmptyDataset {
			t.Errorf("Expected first error to be ErrEmptyDataset, got %s", outputErr.Code())
		}
	}

	// Test lenient mode to collect all errors
	handler := errors.NewDefaultErrorHandler()
	handler.SetMode(errors.ErrorModeLenient)
	output.errorHandler = handler

	err = output.Validate()
	// In lenient mode, should collect errors but not return immediately
	summary := handler.GetSummary()
	if summary.TotalErrors < 2 {
		t.Errorf("Expected at least 2 errors collected, got %d", summary.TotalErrors)
	}
}
