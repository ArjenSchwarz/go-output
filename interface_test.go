package format

import (
	"testing"
)

// TestInterfaceImplementation verifies that our error types implement the expected interfaces
func TestInterfaceImplementation(t *testing.T) {
	// Test that baseError implements OutputError
	var _ OutputError = &baseError{}

	// Test that validationError implements ValidationError and OutputError
	var _ ValidationError = &validationError{}
	var _ OutputError = &validationError{}

	// Test that processingError implements ProcessingError and OutputError
	var _ ProcessingError = &processingError{}
	var _ OutputError = &processingError{}

	// Test that all error types implement the standard error interface
	var _ error = &baseError{}
	var _ error = &validationError{}
	var _ error = &processingError{}
}

// TestErrorTypeCreation verifies that constructor functions return the correct types
func TestErrorTypeCreation(t *testing.T) {
	// Test OutputError creation
	outputErr := NewOutputError(ErrInvalidFormat, SeverityError, "test")
	if outputErr == nil {
		t.Error("NewOutputError should not return nil")
	}
	if outputErr.Code() != ErrInvalidFormat {
		t.Error("OutputError should have correct error code")
	}

	// Test ValidationError creation and interface compliance
	validationErr := NewValidationError(ErrMissingColumn, "test")
	if validationErr == nil {
		t.Error("NewValidationError should not return nil")
	}
	// Test that ValidationError can be used as OutputError
	var outputInterface OutputError = validationErr
	if outputInterface.Code() != ErrMissingColumn {
		t.Error("ValidationError should work as OutputError interface")
	}

	// Test ProcessingError creation and interface compliance
	processingErr := NewProcessingError(ErrFileWrite, "test", true)
	if processingErr == nil {
		t.Error("NewProcessingError should not return nil")
	}
	// Test that ProcessingError can be used as OutputError
	var outputInterface2 OutputError = processingErr
	if outputInterface2.Code() != ErrFileWrite {
		t.Error("ProcessingError should work as OutputError interface")
	}

	// Test ConfigError creation
	configErr := NewConfigError(ErrInvalidFormat, "test")
	if configErr == nil {
		t.Error("NewConfigError should not return nil")
	}
	if configErr.Code() != ErrInvalidFormat {
		t.Error("ConfigError should have correct error code")
	}
}
