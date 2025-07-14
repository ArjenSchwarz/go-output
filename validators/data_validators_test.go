package validators

import (
	"reflect"
	"testing"

	"github.com/ArjenSchwarz/go-output/errors"
)

// Mock OutputArray structure for testing
type mockOutputArray struct {
	Keys     []string
	Contents []mockOutputHolder
}

// Implement OutputArray interface
func (m *mockOutputArray) GetKeys() []string {
	return m.Keys
}

func (m *mockOutputArray) GetContents() []OutputHolder {
	holders := make([]OutputHolder, len(m.Contents))
	for i, content := range m.Contents {
		holders[i] = &content
	}
	return holders
}

type mockOutputHolder struct {
	Contents map[string]interface{}
}

// Implement OutputHolder interface
func (m *mockOutputHolder) GetContents() map[string]interface{} {
	return m.Contents
}

func TestRequiredColumnsValidator(t *testing.T) {
	tests := []struct {
		name          string
		requiredCols  []string
		outputKeys    []string
		expectedError bool
		expectedCode  errors.ErrorCode
		expectedField string
	}{
		{
			name:          "All required columns present",
			requiredCols:  []string{"Name", "Value"},
			outputKeys:    []string{"Name", "Value", "Description"},
			expectedError: false,
		},
		{
			name:          "Missing single required column",
			requiredCols:  []string{"Name", "Value", "Status"},
			outputKeys:    []string{"Name", "Value"},
			expectedError: true,
			expectedCode:  errors.ErrMissingColumn,
			expectedField: "Status",
		},
		{
			name:          "Missing multiple required columns",
			requiredCols:  []string{"Name", "Value", "Status", "Type"},
			outputKeys:    []string{"Name"},
			expectedError: true,
			expectedCode:  errors.ErrMissingColumn,
		},
		{
			name:          "Empty required columns list",
			requiredCols:  []string{},
			outputKeys:    []string{"Name", "Value"},
			expectedError: false,
		},
		{
			name:          "Empty output keys",
			requiredCols:  []string{"Name"},
			outputKeys:    []string{},
			expectedError: true,
			expectedCode:  errors.ErrMissingColumn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewRequiredColumnsValidator(tt.requiredCols...)

			mockOutput := &mockOutputArray{
				Keys: tt.outputKeys,
			}

			err := validator.Validate(mockOutput)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, validationErr.Code())
				}

				violations := validationErr.Violations()
				if len(violations) == 0 {
					t.Errorf("Expected violations, got none")
				}

				if tt.expectedField != "" {
					found := false
					for _, violation := range violations {
						if violation.Field == tt.expectedField {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected violation for field %s", tt.expectedField)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestRequiredColumnsValidatorName(t *testing.T) {
	validator := NewRequiredColumnsValidator("Name", "Value")
	expected := "RequiredColumnsValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}

func TestRequiredColumnsValidatorWithInvalidInput(t *testing.T) {
	validator := NewRequiredColumnsValidator("Name")

	// Test with non-OutputArray input
	err := validator.Validate("invalid input")
	if err == nil {
		t.Error("Expected error for invalid input type")
	}

	validationErr, ok := err.(errors.ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	}

	if validationErr.Code() != errors.ErrInvalidDataType {
		t.Errorf("Expected error code %s, got %s", errors.ErrInvalidDataType, validationErr.Code())
	}
}

func TestDataTypeValidator(t *testing.T) {
	tests := []struct {
		name          string
		columnTypes   map[string]reflect.Type
		mockData      []mockOutputHolder
		expectedError bool
		expectedCode  errors.ErrorCode
		expectedField string
	}{
		{
			name: "All types match",
			columnTypes: map[string]reflect.Type{
				"Name":  reflect.TypeOf(""),
				"Count": reflect.TypeOf(0),
				"Price": reflect.TypeOf(0.0),
			},
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Name": "test", "Count": 5, "Price": 10.5}},
				{Contents: map[string]interface{}{"Name": "test2", "Count": 3, "Price": 20.0}},
			},
			expectedError: false,
		},
		{
			name: "Type mismatch on string field",
			columnTypes: map[string]reflect.Type{
				"Name":  reflect.TypeOf(""),
				"Count": reflect.TypeOf(0),
			},
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Name": "test", "Count": 5}},
				{Contents: map[string]interface{}{"Name": 123, "Count": 3}}, // Name should be string
			},
			expectedError: true,
			expectedCode:  errors.ErrInvalidDataType,
			expectedField: "Name",
		},
		{
			name: "Type mismatch on numeric field",
			columnTypes: map[string]reflect.Type{
				"Count": reflect.TypeOf(0),
			},
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Count": "not a number"}},
			},
			expectedError: true,
			expectedCode:  errors.ErrInvalidDataType,
			expectedField: "Count",
		},
		{
			name: "Missing field in data",
			columnTypes: map[string]reflect.Type{
				"Name": reflect.TypeOf(""),
			},
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Other": "value"}},
			},
			expectedError: false, // Missing fields are not a type error
		},
		{
			name:        "Empty type specification",
			columnTypes: map[string]reflect.Type{},
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Name": "test"}},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewDataTypeValidator(tt.columnTypes)

			mockOutput := &mockOutputArray{
				Keys:     getKeysFromTypes(tt.columnTypes),
				Contents: tt.mockData,
			}

			err := validator.Validate(mockOutput)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, validationErr.Code())
				}

				if tt.expectedField != "" {
					violations := validationErr.Violations()
					found := false
					for _, violation := range violations {
						if violation.Field == tt.expectedField {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected violation for field %s", tt.expectedField)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestDataTypeValidatorName(t *testing.T) {
	validator := NewDataTypeValidator(map[string]reflect.Type{})
	expected := "DataTypeValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}

func TestNotEmptyValidator(t *testing.T) {
	tests := []struct {
		name          string
		mockData      []mockOutputHolder
		expectedError bool
		expectedCode  errors.ErrorCode
	}{
		{
			name: "Non-empty dataset",
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Name": "test"}},
			},
			expectedError: false,
		},
		{
			name:          "Empty dataset",
			mockData:      []mockOutputHolder{},
			expectedError: true,
			expectedCode:  errors.ErrEmptyDataset,
		},
		{
			name:          "Nil dataset",
			mockData:      nil,
			expectedError: true,
			expectedCode:  errors.ErrEmptyDataset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewNotEmptyValidator()

			mockOutput := &mockOutputArray{
				Contents: tt.mockData,
			}

			err := validator.Validate(mockOutput)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, validationErr.Code())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestNotEmptyValidatorName(t *testing.T) {
	validator := NewNotEmptyValidator()
	expected := "NotEmptyValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}

func TestNotEmptyValidatorWithInvalidInput(t *testing.T) {
	validator := NewNotEmptyValidator()

	err := validator.Validate("invalid input")
	if err == nil {
		t.Error("Expected error for invalid input type")
	}

	validationErr, ok := err.(errors.ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	}

	if validationErr.Code() != errors.ErrInvalidDataType {
		t.Errorf("Expected error code %s, got %s", errors.ErrInvalidDataType, validationErr.Code())
	}
}

func TestConstraint(t *testing.T) {
	// Test the Constraint interface with a simple implementation
	constraint := &mockConstraint{
		description: "Value must be positive",
		checkFunc: func(row map[string]interface{}) error {
			if val, ok := row["Value"].(int); ok && val <= 0 {
				return errors.NewValidationError(errors.ErrConstraintViolation, "Value must be positive")
			}
			return nil
		},
	}

	// Test Description method
	if constraint.Description() != "Value must be positive" {
		t.Errorf("Expected description 'Value must be positive', got %s", constraint.Description())
	}

	// Test Check method with valid data
	validRow := map[string]interface{}{"Value": 5}
	if err := constraint.Check(validRow); err != nil {
		t.Errorf("Expected no error for valid data, got %v", err)
	}

	// Test Check method with invalid data
	invalidRow := map[string]interface{}{"Value": -1}
	if err := constraint.Check(invalidRow); err == nil {
		t.Error("Expected error for invalid data, got nil")
	}
}

func TestConstraintValidator(t *testing.T) {
	positiveConstraint := &mockConstraint{
		description: "Value must be positive",
		checkFunc: func(row map[string]interface{}) error {
			if val, ok := row["Value"].(int); ok && val <= 0 {
				return errors.NewValidationError(errors.ErrConstraintViolation, "Value must be positive")
			}
			return nil
		},
	}

	nameConstraint := &mockConstraint{
		description: "Name must not be empty",
		checkFunc: func(row map[string]interface{}) error {
			if val, ok := row["Name"].(string); ok && val == "" {
				return errors.NewValidationError(errors.ErrConstraintViolation, "Name cannot be empty")
			}
			return nil
		},
	}

	tests := []struct {
		name          string
		constraints   []Constraint
		mockData      []mockOutputHolder
		expectedError bool
		expectedCode  errors.ErrorCode
	}{
		{
			name:        "All constraints pass",
			constraints: []Constraint{positiveConstraint},
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Value": 5}},
				{Contents: map[string]interface{}{"Value": 10}},
			},
			expectedError: false,
		},
		{
			name:        "Single constraint violation",
			constraints: []Constraint{positiveConstraint},
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Value": 5}},
				{Contents: map[string]interface{}{"Value": -1}},
			},
			expectedError: true,
			expectedCode:  errors.ErrCompositeValidation,
		},
		{
			name:        "Multiple constraint violations",
			constraints: []Constraint{positiveConstraint, nameConstraint},
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Value": -1, "Name": ""}},
			},
			expectedError: true,
			expectedCode:  errors.ErrCompositeValidation,
		},
		{
			name:        "No constraints",
			constraints: []Constraint{},
			mockData: []mockOutputHolder{
				{Contents: map[string]interface{}{"Value": -1}},
			},
			expectedError: false,
		},
		{
			name:          "Empty dataset with constraints",
			constraints:   []Constraint{positiveConstraint},
			mockData:      []mockOutputHolder{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewConstraintValidator(tt.constraints...)

			mockOutput := &mockOutputArray{
				Contents: tt.mockData,
			}

			err := validator.Validate(mockOutput)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}

				validationErr, ok := err.(errors.ValidationError)
				if !ok {
					t.Errorf("Expected ValidationError, got %T", err)
					return
				}

				if validationErr.Code() != tt.expectedCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedCode, validationErr.Code())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestConstraintValidatorName(t *testing.T) {
	validator := NewConstraintValidator()
	expected := "ConstraintValidator"
	if validator.Name() != expected {
		t.Errorf("Expected name %s, got %s", expected, validator.Name())
	}
}

func TestConstraintValidatorWithInvalidInput(t *testing.T) {
	validator := NewConstraintValidator()

	err := validator.Validate("invalid input")
	if err == nil {
		t.Error("Expected error for invalid input type")
	}

	validationErr, ok := err.(errors.ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	}

	if validationErr.Code() != errors.ErrInvalidDataType {
		t.Errorf("Expected error code %s, got %s", errors.ErrInvalidDataType, validationErr.Code())
	}
}

// Helper functions and mock types

func getKeysFromTypes(columnTypes map[string]reflect.Type) []string {
	keys := make([]string, 0, len(columnTypes))
	for key := range columnTypes {
		keys = append(keys, key)
	}
	return keys
}

// mockConstraint implements the Constraint interface for testing
type mockConstraint struct {
	description string
	checkFunc   func(map[string]interface{}) error
}

func (mc *mockConstraint) Check(row map[string]interface{}) error {
	return mc.checkFunc(row)
}

func (mc *mockConstraint) Description() string {
	return mc.description
}
