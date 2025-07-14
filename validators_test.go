package format

import (
	"fmt"
	"strings"
	"testing"
)

// Test data helpers
func createTestOutputArray(keys []string, contents []map[string]any) *OutputArray {
	output := &OutputArray{
		Settings: NewOutputSettings(),
		Keys:     keys,
		Contents: make([]OutputHolder, len(contents)),
	}

	for i, content := range contents {
		output.Contents[i] = OutputHolder{Contents: content}
	}

	return output
}

func TestRequiredColumnsValidator(t *testing.T) {
	t.Run("all required columns present", func(t *testing.T) {
		validator := NewRequiredColumnsValidator("Name", "Age", "Email")
		output := createTestOutputArray(
			[]string{"Name", "Age", "Email", "Department"},
			[]map[string]any{
				{"Name": "Alice", "Age": 30, "Email": "alice@example.com", "Department": "Engineering"},
			},
		)

		err := validator.Validate(output)
		if err != nil {
			t.Errorf("RequiredColumnsValidator.Validate() unexpected error: %v", err)
		}
	})

	t.Run("missing required columns", func(t *testing.T) {
		validator := NewRequiredColumnsValidator("Name", "Age", "Email", "Phone")
		output := createTestOutputArray(
			[]string{"Name", "Age", "Department"},
			[]map[string]any{
				{"Name": "Alice", "Age": 30, "Department": "Engineering"},
			},
		)

		err := validator.Validate(output)
		if err == nil {
			t.Error("RequiredColumnsValidator.Validate() expected error for missing columns")
		}

		validationErr, ok := err.(ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		}

		violations := validationErr.Violations()
		if len(violations) != 2 {
			t.Errorf("Expected 2 violations, got %d", len(violations))
		}

		// Check that missing columns are reported
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "Email") || !strings.Contains(errorMsg, "Phone") {
			t.Errorf("Error message should contain missing columns: %v", errorMsg)
		}
	})

	t.Run("invalid subject type", func(t *testing.T) {
		validator := NewRequiredColumnsValidator("Name")
		err := validator.Validate("not an OutputArray")

		if err == nil {
			t.Error("RequiredColumnsValidator.Validate() expected error for invalid type")
		}

		if !strings.Contains(err.Error(), "expected OutputArray") {
			t.Errorf("Error should mention expected type: %v", err)
		}
	})

	t.Run("validator name", func(t *testing.T) {
		validator := NewRequiredColumnsValidator("Name", "Age")
		expectedName := "required columns validator (Name, Age)"
		if validator.Name() != expectedName {
			t.Errorf("RequiredColumnsValidator.Name() = %v, want %v", validator.Name(), expectedName)
		}
	})
}

func TestDataTypeValidator(t *testing.T) {
	t.Run("all types match", func(t *testing.T) {
		validator := NewDataTypeValidator().
			WithStringColumn("Name").
			WithIntColumn("Age").
			WithFloatColumn("Salary").
			WithBoolColumn("Active")

		output := createTestOutputArray(
			[]string{"Name", "Age", "Salary", "Active"},
			[]map[string]any{
				{"Name": "Alice", "Age": 30, "Salary": 75000.0, "Active": true},
				{"Name": "Bob", "Age": 25, "Salary": 65000.0, "Active": false},
			},
		)

		err := validator.Validate(output)
		if err != nil {
			t.Errorf("DataTypeValidator.Validate() unexpected error: %v", err)
		}
	})

	t.Run("type mismatches", func(t *testing.T) {
		validator := NewDataTypeValidator().
			WithStringColumn("Name").
			WithIntColumn("Age").
			WithFloatColumn("Salary")

		output := createTestOutputArray(
			[]string{"Name", "Age", "Salary"},
			[]map[string]any{
				{"Name": 123, "Age": "thirty", "Salary": "75000"},
			},
		)

		err := validator.Validate(output)
		if err == nil {
			t.Error("DataTypeValidator.Validate() expected error for type mismatches")
		}

		validationErr, ok := err.(ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		}

		violations := validationErr.Violations()
		if len(violations) != 3 {
			t.Errorf("Expected 3 violations, got %d", len(violations))
		}
	})

	t.Run("nil values", func(t *testing.T) {
		validator := NewDataTypeValidator().WithStringColumn("Name")

		output := createTestOutputArray(
			[]string{"Name"},
			[]map[string]any{
				{"Name": nil},
			},
		)

		err := validator.Validate(output)
		if err == nil {
			t.Error("DataTypeValidator.Validate() expected error for nil value")
		}

		if !strings.Contains(err.Error(), "got nil") {
			t.Errorf("Error should mention nil value: %v", err)
		}
	})

	t.Run("numeric type compatibility", func(t *testing.T) {
		validator := NewDataTypeValidator().WithIntColumn("Number")

		output := createTestOutputArray(
			[]string{"Number"},
			[]map[string]any{
				{"Number": int64(42)},
				{"Number": float64(3.14)},
			},
		)

		err := validator.Validate(output)
		if err != nil {
			t.Errorf("DataTypeValidator.Validate() should allow numeric compatibility: %v", err)
		}
	})

	t.Run("validator name", func(t *testing.T) {
		validator := NewDataTypeValidator().
			WithStringColumn("Name").
			WithIntColumn("Age")

		name := validator.Name()
		if !strings.Contains(name, "data type validator") {
			t.Errorf("DataTypeValidator.Name() should contain 'data type validator': %v", name)
		}
	})

	t.Run("invalid subject type", func(t *testing.T) {
		validator := NewDataTypeValidator()
		err := validator.Validate("not an OutputArray")

		if err == nil {
			t.Error("DataTypeValidator.Validate() expected error for invalid type")
		}
	})
}

func TestConstraintValidator(t *testing.T) {
	t.Run("all constraints pass", func(t *testing.T) {
		validator := NewConstraintValidator().
			AddConstraint(PositiveNumberConstraint("Age")).
			AddConstraint(NonEmptyStringConstraint("Name"))

		output := createTestOutputArray(
			[]string{"Name", "Age"},
			[]map[string]any{
				{"Name": "Alice", "Age": 30},
				{"Name": "Bob", "Age": 25},
			},
		)

		err := validator.Validate(output)
		if err != nil {
			t.Errorf("ConstraintValidator.Validate() unexpected error: %v", err)
		}
	})

	t.Run("constraint violations", func(t *testing.T) {
		validator := NewConstraintValidator().
			AddConstraint(PositiveNumberConstraint("Age")).
			AddConstraint(NonEmptyStringConstraint("Name"))

		output := createTestOutputArray(
			[]string{"Name", "Age"},
			[]map[string]any{
				{"Name": "", "Age": -5},
				{"Name": "Bob", "Age": 0},
			},
		)

		err := validator.Validate(output)
		if err == nil {
			t.Error("ConstraintValidator.Validate() expected error for constraint violations")
		}

		validationErr, ok := err.(ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		}

		violations := validationErr.Violations()
		if len(violations) != 3 {
			t.Errorf("Expected 3 violations, got %d", len(violations))
		}
	})

	t.Run("range constraint", func(t *testing.T) {
		validator := NewConstraintValidator().
			AddConstraint(RangeConstraint("Score", 0, 100))

		output := createTestOutputArray(
			[]string{"Score"},
			[]map[string]any{
				{"Score": 85},
				{"Score": 150}, // violation
				{"Score": -10}, // violation
			},
		)

		err := validator.Validate(output)
		if err == nil {
			t.Error("ConstraintValidator.Validate() expected error for range violations")
		}

		validationErr, ok := err.(ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		}

		violations := validationErr.Violations()
		if len(violations) != 2 {
			t.Errorf("Expected 2 violations, got %d", len(violations))
		}
	})

	t.Run("custom constraint", func(t *testing.T) {
		emailConstraint := NewConstraintFunc(
			"valid_email",
			"email must contain @ symbol",
			func(row map[string]any) error {
				if email, exists := row["Email"]; exists {
					if str, ok := email.(string); ok {
						if !strings.Contains(str, "@") {
							return fmt.Errorf("invalid email format: %s", str)
						}
					}
				}
				return nil
			},
		)

		validator := NewConstraintValidator().AddConstraint(emailConstraint)

		output := createTestOutputArray(
			[]string{"Email"},
			[]map[string]any{
				{"Email": "alice@example.com"},
				{"Email": "invalid-email"},
			},
		)

		err := validator.Validate(output)
		if err == nil {
			t.Error("ConstraintValidator.Validate() expected error for custom constraint violation")
		}
	})

	t.Run("validator name", func(t *testing.T) {
		validator := NewConstraintValidator().
			AddConstraint(PositiveNumberConstraint("Age")).
			AddConstraint(NonEmptyStringConstraint("Name"))

		name := validator.Name()
		if !strings.Contains(name, "constraint validator") {
			t.Errorf("ConstraintValidator.Name() should contain 'constraint validator': %v", name)
		}
	})

	t.Run("no constraints", func(t *testing.T) {
		validator := NewConstraintValidator()
		output := createTestOutputArray([]string{"Name"}, []map[string]any{{"Name": "Alice"}})

		err := validator.Validate(output)
		if err != nil {
			t.Errorf("ConstraintValidator.Validate() with no constraints should not error: %v", err)
		}

		if !strings.Contains(validator.Name(), "no constraints") {
			t.Errorf("ConstraintValidator.Name() should indicate no constraints: %v", validator.Name())
		}
	})
}

func TestEmptyDatasetValidator(t *testing.T) {
	t.Run("non-empty dataset allowed", func(t *testing.T) {
		validator := NewEmptyDatasetValidator(false)
		output := createTestOutputArray(
			[]string{"Name"},
			[]map[string]any{{"Name": "Alice"}},
		)

		err := validator.Validate(output)
		if err != nil {
			t.Errorf("EmptyDatasetValidator.Validate() unexpected error: %v", err)
		}
	})

	t.Run("empty dataset not allowed", func(t *testing.T) {
		validator := NewEmptyDatasetValidator(false)
		output := createTestOutputArray([]string{"Name"}, []map[string]any{})

		err := validator.Validate(output)
		if err == nil {
			t.Error("EmptyDatasetValidator.Validate() expected error for empty dataset")
		}

		validationErr, ok := err.(ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		}

		violations := validationErr.Violations()
		if len(violations) != 1 {
			t.Errorf("Expected 1 violation, got %d", len(violations))
		}

		if violations[0].Field != "Contents" {
			t.Errorf("Expected violation on Contents field, got %s", violations[0].Field)
		}
	})

	t.Run("empty dataset allowed", func(t *testing.T) {
		validator := NewEmptyDatasetValidator(true)
		output := createTestOutputArray([]string{"Name"}, []map[string]any{})

		err := validator.Validate(output)
		if err != nil {
			t.Errorf("EmptyDatasetValidator.Validate() unexpected error when empty allowed: %v", err)
		}
	})

	t.Run("validator names", func(t *testing.T) {
		allowEmpty := NewEmptyDatasetValidator(true)
		requireData := NewEmptyDatasetValidator(false)

		if !strings.Contains(allowEmpty.Name(), "allows empty") {
			t.Errorf("EmptyDatasetValidator.Name() should indicate allows empty: %v", allowEmpty.Name())
		}

		if !strings.Contains(requireData.Name(), "requires data") {
			t.Errorf("EmptyDatasetValidator.Name() should indicate requires data: %v", requireData.Name())
		}
	})
}

func TestMalformedDataValidator(t *testing.T) {
	t.Run("well-formed data", func(t *testing.T) {
		validator := NewMalformedDataValidator(false)
		output := createTestOutputArray(
			[]string{"Name", "Age"},
			[]map[string]any{
				{"Name": "Alice", "Age": 30},
				{"Name": "Bob", "Age": 25},
			},
		)

		err := validator.Validate(output)
		if err != nil {
			t.Errorf("MalformedDataValidator.Validate() unexpected error: %v", err)
		}
	})

	t.Run("nil contents", func(t *testing.T) {
		validator := NewMalformedDataValidator(false)
		output := &OutputArray{
			Settings: NewOutputSettings(),
			Keys:     []string{"Name"},
			Contents: []OutputHolder{
				{Contents: nil},
			},
		}

		err := validator.Validate(output)
		if err == nil {
			t.Error("MalformedDataValidator.Validate() expected error for nil contents")
		}

		if !strings.Contains(err.Error(), "contents cannot be nil") {
			t.Errorf("Error should mention nil contents: %v", err)
		}
	})

	t.Run("malformed string values", func(t *testing.T) {
		validator := NewMalformedDataValidator(false)
		output := createTestOutputArray(
			[]string{"Name"},
			[]map[string]any{
				{"Name": "Alice\x00Bob"}, // null byte
				{"Name": "Valid Name"},
				{"Name": "Bad\ufffdData"}, // replacement character
			},
		)

		err := validator.Validate(output)
		if err == nil {
			t.Error("MalformedDataValidator.Validate() expected error for malformed strings")
		}

		validationErr, ok := err.(ValidationError)
		if !ok {
			t.Errorf("Expected ValidationError, got %T", err)
		}

		violations := validationErr.Violations()
		if len(violations) != 2 {
			t.Errorf("Expected 2 violations, got %d", len(violations))
		}
	})

	t.Run("strict mode - missing keys", func(t *testing.T) {
		validator := NewMalformedDataValidator(true)
		output := createTestOutputArray(
			[]string{"Name", "Age", "Email"},
			[]map[string]any{
				{"Name": "Alice", "Age": 30}, // missing Email
				{"Name": "Bob", "Age": 25, "Email": "bob@example.com"},
			},
		)

		err := validator.Validate(output)
		if err == nil {
			t.Error("MalformedDataValidator.Validate() in strict mode expected error for missing keys")
		}

		if !strings.Contains(err.Error(), "Email") {
			t.Errorf("Error should mention missing Email key: %v", err)
		}
	})

	t.Run("lenient mode - missing keys allowed", func(t *testing.T) {
		validator := NewMalformedDataValidator(false)
		output := createTestOutputArray(
			[]string{"Name", "Age", "Email"},
			[]map[string]any{
				{"Name": "Alice", "Age": 30}, // missing Email - should be OK in lenient mode
			},
		)

		err := validator.Validate(output)
		if err != nil {
			t.Errorf("MalformedDataValidator.Validate() in lenient mode unexpected error: %v", err)
		}
	})

	t.Run("validator names", func(t *testing.T) {
		strict := NewMalformedDataValidator(true)
		lenient := NewMalformedDataValidator(false)

		if !strings.Contains(strict.Name(), "strict mode") {
			t.Errorf("MalformedDataValidator.Name() should indicate strict mode: %v", strict.Name())
		}

		if !strings.Contains(lenient.Name(), "lenient mode") {
			t.Errorf("MalformedDataValidator.Name() should indicate lenient mode: %v", lenient.Name())
		}
	})
}

func TestConstraintImplementations(t *testing.T) {
	t.Run("PositiveNumberConstraint", func(t *testing.T) {
		constraint := PositiveNumberConstraint("Score")

		// Test positive number
		err := constraint.Check(map[string]any{"Score": 85})
		if err != nil {
			t.Errorf("PositiveNumberConstraint unexpected error for positive number: %v", err)
		}

		// Test negative number
		err = constraint.Check(map[string]any{"Score": -10})
		if err == nil {
			t.Error("PositiveNumberConstraint expected error for negative number")
		}

		// Test zero
		err = constraint.Check(map[string]any{"Score": 0})
		if err == nil {
			t.Error("PositiveNumberConstraint expected error for zero")
		}

		// Test non-numeric value
		err = constraint.Check(map[string]any{"Score": "not a number"})
		if err == nil {
			t.Error("PositiveNumberConstraint expected error for non-numeric value")
		}

		// Test different numeric types
		testCases := []any{int(5), int64(10), float32(3.14), float64(2.71)}
		for _, value := range testCases {
			err = constraint.Check(map[string]any{"Score": value})
			if err != nil {
				t.Errorf("PositiveNumberConstraint unexpected error for %T: %v", value, err)
			}
		}
	})

	t.Run("NonEmptyStringConstraint", func(t *testing.T) {
		constraint := NonEmptyStringConstraint("Name")

		// Test non-empty string
		err := constraint.Check(map[string]any{"Name": "Alice"})
		if err != nil {
			t.Errorf("NonEmptyStringConstraint unexpected error for non-empty string: %v", err)
		}

		// Test empty string
		err = constraint.Check(map[string]any{"Name": ""})
		if err == nil {
			t.Error("NonEmptyStringConstraint expected error for empty string")
		}

		// Test whitespace-only string
		err = constraint.Check(map[string]any{"Name": "   "})
		if err == nil {
			t.Error("NonEmptyStringConstraint expected error for whitespace-only string")
		}

		// Test non-string value
		err = constraint.Check(map[string]any{"Name": 123})
		if err == nil {
			t.Error("NonEmptyStringConstraint expected error for non-string value")
		}
	})

	t.Run("RangeConstraint", func(t *testing.T) {
		constraint := RangeConstraint("Score", 0, 100)

		// Test value in range
		err := constraint.Check(map[string]any{"Score": 85})
		if err != nil {
			t.Errorf("RangeConstraint unexpected error for value in range: %v", err)
		}

		// Test boundary values
		err = constraint.Check(map[string]any{"Score": 0})
		if err != nil {
			t.Errorf("RangeConstraint unexpected error for minimum boundary: %v", err)
		}

		err = constraint.Check(map[string]any{"Score": 100})
		if err != nil {
			t.Errorf("RangeConstraint unexpected error for maximum boundary: %v", err)
		}

		// Test value below range
		err = constraint.Check(map[string]any{"Score": -10})
		if err == nil {
			t.Error("RangeConstraint expected error for value below range")
		}

		// Test value above range
		err = constraint.Check(map[string]any{"Score": 150})
		if err == nil {
			t.Error("RangeConstraint expected error for value above range")
		}

		// Test non-numeric value
		err = constraint.Check(map[string]any{"Score": "not a number"})
		if err == nil {
			t.Error("RangeConstraint expected error for non-numeric value")
		}
	})
}

func TestConstraintFunc(t *testing.T) {
	constraint := NewConstraintFunc(
		"test_constraint",
		"test constraint description",
		func(row map[string]any) error {
			if value, exists := row["test"]; exists && value == "fail" {
				return fmt.Errorf("test failure")
			}
			return nil
		},
	)

	if constraint.Name() != "test_constraint" {
		t.Errorf("ConstraintFunc.Name() = %v, want 'test_constraint'", constraint.Name())
	}

	if constraint.Description() != "test constraint description" {
		t.Errorf("ConstraintFunc.Description() = %v, want 'test constraint description'", constraint.Description())
	}

	// Test passing case
	err := constraint.Check(map[string]any{"test": "pass"})
	if err != nil {
		t.Errorf("ConstraintFunc.Check() unexpected error: %v", err)
	}

	// Test failing case
	err = constraint.Check(map[string]any{"test": "fail"})
	if err == nil {
		t.Error("ConstraintFunc.Check() expected error for failing case")
	}
}

// Integration tests
func TestValidatorIntegration(t *testing.T) {
	t.Run("multiple validators with ValidationRunner", func(t *testing.T) {
		// Create a comprehensive validation setup
		requiredCols := NewRequiredColumnsValidator("Name", "Age", "Email")
		dataTypes := NewDataTypeValidator().
			WithStringColumn("Name").
			WithIntColumn("Age").
			WithStringColumn("Email")
		constraints := NewConstraintValidator().
			AddConstraint(PositiveNumberConstraint("Age")).
			AddConstraint(NonEmptyStringConstraint("Name"))
		emptyCheck := NewEmptyDatasetValidator(false)
		malformedCheck := NewMalformedDataValidator(true)

		runner := NewValidationRunner(ValidationModeCollectAll).
			AddValidators(requiredCols, dataTypes, constraints, emptyCheck, malformedCheck)

		// Test with valid data
		validOutput := createTestOutputArray(
			[]string{"Name", "Age", "Email"},
			[]map[string]any{
				{"Name": "Alice", "Age": 30, "Email": "alice@example.com"},
				{"Name": "Bob", "Age": 25, "Email": "bob@example.com"},
			},
		)

		err := runner.Validate(validOutput)
		if err != nil {
			t.Errorf("ValidationRunner with valid data unexpected error: %v", err)
		}

		// Test with invalid data
		invalidOutput := createTestOutputArray(
			[]string{"Name", "Age"}, // missing Email
			[]map[string]any{
				{"Name": "", "Age": -5}, // empty name, negative age
				{"Name": "Bob"},         // missing Age in strict mode
			},
		)

		err = runner.Validate(invalidOutput)
		if err == nil {
			t.Error("ValidationRunner with invalid data expected error")
		}

		// Should be a composite error with multiple violations
		if composite, ok := err.(*CompositeError); ok {
			if composite.Count() < 3 {
				t.Errorf("Expected multiple validation errors, got %d", composite.Count())
			}
		}
	})

	t.Run("validator chaining", func(t *testing.T) {
		// Create a chain of validators
		chain := NewValidatorChain("comprehensive validation").
			Add(NewEmptyDatasetValidator(false)).
			Add(NewRequiredColumnsValidator("Name", "Score")).
			Add(NewDataTypeValidator().WithStringColumn("Name").WithIntColumn("Score")).
			Add(NewConstraintValidator().AddConstraint(RangeConstraint("Score", 0, 100)))

		// Test with valid data
		validOutput := createTestOutputArray(
			[]string{"Name", "Score"},
			[]map[string]any{
				{"Name": "Alice", "Score": 85},
			},
		)

		err := chain.Validate(validOutput)
		if err != nil {
			t.Errorf("ValidatorChain with valid data unexpected error: %v", err)
		}

		// Test with invalid data (fails on constraint)
		invalidOutput := createTestOutputArray(
			[]string{"Name", "Score"},
			[]map[string]any{
				{"Name": "Alice", "Score": 150}, // score out of range
			},
		)

		err = chain.Validate(invalidOutput)
		if err == nil {
			t.Error("ValidatorChain with invalid data expected error")
		}
	})
}

// Benchmark tests
func BenchmarkRequiredColumnsValidator(b *testing.B) {
	validator := NewRequiredColumnsValidator("Name", "Age", "Email")
	output := createTestOutputArray(
		[]string{"Name", "Age", "Email", "Department"},
		[]map[string]any{
			{"Name": "Alice", "Age": 30, "Email": "alice@example.com", "Department": "Engineering"},
		},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(output)
	}
}

func BenchmarkDataTypeValidator(b *testing.B) {
	validator := NewDataTypeValidator().
		WithStringColumn("Name").
		WithIntColumn("Age").
		WithFloatColumn("Salary")

	output := createTestOutputArray(
		[]string{"Name", "Age", "Salary"},
		[]map[string]any{
			{"Name": "Alice", "Age": 30, "Salary": 75000.0},
			{"Name": "Bob", "Age": 25, "Salary": 65000.0},
		},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(output)
	}
}

func BenchmarkConstraintValidator(b *testing.B) {
	validator := NewConstraintValidator().
		AddConstraint(PositiveNumberConstraint("Age")).
		AddConstraint(NonEmptyStringConstraint("Name")).
		AddConstraint(RangeConstraint("Score", 0, 100))

	output := createTestOutputArray(
		[]string{"Name", "Age", "Score"},
		[]map[string]any{
			{"Name": "Alice", "Age": 30, "Score": 85},
			{"Name": "Bob", "Age": 25, "Score": 92},
		},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate(output)
	}
}
