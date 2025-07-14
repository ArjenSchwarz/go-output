package format

import (
	"strings"
	"testing"
)

// TestOutputArrayValidation tests the Validate method
func TestOutputArrayValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupOutput func() *OutputArray
		expectError bool
		errorCode   ErrorCode
	}{
		{
			name: "valid output array with json format",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": 123}},
					},
				}
			},
			expectError: false,
		},
		{
			name: "missing settings",
			setupOutput: func() *OutputArray {
				return &OutputArray{
					Settings: nil,
					Keys:     []string{"Name"},
				}
			},
			expectError: true,
			errorCode:   ErrMissingRequired,
		},
		{
			name: "mermaid format without required configuration",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
				}
			},
			expectError: true,
			errorCode:   ErrMissingRequired,
		},
		{
			name: "dot format without FromToColumns",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("dot")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
				}
			},
			expectError: true,
			errorCode:   ErrMissingRequired,
		},
		{
			name: "valid mermaid format with FromToColumns",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid")
				settings.AddFromToColumns("From", "To")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"From", "To"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"From": "A", "To": "B"}},
					},
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()
			err := output.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if outputErr, ok := err.(OutputError); ok {
					if outputErr.Code() != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, outputErr.Code())
					}
				} else {
					t.Errorf("expected OutputError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestOutputArrayWithValidators tests validation with custom validators
func TestOutputArrayWithValidators(t *testing.T) {
	tests := []struct {
		name        string
		setupOutput func() *OutputArray
		expectError bool
		errorCode   ErrorCode
	}{
		{
			name: "required columns validator - success",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value", "Status"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": 123, "Status": "active"}},
					},
				}
				output.AddValidator(NewRequiredColumnsValidator("Name", "Value"))
				return output
			},
			expectError: false,
		},
		{
			name: "required columns validator - missing column",
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
				output.AddValidator(NewRequiredColumnsValidator("Name", "Value", "MissingColumn"))
				return output
			},
			expectError: true,
			errorCode:   ErrMissingColumn,
		},
		{
			name: "data type validator - success",
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
				validator := NewDataTypeValidator().
					WithStringColumn("Name").
					WithIntColumn("Value")
				output.AddValidator(validator)
				return output
			},
			expectError: false,
		},
		{
			name: "data type validator - type mismatch",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": "not_a_number"}},
					},
				}
				validator := NewDataTypeValidator().
					WithStringColumn("Name").
					WithIntColumn("Value")
				output.AddValidator(validator)
				return output
			},
			expectError: true,
			errorCode:   ErrInvalidDataType,
		},
		{
			name: "empty dataset validator - allow empty",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
					Contents: []OutputHolder{}, // Empty
				}
				output.AddValidator(NewEmptyDatasetValidator(true))
				return output
			},
			expectError: false,
		},
		{
			name: "empty dataset validator - disallow empty",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
					Contents: []OutputHolder{}, // Empty
				}
				output.AddValidator(NewEmptyDatasetValidator(false))
				return output
			},
			expectError: true,
			errorCode:   ErrEmptyDataset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()
			err := output.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if outputErr, ok := err.(OutputError); ok {
					if outputErr.Code() != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, outputErr.Code())
					}
				} else {
					t.Errorf("expected OutputError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestOutputArrayErrorHandling tests different error handling modes
func TestOutputArrayErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		errorMode    ErrorMode
		setupOutput  func() *OutputArray
		expectError  bool
		expectStrict bool // Whether strict mode should fail
	}{
		{
			name:      "strict mode - validation error",
			errorMode: ErrorModeStrict,
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
				output.AddValidator(NewRequiredColumnsValidator("Name", "Value", "MissingColumn"))
				return output
			},
			expectError:  true,
			expectStrict: true,
		},
		{
			name:      "lenient mode - validation error",
			errorMode: ErrorModeLenient,
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
				output.AddValidator(NewRequiredColumnsValidator("Name", "Value", "MissingColumn"))
				return output
			},
			expectError:  false, // Lenient mode should collect errors but not fail
			expectStrict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()
			handler := NewErrorHandlerWithMode(tt.errorMode)
			output.WithErrorHandler(handler)

			err := output.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}

			// Check collected errors in lenient mode
			if tt.errorMode == ErrorModeLenient {
				collectedErrors := handler.GetCollectedErrors()
				if len(collectedErrors) == 0 {
					t.Errorf("expected collected errors in lenient mode")
				}
			}
		})
	}
}

// TestOutputArrayWrite tests the Write method with error handling
func TestOutputArrayWrite(t *testing.T) {
	tests := []struct {
		name        string
		setupOutput func() *OutputArray
		expectError bool
		errorCode   ErrorCode
	}{
		{
			name: "successful write - json format",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": 123}},
					},
				}
			},
			expectError: false,
		},
		{
			name: "write with validation error",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid") // Requires FromToColumns
				return &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Value"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "test", "Value": 123}},
					},
				}
			},
			expectError: true,
			errorCode:   ErrMissingRequired,
		},
		{
			name: "write with custom validator error",
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
				output.AddValidator(NewRequiredColumnsValidator("Name", "Value", "MissingColumn"))
				return output
			},
			expectError: true,
			errorCode:   ErrMissingColumn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()
			err := output.Write()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if outputErr, ok := err.(OutputError); ok {
					if outputErr.Code() != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, outputErr.Code())
					}
				} else {
					t.Errorf("expected OutputError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestOutputArrayAddValidator tests the AddValidator method
func TestOutputArrayAddValidator(t *testing.T) {
	output := &OutputArray{
		Settings: NewOutputSettings(),
		Keys:     []string{"Name", "Value"},
	}

	// Initially no validators
	if len(output.validators) != 0 {
		t.Errorf("expected 0 validators initially, got %d", len(output.validators))
	}

	// Add first validator
	validator1 := NewRequiredColumnsValidator("Name")
	result := output.AddValidator(validator1)

	// Should return the same OutputArray for chaining
	if result != output {
		t.Errorf("AddValidator should return the same OutputArray for chaining")
	}

	if len(output.validators) != 1 {
		t.Errorf("expected 1 validator after adding, got %d", len(output.validators))
	}

	// Add second validator
	validator2 := NewEmptyDatasetValidator(false)
	output.AddValidator(validator2)

	if len(output.validators) != 2 {
		t.Errorf("expected 2 validators after adding second, got %d", len(output.validators))
	}
}

// TestOutputArrayWithErrorHandler tests the WithErrorHandler method
func TestOutputArrayWithErrorHandler(t *testing.T) {
	output := &OutputArray{
		Settings: NewOutputSettings(),
		Keys:     []string{"Name", "Value"},
	}

	// Initially no error handler
	if output.errorHandler != nil {
		t.Errorf("expected no error handler initially")
	}

	// Set error handler
	handler := NewErrorHandlerWithMode(ErrorModeLenient)
	result := output.WithErrorHandler(handler)

	// Should return the same OutputArray for chaining
	if result != output {
		t.Errorf("WithErrorHandler should return the same OutputArray for chaining")
	}

	if output.errorHandler != handler {
		t.Errorf("error handler was not set correctly")
	}
}

// TestOutputArrayConstraintValidator tests constraint validation
func TestOutputArrayConstraintValidator(t *testing.T) {
	tests := []struct {
		name        string
		setupOutput func() *OutputArray
		expectError bool
		errorCode   ErrorCode
	}{
		{
			name: "positive number constraint - success",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Price"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "item1", "Price": 10.5}},
						{Contents: map[string]interface{}{"Name": "item2", "Price": 25}},
					},
				}
				validator := NewConstraintValidator().
					AddConstraint(PositiveNumberConstraint("Price"))
				output.AddValidator(validator)
				return output
			},
			expectError: false,
		},
		{
			name: "positive number constraint - violation",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Price"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "item1", "Price": 10.5}},
						{Contents: map[string]interface{}{"Name": "item2", "Price": -5}}, // Negative price
					},
				}
				validator := NewConstraintValidator().
					AddConstraint(PositiveNumberConstraint("Price"))
				output.AddValidator(validator)
				return output
			},
			expectError: true,
			errorCode:   ErrConstraintViolation,
		},
		{
			name: "non-empty string constraint - success",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Description"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "item1", "Description": "A good item"}},
						{Contents: map[string]interface{}{"Name": "item2", "Description": "Another item"}},
					},
				}
				validator := NewConstraintValidator().
					AddConstraint(NonEmptyStringConstraint("Description"))
				output.AddValidator(validator)
				return output
			},
			expectError: false,
		},
		{
			name: "non-empty string constraint - violation",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Description"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "item1", "Description": "A good item"}},
						{Contents: map[string]interface{}{"Name": "item2", "Description": ""}}, // Empty description
					},
				}
				validator := NewConstraintValidator().
					AddConstraint(NonEmptyStringConstraint("Description"))
				output.AddValidator(validator)
				return output
			},
			expectError: true,
			errorCode:   ErrConstraintViolation,
		},
		{
			name: "range constraint - success",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Score"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "student1", "Score": 85.5}},
						{Contents: map[string]interface{}{"Name": "student2", "Score": 92}},
					},
				}
				validator := NewConstraintValidator().
					AddConstraint(RangeConstraint("Score", 0, 100))
				output.AddValidator(validator)
				return output
			},
			expectError: false,
		},
		{
			name: "range constraint - violation",
			setupOutput: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				output := &OutputArray{
					Settings: settings,
					Keys:     []string{"Name", "Score"},
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{"Name": "student1", "Score": 85.5}},
						{Contents: map[string]interface{}{"Name": "student2", "Score": 105}}, // Out of range
					},
				}
				validator := NewConstraintValidator().
					AddConstraint(RangeConstraint("Score", 0, 100))
				output.AddValidator(validator)
				return output
			},
			expectError: true,
			errorCode:   ErrConstraintViolation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupOutput()
			err := output.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if outputErr, ok := err.(OutputError); ok {
					if outputErr.Code() != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, outputErr.Code())
					}
				} else {
					t.Errorf("expected OutputError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestOutputArrayChainedValidation tests chaining multiple validators
func TestOutputArrayChainedValidation(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")
	output := &OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Price", "Description"},
		Contents: []OutputHolder{
			{Contents: map[string]interface{}{"Name": "item1", "Price": 10.5, "Description": "Good item"}},
			{Contents: map[string]interface{}{"Name": "item2", "Price": -5, "Description": ""}}, // Multiple violations
		},
	}

	// Add multiple validators
	output.AddValidator(NewRequiredColumnsValidator("Name", "Price", "Description")).
		AddValidator(NewConstraintValidator().
			AddConstraint(PositiveNumberConstraint("Price")).
			AddConstraint(NonEmptyStringConstraint("Description")))

	err := output.Validate()
	if err == nil {
		t.Errorf("expected error due to constraint violations")
		return
	}

	// Should get the first error (constraint violation)
	if outputErr, ok := err.(OutputError); ok {
		if outputErr.Code() != ErrConstraintViolation {
			t.Errorf("expected error code %s, got %s", ErrConstraintViolation, outputErr.Code())
		}
	} else {
		t.Errorf("expected OutputError, got %T", err)
	}
}

// TestOutputArrayLenientModeCollectsErrors tests that lenient mode collects all errors
func TestOutputArrayLenientModeCollectsErrors(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")
	output := &OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Price"},
		Contents: []OutputHolder{
			{Contents: map[string]interface{}{"Name": "item1", "Price": -5}}, // Negative price
		},
	}

	// Add validator that will fail
	output.AddValidator(NewConstraintValidator().
		AddConstraint(PositiveNumberConstraint("Price")))

	// Use lenient mode
	handler := NewErrorHandlerWithMode(ErrorModeLenient)
	output.WithErrorHandler(handler)

	err := output.Validate()

	// In lenient mode, validation should not return an error for non-fatal issues
	if err != nil {
		t.Errorf("expected no error in lenient mode, got: %v", err)
	}

	// But errors should be collected
	collectedErrors := handler.GetCollectedErrors()
	if len(collectedErrors) == 0 {
		t.Errorf("expected collected errors in lenient mode")
	}

	// Check error summary
	summary := handler.Summary()
	if summary.TotalErrors == 0 {
		t.Errorf("expected errors in summary")
	}
	if summary.ByCategory[ErrConstraintViolation] == 0 {
		t.Errorf("expected constraint violation errors in summary")
	}
}

// TestOutputArrayErrorMessages tests error message formatting
func TestOutputArrayErrorMessages(t *testing.T) {
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
	output.AddValidator(NewRequiredColumnsValidator("Name", "Value", "MissingColumn"))

	err := output.Validate()
	if err == nil {
		t.Errorf("expected error but got none")
		return
	}

	errorMsg := err.Error()

	// Check that error message contains expected elements
	expectedElements := []string{
		"OUT-2001", // Error code
		"missing required columns",
		"MissingColumn",
		"Suggestions:",
	}

	for _, element := range expectedElements {
		if !strings.Contains(errorMsg, element) {
			t.Errorf("error message should contain '%s', got: %s", element, errorMsg)
		}
	}
}
