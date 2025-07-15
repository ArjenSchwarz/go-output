package validators

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ArjenSchwarz/go-output/errors"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	formatMermaid = "mermaid"
	formatDot     = "dot"
)

// OutputSettings interface defines the methods required for configuration validation
// This interface allows validators to work with different OutputSettings implementations
type OutputSettings interface {
	GetOutputFormat() string
	GetOutputFile() string
	GetS3Bucket() S3Output
	GetFromToColumns() *FromToColumns
	GetMermaidSettings() *MermaidSettings
}

// S3Output interface defines the methods required for S3 configuration validation
type S3Output interface {
	GetS3Client() *s3.Client
	GetBucket() string
	GetPath() string
}

// FromToColumns interface defines the methods required for from-to column validation
type FromToColumns interface {
	GetFrom() string
	GetTo() string
	GetLabel() string
}

// MermaidSettings interface defines the methods required for Mermaid configuration validation
type MermaidSettings interface {
	GetChartType() string
}

// FormatValidator validates that the output format is one of the allowed formats
type FormatValidator struct {
	allowedFormats map[string]bool
}

// NewFormatValidator creates a new FormatValidator with the specified allowed formats
func NewFormatValidator(allowedFormats ...string) *FormatValidator {
	formatMap := make(map[string]bool)
	for _, format := range allowedFormats {
		formatMap[strings.ToLower(format)] = true
	}
	return &FormatValidator{
		allowedFormats: formatMap,
	}
}

// Validate checks if the output format is in the list of allowed formats
func (v *FormatValidator) Validate(subject interface{}) error {
	settings, ok := subject.(OutputSettings)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"FormatValidator requires an OutputSettings",
		)
	}

	return v.validateSettings(settings)
}

// validateSettings validates real OutputSettings
func (v *FormatValidator) validateSettings(settings OutputSettings) error {
	return v.validateFormat(settings.GetOutputFormat())
}

// validateFormat validates the format string
func (v *FormatValidator) validateFormat(format string) error {
	if format == "" {
		return v.createFormatError(format, "Output format cannot be empty")
	}

	normalizedFormat := strings.ToLower(format)
	if !v.allowedFormats[normalizedFormat] {
		allowedList := make([]string, 0, len(v.allowedFormats))
		for format := range v.allowedFormats {
			allowedList = append(allowedList, format)
		}
		return v.createFormatError(format, fmt.Sprintf("Unsupported output format '%s'", format)).
			WithSuggestions(fmt.Sprintf("Supported formats: %s", strings.Join(allowedList, ", "))).(errors.ValidationError)
	}

	return nil
}

// createFormatError creates a validation error for format issues
func (v *FormatValidator) createFormatError(format, message string) errors.ValidationError {
	return errors.NewValidationErrorWithViolations(
		errors.ErrInvalidFormat,
		message,
		errors.Violation{
			Field:      "OutputFormat",
			Value:      format,
			Constraint: "allowed_formats",
			Message:    message,
		},
	)
}

// Name returns the validator name
func (v *FormatValidator) Name() string {
	return "FormatValidator"
}

// FilePathValidator validates file paths and permissions
type FilePathValidator struct{}

// NewFilePathValidator creates a new FilePathValidator
func NewFilePathValidator() *FilePathValidator {
	return &FilePathValidator{}
}

// Validate checks if the file path is valid and writable
func (v *FilePathValidator) Validate(subject interface{}) error {
	// Handle the mock structure for testing
	if mockSettings, ok := subject.(*mockOutputSettings); ok {
		return v.validateMockSettings(mockSettings)
	}

	// Handle the real OutputSettings structure
	settings, ok := subject.(OutputSettings)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"FilePathValidator requires an OutputSettings",
		)
	}

	return v.validateSettings(settings)
}

// validateMockSettings validates mock settings for testing
func (v *FilePathValidator) validateMockSettings(settings *mockOutputSettings) error {
	return v.validateFilePath(settings.OutputFile)
}

// validateSettings validates real OutputSettings
func (v *FilePathValidator) validateSettings(settings OutputSettings) error {
	return v.validateFilePath(settings.GetOutputFile())
}

// validateFilePath validates the file path
func (v *FilePathValidator) validateFilePath(outputFile string) error {
	if outputFile == "" {
		return nil // Empty file path is valid (output to stdout)
	}

	// Check if the directory exists and is writable
	dir := filepath.Dir(outputFile)

	// Convert relative paths to absolute for validation
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return v.createFilePathError(outputFile, "Invalid file path")
	}

	// Check if directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return v.createFilePathError(outputFile, fmt.Sprintf("Directory does not exist: %s", dir))
	}

	// Check if directory is writable by attempting to create a temporary file
	tempFile := filepath.Join(absDir, ".go-output-permission-test")
	if file, err := os.Create(tempFile); err != nil {
		return v.createFilePathError(outputFile, fmt.Sprintf("Directory is not writable: %s", dir))
	} else {
		_ = file.Close()
		_ = os.Remove(tempFile) // Clean up
	}

	return nil
}

// createFilePathError creates a validation error for file path issues
func (v *FilePathValidator) createFilePathError(filePath, message string) errors.ValidationError {
	return errors.NewValidationErrorWithViolations(
		errors.ErrInvalidFilePath,
		message,
		errors.Violation{
			Field:      "OutputFile",
			Value:      filePath,
			Constraint: "writable_path",
			Message:    message,
		},
	)
}

// Name returns the validator name
func (v *FilePathValidator) Name() string {
	return "FilePathValidator"
}

// S3ConfigValidator validates S3 bucket and key configurations
type S3ConfigValidator struct{}

// NewS3ConfigValidator creates a new S3ConfigValidator
func NewS3ConfigValidator() *S3ConfigValidator {
	return &S3ConfigValidator{}
}

// Validate checks if S3 configuration is valid
func (v *S3ConfigValidator) Validate(subject interface{}) error {
	// Handle the mock structure for testing
	if mockSettings, ok := subject.(*mockOutputSettings); ok {
		return v.validateMockSettings(mockSettings)
	}

	// Handle the real OutputSettings structure
	settings, ok := subject.(OutputSettings)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"S3ConfigValidator requires an OutputSettings",
		)
	}

	return v.validateSettings(settings)
}

// validateMockSettings validates mock settings for testing
func (v *S3ConfigValidator) validateMockSettings(settings *mockOutputSettings) error {
	s3Config := settings.S3Bucket

	// If no bucket specified, no validation needed
	if s3Config.Bucket == "" {
		return nil
	}

	return v.validateS3Config(s3Config.S3Client, s3Config.Bucket, s3Config.Path)
}

// validateSettings validates real OutputSettings
func (v *S3ConfigValidator) validateSettings(settings OutputSettings) error {
	s3Config := settings.GetS3Bucket()

	// If no bucket specified, no validation needed
	if s3Config.GetBucket() == "" {
		return nil
	}

	return v.validateS3Config(s3Config.GetS3Client(), s3Config.GetBucket(), s3Config.GetPath())
}

// validateS3Config validates S3 configuration parameters
func (v *S3ConfigValidator) validateS3Config(client *s3.Client, bucket, path string) error {
	composite := errors.NewCompositeError()

	// Validate S3 client
	if client == nil {
		composite.Add(errors.NewValidationErrorWithViolations(
			errors.ErrIncompatibleConfig,
			"S3 client is required when S3 bucket is specified",
			errors.Violation{
				Field:      "S3Bucket.S3Client",
				Value:      nil,
				Constraint: "required",
				Message:    "S3 client cannot be nil",
			},
		))
	}

	// Validate bucket name according to AWS S3 naming rules
	if err := v.validateBucketName(bucket); err != nil {
		if validationErr, ok := err.(errors.ValidationError); ok {
			composite.Add(validationErr)
		} else {
			// Wrap regular errors as validation errors
			wrappedErr := errors.NewValidationError(
				errors.ErrIncompatibleConfig,
				"S3 bucket validation failed: "+err.Error(),
			)
			composite.Add(wrappedErr)
		}
	}

	// Validate path/key
	if path == "" {
		composite.Add(errors.NewValidationErrorWithViolations(
			errors.ErrMissingRequired,
			"S3 path/key is required",
			errors.Violation{
				Field:      "S3Bucket.Path",
				Value:      path,
				Constraint: "required",
				Message:    "S3 path cannot be empty",
			},
		))
	}

	return composite.ErrorOrNil()
}

// validateBucketName validates S3 bucket name according to AWS naming rules
func (v *S3ConfigValidator) validateBucketName(bucket string) error {
	// AWS S3 bucket naming rules (simplified)
	if len(bucket) < 3 || len(bucket) > 63 {
		return errors.NewValidationErrorWithViolations(
			errors.ErrIncompatibleConfig,
			"Invalid S3 bucket name length",
			errors.Violation{
				Field:      "S3Bucket.Bucket",
				Value:      bucket,
				Constraint: "length:3-63",
				Message:    "Bucket name must be between 3 and 63 characters",
			},
		)
	}

	// Check for lowercase and valid characters
	validBucketName := regexp.MustCompile(`^[a-z0-9.-]+$`)
	if !validBucketName.MatchString(bucket) {
		return errors.NewValidationErrorWithViolations(
			errors.ErrIncompatibleConfig,
			"Invalid S3 bucket name format",
			errors.Violation{
				Field:      "S3Bucket.Bucket",
				Value:      bucket,
				Constraint: "format:aws_bucket",
				Message:    "Bucket name must contain only lowercase letters, numbers, dots, and hyphens",
			},
		)
	}

	return nil
}

// Name returns the validator name
func (v *S3ConfigValidator) Name() string {
	return "S3ConfigValidator"
}

// CompatibilityValidator checks for incompatible setting combinations
type CompatibilityValidator struct{}

// NewCompatibilityValidator creates a new CompatibilityValidator
func NewCompatibilityValidator() *CompatibilityValidator {
	return &CompatibilityValidator{}
}

// Validate checks for incompatible configuration combinations
func (v *CompatibilityValidator) Validate(subject interface{}) error {
	// Handle the mock structure for testing
	if mockSettings, ok := subject.(*mockOutputSettings); ok {
		return v.validateMockSettings(mockSettings)
	}

	// Handle the real OutputSettings structure
	settings, ok := subject.(OutputSettings)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"CompatibilityValidator requires an OutputSettings",
		)
	}

	return v.validateSettings(settings)
}

// validateMockSettings validates mock settings for testing
func (v *CompatibilityValidator) validateMockSettings(settings *mockOutputSettings) error {
	composite := errors.NewCompositeError()

	// Check format-specific requirements
	switch settings.OutputFormat {
	case formatMermaid:
		if settings.FromToColumns == nil && settings.MermaidSettings == nil {
			composite.Add(v.createRequirementError(
				"mermaid format requires either FromToColumns or MermaidSettings",
				"Configure FromToColumns for flowcharts or MermaidSettings for other chart types",
			))
		}
	case formatDot:
		if settings.FromToColumns == nil {
			composite.Add(v.createRequirementError(
				"dot format requires FromToColumns configuration",
				"Use AddFromToColumns() to specify source and target columns",
			))
		}
	}

	// Check for conflicting output destinations
	if settings.OutputFile != "" && settings.S3Bucket.Bucket != "" {
		composite.Add(v.createCompatibilityError(
			"Cannot specify both OutputFile and S3Bucket",
			"Choose either file output or S3 output, not both",
		))
	}

	return composite.ErrorOrNil()
}

// validateSettings validates real OutputSettings
func (v *CompatibilityValidator) validateSettings(settings OutputSettings) error {
	composite := errors.NewCompositeError()

	// Check format-specific requirements
	switch settings.GetOutputFormat() {
	case formatMermaid:
		if settings.GetFromToColumns() == nil && settings.GetMermaidSettings() == nil {
			composite.Add(v.createRequirementError(
				"mermaid format requires either FromToColumns or MermaidSettings",
				"Configure FromToColumns for flowcharts or MermaidSettings for other chart types",
			))
		}
	case formatDot:
		if settings.GetFromToColumns() == nil {
			composite.Add(v.createRequirementError(
				"dot format requires FromToColumns configuration",
				"Use AddFromToColumns() to specify source and target columns",
			))
		}
	}

	// Check for conflicting output destinations
	if settings.GetOutputFile() != "" && settings.GetS3Bucket().GetBucket() != "" {
		composite.Add(v.createCompatibilityError(
			"Cannot specify both OutputFile and S3Bucket",
			"Choose either file output or S3 output, not both",
		))
	}

	return composite.ErrorOrNil()
}

// createRequirementError creates a validation error for missing requirements
func (v *CompatibilityValidator) createRequirementError(message, suggestion string) errors.ValidationError {
	return errors.NewValidationError(
		errors.ErrMissingRequired,
		message,
	).WithSuggestions(suggestion).(errors.ValidationError)
}

// createCompatibilityError creates a validation error for incompatible settings
func (v *CompatibilityValidator) createCompatibilityError(message, suggestion string) errors.ValidationError {
	return errors.NewValidationError(
		errors.ErrIncompatibleConfig,
		message,
	).WithSuggestions(suggestion).(errors.ValidationError)
}

// Name returns the validator name
func (v *CompatibilityValidator) Name() string {
	return "CompatibilityValidator"
}

// MermaidValidator validates Mermaid-specific configuration
type MermaidValidator struct{}

// NewMermaidValidator creates a new MermaidValidator
func NewMermaidValidator() *MermaidValidator {
	return &MermaidValidator{}
}

// Validate checks Mermaid-specific configuration
func (v *MermaidValidator) Validate(subject interface{}) error {
	// Handle the mock structure for testing
	if mockSettings, ok := subject.(*mockOutputSettings); ok {
		return v.validateMockSettings(mockSettings)
	}

	// Handle the real OutputSettings structure
	settings, ok := subject.(OutputSettings)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"MermaidValidator requires an OutputSettings",
		)
	}

	return v.validateSettings(settings)
}

// validateMockSettings validates mock settings for testing
func (v *MermaidValidator) validateMockSettings(settings *mockOutputSettings) error {
	if settings.OutputFormat != "mermaid" {
		return nil // Only validate mermaid-specific settings
	}

	if settings.MermaidSettings != nil {
		return v.validateChartType(settings.MermaidSettings.ChartType)
	}

	return nil
}

// validateSettings validates real OutputSettings
func (v *MermaidValidator) validateSettings(settings OutputSettings) error {
	if settings.GetOutputFormat() != "mermaid" {
		return nil // Only validate mermaid-specific settings
	}

	mermaidSettings := settings.GetMermaidSettings()
	if mermaidSettings != nil {
		return v.validateChartType((*mermaidSettings).GetChartType())
	}

	return nil
}

// validateChartType validates the Mermaid chart type
func (v *MermaidValidator) validateChartType(chartType string) error {
	validChartTypes := map[string]bool{
		"":           true, // Default to flowchart
		"flowchart":  true,
		"piechart":   true,
		"ganttchart": true,
	}

	if !validChartTypes[chartType] {
		return errors.NewValidationErrorWithViolations(
			errors.ErrInvalidFormat,
			fmt.Sprintf("Invalid Mermaid chart type: %s", chartType),
			errors.Violation{
				Field:      "MermaidSettings.ChartType",
				Value:      chartType,
				Constraint: "valid_chart_type",
				Message:    "Chart type must be one of: flowchart, piechart, ganttchart",
			},
		)
	}

	return nil
}

// Name returns the validator name
func (v *MermaidValidator) Name() string {
	return "MermaidValidator"
}

// DotValidator validates DOT/Graphviz-specific configuration
type DotValidator struct{}

// NewDotValidator creates a new DotValidator
func NewDotValidator() *DotValidator {
	return &DotValidator{}
}

// Validate checks DOT-specific configuration
func (v *DotValidator) Validate(subject interface{}) error {
	// Handle the mock structure for testing
	if mockSettings, ok := subject.(*mockOutputSettings); ok {
		return v.validateMockSettings(mockSettings)
	}

	// Handle the real OutputSettings structure
	settings, ok := subject.(OutputSettings)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"DotValidator requires an OutputSettings",
		)
	}

	return v.validateSettings(settings)
}

// validateMockSettings validates mock settings for testing
func (v *DotValidator) validateMockSettings(settings *mockOutputSettings) error {
	if settings.OutputFormat != "dot" {
		return nil // Only validate dot-specific settings
	}

	if settings.FromToColumns == nil {
		return v.createDotRequirementError()
	}

	return nil
}

// validateSettings validates real OutputSettings
func (v *DotValidator) validateSettings(settings OutputSettings) error {
	if settings.GetOutputFormat() != "dot" {
		return nil // Only validate dot-specific settings
	}

	if settings.GetFromToColumns() == nil {
		return v.createDotRequirementError()
	}

	return nil
}

// createDotRequirementError creates a validation error for DOT requirements
func (v *DotValidator) createDotRequirementError() errors.ValidationError {
	return errors.NewValidationError(
		errors.ErrMissingRequired,
		"DOT format requires FromToColumns configuration",
	).WithSuggestions(
		"Use AddFromToColumns() to specify source and target columns for graph visualization",
	).(errors.ValidationError)
}

// Name returns the validator name
func (v *DotValidator) Name() string {
	return "DotValidator"
}
