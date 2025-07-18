package format

import (
	"fmt"
	"reflect"
	"strings"
)

// RequiredColumnsValidator validates that all required columns exist in the dataset
type RequiredColumnsValidator struct {
	columns []string
	// Performance optimization: cache validation results
	lastKeys   []string
	lastResult error
	cacheValid bool
}

// NewRequiredColumnsValidator creates a new RequiredColumnsValidator
func NewRequiredColumnsValidator(columns ...string) *RequiredColumnsValidator {
	return &RequiredColumnsValidator{
		columns: columns,
	}
}

// Validate implements the Validator interface with performance optimizations
func (v *RequiredColumnsValidator) Validate(subject any) error {
	output, ok := subject.(*OutputArray)
	if !ok {
		return NewValidationErrorBuilder(ErrInvalidDataType, "expected OutputArray").
			WithSuggestions("ensure you're validating an OutputArray instance").
			Build()
	}

	// Performance optimization: check cache first
	if v.cacheValid && v.keysEqual(output.Keys, v.lastKeys) {
		return v.lastResult
	}

	// Performance optimization: use map for O(1) lookup instead of O(n) search
	keyMap := make(map[string]bool, len(output.Keys))
	for _, key := range output.Keys {
		keyMap[key] = true
	}

	// Pre-allocate slice with capacity to avoid reallocations
	missing := make([]string, 0, len(v.columns))
	for _, required := range v.columns {
		if !keyMap[required] {
			missing = append(missing, required)
		}
	}

	var result error
	if len(missing) > 0 {
		builder := NewValidationErrorBuilder(ErrMissingColumn,
			fmt.Sprintf("missing required columns: %s", strings.Join(missing, ", ")))

		for _, col := range missing {
			builder.WithViolation(col, "required", "column is required but not found", nil)
		}

		result = builder.WithSuggestions(
			fmt.Sprintf("add the missing columns: %s", strings.Join(missing, ", ")),
			"check your data source to ensure all required columns are included",
		).Build()
	}

	// Cache the result
	v.lastKeys = make([]string, len(output.Keys))
	copy(v.lastKeys, output.Keys)
	v.lastResult = result
	v.cacheValid = true

	return result
}

// keysEqual checks if two key slices are equal (performance optimized)
func (v *RequiredColumnsValidator) keysEqual(keys1, keys2 []string) bool {
	if len(keys1) != len(keys2) {
		return false
	}
	for i, key := range keys1 {
		if key != keys2[i] {
			return false
		}
	}
	return true
}

// Name implements the Validator interface
func (v *RequiredColumnsValidator) Name() string {
	return fmt.Sprintf("required columns validator (%s)", strings.Join(v.columns, ", "))
}

// EstimatedCost implements the PerformanceAwareValidator interface
// Required columns validation is very fast - just checking keys
func (v *RequiredColumnsValidator) EstimatedCost() int {
	return 1 // Very low cost
}

// IsFailFast implements the PerformanceAwareValidator interface
// This should run early since missing columns will cause other validators to fail
func (v *RequiredColumnsValidator) IsFailFast() bool {
	return true
}

// DataTypeValidator validates that column values match expected data types
type DataTypeValidator struct {
	columnTypes map[string]reflect.Type
}

// NewDataTypeValidator creates a new DataTypeValidator
func NewDataTypeValidator() *DataTypeValidator {
	return &DataTypeValidator{
		columnTypes: make(map[string]reflect.Type),
	}
}

// WithColumnType adds a column type requirement
func (v *DataTypeValidator) WithColumnType(column string, expectedType reflect.Type) *DataTypeValidator {
	v.columnTypes[column] = expectedType
	return v
}

// WithStringColumn adds a string type requirement for a column
func (v *DataTypeValidator) WithStringColumn(column string) *DataTypeValidator {
	return v.WithColumnType(column, reflect.TypeOf(""))
}

// WithIntColumn adds an int type requirement for a column
func (v *DataTypeValidator) WithIntColumn(column string) *DataTypeValidator {
	return v.WithColumnType(column, reflect.TypeOf(0))
}

// WithFloatColumn adds a float64 type requirement for a column
func (v *DataTypeValidator) WithFloatColumn(column string) *DataTypeValidator {
	return v.WithColumnType(column, reflect.TypeOf(0.0))
}

// WithBoolColumn adds a bool type requirement for a column
func (v *DataTypeValidator) WithBoolColumn(column string) *DataTypeValidator {
	return v.WithColumnType(column, reflect.TypeOf(true))
}

// Validate implements the Validator interface
func (v *DataTypeValidator) Validate(subject any) error {
	output, ok := subject.(*OutputArray)
	if !ok {
		return NewValidationErrorBuilder(ErrInvalidDataType, "expected OutputArray").
			WithSuggestions("ensure you're validating an OutputArray instance").
			Build()
	}

	builder := NewValidationErrorBuilder(ErrInvalidDataType, "data type validation failed")
	hasViolations := false

	for rowIndex, holder := range output.Contents {
		for column, expectedType := range v.columnTypes {
			if value, exists := holder.Contents[column]; exists {
				actualType := reflect.TypeOf(value)

				// Handle nil values
				if value == nil {
					builder.WithViolation(
						fmt.Sprintf("%s[%d]", column, rowIndex),
						"type_match",
						fmt.Sprintf("expected %s, got nil", expectedType.String()),
						value,
					)
					hasViolations = true
					continue
				}

				// Check type compatibility
				if !v.isTypeCompatible(actualType, expectedType) {
					builder.WithViolation(
						fmt.Sprintf("%s[%d]", column, rowIndex),
						"type_match",
						fmt.Sprintf("expected %s, got %s", expectedType.String(), actualType.String()),
						value,
					)
					hasViolations = true
				}
			}
		}
	}

	if hasViolations {
		return builder.WithSuggestions(
			"ensure all column values match their expected data types",
			"check your data source for type consistency",
			"consider using type conversion before adding data to OutputArray",
		).Build()
	}

	return nil
}

// isTypeCompatible checks if two types are compatible
func (v *DataTypeValidator) isTypeCompatible(actual, expected reflect.Type) bool {
	// Exact match
	if actual == expected {
		return true
	}

	// Handle numeric type compatibility
	if v.isNumericType(actual) && v.isNumericType(expected) {
		return true
	}

	// Handle interface{} compatibility
	if expected.Kind() == reflect.Interface {
		return true
	}

	return false
}

// isNumericType checks if a type is numeric
func (v *DataTypeValidator) isNumericType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// Name implements the Validator interface
func (v *DataTypeValidator) Name() string {
	columns := make([]string, 0, len(v.columnTypes))
	for col, typ := range v.columnTypes {
		columns = append(columns, fmt.Sprintf("%s:%s", col, typ.String()))
	}
	return fmt.Sprintf("data type validator (%s)", strings.Join(columns, ", "))
}

// EstimatedCost implements the PerformanceAwareValidator interface
// Data type validation requires reflection and is moderately expensive
func (v *DataTypeValidator) EstimatedCost() int {
	return 5 * len(v.columnTypes) // Cost scales with number of columns to check
}

// IsFailFast implements the PerformanceAwareValidator interface
// Data type validation is not typically a fail-fast validator
func (v *DataTypeValidator) IsFailFast() bool {
	return false
}

// Constraint defines a custom business rule constraint
type Constraint interface {
	// Check validates a single row against the constraint
	Check(row map[string]any) error
	// Description returns a human-readable description of the constraint
	Description() string
	// Name returns the constraint name for violation reporting
	Name() string
}

// ConstraintFunc is a function type that implements the Constraint interface
type ConstraintFunc struct {
	name        string
	description string
	checkFunc   func(map[string]any) error
}

// NewConstraintFunc creates a new ConstraintFunc
func NewConstraintFunc(name, description string, checkFunc func(map[string]any) error) *ConstraintFunc {
	return &ConstraintFunc{
		name:        name,
		description: description,
		checkFunc:   checkFunc,
	}
}

// Check implements the Constraint interface
func (c *ConstraintFunc) Check(row map[string]any) error {
	return c.checkFunc(row)
}

// Description implements the Constraint interface
func (c *ConstraintFunc) Description() string {
	return c.description
}

// Name implements the Constraint interface
func (c *ConstraintFunc) Name() string {
	return c.name
}

// ConstraintValidator validates data against custom business rule constraints
type ConstraintValidator struct {
	constraints []Constraint
}

// NewConstraintValidator creates a new ConstraintValidator
func NewConstraintValidator() *ConstraintValidator {
	return &ConstraintValidator{
		constraints: make([]Constraint, 0),
	}
}

// AddConstraint adds a constraint to the validator
func (v *ConstraintValidator) AddConstraint(constraint Constraint) *ConstraintValidator {
	v.constraints = append(v.constraints, constraint)
	return v
}

// AddConstraints adds multiple constraints to the validator
func (v *ConstraintValidator) AddConstraints(constraints ...Constraint) *ConstraintValidator {
	v.constraints = append(v.constraints, constraints...)
	return v
}

// Validate implements the Validator interface
func (v *ConstraintValidator) Validate(subject any) error {
	output, ok := subject.(*OutputArray)
	if !ok {
		return NewValidationErrorBuilder(ErrInvalidDataType, "expected OutputArray").
			WithSuggestions("ensure you're validating an OutputArray instance").
			Build()
	}

	builder := NewValidationErrorBuilder(ErrConstraintViolation, "constraint validation failed")
	hasViolations := false

	for rowIndex, holder := range output.Contents {
		for _, constraint := range v.constraints {
			if err := constraint.Check(holder.Contents); err != nil {
				builder.WithViolation(
					fmt.Sprintf("row[%d]", rowIndex),
					constraint.Name(),
					err.Error(),
					holder.Contents,
				)
				hasViolations = true
			}
		}
	}

	if hasViolations {
		suggestions := []string{
			"review your data to ensure it meets all business rule constraints",
			"check constraint definitions for accuracy",
		}

		// Add constraint-specific suggestions
		for _, constraint := range v.constraints {
			suggestions = append(suggestions, fmt.Sprintf("constraint '%s': %s",
				constraint.Name(), constraint.Description()))
		}

		return builder.WithSuggestions(suggestions...).Build()
	}

	return nil
}

// Name implements the Validator interface
func (v *ConstraintValidator) Name() string {
	if len(v.constraints) == 0 {
		return "constraint validator (no constraints)"
	}

	names := make([]string, len(v.constraints))
	for i, constraint := range v.constraints {
		names[i] = constraint.Name()
	}
	return fmt.Sprintf("constraint validator (%s)", strings.Join(names, ", "))
}

// EstimatedCost implements the PerformanceAwareValidator interface
// Constraint validation can be expensive as it runs custom logic on each row
func (v *ConstraintValidator) EstimatedCost() int {
	return 10 * len(v.constraints) // High cost, scales with number of constraints
}

// IsFailFast implements the PerformanceAwareValidator interface
// Constraint validation is not typically a fail-fast validator
func (v *ConstraintValidator) IsFailFast() bool {
	return false
}

// EmptyDatasetValidator validates against empty datasets
type EmptyDatasetValidator struct {
	allowEmpty bool
}

// NewEmptyDatasetValidator creates a new EmptyDatasetValidator
func NewEmptyDatasetValidator(allowEmpty bool) *EmptyDatasetValidator {
	return &EmptyDatasetValidator{
		allowEmpty: allowEmpty,
	}
}

// Validate implements the Validator interface
func (v *EmptyDatasetValidator) Validate(subject any) error {
	output, ok := subject.(*OutputArray)
	if !ok {
		return NewValidationErrorBuilder(ErrInvalidDataType, "expected OutputArray").
			WithSuggestions("ensure you're validating an OutputArray instance").
			Build()
	}

	if len(output.Contents) == 0 && !v.allowEmpty {
		return NewValidationErrorBuilder(ErrEmptyDataset, "dataset cannot be empty").
			WithViolation("Contents", "not_empty", "dataset must contain at least one record", len(output.Contents)).
			WithSuggestions(
				"add data to the OutputArray before validation",
				"if empty datasets are acceptable, configure the validator to allow them",
				"check your data source to ensure it contains records",
			).Build()
	}

	return nil
}

// Name implements the Validator interface
func (v *EmptyDatasetValidator) Name() string {
	if v.allowEmpty {
		return "empty dataset validator (allows empty)"
	}
	return "empty dataset validator (requires data)"
}

// EstimatedCost implements the PerformanceAwareValidator interface
// Empty dataset validation is very fast - just checking length
func (v *EmptyDatasetValidator) EstimatedCost() int {
	return 1 // Very low cost
}

// IsFailFast implements the PerformanceAwareValidator interface
// This should run early since empty datasets will cause other validators to behave differently
func (v *EmptyDatasetValidator) IsFailFast() bool {
	return true
}

// MalformedDataValidator validates against malformed data
type MalformedDataValidator struct {
	strictMode bool
}

// NewMalformedDataValidator creates a new MalformedDataValidator
func NewMalformedDataValidator(strictMode bool) *MalformedDataValidator {
	return &MalformedDataValidator{
		strictMode: strictMode,
	}
}

// Validate implements the Validator interface
func (v *MalformedDataValidator) Validate(subject any) error {
	output, ok := subject.(*OutputArray)
	if !ok {
		return NewValidationErrorBuilder(ErrInvalidDataType, "expected OutputArray").
			WithSuggestions("ensure you're validating an OutputArray instance").
			Build()
	}

	builder := NewValidationErrorBuilder(ErrMalformedData, "malformed data detected")
	hasViolations := false

	for rowIndex, holder := range output.Contents {
		// Check for nil Contents map
		if holder.Contents == nil {
			builder.WithViolation(
				fmt.Sprintf("row[%d]", rowIndex),
				"contents_not_nil",
				"row contents cannot be nil",
				nil,
			)
			hasViolations = true
			continue
		}

		// In strict mode, check that all expected keys are present
		if v.strictMode {
			for _, expectedKey := range output.Keys {
				if _, exists := holder.Contents[expectedKey]; !exists {
					builder.WithViolation(
						fmt.Sprintf("row[%d].%s", rowIndex, expectedKey),
						"key_exists",
						fmt.Sprintf("expected key '%s' is missing", expectedKey),
						holder.Contents,
					)
					hasViolations = true
				}
			}
		}

		// Check for obviously malformed values
		for key, value := range holder.Contents {
			if v.isMalformedValue(value) {
				builder.WithViolation(
					fmt.Sprintf("row[%d].%s", rowIndex, key),
					"value_well_formed",
					"value appears to be malformed or corrupted",
					value,
				)
				hasViolations = true
			}
		}
	}

	if hasViolations {
		suggestions := []string{
			"check your data source for corruption or formatting issues",
			"ensure all required fields are present in each record",
		}

		if v.strictMode {
			suggestions = append(suggestions, "in strict mode, all expected keys must be present in each row")
		} else {
			suggestions = append(suggestions, "consider enabling strict mode for more thorough validation")
		}

		return builder.WithSuggestions(suggestions...).Build()
	}

	return nil
}

// isMalformedValue checks if a value appears to be malformed
func (v *MalformedDataValidator) isMalformedValue(value any) bool {
	if value == nil {
		return false // nil is acceptable
	}

	// Check for obviously malformed string values
	if str, ok := value.(string); ok {
		// Check for control characters (except common whitespace)
		for _, r := range str {
			if r < 32 && r != '\t' && r != '\n' && r != '\r' {
				return true
			}
		}

		// Check for common malformed patterns
		malformedPatterns := []string{
			"\x00",   // null bytes
			"\ufffd", // replacement character (indicates encoding issues)
		}

		for _, pattern := range malformedPatterns {
			if strings.Contains(str, pattern) {
				return true
			}
		}
	}

	return false
}

// Name implements the Validator interface
func (v *MalformedDataValidator) Name() string {
	if v.strictMode {
		return "malformed data validator (strict mode)"
	}
	return "malformed data validator (lenient mode)"
}

// EstimatedCost implements the PerformanceAwareValidator interface
// Malformed data validation requires string analysis and is moderately expensive
func (v *MalformedDataValidator) EstimatedCost() int {
	if v.strictMode {
		return 8 // Higher cost in strict mode due to additional checks
	}
	return 6 // Moderate cost in lenient mode
}

// IsFailFast implements the PerformanceAwareValidator interface
// Malformed data validation is not typically a fail-fast validator
func (v *MalformedDataValidator) IsFailFast() bool {
	return false
}

// Common constraint implementations

// PositiveNumberConstraint ensures numeric values are positive
func PositiveNumberConstraint(column string) Constraint {
	return NewConstraintFunc(
		fmt.Sprintf("positive_%s", column),
		fmt.Sprintf("column '%s' must contain positive numbers", column),
		func(row map[string]any) error {
			if value, exists := row[column]; exists {
				switch v := value.(type) {
				case int:
					if v <= 0 {
						return fmt.Errorf("value %d is not positive", v)
					}
				case int64:
					if v <= 0 {
						return fmt.Errorf("value %d is not positive", v)
					}
				case float64:
					if v <= 0 {
						return fmt.Errorf("value %f is not positive", v)
					}
				case float32:
					if v <= 0 {
						return fmt.Errorf("value %f is not positive", v)
					}
				default:
					return fmt.Errorf("value is not a number: %v", value)
				}
			}
			return nil
		},
	)
}

// NonEmptyStringConstraint ensures string values are not empty
func NonEmptyStringConstraint(column string) Constraint {
	return NewConstraintFunc(
		fmt.Sprintf("non_empty_%s", column),
		fmt.Sprintf("column '%s' must contain non-empty strings", column),
		func(row map[string]any) error {
			if value, exists := row[column]; exists {
				if str, ok := value.(string); ok {
					if strings.TrimSpace(str) == "" {
						return fmt.Errorf("string value cannot be empty")
					}
				} else {
					return fmt.Errorf("value is not a string: %v", value)
				}
			}
			return nil
		},
	)
}

// RangeConstraint ensures numeric values are within a specified range
func RangeConstraint(column string, min, max float64) Constraint {
	return NewConstraintFunc(
		fmt.Sprintf("range_%s_%g_%g", column, min, max),
		fmt.Sprintf("column '%s' must be between %g and %g", column, min, max),
		func(row map[string]any) error {
			if value, exists := row[column]; exists {
				var numValue float64
				var ok bool

				switch v := value.(type) {
				case int:
					numValue = float64(v)
					ok = true
				case int64:
					numValue = float64(v)
					ok = true
				case float64:
					numValue = v
					ok = true
				case float32:
					numValue = float64(v)
					ok = true
				}

				if !ok {
					return fmt.Errorf("value is not a number: %v", value)
				}

				if numValue < min || numValue > max {
					return fmt.Errorf("value %g is outside range [%g, %g]", numValue, min, max)
				}
			}
			return nil
		},
	)
}
