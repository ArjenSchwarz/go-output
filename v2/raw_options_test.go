package output

import (
	"testing"
)

// TestWithFormatValidation verifies format validation option
func TestWithFormatValidation(t *testing.T) {
	tests := map[string]struct {
		validate bool
	}{"disable validation": {

		validate: false,
	}, "enable validation": {

		validate: true,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rc := &rawConfig{
				validateFormat: !tt.validate, // Start with opposite
				preserveData:   true,
			}
			opt := WithFormatValidation(tt.validate)
			opt(rc)

			if rc.validateFormat != tt.validate {
				t.Errorf("validateFormat = %v, want %v", rc.validateFormat, tt.validate)
			}
			// Ensure other properties are preserved
			if !rc.preserveData {
				t.Error("preserveData should be preserved")
			}
		})
	}
}

// TestWithDataPreservation verifies data preservation option
func TestWithDataPreservation(t *testing.T) {
	tests := map[string]struct {
		preserve bool
	}{"disable preservation": {

		preserve: false,
	}, "enable preservation": {

		preserve: true,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rc := &rawConfig{
				validateFormat: true,
				preserveData:   !tt.preserve, // Start with opposite
			}
			opt := WithDataPreservation(tt.preserve)
			opt(rc)

			if rc.preserveData != tt.preserve {
				t.Errorf("preserveData = %v, want %v", rc.preserveData, tt.preserve)
			}
			// Ensure other properties are preserved
			if !rc.validateFormat {
				t.Error("validateFormat should be preserved")
			}
		})
	}
}

// TestApplyRawOptions verifies option application with defaults
func TestApplyRawOptions(t *testing.T) {
	tests := map[string]struct {
		opts               []RawOption
		expectedValidation bool
		expectedPreserve   bool
	}{"disable both": {

		opts: []RawOption{
			WithFormatValidation(false),
			WithDataPreservation(false),
		},
		expectedValidation: false,
		expectedPreserve:   false,
	}, "disable preservation only": {

		opts: []RawOption{
			WithDataPreservation(false),
		},
		expectedValidation: true,
		expectedPreserve:   false,
	}, "disable validation only": {

		opts: []RawOption{
			WithFormatValidation(false),
		},
		expectedValidation: false,
		expectedPreserve:   true,
	}, "last option wins": {

		opts: []RawOption{
			WithFormatValidation(false),
			WithFormatValidation(true),
			WithDataPreservation(false),
			WithDataPreservation(true),
		},
		expectedValidation: true,
		expectedPreserve:   true,
	}, "mixed order": {

		opts: []RawOption{
			WithDataPreservation(false),
			WithFormatValidation(false),
			WithDataPreservation(true),
		},
		expectedValidation: false,
		expectedPreserve:   true,
	}, "no options uses defaults": {

		opts:               []RawOption{},
		expectedValidation: true,
		expectedPreserve:   true,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rc := ApplyRawOptions(tt.opts...)

			if rc.validateFormat != tt.expectedValidation {
				t.Errorf("validateFormat = %v, want %v", rc.validateFormat, tt.expectedValidation)
			}
			if rc.preserveData != tt.expectedPreserve {
				t.Errorf("preserveData = %v, want %v", rc.preserveData, tt.expectedPreserve)
			}
		})
	}
}

// TestRawOptionsDefaults verifies the default values are sensible
func TestRawOptionsDefaults(t *testing.T) {
	rc := ApplyRawOptions()

	// Defaults should be true for safety
	if !rc.validateFormat {
		t.Error("validateFormat should default to true for safety")
	}
	if !rc.preserveData {
		t.Error("preserveData should default to true for data integrity")
	}
}

// TestRawOptionsIndependence verifies options don't affect each other
func TestRawOptionsIndependence(t *testing.T) {
	// Create first configuration
	rc1 := ApplyRawOptions(
		WithFormatValidation(true),
		WithDataPreservation(false),
	)

	// Create second configuration
	rc2 := ApplyRawOptions(
		WithFormatValidation(false),
		WithDataPreservation(true),
	)

	// Verify they're independent
	if !rc1.validateFormat {
		t.Error("rc1 should have validation enabled")
	}
	if rc1.preserveData {
		t.Error("rc1 should have preservation disabled")
	}
	if rc2.validateFormat {
		t.Error("rc2 should have validation disabled")
	}
	if !rc2.preserveData {
		t.Error("rc2 should have preservation enabled")
	}
}

// TestRawOptionsCombinations tests various practical combinations
func TestRawOptionsCombinations(t *testing.T) {
	// Performance mode: no validation, no preservation
	perfMode := ApplyRawOptions(
		WithFormatValidation(false),
		WithDataPreservation(false),
	)
	if perfMode.validateFormat || perfMode.preserveData {
		t.Error("Performance mode should disable both validation and preservation")
	}

	// Safe mode: everything enabled (default)
	safeMode := ApplyRawOptions()
	if !safeMode.validateFormat || !safeMode.preserveData {
		t.Error("Safe mode should enable both validation and preservation")
	}

	// Trust mode: no validation but preserve data
	trustMode := ApplyRawOptions(
		WithFormatValidation(false),
		WithDataPreservation(true),
	)
	if trustMode.validateFormat || !trustMode.preserveData {
		t.Error("Trust mode should disable validation but preserve data")
	}
}
