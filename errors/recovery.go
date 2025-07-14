package errors

import (
	"fmt"
	"time"
)

// RecoveryHandler interface defines the contract for error recovery
type RecoveryHandler interface {
	Recover(err OutputError) error                             // Attempt to recover from an error
	RecoverWithContext(err OutputError, ctx interface{}) error // Recover with additional context
	CanRecover(err OutputError) bool                           // Check if an error can be recovered
	AddStrategy(strategy RecoveryStrategy)                     // Add a recovery strategy
	Strategies() []RecoveryStrategy                            // Get all recovery strategies
}

// RecoveryStrategy interface defines how specific types of errors can be recovered
type RecoveryStrategy interface {
	Apply(err OutputError, context interface{}) (interface{}, error) // Apply recovery strategy
	ApplicableFor(err OutputError) bool                              // Check if strategy applies to error
	Name() string                                                    // Strategy name for logging/debugging
}

// RetryInfo contains information about retry configuration
type RetryInfo struct {
	MaxAttempts  int               // Maximum number of retry attempts
	InitialDelay time.Duration     // Initial delay before first retry
	Calculator   BackoffCalculator // Backoff calculator for progressive delays
}

// BackoffCalculator calculates retry delays
type BackoffCalculator interface {
	Calculate(attempt int) time.Duration // Calculate delay for specific attempt number
}

// DefaultRecoveryHandler is the main implementation of RecoveryHandler
type DefaultRecoveryHandler struct {
	strategies []RecoveryStrategy
}

// NewDefaultRecoveryHandler creates a new DefaultRecoveryHandler
func NewDefaultRecoveryHandler() *DefaultRecoveryHandler {
	return &DefaultRecoveryHandler{
		strategies: make([]RecoveryStrategy, 0),
	}
}

// Recover attempts to recover from an error using available strategies
func (h *DefaultRecoveryHandler) Recover(err OutputError) error {
	return h.RecoverWithContext(err, nil)
}

// RecoverWithContext attempts to recover from an error with additional context
func (h *DefaultRecoveryHandler) RecoverWithContext(err OutputError, ctx interface{}) error {
	if err == nil {
		return nil
	}

	// Don't attempt recovery for fatal errors
	if err.Severity() == SeverityFatal {
		return err
	}

	// Try each applicable strategy
	for _, strategy := range h.strategies {
		if strategy.ApplicableFor(err) {
			_, applyErr := strategy.Apply(err, ctx)
			if applyErr == nil {
				// Recovery succeeded
				return nil
			}
		}
	}

	// No strategy could recover the error
	return err
}

// CanRecover checks if any strategy can recover from the error
func (h *DefaultRecoveryHandler) CanRecover(err OutputError) bool {
	if err == nil {
		return true
	}

	// Fatal errors cannot be recovered
	if err.Severity() == SeverityFatal {
		return false
	}

	// Check if any strategy is applicable
	for _, strategy := range h.strategies {
		if strategy.ApplicableFor(err) {
			return true
		}
	}

	return false
}

// AddStrategy adds a recovery strategy to the handler
func (h *DefaultRecoveryHandler) AddStrategy(strategy RecoveryStrategy) {
	if strategy != nil {
		h.strategies = append(h.strategies, strategy)
	}
}

// Strategies returns all recovery strategies
func (h *DefaultRecoveryHandler) Strategies() []RecoveryStrategy {
	// Return a copy to prevent external modification
	strategies := make([]RecoveryStrategy, len(h.strategies))
	copy(strategies, h.strategies)
	return strategies
}

// FormatFallbackStrategy implements fallback to simpler output formats
type FormatFallbackStrategy struct {
	fallbackChain []string
}

// NewFormatFallbackStrategy creates a new format fallback strategy
func NewFormatFallbackStrategy(fallbackChain []string) *FormatFallbackStrategy {
	return &FormatFallbackStrategy{
		fallbackChain: fallbackChain,
	}
}

// Apply implements the format fallback logic
func (s *FormatFallbackStrategy) Apply(err OutputError, context interface{}) (interface{}, error) {
	settings, ok := context.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid context for format fallback: expected map[string]interface{}")
	}

	currentFormat, exists := settings["OutputFormat"]
	if !exists {
		return nil, fmt.Errorf("OutputFormat not found in context")
	}

	currentFormatStr, ok := currentFormat.(string)
	if !ok {
		return nil, fmt.Errorf("OutputFormat must be string")
	}

	// Find current format in fallback chain
	currentIdx := -1
	for i, format := range s.fallbackChain {
		if format == currentFormatStr {
			currentIdx = i
			break
		}
	}

	// If current format is not in chain or is the last format, can't fallback
	if currentIdx < 0 || currentIdx >= len(s.fallbackChain)-1 {
		return nil, fmt.Errorf("no fallback format available for %s", currentFormatStr)
	}

	// Update to next format in chain
	newSettings := make(map[string]interface{})
	for k, v := range settings {
		newSettings[k] = v
	}
	newSettings["OutputFormat"] = s.fallbackChain[currentIdx+1]

	return newSettings, nil
}

// ApplicableFor checks if this strategy applies to format-related errors
func (s *FormatFallbackStrategy) ApplicableFor(err OutputError) bool {
	switch err.Code() {
	case ErrInvalidFormat, ErrTemplateRender:
		return true
	default:
		return false
	}
}

// Name returns the strategy name
func (s *FormatFallbackStrategy) Name() string {
	return "FormatFallback"
}

// DefaultValueStrategy provides default values for missing data
type DefaultValueStrategy struct {
	defaults map[string]interface{}
}

// NewDefaultValueStrategy creates a new default value strategy
func NewDefaultValueStrategy(defaults map[string]interface{}) *DefaultValueStrategy {
	return &DefaultValueStrategy{
		defaults: defaults,
	}
}

// Apply implements the default value logic
func (s *DefaultValueStrategy) Apply(err OutputError, context interface{}) (interface{}, error) {
	contextMap, ok := context.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid context for default value strategy: expected map[string]interface{}")
	}

	missingField, exists := contextMap["MissingField"]
	if !exists {
		return nil, fmt.Errorf("MissingField not specified in context")
	}

	fieldName, ok := missingField.(string)
	if !ok {
		return nil, fmt.Errorf("MissingField must be string")
	}

	defaultValue, hasDefault := s.defaults[fieldName]
	if !hasDefault {
		return nil, fmt.Errorf("no default value available for field %s", fieldName)
	}

	data, exists := contextMap["Data"]
	if !exists {
		return nil, fmt.Errorf("Data not found in context")
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Data must be map[string]interface{}")
	}

	// Create updated data with default value
	updatedData := make(map[string]interface{})
	for k, v := range dataMap {
		updatedData[k] = v
	}
	updatedData[fieldName] = defaultValue

	return updatedData, nil
}

// ApplicableFor checks if this strategy applies to missing data errors
func (s *DefaultValueStrategy) ApplicableFor(err OutputError) bool {
	switch err.Code() {
	case ErrMissingRequired, ErrMissingColumn:
		return true
	default:
		return false
	}
}

// Name returns the strategy name
func (s *DefaultValueStrategy) Name() string {
	return "DefaultValue"
}

// RetryStrategy handles retryable errors with exponential backoff
type RetryStrategy struct {
	maxAttempts  int
	initialDelay time.Duration
	retryableFn  func(error) bool
}

// NewRetryStrategy creates a new retry strategy
func NewRetryStrategy(maxAttempts int, initialDelay time.Duration, retryableFn func(error) bool) *RetryStrategy {
	if retryableFn == nil {
		retryableFn = IsTransient
	}

	return &RetryStrategy{
		maxAttempts:  maxAttempts,
		initialDelay: initialDelay,
		retryableFn:  retryableFn,
	}
}

// Apply creates retry information for retryable errors
func (s *RetryStrategy) Apply(err OutputError, context interface{}) (interface{}, error) {
	if !s.retryableFn(err) {
		return nil, fmt.Errorf("error is not retryable")
	}

	return RetryInfo{
		MaxAttempts:  s.maxAttempts,
		InitialDelay: s.initialDelay,
		Calculator:   NewExponentialBackoffCalculator(s.initialDelay, 2.0, 30*time.Second),
	}, nil
}

// ApplicableFor checks if this strategy applies to retryable errors
func (s *RetryStrategy) ApplicableFor(err OutputError) bool {
	return s.retryableFn(err)
}

// Name returns the strategy name
func (s *RetryStrategy) Name() string {
	return "Retry"
}

// ExponentialBackoffCalculator implements exponential backoff with jitter
type ExponentialBackoffCalculator struct {
	initialDelay time.Duration
	multiplier   float64
	maxDelay     time.Duration
}

// NewExponentialBackoffCalculator creates a new exponential backoff calculator
func NewExponentialBackoffCalculator(initialDelay time.Duration, multiplier float64, maxDelay time.Duration) *ExponentialBackoffCalculator {
	return &ExponentialBackoffCalculator{
		initialDelay: initialDelay,
		multiplier:   multiplier,
		maxDelay:     maxDelay,
	}
}

// Calculate computes the delay for a specific attempt number
func (c *ExponentialBackoffCalculator) Calculate(attempt int) time.Duration {
	if attempt <= 0 {
		return c.initialDelay
	}

	// Calculate exponential backoff: initialDelay * multiplier^(attempt-1)
	delay := c.initialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * c.multiplier)
		if delay >= c.maxDelay {
			return c.maxDelay
		}
	}

	return delay
}

// ChainedRecoveryHandler chains multiple recovery handlers
type ChainedRecoveryHandler struct {
	handlers []RecoveryHandler
}

// NewChainedRecoveryHandler creates a new chained recovery handler
func NewChainedRecoveryHandler(handlers ...RecoveryHandler) *ChainedRecoveryHandler {
	return &ChainedRecoveryHandler{
		handlers: handlers,
	}
}

// Recover attempts recovery using each handler in sequence
func (h *ChainedRecoveryHandler) Recover(err OutputError) error {
	return h.RecoverWithContext(err, nil)
}

// RecoverWithContext attempts recovery using each handler in sequence with context
func (h *ChainedRecoveryHandler) RecoverWithContext(err OutputError, ctx interface{}) error {
	if err == nil {
		return nil
	}

	for _, handler := range h.handlers {
		if handler.CanRecover(err) {
			if recoveryErr := handler.RecoverWithContext(err, ctx); recoveryErr == nil {
				return nil // Recovery succeeded
			}
		}
	}

	return err // No handler could recover
}

// CanRecover checks if any handler can recover the error
func (h *ChainedRecoveryHandler) CanRecover(err OutputError) bool {
	for _, handler := range h.handlers {
		if handler.CanRecover(err) {
			return true
		}
	}
	return false
}

// AddStrategy adds a strategy to the first handler (or creates a new one)
func (h *ChainedRecoveryHandler) AddStrategy(strategy RecoveryStrategy) {
	if len(h.handlers) == 0 {
		h.handlers = append(h.handlers, NewDefaultRecoveryHandler())
	}
	h.handlers[0].AddStrategy(strategy)
}

// Strategies returns all strategies from all handlers
func (h *ChainedRecoveryHandler) Strategies() []RecoveryStrategy {
	var allStrategies []RecoveryStrategy
	for _, handler := range h.handlers {
		allStrategies = append(allStrategies, handler.Strategies()...)
	}
	return allStrategies
}
