package validators

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ArjenSchwarz/go-output/errors"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Mock types are defined in mock_types.go

func TestFormatValidator(t *testing.T) {
	tests := []struct {
		name           string
		allowedFormats []string
		format         string
		expectedError  bool
		expectedCode   errors.ErrorCode
	}{
		{
			name:           "Valid format - json",
			allowedFormats: []string{"json", "csv", "table", "html", "markdown", "mermaid", "drawio", "dot", "yaml"},
			format:         "json",
			expectedError:  false,
		},
		{
			name:           "Valid format - case insensitive",
			allowedFormats: []string{"json", "csv", "table"},
			format:         "JSON",
			expectedError:  false,
		},
		{
			name:           "Invalid format",
			allowedFormats: []string{"json", "csv", "table"},
			format:         "xml",
			expectedError:  true,
			expectedCode:   errors.ErrInvalidFormat,
		},
		{
			name:           "Empty format",
			allowedFormats: []string{"json", "csv"},
			format:         "",
			expectedError:  true,
			expectedCode:   errors.ErrInvalidFormat,
		},
		{
			name:           "All standard formats allowed",
			allowedFormats: []string{"json", "csv", "table", "html", "markdown", "mermaid", "drawio", "dot", "yaml"},
			format:         "mermaid",
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewFormatValidator(tt.allowedFormats...)

			mockSettings := &mockOutputSettings{
				OutputFormat: tt.format,
			}

			err := validator.Validate(mockSettings)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, validationErr.Code())
				}

				violations := validationErr.Violations()
				if len(violations) == 0 {
					t.Errorf("Expected violations, got none")
				} else {
					if violations[0].Field != "OutputFormat" {
						t.Errorf("Expected violation for OutputFormat field, got %s", violations[0].Field)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestFormatValidatorName(t *testing.T) {
	validator := NewFormatValidator("json", "csv")
	expected := "FormatValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}

func TestFormatValidatorWithInvalidInput(t *testing.T) {
	validator := NewFormatValidator("json")

	err := validator.Validate("invalid input")
	if err == nil {
		t.Error("Expected error for invalid input type")
	}

	validationErr, ok := err.(errors.ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	}

	if validationErr.Code() != errors.ErrInvalidDataType {
		t.Errorf("Expected error code %s, got %s", errors.ErrInvalidDataType, validationErr.Code())
	}
}

func TestFilePathValidator(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	validPath := filepath.Join(tempDir, "test.json")
	readOnlyDir := filepath.Join(tempDir, "readonly")

	// Create read-only directory (if possible)
	os.Mkdir(readOnlyDir, 0444)
	defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

	tests := []struct {
		name          string
		outputFile    string
		expectedError bool
		expectedCode  errors.ErrorCode
	}{
		{
			name:          "Valid file path",
			outputFile:    validPath,
			expectedError: false,
		},
		{
			name:          "Empty file path (no validation needed)",
			outputFile:    "",
			expectedError: false,
		},
		{
			name:          "Non-existent directory",
			outputFile:    "/nonexistent/directory/file.json",
			expectedError: true,
			expectedCode:  errors.ErrInvalidFilePath,
		},
		{
			name:          "Read-only directory (permission check)",
			outputFile:    filepath.Join(readOnlyDir, "file.json"),
			expectedError: true,
			expectedCode:  errors.ErrInvalidFilePath,
		},
		{
			name:          "Relative path",
			outputFile:    "output.json",
			expectedError: false, // Current directory should be writable in tests
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewFilePathValidator()

			mockSettings := &mockOutputSettings{
				OutputFile: tt.outputFile,
			}

			err := validator.Validate(mockSettings)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, validationErr.Code())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestFilePathValidatorName(t *testing.T) {
	validator := NewFilePathValidator()
	expected := "FilePathValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}

func TestS3ConfigValidator(t *testing.T) {
	tests := []struct {
		name          string
		s3Config      mockS3Output
		expectedError bool
		expectedCode  errors.ErrorCode
	}{
		{
			name: "Valid S3 configuration",
			s3Config: mockS3Output{
				S3Client: &s3.Client{}, // Mock client
				Bucket:   "my-bucket",
				Path:     "path/to/file.json",
			},
			expectedError: false,
		},
		{
			name: "Empty bucket (no validation needed)",
			s3Config: mockS3Output{
				Bucket: "",
			},
			expectedError: false,
		},
		{
			name: "Missing S3 client",
			s3Config: mockS3Output{
				S3Client: nil,
				Bucket:   "my-bucket",
				Path:     "path/to/file.json",
			},
			expectedError: true,
			expectedCode:  errors.ErrCompositeValidation,
		},
		{
			name: "Invalid bucket name (contains uppercase)",
			s3Config: mockS3Output{
				S3Client: &s3.Client{},
				Bucket:   "My-Bucket",
				Path:     "path/to/file.json",
			},
			expectedError: true,
			expectedCode:  errors.ErrCompositeValidation,
		},
		{
			name: "Invalid bucket name (too short)",
			s3Config: mockS3Output{
				S3Client: &s3.Client{},
				Bucket:   "ab",
				Path:     "path/to/file.json",
			},
			expectedError: true,
			expectedCode:  errors.ErrCompositeValidation,
		},
		{
			name: "Empty path",
			s3Config: mockS3Output{
				S3Client: &s3.Client{},
				Bucket:   "my-bucket",
				Path:     "",
			},
			expectedError: true,
			expectedCode:  errors.ErrCompositeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewS3ConfigValidator()

			mockSettings := &mockOutputSettings{
				S3Bucket: tt.s3Config,
			}

			err := validator.Validate(mockSettings)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, validationErr.Code())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestS3ConfigValidatorName(t *testing.T) {
	validator := NewS3ConfigValidator()
	expected := "S3ConfigValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}

func TestCompatibilityValidator(t *testing.T) {
	tests := []struct {
		name          string
		settings      mockOutputSettings
		expectedError bool
		expectedCode  errors.ErrorCode
		description   string
	}{
		{
			name: "Valid mermaid with FromToColumns",
			settings: mockOutputSettings{
				OutputFormat: "mermaid",
				FromToColumns: &mockFromToColumns{
					From: "Source",
					To:   "Target",
				},
			},
			expectedError: false,
			description:   "Mermaid with FromToColumns should be valid",
		},
		{
			name: "Valid mermaid with MermaidSettings",
			settings: mockOutputSettings{
				OutputFormat: "mermaid",
				MermaidSettings: &mockMermaidSettings{
					ChartType: "flowchart",
				},
			},
			expectedError: false,
			description:   "Mermaid with MermaidSettings should be valid",
		},
		{
			name: "Invalid mermaid - missing both FromToColumns and MermaidSettings",
			settings: mockOutputSettings{
				OutputFormat:    "mermaid",
				FromToColumns:   nil,
				MermaidSettings: nil,
			},
			expectedError: true,
			expectedCode:  errors.ErrCompositeValidation,
			description:   "Mermaid requires either FromToColumns or MermaidSettings",
		},
		{
			name: "Valid dot with FromToColumns",
			settings: mockOutputSettings{
				OutputFormat: "dot",
				FromToColumns: &mockFromToColumns{
					From: "Source",
					To:   "Target",
				},
			},
			expectedError: false,
			description:   "DOT format with FromToColumns should be valid",
		},
		{
			name: "Invalid dot - missing FromToColumns",
			settings: mockOutputSettings{
				OutputFormat:  "dot",
				FromToColumns: nil,
			},
			expectedError: true,
			expectedCode:  errors.ErrCompositeValidation,
			description:   "DOT format requires FromToColumns",
		},
		{
			name: "Valid json format (no special requirements)",
			settings: mockOutputSettings{
				OutputFormat: "json",
			},
			expectedError: false,
			description:   "JSON format has no special requirements",
		},
		{
			name: "Valid table format (no special requirements)",
			settings: mockOutputSettings{
				OutputFormat: "table",
			},
			expectedError: false,
			description:   "Table format has no special requirements",
		},
		{
			name: "Both file and S3 output specified",
			settings: mockOutputSettings{
				OutputFormat: "json",
				OutputFile:   "output.json",
				S3Bucket: mockS3Output{
					Bucket: "my-bucket",
					Path:   "path/to/file.json",
				},
			},
			expectedError: true,
			expectedCode:  errors.ErrCompositeValidation,
			description:   "Cannot specify both file and S3 output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewCompatibilityValidator()

			err := validator.Validate(&tt.settings)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil. %s", tt.description)
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T. %s", err, tt.description)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s. %s", tt.expectedCode, validationErr.Code(), tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v. %s", err, tt.description)
				}
			}
		})
	}
}

func TestCompatibilityValidatorName(t *testing.T) {
	validator := NewCompatibilityValidator()
	expected := "CompatibilityValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}

func TestMermaidValidator(t *testing.T) {
	tests := []struct {
		name          string
		settings      mockOutputSettings
		expectedError bool
		expectedCode  errors.ErrorCode
	}{
		{
			name: "Valid mermaid flowchart",
			settings: mockOutputSettings{
				OutputFormat: "mermaid",
				FromToColumns: &mockFromToColumns{
					From: "Source",
					To:   "Target",
				},
			},
			expectedError: false,
		},
		{
			name: "Valid mermaid piechart",
			settings: mockOutputSettings{
				OutputFormat: "mermaid",
				MermaidSettings: &mockMermaidSettings{
					ChartType: "piechart",
				},
			},
			expectedError: false,
		},
		{
			name: "Invalid mermaid chart type",
			settings: mockOutputSettings{
				OutputFormat: "mermaid",
				MermaidSettings: &mockMermaidSettings{
					ChartType: "invalidchart",
				},
			},
			expectedError: true,
			expectedCode:  errors.ErrInvalidFormat,
		},
		{
			name: "Non-mermaid format (should pass)",
			settings: mockOutputSettings{
				OutputFormat: "json",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewMermaidValidator()

			err := validator.Validate(&tt.settings)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, validationErr.Code())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestMermaidValidatorName(t *testing.T) {
	validator := NewMermaidValidator()
	expected := "MermaidValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}

func TestDotValidator(t *testing.T) {
	tests := []struct {
		name          string
		settings      mockOutputSettings
		expectedError bool
		expectedCode  errors.ErrorCode
	}{
		{
			name: "Valid dot configuration",
			settings: mockOutputSettings{
				OutputFormat: "dot",
				FromToColumns: &mockFromToColumns{
					From: "Source",
					To:   "Target",
				},
			},
			expectedError: false,
		},
		{
			name: "Invalid dot - missing FromToColumns",
			settings: mockOutputSettings{
				OutputFormat:  "dot",
				FromToColumns: nil,
			},
			expectedError: true,
			expectedCode:  errors.ErrMissingRequired,
		},
		{
			name: "Non-dot format (should pass)",
			settings: mockOutputSettings{
				OutputFormat: "json",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewDotValidator()

			err := validator.Validate(&tt.settings)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, validationErr.Code())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestDotValidatorName(t *testing.T) {
	validator := NewDotValidator()
	expected := "DotValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}
