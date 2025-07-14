package format

import (
	"fmt"
	"testing"
	"time"
)

func TestDefaultRecoveryHandler(t *testing.T) {
	t.Run("NewDefaultRecoveryHandler", func(t *testing.T) {
		handler := NewDefaultRecoveryHandler()
		if handler == nil {
			t.Fatal("expected non-nil handler")
		}
		if len(handler.GetStrategies()) != 0 {
			t.Errorf("expected empty strategies, got %d", len(handler.GetStrategies()))
		}
	})

	t.Run("AddStrategy", func(t *testing.T) {
		handler := NewDefaultRecoveryHandler()
		strategy := NewFormatFallbackStrategy("table", "csv", "json")

		handler.AddStrategy(strategy)

		strategies := handler.GetStrategies()
		if len(strategies) != 1 {
			t.Errorf("expected 1 strategy, got %d", len(strategies))
		}
		if strategies[0].Name() != "format-fallback" {
			t.Errorf("expected format-fallback strategy, got %s", strategies[0].Name())
		}
	})

	t.Run("CanRecover", func(t *testing.T) {
		handler := NewDefaultRecoveryHandler()
		err := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")

		// Should not be able to recover without strategies
		if handler.CanRecover(err) {
			t.Error("expected false for CanRecover without strategies")
		}

		// Add applicable strategy
		strategy := NewFormatFallbackStrategy("table", "csv", "json")
		handler.AddStrategy(strategy)

		// Should be able to recover now
		if !handler.CanRecover(err) {
			t.Error("expected true for CanRecover with applicable strategy")
		}

		// Should not be able to recover from non-applicable error
		nonApplicableErr := NewOutputError(ErrMissingColumn, SeverityError, "missing column")
		if handler.CanRecover(nonApplicableErr) {
			t.Error("expected false for CanRecover with non-applicable error")
		}
	})

	t.Run("NewRecoveryHandlerWithStrategies", func(t *testing.T) {
		strategy1 := NewFormatFallbackStrategy("table", "csv")
		strategy2 := NewDefaultValueStrategy(map[string]any{"field": "default"})

		handler := NewRecoveryHandlerWithStrategies(strategy1, strategy2)

		strategies := handler.GetStrategies()
		if len(strategies) != 2 {
			t.Errorf("expected 2 strategies, got %d", len(strategies))
		}
	})
}

func TestFormatFallbackStrategy(t *testing.T) {
	t.Run("NewFormatFallbackStrategy", func(t *testing.T) {
		strategy := NewFormatFallbackStrategy("table", "csv", "json")

		if strategy.Name() != "format-fallback" {
			t.Errorf("expected name 'format-fallback', got %s", strategy.Name())
		}
		if strategy.Priority() != 10 {
			t.Errorf("expected priority 10, got %d", strategy.Priority())
		}
	})

	t.Run("ApplicableFor", func(t *testing.T) {
		strategy := NewFormatFallbackStrategy("table", "csv", "json")

		testCases := []struct {
			name       string
			errorCode  ErrorCode
			applicable bool
		}{
			{"InvalidFormat", ErrInvalidFormat, true},
			{"FormatGeneration", ErrFormatGeneration, true},
			{"TemplateRender", ErrTemplateRender, true},
			{"IncompatibleConfig", ErrIncompatibleConfig, true},
			{"MissingColumn", ErrMissingColumn, false},
			{"FileWrite", ErrFileWrite, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := NewOutputError(tc.errorCode, SeverityError, "test error")
				result := strategy.ApplicableFor(err)
				if result != tc.applicable {
					t.Errorf("expected ApplicableFor to return %v for %s, got %v",
						tc.applicable, tc.errorCode, result)
				}
			})
		}
	})

	t.Run("Apply", func(t *testing.T) {
		strategy := NewFormatFallbackStrategy("table", "csv", "json")

		t.Run("SuccessfulFallback", func(t *testing.T) {
			settings := &OutputSettings{OutputFormat: "table"}
			err := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")

			result, applyErr := strategy.Apply(err, settings)
			if applyErr != nil {
				t.Errorf("expected no error, got %v", applyErr)
			}

			modifiedSettings, ok := result.(*OutputSettings)
			if !ok {
				t.Fatal("expected OutputSettings result")
			}

			if modifiedSettings.OutputFormat != "csv" {
				t.Errorf("expected format to be 'csv', got %s", modifiedSettings.OutputFormat)
			}
		})

		t.Run("NoFallbackAvailable", func(t *testing.T) {
			settings := &OutputSettings{OutputFormat: "json"} // Last in chain
			err := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")

			_, applyErr := strategy.Apply(err, settings)
			if applyErr == nil {
				t.Error("expected error when no fallback available")
			}
		})

		t.Run("InvalidContext", func(t *testing.T) {
			err := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")

			_, applyErr := strategy.Apply(err, "invalid context")
			if applyErr == nil {
				t.Error("expected error with invalid context")
			}
		})
	})
}

func TestDefaultValueStrategy(t *testing.T) {
	t.Run("NewDefaultValueStrategy", func(t *testing.T) {
		defaults := map[string]any{
			"name":   "Unknown",
			"count":  0,
			"active": false,
		}
		strategy := NewDefaultValueStrategy(defaults)

		if strategy.Name() != "default-value" {
			t.Errorf("expected name 'default-value', got %s", strategy.Name())
		}
		if strategy.Priority() != 20 {
			t.Errorf("expected priority 20, got %d", strategy.Priority())
		}
	})

	t.Run("ApplicableFor", func(t *testing.T) {
		strategy := NewDefaultValueStrategy(map[string]any{"field": "default"})

		testCases := []struct {
			name       string
			errorCode  ErrorCode
			applicable bool
		}{
			{"MissingColumn", ErrMissingColumn, true},
			{"EmptyDataset", ErrEmptyDataset, true},
			{"MalformedData", ErrMalformedData, true},
			{"InvalidFormat", ErrInvalidFormat, false},
			{"FileWrite", ErrFileWrite, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := NewOutputError(tc.errorCode, SeverityError, "test error")
				result := strategy.ApplicableFor(err)
				if result != tc.applicable {
					t.Errorf("expected ApplicableFor to return %v for %s, got %v",
						tc.applicable, tc.errorCode, result)
				}
			})
		}
	})

	t.Run("Apply", func(t *testing.T) {
		defaults := map[string]any{
			"name":  "Unknown",
			"count": 42,
		}
		strategy := NewDefaultValueStrategy(defaults)

		t.Run("SuccessfulDefault", func(t *testing.T) {
			err := NewErrorBuilder(ErrMissingColumn, "missing column").
				WithField("name").
				Build()

			result, applyErr := strategy.Apply(err, nil)
			if applyErr != nil {
				t.Errorf("expected no error, got %v", applyErr)
			}

			if result != "Unknown" {
				t.Errorf("expected default value 'Unknown', got %v", result)
			}
		})

		t.Run("NoDefaultAvailable", func(t *testing.T) {
			err := NewErrorBuilder(ErrMissingColumn, "missing column").
				WithField("nonexistent").
				Build()

			_, applyErr := strategy.Apply(err, nil)
			if applyErr == nil {
				t.Error("expected error when no default available")
			}
		})

		t.Run("NoFieldInContext", func(t *testing.T) {
			err := NewOutputError(ErrMissingColumn, SeverityError, "missing column")

			_, applyErr := strategy.Apply(err, nil)
			if applyErr == nil {
				t.Error("expected error when no field in context")
			}
		})
	})
}

func TestExponentialBackoff(t *testing.T) {
	t.Run("NewExponentialBackoff", func(t *testing.T) {
		backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 5)

		if backoff.MaxAttempts() != 5 {
			t.Errorf("expected max attempts 5, got %d", backoff.MaxAttempts())
		}
	})

	t.Run("NextDelay", func(t *testing.T) {
		backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 5)

		testCases := []struct {
			attempt     int
			expectedMin time.Duration
			expectedMax time.Duration
		}{
			{0, 100 * time.Millisecond, 100 * time.Millisecond},
			{1, 200 * time.Millisecond, 200 * time.Millisecond},
			{2, 400 * time.Millisecond, 400 * time.Millisecond},
			{3, 800 * time.Millisecond, 800 * time.Millisecond},
			{4, 1600 * time.Millisecond, 1600 * time.Millisecond},
			{10, 5 * time.Second, 5 * time.Second}, // Should be capped at maxDelay
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("Attempt%d", tc.attempt), func(t *testing.T) {
				delay := backoff.NextDelay(tc.attempt)
				if delay < tc.expectedMin || delay > tc.expectedMax {
					t.Errorf("expected delay between %v and %v, got %v",
						tc.expectedMin, tc.expectedMax, delay)
				}
			})
		}
	})
}

func TestRetryStrategy(t *testing.T) {
	t.Run("NewRetryStrategy", func(t *testing.T) {
		backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 3)
		strategy := NewRetryStrategy(backoff)

		if strategy.Name() != "retry" {
			t.Errorf("expected name 'retry', got %s", strategy.Name())
		}
		if strategy.Priority() != 30 {
			t.Errorf("expected priority 30, got %d", strategy.Priority())
		}
	})

	t.Run("ApplicableFor", func(t *testing.T) {
		backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 3)
		strategy := NewRetryStrategy(backoff)

		testCases := []struct {
			name       string
			errorCode  ErrorCode
			applicable bool
		}{
			{"NetworkTimeout", ErrNetworkTimeout, true},
			{"ServiceUnavailable", ErrServiceUnavailable, true},
			{"S3Upload", ErrS3Upload, true},
			{"FileWrite", ErrFileWrite, true},
			{"InvalidFormat", ErrInvalidFormat, false},
			{"MissingColumn", ErrMissingColumn, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := NewOutputError(tc.errorCode, SeverityError, "test error")
				result := strategy.ApplicableFor(err)
				if result != tc.applicable {
					t.Errorf("expected ApplicableFor to return %v for %s, got %v",
						tc.applicable, tc.errorCode, result)
				}
			})
		}
	})

	t.Run("ApplicableForProcessingError", func(t *testing.T) {
		backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 3)
		strategy := NewRetryStrategy(backoff)

		retryableErr := NewProcessingError(ErrFormatGeneration, "processing failed", true)
		nonRetryableErr := NewProcessingError(ErrFormatGeneration, "processing failed", false)

		if !strategy.ApplicableFor(retryableErr) {
			t.Error("expected strategy to be applicable for retryable ProcessingError")
		}

		if strategy.ApplicableFor(nonRetryableErr) {
			t.Error("expected strategy to not be applicable for non-retryable ProcessingError")
		}
	})

	t.Run("Apply", func(t *testing.T) {
		backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 3)
		strategy := NewRetryStrategy(backoff)

		t.Run("RetryableError", func(t *testing.T) {
			err := NewOutputError(ErrNetworkTimeout, SeverityError, "network timeout")

			result, applyErr := strategy.Apply(err, nil)
			if applyErr != nil {
				t.Errorf("expected no error, got %v", applyErr)
			}

			retryInfo, ok := result.(map[string]any)
			if !ok {
				t.Fatal("expected map result")
			}

			if retryInfo["strategy"] != "retry" {
				t.Errorf("expected strategy 'retry', got %v", retryInfo["strategy"])
			}
			if retryInfo["max_attempts"] != 3 {
				t.Errorf("expected max_attempts 3, got %v", retryInfo["max_attempts"])
			}
		})

		t.Run("NonRetryableError", func(t *testing.T) {
			err := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")

			_, applyErr := strategy.Apply(err, nil)
			if applyErr == nil {
				t.Error("expected error for non-retryable error")
			}
		})
	})

	t.Run("CustomRetryableFunc", func(t *testing.T) {
		backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 3)
		customFunc := func(err OutputError) bool {
			return err.Code() == ErrInvalidFormat // Custom logic
		}
		strategy := NewRetryStrategyWithFunc(backoff, customFunc)

		err := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")
		if !strategy.ApplicableFor(err) {
			t.Error("expected custom retryable function to make ErrInvalidFormat retryable")
		}
	})
}

func TestCompositeRecoveryStrategy(t *testing.T) {
	t.Run("NewCompositeRecoveryStrategy", func(t *testing.T) {
		strategy1 := NewFormatFallbackStrategy("table", "csv")
		strategy2 := NewDefaultValueStrategy(map[string]any{"field": "default"})

		composite := NewCompositeRecoveryStrategy("test-composite", strategy1, strategy2)

		if composite.Name() != "test-composite" {
			t.Errorf("expected name 'test-composite', got %s", composite.Name())
		}
		if composite.Priority() != 5 {
			t.Errorf("expected priority 5, got %d", composite.Priority())
		}
	})

	t.Run("ApplicableFor", func(t *testing.T) {
		strategy1 := NewFormatFallbackStrategy("table", "csv")
		strategy2 := NewDefaultValueStrategy(map[string]any{"field": "default"})

		composite := NewCompositeRecoveryStrategy("test-composite", strategy1, strategy2)

		// Should be applicable for format errors (strategy1)
		formatErr := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")
		if !composite.ApplicableFor(formatErr) {
			t.Error("expected composite to be applicable for format error")
		}

		// Should be applicable for missing column errors (strategy2)
		columnErr := NewOutputError(ErrMissingColumn, SeverityError, "missing column")
		if !composite.ApplicableFor(columnErr) {
			t.Error("expected composite to be applicable for missing column error")
		}

		// Should not be applicable for unhandled errors
		unhandledErr := NewOutputError(ErrNetworkTimeout, SeverityError, "network timeout")
		if composite.ApplicableFor(unhandledErr) {
			t.Error("expected composite to not be applicable for unhandled error")
		}
	})
}

func TestRecoveryHandlerPriority(t *testing.T) {
	t.Run("StrategiesSortedByPriority", func(t *testing.T) {
		handler := NewDefaultRecoveryHandler()

		// Add strategies in reverse priority order
		strategy1 := NewRetryStrategy(NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 3)) // Priority 30
		strategy2 := NewDefaultValueStrategy(map[string]any{"field": "default"})                     // Priority 20
		strategy3 := NewFormatFallbackStrategy("table", "csv")                                       // Priority 10
		strategy4 := NewCompositeRecoveryStrategy("composite", strategy3)                            // Priority 5

		handler.AddStrategy(strategy1)
		handler.AddStrategy(strategy2)
		handler.AddStrategy(strategy3)
		handler.AddStrategy(strategy4)

		// Get sorted strategies
		sorted := handler.getSortedStrategies()

		// Should be sorted by priority (lower numbers first)
		expectedPriorities := []int{5, 10, 20, 30}
		for i, strategy := range sorted {
			if strategy.Priority() != expectedPriorities[i] {
				t.Errorf("expected priority %d at index %d, got %d",
					expectedPriorities[i], i, strategy.Priority())
			}
		}
	})
}

func TestContextualRecoveryAdapter(t *testing.T) {
	t.Run("AsContextualRecovery", func(t *testing.T) {
		strategy := NewFormatFallbackStrategy("table", "csv")
		contextual := AsContextualRecovery(strategy)

		if contextual.Name() != strategy.Name() {
			t.Errorf("expected name %s, got %s", strategy.Name(), contextual.Name())
		}
		if contextual.Priority() != strategy.Priority() {
			t.Errorf("expected priority %d, got %d", strategy.Priority(), contextual.Priority())
		}
	})

	t.Run("ApplyWithContext", func(t *testing.T) {
		strategy := NewFormatFallbackStrategy("table", "csv")
		contextual := AsContextualRecovery(strategy)

		settings := &OutputSettings{OutputFormat: "table"}
		err := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")

		ctx := RecoveryContext{
			OriginalError: err,
			Settings:      settings,
		}

		result, applyErr := contextual.ApplyWithContext(ctx)
		if applyErr != nil {
			t.Errorf("expected no error, got %v", applyErr)
		}

		modifiedSettings, ok := result.(*OutputSettings)
		if !ok {
			t.Fatal("expected OutputSettings result")
		}

		if modifiedSettings.OutputFormat != "csv" {
			t.Errorf("expected format to be 'csv', got %s", modifiedSettings.OutputFormat)
		}
	})
}

func TestRecoveryIntegration(t *testing.T) {
	t.Run("CompleteRecoveryFlow", func(t *testing.T) {
		// Create a recovery handler with multiple strategies
		handler := NewDefaultRecoveryHandler()

		// Add strategies in order of preference
		fallbackStrategy := NewFormatFallbackStrategy("table", "csv", "json")
		defaultStrategy := NewDefaultValueStrategy(map[string]any{
			"name":   "Unknown",
			"status": "N/A",
		})
		retryStrategy := NewRetryStrategy(NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 3))

		handler.AddStrategy(fallbackStrategy)
		handler.AddStrategy(defaultStrategy)
		handler.AddStrategy(retryStrategy)

		// Test format error recovery
		formatErr := NewOutputError(ErrInvalidFormat, SeverityError, "invalid format")
		if !handler.CanRecover(formatErr) {
			t.Error("expected handler to be able to recover from format error")
		}

		// Test missing column error recovery
		columnErr := NewErrorBuilder(ErrMissingColumn, "missing column").
			WithField("name").
			Build()
		if !handler.CanRecover(columnErr) {
			t.Error("expected handler to be able to recover from missing column error")
		}

		// Test network error recovery
		networkErr := NewOutputError(ErrNetworkTimeout, SeverityError, "network timeout")
		if !handler.CanRecover(networkErr) {
			t.Error("expected handler to be able to recover from network error")
		}

		// Test unrecoverable error
		unrecoverableErr := NewOutputError(ErrPermissionDenied, SeverityError, "permission denied")
		if handler.CanRecover(unrecoverableErr) {
			t.Error("expected handler to not be able to recover from permission error")
		}
	})
}
