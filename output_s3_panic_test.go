package format

import (
	"testing"
)

// TestPrintByteSlice_S3BucketWithoutClient tests the bug where PrintByteSlice panics
// when S3Output has Bucket set but S3Client is nil (T-127)
func TestPrintByteSlice_S3BucketWithoutClient(t *testing.T) {
	// This test reproduces the panic condition described in T-127
	s3Output := S3Output{
		Bucket: "test-bucket",
		Path:   "test/path.txt",
		// S3Client is nil (zero value)
	}

	content := []byte("test content")

	// This should return an error, not panic
	err := PrintByteSlice(content, "", s3Output)

	if err == nil {
		t.Fatal("Expected error when S3Client is nil but Bucket is set, got nil")
	}

	// Verify the error message is helpful
	expectedSubstring := "S3Client is nil"
	if !containsString(err.Error(), expectedSubstring) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedSubstring, err.Error())
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsStringHelper(s, substr))))
}

func containsStringHelper(s, substr string) bool {
	for i := 1; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
