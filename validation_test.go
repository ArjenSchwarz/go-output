package format

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Test data structures
type testData struct {
	Name  string
	Value int
	Valid bool
}

// Test validators
func TestValidatorFunc(t *testing.T) {
	// Create a simple validator function
	validator := ValidatorFunc(func(subject any) error {
		data, ok := subject.(*testData)
		if !ok {
			return errors.New("invalid subject type")
		}
		if data.Value < 0 {
			return errors.New("value cannot be negative")
		}
		return nil
	})

	tests := []struct {
		name      string
		subject   any
		wantError bool
	}{
		{
			name:      "valid data",
			subject:   &testData{Name: "test", Value: 10, Valid: true},
			wantError: false,
		},
		{
			name:      "negative value",
			subject:   &testData{Name: "test", Value: -5, Valid: true},
			wantError: true,
		},
		{
			name:      "invalid type",
			subject:   "not a testData",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.subject)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatorFunc.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}

	// Test Name method
	if validator.Name() != "function validator" {
		t.Errorf("ValidatorFunc.Name() = %v, want %v", validator.Name(), "function validator")
	}
}

func TestNamedValidatorFunc(t *testing.T) {
	validator := NamedValidatorFunc("positive value validator", func(subject any) error {
		data, ok := subject.(*testData)
		if !ok {
			return errors.New("invalid subject type")
		}
		if data.Value <= 0 {
			return errors.New("value must be positive")
		}
		return nil
	})

	// Test validation
	err := validator.Validate(&testData{Name: "test", Value: -1})
	if err == nil {
		t.Error("NamedValidatorFunc.Validate() expected error for negative value")
	}

	err = validator.Validate(&testData{Name: "test", Value: 5})
	if err != nil {
		t.Errorf("NamedValidatorFunc.Validate() unexpected error: %v", err)
	}

	// Test name
	if validator.Name() != "positive value validator" {
		t.Errorf("NamedValidatorFunc.Name() = %v, want %v", validator.Name(), "positive value validator")
	}
}

func TestCompositeError(t *testing.T) {
	t.Run("empty composite error", func(t *testing.T) {
		composite := NewCompositeError()

		if composite.HasErrors() {
			t.Error("NewCompositeError() should not have errors initially")
		}

		if composite.Count() != 0 {
			t.Errorf("NewCompositeError().Count() = %v, want 0", composite.Count())
		}

		if composite.ErrorOrNil() != nil {
			t.Error("NewCompositeError().ErrorOrNil() should return nil when no errors")
		}

		if composite.Error() != "no errors" {
			t.Errorf("NewCompositeError().Error() = %v, want 'no errors'", composite.Error())
		}
	})

	t.Run("single error", func(t *testing.T) {
		composite := NewCompositeError()
		err := errors.New("test error")
		composite.Add(err)

		if !composite.HasErrors() {
			t.Error("CompositeError should have errors after adding one")
		}

		if composite.Count() != 1 {
			t.Errorf("CompositeError.Count() = %v, want 1", composite.Count())
		}

		if composite.ErrorOrNil() == nil {
			t.Error("CompositeError.ErrorOrNil() should not return nil when errors exist")
		}

		if composite.Error() != "test error" {
			t.Errorf("CompositeError.Error() = %v, want 'test error'", composite.Error())
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		composite := NewCompositeError()
		err1 := errors.New("first error")
		err2 := errors.New("second error")
		err3 := errors.New("third error")

		composite.Add(err1)
		composite.Add(err2)
		composite.Add(err3)

		if composite.Count() != 3 {
			t.Errorf("CompositeError.Count() = %v, want 3", composite.Count())
		}

		errorMsg := composite.Error()
		if !strings.Contains(errorMsg, "multiple validation errors (3)") {
			t.Errorf("CompositeError.Error() should contain count, got: %v", errorMsg)
		}

		if !strings.Contains(errorMsg, "first error") {
			t.Errorf("CompositeError.Error() should contain first error, got: %v", errorMsg)
		}

		if !strings.Contains(errorMsg, "second error") {
			t.Errorf("CompositeError.Error() should contain second error, got: %v", errorMsg)
		}

		if !strings.Contains(errorMsg, "third error") {
			t.Errorf("CompositeError.Error() should contain third error, got: %v", errorMsg)
		}
	})

	t.Run("add nil error", func(t *testing.T) {
		composite := NewCompositeError()
		composite.Add(nil)

		if composite.HasErrors() {
			t.Error("CompositeError should not have errors after adding nil")
		}
	})

	t.Run("add all errors", func(t *testing.T) {
		composite := NewCompositeError()
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")

		composite.AddAll(err1, nil, err2)

		if composite.Count() != 2 {
			t.Errorf("CompositeError.Count() = %v, want 2", composite.Count())
		}
	})
}

func TestValidationRunner(t *testing.T) {
	// Create test validators
	positiveValidator := NamedValidatorFunc("positive", func(subject any) error {
		data := subject.(*testData)
		if data.Value <= 0 {
			return errors.New("value must be positive")
		}
		return nil
	})

	nameValidator := NamedValidatorFunc("name required", func(subject any) error {
		data := subject.(*testData)
		if data.Name == "" {
			return errors.New("name is required")
		}
		return nil
	})

	validFlagValidator := NamedValidatorFunc("valid flag", func(subject any) error {
		data := subject.(*testData)
		if !data.Valid {
			return errors.New("valid flag must be true")
		}
		return nil
	})

	t.Run("fail fast mode - no errors", func(t *testing.T) {
		runner := NewValidationRunner(ValidationModeFailFast)
		runner.AddValidator(positiveValidator)
		runner.AddValidator(nameValidator)

		data := &testData{Name: "test", Value: 10, Valid: true}
		err := runner.Validate(data)

		if err != nil {
			t.Errorf("ValidationRunner.Validate() unexpected error: %v", err)
		}
	})

	t.Run("fail fast mode - first error", func(t *testing.T) {
		runner := NewValidationRunner(ValidationModeFailFast)
		runner.AddValidator(positiveValidator)
		runner.AddValidator(nameValidator)

		data := &testData{Name: "", Value: -5, Valid: false}
		err := runner.Validate(data)

		if err == nil {
			t.Error("ValidationRunner.Validate() expected error")
		}

		// Should get the first error (positive validator)
		if !strings.Contains(err.Error(), "positive") {
			t.Errorf("ValidationRunner.Validate() expected positive error, got: %v", err)
		}
	})

	t.Run("collect all mode - multiple errors", func(t *testing.T) {
		runner := NewValidationRunner(ValidationModeCollectAll)
		runner.AddValidators(positiveValidator, nameValidator, validFlagValidator)

		data := &testData{Name: "", Value: -5, Valid: false}
		err := runner.Validate(data)

		if err == nil {
			t.Error("ValidationRunner.Validate() expected error")
		}

		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "multiple validation errors (3)") {
			t.Errorf("ValidationRunner.Validate() expected multiple errors, got: %v", errorMsg)
		}
	})

	t.Run("collect all mode - no errors", func(t *testing.T) {
		runner := NewValidationRunner(ValidationModeCollectAll)
		runner.AddValidators(positiveValidator, nameValidator)

		data := &testData{Name: "test", Value: 10, Valid: true}
		err := runner.Validate(data)

		if err != nil {
			t.Errorf("ValidationRunner.Validate() unexpected error: %v", err)
		}
	})

	t.Run("no validators", func(t *testing.T) {
		runner := NewValidationRunner(ValidationModeFailFast)
		data := &testData{Name: "test", Value: 10, Valid: true}
		err := runner.Validate(data)

		if err != nil {
			t.Errorf("ValidationRunner.Validate() with no validators should not error: %v", err)
		}
	})
}

func TestValidatorChain(t *testing.T) {
	// Create test validators
	typeValidator := NamedValidatorFunc("type check", func(subject any) error {
		if _, ok := subject.(*testData); !ok {
			return errors.New("invalid type")
		}
		return nil
	})

	positiveValidator := NamedValidatorFunc("positive", func(subject any) error {
		data := subject.(*testData)
		if data.Value <= 0 {
			return errors.New("value must be positive")
		}
		return nil
	})

	t.Run("successful chain", func(t *testing.T) {
		chain := NewValidatorChain("test chain")
		chain.Add(typeValidator).Add(positiveValidator)

		data := &testData{Name: "test", Value: 10, Valid: true}
		err := chain.Validate(data)

		if err != nil {
			t.Errorf("ValidatorChain.Validate() unexpected error: %v", err)
		}

		if chain.Name() != "test chain" {
			t.Errorf("ValidatorChain.Name() = %v, want 'test chain'", chain.Name())
		}
	})

	t.Run("chain fails on first validator", func(t *testing.T) {
		chain := NewValidatorChain("")
		chain.Add(typeValidator).Add(positiveValidator)

		err := chain.Validate("not a testData")

		if err == nil {
			t.Error("ValidatorChain.Validate() expected error for invalid type")
		}

		if !strings.Contains(err.Error(), "invalid type") {
			t.Errorf("ValidatorChain.Validate() expected type error, got: %v", err)
		}

		if chain.Name() != "validator chain" {
			t.Errorf("ValidatorChain.Name() = %v, want 'validator chain'", chain.Name())
		}
	})

	t.Run("chain fails on second validator", func(t *testing.T) {
		chain := NewValidatorChain("test chain")
		chain.Add(typeValidator).Add(positiveValidator)

		data := &testData{Name: "test", Value: -5, Valid: true}
		err := chain.Validate(data)

		if err == nil {
			t.Error("ValidatorChain.Validate() expected error for negative value")
		}

		if !strings.Contains(err.Error(), "positive") {
			t.Errorf("ValidatorChain.Validate() expected positive error, got: %v", err)
		}
	})
}

func TestConditionalValidator(t *testing.T) {
	// Create a validator that only runs when Valid flag is true
	positiveValidator := NamedValidatorFunc("positive", func(subject any) error {
		data := subject.(*testData)
		if data.Value <= 0 {
			return errors.New("value must be positive")
		}
		return nil
	})

	condition := func(subject any) bool {
		data := subject.(*testData)
		return data.Valid
	}

	conditional := NewConditionalValidator("conditional positive", condition, positiveValidator)

	t.Run("condition true - validation runs", func(t *testing.T) {
		data := &testData{Name: "test", Value: -5, Valid: true}
		err := conditional.Validate(data)

		if err == nil {
			t.Error("ConditionalValidator.Validate() expected error when condition is true")
		}
	})

	t.Run("condition false - validation skipped", func(t *testing.T) {
		data := &testData{Name: "test", Value: -5, Valid: false}
		err := conditional.Validate(data)

		if err != nil {
			t.Errorf("ConditionalValidator.Validate() unexpected error when condition is false: %v", err)
		}
	})

	t.Run("name", func(t *testing.T) {
		if conditional.Name() != "conditional positive" {
			t.Errorf("ConditionalValidator.Name() = %v, want 'conditional positive'", conditional.Name())
		}

		// Test default name
		unnamed := NewConditionalValidator("", condition, positiveValidator)
		expectedName := fmt.Sprintf("conditional(%s)", positiveValidator.Name())
		if unnamed.Name() != expectedName {
			t.Errorf("ConditionalValidator.Name() = %v, want %v", unnamed.Name(), expectedName)
		}
	})
}

func TestValidationContext(t *testing.T) {
	validator := NamedValidatorFunc("test", func(subject any) error {
		return nil
	})

	contextual := AsContextual(validator)

	t.Run("contextual validator adapter", func(t *testing.T) {
		ctx := ValidationContext{
			Subject:   &testData{Name: "test", Value: 10},
			Path:      "test.field",
			Metadata:  map[string]any{"key": "value"},
			Validator: validator,
		}

		err := contextual.ValidateWithContext(ctx)
		if err != nil {
			t.Errorf("ContextualValidator.ValidateWithContext() unexpected error: %v", err)
		}

		err = contextual.Validate(&testData{Name: "test", Value: 10})
		if err != nil {
			t.Errorf("ContextualValidator.Validate() unexpected error: %v", err)
		}

		if contextual.Name() != validator.Name() {
			t.Errorf("ContextualValidator.Name() = %v, want %v", contextual.Name(), validator.Name())
		}
	})

	t.Run("already contextual validator", func(t *testing.T) {
		// Test that AsContextual returns the same instance if already contextual
		contextual2 := AsContextual(contextual)
		if contextual2 != contextual {
			t.Error("AsContextual should return same instance for already contextual validator")
		}
	})
}

// Benchmark tests
func BenchmarkValidatorFunc(b *testing.B) {
	validator := ValidatorFunc(func(subject any) error {
		data := subject.(*testData)
		if data.Value < 0 {
			return errors.New("negative value")
		}
		return nil
	})

	data := &testData{Name: "test", Value: 10, Valid: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(data)
	}
}

func BenchmarkValidationRunner(b *testing.B) {
	runner := NewValidationRunner(ValidationModeFailFast)
	runner.AddValidator(NamedValidatorFunc("positive", func(subject any) error {
		data := subject.(*testData)
		if data.Value <= 0 {
			return errors.New("value must be positive")
		}
		return nil
	}))
	runner.AddValidator(NamedValidatorFunc("name", func(subject any) error {
		data := subject.(*testData)
		if data.Name == "" {
			return errors.New("name required")
		}
		return nil
	}))

	data := &testData{Name: "test", Value: 10, Valid: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Validate(data)
	}
}
