package errors

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Violation represents a single validation failure with detailed context
type Violation struct {
	Field      string      // The field that failed validation
	Value      interface{} // The value that caused the violation
	Constraint string      // The constraint that was violated (e.g., "required", "format", "range")
	Message    string      // Human-readable description of the violation
}

// ValidationError extends OutputError with validation-specific functionality
type ValidationError interface {
	OutputError
	Violations() []Violation                     // List of all validation violations
	IsComposite() bool                           // Whether this represents multiple validation errors
	WithViolations(...Violation) ValidationError // Builder method to add violations
}

// validationError is the concrete implementation of ValidationError
type validationError struct {
	*baseError
	violations []Violation
}

// NewValidationError creates a new ValidationError with the specified code and message
func NewValidationError(code ErrorCode, message string) ValidationError {
	return &validationError{
		baseError:  newBaseError(code, message),
		violations: make([]Violation, 0),
	}
}

// NewValidationErrorWithViolations creates a new ValidationError with initial violations
func NewValidationErrorWithViolations(code ErrorCode, message string, violations ...Violation) ValidationError {
	return &validationError{
		baseError:  newBaseError(code, message),
		violations: violations,
	}
}

// Error returns the formatted error message including violations
func (e *validationError) Error() string {
	var b strings.Builder

	// Start with base error message
	b.WriteString(e.baseError.Error())

	// Add validation violations
	if len(e.violations) > 0 {
		b.WriteString("\nValidation violations:\n")
		for _, violation := range e.violations {
			fmt.Fprintf(&b, "  - %s: %s", violation.Field, violation.Message)
			if violation.Value != nil {
				fmt.Fprintf(&b, " (value: %v)", violation.Value)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// Violations returns the list of validation violations
func (e *validationError) Violations() []Violation {
	return e.violations
}

// IsComposite returns false for single validation errors
func (e *validationError) IsComposite() bool {
	return false
}

// WithViolations creates a new ValidationError with the specified violations
func (e *validationError) WithViolations(violations ...Violation) ValidationError {
	newErr := &validationError{
		baseError:  e.clone(),
		violations: make([]Violation, len(violations)),
	}
	copy(newErr.violations, violations)
	return newErr
}

// WithContext creates a new ValidationError with the specified context
func (e *validationError) WithContext(context ErrorContext) OutputError {
	newErr := &validationError{
		baseError:  e.baseError.clone().WithContext(context).(*baseError),
		violations: make([]Violation, len(e.violations)),
	}
	copy(newErr.violations, e.violations)
	return newErr
}

// WithSuggestions creates a new ValidationError with the specified suggestions
func (e *validationError) WithSuggestions(suggestions ...string) OutputError {
	newErr := &validationError{
		baseError:  e.baseError.clone().WithSuggestions(suggestions...).(*baseError),
		violations: make([]Violation, len(e.violations)),
	}
	copy(newErr.violations, e.violations)
	return newErr
}

// WithSeverity creates a new ValidationError with the specified severity
func (e *validationError) WithSeverity(severity ErrorSeverity) OutputError {
	newErr := &validationError{
		baseError:  e.baseError.clone().WithSeverity(severity).(*baseError),
		violations: make([]Violation, len(e.violations)),
	}
	copy(newErr.violations, e.violations)
	return newErr
}

// Wrap creates a new ValidationError that wraps the given error as the cause
func (e *validationError) Wrap(cause error) OutputError {
	newErr := &validationError{
		baseError:  e.baseError.clone().Wrap(cause).(*baseError),
		violations: make([]Violation, len(e.violations)),
	}
	copy(newErr.violations, e.violations)
	return newErr
}

// MarshalJSON implements json.Marshaler interface for structured logging
func (e *validationError) MarshalJSON() ([]byte, error) {
	var causeStr string
	if e.cause != nil {
		causeStr = e.cause.Error()
	}

	return json.Marshal(struct {
		Code        ErrorCode    `json:"code"`
		Severity    string       `json:"severity"`
		Message     string       `json:"message"`
		Context     ErrorContext `json:"context,omitempty"`
		Suggestions []string     `json:"suggestions,omitempty"`
		Cause       string       `json:"cause,omitempty"`
		Violations  []Violation  `json:"violations,omitempty"`
		IsComposite bool         `json:"is_composite"`
	}{
		Code:        e.code,
		Severity:    e.severity.String(),
		Message:     e.message,
		Context:     e.context,
		Suggestions: e.suggestions,
		Cause:       causeStr,
		Violations:  e.violations,
		IsComposite: false,
	})
}

// CompositeError represents multiple validation errors collected together
type CompositeError interface {
	ValidationError
	Add(ValidationError)    // Add a validation error to the collection
	AddViolation(Violation) // Add a single violation to the collection
	HasErrors() bool        // Whether there are any errors in the collection
	ErrorOrNil() error      // Returns self if has errors, nil otherwise
}

// compositeError is the concrete implementation of CompositeError
type compositeError struct {
	*baseError
	errors []ValidationError
}

// NewCompositeError creates a new CompositeError for collecting multiple validation errors
func NewCompositeError() CompositeError {
	return &compositeError{
		baseError: newBaseError(ErrCompositeValidation, "Multiple validation errors occurred"),
		errors:    make([]ValidationError, 0),
	}
}

// Error returns the formatted error message for all collected errors
func (e *compositeError) Error() string {
	var b strings.Builder

	totalViolations := len(e.Violations())
	if totalViolations == 0 {
		return e.baseError.Error()
	}

	// Build summary message
	errorWord := "error"
	if totalViolations > 1 {
		errorWord = "errors"
	}
	fmt.Fprintf(&b, "[%s] %d validation %s occurred", e.code, totalViolations, errorWord)

	// Add violations from all errors
	b.WriteString("\nValidation violations:\n")
	for _, violation := range e.Violations() {
		fmt.Fprintf(&b, "  - %s: %s", violation.Field, violation.Message)
		if violation.Value != nil {
			fmt.Fprintf(&b, " (value: %v)", violation.Value)
		}
		b.WriteString("\n")
	}

	// Add suggestions from base error
	if len(e.suggestions) > 0 {
		b.WriteString("Suggestions:\n")
		for _, suggestion := range e.suggestions {
			fmt.Fprintf(&b, "  - %s\n", suggestion)
		}
	}

	return b.String()
}

// Violations returns all violations from all collected errors
func (e *compositeError) Violations() []Violation {
	var allViolations []Violation
	for _, err := range e.errors {
		allViolations = append(allViolations, err.Violations()...)
	}
	return allViolations
}

// IsComposite returns true for composite errors
func (e *compositeError) IsComposite() bool {
	return true
}

// WithViolations creates a new CompositeError with additional violations
func (e *compositeError) WithViolations(violations ...Violation) ValidationError {
	newErr := &compositeError{
		baseError: e.clone(),
		errors:    make([]ValidationError, len(e.errors)),
	}
	copy(newErr.errors, e.errors)

	// Add violations as a new validation error
	if len(violations) > 0 {
		validationErr := NewValidationErrorWithViolations(ErrConstraintViolation, "Additional violations", violations...)
		newErr.errors = append(newErr.errors, validationErr)
	}

	return newErr
}

// Add includes a validation error in the collection
func (e *compositeError) Add(err ValidationError) {
	if err == nil {
		return
	}
	e.errors = append(e.errors, err)
}

// AddViolation adds a single violation as a new validation error
func (e *compositeError) AddViolation(violation Violation) {
	validationErr := NewValidationErrorWithViolations(ErrConstraintViolation, "Field validation failed", violation)
	e.Add(validationErr)
}

// HasErrors returns true if there are any errors in the collection
func (e *compositeError) HasErrors() bool {
	return len(e.errors) > 0
}

// ErrorOrNil returns self if there are errors, nil otherwise
func (e *compositeError) ErrorOrNil() error {
	if e.HasErrors() {
		return e
	}
	return nil
}

// Severity returns the highest severity level among all collected errors
func (e *compositeError) Severity() ErrorSeverity {
	if len(e.errors) == 0 {
		return e.severity
	}

	maxSeverity := SeverityInfo
	for _, err := range e.errors {
		if err.Severity() < maxSeverity {
			maxSeverity = err.Severity()
		}
	}
	return maxSeverity
}

// WithContext creates a new CompositeError with the specified context
func (e *compositeError) WithContext(context ErrorContext) OutputError {
	newErr := &compositeError{
		baseError: e.baseError.clone().WithContext(context).(*baseError),
		errors:    make([]ValidationError, len(e.errors)),
	}
	copy(newErr.errors, e.errors)
	return newErr
}

// WithSuggestions creates a new CompositeError with the specified suggestions
func (e *compositeError) WithSuggestions(suggestions ...string) OutputError {
	newErr := &compositeError{
		baseError: e.baseError.clone().WithSuggestions(suggestions...).(*baseError),
		errors:    make([]ValidationError, len(e.errors)),
	}
	copy(newErr.errors, e.errors)
	return newErr
}

// WithSeverity creates a new CompositeError with the specified severity
func (e *compositeError) WithSeverity(severity ErrorSeverity) OutputError {
	newErr := &compositeError{
		baseError: e.baseError.clone().WithSeverity(severity).(*baseError),
		errors:    make([]ValidationError, len(e.errors)),
	}
	copy(newErr.errors, e.errors)
	return newErr
}

// Wrap creates a new CompositeError that wraps the given error as the cause
func (e *compositeError) Wrap(cause error) OutputError {
	newErr := &compositeError{
		baseError: e.baseError.clone().Wrap(cause).(*baseError),
		errors:    make([]ValidationError, len(e.errors)),
	}
	copy(newErr.errors, e.errors)
	return newErr
}

// MarshalJSON implements json.Marshaler interface for structured logging
func (e *compositeError) MarshalJSON() ([]byte, error) {
	var causeStr string
	if e.cause != nil {
		causeStr = e.cause.Error()
	}

	return json.Marshal(struct {
		Code        ErrorCode    `json:"code"`
		Severity    string       `json:"severity"`
		Message     string       `json:"message"`
		Context     ErrorContext `json:"context,omitempty"`
		Suggestions []string     `json:"suggestions,omitempty"`
		Cause       string       `json:"cause,omitempty"`
		Violations  []Violation  `json:"violations,omitempty"`
		IsComposite bool         `json:"is_composite"`
		ErrorCount  int          `json:"error_count"`
	}{
		Code:        e.code,
		Severity:    e.Severity().String(),
		Message:     e.message,
		Context:     e.context,
		Suggestions: e.suggestions,
		Cause:       causeStr,
		Violations:  e.Violations(),
		IsComposite: true,
		ErrorCount:  len(e.errors),
	})
}

// newBaseError is a helper function to create a baseError (used internally)
func newBaseError(code ErrorCode, message string) *baseError {
	return &baseError{
		code:        code,
		severity:    SeverityError,
		message:     message,
		context:     ErrorContext{},
		suggestions: make([]string, 0),
	}
}
