package icons

import (
	"slices"
	"strings"
	"sync"
	"testing"
)

// TestGetAWSShape_Success tests successful shape lookups
func TestGetAWSShape_Success(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
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

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := GetAWSShape(tc.group, tc.title)
			if err != nil {
				t.Fatalf("GetAWSShape(%q, %q) unexpected error: %v", tc.group, tc.title, err)
			}

			if got == "" {
				t.Errorf("GetAWSShape(%q, %q) returned empty string", tc.group, tc.title)
			}

			// Verify it's a Draw.io style string
			if !strings.Contains(got, "shape=") {
				t.Errorf("GetAWSShape(%q, %q) returned %q, expected Draw.io style string with 'shape='", tc.group, tc.title, got)
			}
		})
	}
}

// TestGetAWSShape_MissingGroup tests error handling for missing groups
func TestGetAWSShape_MissingGroup(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		group       string
		title       string
		wantErrText string
	}{
		"nonexistent_group": {
			group:       "NonExistentGroup",
			title:       "SomeService",
			wantErrText: `shape group "NonExistentGroup" not found`,
		},
		"empty_group": {
			group:       "",
			title:       "EC2",
			wantErrText: `shape group "" not found`,
		},
		"wrong_case_group": {
			group:       "compute",
			title:       "EC2",
			wantErrText: `shape group "compute" not found`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := GetAWSShape(tc.group, tc.title)
			if err == nil {
				t.Fatalf("GetAWSShape(%q, %q) expected error, got result: %q", tc.group, tc.title, got)
			}

			if got != "" {
				t.Errorf("GetAWSShape(%q, %q) expected empty string on error, got: %q", tc.group, tc.title, got)
			}

			if !strings.Contains(err.Error(), tc.wantErrText) {
				t.Errorf("GetAWSShape(%q, %q) error = %q, want error containing %q", tc.group, tc.title, err.Error(), tc.wantErrText)
			}
		})
	}
}

// TestGetAWSShape_MissingShape tests error handling for missing shapes within valid groups
func TestGetAWSShape_MissingShape(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		group       string
		title       string
		wantErrText string
	}{
		"nonexistent_shape_in_compute": {
			group:       "Compute",
			title:       "NonExistentService",
			wantErrText: `shape "NonExistentService" not found in group "Compute"`,
		},
		"empty_title": {
			group:       "Compute",
			title:       "",
			wantErrText: `shape "" not found in group "Compute"`,
		},
		"wrong_case_title": {
			group:       "Compute",
			title:       "ec2",
			wantErrText: `shape "ec2" not found in group "Compute"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := GetAWSShape(tc.group, tc.title)
			if err == nil {
				t.Fatalf("GetAWSShape(%q, %q) expected error, got result: %q", tc.group, tc.title, got)
			}

			if got != "" {
				t.Errorf("GetAWSShape(%q, %q) expected empty string on error, got: %q", tc.group, tc.title, got)
			}

			if !strings.Contains(err.Error(), tc.wantErrText) {
				t.Errorf("GetAWSShape(%q, %q) error = %q, want error containing %q", tc.group, tc.title, err.Error(), tc.wantErrText)
			}
		})
	}
}

// TestGetAWSShape_ThreadSafety tests concurrent access to GetAWSShape
func TestGetAWSShape_ThreadSafety(t *testing.T) {
	t.Parallel()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			_, _ = GetAWSShape("Compute", "EC2")
			_, _ = GetAWSShape("Storage", "S3")
			_, _ = GetAWSShape("Database", "DynamoDB")
		}()
	}

	wg.Wait()
}

// TestGetAWSShape_ConcurrentSameShape tests multiple goroutines accessing the same shape
// This specifically tests for race conditions when accessing the same map entries
func TestGetAWSShape_ConcurrentSameShape(t *testing.T) {
	t.Parallel()

	const numGoroutines = 1000
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// All goroutines access the same shape
	for range numGoroutines {
		go func() {
			defer wg.Done()
			shape, err := GetAWSShape("Compute", "EC2")
			if err != nil {
				t.Errorf("GetAWSShape failed: %v", err)
			}
			if shape == "" {
				t.Error("GetAWSShape returned empty shape")
			}
		}()
	}

	wg.Wait()
}

// TestGetAWSShape_ConcurrentDifferentShapes tests concurrent access to different shapes
func TestGetAWSShape_ConcurrentDifferentShapes(t *testing.T) {
	t.Parallel()

	// Test shapes from different groups - using only known valid shapes
	testCases := []struct {
		group string
		title string
	}{
		{"Compute", "EC2"},
		{"Storage", "Elastic Block Store"},
		{"Database", "DynamoDB"},
		{"Analytics", "Kinesis"},
	}

	const numIterations = 100
	var wg sync.WaitGroup
	wg.Add(len(testCases) * numIterations)

	for _, tc := range testCases {
		tc := tc // Capture for goroutine
		for range numIterations {
			go func() {
				defer wg.Done()
				shape, err := GetAWSShape(tc.group, tc.title)
				if err != nil {
					t.Errorf("GetAWSShape(%q, %q) failed: %v", tc.group, tc.title, err)
				}
				if shape == "" {
					t.Errorf("GetAWSShape(%q, %q) returned empty shape", tc.group, tc.title)
				}
			}()
		}
	}

	wg.Wait()
}

// TestGetAWSShape_RaceDetector tests for race conditions using go test -race
// This test is specifically designed to trigger race detector warnings if any exist
func TestGetAWSShape_RaceDetector(t *testing.T) {
	// Note: Run with `go test -race` to enable race detection
	t.Parallel()

	const numGoroutines = 50
	done := make(chan bool)

	// Start multiple readers
	for range numGoroutines {
		go func() {
			for i := 0; i < 100; i++ {
				_, _ = GetAWSShape("Compute", "EC2")
				_, _ = GetAWSShape("Storage", "Elastic Block Store")
			}
			done <- true
		}()
	}

	// Start helper function accessors
	for range 10 {
		go func() {
			for i := 0; i < 100; i++ {
				_ = AllAWSGroups()
				_ = HasAWSShape("Compute", "EC2")
				_, _ = AWSShapesInGroup("Storage")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for range numGoroutines + 10 {
		<-done
	}
}

// TestAllAWSGroups tests the helper function for listing all groups
func TestAllAWSGroups(t *testing.T) {
	t.Parallel()

	groups := AllAWSGroups()

	if len(groups) == 0 {
		t.Fatal("AllAWSGroups() returned empty slice")
	}

	// Verify alphabetical ordering
	for i := 1; i < len(groups); i++ {
		if groups[i-1] >= groups[i] {
			t.Errorf("AllAWSGroups() not in alphabetical order: %q >= %q at indices %d, %d", groups[i-1], groups[i], i-1, i)
		}
	}

	// Verify known groups are present
	knownGroups := []string{"Compute", "Storage", "Database", "Analytics"}
	for _, known := range knownGroups {
		if !slices.Contains(groups, known) {
			t.Errorf("AllAWSGroups() missing expected group: %q", known)
		}
	}
}

// TestAWSShapesInGroup tests the helper function for listing shapes in a group
func TestAWSShapesInGroup(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		group       string
		wantErr     bool
		wantMinSize int
	}{
		"compute_group": {
			group:       "Compute",
			wantErr:     false,
			wantMinSize: 1,
		},
		"storage_group": {
			group:       "Storage",
			wantErr:     false,
			wantMinSize: 1,
		},
		"nonexistent_group": {
			group:   "NonExistent",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := AWSShapesInGroup(tc.group)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("AWSShapesInGroup(%q) expected error, got shapes: %v", tc.group, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("AWSShapesInGroup(%q) unexpected error: %v", tc.group, err)
			}

			if len(got) < tc.wantMinSize {
				t.Errorf("AWSShapesInGroup(%q) returned %d shapes, want at least %d", tc.group, len(got), tc.wantMinSize)
			}

			// Verify alphabetical ordering
			for i := 1; i < len(got); i++ {
				if got[i-1] >= got[i] {
					t.Errorf("AWSShapesInGroup(%q) not in alphabetical order: %q >= %q at indices %d, %d", tc.group, got[i-1], got[i], i-1, i)
				}
			}
		})
	}
}

// TestHasAWSShape tests the helper function for checking shape existence
func TestHasAWSShape(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		group string
		title string
		want  bool
	}{
		"existing_shape": {
			group: "Compute",
			title: "EC2",
			want:  true,
		},
		"nonexistent_group": {
			group: "NonExistent",
			title: "EC2",
			want:  false,
		},
		"nonexistent_shape": {
			group: "Compute",
			title: "NonExistent",
			want:  false,
		},
		"empty_group": {
			group: "",
			title: "EC2",
			want:  false,
		},
		"empty_title": {
			group: "Compute",
			title: "",
			want:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := HasAWSShape(tc.group, tc.title)
			if got != tc.want {
				t.Errorf("HasAWSShape(%q, %q) = %v, want %v", tc.group, tc.title, got, tc.want)
			}
		})
	}
}

// BenchmarkGetAWSShape tests single lookup performance
func BenchmarkGetAWSShape(b *testing.B) {
	for b.Loop() {
		_, _ = GetAWSShape("Compute", "EC2")
	}
}

// BenchmarkGetAWSShape_NotFound benchmarks the error path for missing shapes
func BenchmarkGetAWSShape_NotFound(b *testing.B) {
	for b.Loop() {
		_, _ = GetAWSShape("NonExistent", "Service")
	}
}

// BenchmarkConcurrentLookups tests parallel access performance
func BenchmarkConcurrentLookups(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = GetAWSShape("Compute", "EC2")
		}
	})
}

// BenchmarkAllAWSGroups benchmarks the helper function performance
func BenchmarkAllAWSGroups(b *testing.B) {
	for b.Loop() {
		_ = AllAWSGroups()
	}
}

// BenchmarkAWSShapesInGroup benchmarks listing shapes in a group
func BenchmarkAWSShapesInGroup(b *testing.B) {
	for b.Loop() {
		_, _ = AWSShapesInGroup("Compute")
	}
}

// BenchmarkHasAWSShape benchmarks shape existence check
func BenchmarkHasAWSShape(b *testing.B) {
	for b.Loop() {
		_ = HasAWSShape("Compute", "EC2")
	}
}

// BenchmarkLargeScaleLookups verifies O(1) performance with many different lookups
func BenchmarkLargeScaleLookups(b *testing.B) {
	// Test with various services from different groups to verify O(1) behavior
	// Using known valid shapes from TestGetAWSShape_Success
	testCases := []struct {
		group string
		title string
	}{
		{"Compute", "EC2"},
		{"Storage", "Elastic Block Store"},
		{"Database", "DynamoDB"},
		{"Analytics", "Kinesis"},
	}

	b.ResetTimer()
	for b.Loop() {
		// Access different shapes in rotation to test map performance
		for _, tc := range testCases {
			_, _ = GetAWSShape(tc.group, tc.title)
		}
	}
}

// Example demonstrates basic usage of GetAWSShape
func Example() {
	// Get a specific AWS icon
	style, err := GetAWSShape("Compute", "EC2")
	if err != nil {
		panic(err)
	}

	// The style can be used in Draw.io diagrams
	_ = style

	// Output:
}

// ExampleGetAWSShape demonstrates retrieving an AWS icon
func ExampleGetAWSShape() {
	style, err := GetAWSShape("Compute", "EC2")
	if err != nil {
		panic(err)
	}

	// Use the style in your Draw.io diagram
	_ = style

	// Output:
}

// ExampleGetAWSShape_errorHandling demonstrates proper error handling
func ExampleGetAWSShape_errorHandling() {
	style, err := GetAWSShape("Compute", "NonExistentService")
	if err != nil {
		// Handle the error - shape not found
		_ = err.Error()
		return
	}

	// Use the style
	_ = style

	// Output:
}

// ExampleAllAWSGroups demonstrates discovering available service groups
func ExampleAllAWSGroups() {
	groups := AllAWSGroups()

	// Print first few groups (alphabetically sorted)
	for i, group := range groups {
		if i >= 3 {
			break
		}
		_ = group
	}

	// Output:
}

// ExampleAWSShapesInGroup demonstrates discovering shapes within a group
func ExampleAWSShapesInGroup() {
	shapes, err := AWSShapesInGroup("Compute")
	if err != nil {
		panic(err)
	}

	// Print first few shapes (alphabetically sorted)
	for i, shape := range shapes {
		if i >= 3 {
			break
		}
		_ = shape
	}

	// Output:
}

// ExampleHasAWSShape demonstrates checking if a shape exists
func ExampleHasAWSShape() {
	if HasAWSShape("Compute", "EC2") {
		style, _ := GetAWSShape("Compute", "EC2")
		_ = style
	}

	// Output:
}

// ExampleGetAWSShape_drawioIntegration demonstrates using AWS icons with Draw.io diagrams
func ExampleGetAWSShape_drawioIntegration() {
	// Prepare data with AWS service information
	data := []map[string]any{
		{"Name": "Web Server", "ServiceType": "EC2", "ServiceGroup": "Compute"},
		{"Name": "Database", "ServiceType": "RDS", "ServiceGroup": "Database"},
		{"Name": "File Storage", "ServiceType": "S3", "ServiceGroup": "Storage"},
	}

	// Add AWS icon styles to each record
	for _, record := range data {
		group := record["ServiceGroup"].(string)
		serviceType := record["ServiceType"].(string)

		if style, err := GetAWSShape(group, serviceType); err == nil {
			record["AWSIcon"] = style
		}
	}

	// The AWSIcon field can now be used in Draw.io diagrams with placeholders
	// For example, in DrawIOHeader: Style: "%AWSIcon%"
	_ = data

	// Output:
}
