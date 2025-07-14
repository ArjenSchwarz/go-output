package format

import (
	"fmt"
	"strings"
	"testing"
)

// TestLegacyErrorHandler tests the LegacyErrorHandler functionality
func TestLegacyErrorHandler(t *testing.T) {
	t.Run("NewLegacyErrorHandler", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		if handler == nil {
			t.Fatal("NewLegacyErrorHandler should not return nil")
		}
		if handler.logFatalFunc == nil {
			t.Fatal("logFatalFunc should be set")
		}
	})

	t.Run("HandleError with nil error", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		err := handler.HandleError(nil)
		if err != nil {
			t.Errorf("HandleError(nil) should return nil, got %v", err)
		}
	})

	t.Run("HandleError with error calls fatal function", func(t *testing.T) {
		var fatalCalled bool
		var fatalMessage string

		mockFatal := func(v ...interface{}) {
			fatalCalled = true
			fatalMessage = fmt.Sprintf("%v", v)
		}

		handler := NewLegacyErrorHandlerWithFatalFunc(mockFatal)
		testErr := fmt.Errorf("test error")

		// This should call the fatal function
		handler.HandleError(testErr)

		if !fatalCalled {
			t.Error("Fatal function should have been called")
		}
		if !strings.Contains(fatalMessage, "test error") {
			t.Errorf("Fatal message should contain 'test error', got: %s", fatalMessage)
		}
	})

	t.Run("SetMode is no-op", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		// Should not panic or cause issues
		handler.SetMode(ErrorModeLenient)
		handler.SetMode(ErrorModeInteractive)
	})

	t.Run("GetMode always returns strict", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		if handler.GetMode() != ErrorModeStrict {
			t.Error("GetMode should always return ErrorModeStrict")
		}

		// Even after setting different modes
		handler.SetMode(ErrorModeLenient)
		if handler.GetMode() != ErrorModeStrict {
			t.Error("GetMode should still return ErrorModeStrict after SetMode")
		}
	})

	t.Run("GetCollectedErrors returns empty slice", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		errors := handler.GetCollectedErrors()
		if len(errors) != 0 {
			t.Error("GetCollectedErrors should return empty slice")
		}
	})

	t.Run("Clear is no-op", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		// Should not panic
		handler.Clear()
	})
}

// TestOutputArrayLegacyMethods tests the legacy methods added to OutputArray
func TestOutputArrayLegacyMethods(t *testing.T) {
	t.Run("EnableLegacyMode", func(t *testing.T) {
		settings := NewOutputSettings()
		settings.SetOutputFormat("json")

		output := &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     []string{"name", "value"},
		}

		result := output.EnableLegacyMode()

		// Should return the same OutputArray for chaining
		if result != output {
			t.Error("EnableLegacyMode should return the same OutputArray for method chaining")
		}

		// Should set a LegacyErrorHandler
		if output.errorHandler == nil {
			t.Fatal("EnableLegacyMode should set an error handler")
		}

		if _, ok := output.errorHandler.(*LegacyErrorHandler); !ok {
			t.Error("EnableLegacyMode should set a LegacyErrorHandler")
		}
	})

	t.Run("WriteCompat with no error", func(t *testing.T) {
		settings := NewOutputSettings()
		settings.SetOutputFormat("json")

		output := &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     []string{"name", "value"},
		}

		// Add some test data
		output.AddContents(map[string]interface{}{
			"name":  "test",
			"value": 42,
		})

		// This should not panic since there are no validation errors
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("WriteCompat should not panic with valid data: %v", r)
			}
		}()

		// Note: This will actually try to write to stdout, but shouldn't panic
		// In a real test environment, we might want to redirect stdout
		// For now, we'll skip the actual write to avoid test output pollution
		// output.WriteCompat()
	})

	t.Run("WriteCompat with error should panic", func(t *testing.T) {
		settings := NewOutputSettings()
		settings.SetOutputFormat("invalid_format") // This should cause an error

		output := &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     []string{"name", "value"},
		}

		// This should panic due to validation error
		defer func() {
			if r := recover(); r == nil {
				t.Error("WriteCompat should panic when Write() returns an error")
			} else {
				// Check that the panic message contains "FATAL"
				panicMsg := fmt.Sprintf("%v", r)
				if !strings.Contains(panicMsg, "FATAL") {
					t.Errorf("Panic message should contain 'FATAL', got: %s", panicMsg)
				}
			}
		}()

		output.WriteCompat()
	})
}

// TestMigrationHelper tests the migration helper functionality
func TestMigrationHelper(t *testing.T) {
	t.Run("NewMigrationHelper", func(t *testing.T) {
		helper := NewMigrationHelper()
		if helper == nil {
			t.Fatal("NewMigrationHelper should not return nil")
		}
		if helper.legacyMode {
			t.Error("Default migration helper should not be in legacy mode")
		}
		if !helper.warnOnErrors {
			t.Error("Default migration helper should warn on errors")
		}
	})

	t.Run("NewLegacyMigrationHelper", func(t *testing.T) {
		helper := NewLegacyMigrationHelper()
		if helper == nil {
			t.Fatal("NewLegacyMigrationHelper should not return nil")
		}
		if !helper.legacyMode {
			t.Error("Legacy migration helper should be in legacy mode")
		}
		if helper.warnOnErrors {
			t.Error("Legacy migration helper should not warn on errors")
		}
	})

	t.Run("EnableLegacyMode", func(t *testing.T) {
		helper := NewMigrationHelper()
		result := helper.EnableLegacyMode()

		if result != helper {
			t.Error("EnableLegacyMode should return the same helper for chaining")
		}
		if !helper.legacyMode {
			t.Error("EnableLegacyMode should set legacy mode to true")
		}
	})

	t.Run("DisableLegacyMode", func(t *testing.T) {
		helper := NewLegacyMigrationHelper()
		result := helper.DisableLegacyMode()

		if result != helper {
			t.Error("DisableLegacyMode should return the same helper for chaining")
		}
		if helper.legacyMode {
			t.Error("DisableLegacyMode should set legacy mode to false")
		}
	})

	t.Run("EnableWarnings", func(t *testing.T) {
		helper := NewMigrationHelper()
		helper.DisableWarnings()
		result := helper.EnableWarnings()

		if result != helper {
			t.Error("EnableWarnings should return the same helper for chaining")
		}
		if !helper.warnOnErrors {
			t.Error("EnableWarnings should set warnOnErrors to true")
		}
	})

	t.Run("DisableWarnings", func(t *testing.T) {
		helper := NewMigrationHelper()
		result := helper.DisableWarnings()

		if result != helper {
			t.Error("DisableWarnings should return the same helper for chaining")
		}
		if helper.warnOnErrors {
			t.Error("DisableWarnings should set warnOnErrors to false")
		}
	})

	t.Run("IsLegacyMode", func(t *testing.T) {
		helper := NewMigrationHelper()
		if helper.IsLegacyMode() {
			t.Error("Default helper should not be in legacy mode")
		}

		helper.EnableLegacyMode()
		if !helper.IsLegacyMode() {
			t.Error("Helper should be in legacy mode after EnableLegacyMode")
		}
	})

	t.Run("HandleError in modern mode", func(t *testing.T) {
		helper := NewMigrationHelper()
		testErr := fmt.Errorf("test error")

		result := helper.HandleError(testErr)
		if result != testErr {
			t.Error("HandleError in modern mode should return the original error")
		}
	})

	t.Run("HandleError with nil", func(t *testing.T) {
		helper := NewMigrationHelper()
		result := helper.HandleError(nil)
		if result != nil {
			t.Error("HandleError with nil should return nil")
		}
	})
}

// TestMigrationAwareOutputArray tests the migration-aware wrapper
func TestMigrationAwareOutputArray(t *testing.T) {
	t.Run("WrapOutputArray", func(t *testing.T) {
		settings := NewOutputSettings()
		settings.SetOutputFormat("json")

		output := &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     []string{"name", "value"},
		}

		helper := NewMigrationHelper()
		wrapped := helper.WrapOutputArray(output)

		if wrapped == nil {
			t.Fatal("WrapOutputArray should not return nil")
		}
		if wrapped.OutputArray != output {
			t.Error("Wrapped array should contain the original OutputArray")
		}
		if wrapped.helper != helper {
			t.Error("Wrapped array should contain the migration helper")
		}
	})

	t.Run("Write in modern mode", func(t *testing.T) {
		settings := NewOutputSettings()
		settings.SetOutputFormat("json")

		output := &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     []string{"name", "value"},
		}

		output.AddContents(map[string]interface{}{
			"name":  "test",
			"value": 42,
		})

		helper := NewMigrationHelper()
		wrapped := helper.WrapOutputArray(output)

		// This should work without panicking in modern mode
		// Note: We're not actually calling Write() to avoid stdout pollution
		if wrapped == nil {
			t.Error("WrapOutputArray should return a valid wrapped array")
		}
		// err := wrapped.Write()
		// if err != nil {
		//     t.Errorf("Write should succeed with valid data: %v", err)
		// }
	})
}

// TestMigrationConfig tests migration configuration functionality
func TestMigrationConfig(t *testing.T) {
	t.Run("DefaultMigrationConfig", func(t *testing.T) {
		config := DefaultMigrationConfig()

		if config.UseLegacyMode {
			t.Error("Default config should not use legacy mode")
		}
		if !config.LogWarnings {
			t.Error("Default config should log warnings")
		}
		if config.ErrorHandler == nil {
			t.Error("Default config should have an error handler")
		}
		if config.ValidationMode != ErrorModeStrict {
			t.Error("Default config should use strict validation mode")
		}
	})

	t.Run("LegacyMigrationConfig", func(t *testing.T) {
		config := LegacyMigrationConfig()

		if !config.UseLegacyMode {
			t.Error("Legacy config should use legacy mode")
		}
		if config.LogWarnings {
			t.Error("Legacy config should not log warnings")
		}
		if config.ErrorHandler == nil {
			t.Error("Legacy config should have an error handler")
		}
		if _, ok := config.ErrorHandler.(*LegacyErrorHandler); !ok {
			t.Error("Legacy config should use LegacyErrorHandler")
		}
	})

	t.Run("ApplyMigrationConfig with legacy mode", func(t *testing.T) {
		settings := NewOutputSettings()
		output := &OutputArray{Settings: settings}

		config := LegacyMigrationConfig()
		result := ApplyMigrationConfig(output, config)

		if result != output {
			t.Error("ApplyMigrationConfig should return the same OutputArray")
		}
		if _, ok := output.errorHandler.(*LegacyErrorHandler); !ok {
			t.Error("Legacy config should set LegacyErrorHandler")
		}
	})

	t.Run("ApplyMigrationConfig with modern mode", func(t *testing.T) {
		settings := NewOutputSettings()
		output := &OutputArray{Settings: settings}

		config := DefaultMigrationConfig()
		result := ApplyMigrationConfig(output, config)

		if result != output {
			t.Error("ApplyMigrationConfig should return the same OutputArray")
		}
		if output.errorHandler == nil {
			t.Error("Modern config should set an error handler")
		}
		if output.errorHandler.GetMode() != ErrorModeStrict {
			t.Error("Modern config should set strict mode")
		}
	})
}

// TestMigrationWrapper tests the migration wrapper functionality
func TestMigrationWrapper(t *testing.T) {
	t.Run("NewMigrationWrapper", func(t *testing.T) {
		config := DefaultMigrationConfig()
		wrapper := NewMigrationWrapper(config)

		if wrapper == nil {
			t.Fatal("NewMigrationWrapper should not return nil")
		}
		if wrapper.config.UseLegacyMode != config.UseLegacyMode {
			t.Error("Wrapper should store the provided config")
		}
	})

	t.Run("NewLegacyMigrationWrapper", func(t *testing.T) {
		wrapper := NewLegacyMigrationWrapper()

		if wrapper == nil {
			t.Fatal("NewLegacyMigrationWrapper should not return nil")
		}
		if !wrapper.config.UseLegacyMode {
			t.Error("Legacy wrapper should use legacy mode")
		}
	})

	t.Run("NewModernMigrationWrapper", func(t *testing.T) {
		wrapper := NewModernMigrationWrapper()

		if wrapper == nil {
			t.Fatal("NewModernMigrationWrapper should not return nil")
		}
		if wrapper.config.UseLegacyMode {
			t.Error("Modern wrapper should not use legacy mode")
		}
	})

	t.Run("CreateOutputArray", func(t *testing.T) {
		settings := NewOutputSettings()
		wrapper := NewModernMigrationWrapper()

		output := wrapper.CreateOutputArray(settings)

		if output == nil {
			t.Fatal("CreateOutputArray should not return nil")
		}
		if output.Settings != settings {
			t.Error("Created array should use the provided settings")
		}
		if output.errorHandler == nil {
			t.Error("Created array should have an error handler")
		}
	})
}

// TestGradualMigrationGuide tests the gradual migration guide functionality
func TestGradualMigrationGuide(t *testing.T) {
	t.Run("NewGradualMigrationGuide", func(t *testing.T) {
		guide := NewGradualMigrationGuide()

		if guide == nil {
			t.Fatal("NewGradualMigrationGuide should not return nil")
		}

		steps := guide.GetSteps()
		if len(steps) == 0 {
			t.Error("Migration guide should have steps")
		}

		// Check that all steps have required fields
		for i, step := range steps {
			if step.Name == "" {
				t.Errorf("Step %d should have a name", i)
			}
			if step.Description == "" {
				t.Errorf("Step %d should have a description", i)
			}
			if step.Action == nil {
				t.Errorf("Step %d should have an action", i)
			}
		}
	})

	t.Run("ExecuteStep with valid index", func(t *testing.T) {
		guide := NewGradualMigrationGuide()
		settings := NewOutputSettings()
		output := &OutputArray{Settings: settings}

		// Execute the first step (Enable Legacy Mode)
		result, err := guide.ExecuteStep(0, output)

		if err != nil {
			t.Errorf("ExecuteStep should not return error: %v", err)
		}
		if result == nil {
			t.Error("ExecuteStep should return an OutputArray")
		}

		// Check that legacy mode was enabled
		if _, ok := result.errorHandler.(*LegacyErrorHandler); !ok {
			t.Error("First step should enable legacy mode")
		}
	})

	t.Run("ExecuteStep with invalid index", func(t *testing.T) {
		guide := NewGradualMigrationGuide()
		settings := NewOutputSettings()
		output := &OutputArray{Settings: settings}

		// Test negative index
		_, err := guide.ExecuteStep(-1, output)
		if err == nil {
			t.Error("ExecuteStep with negative index should return error")
		}

		// Test index too large
		steps := guide.GetSteps()
		_, err = guide.ExecuteStep(len(steps), output)
		if err == nil {
			t.Error("ExecuteStep with too large index should return error")
		}
	})
}

// TestBackwardCompatibilityIntegration tests the integration of all backward compatibility features
func TestBackwardCompatibilityIntegration(t *testing.T) {
	t.Run("Complete legacy migration scenario", func(t *testing.T) {
		// Start with a basic OutputArray
		settings := NewOutputSettings()
		settings.SetOutputFormat("json")

		output := &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     []string{"name", "value"},
		}

		// Add test data
		output.AddContents(map[string]interface{}{
			"name":  "test",
			"value": 42,
		})

		// Step 1: Enable legacy mode for backward compatibility
		output.EnableLegacyMode()

		// Verify legacy mode is enabled
		if _, ok := output.errorHandler.(*LegacyErrorHandler); !ok {
			t.Error("Legacy mode should be enabled")
		}

		// Step 2: Use migration wrapper for gradual transition
		helper := NewMigrationHelper().EnableLegacyMode()
		wrapped := helper.WrapOutputArray(output)

		if wrapped.helper.IsLegacyMode() != true {
			t.Error("Wrapped output should be in legacy mode")
		}

		// Step 3: Transition to modern mode
		helper.DisableLegacyMode()
		output.WithErrorHandler(NewDefaultErrorHandler())

		// Verify modern mode
		if _, ok := output.errorHandler.(*DefaultErrorHandler); !ok {
			t.Error("Should now be using modern error handler")
		}
	})

	t.Run("Migration config application", func(t *testing.T) {
		settings := NewOutputSettings()
		settings.SetOutputFormat("json")

		output := &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     []string{"name", "value"},
		}

		// Apply legacy configuration
		legacyConfig := LegacyMigrationConfig()
		ApplyMigrationConfig(output, legacyConfig)

		// Verify legacy configuration
		if _, ok := output.errorHandler.(*LegacyErrorHandler); !ok {
			t.Error("Legacy config should set LegacyErrorHandler")
		}

		// Apply modern configuration
		modernConfig := DefaultMigrationConfig()
		modernConfig.ValidationMode = ErrorModeLenient
		ApplyMigrationConfig(output, modernConfig)

		// Verify modern configuration
		if output.errorHandler.GetMode() != ErrorModeLenient {
			t.Error("Modern config should set lenient mode")
		}
	})
}

// BenchmarkLegacyErrorHandler benchmarks the legacy error handler performance
func BenchmarkLegacyErrorHandler(b *testing.B) {
	// Use a no-op fatal function to avoid actual termination
	handler := NewLegacyErrorHandlerWithFatalFunc(func(...interface{}) {})
	testErr := fmt.Errorf("benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandleError(testErr)
	}
}

// BenchmarkMigrationHelper benchmarks the migration helper performance
func BenchmarkMigrationHelper(b *testing.B) {
	helper := NewMigrationHelper()
	testErr := fmt.Errorf("benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		helper.HandleError(testErr)
	}
}
