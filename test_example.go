package format

import "fmt"

// ExampleErrorUsage demonstrates how to use the error types
func ExampleErrorUsage() {
	// Create a basic output error
	err1 := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format specified")
	fmt.Printf("Error: %v\n", err1)

	// Create a validation error with violations
	err2 := NewValidationErrorBuilder(ErrConstraintViolation, "validation failed").
		WithViolation("Name", "required", "Name field is required", nil).
		WithViolation("Age", "positive", "Age must be positive", -1).
		Build()
	fmt.Printf("Validation Error: %v\n", err2)

	// Create a processing error
	err3 := NewProcessingError(ErrS3Upload, "failed to upload file", true)
	fmt.Printf("Processing Error: %v, Retryable: %v\n", err3, err3.Retryable())

	// Use error builder for complex errors
	err4 := NewErrorBuilder(ErrFileWrite, "failed to write output file").
		WithSeverity(SeverityWarning).
		WithField("OutputFile").
		WithOperation("file_write").
		WithValue("/invalid/path/file.json").
		WithSuggestions(
			"Check if the directory exists",
			"Verify write permissions",
			"Use a different output path",
		).
		Build()
	fmt.Printf("Complex Error: %v\n", err4)
}
