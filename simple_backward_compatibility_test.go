package format

import (
	"fmt"
	"strings"
	"testing"
)

// TestSimpleLegacyErrorHandler tests basic LegacyErrorHandler functionality
func TestSimpleLegacyErrorHandler(t *testing.T) {
	t.Run("NewLegacyErrorHandler creates handler", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		if handler == nil {
			t.Fatal("NewLegacyErrorHandler should not return nil")
		}
	})

	t.Run("HandleError with nil returns nil", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		err := handler.HandleError(nil)
		if err != nil {
			t.Errorf("HandleError(nil) should return nil, got %v", err)
		}
	})

	t.Run("GetMode returns strict", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		if handler.GetMode() != ErrorModeStrict {
			t.Error("GetMode should return ErrorModeStrict")
		}
	})

	t.Run("GetCollectedErrors returns empty", func(t *testing.T) {
		handler := NewLegacyErrorHandler()
		errors := handler.GetCollectedErrors()
		if len(errors) != 0 {
			t.Error("GetCollectedErrors should return empty slice")
		}
	})
}

// TestSimpleOutputArrayLegacyMethods tests basic OutputArray legacy methods
func TestSimpleOutputArrayLegacyMethods(t *testing.T) {
	t.Run("EnableLegacyMode sets handler", func(t *testing.T) {
		settings := NewOutputSettings()
		settings.SetOutputFormat("json")

		output := &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     []string{"name", "value"},
		}

		result := output.EnableLegacyMode()

		if result != output {
			t.Error("EnableLegacyMode should return the same OutputArray")
		}

		if output.errorHandler == nil {
			t.Fatal("EnableLegacyMode should set an error handler")
		}

		if _, ok := output.errorHandler.(*LegacyErrorHandler); !ok {
			t.Error("EnableLegacyMode should set a LegacyErrorHandler")
		}
	})
}

// TestSimpleMigrationHelper tests basic MigrationHelper functionality
func TestSimpleMigrationHelper(t *testing.T) {
	t.Run("NewMigrationHelper creates helper", func(t *testing.T) {
		helper := NewMigrationHelper()
		if helper == nil {
			t.Fatal("NewMigrationHelper should not return nil")
		}
		if helper.legacyMode {
			t.Error("Default helper should not be in legacy mode")
		}
	})

	t.Run("EnableLegacyMode works", func(t *testing.T) {
		helper := NewMigrationHelper()
		result := helper.EnableLegacyMode()

		if result != helper {
			t.Error("EnableLegacyMode should return the same helper")
		}
		if !helper.legacyMode {
			t.Error("EnableLegacyMode should set legacy mode to true")
		}
	})

	t.Run("HandleError with nil returns nil", func(t *testing.T) {
		helper := NewMigrationHelper()
		result := helper.HandleError(nil)
		if result != nil {
			t.Error("HandleError with nil should return nil")
		}
	})

	t.Run("HandleError in modern mode returns error", func(t *testing.T) {
		helper := NewMigrationHelper()
		testErr := fmt.Errorf("test error")

		result := helper.HandleError(testErr)
		if result != testErr {
			t.Error("HandleError in modern mode should return the original error")
		}
	})
}

// TestSimpleMigrationConfig tests basic migration configuration
func TestSimpleMigrationConfig(t *testing.T) {
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
	})
}

// TestWriteCompatPanic tests that WriteCompat panics on errors
func TestWriteCompatPanic(t *testing.T) {
	t.Run("WriteCompat with custom validator that fails", func(t *testing.T) {
		settings := NewOutputSettings()
		settings.SetOutputFormat("json") // Use a simple format

		output := &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     []string{"name", "value"},
		}

		// Add a validator that always fails
		output.AddValidator(ValidatorFunc(func(subject any) error {
			return fmt.Errorf("test validation error")
		}))

		// Add some data
		output.AddContents(map[string]interface{}{
			"name":  "test",
			"value": 42,
		})

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

// TestLegacyErrorHandlerPanic tests that LegacyErrorHandler calls fatal function
func TestLegacyErrorHandlerPanic(t *testing.T) {
	t.Run("HandleError calls fatal function", func(t *testing.T) {
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
}
