package icons

import (
	"testing"
	"time"
)

// This test verifies that package initialization (JSON parsing) happens quickly
func TestStartupTime(t *testing.T) {
	// The package is already initialized when this test runs,
	// but we can measure the time to access the data structure
	start := time.Now()

	// Access all the data to ensure it's fully loaded
	groups := AllAWSGroups()
	for _, g := range groups {
		_, _ = AWSShapesInGroup(g)
	}

	elapsed := time.Since(start)
	t.Logf("Time to access all groups and shapes: %v", elapsed)

	// Per design requirements, parsing should be 5-10ms
	// Accessing the already-parsed data should be much faster
	if elapsed > 100*time.Millisecond {
		t.Errorf("Data access took too long: %v (expected < 100ms)", elapsed)
	}
}

// Note: We cannot benchmark initialization time directly because
// awsRaw is set to nil after init() to save memory. The init()
// function parses the 225KB JSON file once at startup, which per
// the design requirements should take 5-10ms.
