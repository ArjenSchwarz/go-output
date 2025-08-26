package output

import (
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// DebugLevel represents the level of debug output
type DebugLevel int

const (
	// DebugOff disables all debug output
	DebugOff DebugLevel = iota
	// DebugError shows only errors and panics
	DebugError
	// DebugWarn shows warnings, errors, and panics
	DebugWarn
	// DebugInfo shows general information, warnings, errors, and panics
	DebugInfo
	// DebugTrace shows detailed trace information including all operations
	DebugTrace
)

// String returns the string representation of the debug level
func (dl DebugLevel) String() string {
	switch dl {
	case DebugOff:
		return "OFF"
	case DebugError:
		return "ERROR"
	case DebugWarn:
		return "WARN"
	case DebugInfo:
		return "INFO"
	case DebugTrace:
		return "TRACE"
	default:
		return "UNKNOWN"
	}
}

// DebugTracer provides debug tracing capabilities for the rendering pipeline
type DebugTracer struct {
	mu       sync.RWMutex
	level    DebugLevel
	output   io.Writer
	logger   *log.Logger
	enabled  bool
	contexts map[string]any // Additional context information
}

// NewDebugTracer creates a new debug tracer
func NewDebugTracer(level DebugLevel, output io.Writer) *DebugTracer {
	if output == nil {
		output = os.Stderr
	}

	return &DebugTracer{
		level:    level,
		output:   output,
		logger:   log.New(output, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds),
		enabled:  level > DebugOff,
		contexts: make(map[string]any),
	}
}

// SetLevel sets the debug level
func (dt *DebugTracer) SetLevel(level DebugLevel) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.level = level
	dt.enabled = level > DebugOff
}

// GetLevel returns the current debug level
func (dt *DebugTracer) GetLevel() DebugLevel {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	return dt.level
}

// IsEnabled returns whether debug tracing is enabled
func (dt *DebugTracer) IsEnabled() bool {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	return dt.enabled
}

// AddContext adds context information to the tracer
func (dt *DebugTracer) AddContext(key string, value any) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.contexts[key] = value
}

// RemoveContext removes context information from the tracer
func (dt *DebugTracer) RemoveContext(key string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	delete(dt.contexts, key)
}

// ClearContext clears all context information
func (dt *DebugTracer) ClearContext() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.contexts = make(map[string]any)
}

// shouldLog checks if a message at the given level should be logged
func (dt *DebugTracer) shouldLog(level DebugLevel) bool {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	return dt.enabled && level <= dt.level
}

// formatMessage formats a debug message with context and caller information
func (dt *DebugTracer) formatMessage(level DebugLevel, operation string, message string, args ...any) string {
	dt.mu.RLock()
	contexts := make(map[string]any, len(dt.contexts))
	maps.Copy(contexts, dt.contexts)
	dt.mu.RUnlock()

	// Get caller information
	_, file, line, ok := runtime.Caller(3) // Skip formatMessage, log method, and calling method
	caller := "unknown"
	if ok {
		// Extract just the filename from the full path
		parts := strings.Split(file, "/")
		if len(parts) > 0 {
			caller = fmt.Sprintf("%s:%d", parts[len(parts)-1], line)
		}
	}

	// Format the message
	formattedMessage := fmt.Sprintf(message, args...)

	// Build the complete message
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s]", level.String()))
	parts = append(parts, fmt.Sprintf("[%s]", caller))

	if operation != "" {
		parts = append(parts, fmt.Sprintf("[%s]", operation))
	}

	// Add context information
	if len(contexts) > 0 {
		var contextParts []string
		for key, value := range contexts {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
		}
		parts = append(parts, fmt.Sprintf("[%s]", strings.Join(contextParts, ", ")))
	}

	parts = append(parts, formattedMessage)

	return strings.Join(parts, " ")
}

// Trace logs a trace-level message
func (dt *DebugTracer) Trace(operation string, message string, args ...any) {
	if dt.shouldLog(DebugTrace) {
		formatted := dt.formatMessage(DebugTrace, operation, message, args...)
		dt.logger.Print(formatted)
	}
}

// Info logs an info-level message
func (dt *DebugTracer) Info(operation string, message string, args ...any) {
	if dt.shouldLog(DebugInfo) {
		formatted := dt.formatMessage(DebugInfo, operation, message, args...)
		dt.logger.Print(formatted)
	}
}

// Warn logs a warning-level message
func (dt *DebugTracer) Warn(operation string, message string, args ...any) {
	if dt.shouldLog(DebugWarn) {
		formatted := dt.formatMessage(DebugWarn, operation, message, args...)
		dt.logger.Print(formatted)
	}
}

// Error logs an error-level message
func (dt *DebugTracer) Error(operation string, message string, args ...any) {
	if dt.shouldLog(DebugError) {
		formatted := dt.formatMessage(DebugError, operation, message, args...)
		dt.logger.Print(formatted)
	}
}

// TraceOperation traces an operation with start and end messages
func (dt *DebugTracer) TraceOperation(operation string, fn func() error) error {
	if !dt.shouldLog(DebugTrace) {
		return fn()
	}

	start := time.Now()
	dt.Trace(operation, "starting operation")

	err := fn()
	duration := time.Since(start)

	if err != nil {
		dt.Error(operation, "operation failed after %v: %v", duration, err)
	} else {
		dt.Trace(operation, "operation completed successfully in %v", duration)
	}

	return err
}

// TraceOperationWithResult traces an operation with start, end, and result information
func (dt *DebugTracer) TraceOperationWithResult(operation string, fn func() (any, error)) (any, error) {
	if !dt.shouldLog(DebugTrace) {
		return fn()
	}

	start := time.Now()
	dt.Trace(operation, "starting operation")

	result, err := fn()
	duration := time.Since(start)

	if err != nil {
		dt.Error(operation, "operation failed after %v: %v", duration, err)
	} else {
		dt.Trace(operation, "operation completed successfully in %v with result type %T", duration, result)
	}

	return result, err
}

// PanicError represents an error that was recovered from a panic
type PanicError struct {
	Operation string // The operation that panicked
	Value     any    // The panic value
	Stack     []byte // The stack trace
	Cause     error  // Optional underlying error if the panic value was an error
}

// Error returns the error message
func (pe *PanicError) Error() string {
	if pe.Operation != "" {
		return fmt.Sprintf("panic in operation %q: %v", pe.Operation, pe.Value)
	}
	return fmt.Sprintf("panic: %v", pe.Value)
}

// Unwrap returns the underlying error if the panic value was an error
func (pe *PanicError) Unwrap() error {
	return pe.Cause
}

// StackTrace returns the stack trace as a string
func (pe *PanicError) StackTrace() string {
	return string(pe.Stack)
}

// NewPanicError creates a new panic error
func NewPanicError(operation string, panicValue any) *PanicError {
	pe := &PanicError{
		Operation: operation,
		Value:     panicValue,
		Stack:     make([]byte, 4096),
	}

	// Capture stack trace
	n := runtime.Stack(pe.Stack, false)
	pe.Stack = pe.Stack[:n]

	// Check if the panic value is an error
	if err, ok := panicValue.(error); ok {
		pe.Cause = err
	}

	return pe
}

// SafeExecute executes a function with panic recovery
func SafeExecute(operation string, fn func() error) error {
	return SafeExecuteWithTracer(nil, operation, fn)
}

// SafeExecuteWithTracer executes a function with panic recovery and optional debug tracing
func SafeExecuteWithTracer(tracer *DebugTracer, operation string, fn func() error) (finalErr error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr := NewPanicError(operation, r)

			if tracer != nil {
				tracer.Error(operation, "panic recovered: %v\nStack trace:\n%s", r, panicErr.StackTrace())
			}

			finalErr = panicErr
		}
	}()

	if tracer != nil {
		return tracer.TraceOperation(operation, fn)
	}

	return fn()
}

// SafeExecuteWithResult executes a function with panic recovery and returns both result and error
func SafeExecuteWithResult(operation string, fn func() (any, error)) (any, error) {
	return SafeExecuteWithResultAndTracer(nil, operation, fn)
}

// SafeExecuteWithResultAndTracer executes a function with panic recovery, result return, and optional debug tracing
func SafeExecuteWithResultAndTracer(tracer *DebugTracer, operation string, fn func() (any, error)) (result any, finalErr error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr := NewPanicError(operation, r)

			if tracer != nil {
				tracer.Error(operation, "panic recovered: %v\nStack trace:\n%s", r, panicErr.StackTrace())
			}

			result = nil
			finalErr = panicErr
		}
	}()

	if tracer != nil {
		return tracer.TraceOperationWithResult(operation, fn)
	}

	return fn()
}

// IsPanic checks if an error represents a recovered panic
func IsPanic(err error) bool {
	if err == nil {
		return false
	}

	var panicErr *PanicError
	return AsError(err, &panicErr)
}

// GetPanicValue extracts the panic value from a PanicError
func GetPanicValue(err error) any {
	if err == nil {
		return nil
	}

	var panicErr *PanicError
	if AsError(err, &panicErr) {
		return panicErr.Value
	}

	return nil
}

// GetStackTrace extracts the stack trace from a PanicError
func GetStackTrace(err error) string {
	if err == nil {
		return ""
	}

	var panicErr *PanicError
	if AsError(err, &panicErr) {
		return panicErr.StackTrace()
	}

	return ""
}

// Global debug tracer instance
var globalTracer *DebugTracer
var globalTracerMu sync.RWMutex

// SetGlobalDebugTracer sets the global debug tracer
func SetGlobalDebugTracer(tracer *DebugTracer) {
	globalTracerMu.Lock()
	defer globalTracerMu.Unlock()

	globalTracer = tracer
}

// GetGlobalDebugTracer returns the global debug tracer
func GetGlobalDebugTracer() *DebugTracer {
	globalTracerMu.RLock()
	defer globalTracerMu.RUnlock()

	return globalTracer
}

// EnableGlobalDebugTracing enables global debug tracing at the specified level
func EnableGlobalDebugTracing(level DebugLevel, output io.Writer) {
	tracer := NewDebugTracer(level, output)
	SetGlobalDebugTracer(tracer)
}

// DisableGlobalDebugTracing disables global debug tracing
func DisableGlobalDebugTracing() {
	SetGlobalDebugTracer(nil)
}

// GlobalTrace logs a trace message using the global tracer
func GlobalTrace(operation string, message string, args ...any) {
	if tracer := GetGlobalDebugTracer(); tracer != nil {
		tracer.Trace(operation, message, args...)
	}
}

// GlobalInfo logs an info message using the global tracer
func GlobalInfo(operation string, message string, args ...any) {
	if tracer := GetGlobalDebugTracer(); tracer != nil {
		tracer.Info(operation, message, args...)
	}
}

// GlobalWarn logs a warning message using the global tracer
func GlobalWarn(operation string, message string, args ...any) {
	if tracer := GetGlobalDebugTracer(); tracer != nil {
		tracer.Warn(operation, message, args...)
	}
}

// GlobalError logs an error message using the global tracer
func GlobalError(operation string, message string, args ...any) {
	if tracer := GetGlobalDebugTracer(); tracer != nil {
		tracer.Error(operation, message, args...)
	}
}
