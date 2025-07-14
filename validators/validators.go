// Package validators provides a flexible validation framework for the go-output library.
// It includes core validator interfaces, validation chains, and context management
// for comprehensive data and configuration validation.
package validators

import (
	"github.com/ArjenSchwarz/go-output/errors"
)

// Validator defines the interface for all validators in the system
type Validator interface {
	Validate(subject interface{}) error // Validates the given subject
	Name() string                       // Returns the name of the validator for identification
}

// ValidatorFunc is a function type that implements the Validator interface
// This allows simple functions to be used as validators
type ValidatorFunc func(interface{}) error

// Validate implements the Validator interface for ValidatorFunc
func (f ValidatorFunc) Validate(subject interface{}) error {
	return f(subject)
}

// Name returns the name of the ValidatorFunc
func (f ValidatorFunc) Name() string {
	return "ValidatorFunc"
}

// ValidationContext provides additional context for validation operations
type ValidationContext struct {
	Operation string                 // The operation being performed (e.g., "pre-output", "settings-validation")
	Field     string                 // The specific field being validated (if applicable)
	Metadata  map[string]interface{} // Additional context-specific metadata
}

// ValidatorChain allows running multiple validators in sequence with configurable behavior
type ValidatorChain struct {
	validators []Validator
	failFast   bool // Whether to stop on first error or collect all errors
}

// NewValidatorChain creates a new ValidatorChain with fail-fast mode enabled by default
func NewValidatorChain() *ValidatorChain {
	return &ValidatorChain{
		validators: make([]Validator, 0),
		failFast:   true,
	}
}

// Add appends a validator to the chain
func (vc *ValidatorChain) Add(validator Validator) {
	if validator != nil {
		vc.validators = append(vc.validators, validator)
	}
}

// SetFailFast configures whether the chain should stop on first error (true)
// or collect all errors (false)
func (vc *ValidatorChain) SetFailFast(failFast bool) {
	vc.failFast = failFast
}

// Validate runs all validators in the chain against the subject
func (vc *ValidatorChain) Validate(subject interface{}) error {
	return vc.ValidateWithContext(subject, ValidationContext{})
}

// ValidateWithContext runs all validators with additional context
func (vc *ValidatorChain) ValidateWithContext(subject interface{}, context ValidationContext) error {
	if len(vc.validators) == 0 {
		return nil // Empty chain always passes
	}

	if vc.failFast {
		return vc.validateFailFast(subject, context)
	}
	return vc.validateCollectAll(subject, context)
}

// validateFailFast stops on the first validation error
func (vc *ValidatorChain) validateFailFast(subject interface{}, context ValidationContext) error {
	for _, validator := range vc.validators {
		if err := validator.Validate(subject); err != nil {
			// Add context information to the error if it's a validation error
			if validationErr, ok := err.(errors.ValidationError); ok {
				return validationErr.WithContext(errors.ErrorContext{
					Operation: context.Operation,
					Field:     context.Field,
					Metadata:  context.Metadata,
				})
			}
			return err
		}
	}
	return nil
}

// validateCollectAll collects all validation errors before returning
func (vc *ValidatorChain) validateCollectAll(subject interface{}, context ValidationContext) error {
	composite := errors.NewCompositeError()

	for _, validator := range vc.validators {
		if err := validator.Validate(subject); err != nil {
			if validationErr, ok := err.(errors.ValidationError); ok {
				// Add context to the validation error
				contextualErr := validationErr.WithContext(errors.ErrorContext{
					Operation: context.Operation,
					Field:     context.Field,
					Metadata:  context.Metadata,
				})
				composite.Add(contextualErr.(errors.ValidationError))
			} else {
				// Wrap regular errors as validation errors
				wrappedErr := errors.NewValidationError(
					errors.ErrConstraintViolation,
					"Validation failed: "+err.Error(),
				)
				composite.Add(wrappedErr)
			}
		}
	}

	return composite.ErrorOrNil()
}

// Name returns the name of the ValidatorChain
func (vc *ValidatorChain) Name() string {
	return "ValidatorChain"
}

// namedValidator wraps another validator with a custom name
type namedValidator struct {
	name      string
	validator Validator
}

// NewNamedValidator creates a validator with a custom name
func NewNamedValidator(name string, validator Validator) Validator {
	return &namedValidator{
		name:      name,
		validator: validator,
	}
}

// Validate delegates to the wrapped validator
func (nv *namedValidator) Validate(subject interface{}) error {
	return nv.validator.Validate(subject)
}

// Name returns the custom name
func (nv *namedValidator) Name() string {
	return nv.name
}
