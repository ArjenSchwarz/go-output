package output

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDebugLevel(t *testing.T) {
	tests := []struct {
		level    DebugLevel
		expected string
	}{
		{DebugOff, "OFF"},
		{DebugError, "ERROR"},
		{DebugWarn, "WARN"},
		{DebugInfo, "INFO"},
		{DebugTrace, "TRACE"},
		{DebugLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("DebugLevel.String() = %q, want %q", tt.level.String(), tt.expected)
			}
		})
	}
}

func TestNewDebugTracer(t *testing.T) {
	t.Run("with output writer", func(t *testing.T) {
		var buf bytes.Buffer
		tracer := NewDebugTracer(DebugInfo, &buf)

		if tracer.GetLevel() != DebugInfo {
			t.Errorf("tracer level = %v, want %v", tracer.GetLevel(), DebugInfo)
		}

		if !tracer.IsEnabled() {
			t.Errorf("tracer should be enabled for level > DebugOff")
		}
	})

	t.Run("with nil output", func(t *testing.T) {
		tracer := NewDebugTracer(DebugTrace, nil)

		if tracer.output != os.Stderr {
			t.Errorf("tracer should default to os.Stderr when output is nil")
		}
	})

	t.Run("with DebugOff", func(t *testing.T) {
		tracer := NewDebugTracer(DebugOff, nil)

		if tracer.IsEnabled() {
			t.Errorf("tracer should be disabled for DebugOff level")
		}
	})
}

func TestDebugTracerLevelManagement(t *testing.T) {
	var buf bytes.Buffer
	tracer := NewDebugTracer(DebugOff, &buf)

	if tracer.IsEnabled() {
		t.Errorf("tracer should start disabled")
	}

	tracer.SetLevel(DebugInfo)

	if tracer.GetLevel() != DebugInfo {
		t.Errorf("tracer level = %v, want %v", tracer.GetLevel(), DebugInfo)
	}

	if !tracer.IsEnabled() {
		t.Errorf("tracer should be enabled after setting level > DebugOff")
	}

	tracer.SetLevel(DebugOff)

	if tracer.IsEnabled() {
		t.Errorf("tracer should be disabled after setting level to DebugOff")
	}
}

func TestDebugTracerContext(t *testing.T) {
	var buf bytes.Buffer
	tracer := NewDebugTracer(DebugTrace, &buf)

	tracer.AddContext("format", "json")
	tracer.AddContext("operation", "render")

	tracer.Info("test", "test message")

	output := buf.String()
	if !strings.Contains(output, "format=json") {
		t.Errorf("output should contain context: %s", output)
	}
	if !strings.Contains(output, "operation=render") {
		t.Errorf("output should contain context: %s", output)
	}

	// Test removing context
	tracer.RemoveContext("format")
	buf.Reset()
	tracer.Info("test", "test message 2")

	output = buf.String()
	if strings.Contains(output, "format=json") {
		t.Errorf("output should not contain removed context: %s", output)
	}
	if !strings.Contains(output, "operation=render") {
		t.Errorf("output should still contain remaining context: %s", output)
	}

	// Test clearing context
	tracer.ClearContext()
	buf.Reset()
	tracer.Info("test", "test message 3")

	output = buf.String()
	if strings.Contains(output, "operation=render") {
		t.Errorf("output should not contain any context after clear: %s", output)
	}
}

func TestDebugTracerLogging(t *testing.T) {
	tests := []struct {
		name         string
		tracerLevel  DebugLevel
		messageLevel DebugLevel
		shouldLog    bool
	}{
		{"trace logs at trace level", DebugTrace, DebugTrace, true},
		{"trace logs at info level", DebugTrace, DebugInfo, true},
		{"trace logs at warn level", DebugTrace, DebugWarn, true},
		{"trace logs at error level", DebugTrace, DebugError, true},
		{"info doesn't log trace", DebugInfo, DebugTrace, false},
		{"info logs at info level", DebugInfo, DebugInfo, true},
		{"info logs at warn level", DebugInfo, DebugWarn, true},
		{"info logs at error level", DebugInfo, DebugError, true},
		{"warn doesn't log trace", DebugWarn, DebugTrace, false},
		{"warn doesn't log info", DebugWarn, DebugInfo, false},
		{"warn logs at warn level", DebugWarn, DebugWarn, true},
		{"warn logs at error level", DebugWarn, DebugError, true},
		{"error doesn't log trace", DebugError, DebugTrace, false},
		{"error doesn't log info", DebugError, DebugInfo, false},
		{"error doesn't log warn", DebugError, DebugWarn, false},
		{"error logs at error level", DebugError, DebugError, true},
		{"off doesn't log anything", DebugOff, DebugError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tracer := NewDebugTracer(tt.tracerLevel, &buf)

			// Call the appropriate logging method
			switch tt.messageLevel {
			case DebugTrace:
				tracer.Trace("test", "trace message")
			case DebugInfo:
				tracer.Info("test", "info message")
			case DebugWarn:
				tracer.Warn("test", "warn message")
			case DebugError:
				tracer.Error("test", "error message")
			}

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.shouldLog {
				t.Errorf("expected shouldLog=%v, but hasOutput=%v, output: %q", tt.shouldLog, hasOutput, output)
			}

			if hasOutput {
				expectedLevel := fmt.Sprintf("[%s]", tt.messageLevel.String())
				if !strings.Contains(output, expectedLevel) {
					t.Errorf("output should contain level %s: %s", expectedLevel, output)
				}
			}
		})
	}
}

func TestDebugTracerOperationTracing(t *testing.T) {
	var buf bytes.Buffer
	tracer := NewDebugTracer(DebugTrace, &buf)

	t.Run("successful operation", func(t *testing.T) {
		buf.Reset()

		err := tracer.TraceOperation("test-op", func() error {
			time.Sleep(1 * time.Millisecond) // Small delay to test duration
			return nil
		})

		if err != nil {
			t.Errorf("TraceOperation should not return error for successful operation")
		}

		output := buf.String()
		if !strings.Contains(output, "starting operation") {
			t.Errorf("output should contain start message: %s", output)
		}
		if !strings.Contains(output, "completed successfully") {
			t.Errorf("output should contain success message: %s", output)
		}
	})

	t.Run("failed operation", func(t *testing.T) {
		buf.Reset()
		testError := errors.New("test error")

		err := tracer.TraceOperation("test-op", func() error {
			return testError
		})

		if err != testError {
			t.Errorf("TraceOperation should return the original error")
		}

		output := buf.String()
		if !strings.Contains(output, "starting operation") {
			t.Errorf("output should contain start message: %s", output)
		}
		if !strings.Contains(output, "operation failed") {
			t.Errorf("output should contain failure message: %s", output)
		}
		if !strings.Contains(output, "test error") {
			t.Errorf("output should contain error message: %s", output)
		}
	})
}

func TestDebugTracerOperationWithResult(t *testing.T) {
	var buf bytes.Buffer
	tracer := NewDebugTracer(DebugTrace, &buf)

	t.Run("successful operation with result", func(t *testing.T) {
		buf.Reset()

		result, err := tracer.TraceOperationWithResult("test-op", func() (any, error) {
			return "test result", nil
		})

		if err != nil {
			t.Errorf("TraceOperationWithResult should not return error for successful operation")
		}

		if result != "test result" {
			t.Errorf("TraceOperationWithResult should return the result: got %q, want %q", result, "test result")
		}

		output := buf.String()
		if !strings.Contains(output, "starting operation") {
			t.Errorf("output should contain start message: %s", output)
		}
		if !strings.Contains(output, "completed successfully") {
			t.Errorf("output should contain success message: %s", output)
		}
	})

	t.Run("failed operation with result", func(t *testing.T) {
		buf.Reset()
		testError := errors.New("test error")

		result, err := tracer.TraceOperationWithResult("test-op", func() (any, error) {
			return 42, testError
		})

		if err != testError {
			t.Errorf("TraceOperationWithResult should return the original error")
		}

		if result != 42 {
			t.Errorf("TraceOperationWithResult should return the result even on error: got %d, want %d", result, 42)
		}

		output := buf.String()
		if !strings.Contains(output, "operation failed") {
			t.Errorf("output should contain failure message: %s", output)
		}
	})
}

func TestPanicError(t *testing.T) {
	t.Run("panic with string", func(t *testing.T) {
		panicErr := NewPanicError("test-op", "panic message")

		expectedMsg := `panic in operation "test-op": panic message`
		if panicErr.Error() != expectedMsg {
			t.Errorf("PanicError.Error() = %q, want %q", panicErr.Error(), expectedMsg)
		}

		if panicErr.Operation != "test-op" {
			t.Errorf("PanicError.Operation = %q, want %q", panicErr.Operation, "test-op")
		}

		if panicErr.Value != "panic message" {
			t.Errorf("PanicError.Value = %v, want %v", panicErr.Value, "panic message")
		}

		if len(panicErr.StackTrace()) == 0 {
			t.Errorf("PanicError should have stack trace")
		}
	})

	t.Run("panic with error", func(t *testing.T) {
		originalError := errors.New("original error")
		panicErr := NewPanicError("test-op", originalError)

		if panicErr.Unwrap() != originalError {
			t.Errorf("PanicError should wrap the original error")
		}

		if !errors.Is(panicErr, originalError) {
			t.Errorf("PanicError should be identifiable as the original error")
		}
	})

	t.Run("panic without operation", func(t *testing.T) {
		panicErr := NewPanicError("", "panic message")

		expectedMsg := "panic: panic message"
		if panicErr.Error() != expectedMsg {
			t.Errorf("PanicError.Error() = %q, want %q", panicErr.Error(), expectedMsg)
		}
	})
}

func TestSafeExecute(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		err := SafeExecute("test-op", func() error {
			return nil
		})

		if err != nil {
			t.Errorf("SafeExecute should not return error for successful operation")
		}
	})

	t.Run("error execution", func(t *testing.T) {
		testError := errors.New("test error")
		err := SafeExecute("test-op", func() error {
			return testError
		})

		if err != testError {
			t.Errorf("SafeExecute should return the original error")
		}
	})

	t.Run("panic recovery", func(t *testing.T) {
		err := SafeExecute("test-op", func() error {
			panic("test panic")
		})

		if err == nil {
			t.Errorf("SafeExecute should return error for panic")
		}

		var panicErr *PanicError
		if !AsError(err, &panicErr) {
			t.Errorf("SafeExecute should return PanicError for panic")
		}

		if panicErr.Value != "test panic" {
			t.Errorf("PanicError should contain the panic value")
		}

		if panicErr.Operation != "test-op" {
			t.Errorf("PanicError should contain the operation name")
		}
	})
}

func TestSafeExecuteWithResult(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		result, err := SafeExecuteWithResult("test-op", func() (any, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("SafeExecuteWithResult should not return error for successful operation")
		}

		if result != "success" {
			t.Errorf("SafeExecuteWithResult should return the result")
		}
	})

	t.Run("error execution", func(t *testing.T) {
		testError := errors.New("test error")
		result, err := SafeExecuteWithResult("test-op", func() (any, error) {
			return 42, testError
		})

		if err != testError {
			t.Errorf("SafeExecuteWithResult should return the original error")
		}

		if result != 42 {
			t.Errorf("SafeExecuteWithResult should return the result even on error")
		}
	})

	t.Run("panic recovery", func(t *testing.T) {
		result, err := SafeExecuteWithResult("test-op", func() (any, error) {
			panic("test panic")
		})

		if err == nil {
			t.Errorf("SafeExecuteWithResult should return error for panic")
		}

		if result != nil {
			t.Errorf("SafeExecuteWithResult should return nil for panic")
		}

		var panicErr *PanicError
		if !AsError(err, &panicErr) {
			t.Errorf("SafeExecuteWithResult should return PanicError for panic")
		}
	})
}

func TestSafeExecuteWithTracer(t *testing.T) {
	var buf bytes.Buffer
	tracer := NewDebugTracer(DebugTrace, &buf)

	t.Run("with tracer - successful", func(t *testing.T) {
		buf.Reset()

		err := SafeExecuteWithTracer(tracer, "test-op", func() error {
			return nil
		})

		if err != nil {
			t.Errorf("SafeExecuteWithTracer should not return error for successful operation")
		}

		output := buf.String()
		if !strings.Contains(output, "starting operation") {
			t.Errorf("output should contain tracing messages: %s", output)
		}
	})

	t.Run("with tracer - panic", func(t *testing.T) {
		buf.Reset()

		err := SafeExecuteWithTracer(tracer, "test-op", func() error {
			panic("test panic")
		})

		if err == nil {
			t.Errorf("SafeExecuteWithTracer should return error for panic")
		}

		output := buf.String()
		if !strings.Contains(output, "panic recovered") {
			t.Errorf("output should contain panic recovery message: %s", output)
		}
		if !strings.Contains(output, "test panic") {
			t.Errorf("output should contain panic value: %s", output)
		}
	})
}

func TestPanicHelpers(t *testing.T) {
	t.Run("IsPanic", func(t *testing.T) {
		regularError := errors.New("regular error")
		panicError := NewPanicError("test", "panic")

		if IsPanic(nil) {
			t.Errorf("IsPanic should return false for nil")
		}

		if IsPanic(regularError) {
			t.Errorf("IsPanic should return false for regular error")
		}

		if !IsPanic(panicError) {
			t.Errorf("IsPanic should return true for PanicError")
		}

		wrappedPanicError := ErrorWithContext("test", panicError)
		if !IsPanic(wrappedPanicError) {
			t.Errorf("IsPanic should return true for wrapped PanicError")
		}
	})

	t.Run("GetPanicValue", func(t *testing.T) {
		panicError := NewPanicError("test", "panic value")

		value := GetPanicValue(panicError)
		if value != "panic value" {
			t.Errorf("GetPanicValue should return the panic value")
		}

		value = GetPanicValue(errors.New("regular error"))
		if value != nil {
			t.Errorf("GetPanicValue should return nil for non-panic error")
		}

		value = GetPanicValue(nil)
		if value != nil {
			t.Errorf("GetPanicValue should return nil for nil error")
		}
	})

	t.Run("GetStackTrace", func(t *testing.T) {
		panicError := NewPanicError("test", "panic value")

		trace := GetStackTrace(panicError)
		if len(trace) == 0 {
			t.Errorf("GetStackTrace should return non-empty stack trace")
		}

		trace = GetStackTrace(errors.New("regular error"))
		if trace != "" {
			t.Errorf("GetStackTrace should return empty string for non-panic error")
		}

		trace = GetStackTrace(nil)
		if trace != "" {
			t.Errorf("GetStackTrace should return empty string for nil error")
		}
	})
}

func TestGlobalDebugTracer(t *testing.T) {
	// Save original state
	originalTracer := GetGlobalDebugTracer()
	defer SetGlobalDebugTracer(originalTracer)

	t.Run("set and get global tracer", func(t *testing.T) {
		var buf bytes.Buffer
		tracer := NewDebugTracer(DebugInfo, &buf)

		SetGlobalDebugTracer(tracer)

		retrieved := GetGlobalDebugTracer()
		if retrieved != tracer {
			t.Errorf("GetGlobalDebugTracer should return the set tracer")
		}
	})

	t.Run("enable global debug tracing", func(t *testing.T) {
		var buf bytes.Buffer
		EnableGlobalDebugTracing(DebugWarn, &buf)

		tracer := GetGlobalDebugTracer()
		if tracer == nil {
			t.Errorf("EnableGlobalDebugTracing should set a global tracer")
		}

		if tracer.GetLevel() != DebugWarn {
			t.Errorf("Global tracer should have the specified level")
		}
	})

	t.Run("disable global debug tracing", func(t *testing.T) {
		EnableGlobalDebugTracing(DebugInfo, os.Stderr)
		DisableGlobalDebugTracing()

		tracer := GetGlobalDebugTracer()
		if tracer != nil {
			t.Errorf("DisableGlobalDebugTracing should clear the global tracer")
		}
	})

	t.Run("global debug functions", func(t *testing.T) {
		var buf bytes.Buffer
		EnableGlobalDebugTracing(DebugTrace, &buf)

		GlobalTrace("test", "trace message")
		GlobalInfo("test", "info message")
		GlobalWarn("test", "warn message")
		GlobalError("test", "error message")

		output := buf.String()
		if !strings.Contains(output, "trace message") {
			t.Errorf("output should contain trace message: %s", output)
		}
		if !strings.Contains(output, "info message") {
			t.Errorf("output should contain info message: %s", output)
		}
		if !strings.Contains(output, "warn message") {
			t.Errorf("output should contain warn message: %s", output)
		}
		if !strings.Contains(output, "error message") {
			t.Errorf("output should contain error message: %s", output)
		}
	})

	t.Run("global debug functions with no tracer", func(t *testing.T) {
		DisableGlobalDebugTracing()

		// These should not panic even with no global tracer
		GlobalTrace("test", "trace message")
		GlobalInfo("test", "info message")
		GlobalWarn("test", "warn message")
		GlobalError("test", "error message")
	})
}

func TestDebugTracerConcurrency(t *testing.T) {
	var buf bytes.Buffer
	tracer := NewDebugTracer(DebugTrace, &buf)

	// Test concurrent access to tracer methods
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				tracer.AddContext(fmt.Sprintf("goroutine_%d", id), j)
				tracer.Info("concurrent", "message %d from goroutine %d", j, id)
				tracer.RemoveContext(fmt.Sprintf("goroutine_%d", id))

				// Test level changes
				if j%10 == 0 {
					tracer.SetLevel(DebugWarn)
					tracer.SetLevel(DebugTrace)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify that no race conditions occurred
	if !tracer.IsEnabled() {
		t.Errorf("Tracer should still be enabled after concurrent operations")
	}

	if tracer.GetLevel() != DebugTrace {
		t.Errorf("Tracer level should be DebugTrace after concurrent operations")
	}
}
