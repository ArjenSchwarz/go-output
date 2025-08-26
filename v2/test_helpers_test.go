package output

import (
	"os"
	"testing"
)

// TestSkipIfNotIntegration_WithoutEnvVar tests that the helper skips tests when INTEGRATION is not set.
func TestSkipIfNotIntegration_WithoutEnvVar(t *testing.T) {
	// Save current INTEGRATION value and restore it after test
	originalValue := os.Getenv("INTEGRATION")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("INTEGRATION")
		} else {
			os.Setenv("INTEGRATION", originalValue)
		}
	}()

	// Clear INTEGRATION environment variable
	os.Unsetenv("INTEGRATION")

	// Create a mock test to verify skip behavior
	mockT := &mockTestingT{}
	skipIfNotIntegrationWithHelper(mockT)

	if !mockT.skipped {
		t.Error("Expected test to be skipped when INTEGRATION is not set")
	}

	if mockT.skipMessage != "Skipping integration test (set INTEGRATION=1 to run)" {
		t.Errorf("Expected skip message %q, got %q",
			"Skipping integration test (set INTEGRATION=1 to run)",
			mockT.skipMessage)
	}
}

// TestSkipIfNotIntegration_WithEnvVarSet tests that the helper does not skip tests when INTEGRATION=1.
func TestSkipIfNotIntegration_WithEnvVarSet(t *testing.T) {
	// Save current INTEGRATION value and restore it after test
	originalValue := os.Getenv("INTEGRATION")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("INTEGRATION")
		} else {
			os.Setenv("INTEGRATION", originalValue)
		}
	}()

	// Set INTEGRATION environment variable to 1
	os.Setenv("INTEGRATION", "1")

	// Create a mock test to verify no skip behavior
	mockT := &mockTestingT{}
	skipIfNotIntegrationWithHelper(mockT)

	if mockT.skipped {
		t.Error("Expected test not to be skipped when INTEGRATION=1")
	}
}

// TestSkipIfNotIntegration_WithDifferentValue tests that the helper skips tests when INTEGRATION has a different value.
func TestSkipIfNotIntegration_WithDifferentValue(t *testing.T) {
	// Save current INTEGRATION value and restore it after test
	originalValue := os.Getenv("INTEGRATION")
	defer func() {
		if originalValue == "" {
			os.Unsetenv("INTEGRATION")
		} else {
			os.Setenv("INTEGRATION", originalValue)
		}
	}()

	// Test various non-"1" values
	testCases := []string{"true", "yes", "0", "false", "2", ""}

	for _, value := range testCases {
		t.Run("INTEGRATION="+value, func(t *testing.T) {
			os.Setenv("INTEGRATION", value)

			mockT := &mockTestingT{}
			skipIfNotIntegrationWithHelper(mockT)

			if !mockT.skipped {
				t.Errorf("Expected test to be skipped when INTEGRATION=%q", value)
			}
		})
	}
}

// mockTestingT is a mock implementation of testHelper interface for testing purposes.
type mockTestingT struct {
	skipped     bool
	skipMessage string
	helperCalls int
}

func (m *mockTestingT) Helper() {
	m.helperCalls++
}

func (m *mockTestingT) Skip(args ...any) {
	m.skipped = true
	if len(args) > 0 {
		if msg, ok := args[0].(string); ok {
			m.skipMessage = msg
		}
	}
}
