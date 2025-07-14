package validators

import (
	"fmt"
	"reflect"

	"github.com/ArjenSchwarz/go-output/errors"
)

// OutputArray interface defines the methods required for data validation
// This interface allows validators to work with different OutputArray implementations
type OutputArray interface {
	GetKeys() []string
	GetContents() []OutputHolder
}

// OutputHolder interface defines the methods required for accessing row data
type OutputHolder interface {
	GetContents() map[string]interface{}
}

// RequiredColumnsValidator validates that specified columns exist in the data
type RequiredColumnsValidator struct {
	requiredColumns []string
}

// NewRequiredColumnsValidator creates a new RequiredColumnsValidator
func NewRequiredColumnsValidator(columns ...string) *RequiredColumnsValidator {
	return &RequiredColumnsValidator{
		requiredColumns: columns,
	}
}

// Validate checks if all required columns are present in the OutputArray
func (v *RequiredColumnsValidator) Validate(subject interface{}) error {
	// Handle the mock structure for testing
	if mockOutput, ok := subject.(*mockOutputArray); ok {
		return v.validateMockOutput(mockOutput)
	}

	// Handle the real OutputArray structure (will be implemented when integrating)
	outputArray, ok := subject.(OutputArray)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"RequiredColumnsValidator requires an OutputArray",
		).WithContext(errors.ErrorContext{
			Operation: "column_validation",
			Value:     fmt.Sprintf("%T", subject),
		})
	}

	return v.validateOutputArray(outputArray)
}

// validateMockOutput validates mock output for testing
func (v *RequiredColumnsValidator) validateMockOutput(mockOutput *mockOutputArray) error {
	missing := v.findMissingColumns(mockOutput.Keys)
	if len(missing) == 0 {
		return nil
	}

	return v.createMissingColumnsError(missing)
}

// validateOutputArray validates real OutputArray
func (v *RequiredColumnsValidator) validateOutputArray(outputArray OutputArray) error {
	missing := v.findMissingColumns(outputArray.GetKeys())
	if len(missing) == 0 {
		return nil
	}

	return v.createMissingColumnsError(missing)
}

// findMissingColumns returns a slice of missing required columns
func (v *RequiredColumnsValidator) findMissingColumns(availableKeys []string) []string {
	missing := make([]string, 0)
	
	for _, required := range v.requiredColumns {
		found := false
		for _, available := range availableKeys {
			if available == required {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, required)
		}
	}
	
	return missing
}

// createMissingColumnsError creates a validation error for missing columns
func (v *RequiredColumnsValidator) createMissingColumnsError(missing []string) errors.ValidationError {
	violations := make([]errors.Violation, len(missing))
	for i, col := range missing {
		violations[i] = errors.Violation{
			Field:      col,
			Constraint: "required",
			Message:    fmt.Sprintf("Required column '%s' is missing", col),
		}
	}

	message := fmt.Sprintf("Missing %d required column(s): %v", len(missing), missing)
	return errors.NewValidationErrorWithViolations(
		errors.ErrMissingColumn,
		message,
		violations...,
	)
}

// Name returns the validator name
func (v *RequiredColumnsValidator) Name() string {
	return "RequiredColumnsValidator"
}

// DataTypeValidator validates that column data types match expectations
type DataTypeValidator struct {
	columnTypes map[string]reflect.Type
}

// NewDataTypeValidator creates a new DataTypeValidator
func NewDataTypeValidator(columnTypes map[string]reflect.Type) *DataTypeValidator {
	return &DataTypeValidator{
		columnTypes: columnTypes,
	}
}

// Validate checks if data types match the expected types
func (v *DataTypeValidator) Validate(subject interface{}) error {
	// Handle the mock structure for testing
	if mockOutput, ok := subject.(*mockOutputArray); ok {
		return v.validateMockOutput(mockOutput)
	}

	// Handle the real OutputArray structure (will be implemented when integrating)
	outputArray, ok := subject.(OutputArray)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"DataTypeValidator requires an OutputArray",
		)
	}

	return v.validateOutputArray(outputArray)
}

// validateMockOutput validates mock output for testing
func (v *DataTypeValidator) validateMockOutput(mockOutput *mockOutputArray) error {
	violations := make([]errors.Violation, 0)

	for rowIndex, holder := range mockOutput.Contents {
		rowViolations := v.validateRow(holder.Contents, rowIndex)
		violations = append(violations, rowViolations...)
	}

	if len(violations) == 0 {
		return nil
	}

	return v.createTypeValidationError(violations)
}

// validateOutputArray validates real OutputArray
func (v *DataTypeValidator) validateOutputArray(outputArray OutputArray) error {
	violations := make([]errors.Violation, 0)

	for rowIndex, holder := range outputArray.GetContents() {
		rowViolations := v.validateRow(holder.GetContents(), rowIndex)
		violations = append(violations, rowViolations...)
	}

	if len(violations) == 0 {
		return nil
	}

	return v.createTypeValidationError(violations)
}

// validateRow validates data types for a single row
func (v *DataTypeValidator) validateRow(row map[string]interface{}, rowIndex int) []errors.Violation {
	violations := make([]errors.Violation, 0)

	for column, expectedType := range v.columnTypes {
		value, exists := row[column]
		if !exists {
			// Missing fields are not type errors - they should be caught by RequiredColumnsValidator
			continue
		}

		actualType := reflect.TypeOf(value)
		if actualType != expectedType {
			violations = append(violations, errors.Violation{
				Field:      column,
				Value:      value,
				Constraint: fmt.Sprintf("type:%s", expectedType.String()),
				Message: fmt.Sprintf("Expected type %s, got %s in row %d",
					expectedType.String(), actualType.String(), rowIndex),
			})
		}
	}

	return violations
}

// createTypeValidationError creates a validation error for type mismatches
func (v *DataTypeValidator) createTypeValidationError(violations []errors.Violation) errors.ValidationError {
	message := fmt.Sprintf("Found %d data type violation(s)", len(violations))
	return errors.NewValidationErrorWithViolations(
		errors.ErrInvalidDataType,
		message,
		violations...,
	)
}

// Name returns the validator name
func (v *DataTypeValidator) Name() string {
	return "DataTypeValidator"
}

// NotEmptyValidator validates that the dataset is not empty
type NotEmptyValidator struct{}

// NewNotEmptyValidator creates a new NotEmptyValidator
func NewNotEmptyValidator() *NotEmptyValidator {
	return &NotEmptyValidator{}
}

// Validate checks if the dataset contains at least one row
func (v *NotEmptyValidator) Validate(subject interface{}) error {
	// Handle the mock structure for testing
	if mockOutput, ok := subject.(*mockOutputArray); ok {
		return v.validateMockOutput(mockOutput)
	}

	// Handle the real OutputArray structure (will be implemented when integrating)
	outputArray, ok := subject.(OutputArray)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"NotEmptyValidator requires an OutputArray",
		)
	}

	return v.validateOutputArray(outputArray)
}

// validateMockOutput validates mock output for testing
func (v *NotEmptyValidator) validateMockOutput(mockOutput *mockOutputArray) error {
	if len(mockOutput.Contents) == 0 {
		return v.createEmptyDatasetError()
	}
	return nil
}

// validateOutputArray validates real OutputArray
func (v *NotEmptyValidator) validateOutputArray(outputArray OutputArray) error {
	if len(outputArray.GetContents()) == 0 {
		return v.createEmptyDatasetError()
	}
	return nil
}

// createEmptyDatasetError creates a validation error for empty datasets
func (v *NotEmptyValidator) createEmptyDatasetError() errors.ValidationError {
	err := errors.NewValidationError(
		errors.ErrEmptyDataset,
		"Dataset cannot be empty",
	)
	contextualErr := err.WithContext(errors.ErrorContext{
		Operation: "dataset_validation",
	})
	return contextualErr.(errors.ValidationError)
}

// Name returns the validator name
func (v *NotEmptyValidator) Name() string {
	return "NotEmptyValidator"
}

// Constraint defines an interface for custom business rules that can be applied to data rows
type Constraint interface {
	Check(row map[string]interface{}) error // Validates a single row against the constraint
	Description() string                    // Returns a human-readable description of the constraint
}

// ConstraintValidator applies custom business rules to data rows
type ConstraintValidator struct {
	constraints []Constraint
}

// NewConstraintValidator creates a new ConstraintValidator
func NewConstraintValidator(constraints ...Constraint) *ConstraintValidator {
	return &ConstraintValidator{
		constraints: constraints,
	}
}

// Validate applies all constraints to each row in the dataset
func (v *ConstraintValidator) Validate(subject interface{}) error {
	// Handle the mock structure for testing
	if mockOutput, ok := subject.(*mockOutputArray); ok {
		return v.validateMockOutput(mockOutput)
	}

	// Handle the real OutputArray structure (will be implemented when integrating)
	outputArray, ok := subject.(OutputArray)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			"ConstraintValidator requires an OutputArray",
		)
	}

	return v.validateOutputArray(outputArray)
}

// validateMockOutput validates mock output for testing
func (v *ConstraintValidator) validateMockOutput(mockOutput *mockOutputArray) error {
	composite := errors.NewCompositeError()

	for rowIndex, holder := range mockOutput.Contents {
		for _, constraint := range v.constraints {
			if err := constraint.Check(holder.Contents); err != nil {
				if validationErr, ok := err.(errors.ValidationError); ok {
					// Add row context to the error
					contextualErr := validationErr.WithContext(errors.ErrorContext{
						Operation: "constraint_validation",
						Index:     rowIndex,
						Metadata: map[string]interface{}{
							"constraint": constraint.Description(),
						},
					})
					composite.Add(contextualErr.(errors.ValidationError))
				} else {
					// Wrap regular errors as validation errors
					wrappedErr := errors.NewValidationError(
						errors.ErrConstraintViolation,
						fmt.Sprintf("Constraint violation in row %d: %s", rowIndex, err.Error()),
					)
					composite.Add(wrappedErr)
				}
			}
		}
	}

	return composite.ErrorOrNil()
}

// validateOutputArray validates real OutputArray
func (v *ConstraintValidator) validateOutputArray(outputArray OutputArray) error {
	composite := errors.NewCompositeError()

	for rowIndex, holder := range outputArray.GetContents() {
		for _, constraint := range v.constraints {
			if err := constraint.Check(holder.GetContents()); err != nil {
				if validationErr, ok := err.(errors.ValidationError); ok {
					// Add row context to the error
					contextualErr := validationErr.WithContext(errors.ErrorContext{
						Operation: "constraint_validation",
						Index:     rowIndex,
						Metadata: map[string]interface{}{
							"constraint": constraint.Description(),
						},
					})
					composite.Add(contextualErr.(errors.ValidationError))
				} else {
					// Wrap regular errors as validation errors
					wrappedErr := errors.NewValidationError(
						errors.ErrConstraintViolation,
						fmt.Sprintf("Constraint violation in row %d: %s", rowIndex, err.Error()),
					)
					composite.Add(wrappedErr)
				}
			}
		}
	}

	return composite.ErrorOrNil()
}

// Name returns the validator name
func (v *ConstraintValidator) Name() string {
	return "ConstraintValidator"
}

// Common constraint implementations

// PositiveNumberConstraint ensures numeric fields are positive
type PositiveNumberConstraint struct {
	field string
}

// NewPositiveNumberConstraint creates a constraint that validates positive numbers
func NewPositiveNumberConstraint(field string) *PositiveNumberConstraint {
	return &PositiveNumberConstraint{field: field}
}

// Check validates that the field contains a positive number
func (c *PositiveNumberConstraint) Check(row map[string]interface{}) error {
	value, exists := row[c.field]
	if !exists {
		return nil // Missing fields are handled by other validators
	}

	var isPositive bool
	var actualValue interface{}

	switch v := value.(type) {
	case int:
		isPositive = v > 0
		actualValue = v
	case int64:
		isPositive = v > 0
		actualValue = v
	case float64:
		isPositive = v > 0
		actualValue = v
	case float32:
		isPositive = v > 0
		actualValue = v
	default:
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			fmt.Sprintf("Field '%s' must be numeric for positive validation", c.field),
		)
	}

	if !isPositive {
		return errors.NewValidationErrorWithViolations(
			errors.ErrConstraintViolation,
			fmt.Sprintf("Field '%s' must be positive", c.field),
			errors.Violation{
				Field:      c.field,
				Value:      actualValue,
				Constraint: "positive",
				Message:    fmt.Sprintf("Value %v is not positive", actualValue),
			},
		)
	}

	return nil
}

// Description returns the constraint description
func (c *PositiveNumberConstraint) Description() string {
	return fmt.Sprintf("Field '%s' must be positive", c.field)
}

// NonEmptyStringConstraint ensures string fields are not empty
type NonEmptyStringConstraint struct {
	field string
}

// NewNonEmptyStringConstraint creates a constraint that validates non-empty strings
func NewNonEmptyStringConstraint(field string) *NonEmptyStringConstraint {
	return &NonEmptyStringConstraint{field: field}
}

// Check validates that the field contains a non-empty string
func (c *NonEmptyStringConstraint) Check(row map[string]interface{}) error {
	value, exists := row[c.field]
	if !exists {
		return nil // Missing fields are handled by other validators
	}

	strValue, ok := value.(string)
	if !ok {
		return errors.NewValidationError(
			errors.ErrInvalidDataType,
			fmt.Sprintf("Field '%s' must be a string for non-empty validation", c.field),
		)
	}

	if strValue == "" {
		return errors.NewValidationErrorWithViolations(
			errors.ErrConstraintViolation,
			fmt.Sprintf("Field '%s' cannot be empty", c.field),
			errors.Violation{
				Field:      c.field,
				Value:      strValue,
				Constraint: "non-empty",
				Message:    "String value cannot be empty",
			},
		)
	}

	return nil
}

// Description returns the constraint description
func (c *NonEmptyStringConstraint) Description() string {
	return fmt.Sprintf("Field '%s' cannot be empty", c.field)
}