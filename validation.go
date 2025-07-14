package format

import (
	"fmt"
	"strings"
)

// Validator defines the interface for all validators
type Validator interface {
	// Validate performs validation on the given subject
	Validate(subject any) error
	// Name returns a human-readable name for this validator
	Name() string
}

// ValidatorFunc is a function type that implements the Validator interface
// This allows for easy creation of validators using anonymous functions
type ValidatorFunc func(any) error

// Validate implements the Validator interface for ValidatorFunc
func (f ValidatorFunc) Validate(subject any) error {
	return f(subject)
}

// Name implements the Validator interface for ValidatorFunc
// Returns a generic name since function validators don't have explicit names
func (f ValidatorFunc) Name() string {
	return "function validator"
}

// NamedValidatorFunc creates a ValidatorFunc with a custom name
func NamedValidatorFunc(name string, fn func(any) error) Validator {
	return &namedValidatorFunc{
		name: name,
		fn:   fn,
	}
}

// namedValidatorFunc is a ValidatorFunc with a custom name
type namedValidatorFunc struct {
	name string
	fn   func(any) error
}

// Validate implements the Validator interface
func (n *namedValidatorFunc) Validate(subject any) error {
	return n.fn(subject)
}

// Name implements the Validator interface
func (n *namedValidatorFunc) Name() string {
	return n.name
}

// CompositeError represents multiple validation errors collected together
type CompositeError struct {
	errors []error
}

// NewCompositeError creates a new composite error
func NewCompositeError() *CompositeError {
	return &CompositeError{
		errors: make([]error, 0),
	}
}

// Add adds an error to the composite error
func (c *CompositeError) Add(err error) {
	if err != nil {
		c.errors = append(c.errors, err)
	}
}

// AddAll adds multiple errors to the composite error
func (c *CompositeError) AddAll(errors ...error) {
	for _, err := range errors {
		c.Add(err)
	}
}

// HasErrors returns true if the composite error contains any errors
func (c *CompositeError) HasErrors() bool {
	return len(c.errors) > 0
}

// Count returns the number of errors in the composite error
func (c *CompositeError) Count() int {
	return len(c.errors)
}

// Errors returns all errors in the composite error
func (c *CompositeError) Errors() []error {
	return c.errors
}

// Error implements the error interface
func (c *CompositeError) Error() string {
	if len(c.errors) == 0 {
		return "no errors"
	}

	if len(c.errors) == 1 {
		return c.errors[0].Error()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "multiple validation errors (%d):", len(c.errors))
	for i, err := range c.errors {
		fmt.Fprintf(&b, "\n  %d. %s", i+1, err.Error())
	}

	return b.String()
}

// ErrorOrNil returns the composite error if it contains errors, otherwise nil
func (c *CompositeError) ErrorOrNil() error {
	if c.HasErrors() {
		return c
	}
	return nil
}

// ValidationRunner provides utilities for running multiple validators
type ValidationRunner struct {
	validators []Validator
	mode       ValidationMode
}

// ValidationMode determines how validation errors are handled
type ValidationMode int

const (
	// ValidationModeFailFast stops validation on the first error
	ValidationModeFailFast ValidationMode = iota
	// ValidationModeCollectAll collects all validation errors before returning
	ValidationModeCollectAll
)

// NewValidationRunner creates a new validation runner
func NewValidationRunner(mode ValidationMode) *ValidationRunner {
	return &ValidationRunner{
		validators: make([]Validator, 0),
		mode:       mode,
	}
}

// AddValidator adds a validator to the runner
func (r *ValidationRunner) AddValidator(validator Validator) *ValidationRunner {
	r.validators = append(r.validators, validator)
	return r
}

// AddValidators adds multiple validators to the runner
func (r *ValidationRunner) AddValidators(validators ...Validator) *ValidationRunner {
	r.validators = append(r.validators, validators...)
	return r
}

// Validate runs all validators against the subject
func (r *ValidationRunner) Validate(subject any) error {
	if len(r.validators) == 0 {
		return nil
	}

	switch r.mode {
	case ValidationModeFailFast:
		return r.validateFailFast(subject)
	case ValidationModeCollectAll:
		return r.validateCollectAll(subject)
	default:
		return r.validateFailFast(subject)
	}
}

// validateFailFast runs validators and returns on first error
func (r *ValidationRunner) validateFailFast(subject any) error {
	for _, validator := range r.validators {
		if err := validator.Validate(subject); err != nil {
			return err
		}
	}
	return nil
}

// validateCollectAll runs all validators and collects all errors
func (r *ValidationRunner) validateCollectAll(subject any) error {
	composite := NewCompositeError()

	for _, validator := range r.validators {
		if err := validator.Validate(subject); err != nil {
			composite.Add(err)
		}
	}

	return composite.ErrorOrNil()
}

// ValidatorChain allows chaining multiple validators together
type ValidatorChain struct {
	validators []Validator
	name       string
}

// NewValidatorChain creates a new validator chain
func NewValidatorChain(name string) *ValidatorChain {
	return &ValidatorChain{
		validators: make([]Validator, 0),
		name:       name,
	}
}

// Add adds a validator to the chain
func (c *ValidatorChain) Add(validator Validator) *ValidatorChain {
	c.validators = append(c.validators, validator)
	return c
}

// Validate implements the Validator interface
// Runs all validators in sequence and returns the first error encountered
func (c *ValidatorChain) Validate(subject any) error {
	for _, validator := range c.validators {
		if err := validator.Validate(subject); err != nil {
			return err
		}
	}
	return nil
}

// Name implements the Validator interface
func (c *ValidatorChain) Name() string {
	if c.name != "" {
		return c.name
	}
	return "validator chain"
}

// ConditionalValidator runs a validator only if a condition is met
type ConditionalValidator struct {
	condition func(any) bool
	validator Validator
	name      string
}

// NewConditionalValidator creates a new conditional validator
func NewConditionalValidator(name string, condition func(any) bool, validator Validator) *ConditionalValidator {
	return &ConditionalValidator{
		condition: condition,
		validator: validator,
		name:      name,
	}
}

// Validate implements the Validator interface
func (c *ConditionalValidator) Validate(subject any) error {
	if c.condition(subject) {
		return c.validator.Validate(subject)
	}
	return nil
}

// Name implements the Validator interface
func (c *ConditionalValidator) Name() string {
	if c.name != "" {
		return c.name
	}
	return fmt.Sprintf("conditional(%s)", c.validator.Name())
}

// ValidationContext provides context information during validation
type ValidationContext struct {
	Subject   any
	Path      string
	Metadata  map[string]any
	Validator Validator
}

// ContextualValidator is a validator that receives validation context
type ContextualValidator interface {
	Validator
	ValidateWithContext(ctx ValidationContext) error
}

// contextualValidatorAdapter adapts a regular validator to work with context
type contextualValidatorAdapter struct {
	validator Validator
}

// ValidateWithContext implements ContextualValidator interface
func (a *contextualValidatorAdapter) ValidateWithContext(ctx ValidationContext) error {
	return a.validator.Validate(ctx.Subject)
}

// Validate implements the Validator interface
func (a *contextualValidatorAdapter) Validate(subject any) error {
	return a.validator.Validate(subject)
}

// Name implements the Validator interface
func (a *contextualValidatorAdapter) Name() string {
	return a.validator.Name()
}

// AsContextual wraps a regular validator to work with validation context
func AsContextual(validator Validator) ContextualValidator {
	if contextual, ok := validator.(ContextualValidator); ok {
		return contextual
	}
	return &contextualValidatorAdapter{validator: validator}
}
