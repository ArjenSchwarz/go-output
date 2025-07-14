package format

import (
	"testing"
	"time"
)

func TestSimpleRecovery(t *testing.T) {
	// Test basic recovery handler creation
	handler := NewDefaultRecoveryHandler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// Test format fallback strategy
	fallbackStrategy := NewFormatFallbackStrategy("table", "csv", "json")
	if fallbackStrategy.Name() != "format-fallback" {
		t.Errorf("expected name 'format-fallback', got %s", fallbackStrategy.Name())
	}

	// Test default value strategy
	defaults := map[string]any{
		"name":   "Unknown",
		"status": "N/A",
	}
	defaultStrategy := NewDefaultValueStrategy(defaults)
	if defaultStrategy.Name() != "default-value" {
		t.Errorf("expected name 'default-value', got %s", defaultStrategy.Name())
	}

	// Test retry strategy
	backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 3)
	retryStrategy := NewRetryStrategy(backoff)
	if retryStrategy.Name() != "retry" {
		t.Errorf("expected name 'retry', got %s", retryStrategy.Name())
	}

	// Add strategies to handler
	handler.AddStrategy(fallbackStrategy)
	handler.AddStrategy(defaultStrategy)
	handler.AddStrategy(retryStrategy)

	if len(handler.GetStrategies()) != 3 {
		t.Errorf("expected 3 strategies, got %d", len(handler.GetStrategies()))
	}

	// Test error recovery capability
	formatErr := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")
	if !handler.CanRecover(formatErr) {
		t.Error("expected handler to be able to recover from format error")
	}

	columnErr := NewOutputError(ErrMissingColumn, SeverityError, "missing column")
	if !handler.CanRecover(columnErr) {
		t.Error("expected handler to be able to recover from missing column error")
	}

	networkErr := NewOutputError(ErrNetworkTimeout, SeverityError, "network timeout")
	if !handler.CanRecover(networkErr) {
		t.Error("expected handler to be able to recover from network error")
	}

	t.Log("Recovery framework test completed successfully!")
}
