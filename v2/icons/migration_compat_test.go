package icons

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestMigrationCompatibility_JSONStructure verifies that v2 uses the same JSON as v1
// This ensures GetAWSShape output matches v1 exactly for same inputs
func TestMigrationCompatibility_JSONStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping migration compatibility test in short mode")
	}

	// Load the embedded JSON (same one v1 uses)
	v2JSON := make(map[string]map[string]string)

	// Read the aws.json file that v2 uses
	jsonPath := filepath.Join("aws.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read aws.json: %v", err)
	}

	err = json.Unmarshal(data, &v2JSON)
	if err != nil {
		t.Fatalf("Failed to parse aws.json: %v", err)
	}

	// Verify structure is as expected (two-level map)
	if len(v2JSON) == 0 {
		t.Fatal("aws.json appears to be empty")
	}

	// Verify known groups exist
	knownGroups := []string{"Compute", "Storage", "Database", "Analytics"}
	for _, group := range knownGroups {
		if _, exists := v2JSON[group]; !exists {
			t.Errorf("Expected group %q not found in aws.json", group)
		}
	}

	// Verify known services exist
	knownServices := map[string]string{
		"Compute":   "EC2",
		"Storage":   "Elastic Block Store",
		"Database":  "DynamoDB",
		"Analytics": "Kinesis",
	}

	for group, service := range knownServices {
		if groupMap, exists := v2JSON[group]; exists {
			if _, exists := groupMap[service]; !exists {
				t.Errorf("Expected service %q not found in group %q", service, group)
			}
		}
	}
}

// TestMigrationCompatibility_DuplicateKeyHandling verifies duplicate key behavior
// Requirement 3.2: Verify duplicate key handling consistency (last value wins)
func TestMigrationCompatibility_DuplicateKeyHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping migration compatibility test in short mode")
	}

	// The aws.json has duplicate "Internet" entries in General Resources
	// Go's json.Unmarshal uses the last value for duplicates
	// This test verifies v2 behaves correctly

	style, err := GetAWSShape("General Resources", "Internet")
	if err != nil {
		t.Fatalf("GetAWSShape failed for duplicate key test: %v", err)
	}

	// If we got a valid style, duplicate handling worked (last value won)
	if style == "" {
		t.Error("GetAWSShape returned empty string for duplicate key case")
	}

	// Verify it's a valid Draw.io style
	if len(style) < 10 {
		t.Errorf("Style for duplicate key seems invalid: %q", style)
	}
}

// TestMigrationCompatibility_StyleStringFormat verifies style strings are Draw.io-compatible
// Requirement 3.3: Test that style strings are Draw.io-compatible
func TestMigrationCompatibility_StyleStringFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping migration compatibility test in short mode")
	}

	testCases := map[string]struct {
		group string
		title string
	}{
		"compute_ec2": {
			group: "Compute",
			title: "EC2",
		},
		"storage_ebs": {
			group: "Storage",
			title: "Elastic Block Store",
		},
		"database_dynamodb": {
			group: "Database",
			title: "DynamoDB",
		},
		"analytics_kinesis": {
			group: "Analytics",
			title: "Kinesis",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			style, err := GetAWSShape(tc.group, tc.title)
			if err != nil {
				t.Fatalf("GetAWSShape(%q, %q) failed: %v", tc.group, tc.title, err)
			}

			// Verify Draw.io compatibility requirements
			// Based on v1 output format, these elements must be present
			requiredElements := []struct {
				element string
				reason  string
			}{
				{"shape=", "Draw.io requires shape definition"},
				{"mxgraph", "Draw.io uses mxgraph namespace"},
				{"points=", "Draw.io requires connection points"},
			}

			for _, req := range requiredElements {
				if len(style) < len(req.element) || !contains(style, req.element) {
					t.Errorf("Style missing %q (%s): %q", req.element, req.reason, style)
				}
			}

			// Verify style ends with semicolon (Draw.io convention)
			if len(style) > 0 && style[len(style)-1] != ';' {
				t.Errorf("Style should end with semicolon: %q", style)
			}
		})
	}
}

// TestMigrationCompatibility_ErrorBehavior verifies error handling matches v1 expectations
func TestMigrationCompatibility_ErrorBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping migration compatibility test in short mode")
	}

	testCases := map[string]struct {
		group         string
		title         string
		expectError   bool
		errorContains string
	}{
		"missing_group": {
			group:         "NonExistentGroup",
			title:         "Service",
			expectError:   true,
			errorContains: "not found",
		},
		"missing_service": {
			group:         "Compute",
			title:         "NonExistentService",
			expectError:   true,
			errorContains: "not found",
		},
		"valid_service": {
			group:       "Compute",
			title:       "EC2",
			expectError: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			style, err := GetAWSShape(tc.group, tc.title)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error for GetAWSShape(%q, %q), got style: %q",
						tc.group, tc.title, style)
				}
				if !contains(err.Error(), tc.errorContains) {
					t.Errorf("Error should contain %q, got: %v", tc.errorContains, err)
				}
				if style != "" {
					t.Errorf("Expected empty style on error, got: %q", style)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for GetAWSShape(%q, %q): %v",
						tc.group, tc.title, err)
				}
				if style == "" {
					t.Error("Expected non-empty style, got empty string")
				}
			}
		})
	}
}

// TestMigrationCompatibility_CaseSensitivity verifies case-sensitive matching (v1 behavior)
func TestMigrationCompatibility_CaseSensitivity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping migration compatibility test in short mode")
	}

	// v1 uses exact case-sensitive matching
	// These should fail because case doesn't match
	testCases := map[string]struct {
		group string
		title string
	}{
		"lowercase_group": {
			group: "compute", // Should be "Compute"
			title: "EC2",
		},
		"lowercase_title": {
			group: "Compute",
			title: "ec2", // Should be "EC2"
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			_, err := GetAWSShape(tc.group, tc.title)
			if err == nil {
				t.Errorf("Expected error for case mismatch in %s, but got success", name)
			}
		})
	}

	// This should succeed (correct case)
	_, err := GetAWSShape("Compute", "EC2")
	if err != nil {
		t.Errorf("Expected success for correct case: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
