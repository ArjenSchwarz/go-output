package format

import (
	"testing"

	"github.com/ArjenSchwarz/go-output/errors"
)

// Test basic OutputArray error integration without validators
func TestOutputArray_BasicErrorIntegration(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")

	output := &OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Value"},
		Contents: []OutputHolder{{Contents: map[string]interface{}{"Name": "test", "Value": 123}}},
	}

	// Test that basic fields can be set
	output.errorHandler = errors.NewDefaultErrorHandler()

	// Test that methods exist and work
	handler := errors.NewDefaultErrorHandler()
	result := output.WithErrorHandler(handler)
	if result != output {
		t.Error("WithErrorHandler should return the same OutputArray instance")
	}

	// Test basic validation without validators
	err := output.Validate()
	if err != nil {
		t.Errorf("Basic validation should pass: %v", err)
	}
}

// Test OutputSettings validation
func TestOutputSettings_Validation(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func() *OutputSettings
		shouldFail bool
	}{
		{
			name: "valid json format",
			setup: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				return settings
			},
			shouldFail: false,
		},
		{
			name: "invalid format",
			setup: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("invalid-format")
				return settings
			},
			shouldFail: true,
		},
		{
			name: "mermaid without required settings",
			setup: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid")
				// Clear the MermaidSettings that are set by default
				settings.MermaidSettings = nil
				// Don't set FromToColumns or MermaidSettings
				return settings
			},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			settings := tc.setup()
			err := settings.Validate()

			if tc.shouldFail {
				if err == nil {
					t.Error("Expected validation to fail")
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass, got: %v", err)
				}
			}
		})
	}
}

// Test error handler integration
func TestOutputArray_ErrorHandlerModes(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")

	output := &OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Value"},
		Contents: []OutputHolder{{Contents: map[string]interface{}{"Name": "test", "Value": 123}}},
	}

	// Test strict mode (default)
	output.SetErrorMode(errors.ErrorModeStrict)
	if output.errorHandler.Mode() != errors.ErrorModeStrict {
		t.Error("Expected strict mode")
	}

	// Test lenient mode
	output.SetErrorMode(errors.ErrorModeLenient)
	if output.errorHandler.Mode() != errors.ErrorModeLenient {
		t.Error("Expected lenient mode")
	}

	// Test legacy mode
	output.EnableLegacyMode()
	if output.errorHandler.Mode() != errors.ErrorModeStrict {
		t.Error("Legacy mode should be strict")
	}
}

// Test WriteWithValidation basic functionality
func TestOutputArray_WriteWithValidation_Basic(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")

	output := &OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Value"},
		Contents: []OutputHolder{{Contents: map[string]interface{}{"Name": "test", "Value": 123}}},
	}

	// Basic test - should not fail
	err := output.WriteWithValidation()
	if err != nil {
		t.Errorf("WriteWithValidation should succeed: %v", err)
	}
}

// Test adapter interface implementation
func TestOutputArray_AdapterInterface(t *testing.T) {
	output := &OutputArray{
		Keys:     []string{"Name", "Value"},
		Contents: []OutputHolder{{Contents: map[string]interface{}{"Name": "test", "Value": 123}}},
	}

	// Test GetKeys
	keys := output.GetKeys()
	if len(keys) != 2 || keys[0] != "Name" || keys[1] != "Value" {
		t.Error("GetKeys should return the correct keys")
	}

	// Test GetContents
	contents := output.GetContents()
	if len(contents) != 1 {
		t.Error("GetContents should return one holder")
	}

	holder := contents[0]
	holderContents := holder.GetContents()
	if holderContents["Name"] != "test" || holderContents["Value"] != 123 {
		t.Error("GetContents should return the correct content")
	}
}
