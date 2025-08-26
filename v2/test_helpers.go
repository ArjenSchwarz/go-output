package output

import (
	"os"
	"testing"
)

// testHelper is an interface that represents the subset of testing.TB we use.
// This allows for easier testing of the helper functions.
type testHelper interface {
	Helper()
	Skip(args ...any)
}

// skipIfNotIntegration skips the test if the INTEGRATION environment variable is not set to "1".
// This helper function allows integration tests to be separated from unit tests.
// Integration tests typically require external resources or full system setup.
//
// Usage:
//
//	func TestIntegrationFeature(t *testing.T) {
//	    skipIfNotIntegration(t)
//	    // Integration test code here
//	}
//
// To run integration tests:
//
//	INTEGRATION=1 go test ./...
func skipIfNotIntegration(t testing.TB) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("Skipping integration test (set INTEGRATION=1 to run)")
	}
}

// skipIfNotIntegrationWithHelper is an internal helper that accepts the testHelper interface.
// This is used for testing purposes.
func skipIfNotIntegrationWithHelper(t testHelper) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("Skipping integration test (set INTEGRATION=1 to run)")
	}
}
