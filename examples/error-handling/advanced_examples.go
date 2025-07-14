package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/errors"
	"github.com/ArjenSchwarz/go-output/validators"
)

func main() {
	fmt.Println("=== Advanced Error Handling Examples ===")

	// Example 1: Custom validators
	fmt.Println("\n1. Custom Validators")
	customValidators()

	// Example 2: Custom recovery strategies
	fmt.Println("\n2. Custom Recovery Strategies")
	customRecoveryStrategies()

	// Example 3: Interactive error handling
	fmt.Println("\n3. Interactive Error Handling")
	interactiveErrorHandling()

	// Example 4: Error reporting and metrics
	fmt.Println("\n4. Error Reporting and Metrics")
	errorReportingAndMetrics()

	// Example 5: Complex validation chains
	fmt.Println("\n5. Complex Validation Chains")
	complexValidationChains()
}

// customValidators demonstrates creating and using custom validators
func customValidators() {
	fmt.Println("Creating custom validators for business rules:")

	settings := format.NewOutputSettings()
	settings.SetOutputFormat("table")
	settings.Title = "Employee Records"

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"ID", "Name", "Email", "Department", "Salary", "StartDate"},
	}

	// Add custom validators
	output.AddValidator(NewEmailValidator())
	output.AddValidator(NewSalaryRangeValidator(30000, 200000))
	output.AddValidator(NewDateValidator("StartDate"))
	output.AddValidator(NewDepartmentValidator([]string{"Engineering", "Sales", "Marketing", "HR"}))

	// Add test data with various validation issues
	testData := []map[string]interface{}{
		{
			"ID":         "EMP001",
			"Name":       "John Doe",
			"Email":      "john.doe@company.com",
			"Department": "Engineering",
			"Salary":     85000,
			"StartDate":  "2023-01-15",
		},
		{
			"ID":         "EMP002",
			"Name":       "Jane Smith",
			"Email":      "invalid-email",  // Invalid email
			"Department": "Sales",
			"Salary":     250000,  // Salary too high
			"StartDate":  "2023-02-30",  // Invalid date
		},
		{
			"ID":         "EMP003",
			"Name":       "Bob Johnson",
			"Email":      "bob@company.com",
			"Department": "InvalidDept",  // Invalid department
			"Salary":     15000,  // Salary too low
			"StartDate":  "2023-03-10",
		},
	}

	for _, data := range testData {
		output.AddContents(data)
	}

	// Set lenient mode to collect all validation errors
	output.SetErrorMode(errors.ErrorModeLenient)

	// Validate and process
	if err := output.WriteWithValidation(); err != nil {
		fmt.Printf("Validation completed with errors: %v\n", err)
	}

	// Show validation summary
	summary := output.GetErrorSummary()
	fmt.Printf("Validation Summary:\n")
	fmt.Printf("  Total errors: %d\n", summary.TotalErrors)
	fmt.Printf("  Fixable errors: %d\n", summary.FixableErrors)

	if len(summary.Suggestions) > 0 {
		fmt.Println("  Suggestions:")
		for _, suggestion := range summary.Suggestions {
			fmt.Printf("    - %s\n", suggestion)
		}
	}
}

// customRecoveryStrategies demonstrates custom recovery strategies
func customRecoveryStrategies() {
	fmt.Println("Using custom recovery strategies:")

	// Create a custom recovery strategy for data cleanup
	dataCleanupStrategy := &DataCleanupStrategy{
		cleanupRules: map[string]func(interface{}) interface{}{
			"Email": func(value interface{}) interface{} {
				if str, ok := value.(string); ok {
					// Simple email cleanup
					if !strings.Contains(str, "@") {
						return str + "@company.com"
					}
				}
				return value
			},
			"Department": func(value interface{}) interface{} {
				if str, ok := value.(string); ok {
					// Normalize department names
					normalized := strings.ToLower(str)
					switch normalized {
					case "eng", "engineering", "tech":
						return "Engineering"
					case "sales", "selling":
						return "Sales"
					case "marketing", "mktg":
						return "Marketing"
					case "hr", "human resources":
						return "HR"
					}
				}
				return value
			},
		},
	}

	// Set up recovery handler
	recovery := errors.NewDefaultRecoveryHandler()
	recovery.AddStrategy(dataCleanupStrategy)

	// Create output with recovery
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("json")

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Email", "Department"},
	}

	output.SetErrorMode(errors.ErrorModeLenient)
	output.WithRecoveryHandler(recovery)

	// Add data that needs cleanup
	output.AddContents(map[string]interface{}{
		"Name":       "Alice Johnson",
		"Email":      "alice.johnson",  // Missing domain
		"Department": "eng",           // Needs normalization
	})

	if err := output.WriteWithValidation(); err != nil {
		fmt.Printf("Processing with recovery: %v\n", err)
	}

	summary := output.GetErrorSummary()
	fmt.Printf("Recovery Summary: %d errors processed\n", summary.TotalErrors)
}

// interactiveErrorHandling demonstrates interactive error resolution
func interactiveErrorHandling() {
	fmt.Println("Interactive error handling (simulated):")

	// Create a mock interactive handler for demonstration
	interactiveHandler := &MockInteractiveHandler{
		responses: map[string]string{
			"format_error": "fix",
			"missing_data": "provide",
			"invalid_data": "ignore",
		},
	}

	// Simulate interactive error handling
	testError := errors.NewValidationError(
		errors.ErrInvalidFormat,
		"Invalid output format 'xml'",
	).WithSuggestions(
		"Use 'json', 'yaml', or 'csv' instead",
		"Check documentation for supported formats",
	)

	fmt.Printf("Error: %v\n", testError)
	fmt.Printf("Suggestions: %v\n", testError.Suggestions())

	// Simulate user interaction
	response := interactiveHandler.HandleError(testError)
	fmt.Printf("User chose: %s\n", response)

	switch response {
	case "fix":
		fmt.Println("  -> Applying automatic fix...")
		fmt.Println("  -> Format changed to 'json'")
	case "ignore":
		fmt.Println("  -> Ignoring error and continuing...")
	case "retry":
		fmt.Println("  -> Retrying operation...")
	}
}

// errorReportingAndMetrics demonstrates error reporting and metrics
func errorReportingAndMetrics() {
	fmt.Println("Error reporting and metrics collection:")

	// Create error reporter
	reporter := &ErrorReporter{
		errors: make([]errors.OutputError, 0),
	}

	// Simulate collecting various errors
	testErrors := []errors.OutputError{
		errors.NewValidationError(errors.ErrInvalidFormat, "Invalid format"),
		errors.NewValidationError(errors.ErrMissingRequired, "Missing field"),
		errors.NewProcessingError(errors.ErrFileWrite, "Write failed"),
		errors.NewValidationError(errors.ErrInvalidFormat, "Another format error"),
		errors.NewError(errors.ErrInvalidDataType, "Type mismatch"),
	}

	for _, err := range testErrors {
		reporter.Report(err)
	}

	// Generate report
	report := reporter.GenerateReport()
	fmt.Printf("Error Report:\n%s\n", report)

	// Show metrics
	metrics := reporter.GetMetrics()
	fmt.Printf("Metrics:\n")
	fmt.Printf("  Total errors: %d\n", metrics.TotalErrors)
	fmt.Printf("  Most common: %s (%d occurrences)\n", metrics.MostCommonError, metrics.MostCommonCount)
	fmt.Printf("  Error rate: %.2f errors/minute\n", metrics.ErrorRate)
}

// complexValidationChains demonstrates complex validation chains
func complexValidationChains() {
	fmt.Println("Complex validation chains:")

	// Create a validator chain
	chain := &ValidationChain{
		validators: []validators.Validator{
			NewEmailValidator(),
			NewSalaryRangeValidator(30000, 200000),
			NewBusinessRuleValidator(),
		},
		stopOnFirst: false, // Collect all errors
	}

	// Test data
	testData := map[string]interface{}{
		"Name":       "John Doe",
		"Email":      "john@company.com",
		"Department": "Engineering",
		"Salary":     85000,
		"Level":      "Senior",
		"StartDate":  "2023-01-15",
	}

	// Run validation chain
	fmt.Printf("Running validation chain on: %v\n", testData)
	if err := chain.Validate(testData); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Println("All validations passed!")
	}

	// Test with invalid data
	invalidData := map[string]interface{}{
		"Name":       "",  // Empty name
		"Email":      "invalid-email",  // Invalid email
		"Department": "Engineering",
		"Salary":     15000,  // Too low
		"Level":      "Senior",
		"StartDate":  "2023-01-15",
	}

	fmt.Printf("\nRunning validation chain on invalid data: %v\n", invalidData)
	if err := chain.Validate(invalidData); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		
		// Show detailed error information
		if validationErr, ok := err.(errors.ValidationError); ok {
			fmt.Printf("Validation violations: %d\n", len(validationErr.Violations()))
			for _, violation := range validationErr.Violations() {
				fmt.Printf("  - %s: %s (value: %v)\n", violation.Field, violation.Message, violation.Value)
			}
		}
	}
}

// Custom validator implementations

// EmailValidator validates email format
type EmailValidator struct{}

func NewEmailValidator() *EmailValidator {
	return &EmailValidator{}
}

func (v *EmailValidator) Validate(subject interface{}) error {
	data, ok := subject.(map[string]interface{})
	if !ok {
		return errors.NewValidationError(errors.ErrInvalidDataType, "expected map[string]interface{}")
	}

	email, exists := data["Email"]
	if !exists {
		return nil // Email is optional in this validator
	}

	emailStr, ok := email.(string)
	if !ok {
		return errors.NewValidationError(errors.ErrInvalidDataType, "email must be a string")
	}

	// Simple email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(emailStr) {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"invalid email format",
		).WithContext(errors.ErrorContext{
			Field: "Email",
			Value: emailStr,
		}).WithSuggestions(
			"Use format: name@domain.com",
			"Check for typos in email address",
		)
	}

	return nil
}

func (v *EmailValidator) Name() string {
	return "EmailValidator"
}

// SalaryRangeValidator validates salary range
type SalaryRangeValidator struct {
	min, max int
}

func NewSalaryRangeValidator(min, max int) *SalaryRangeValidator {
	return &SalaryRangeValidator{min: min, max: max}
}

func (v *SalaryRangeValidator) Validate(subject interface{}) error {
	data, ok := subject.(map[string]interface{})
	if !ok {
		return errors.NewValidationError(errors.ErrInvalidDataType, "expected map[string]interface{}")
	}

	salary, exists := data["Salary"]
	if !exists {
		return nil // Salary is optional
	}

	var salaryInt int
	switch s := salary.(type) {
	case int:
		salaryInt = s
	case float64:
		salaryInt = int(s)
	default:
		return errors.NewValidationError(errors.ErrInvalidDataType, "salary must be a number")
	}

	if salaryInt < v.min || salaryInt > v.max {
		return errors.NewValidationError(
			errors.ErrConstraintViolation,
			fmt.Sprintf("salary must be between %d and %d", v.min, v.max),
		).WithContext(errors.ErrorContext{
			Field: "Salary",
			Value: salaryInt,
		}).WithSuggestions(
			fmt.Sprintf("Adjust salary to be between %d and %d", v.min, v.max),
			"Check salary grade guidelines",
		)
	}

	return nil
}

func (v *SalaryRangeValidator) Name() string {
	return "SalaryRangeValidator"
}

// DateValidator validates date format
type DateValidator struct {
	field string
}

func NewDateValidator(field string) *DateValidator {
	return &DateValidator{field: field}
}

func (v *DateValidator) Validate(subject interface{}) error {
	data, ok := subject.(map[string]interface{})
	if !ok {
		return errors.NewValidationError(errors.ErrInvalidDataType, "expected map[string]interface{}")
	}

	dateValue, exists := data[v.field]
	if !exists {
		return nil // Date is optional
	}

	dateStr, ok := dateValue.(string)
	if !ok {
		return errors.NewValidationError(errors.ErrInvalidDataType, "date must be a string")
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			fmt.Sprintf("invalid date format in field %s", v.field),
		).WithContext(errors.ErrorContext{
			Field: v.field,
			Value: dateStr,
		}).WithSuggestions(
			"Use format: YYYY-MM-DD (e.g., 2023-01-15)",
			"Check for valid day/month combinations",
		)
	}

	return nil
}

func (v *DateValidator) Name() string {
	return "DateValidator"
}

// DepartmentValidator validates department names
type DepartmentValidator struct {
	validDepartments []string
}

func NewDepartmentValidator(validDepartments []string) *DepartmentValidator {
	return &DepartmentValidator{validDepartments: validDepartments}
}

func (v *DepartmentValidator) Validate(subject interface{}) error {
	data, ok := subject.(map[string]interface{})
	if !ok {
		return errors.NewValidationError(errors.ErrInvalidDataType, "expected map[string]interface{}")
	}

	dept, exists := data["Department"]
	if !exists {
		return nil // Department is optional
	}

	deptStr, ok := dept.(string)
	if !ok {
		return errors.NewValidationError(errors.ErrInvalidDataType, "department must be a string")
	}

	// Check if department is valid
	for _, valid := range v.validDepartments {
		if deptStr == valid {
			return nil
		}
	}

	return errors.NewValidationError(
		errors.ErrConstraintViolation,
		fmt.Sprintf("invalid department: %s", deptStr),
	).WithContext(errors.ErrorContext{
		Field: "Department",
		Value: deptStr,
	}).WithSuggestions(
		fmt.Sprintf("Valid departments: %s", strings.Join(v.validDepartments, ", ")),
		"Check department naming conventions",
	)
}

func (v *DepartmentValidator) Name() string {
	return "DepartmentValidator"
}

// BusinessRuleValidator validates complex business rules
type BusinessRuleValidator struct{}

func NewBusinessRuleValidator() *BusinessRuleValidator {
	return &BusinessRuleValidator{}
}

func (v *BusinessRuleValidator) Validate(subject interface{}) error {
	data, ok := subject.(map[string]interface{})
	if !ok {
		return errors.NewValidationError(errors.ErrInvalidDataType, "expected map[string]interface{}")
	}

	// Business rule: Senior level employees must have salary > 70000
	level, hasLevel := data["Level"]
	salary, hasSalary := data["Salary"]

	if hasLevel && hasSalary {
		levelStr, ok1 := level.(string)
		salaryNum, ok2 := salary.(int)
		if !ok2 {
			if salaryFloat, ok3 := salary.(float64); ok3 {
				salaryNum = int(salaryFloat)
				ok2 = true
			}
		}

		if ok1 && ok2 && levelStr == "Senior" && salaryNum <= 70000 {
			return errors.NewValidationError(
				errors.ErrConstraintViolation,
				"senior level employees must have salary > 70000",
			).WithContext(errors.ErrorContext{
				Field: "Salary",
				Value: salaryNum,
				Metadata: map[string]interface{}{
					"Level": levelStr,
				},
			}).WithSuggestions(
				"Increase salary for senior level position",
				"Change level to match salary range",
			)
		}
	}

	return nil
}

func (v *BusinessRuleValidator) Name() string {
	return "BusinessRuleValidator"
}

// Custom recovery strategy
type DataCleanupStrategy struct {
	cleanupRules map[string]func(interface{}) interface{}
}

func (s *DataCleanupStrategy) Apply(err errors.OutputError, context interface{}) (interface{}, error) {
	// This is a simplified example - in reality, you'd implement specific cleanup logic
	if err.Code() == errors.ErrInvalidDataType {
		// Apply data cleanup rules
		return context, nil
	}
	return nil, fmt.Errorf("cannot recover from error: %s", err.Code())
}

func (s *DataCleanupStrategy) ApplicableFor(err errors.OutputError) bool {
	return err.Code() == errors.ErrInvalidDataType
}

func (s *DataCleanupStrategy) Name() string {
	return "DataCleanupStrategy"
}

// Mock interactive handler for demonstration
type MockInteractiveHandler struct {
	responses map[string]string
}

func (h *MockInteractiveHandler) HandleError(err errors.OutputError) string {
	// Simulate user interaction based on error type
	switch err.Code() {
	case errors.ErrInvalidFormat:
		return h.responses["format_error"]
	case errors.ErrMissingRequired:
		return h.responses["missing_data"]
	case errors.ErrInvalidDataType:
		return h.responses["invalid_data"]
	default:
		return "ignore"
	}
}

// Error reporter for metrics
type ErrorReporter struct {
	errors []errors.OutputError
}

func (r *ErrorReporter) Report(err errors.OutputError) {
	r.errors = append(r.errors, err)
}

func (r *ErrorReporter) GenerateReport() string {
	if len(r.errors) == 0 {
		return "No errors to report"
	}

	report := fmt.Sprintf("Error Report - Total: %d errors\n", len(r.errors))
	
	// Count by error code
	counts := make(map[errors.ErrorCode]int)
	for _, err := range r.errors {
		counts[err.Code()]++
	}

	report += "Error breakdown:\n"
	for code, count := range counts {
		report += fmt.Sprintf("  %s: %d occurrences\n", code, count)
	}

	return report
}

type ErrorMetrics struct {
	TotalErrors      int
	MostCommonError  errors.ErrorCode
	MostCommonCount  int
	ErrorRate        float64
}

func (r *ErrorReporter) GetMetrics() ErrorMetrics {
	metrics := ErrorMetrics{
		TotalErrors: len(r.errors),
	}

	if len(r.errors) == 0 {
		return metrics
	}

	// Find most common error
	counts := make(map[errors.ErrorCode]int)
	for _, err := range r.errors {
		counts[err.Code()]++
	}

	maxCount := 0
	for code, count := range counts {
		if count > maxCount {
			maxCount = count
			metrics.MostCommonError = code
			metrics.MostCommonCount = count
		}
	}

	// Calculate error rate (errors per minute - simplified)
	metrics.ErrorRate = float64(len(r.errors)) / 1.0 // Assume 1 minute for demo

	return metrics
}

// Validation chain
type ValidationChain struct {
	validators  []validators.Validator
	stopOnFirst bool
}

func (c *ValidationChain) Validate(subject interface{}) error {
	var collectedErrors []error

	for _, validator := range c.validators {
		if err := validator.Validate(subject); err != nil {
			if c.stopOnFirst {
				return err
			}
			collectedErrors = append(collectedErrors, err)
		}
	}

	if len(collectedErrors) == 0 {
		return nil
	}

	if len(collectedErrors) == 1 {
		return collectedErrors[0]
	}

	// Create composite error
	composite := errors.NewCompositeError()
	for _, err := range collectedErrors {
		if validationErr, ok := err.(errors.ValidationError); ok {
			composite.Add(validationErr)
		}
	}

	return composite
}

func (c *ValidationChain) Name() string {
	return "ValidationChain"
}