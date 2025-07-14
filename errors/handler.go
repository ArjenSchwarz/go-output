package errors

import (
	"sync"
)

// ErrorMode represents different error handling modes
type ErrorMode int

const (
	ErrorModeStrict      ErrorMode = iota // Fail fast on any error
	ErrorModeLenient                      // Collect errors and continue where possible
	ErrorModeInteractive                  // Prompt user for resolution
)

// String returns the string representation of ErrorMode
func (m ErrorMode) String() string {
	switch m {
	case ErrorModeStrict:
		return "Strict"
	case ErrorModeLenient:
		return "Lenient"
	case ErrorModeInteractive:
		return "Interactive"
	default:
		return "Unknown"
	}
}

// ErrorHandler interface defines the contract for handling errors in different modes
type ErrorHandler interface {
	HandleError(err error) error           // Process an error according to the current mode
	SetMode(mode ErrorMode)                // Set the error handling mode
	Mode() ErrorMode                       // Get the current error handling mode
	Errors() []error                       // Get all collected errors
	GetSummary() ErrorSummary              // Get a summary of collected errors
	Clear()                                // Clear all collected errors
	SetWarningHandler(handler func(error)) // Set callback for warnings
	SetInfoHandler(handler func(error))    // Set callback for info messages
}

// ErrorSummary provides statistics and categorization of collected errors
type ErrorSummary struct {
	TotalErrors int                   // Total number of errors collected
	BySeverity  map[ErrorSeverity]int // Count of errors by severity level
	ByCategory  map[ErrorCode]int     // Count of errors by error code
	Suggestions []string              // Aggregated suggestions from all errors
}

// HasErrors returns true if there are any errors in the summary
func (s *ErrorSummary) HasErrors() bool {
	return s.TotalErrors > 0
}

// HasFatalErrors returns true if there are any fatal errors in the summary
func (s *ErrorSummary) HasFatalErrors() bool {
	return s.BySeverity[SeverityFatal] > 0
}

// GetHighestSeverity returns the highest severity level among collected errors
func (s *ErrorSummary) GetHighestSeverity() ErrorSeverity {
	if s.BySeverity[SeverityFatal] > 0 {
		return SeverityFatal
	}
	if s.BySeverity[SeverityError] > 0 {
		return SeverityError
	}
	if s.BySeverity[SeverityWarning] > 0 {
		return SeverityWarning
	}
	return SeverityInfo
}

// DefaultErrorHandler is the main implementation of ErrorHandler
type DefaultErrorHandler struct {
	mode           ErrorMode
	errors         []error
	warningHandler func(error)
	infoHandler    func(error)
	mutex          sync.RWMutex // Protects concurrent access to errors slice
}

// NewDefaultErrorHandler creates a new DefaultErrorHandler with strict mode
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{
		mode:   ErrorModeStrict,
		errors: make([]error, 0),
	}
}

// HandleError processes an error according to the current mode
func (h *DefaultErrorHandler) HandleError(err error) error {
	if err == nil {
		return nil
	}

	// Convert regular errors to OutputError
	var outputErr OutputError
	if oe, ok := err.(OutputError); ok {
		outputErr = oe
	} else {
		outputErr = WrapError(err)
	}

	switch h.mode {
	case ErrorModeStrict:
		return h.handleStrict(outputErr)
	case ErrorModeLenient:
		return h.handleLenient(outputErr)
	case ErrorModeInteractive:
		return h.handleInteractive(outputErr)
	default:
		return h.handleStrict(outputErr)
	}
}

// handleStrict implements strict mode error handling
func (h *DefaultErrorHandler) handleStrict(err OutputError) error {
	severity := err.Severity()

	// Handle warnings and info messages with callbacks
	if severity == SeverityWarning && h.warningHandler != nil {
		h.warningHandler(err)
	}
	if severity == SeverityInfo && h.infoHandler != nil {
		h.infoHandler(err)
	}

	// Return errors and fatal errors immediately, let warnings and info pass through
	if severity <= SeverityError {
		return err
	}

	return nil
}

// handleLenient implements lenient mode error handling
func (h *DefaultErrorHandler) handleLenient(err OutputError) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Collect all errors
	h.errors = append(h.errors, err)

	severity := err.Severity()

	// Handle warnings and info messages with callbacks
	if severity == SeverityWarning && h.warningHandler != nil {
		h.warningHandler(err)
	}
	if severity == SeverityInfo && h.infoHandler != nil {
		h.infoHandler(err)
	}

	// Only return fatal errors immediately
	if severity == SeverityFatal {
		return err
	}

	return nil
}

// handleInteractive implements interactive mode error handling
// This is a basic implementation - full interactive features are in InteractiveErrorHandler
func (h *DefaultErrorHandler) handleInteractive(err OutputError) error {
	// Default implementation falls back to lenient mode
	// For full interactive features, use InteractiveErrorHandler
	return h.handleLenient(err)
}

// SetMode sets the error handling mode
func (h *DefaultErrorHandler) SetMode(mode ErrorMode) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.mode = mode
}

// Mode returns the current error handling mode
func (h *DefaultErrorHandler) Mode() ErrorMode {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.mode
}

// Errors returns all collected errors
func (h *DefaultErrorHandler) Errors() []error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	// Return a copy to prevent external modification
	errorsCopy := make([]error, len(h.errors))
	copy(errorsCopy, h.errors)
	return errorsCopy
}

// GetSummary returns a summary of collected errors
func (h *DefaultErrorHandler) GetSummary() ErrorSummary {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	summary := ErrorSummary{
		TotalErrors: len(h.errors),
		BySeverity:  make(map[ErrorSeverity]int),
		ByCategory:  make(map[ErrorCode]int),
		Suggestions: make([]string, 0),
	}

	suggestionSet := make(map[string]bool) // To avoid duplicate suggestions

	for _, err := range h.errors {
		if outputErr, ok := err.(OutputError); ok {
			// Count by severity
			summary.BySeverity[outputErr.Severity()]++

			// Count by category
			summary.ByCategory[outputErr.Code()]++

			// Collect unique suggestions
			for _, suggestion := range outputErr.Suggestions() {
				if !suggestionSet[suggestion] {
					suggestionSet[suggestion] = true
					summary.Suggestions = append(summary.Suggestions, suggestion)
				}
			}
		} else {
			// Regular errors are treated as SeverityError
			summary.BySeverity[SeverityError]++
		}
	}

	return summary
}

// Clear removes all collected errors
func (h *DefaultErrorHandler) Clear() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.errors = make([]error, 0)
}

// SetWarningHandler sets a callback function for warning messages
func (h *DefaultErrorHandler) SetWarningHandler(handler func(error)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.warningHandler = handler
}

// SetInfoHandler sets a callback function for info messages
func (h *DefaultErrorHandler) SetInfoHandler(handler func(error)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.infoHandler = handler
}

// LegacyErrorHandler mimics the old log.Fatal behavior for backward compatibility
type LegacyErrorHandler struct{}

// NewLegacyErrorHandler creates a new LegacyErrorHandler
func NewLegacyErrorHandler() *LegacyErrorHandler {
	return &LegacyErrorHandler{}
}

// HandleError always returns the error immediately (simulating log.Fatal behavior)
func (h *LegacyErrorHandler) HandleError(err error) error {
	return err
}

// SetMode has no effect on LegacyErrorHandler (always strict)
func (h *LegacyErrorHandler) SetMode(mode ErrorMode) {
	// Intentionally do nothing - legacy handler is always strict
}

// Mode always returns ErrorModeStrict
func (h *LegacyErrorHandler) Mode() ErrorMode {
	return ErrorModeStrict
}

// Errors always returns empty slice (legacy handler doesn't collect errors)
func (h *LegacyErrorHandler) Errors() []error {
	return []error{}
}

// GetSummary always returns empty summary
func (h *LegacyErrorHandler) GetSummary() ErrorSummary {
	return ErrorSummary{
		BySeverity: make(map[ErrorSeverity]int),
		ByCategory: make(map[ErrorCode]int),
	}
}

// Clear has no effect on LegacyErrorHandler
func (h *LegacyErrorHandler) Clear() {
	// Intentionally do nothing
}

// SetWarningHandler has no effect on LegacyErrorHandler
func (h *LegacyErrorHandler) SetWarningHandler(handler func(error)) {
	// Intentionally do nothing
}

// SetInfoHandler has no effect on LegacyErrorHandler
func (h *LegacyErrorHandler) SetInfoHandler(handler func(error)) {
	// Intentionally do nothing
}

// WrapError converts a regular error to an OutputError
func WrapError(err error) OutputError {
	if err == nil {
		return nil
	}

	if outputErr, ok := err.(OutputError); ok {
		return outputErr
	}

	return NewError(ErrRetryable, err.Error()).WithSeverity(SeverityError)
}
