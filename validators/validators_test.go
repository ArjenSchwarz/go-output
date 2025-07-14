package validators

import (
	"testing"

	"github.com/ArjenSchwarz/go-output/errors"
)

func TestValidatorInterface(t *testing.T) {
	// Test ValidatorFunc implements Validator interface
	validatorFunc := ValidatorFunc(func(subject interface{}) error {
		return nil
	})

	var validator Validator = validatorFunc
	if validator == nil {
		t.Error("Expected ValidatorFunc to implement Validator interface")
	}

	if validator.Name() == "" {
		t.Error("Expected validator to have a name")
	}

	err := validator.Validate("test")
	if err != nil {
		t.Errorf("Expected no error from test validator, got %v", err)
	}
}

func TestValidatorFuncCreation(t *testing.T) {
	called := false
	validatorFunc := ValidatorFunc(func(subject interface{}) error {
		called = true
		if subject != "test" {
			return errors.NewValidationError(errors.ErrInvalidDataType, "invalid input")
		}
		return nil
	})

	// Test successful validation
	err := validatorFunc.Validate("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected validator function to be called")
	}

	// Test failed validation
	called = false
	err = validatorFunc.Validate("invalid")
	if err == nil {
		t.Error("Expected validation error for invalid input")
	}
	if !called {
		t.Error("Expected validator function to be called")
	}
}

func TestValidatorFuncName(t *testing.T) {
	validatorFunc := ValidatorFunc(func(subject interface{}) error {
		return nil
	})

	name := validatorFunc.Name()
	if name != "ValidatorFunc" {
		t.Errorf("Expected name to be 'ValidatorFunc', got %s", name)
	}
}

func TestValidatorChainInterface(t *testing.T) {
	chain := NewValidatorChain()

	var validator Validator = chain
	if validator == nil {
		t.Error("Expected ValidatorChain to implement Validator interface")
	}

	if chain.Name() == "" {
		t.Error("Expected validator chain to have a name")
	}

	// Empty chain should pass validation
	err := chain.Validate("test")
	if err != nil {
		t.Errorf("Expected empty chain to pass validation, got %v", err)
	}
}

func TestValidatorChainCreation(t *testing.T) {
	chain := NewValidatorChain()

	if len(chain.validators) != 0 {
		t.Errorf("Expected empty chain to have 0 validators, got %d", len(chain.validators))
	}

	if chain.Name() != "ValidatorChain" {
		t.Errorf("Expected name to be 'ValidatorChain', got %s", chain.Name())
	}
}

func TestValidatorChainAdd(t *testing.T) {
	chain := NewValidatorChain()

	validator1 := ValidatorFunc(func(subject interface{}) error {
		return nil
	})
	validator2 := ValidatorFunc(func(subject interface{}) error {
		return nil
	})

	chain.Add(validator1)
	if len(chain.validators) != 1 {
		t.Errorf("Expected chain to have 1 validator after adding, got %d", len(chain.validators))
	}

	chain.Add(validator2)
	if len(chain.validators) != 2 {
		t.Errorf("Expected chain to have 2 validators after adding, got %d", len(chain.validators))
	}
}

func TestValidatorChainValidateSuccess(t *testing.T) {
	chain := NewValidatorChain()

	// Add validators that all pass
	chain.Add(ValidatorFunc(func(subject interface{}) error {
		return nil
	}))
	chain.Add(ValidatorFunc(func(subject interface{}) error {
		return nil
	}))

	err := chain.Validate("test")
	if err != nil {
		t.Errorf("Expected chain validation to pass, got %v", err)
	}
}

func TestValidatorChainValidateFailFast(t *testing.T) {
	chain := NewValidatorChain()
	chain.SetFailFast(true)

	callCount := 0
	chain.Add(ValidatorFunc(func(subject interface{}) error {
		callCount++
		return nil // First validator passes
	}))
	chain.Add(ValidatorFunc(func(subject interface{}) error {
		callCount++
		return errors.NewValidationError(errors.ErrConstraintViolation, "second validator fails")
	}))
	chain.Add(ValidatorFunc(func(subject interface{}) error {
		callCount++
		return nil // This should not be called in fail-fast mode
	}))

	err := chain.Validate("test")
	if err == nil {
		t.Error("Expected chain validation to fail")
	}

	if callCount != 2 {
		t.Errorf("Expected 2 validators to be called in fail-fast mode, got %d", callCount)
	}
}

func TestValidatorChainValidateCollectAll(t *testing.T) {
	chain := NewValidatorChain()
	chain.SetFailFast(false)

	callCount := 0
	chain.Add(ValidatorFunc(func(subject interface{}) error {
		callCount++
		return errors.NewValidationErrorWithViolations(
			errors.ErrConstraintViolation,
			"first validator fails",
			errors.Violation{Field: "field1", Message: "first validation failed"},
		)
	}))
	chain.Add(ValidatorFunc(func(subject interface{}) error {
		callCount++
		return nil // Second validator passes
	}))
	chain.Add(ValidatorFunc(func(subject interface{}) error {
		callCount++
		return errors.NewValidationErrorWithViolations(
			errors.ErrConstraintViolation,
			"third validator fails",
			errors.Violation{Field: "field3", Message: "third validation failed"},
		)
	}))

	err := chain.Validate("test")
	if err == nil {
		t.Error("Expected chain validation to fail")
	}

	if callCount != 3 {
		t.Errorf("Expected all 3 validators to be called in collect-all mode, got %d", callCount)
	}

	// Check that it's a composite error with multiple violations
	if validationErr, ok := err.(errors.ValidationError); ok {
		if !validationErr.IsComposite() {
			t.Error("Expected composite validation error when collecting all errors")
		}
		violations := validationErr.Violations()
		if len(violations) < 2 {
			t.Errorf("Expected at least 2 violations, got %d", len(violations))
		}
	} else {
		t.Error("Expected ValidationError type")
	}
}

func TestValidatorChainWithContext(t *testing.T) {
	chain := NewValidatorChain()
	chain.SetFailFast(true)

	context := ValidationContext{
		Operation: "test_validation",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	chain.Add(ValidatorFunc(func(subject interface{}) error {
		// Validator can access context through the subject if needed
		return nil
	}))

	err := chain.ValidateWithContext("test", context)
	if err != nil {
		t.Errorf("Expected validation with context to pass, got %v", err)
	}
}

func TestValidationContext(t *testing.T) {
	context := ValidationContext{
		Operation: "data_validation",
		Field:     "username",
		Metadata: map[string]interface{}{
			"required":  true,
			"minLength": 3,
		},
	}

	if context.Operation != "data_validation" {
		t.Errorf("Expected Operation to be 'data_validation', got %s", context.Operation)
	}
	if context.Field != "username" {
		t.Errorf("Expected Field to be 'username', got %s", context.Field)
	}
	if context.Metadata["required"] != true {
		t.Errorf("Expected required to be true, got %v", context.Metadata["required"])
	}
}

func TestValidatorChainFailFastSetting(t *testing.T) {
	chain := NewValidatorChain()

	// Test default fail-fast setting
	if !chain.failFast {
		t.Error("Expected chain to default to fail-fast mode")
	}

	// Test setting fail-fast to false
	chain.SetFailFast(false)
	if chain.failFast {
		t.Error("Expected chain to be in collect-all mode after SetFailFast(false)")
	}

	// Test setting fail-fast to true
	chain.SetFailFast(true)
	if !chain.failFast {
		t.Error("Expected chain to be in fail-fast mode after SetFailFast(true)")
	}
}

func TestValidatorChainEmpty(t *testing.T) {
	chain := NewValidatorChain()

	// Empty chain should always pass validation
	err := chain.Validate("anything")
	if err != nil {
		t.Errorf("Expected empty chain to pass validation, got %v", err)
	}

	err = chain.ValidateWithContext("anything", ValidationContext{})
	if err != nil {
		t.Errorf("Expected empty chain with context to pass validation, got %v", err)
	}
}

func TestNamedValidator(t *testing.T) {
	namedValidator := NewNamedValidator("custom-validator", ValidatorFunc(func(subject interface{}) error {
		return nil
	}))

	if namedValidator.Name() != "custom-validator" {
		t.Errorf("Expected name to be 'custom-validator', got %s", namedValidator.Name())
	}

	err := namedValidator.Validate("test")
	if err != nil {
		t.Errorf("Expected validation to pass, got %v", err)
	}
}

func TestValidatorChainWithNamedValidators(t *testing.T) {
	chain := NewValidatorChain()

	validator1 := NewNamedValidator("first", ValidatorFunc(func(subject interface{}) error {
		return nil
	}))
	validator2 := NewNamedValidator("second", ValidatorFunc(func(subject interface{}) error {
		return errors.NewValidationError(errors.ErrConstraintViolation, "validation failed")
	}))

	chain.Add(validator1)
	chain.Add(validator2)
	chain.SetFailFast(true)

	err := chain.Validate("test")
	if err == nil {
		t.Error("Expected validation to fail")
	}

	// The error should contain information about which validator failed
	if validationErr, ok := err.(errors.ValidationError); ok {
		errorStr := validationErr.Error()
		if !contains(errorStr, "validation failed") {
			t.Errorf("Expected error message to contain validator failure: %s", errorStr)
		}
	}
}

// Helper function for string contains check (copied from other test files)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
