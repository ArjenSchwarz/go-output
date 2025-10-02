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
