package main

import (
	"fmt"
	"log"
	"os"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/errors"
)

func main() {
	fmt.Println("=== Basic Error Handling Examples ===")

	// Example 1: Basic error handling with validation
	fmt.Println("\n1. Basic Error Handling with Validation")
	basicErrorHandling()

	// Example 2: Different error modes
	fmt.Println("\n2. Different Error Modes")
	errorModes()

	// Example 3: Error collection in lenient mode
	fmt.Println("\n3. Error Collection in Lenient Mode")
	errorCollection()

	// Example 4: Error context and suggestions
	fmt.Println("\n4. Error Context and Suggestions")
	errorContext()
}

// basicErrorHandling demonstrates basic error handling patterns
func basicErrorHandling() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("json")
	settings.Title = "Employee Data"

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Department", "Salary"},
	}

	// Add some test data
	output.AddContents(map[string]interface{}{
		"Name":       "Alice Johnson",
		"Department": "Engineering",
		"Salary":     95000,
	})

	// Method 1: Using WriteWithValidation (new error handling)
	fmt.Println("Using new error handling:")
	if err := output.WriteWithValidation(); err != nil {
		fmt.Printf("Error occurred: %v\n", err)

		// Check if it's an OutputError for more details
		if outputErr, ok := err.(errors.OutputError); ok {
			fmt.Printf("Error Code: %s\n", outputErr.Code())
			fmt.Printf("Severity: %s\n", outputErr.Severity())

			// Show suggestions if available
			if suggestions := outputErr.Suggestions(); len(suggestions) > 0 {
				fmt.Println("Suggestions:")
				for _, suggestion := range suggestions {
					fmt.Printf("  - %s\n", suggestion)
				}
			}
		}
		return
	}
	fmt.Println("Output generated successfully!")

	// Method 2: Using WriteCompat (legacy compatibility)
	fmt.Println("\nUsing legacy compatibility mode:")
	output.WriteCompat() // This will use log.Fatal on errors for backward compatibility
}

// errorModes demonstrates different error handling modes
func errorModes() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("invalid_format") // This will cause an error

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Value"},
	}

	// Strict mode (default) - fails immediately
	fmt.Println("Strict mode:")
	output.SetErrorMode(errors.ErrorModeStrict)
	if err := output.WriteWithValidation(); err != nil {
		fmt.Printf("  Error in strict mode: %v\n", err)
	}

	// Lenient mode - collects errors and continues where possible
	fmt.Println("Lenient mode:")
	output.SetErrorMode(errors.ErrorModeLenient)
	if err := output.WriteWithValidation(); err != nil {
		fmt.Printf("  Error in lenient mode: %v\n", err)
	}

	// Check collected errors
	summary := output.GetErrorSummary()
	fmt.Printf("  Total errors collected: %d\n", summary.TotalErrors)
}

// errorCollection demonstrates error collection in lenient mode
func errorCollection() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("json")

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Age", "Email"},
	}

	// Set lenient mode to collect all errors
	output.SetErrorMode(errors.ErrorModeLenient)

	// Add some data with validation issues
	output.AddContents(map[string]interface{}{
		"Name":  "John Doe",
		"Age":   30,
		"Email": "john@example.com",
	})

	// Process with error collection
	if err := output.WriteWithValidation(); err != nil {
		fmt.Printf("Final error: %v\n", err)
	}

	// Get error summary
	summary := output.GetErrorSummary()
	fmt.Printf("Error Summary:\n")
	fmt.Printf("  Total errors: %d\n", summary.TotalErrors)
	fmt.Printf("  Fixable errors: %d\n", summary.FixableErrors)

	// Show errors by severity
	for severity, count := range summary.BySeverity {
		if count > 0 {
			fmt.Printf("  %s: %d\n", severity, count)
		}
	}

	// Show errors by category
	for category, count := range summary.ByCategory {
		if count > 0 {
			fmt.Printf("  %s: %d\n", category, count)
		}
	}
}

// errorContext demonstrates error context and suggestions
func errorContext() {
	// Create a validation error with context
	validationErr := errors.NewValidationError(
		errors.ErrMissingRequired,
		"Required field 'email' is missing",
	).WithContext(errors.ErrorContext{
		Operation: "user_validation",
		Field:     "email",
		Value:     nil,
		Index:     0,
		Metadata: map[string]interface{}{
			"user_id": "12345",
			"form":    "registration",
		},
	}).WithSuggestions(
		"Add an email field to the user data",
		"Check if the email field is properly named",
		"Verify the data source includes email information",
	)

	// Display the error with full context
	fmt.Println("Validation Error with Context:")
	fmt.Printf("Error: %v\n", validationErr)
	fmt.Printf("Code: %s\n", validationErr.Code())
	fmt.Printf("Severity: %s\n", validationErr.Severity())

	context := validationErr.Context()
	fmt.Printf("Context:\n")
	fmt.Printf("  Operation: %s\n", context.Operation)
	fmt.Printf("  Field: %s\n", context.Field)
	fmt.Printf("  Value: %v\n", context.Value)
	fmt.Printf("  Index: %d\n", context.Index)
	fmt.Printf("  Metadata: %v\n", context.Metadata)

	fmt.Println("Suggestions:")
	for _, suggestion := range validationErr.Suggestions() {
		fmt.Printf("  - %s\n", suggestion)
	}
}

// Example of custom error handling function
func handleErrorGracefully(err error) {
	if err == nil {
		return
	}

	// Log the error
	log.Printf("Error occurred: %v", err)

	// Check if it's a structured error
	if outputErr, ok := err.(errors.OutputError); ok {
		// Handle based on severity
		switch outputErr.Severity() {
		case errors.SeverityFatal:
			log.Printf("Fatal error - exiting: %s", outputErr.Code())
			os.Exit(1)
		case errors.SeverityError:
			log.Printf("Error occurred: %s", outputErr.Code())
			// Could retry or use fallback
		case errors.SeverityWarning:
			log.Printf("Warning: %s", outputErr.Code())
			// Continue processing
		case errors.SeverityInfo:
			log.Printf("Info: %s", outputErr.Code())
			// Just log for debugging
		}

		// Show suggestions to help user fix the issue
		if suggestions := outputErr.Suggestions(); len(suggestions) > 0 {
			log.Println("Suggestions to fix this issue:")
			for _, suggestion := range suggestions {
				log.Printf("  - %s", suggestion)
			}
		}
	}
}
