package format

import (
	"fmt"
	"math"
	"time"
)

// RecoveryHandler defines the interface for handling error recovery
type RecoveryHandler interface {
	// Recover attempts to recover from an error using configured strategies
	Recover(err OutputError) error
	// CanRecover returns true if the error can be recovered from
	CanRecover(err OutputError) bool
	// AddStrategy adds a recovery strategy to the handler
	AddStrategy(strategy RecoveryStrategy)
	// GetStrategies returns all configured recovery strategies
	GetStrategies() []RecoveryStrategy
}

// RecoveryStrategy defines the interface for individual recovery strategies
type RecoveryStrategy interface {
	// Apply attempts to apply the recovery strategy to the given error and context
	Apply(err OutputError, context any) (any, error)
	// ApplicableFor returns true if this strategy can handle the given error
	ApplicableFor(err OutputError) bool
	// Name returns a human-readable name for this strategy
	Name() string
	// Priority returns the priority of this strategy (lower numbers = higher priority)
	Priority() int
}

// RecoveryResult represents the result of a recovery attempt
type RecoveryResult struct {
	Success      bool   // Whether recovery was successful
	Strategy     string // Name of the strategy that was applied
	ModifiedData any    // The modified data after recovery
	Message      string // Human-readable message about the recovery
	Error        error  // Any error that occurred during recovery
}

// DefaultRecoveryHandler provides the default implementation of RecoveryHandler
type DefaultRecoveryHandler struct {
	strategies []RecoveryStrategy
}

// NewDefaultRecoveryHandler creates a new DefaultRecoveryHandler
func NewDefaultRecoveryHandler() *DefaultRecoveryHandler {
	return &DefaultRecoveryHandler{
		strategies: make([]RecoveryStrategy, 0),
	}
}

// NewRecoveryHandlerWithStrategies creates a new DefaultRecoveryHandler with predefined strategies
func NewRecoveryHandlerWithStrategies(strategies ...RecoveryStrategy) *DefaultRecoveryHandler {
	handler := NewDefaultRecoveryHandler()
	for _, strategy := range strategies {
		handler.AddStrategy(strategy)
	}
	return handler
}

// Recover attempts to recover from an error using configured strategies
func (h *DefaultRecoveryHandler) Recover(err OutputError) error {
	if !h.CanRecover(err) {
		return err
	}

	// Try strategies in priority order
	for _, strategy := range h.getSortedStrategies() {
		if strategy.ApplicableFor(err) {
			// For now, we pass nil as context - in a full implementation,
			// this would be the actual context (OutputArray, OutputSettings, etc.)
			if _, recoveryErr := strategy.Apply(err, nil); recoveryErr == nil {
				// Recovery successful
				return nil
			}
		}
	}

	// No strategy could recover from the error
	return err
}

// CanRecover returns true if any strategy can handle the error
func (h *DefaultRecoveryHandler) CanRecover(err OutputError) bool {
	for _, strategy := range h.strategies {
		if strategy.ApplicableFor(err) {
			return true
		}
	}
	return false
}

// AddStrategy adds a recovery strategy to the handler
func (h *DefaultRecoveryHandler) AddStrategy(strategy RecoveryStrategy) {
	h.strategies = append(h.strategies, strategy)
}

// GetStrategies returns all configured recovery strategies
func (h *DefaultRecoveryHandler) GetStrategies() []RecoveryStrategy {
	return h.strategies
}

// getSortedStrategies returns strategies sorted by priority (lower numbers first)
func (h *DefaultRecoveryHandler) getSortedStrategies() []RecoveryStrategy {
	// Create a copy to avoid modifying the original slice
	sorted := make([]RecoveryStrategy, len(h.strategies))
	copy(sorted, h.strategies)

	// Simple bubble sort by priority
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Priority() > sorted[j+1].Priority() {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// FormatFallbackStrategy implements format fallback recovery
type FormatFallbackStrategy struct {
	fallbackChain []string // Chain of formats to try (e.g., ["table", "csv", "json"])
	name          string
	priority      int
}

// NewFormatFallbackStrategy creates a new format fallback strategy
func NewFormatFallbackStrategy(formats ...string) *FormatFallbackStrategy {
	return &FormatFallbackStrategy{
		fallbackChain: formats,
		name:          "format-fallback",
		priority:      10, // Medium priority
	}
}

// Apply attempts to apply format fallback recovery
func (s *FormatFallbackStrategy) Apply(err OutputError, context any) (any, error) {
	// In a full implementation, this would modify the OutputSettings
	// to use the next format in the fallback chain
	settings, ok := context.(*OutputSettings)
	if !ok {
		return nil, fmt.Errorf("format fallback requires OutputSettings context")
	}

	// Find current format in the chain
	currentIdx := -1
	for i, format := range s.fallbackChain {
		if format == settings.OutputFormat {
			currentIdx = i
			break
		}
	}

	// If current format is not in chain or is the last one, can't fallback
	if currentIdx == -1 || currentIdx >= len(s.fallbackChain)-1 {
		return nil, fmt.Errorf("no fallback format available for %s", settings.OutputFormat)
	}

	// Set the next format in the chain
	nextFormat := s.fallbackChain[currentIdx+1]
	settings.OutputFormat = nextFormat

	return settings, nil
}

// ApplicableFor returns true if this strategy can handle format-related errors
func (s *FormatFallbackStrategy) ApplicableFor(err OutputError) bool {
	code := err.Code()
	return code == ErrInvalidFormat ||
		code == ErrFormatGeneration ||
		code == ErrTemplateRender ||
		code == ErrIncompatibleConfig
}

// Name returns the strategy name
func (s *FormatFallbackStrategy) Name() string {
	return s.name
}

// Priority returns the strategy priority
func (s *FormatFallbackStrategy) Priority() int {
	return s.priority
}

// DefaultValueStrategy implements default value substitution for missing data
type DefaultValueStrategy struct {
	defaults map[string]any // Map of field names to default values
	name     string
	priority int
}

// NewDefaultValueStrategy creates a new default value strategy
func NewDefaultValueStrategy(defaults map[string]any) *DefaultValueStrategy {
	return &DefaultValueStrategy{
		defaults: defaults,
		name:     "default-value",
		priority: 20, // Lower priority than format fallback
	}
}

// Apply attempts to apply default value substitution
func (s *DefaultValueStrategy) Apply(err OutputError, context any) (any, error) {
	// In a full implementation, this would modify the data to include default values
	// For now, we'll simulate success if we have a default for the problematic field

	fieldName := err.Context().Field
	if fieldName == "" {
		return nil, fmt.Errorf("cannot apply default value strategy: no field specified in error context")
	}

	defaultValue, exists := s.defaults[fieldName]
	if !exists {
		return nil, fmt.Errorf("no default value configured for field: %s", fieldName)
	}

	// Return the default value as the recovery result
	return defaultValue, nil
}

// ApplicableFor returns true if this strategy can handle missing data errors
func (s *DefaultValueStrategy) ApplicableFor(err OutputError) bool {
	code := err.Code()
	return code == ErrMissingColumn ||
		code == ErrEmptyDataset ||
		code == ErrMalformedData
}

// Name returns the strategy name
func (s *DefaultValueStrategy) Name() string {
	return s.name
}

// Priority returns the strategy priority
func (s *DefaultValueStrategy) Priority() int {
	return s.priority
}

// BackoffStrategy defines different backoff strategies for retries
type BackoffStrategy interface {
	// NextDelay calculates the delay for the next retry attempt
	NextDelay(attempt int) time.Duration
	// MaxAttempts returns the maximum number of retry attempts
	MaxAttempts() int
}

// ExponentialBackoff implements exponential backoff with jitter
type ExponentialBackoff struct {
	baseDelay   time.Duration
	maxDelay    time.Duration
	maxAttempts int
	multiplier  float64
}

// NewExponentialBackoff creates a new exponential backoff strategy
func NewExponentialBackoff(baseDelay, maxDelay time.Duration, maxAttempts int) *ExponentialBackoff {
	return &ExponentialBackoff{
		baseDelay:   baseDelay,
		maxDelay:    maxDelay,
		maxAttempts: maxAttempts,
		multiplier:  2.0, // Double the delay each time
	}
}

// NextDelay calculates the delay for the next retry attempt
func (b *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return b.baseDelay
	}

	// Calculate exponential delay: baseDelay * multiplier^attempt
	delay := time.Duration(float64(b.baseDelay) * math.Pow(b.multiplier, float64(attempt)))

	// Cap at maximum delay
	if delay > b.maxDelay {
		delay = b.maxDelay
	}

	return delay
}

// MaxAttempts returns the maximum number of retry attempts
func (b *ExponentialBackoff) MaxAttempts() int {
	return b.maxAttempts
}

// RetryStrategy implements retry logic with configurable backoff for transient errors
type RetryStrategy struct {
	backoff       BackoffStrategy
	retryableFunc func(OutputError) bool // Function to determine if an error is retryable
	name          string
	priority      int
}

// NewRetryStrategy creates a new retry strategy
func NewRetryStrategy(backoff BackoffStrategy) *RetryStrategy {
	return &RetryStrategy{
		backoff:       backoff,
		retryableFunc: defaultRetryableFunc,
		name:          "retry",
		priority:      30, // Lowest priority - try other strategies first
	}
}

// NewRetryStrategyWithFunc creates a new retry strategy with a custom retryable function
func NewRetryStrategyWithFunc(backoff BackoffStrategy, retryableFunc func(OutputError) bool) *RetryStrategy {
	return &RetryStrategy{
		backoff:       backoff,
		retryableFunc: retryableFunc,
		name:          "retry",
		priority:      30,
	}
}

// Apply attempts to apply retry recovery (simulation - actual retry would happen at a higher level)
func (s *RetryStrategy) Apply(err OutputError, context any) (any, error) {
	// In a real implementation, this would coordinate with the calling code
	// to retry the operation. For now, we'll simulate the retry logic.

	if !s.isRetryable(err) {
		return nil, fmt.Errorf("error is not retryable: %s", err.Code())
	}

	// Simulate retry attempt tracking
	retryInfo := map[string]any{
		"strategy":     s.name,
		"max_attempts": s.backoff.MaxAttempts(),
		"base_delay":   s.backoff.NextDelay(0),
		"retryable":    true,
	}

	return retryInfo, nil
}

// ApplicableFor returns true if this strategy can handle retryable errors
func (s *RetryStrategy) ApplicableFor(err OutputError) bool {
	return s.isRetryable(err)
}

// isRetryable determines if an error is retryable
func (s *RetryStrategy) isRetryable(err OutputError) bool {
	if s.retryableFunc != nil {
		return s.retryableFunc(err)
	}
	return defaultRetryableFunc(err)
}

// Name returns the strategy name
func (s *RetryStrategy) Name() string {
	return s.name
}

// Priority returns the strategy priority
func (s *RetryStrategy) Priority() int {
	return s.priority
}

// defaultRetryableFunc is the default function for determining if an error is retryable
func defaultRetryableFunc(err OutputError) bool {
	code := err.Code()

	// Network and transient errors are typically retryable
	retryableCodes := []ErrorCode{
		ErrNetworkTimeout,
		ErrServiceUnavailable,
		ErrS3Upload,  // S3 uploads can be transient
		ErrFileWrite, // File writes might fail due to temporary issues
	}

	for _, retryableCode := range retryableCodes {
		if code == retryableCode {
			return true
		}
	}

	// Also check if it's a ProcessingError that's marked as retryable
	if procErr, ok := err.(ProcessingError); ok {
		return procErr.Retryable()
	}

	return false
}

// CompositeRecoveryStrategy combines multiple recovery strategies
type CompositeRecoveryStrategy struct {
	strategies []RecoveryStrategy
	name       string
	priority   int
}

// NewCompositeRecoveryStrategy creates a new composite recovery strategy
func NewCompositeRecoveryStrategy(name string, strategies ...RecoveryStrategy) *CompositeRecoveryStrategy {
	return &CompositeRecoveryStrategy{
		strategies: strategies,
		name:       name,
		priority:   5, // High priority for composite strategies
	}
}

// Apply attempts to apply all contained strategies in order
func (s *CompositeRecoveryStrategy) Apply(err OutputError, context any) (any, error) {
	var lastResult any
	var lastError error

	for _, strategy := range s.strategies {
		if strategy.ApplicableFor(err) {
			result, strategyErr := strategy.Apply(err, context)
			if strategyErr == nil {
				// Strategy succeeded, return its result
				return result, nil
			}
			// Keep track of the last attempt
			lastResult = result
			lastError = strategyErr
		}
	}

	// No strategy succeeded
	if lastError != nil {
		return lastResult, lastError
	}

	return nil, fmt.Errorf("no applicable recovery strategy found")
}

// ApplicableFor returns true if any contained strategy can handle the error
func (s *CompositeRecoveryStrategy) ApplicableFor(err OutputError) bool {
	for _, strategy := range s.strategies {
		if strategy.ApplicableFor(err) {
			return true
		}
	}
	return false
}

// Name returns the strategy name
func (s *CompositeRecoveryStrategy) Name() string {
	return s.name
}

// Priority returns the strategy priority
func (s *CompositeRecoveryStrategy) Priority() int {
	return s.priority
}

// RecoveryContext provides context information for recovery operations
type RecoveryContext struct {
	OriginalError OutputError
	Attempt       int
	Settings      *OutputSettings
	Data          any
	Metadata      map[string]any
}

// ContextualRecoveryStrategy is a recovery strategy that receives context
type ContextualRecoveryStrategy interface {
	RecoveryStrategy
	ApplyWithContext(ctx RecoveryContext) (any, error)
}

// contextualRecoveryAdapter adapts a regular recovery strategy to work with context
type contextualRecoveryAdapter struct {
	strategy RecoveryStrategy
}

// ApplyWithContext implements ContextualRecoveryStrategy interface
func (a *contextualRecoveryAdapter) ApplyWithContext(ctx RecoveryContext) (any, error) {
	return a.strategy.Apply(ctx.OriginalError, ctx.Settings)
}

// Apply implements the RecoveryStrategy interface
func (a *contextualRecoveryAdapter) Apply(err OutputError, context any) (any, error) {
	return a.strategy.Apply(err, context)
}

// ApplicableFor implements the RecoveryStrategy interface
func (a *contextualRecoveryAdapter) ApplicableFor(err OutputError) bool {
	return a.strategy.ApplicableFor(err)
}

// Name implements the RecoveryStrategy interface
func (a *contextualRecoveryAdapter) Name() string {
	return a.strategy.Name()
}

// Priority implements the RecoveryStrategy interface
func (a *contextualRecoveryAdapter) Priority() int {
	return a.strategy.Priority()
}

// AsContextualRecovery wraps a regular recovery strategy to work with context
func AsContextualRecovery(strategy RecoveryStrategy) ContextualRecoveryStrategy {
	if contextual, ok := strategy.(ContextualRecoveryStrategy); ok {
		return contextual
	}
	return &contextualRecoveryAdapter{strategy: strategy}
}
