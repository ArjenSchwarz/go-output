package drawio

import "testing"

// These tests verify the behavior of the AWS shape retrieval helpers.

// TestGetAWSShapeFound ensures that GetAWSShape returns a shape when the
// requested group and title exist.
func TestGetAWSShapeFound(t *testing.T) {
	shape, err := GetAWSShape("Analytics", "Athena")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if shape == "" {
		t.Fatalf("expected shape, got empty string")
	}
}

// TestGetAWSShapeGroupMissing verifies that an error is returned when the
// requested shape group does not exist.
func TestGetAWSShapeGroupMissing(t *testing.T) {
	if _, err := GetAWSShape("UnknownGroup", "Foo"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// TestGetAWSShapeTitleMissing verifies that an error is returned when the
// requested shape title is missing from an existing group.
func TestGetAWSShapeTitleMissing(t *testing.T) {
	if _, err := GetAWSShape("Analytics", "UnknownShape"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// TestAWSShapeCompatibility confirms that the legacy AWSShape function still
// returns a shape when available and an empty string when it is not.
func TestAWSShapeCompatibility(t *testing.T) {
	if s := AWSShape("Analytics", "Athena"); s == "" {
		t.Fatalf("expected shape, got empty string")
	}
	if s := AWSShape("UnknownGroup", "Foo"); s != "" {
		t.Fatalf("expected empty string for unknown shape, got %s", s)
	}
}
