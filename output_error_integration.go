package format

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/ArjenSchwarz/go-output/errors"
	"github.com/ArjenSchwarz/go-output/validators"
)

// Add error handling fields to OutputArray struct (we'll modify the original struct after tests pass)
type ErrorEnabledOutputArray struct {
	*OutputArray
	validators      []validators.Validator
	errorHandler    errors.ErrorHandler
	recoveryHandler errors.RecoveryHandler
}

// NewErrorEnabledOutputArray creates a new error-enabled OutputArray
func NewErrorEnabledOutputArray(settings *OutputSettings) *ErrorEnabledOutputArray {
	return &ErrorEnabledOutputArray{
		OutputArray: &OutputArray{
			Settings: settings,
			Contents: make([]OutputHolder, 0),
			Keys:     make([]string, 0),
		},
		validators:      make([]validators.Validator, 0),
		errorHandler:    errors.NewDefaultErrorHandler(),
		recoveryHandler: nil,
	}
}

// Validate runs all validators against the OutputArray
func (output *OutputArray) Validate() error {
	// Initialize error handling fields if not present (for backward compatibility)
	if output.validators == nil {
		output.validators = make([]validators.Validator, 0)
	}
	if output.errorHandler == nil {
		output.errorHandler = errors.NewDefaultErrorHandler()
	}

	// Settings validation
	if err := output.Settings.Validate(); err != nil {
		return output.handleError(err)
	}

	// Format-specific validation
	if err := output.validateForFormat(); err != nil {
		return output.handleError(err)
	}

	// Data validation
	for _, validator := range output.validators {
		if err := validator.Validate(output); err != nil {
			handledErr := output.handleError(err)
			if handledErr != nil {
				return handledErr
			}
		}
	}

	return nil
}

// AddValidator adds a custom validator to the OutputArray
func (output *OutputArray) AddValidator(validator validators.Validator) {
	if output.validators == nil {
		output.validators = make([]validators.Validator, 0)
	}
	output.validators = append(output.validators, validator)
}

// WithErrorHandler sets a custom error handler and returns the OutputArray for method chaining
func (output *OutputArray) WithErrorHandler(handler errors.ErrorHandler) *OutputArray {
	output.errorHandler = handler
	return output
}

// WithRecoveryHandler sets a custom recovery handler and returns the OutputArray for method chaining
func (output *OutputArray) WithRecoveryHandler(handler errors.RecoveryHandler) *OutputArray {
	output.recoveryHandler = handler
	return output
}

// WriteWithValidation provides the new error-returning API for output generation
func (output *OutputArray) WriteWithValidation() error {
	// Validate first
	if err := output.Validate(); err != nil {
		return err
	}

	// Stop any active progress
	stopActiveProgress()

	// Generate output with error handling
	result, err := output.generateWithErrorHandling()
	if err != nil {
		return output.handleError(err)
	}

	// Write output
	if err := output.writeOutputWithErrorHandling(result); err != nil {
		return output.handleError(err)
	}

	return nil
}

// WriteCompat provides backward-compatible Write method that maintains old behavior
func (output *OutputArray) WriteCompat() {
	if err := output.WriteWithValidation(); err != nil {
		log.Fatal(err)
	}
}

// GetErrorSummary returns a summary of collected errors (for lenient mode)
func (output *OutputArray) GetErrorSummary() errors.ErrorSummary {
	if output.errorHandler == nil {
		return errors.ErrorSummary{
			BySeverity: make(map[errors.ErrorSeverity]int),
			ByCategory: make(map[errors.ErrorCode]int),
		}
	}
	return output.errorHandler.GetSummary()
}

// ClearErrors clears all collected errors
func (output *OutputArray) ClearErrors() {
	if output.errorHandler != nil {
		output.errorHandler.Clear()
	}
}

// EnableLegacyMode configures the OutputArray to use legacy error handling (log.Fatal)
func (output *OutputArray) EnableLegacyMode() *OutputArray {
	output.errorHandler = errors.NewLegacyErrorHandler()
	return output
}

// SetErrorMode sets the error handling mode
func (output *OutputArray) SetErrorMode(mode errors.ErrorMode) *OutputArray {
	if output.errorHandler == nil {
		output.errorHandler = errors.NewDefaultErrorHandler()
	}
	output.errorHandler.SetMode(mode)
	return output
}

// handleError processes errors according to the configured error handler
func (output *OutputArray) handleError(err error) error {
	if output.errorHandler == nil {
		output.errorHandler = errors.NewDefaultErrorHandler()
	}

	// Try recovery if available
	if output.recoveryHandler != nil {
		if outputErr, ok := err.(errors.OutputError); ok {
			if output.recoveryHandler.CanRecover(outputErr) {
				recoveryContext := output.createRecoveryContext()
				if recoveryErr := output.recoveryHandler.RecoverWithContext(outputErr, recoveryContext); recoveryErr == nil {
					// Recovery successful, continue processing
					return nil
				}
			}
		}
	}

	// Handle the error according to the current mode
	return output.errorHandler.HandleError(err)
}

// createRecoveryContext creates context for error recovery
func (output *OutputArray) createRecoveryContext() map[string]interface{} {
	context := make(map[string]interface{})

	if output.Settings != nil {
		context["OutputFormat"] = output.Settings.OutputFormat
		context["OutputFile"] = output.Settings.OutputFile
		context["Settings"] = output.Settings
	}

	context["Keys"] = output.Keys
	context["DataCount"] = len(output.Contents)

	return context
}

// validateForFormat performs format-specific validation
func (output *OutputArray) validateForFormat() error {
	if output.Settings == nil {
		return errors.NewError(errors.ErrMissingRequired, "OutputSettings is required")
	}

	switch output.Settings.OutputFormat {
	case "mermaid":
		if output.Settings.FromToColumns == nil && output.Settings.MermaidSettings == nil {
			return errors.NewError(
				errors.ErrMissingRequired,
				"mermaid format requires FromToColumns or MermaidSettings",
			).WithSuggestions(
				"Use AddFromToColumns() to set source and target columns",
				"Or configure MermaidSettings for chart generation",
			).WithContext(errors.ErrorContext{
				Operation: "format_validation",
				Field:     "OutputFormat",
				Value:     "mermaid",
			})
		}
	case "drawio":
		if !output.Settings.DrawIOHeader.IsSet() {
			return errors.NewError(
				errors.ErrMissingRequired,
				"drawio format requires DrawIOHeader configuration",
			).WithSuggestions(
				"Configure DrawIOHeader before using drawio format",
			).WithContext(errors.ErrorContext{
				Operation: "format_validation",
				Field:     "OutputFormat",
				Value:     "drawio",
			})
		}
	case "dot":
		if output.Settings.FromToColumns == nil {
			return errors.NewError(
				errors.ErrMissingRequired,
				"dot format requires FromToColumns configuration",
			).WithSuggestions(
				"Use AddFromToColumns() to set source and target columns",
			).WithContext(errors.ErrorContext{
				Operation: "format_validation",
				Field:     "OutputFormat",
				Value:     "dot",
			})
		}
	}

	return nil
}

// generateWithErrorHandling generates output with comprehensive error handling
func (output *OutputArray) generateWithErrorHandling() ([]byte, error) {
	var result []byte
	var err error

	defer func() {
		if r := recover(); r != nil {
			err = errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("Output generation panicked: %v", r),
			).WithSeverity(errors.SeverityFatal)
		}
	}()

	switch output.Settings.OutputFormat {
	case "csv":
		if buffer.Len() == 0 {
			result, err = output.toCSVWithError()
		} else {
			result = buffer.Bytes()
		}
	case "html":
		if buffer.Len() == 0 {
			result, err = output.toHTMLWithError()
		} else {
			result, err = output.bufferToHTMLWithError()
		}
	case "table":
		if buffer.Len() == 0 {
			result, err = output.toTableWithError()
		} else {
			result = buffer.Bytes()
		}
	case "markdown":
		if buffer.Len() == 0 {
			result, err = output.toMarkdownWithError()
		} else {
			result, err = output.bufferToMarkdownWithError()
		}
	case "mermaid":
		result, err = output.toMermaidWithError()
	case "drawio":
		err = output.toDrawIOWithError()
	case "dot":
		result, err = output.toDotWithError()
	case "yaml":
		if buffer.Len() == 0 {
			result, err = output.toYAMLWithError()
		} else {
			result = buffer.Bytes()
		}
	default:
		if buffer.Len() == 0 {
			result, err = output.toJSONWithError()
		} else {
			result = buffer.Bytes()
		}
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

// writeOutputWithErrorHandling writes output with error handling
func (output *OutputArray) writeOutputWithErrorHandling(result []byte) error {
	if len(result) != 0 {
		err := PrintByteSlice(result, "", output.Settings.S3Bucket)
		if err != nil {
			return errors.NewProcessingError(
				errors.ErrFileWrite,
				fmt.Sprintf("Failed to write output: %v", err),
			).Wrap(err)
		}
		buffer.Reset()
	}

	if output.Settings.OutputFile != "" {
		if output.Settings.OutputFileFormat == "" {
			output.Settings.OutputFileFormat = output.Settings.OutputFormat
		}

		fileResult, err := output.generateFileOutput()
		if err != nil {
			return err
		}

		if len(fileResult) != 0 {
			err := PrintByteSlice(fileResult, output.Settings.OutputFile, output.Settings.S3Bucket)
			if err != nil {
				return errors.NewProcessingError(
					errors.ErrFileWrite,
					fmt.Sprintf("Failed to write output file: %v", err),
				).Wrap(err)
			}
			buffer.Reset()
		}
	}

	return nil
}

// generateFileOutput generates output for file writing
func (output *OutputArray) generateFileOutput() ([]byte, error) {
	var result []byte
	var err error

	switch output.Settings.OutputFileFormat {
	case "csv":
		if buffer.Len() == 0 {
			result, err = output.toCSVWithError()
		} else {
			result = buffer.Bytes()
		}
	case "html":
		if buffer.Len() == 0 {
			result, err = output.toHTMLWithError()
		} else {
			result, err = output.bufferToHTMLWithError()
		}
	case "table":
		if buffer.Len() == 0 {
			result, err = output.toTableWithError()
		} else {
			result = buffer.Bytes()
		}
	case "markdown":
		if buffer.Len() == 0 {
			result, err = output.toMarkdownWithError()
		} else {
			result, err = output.bufferToMarkdownWithError()
		}
	case "mermaid":
		result, err = output.toMermaidWithError()
	case "drawio":
		err = output.toDrawIOWithError()
	case "dot":
		result, err = output.toDotWithError()
	case "yaml":
		if buffer.Len() == 0 {
			result, err = output.toYAMLWithError()
		} else {
			result = buffer.Bytes()
		}
	default:
		if buffer.Len() == 0 {
			result, err = output.toJSONWithError()
		} else {
			result = buffer.Bytes()
		}
	}

	return result, err
}

// Error-handling versions of format methods
func (output *OutputArray) toJSONWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("JSON generation failed: %v", r),
			))
		}
	}()

	jsonString, err := json.Marshal(output.GetContentsMapRaw())
	if err != nil {
		return nil, errors.NewProcessingError(
			errors.ErrTemplateRender,
			"Failed to marshal JSON",
		).Wrap(err)
	}
	return jsonString, nil
}

func (output *OutputArray) toCSVWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("CSV generation failed: %v", r),
			))
		}
	}()

	return output.toCSV(), nil
}

func (output *OutputArray) toTableWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("Table generation failed: %v", r),
			))
		}
	}()

	return output.toTable(), nil
}

func (output *OutputArray) toMarkdownWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("Markdown generation failed: %v", r),
			))
		}
	}()

	return output.toMarkdown(), nil
}

func (output *OutputArray) toHTMLWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("HTML generation failed: %v", r),
			))
		}
	}()

	return output.toHTML(), nil
}

func (output *OutputArray) toYAMLWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("YAML generation failed: %v", r),
			))
		}
	}()

	return output.toYAML(), nil
}

func (output *OutputArray) toMermaidWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("Mermaid generation failed: %v", r),
			))
		}
	}()

	return output.toMermaid(), nil
}

func (output *OutputArray) toDotWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("DOT generation failed: %v", r),
			))
		}
	}()

	return output.toDot(), nil
}

func (output *OutputArray) toDrawIOWithError() error {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("DrawIO generation failed: %v", r),
			))
		}
	}()

	drawio.CreateCSV(output.Settings.DrawIOHeader, output.Keys, output.GetContentsMap(), output.Settings.OutputFile)
	return nil
}

func (output *OutputArray) bufferToHTMLWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("HTML buffer processing failed: %v", r),
			))
		}
	}()

	return output.bufferToHTML(), nil
}

func (output *OutputArray) bufferToMarkdownWithError() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("Markdown buffer processing failed: %v", r),
			))
		}
	}()

	return output.bufferToMarkdown(), nil
}

// Adapter methods to implement validators.OutputArray interface
func (output *OutputArray) GetKeys() []string {
	return output.Keys
}

func (output *OutputArray) GetContents() []validators.OutputHolder {
	holders := make([]validators.OutputHolder, len(output.Contents))
	for i, holder := range output.Contents {
		holders[i] = &outputHolderAdapter{holder}
	}
	return holders
}

// outputHolderAdapter adapts OutputHolder to validators.OutputHolder interface
type outputHolderAdapter struct {
	holder OutputHolder
}

func (adapter *outputHolderAdapter) GetContents() map[string]interface{} {
	return adapter.holder.Contents
}
