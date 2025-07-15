package errors

import (
	"encoding/json"
	"testing"
)

func TestViolation(t *testing.T) {
	violation := Violation{
		Field:      "username",
		Value:      "",
		Constraint: "required",
		Message:    "username is required",
	}

	if violation.Field != "username" {
		t.Errorf("Expected Field to be 'username', got %s", violation.Field)
	}
	if violation.Value != "" {
		t.Errorf("Expected Value to be empty string, got %v", violation.Value)
	}
	if violation.Constraint != "required" {
		t.Errorf("Expected Constraint to be 'required', got %s", violation.Constraint)
	}
	if violation.Message != "username is required" {
		t.Errorf("Expected Message to be 'username is required', got %s", violation.Message)
	}
}

func TestValidationErrorInterface(t *testing.T) {
	// Test that validationError implements ValidationError interface
	err := NewValidationError(ErrMissingColumn, "missing column")

	var validationErr ValidationError = err
	if validationErr == nil {
		t.Error("Expected validationError to implement ValidationError interface")
	}

	// Test ValidationError interface methods
	if len(validationErr.Violations()) != 0 {
		t.Errorf("Expected no violations initially, got %d", len(validationErr.Violations()))
	}

	if validationErr.IsComposite() {
		t.Error("Expected single validation error to not be composite")
	}

	// Test that it also implements OutputError
	var outputErr OutputError = err
	if outputErr == nil {
		t.Error("Expected validationError to implement OutputError interface")
	}
}

func TestValidationErrorCreation(t *testing.T) {
	err := NewValidationError(ErrMissingColumn, "required column missing")

	if err.Code() != ErrMissingColumn {
		t.Errorf("Expected Code to be %s, got %s", ErrMissingColumn, err.Code())
	}

	if err.Severity() != SeverityError {
		t.Errorf("Expected default Severity to be %d, got %d", SeverityError, err.Severity())
	}

	expectedMsg := "[OUT-2001] required column missing"
	if err.Error() != expectedMsg {
		t.Errorf("Expected Error message to be '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestValidationErrorWithViolations(t *testing.T) {
	violations := []Violation{
		{Field: "name", Value: "", Constraint: "required", Message: "name is required"},
		{Field: "age", Value: -1, Constraint: "positive", Message: "age must be positive"},
	}

	err := NewValidationError(ErrConstraintViolation, "validation failed").
		WithViolations(violations...)

	if len(err.Violations()) != 2 {
		t.Errorf("Expected 2 violations, got %d", len(err.Violations()))
	}

	errorStr := err.Error()
	if !contains(errorStr, "name is required") {
		t.Errorf("Error message should contain first violation: %s", errorStr)
	}
	if !contains(errorStr, "age must be positive") {
		t.Errorf("Error message should contain second violation: %s", errorStr)
	}
}

func TestValidationErrorBuilderPattern(t *testing.T) {
	violation := Violation{Field: "email", Value: "invalid", Constraint: "format", Message: "invalid email format"}

	err := NewValidationError(ErrInvalidDataType, "data validation failed").
		WithViolations(violation)

	// Need to type assert to ValidationError to access Violations
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatal("Expected ValidationError type")
	}

	// Continue building with other methods
	finalErr := validationErr.WithContext(ErrorContext{Operation: "user_validation"}).
		WithSuggestions("check email format").
		WithSeverity(SeverityWarning)

	if finalErr.Code() != ErrInvalidDataType {
		t.Errorf("Expected Code to be %s, got %s", ErrInvalidDataType, finalErr.Code())
	}
	if finalErr.Severity() != SeverityWarning {
		t.Errorf("Expected Severity to be %d, got %d", SeverityWarning, finalErr.Severity())
	}
	if len(validationErr.Violations()) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(validationErr.Violations()))
	}
	if finalErr.Context().Operation != "user_validation" {
		t.Errorf("Expected Operation to be 'user_validation', got %s", finalErr.Context().Operation)
	}
	if len(finalErr.Suggestions()) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(finalErr.Suggestions()))
	}
}

func TestCompositeErrorInterface(t *testing.T) {
	// Test that compositeError implements ValidationError interface
	composite := NewCompositeError()

	var validationErr ValidationError = composite
	if validationErr == nil {
		t.Error("Expected compositeError to implement ValidationError interface")
	}

	// Test ValidationError interface methods
	if len(validationErr.Violations()) != 0 {
		t.Errorf("Expected no violations initially, got %d", len(validationErr.Violations()))
	}

	if !validationErr.IsComposite() {
		t.Error("Expected composite error to be composite")
	}

	// Test that it also implements OutputError
	var outputErr OutputError = composite
	if outputErr == nil {
		t.Error("Expected compositeError to implement OutputError interface")
	}
}

func TestCompositeErrorCreation(t *testing.T) {
	composite := NewCompositeError()

	if composite.Code() != ErrCompositeValidation {
		t.Errorf("Expected Code to be %s, got %s", ErrCompositeValidation, composite.Code())
	}

	if composite.Severity() != SeverityError {
		t.Errorf("Expected default Severity to be %d, got %d", SeverityError, composite.Severity())
	}

	if !composite.IsComposite() {
		t.Error("Expected composite error to be composite")
	}

	if len(composite.Violations()) != 0 {
		t.Errorf("Expected no violations initially, got %d", len(composite.Violations()))
	}

	if composite.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestCompositeErrorAddError(t *testing.T) {
	composite := NewCompositeError()

	// Add individual errors
	err1 := NewValidationError(ErrMissingColumn, "missing name column").
		WithViolations(Violation{Field: "name", Message: "name column required"})
	err2 := NewValidationError(ErrInvalidDataType, "invalid age type").
		WithViolations(Violation{Field: "age", Message: "age must be number"})

	composite.Add(err1)
	composite.Add(err2)

	if len(composite.Violations()) != 2 {
		t.Errorf("Expected 2 violations after adding 2 errors, got %d", len(composite.Violations()))
	}

	// Check that all violations are included
	violations := composite.Violations()
	foundName, foundAge := false, false
	for _, v := range violations {
		if v.Field == "name" {
			foundName = true
		}
		if v.Field == "age" {
			foundAge = true
		}
	}
	if !foundName {
		t.Error("Expected to find name field violation")
	}
	if !foundAge {
		t.Error("Expected to find age field violation")
	}
}

func TestCompositeErrorAddViolation(t *testing.T) {
	composite := NewCompositeError()

	violation1 := Violation{Field: "username", Message: "username required"}
	violation2 := Violation{Field: "password", Message: "password too short"}

	composite.AddViolation(violation1)
	composite.AddViolation(violation2)

	if len(composite.Violations()) != 2 {
		t.Errorf("Expected 2 violations, got %d", len(composite.Violations()))
	}

	// Verify violations are stored correctly
	violations := composite.Violations()
	if violations[0].Field != "username" || violations[1].Field != "password" {
		t.Errorf("Violations not stored in correct order")
	}
}

func TestCompositeErrorHasErrors(t *testing.T) {
	composite := NewCompositeError()

	if composite.HasErrors() {
		t.Error("Expected empty composite to have no errors")
	}

	composite.AddViolation(Violation{Field: "test", Message: "test error"})

	if !composite.HasErrors() {
		t.Error("Expected composite with violations to have errors")
	}
}

func TestCompositeErrorErrorOrNil(t *testing.T) {
	composite := NewCompositeError()

	// Should return nil when no errors
	if err := composite.ErrorOrNil(); err != nil {
		t.Errorf("Expected nil when no errors, got %v", err)
	}

	// Should return self when has errors
	composite.AddViolation(Violation{Field: "test", Message: "test error"})
	if err := composite.ErrorOrNil(); err == nil {
		t.Error("Expected error when has violations")
	}
}

func TestCompositeErrorSeverityCalculation(t *testing.T) {
	composite := NewCompositeError()

	// Add errors with different severities
	err1 := NewValidationError(ErrMissingColumn, "error 1").WithSeverity(SeverityWarning).(ValidationError)
	err2 := NewValidationError(ErrInvalidDataType, "error 2").WithSeverity(SeverityFatal).(ValidationError)
	err3 := NewValidationError(ErrConstraintViolation, "error 3").WithSeverity(SeverityError).(ValidationError)

	composite.Add(err1)
	composite.Add(err2)
	composite.Add(err3)

	// Should use highest severity (Fatal)
	if composite.Severity() != SeverityFatal {
		t.Errorf("Expected severity to be %d (Fatal), got %d", SeverityFatal, composite.Severity())
	}
}

func TestCompositeErrorMessage(t *testing.T) {
	composite := NewCompositeError()

	violation1 := Violation{Field: "name", Message: "name is required"}
	violation2 := Violation{Field: "email", Message: "invalid email format"}

	composite.AddViolation(violation1)
	composite.AddViolation(violation2)

	errorStr := composite.Error()
	if !contains(errorStr, "2 validation error") {
		t.Errorf("Error message should mention number of errors: %s", errorStr)
	}
	if !contains(errorStr, "name is required") {
		t.Errorf("Error message should contain first violation: %s", errorStr)
	}
	if !contains(errorStr, "invalid email format") {
		t.Errorf("Error message should contain second violation: %s", errorStr)
	}
}

func TestValidationErrorJSONMarshaling(t *testing.T) {
	violation := Violation{Field: "test", Value: "value", Constraint: "required", Message: "test message"}
	err := NewValidationError(ErrConstraintViolation, "validation failed").
		WithViolations(violation).
		WithSeverity(SeverityWarning)

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal validation error to JSON: %v", marshalErr)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(data, &result); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	if result["code"] != string(ErrConstraintViolation) {
		t.Errorf("Expected code to be %s, got %v", ErrConstraintViolation, result["code"])
	}

	violations, ok := result["violations"].([]interface{})
	if !ok || len(violations) != 1 {
		t.Errorf("Expected violations array with 1 item, got %v", result["violations"])
	}
}

func TestValidationErrorImmutability(t *testing.T) {
	original := NewValidationError(ErrMissingColumn, "original")
	violation := Violation{Field: "test", Message: "test"}

	// Creating new instances with modifications should not affect original
	withViolations := original.WithViolations(violation)

	// Original should remain unchanged
	if len(original.Violations()) != 0 {
		t.Errorf("Original error should have no violations, got %d", len(original.Violations()))
	}

	// New instance should have the modifications
	if len(withViolations.Violations()) != 1 {
		t.Errorf("WithViolations should add violation, got %d", len(withViolations.Violations()))
	}
}

func TestNewValidationErrorWithViolations(t *testing.T) {
	violations := []Violation{
		{Field: "name", Message: "required"},
		{Field: "age", Message: "invalid"},
	}

	err := NewValidationErrorWithViolations(ErrConstraintViolation, "multiple violations", violations...)

	if len(err.Violations()) != 2 {
		t.Errorf("Expected 2 violations, got %d", len(err.Violations()))
	}

	if err.Code() != ErrConstraintViolation {
		t.Errorf("Expected Code to be %s, got %s", ErrConstraintViolation, err.Code())
	}
}
