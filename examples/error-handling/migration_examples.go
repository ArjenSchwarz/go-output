package main

import (
	"fmt"
	"log"
	"os"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/errors"
)

func main() {
	fmt.Println("=== Migration Examples: From log.Fatal to Structured Error Handling ===")

	// Example 1: Basic migration from log.Fatal
	fmt.Println("\n1. Basic Migration from log.Fatal")
	migrationBasic()

	// Example 2: Migration with error handling
	fmt.Println("\n2. Migration with Error Handling")
	migrationWithErrorHandling()

	// Example 3: Migration with validation
	fmt.Println("\n3. Migration with Validation")
	migrationWithValidation()

	// Example 4: Migration with recovery
	fmt.Println("\n4. Migration with Recovery")
	migrationWithRecovery()

	// Example 5: Using migration helper
	fmt.Println("\n5. Using Migration Helper")
	migrationHelper()
}

// migrationBasic shows the basic migration pattern
func migrationBasic() {
	fmt.Println("BEFORE (using log.Fatal):")
	fmt.Println(`
func processOutput(data []string) {
    settings := format.NewOutputSettings()
    settings.SetOutputFormat("json")
    
    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"Item"},
    }
    
    for _, item := range data {
        output.AddContents(map[string]interface{}{
            "Item": item,
        })
    }
    
    // OLD WAY - this would call log.Fatal on errors
    output.Write()
}`)

	fmt.Println("\nAFTER (using error returns):")
	fmt.Println(`
func processOutput(data []string) error {
    settings := format.NewOutputSettings()
    settings.SetOutputFormat("json")
    
    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"Item"},
    }
    
    for _, item := range data {
        output.AddContents(map[string]interface{}{
            "Item": item,
        })
    }
    
    // NEW WAY - returns error instead of calling log.Fatal
    return output.WriteWithValidation()
}`)

	// Demonstrate the new approach
	fmt.Println("\nDemonstrating the new approach:")
	data := []string{"item1", "item2", "item3"}
	if err := processOutputNew(data); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Success!")
	}
}

// migrationWithErrorHandling shows migration with proper error handling
func migrationWithErrorHandling() {
	fmt.Println("BEFORE (caller doesn't handle errors):")
	fmt.Println(`
func main() {
    data := getData()
    processOutput(data)  // Could call log.Fatal internally
    fmt.Println("Processing complete")
}`)

	fmt.Println("\nAFTER (caller handles errors properly):")
	fmt.Println(`
func main() {
    data := getData()
    if err := processOutput(data); err != nil {
        log.Printf("Error processing output: %v", err)
        
        // Handle based on error type
        if outputErr, ok := err.(errors.OutputError); ok {
            switch outputErr.Severity() {
            case errors.SeverityFatal:
                os.Exit(1)
            case errors.SeverityError:
                // Maybe retry or use fallback
                return
            default:
                // Continue with warnings
            }
        }
        return
    }
    fmt.Println("Processing complete")
}`)

	// Demonstrate improved error handling
	fmt.Println("\nDemonstrating improved error handling:")
	data := []string{"test1", "test2"}
	if err := processOutputNew(data); err != nil {
		handleErrorGracefully(err)
	} else {
		fmt.Println("Processing complete")
	}
}

// migrationWithValidation shows migration with validation
func migrationWithValidation() {
	fmt.Println("BEFORE (no validation):")
	fmt.Println(`
func generateReport(data []map[string]interface{}) {
    settings := format.NewOutputSettings()
    settings.SetOutputFormat("table")
    
    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"Name", "Age", "Email"},
    }
    
    for _, item := range data {
        output.AddContents(item)
    }
    
    output.Write()  // Might fail late in the process
}`)

	fmt.Println("\nAFTER (with validation):")
	fmt.Println(`
func generateReport(data []map[string]interface{}) error {
    // Validate input early
    if len(data) == 0 {
        return errors.NewValidationError(
            errors.ErrEmptyDataset,
            "cannot generate report with empty data",
        ).WithSuggestions(
            "Provide at least one data record",
            "Check data source for availability",
        )
    }
    
    settings := format.NewOutputSettings()
    settings.SetOutputFormat("table")
    
    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"Name", "Age", "Email"},
    }
    
    // Add validation before processing
    output.AddValidator(validators.NewRequiredColumnsValidator([]string{"Name", "Email"}))
    
    for _, item := range data {
        output.AddContents(item)
    }
    
    // Validate first, then write
    if err := output.Validate(); err != nil {
        return err
    }
    
    return output.WriteWithValidation()
}`)

	// Demonstrate validation
	fmt.Println("\nDemonstrating validation:")
	testData := []map[string]interface{}{
		{"Name": "John Doe", "Age": 30, "Email": "john@example.com"},
		{"Name": "Jane Smith", "Age": 25, "Email": "jane@example.com"},
	}
	
	if err := generateReportNew(testData); err != nil {
		fmt.Printf("Validation error: %v\n", err)
	} else {
		fmt.Println("Report generated successfully!")
	}
}

// migrationWithRecovery shows migration with recovery strategies
func migrationWithRecovery() {
	fmt.Println("BEFORE (no recovery):")
	fmt.Println(`
func exportData(data []map[string]interface{}, format string) {
    settings := format.NewOutputSettings()
    settings.SetOutputFormat(format)
    
    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"ID", "Name", "Status"},
    }
    
    for _, item := range data {
        output.AddContents(item)
    }
    
    output.Write()  // Fails completely if format is invalid
}`)

	fmt.Println("\nAFTER (with recovery):")
	fmt.Println(`
func exportData(data []map[string]interface{}, format string) error {
    settings := format.NewOutputSettings()
    settings.SetOutputFormat(format)
    
    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"ID", "Name", "Status"},
    }
    
    // Set lenient mode to allow recovery
    output.SetErrorMode(errors.ErrorModeLenient)
    
    // Add recovery strategies
    recovery := errors.NewDefaultRecoveryHandler()
    recovery.AddStrategy(errors.NewFormatFallbackStrategy([]string{"table", "csv", "json"}))
    output.WithRecoveryHandler(recovery)
    
    for _, item := range data {
        output.AddContents(item)
    }
    
    // Try to write with recovery
    if err := output.WriteWithValidation(); err != nil {
        // Check if any errors were recovered
        summary := output.GetErrorSummary()
        if summary.TotalErrors > 0 {
            fmt.Printf("Processed with %d recoverable errors\n", summary.TotalErrors)
        }
        return err
    }
    
    return nil
}`)

	// Demonstrate recovery
	fmt.Println("\nDemonstrating recovery:")
	testData := []map[string]interface{}{
		{"ID": "1", "Name": "Test Item", "Status": "active"},
	}
	
	if err := exportDataNew(testData, "invalid_format"); err != nil {
		fmt.Printf("Error after recovery attempt: %v\n", err)
	} else {
		fmt.Println("Export completed (possibly with recovery)!")
	}
}

// migrationHelper shows using the migration helper
func migrationHelper() {
	fmt.Println("Using Migration Helper to analyze and guide migration:")
	
	// Create migration helper
	helper := errors.NewMigrationHelper()
	
	// Add some example migration steps
	helper.AddMigrationStep(errors.MigrationStep{
		Name:        "Replace log.Fatal calls",
		Description: "Replace all log.Fatal calls with error returns",
		Check: func() (bool, string) {
			// In a real scenario, this would scan code files
			return false, "Found 3 log.Fatal calls that need replacement"
		},
		Fix: func() error {
			// In a real scenario, this would apply code transformations
			fmt.Println("  -> Applied code transformation to replace log.Fatal")
			return nil
		},
	})
	
	helper.AddMigrationStep(errors.MigrationStep{
		Name:        "Add error handling",
		Description: "Add proper error handling to all callers",
		Check: func() (bool, string) {
			return true, "All callers have proper error handling"
		},
	})
	
	helper.AddMigrationStep(errors.MigrationStep{
		Name:        "Add validation",
		Description: "Add input validation before operations",
		Check: func() (bool, string) {
			return false, "Missing validation in 2 functions"
		},
		Fix: func() error {
			fmt.Println("  -> Added validation to functions")
			return nil
		},
	})
	
	// Check migration status
	fmt.Println("\nMigration Status:")
	status := helper.CheckMigrationStatus()
	fmt.Printf("  %s\n", status.String())
	
	// Run migration
	fmt.Println("\nRunning migration:")
	result := helper.RunMigration()
	fmt.Printf("  %s\n", result.String())
	
	// Show legacy migration helper
	fmt.Println("\nLegacy Code Analysis:")
	legacyHelper := errors.NewLegacyMigrationHelper()
	
	// Analyze some example code
	exampleCode := `
func processData(data []byte) {
    if len(data) == 0 {
        log.Fatal("data cannot be empty")
    }
    if err := validateData(data); err != nil {
        log.Fatalf("validation failed: %v", err)
    }
    // process data...
}

func handleError(err error) {
    if err != nil {
        panic(err)
    }
}
`
	
	legacyHelper.AnalyzeCode(exampleCode)
	fmt.Println(legacyHelper.GetMigrationReport())
}

// Helper functions for examples

func processOutputNew(data []string) error {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("json")
	
	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Item"},
	}
	
	for _, item := range data {
		output.AddContents(map[string]interface{}{
			"Item": item,
		})
	}
	
	return output.WriteWithValidation()
}

func generateReportNew(data []map[string]interface{}) error {
	if len(data) == 0 {
		return errors.NewValidationError(
			errors.ErrEmptyDataset,
			"cannot generate report with empty data",
		).WithSuggestions(
			"Provide at least one data record",
			"Check data source for availability",
		)
	}
	
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("table")
	
	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Age", "Email"},
	}
	
	for _, item := range data {
		output.AddContents(item)
	}
	
	// Validate first, then write
	if err := output.Validate(); err != nil {
		return err
	}
	
	return output.WriteWithValidation()
}

func exportDataNew(data []map[string]interface{}, outputFormat string) error {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat(outputFormat)
	
	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"ID", "Name", "Status"},
	}
	
	// Set lenient mode to allow recovery
	output.SetErrorMode(errors.ErrorModeLenient)
	
	for _, item := range data {
		output.AddContents(item)
	}
	
	// Try to write with recovery
	if err := output.WriteWithValidation(); err != nil {
		// Check if any errors were recovered
		summary := output.GetErrorSummary()
		if summary.TotalErrors > 0 {
			fmt.Printf("Processed with %d recoverable errors\n", summary.TotalErrors)
		}
		return err
	}
	
	return nil
}

func handleErrorGracefully(err error) {
	if err == nil {
		return
	}

	log.Printf("Error occurred: %v", err)

	if outputErr, ok := err.(errors.OutputError); ok {
		switch outputErr.Severity() {
		case errors.SeverityFatal:
			log.Printf("Fatal error - exiting: %s", outputErr.Code())
			os.Exit(1)
		case errors.SeverityError:
			log.Printf("Error occurred: %s", outputErr.Code())
		case errors.SeverityWarning:
			log.Printf("Warning: %s", outputErr.Code())
		case errors.SeverityInfo:
			log.Printf("Info: %s", outputErr.Code())
		}

		if suggestions := outputErr.Suggestions(); len(suggestions) > 0 {
			log.Println("Suggestions to fix this issue:")
			for _, suggestion := range suggestions {
				log.Printf("  - %s", suggestion)
			}
		}
	}
}