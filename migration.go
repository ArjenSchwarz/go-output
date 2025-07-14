package format

import (
	"fmt"
	"log"
)

// MigrationHelper provides utilities for gradual adoption of the new error handling system
type MigrationHelper struct {
	// legacyMode determines whether to use legacy log.Fatal behavior
	legacyMode bool
	// warnOnErrors determines whether to log warnings when errors occur in non-legacy mode
	warnOnErrors bool
}

// NewMigrationHelper creates a new migration helper with default settings
func NewMigrationHelper() *MigrationHelper {
	return &MigrationHelper{
		legacyMode:   false,
		warnOnErrors: true,
	}
}

// NewLegacyMigrationHelper creates a migration helper that maintains legacy behavior
func NewLegacyMigrationHelper() *MigrationHelper {
	return &MigrationHelper{
		legacyMode:   true,
		warnOnErrors: false,
	}
}

// EnableLegacyMode enables legacy log.Fatal behavior for backward compatibility
func (m *MigrationHelper) EnableLegacyMode() *MigrationHelper {
	m.legacyMode = true
	return m
}

// DisableLegacyMode disables legacy behavior and uses modern error handling
func (m *MigrationHelper) DisableLegacyMode() *MigrationHelper {
	m.legacyMode = false
	return m
}

// EnableWarnings enables warning logs when errors occur in non-legacy mode
func (m *MigrationHelper) EnableWarnings() *MigrationHelper {
	m.warnOnErrors = true
	return m
}

// DisableWarnings disables warning logs
func (m *MigrationHelper) DisableWarnings() *MigrationHelper {
	m.warnOnErrors = false
	return m
}

// IsLegacyMode returns true if legacy mode is enabled
func (m *MigrationHelper) IsLegacyMode() bool {
	return m.legacyMode
}

// HandleError processes an error according to the migration helper configuration
// In legacy mode, it calls log.Fatal. In modern mode, it returns the error.
func (m *MigrationHelper) HandleError(err error) error {
	if err == nil {
		return nil
	}

	if m.legacyMode {
		// Use panic to simulate log.Fatal behavior to avoid actual program termination in tests
		panic(fmt.Sprintf("FATAL: %v", err))
		return nil // This line should never be reached
	}

	if m.warnOnErrors {
		log.Printf("Warning: %v", err)
	}

	return err
}

// WrapOutputArray wraps an OutputArray with migration-aware error handling
func (m *MigrationHelper) WrapOutputArray(output *OutputArray) *MigrationAwareOutputArray {
	return &MigrationAwareOutputArray{
		OutputArray: output,
		helper:      m,
	}
}

// MigrationAwareOutputArray wraps OutputArray with migration-aware behavior
type MigrationAwareOutputArray struct {
	*OutputArray
	helper *MigrationHelper
}

// Write provides migration-aware write functionality
// In legacy mode, it behaves like the old WriteCompat method
// In modern mode, it returns errors for proper handling
func (m *MigrationAwareOutputArray) Write() error {
	err := m.OutputArray.Write()
	return m.helper.HandleError(err)
}

// Validate provides migration-aware validation
func (m *MigrationAwareOutputArray) Validate() error {
	err := m.OutputArray.Validate()
	return m.helper.HandleError(err)
}

// MigrationConfig provides configuration options for migration scenarios
type MigrationConfig struct {
	// UseLegacyMode determines whether to use log.Fatal behavior
	UseLegacyMode bool
	// LogWarnings determines whether to log warnings for errors
	LogWarnings bool
	// ErrorHandler allows custom error handling during migration
	ErrorHandler ErrorHandler
	// ValidationMode determines how validation errors are handled
	ValidationMode ErrorMode
}

// DefaultMigrationConfig returns a default migration configuration
func DefaultMigrationConfig() MigrationConfig {
	return MigrationConfig{
		UseLegacyMode:  false,
		LogWarnings:    true,
		ErrorHandler:   NewDefaultErrorHandler(),
		ValidationMode: ErrorModeStrict,
	}
}

// LegacyMigrationConfig returns a migration configuration that maintains legacy behavior
func LegacyMigrationConfig() MigrationConfig {
	return MigrationConfig{
		UseLegacyMode:  true,
		LogWarnings:    false,
		ErrorHandler:   NewLegacyErrorHandler(),
		ValidationMode: ErrorModeStrict,
	}
}

// ApplyMigrationConfig applies migration configuration to an OutputArray
func ApplyMigrationConfig(output *OutputArray, config MigrationConfig) *OutputArray {
	if config.UseLegacyMode {
		output.EnableLegacyMode()
	} else if config.ErrorHandler != nil {
		output.WithErrorHandler(config.ErrorHandler)
		config.ErrorHandler.SetMode(config.ValidationMode)
	}

	return output
}

// MigrationWrapper provides a high-level wrapper for migration scenarios
type MigrationWrapper struct {
	config MigrationConfig
}

// NewMigrationWrapper creates a new migration wrapper with the specified configuration
func NewMigrationWrapper(config MigrationConfig) *MigrationWrapper {
	return &MigrationWrapper{
		config: config,
	}
}

// NewLegacyMigrationWrapper creates a migration wrapper that maintains legacy behavior
func NewLegacyMigrationWrapper() *MigrationWrapper {
	return &MigrationWrapper{
		config: LegacyMigrationConfig(),
	}
}

// NewModernMigrationWrapper creates a migration wrapper that uses modern error handling
func NewModernMigrationWrapper() *MigrationWrapper {
	return &MigrationWrapper{
		config: DefaultMigrationConfig(),
	}
}

// WrapOutputArray applies migration configuration to an OutputArray
func (w *MigrationWrapper) WrapOutputArray(output *OutputArray) *OutputArray {
	return ApplyMigrationConfig(output, w.config)
}

// CreateOutputArray creates a new OutputArray with migration configuration applied
func (w *MigrationWrapper) CreateOutputArray(settings *OutputSettings) *OutputArray {
	output := &OutputArray{
		Settings:   settings,
		Contents:   make([]OutputHolder, 0),
		Keys:       make([]string, 0),
		validators: make([]Validator, 0),
	}

	return w.WrapOutputArray(output)
}

// MigrationExample demonstrates how to use migration utilities
func MigrationExample() {
	// Example 1: Legacy mode for backward compatibility
	legacyWrapper := NewLegacyMigrationWrapper()
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")

	output := legacyWrapper.CreateOutputArray(settings)
	output.AddContents(map[string]interface{}{
		"name":  "example",
		"value": 42,
	})

	// This will use log.Fatal on errors (legacy behavior)
	output.WriteCompat()

	// Example 2: Modern mode with error handling
	modernWrapper := NewModernMigrationWrapper()
	modernOutput := modernWrapper.CreateOutputArray(settings)
	modernOutput.AddContents(map[string]interface{}{
		"name":  "modern example",
		"value": 100,
	})

	// This returns errors for proper handling
	if err := modernOutput.Write(); err != nil {
		fmt.Printf("Error occurred: %v\n", err)
	}

	// Example 3: Custom migration configuration
	customConfig := MigrationConfig{
		UseLegacyMode:  false,
		LogWarnings:    true,
		ErrorHandler:   NewErrorHandlerWithMode(ErrorModeLenient),
		ValidationMode: ErrorModeLenient,
	}

	customWrapper := NewMigrationWrapper(customConfig)
	customOutput := customWrapper.CreateOutputArray(settings)

	// This will collect all errors and continue processing where possible
	if err := customOutput.Write(); err != nil {
		fmt.Printf("Some errors occurred: %v\n", err)
	}
}

// GradualMigrationGuide provides guidance for gradual migration
type GradualMigrationGuide struct {
	steps []MigrationStep
}

// MigrationStep represents a single step in the migration process
type MigrationStep struct {
	Name        string
	Description string
	Action      func(*OutputArray) *OutputArray
	Validation  func(*OutputArray) error
}

// NewGradualMigrationGuide creates a new migration guide
func NewGradualMigrationGuide() *GradualMigrationGuide {
	return &GradualMigrationGuide{
		steps: []MigrationStep{
			{
				Name:        "Enable Legacy Mode",
				Description: "Start by enabling legacy mode to maintain existing behavior",
				Action: func(output *OutputArray) *OutputArray {
					return output.EnableLegacyMode()
				},
				Validation: func(output *OutputArray) error {
					if output.errorHandler == nil {
						return fmt.Errorf("legacy mode not properly enabled")
					}
					return nil
				},
			},
			{
				Name:        "Add Error Handling",
				Description: "Replace WriteCompat() calls with Write() and proper error handling",
				Action: func(output *OutputArray) *OutputArray {
					// This step requires manual code changes
					return output
				},
				Validation: func(output *OutputArray) error {
					// Validation would check if Write() is being used instead of WriteCompat()
					return nil
				},
			},
			{
				Name:        "Switch to Modern Mode",
				Description: "Disable legacy mode and use modern error handling",
				Action: func(output *OutputArray) *OutputArray {
					return output.WithErrorHandler(NewDefaultErrorHandler())
				},
				Validation: func(output *OutputArray) error {
					if _, ok := output.errorHandler.(*LegacyErrorHandler); ok {
						return fmt.Errorf("still using legacy error handler")
					}
					return nil
				},
			},
			{
				Name:        "Add Validation",
				Description: "Add custom validators for your specific use cases",
				Action: func(output *OutputArray) *OutputArray {
					// Example: Add a basic validator
					output.AddValidator(ValidatorFunc(func(subject any) error {
						if arr, ok := subject.(*OutputArray); ok {
							if len(arr.Contents) == 0 {
								return NewValidationError(ErrEmptyDataset, "output array is empty")
							}
						}
						return nil
					}))
					return output
				},
				Validation: func(output *OutputArray) error {
					if len(output.validators) == 0 {
						return fmt.Errorf("no validators configured")
					}
					return nil
				},
			},
			{
				Name:        "Configure Error Modes",
				Description: "Choose appropriate error handling mode (strict, lenient, interactive)",
				Action: func(output *OutputArray) *OutputArray {
					handler := NewErrorHandlerWithMode(ErrorModeLenient)
					return output.WithErrorHandler(handler)
				},
				Validation: func(output *OutputArray) error {
					if output.errorHandler.GetMode() == ErrorModeStrict {
						return fmt.Errorf("consider using lenient mode for better error handling")
					}
					return nil
				},
			},
		},
	}
}

// GetSteps returns all migration steps
func (g *GradualMigrationGuide) GetSteps() []MigrationStep {
	return g.steps
}

// ExecuteStep executes a specific migration step
func (g *GradualMigrationGuide) ExecuteStep(stepIndex int, output *OutputArray) (*OutputArray, error) {
	if stepIndex < 0 || stepIndex >= len(g.steps) {
		return output, fmt.Errorf("invalid step index: %d", stepIndex)
	}

	step := g.steps[stepIndex]
	modifiedOutput := step.Action(output)

	if step.Validation != nil {
		if err := step.Validation(modifiedOutput); err != nil {
			return output, fmt.Errorf("step %d (%s) validation failed: %w", stepIndex, step.Name, err)
		}
	}

	return modifiedOutput, nil
}

// ExecuteAllSteps executes all migration steps in sequence
func (g *GradualMigrationGuide) ExecuteAllSteps(output *OutputArray) (*OutputArray, error) {
	currentOutput := output

	for i, step := range g.steps {
		var err error
		currentOutput, err = g.ExecuteStep(i, currentOutput)
		if err != nil {
			return output, fmt.Errorf("migration failed at step %d (%s): %w", i, step.Name, err)
		}
	}

	return currentOutput, nil
}
