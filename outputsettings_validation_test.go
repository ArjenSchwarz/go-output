package format

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/ArjenSchwarz/go-output/errors"
	"github.com/ArjenSchwarz/go-output/mermaid"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Test comprehensive OutputSettings validation
func TestOutputSettings_Validate(t *testing.T) {
	testCases := []struct {
		name          string
		setupSettings func() *OutputSettings
		expectedError bool
		expectedCode  errors.ErrorCode
		expectedField string
	}{
		{
			name: "valid JSON format",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				return settings
			},
			expectedError: false,
		},
		{
			name: "valid table format",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("table")
				return settings
			},
			expectedError: false,
		},
		{
			name: "empty output format",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.OutputFormat = ""
				return settings
			},
			expectedError: true,
			expectedCode:  errors.ErrMissingRequired,
			expectedField: "OutputFormat",
		},
		{
			name: "invalid output format",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("invalid-format")
				return settings
			},
			expectedError: true,
			expectedCode:  errors.ErrInvalidFormat,
			expectedField: "OutputFormat",
		},
		{
			name: "mermaid format with FromToColumns (valid)",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid")
				settings.AddFromToColumns("source", "target")
				return settings
			},
			expectedError: false,
		},
		{
			name: "mermaid format with MermaidSettings (valid)",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid")
				settings.MermaidSettings = &mermaid.Settings{ChartType: "flowchart"}
				return settings
			},
			expectedError: false,
		},
		{
			name: "mermaid format without required configuration",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid")
				return settings
			},
			expectedError: true,
			expectedCode:  errors.ErrMissingRequired,
		},
		{
			name: "dot format with FromToColumns (valid)",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("dot")
				settings.AddFromToColumns("source", "target")
				return settings
			},
			expectedError: false,
		},
		{
			name: "dot format without FromToColumns",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("dot")
				return settings
			},
			expectedError: true,
			expectedCode:  errors.ErrMissingRequired,
		},
		{
			name: "drawio format with DrawIOHeader (valid)",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("drawio")
				settings.DrawIOHeader = drawio.NewHeader("Name", "shape=rectangle", "")
				return settings
			},
			expectedError: false,
		},
		{
			name: "drawio format without DrawIOHeader",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("drawio")
				return settings
			},
			expectedError: true,
			expectedCode:  errors.ErrMissingRequired,
		},
		{
			name: "valid file output to existing directory",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				// Use current directory which should exist
				settings.OutputFile = "./test-output.json"
				return settings
			},
			expectedError: false,
		},
		{
			name: "file output to non-existent directory",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				settings.OutputFile = "/non-existent-directory/test-output.json"
				return settings
			},
			expectedError: true,
			expectedCode:  errors.ErrInvalidFilePath,
			expectedField: "OutputFile",
		},
		{
			name: "valid S3 configuration",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				settings.S3Bucket.Bucket = "test-bucket"
				settings.S3Bucket.Path = "output/data.json"
				settings.S3Bucket.S3Client = &s3.Client{} // Mock client
				return settings
			},
			expectedError: false,
		},
		{
			name: "S3 bucket without path",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				settings.S3Bucket.Bucket = "test-bucket"
				settings.S3Bucket.S3Client = &s3.Client{}
				// Missing path
				return settings
			},
			expectedError: true,
			expectedCode:  errors.ErrMissingRequired,
			expectedField: "S3Bucket.Path",
		},
		{
			name: "S3 bucket without client",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				settings.S3Bucket.Bucket = "test-bucket"
				settings.S3Bucket.Path = "output/data.json"
				// Missing S3Client
				return settings
			},
			expectedError: true,
			expectedCode:  errors.ErrMissingRequired,
			expectedField: "S3Bucket.S3Client",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			settings := tc.setupSettings()
			err := settings.Validate()

			if tc.expectedError {
				if err == nil {
					t.Error("Expected validation error, got nil")
					return
				}

				// Check if it's the expected error type
				if outputErr, ok := err.(errors.OutputError); ok {
					if tc.expectedCode != "" && outputErr.Code() != tc.expectedCode {
						t.Errorf("Expected error code %s, got %s", tc.expectedCode, outputErr.Code())
					}

					// Check field context if specified
					if tc.expectedField != "" {
						context := outputErr.Context()
						if context.Field != tc.expectedField {
							t.Errorf("Expected error field %s, got %s", tc.expectedField, context.Field)
						}
					}
				} else {
					t.Errorf("Expected OutputError, got %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// Test cross-field validation scenarios
func TestOutputSettings_CrossFieldValidation(t *testing.T) {
	testCases := []struct {
		name          string
		setupSettings func() *OutputSettings
		expectedError bool
		expectedCode  errors.ErrorCode
		description   string
	}{
		{
			name: "both file and S3 output specified",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				settings.OutputFile = "./test-output.json"
				settings.S3Bucket.Bucket = "test-bucket"
				settings.S3Bucket.Path = "output/data.json"
				settings.S3Bucket.S3Client = &s3.Client{}
				return settings
			},
			expectedError: true,
			expectedCode:  errors.ErrIncompatibleConfig,
			description:   "Should not allow both file and S3 output",
		},
		{
			name: "file output with different file format",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				settings.OutputFile = "./test-output.json"
				settings.OutputFileFormat = "csv"
				return settings
			},
			expectedError: false,
			description:   "Should allow different file format",
		},
		{
			name: "empty OutputFileFormat defaults to OutputFormat",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")
				settings.OutputFile = "./test-output.json"
				// OutputFileFormat empty, should default to OutputFormat
				return settings
			},
			expectedError: false,
			description:   "Should handle empty OutputFileFormat",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			settings := tc.setupSettings()
			err := settings.Validate()

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected validation error for %s, got nil", tc.description)
					return
				}

				if tc.expectedCode != "" {
					if outputErr, ok := err.(errors.OutputError); ok {
						if outputErr.Code() != tc.expectedCode {
							t.Errorf("Expected error code %s, got %s", tc.expectedCode, outputErr.Code())
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got: %v", tc.description, err)
				}
			}
		})
	}
}

// Test validation error messages and suggestions
func TestOutputSettings_ValidationErrorMessages(t *testing.T) {
	testCases := []struct {
		name               string
		setupSettings      func() *OutputSettings
		expectedMessage    string
		expectedSuggestion string
	}{
		{
			name: "invalid format with suggestions",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("xml")
				return settings
			},
			expectedMessage:    "Invalid output format: xml",
			expectedSuggestion: "json",
		},
		{
			name: "mermaid without configuration",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid")
				return settings
			},
			expectedMessage:    "mermaid format requires FromToColumns or MermaidSettings",
			expectedSuggestion: "AddFromToColumns",
		},
		{
			name: "dot without FromToColumns",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("dot")
				return settings
			},
			expectedMessage:    "dot format requires FromToColumns configuration",
			expectedSuggestion: "AddFromToColumns",
		},
		{
			name: "drawio without header",
			setupSettings: func() *OutputSettings {
				settings := NewOutputSettings()
				settings.SetOutputFormat("drawio")
				return settings
			},
			expectedMessage:    "drawio format requires DrawIOHeader configuration",
			expectedSuggestion: "Configure DrawIOHeader",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			settings := tc.setupSettings()
			err := settings.Validate()

			if err == nil {
				t.Error("Expected validation error, got nil")
				return
			}

			errorMessage := err.Error()
			if tc.expectedMessage != "" {
				if errorMessage != "" && !contains(errorMessage, tc.expectedMessage) {
					t.Errorf("Expected error message to contain '%s', got: %s", tc.expectedMessage, errorMessage)
				}
			}

			// Check for suggestions in ValidationError
			if validationErr, ok := err.(errors.ValidationError); ok {
				suggestions := validationErr.Suggestions()
				if tc.expectedSuggestion != "" && len(suggestions) > 0 {
					found := false
					for _, suggestion := range suggestions {
						if contains(suggestion, tc.expectedSuggestion) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected suggestion to contain '%s', got: %v", tc.expectedSuggestion, suggestions)
					}
				}
			}
		})
	}
}

// Test file path validation with temporary directories
func TestOutputSettings_FileValidation(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "outputsettings_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testCases := []struct {
		name          string
		setupPath     func() string
		expectedError bool
	}{
		{
			name: "valid path in existing directory",
			setupPath: func() string {
				return filepath.Join(tempDir, "output.json")
			},
			expectedError: false,
		},
		{
			name: "path in non-existent directory",
			setupPath: func() string {
				return filepath.Join(tempDir, "non-existent", "output.json")
			},
			expectedError: true,
		},
		{
			name: "relative path in current directory",
			setupPath: func() string {
				return "output.json"
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			settings := NewOutputSettings()
			settings.SetOutputFormat("json")
			settings.OutputFile = tc.setupPath()

			err := settings.Validate()

			if tc.expectedError {
				if err == nil {
					t.Error("Expected validation error for file path, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for valid file path, got: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}
