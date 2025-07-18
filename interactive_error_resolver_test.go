package format

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestUserAction_String(t *testing.T) {
	tests := []struct {
		action   UserAction
		expected string
	}{
		{UserActionAbort, "abort"},
		{UserActionSkip, "skip"},
		{UserActionRetry, "retry"},
		{UserActionApplyFix, "apply_fix"},
		{UserActionIgnore, "ignore"},
		{UserAction(999), "unknown"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.action.String()
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestNewInteractiveErrorResolver(t *testing.T) {
	resolver := NewInteractiveErrorResolver()

	if resolver == nil {
		t.Fatal("Expected resolver to be created")
	}

	if resolver.input == nil {
		t.Error("Expected input scanner to be initialized")
	}

	if resolver.output != os.Stdout {
		t.Error("Expected output to be stdout")
	}

	if !resolver.autoFixEnabled {
		t.Error("Expected auto-fix to be enabled by default")
	}

	if resolver.retryLimit != 3 {
		t.Errorf("Expected retry limit to be 3, got %d", resolver.retryLimit)
	}

	if resolver.fixRegistry == nil {
		t.Error("Expected fix registry to be initialized")
	}
}

func TestNewInteractiveErrorResolverWithOptions(t *testing.T) {
	input := bufio.NewScanner(strings.NewReader("test input"))
	output := os.Stderr
	autoFixEnabled := false

	resolver := NewInteractiveErrorResolverWithOptions(input, output, autoFixEnabled)

	if resolver.input != input {
		t.Error("Expected custom input scanner")
	}

	if resolver.output != output {
		t.Error("Expected custom output")
	}

	if resolver.autoFixEnabled != autoFixEnabled {
		t.Error("Expected auto-fix to be disabled")
	}
}

func TestInteractiveErrorResolver_SetRetryLimit(t *testing.T) {
	resolver := NewInteractiveErrorResolver()

	// Test setting valid limit
	resolver.SetRetryLimit(5)
	if resolver.retryLimit != 5 {
		t.Errorf("Expected retry limit to be 5, got %d", resolver.retryLimit)
	}

	// Test setting invalid limit (should be ignored)
	resolver.SetRetryLimit(0)
	if resolver.retryLimit != 5 {
		t.Errorf("Expected retry limit to remain 5, got %d", resolver.retryLimit)
	}

	resolver.SetRetryLimit(-1)
	if resolver.retryLimit != 5 {
		t.Errorf("Expected retry limit to remain 5, got %d", resolver.retryLimit)
	}
}

func TestInteractiveErrorResolver_SetAutoFixEnabled(t *testing.T) {
	resolver := NewInteractiveErrorResolver()

	// Initially enabled
	if !resolver.autoFixEnabled {
		t.Error("Expected auto-fix to be enabled initially")
	}

	// Disable
	resolver.SetAutoFixEnabled(false)
	if resolver.autoFixEnabled {
		t.Error("Expected auto-fix to be disabled")
	}

	// Re-enable
	resolver.SetAutoFixEnabled(true)
	if !resolver.autoFixEnabled {
		t.Error("Expected auto-fix to be re-enabled")
	}
}

func TestInteractiveErrorResolver_RegisterAutoFix(t *testing.T) {
	resolver := NewInteractiveErrorResolver()

	fix := AutoFix{
		Name:        "Test Fix",
		Description: "A test fix",
		ApplyFunc: func() error {
			return nil
		},
	}

	// Register fix for a specific error code
	resolver.RegisterAutoFix(ErrInvalidFormat, fix)

	// Check that the fix was registered
	fixes := resolver.fixRegistry[ErrInvalidFormat]
	if len(fixes) != 1 {
		t.Errorf("Expected 1 fix to be registered, got %d", len(fixes))
	}

	if fixes[0].Name != fix.Name {
		t.Errorf("Expected fix name to be '%s', got '%s'", fix.Name, fixes[0].Name)
	}

	// Register another fix for the same error code
	fix2 := AutoFix{
		Name:        "Test Fix 2",
		Description: "Another test fix",
		ApplyFunc: func() error {
			return nil
		},
	}

	resolver.RegisterAutoFix(ErrInvalidFormat, fix2)

	// Check that both fixes are registered
	fixes = resolver.fixRegistry[ErrInvalidFormat]
	if len(fixes) != 2 {
		t.Errorf("Expected 2 fixes to be registered, got %d", len(fixes))
	}
}

func TestInteractiveErrorResolver_GetBuiltInFixes(t *testing.T) {
	resolver := NewInteractiveErrorResolver()

	tests := []struct {
		errorCode     ErrorCode
		expectedFix   string
		shouldHaveFix bool
	}{
		{ErrInvalidFormat, "Use JSON format", true},
		{ErrMissingColumn, "Add default column", true},
		{ErrInvalidFilePath, "Use current directory", true},
		{ErrEmptyDataset, "Create sample data", true},
		{ErrNetworkTimeout, "", false}, // No built-in fix for this
	}

	for _, test := range tests {
		t.Run(string(test.errorCode), func(t *testing.T) {
			err := NewConfigError(test.errorCode, "test error")
			fixes := resolver.getBuiltInFixes(err)

			if test.shouldHaveFix {
				if len(fixes) == 0 {
					t.Errorf("Expected at least one fix for %s", test.errorCode)
					return
				}

				found := false
				for _, fix := range fixes {
					if fix.Name == test.expectedFix {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected to find fix '%s' for error code %s", test.expectedFix, test.errorCode)
				}
			} else {
				if len(fixes) > 0 {
					t.Errorf("Expected no fixes for %s, got %d", test.errorCode, len(fixes))
				}
			}
		})
	}
}

func TestInteractiveErrorResolver_ApplyAutoFix(t *testing.T) {
	resolver := NewInteractiveErrorResolver()

	t.Run("successful fix", func(t *testing.T) {
		applied := false
		fix := &AutoFix{
			Name:        "Test Fix",
			Description: "A test fix",
			ApplyFunc: func() error {
				applied = true
				return nil
			},
		}

		err := resolver.applyAutoFix(fix)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !applied {
			t.Error("Expected fix to be applied")
		}
	})

	t.Run("fix with no apply function", func(t *testing.T) {
		fix := &AutoFix{
			Name:        "Broken Fix",
			Description: "A fix with no apply function",
			ApplyFunc:   nil,
		}

		err := resolver.applyAutoFix(fix)
		if err == nil {
			t.Error("Expected error for fix with no apply function")
		}

		expectedMsg := "fix 'Broken Fix' has no apply function"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("failing fix", func(t *testing.T) {
		fix := &AutoFix{
			Name:        "Failing Fix",
			Description: "A fix that fails",
			ApplyFunc: func() error {
				return fmt.Errorf("fix failed")
			},
		}

		err := resolver.applyAutoFix(fix)
		if err == nil {
			t.Error("Expected error from failing fix")
		}

		if err.Error() != "fix failed" {
			t.Errorf("Expected error message 'fix failed', got '%s'", err.Error())
		}
	})
}

func TestInteractiveErrorResolver_CreatePrompt(t *testing.T) {
	resolver := NewInteractiveErrorResolver()

	t.Run("basic error prompt", func(t *testing.T) {
		err := NewConfigError(ErrInvalidFormat, "invalid format")
		prompt := resolver.createPrompt(err)

		if prompt.Message == "" {
			t.Error("Expected prompt message to be set")
		}

		// Should have at least abort and skip options
		if len(prompt.Options) < 2 {
			t.Errorf("Expected at least 2 options, got %d", len(prompt.Options))
		}

		// Check for abort option
		hasAbort := false
		for _, option := range prompt.Options {
			if option.Action == UserActionAbort {
				hasAbort = true
				break
			}
		}
		if !hasAbort {
			t.Error("Expected abort option to be present")
		}

		// Check for skip option
		hasSkip := false
		for _, option := range prompt.Options {
			if option.Action == UserActionSkip {
				hasSkip = true
				break
			}
		}
		if !hasSkip {
			t.Error("Expected skip option to be present")
		}
	})

	t.Run("retryable error prompt", func(t *testing.T) {
		err := NewProcessingError(ErrS3Upload, "upload failed", true)
		prompt := resolver.createPrompt(err)

		// Should have retry option for retryable errors
		hasRetry := false
		for _, option := range prompt.Options {
			if option.Action == UserActionRetry {
				hasRetry = true
				break
			}
		}
		if !hasRetry {
			t.Error("Expected retry option for retryable error")
		}
	})

	t.Run("prompt with auto-fix disabled", func(t *testing.T) {
		resolver.SetAutoFixEnabled(false)
		err := NewConfigError(ErrInvalidFormat, "invalid format")
		prompt := resolver.createPrompt(err)

		// Should not have auto-fix options
		for _, option := range prompt.Options {
			if option.Action == UserActionApplyFix {
				t.Error("Expected no auto-fix options when auto-fix is disabled")
			}
		}
	})

	t.Run("warning error default", func(t *testing.T) {
		err := NewErrorBuilder(ErrMalformedData, "warning").
			WithSeverity(SeverityWarning).
			Build()
		prompt := resolver.createPrompt(err)

		// Default should be skip for warnings
		if prompt.DefaultIdx >= len(prompt.Options) {
			t.Error("Default index is out of range")
		} else if prompt.Options[prompt.DefaultIdx].Action != UserActionSkip {
			t.Error("Expected default action to be skip for warnings")
		}
	})
}

func TestInteractiveErrorResolver_ResolveError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		resolver := NewInteractiveErrorResolver()
		err := resolver.ResolveError(nil)
		if err != nil {
			t.Errorf("Expected nil error to return nil, got %v", err)
		}
	})

	t.Run("abort action", func(t *testing.T) {
		// Create resolver with simulated input
		input := bufio.NewScanner(strings.NewReader("a\n")) // 'a' for abort
		output := &bytes.Buffer{}
		resolver := NewInteractiveErrorResolverWithOptions(input, &mockFile{output}, true)

		originalErr := NewConfigError(ErrInvalidFormat, "test error")
		err := resolver.ResolveError(originalErr)

		// Should return the original error (abort)
		if err != originalErr {
			t.Error("Expected original error to be returned for abort action")
		}
	})

	t.Run("skip action", func(t *testing.T) {
		// Create resolver with simulated input
		input := bufio.NewScanner(strings.NewReader("s\n")) // 's' for skip
		output := &bytes.Buffer{}
		resolver := NewInteractiveErrorResolverWithOptions(input, &mockFile{output}, true)

		originalErr := NewConfigError(ErrInvalidFormat, "test error")
		err := resolver.ResolveError(originalErr)

		// Should return nil (skip)
		if err != nil {
			t.Errorf("Expected nil for skip action, got %v", err)
		}
	})

	t.Run("invalid input with retry", func(t *testing.T) {
		// Create resolver with invalid input followed by skip
		input := bufio.NewScanner(strings.NewReader("invalid\ns\n"))
		output := &bytes.Buffer{}
		resolver := NewInteractiveErrorResolverWithOptions(input, &mockFile{output}, true)

		originalErr := NewConfigError(ErrInvalidFormat, "test error")
		err := resolver.ResolveError(originalErr)

		// Should eventually skip after invalid input
		if err != nil {
			t.Errorf("Expected nil after invalid input and skip, got %v", err)
		}
	})
}

// mockFile implements a minimal os.File interface for testing
type mockFile struct {
	buffer *bytes.Buffer
}

func (m *mockFile) Write(p []byte) (n int, err error) {
	return m.buffer.Write(p)
}

func (m *mockFile) WriteString(s string) (n int, err error) {
	return m.buffer.WriteString(s)
}

func TestGuidedErrorResolution(t *testing.T) {
	resolver := NewInteractiveErrorResolver()
	guided := NewGuidedErrorResolution(resolver)

	if guided.resolver != resolver {
		t.Error("Expected resolver to be set correctly")
	}

	if len(guided.steps) != 0 {
		t.Error("Expected steps to be empty initially")
	}
}

func TestGuidedErrorResolution_AddStep(t *testing.T) {
	resolver := NewInteractiveErrorResolver()
	guided := NewGuidedErrorResolution(resolver)

	step := ResolutionStep{
		Title:       "Test Step",
		Description: "A test step",
		Action: func() error {
			return nil
		},
	}

	guided.AddStep(step)

	if len(guided.steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(guided.steps))
	}

	if guided.steps[0].Title != step.Title {
		t.Errorf("Expected step title '%s', got '%s'", step.Title, guided.steps[0].Title)
	}
}

func TestRetryMechanism(t *testing.T) {
	resolver := NewInteractiveErrorResolver()
	retry := NewRetryMechanism(resolver)

	if retry.resolver != resolver {
		t.Error("Expected resolver to be set correctly")
	}

	if retry.maxAttempts != 3 {
		t.Errorf("Expected max attempts to be 3, got %d", retry.maxAttempts)
	}

	if !retry.backoffEnabled {
		t.Error("Expected backoff to be enabled by default")
	}

	if !retry.userGuidance {
		t.Error("Expected user guidance to be enabled by default")
	}
}

func TestRetryMechanism_SetMaxAttempts(t *testing.T) {
	resolver := NewInteractiveErrorResolver()
	retry := NewRetryMechanism(resolver)

	// Test setting valid max attempts
	retry.SetMaxAttempts(5)
	if retry.maxAttempts != 5 {
		t.Errorf("Expected max attempts to be 5, got %d", retry.maxAttempts)
	}

	// Test setting invalid max attempts (should be ignored)
	retry.SetMaxAttempts(0)
	if retry.maxAttempts != 5 {
		t.Errorf("Expected max attempts to remain 5, got %d", retry.maxAttempts)
	}

	retry.SetMaxAttempts(-1)
	if retry.maxAttempts != 5 {
		t.Errorf("Expected max attempts to remain 5, got %d", retry.maxAttempts)
	}
}

func TestRetryMechanism_SetBackoffEnabled(t *testing.T) {
	resolver := NewInteractiveErrorResolver()
	retry := NewRetryMechanism(resolver)

	// Initially enabled
	if !retry.backoffEnabled {
		t.Error("Expected backoff to be enabled initially")
	}

	// Disable
	retry.SetBackoffEnabled(false)
	if retry.backoffEnabled {
		t.Error("Expected backoff to be disabled")
	}

	// Re-enable
	retry.SetBackoffEnabled(true)
	if !retry.backoffEnabled {
		t.Error("Expected backoff to be re-enabled")
	}
}

func TestRetryMechanism_SetUserGuidance(t *testing.T) {
	resolver := NewInteractiveErrorResolver()
	retry := NewRetryMechanism(resolver)

	// Initially enabled
	if !retry.userGuidance {
		t.Error("Expected user guidance to be enabled initially")
	}

	// Disable
	retry.SetUserGuidance(false)
	if retry.userGuidance {
		t.Error("Expected user guidance to be disabled")
	}

	// Re-enable
	retry.SetUserGuidance(true)
	if !retry.userGuidance {
		t.Error("Expected user guidance to be re-enabled")
	}
}

func TestErrorResolutionWorkflow(t *testing.T) {
	workflow := NewErrorResolutionWorkflow()

	if workflow.resolver == nil {
		t.Error("Expected resolver to be initialized")
	}

	if workflow.guidedResolution == nil {
		t.Error("Expected guided resolution to be initialized")
	}

	if workflow.retryMechanism == nil {
		t.Error("Expected retry mechanism to be initialized")
	}

	if workflow.autoFixAttempted {
		t.Error("Expected auto-fix attempted to be false initially")
	}
}

func TestErrorResolutionWorkflow_CreateGuidedResolutionForError(t *testing.T) {
	workflow := NewErrorResolutionWorkflow()

	tests := []struct {
		errorCode ErrorCode
		name      string
	}{
		{ErrInvalidFormat, "format error"},
		{ErrMissingColumn, "missing column error"},
		{ErrInvalidFilePath, "file path error"},
		{ErrNetworkTimeout, "generic error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewConfigError(test.errorCode, "test error")

			workflow.CreateGuidedResolutionForError(err)

			// Should have added steps (CreateGuidedResolutionForError creates a new instance)
			if len(workflow.guidedResolution.steps) == 0 {
				t.Error("Expected steps to be added for guided resolution")
			}
		})
	}
}

func TestInteractiveErrorResolver_DisplayError(t *testing.T) {
	output := &bytes.Buffer{}
	resolver := NewInteractiveErrorResolverWithOptions(
		bufio.NewScanner(strings.NewReader("")),
		&mockFile{output},
		true,
	)

	err := NewErrorBuilder(ErrInvalidFormat, "test error").
		WithSeverity(SeverityError).
		WithField("test_field").
		WithOperation("test_operation").
		WithValue("test_value").
		WithSuggestions("Try using JSON format", "Check the documentation").
		Build()

	resolver.displayError(err)

	outputStr := output.String()

	// Check that error information is displayed
	if !strings.Contains(outputStr, "Error Encountered") {
		t.Error("Expected error header to be displayed")
	}

	if !strings.Contains(outputStr, string(ErrInvalidFormat)) {
		t.Error("Expected error code to be displayed")
	}

	if !strings.Contains(outputStr, "error") {
		t.Error("Expected severity to be displayed")
	}

	if !strings.Contains(outputStr, "test_field") {
		t.Error("Expected field to be displayed")
	}

	if !strings.Contains(outputStr, "test_operation") {
		t.Error("Expected operation to be displayed")
	}

	if !strings.Contains(outputStr, "test_value") {
		t.Error("Expected value to be displayed")
	}

	if !strings.Contains(outputStr, "Try using JSON format") {
		t.Error("Expected suggestions to be displayed")
	}
}

func TestInteractiveErrorResolver_PromptUser(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		input := bufio.NewScanner(strings.NewReader("a\n"))
		output := &bytes.Buffer{}
		resolver := NewInteractiveErrorResolverWithOptions(input, &mockFile{output}, true)

		prompt := UserPrompt{
			Message: "Test prompt",
			Options: []PromptOption{
				{Key: "a", Description: "Abort", Action: UserActionAbort},
				{Key: "s", Description: "Skip", Action: UserActionSkip},
			},
			DefaultIdx: 0,
		}

		action, autoFix := resolver.promptUser(prompt)

		if action != UserActionAbort {
			t.Errorf("Expected UserActionAbort, got %v", action)
		}

		if autoFix != nil {
			t.Error("Expected no auto-fix for abort action")
		}
	})

	t.Run("empty input uses default", func(t *testing.T) {
		input := bufio.NewScanner(strings.NewReader("\n"))
		output := &bytes.Buffer{}
		resolver := NewInteractiveErrorResolverWithOptions(input, &mockFile{output}, true)

		prompt := UserPrompt{
			Message: "Test prompt",
			Options: []PromptOption{
				{Key: "a", Description: "Abort", Action: UserActionAbort},
				{Key: "s", Description: "Skip", Action: UserActionSkip},
			},
			DefaultIdx: 1, // Skip is default
		}

		action, _ := resolver.promptUser(prompt)

		if action != UserActionSkip {
			t.Errorf("Expected UserActionSkip (default), got %v", action)
		}
	})

	t.Run("input failure uses default", func(t *testing.T) {
		// Create a scanner that will fail to scan
		input := bufio.NewScanner(strings.NewReader(""))
		input.Scan() // Consume the empty input to make next Scan() return false

		output := &bytes.Buffer{}
		resolver := NewInteractiveErrorResolverWithOptions(input, &mockFile{output}, true)

		prompt := UserPrompt{
			Message: "Test prompt",
			Options: []PromptOption{
				{Key: "a", Description: "Abort", Action: UserActionAbort},
				{Key: "s", Description: "Skip", Action: UserActionSkip},
			},
			DefaultIdx: 0, // Abort is default
		}

		action, _ := resolver.promptUser(prompt)

		if action != UserActionAbort {
			t.Errorf("Expected UserActionAbort (default), got %v", action)
		}
	})
}

func TestInteractiveErrorResolver_Integration(t *testing.T) {
	t.Run("complete error resolution flow", func(t *testing.T) {
		// Simulate user choosing to skip the error
		input := bufio.NewScanner(strings.NewReader("s\n"))
		output := &bytes.Buffer{}
		resolver := NewInteractiveErrorResolverWithOptions(input, &mockFile{output}, true)

		err := NewErrorBuilder(ErrInvalidFormat, "invalid format").
			WithSeverity(SeverityError).
			WithSuggestions("Use JSON format").
			Build()

		result := resolver.ResolveError(err)

		// Should return nil (error was skipped)
		if result != nil {
			t.Errorf("Expected nil result for skipped error, got %v", result)
		}

		// Check that error information was displayed
		outputStr := output.String()
		if !strings.Contains(outputStr, "Error Encountered") {
			t.Error("Expected error to be displayed")
		}

		if !strings.Contains(outputStr, "How would you like to handle") {
			t.Error("Expected prompt to be displayed")
		}
	})
}
