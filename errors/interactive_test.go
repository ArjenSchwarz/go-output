package errors

import (
	"errors"
	"testing"
)

// Mock UserInteraction for testing
type MockUserInteraction struct {
	resolutionChoice         ResolutionChoice
	confirmAutoFix           bool
	retryResponse            bool
	configurationValue       string
	configurationProvided    bool
	interactionPossible      bool
	promptCount              int
	fixConfirmationCount     int
	retryPromptCount         int
	configurationPromptCount int
}

func NewMockUserInteraction() *MockUserInteraction {
	return &MockUserInteraction{
		resolutionChoice:    ResolutionAbort,
		confirmAutoFix:      false,
		retryResponse:       false,
		interactionPossible: true,
	}
}

func (m *MockUserInteraction) PromptForResolution(err OutputError) ResolutionChoice {
	m.promptCount++
	return m.resolutionChoice
}

func (m *MockUserInteraction) ConfirmAutoFix(err OutputError, fix AutoFix) bool {
	m.fixConfirmationCount++
	return m.confirmAutoFix
}

func (m *MockUserInteraction) PromptForRetry(err OutputError, attemptCount int) bool {
	m.retryPromptCount++
	return m.retryResponse
}

func (m *MockUserInteraction) PromptForConfiguration(err OutputError, field string, suggestions []string) (string, bool) {
	m.configurationPromptCount++
	return m.configurationValue, m.configurationProvided
}

func (m *MockUserInteraction) IsInteractionPossible() bool {
	return m.interactionPossible
}

// Mock AutoFix for testing
type MockAutoFix struct {
	description string
	reversible  bool
	applyError  error
	undoError   error
	applyCount  int
	undoCount   int
}

func NewMockAutoFix(description string, reversible bool) *MockAutoFix {
	return &MockAutoFix{
		description: description,
		reversible:  reversible,
	}
}

func (m *MockAutoFix) Description() string {
	return m.description
}

func (m *MockAutoFix) Apply() error {
	m.applyCount++
	return m.applyError
}

func (m *MockAutoFix) IsReversible() bool {
	return m.reversible
}

func (m *MockAutoFix) Undo() error {
	m.undoCount++
	if !m.reversible {
		return errors.New("fix is not reversible")
	}
	return m.undoError
}

// Mock InteractiveErrorResolver for testing
type MockInteractiveErrorResolver struct {
	canResolve       bool
	autoFixes        []AutoFix
	resolveError     error
	resolveCallCount int
	canResolveCount  int
	getFixesCount    int
}

func NewMockInteractiveErrorResolver() *MockInteractiveErrorResolver {
	return &MockInteractiveErrorResolver{
		canResolve: true,
		autoFixes:  make([]AutoFix, 0),
	}
}

func (m *MockInteractiveErrorResolver) CanResolve(err OutputError) bool {
	m.canResolveCount++
	return m.canResolve
}

func (m *MockInteractiveErrorResolver) GetAutoFixes(err OutputError) []AutoFix {
	m.getFixesCount++
	return m.autoFixes
}

func (m *MockInteractiveErrorResolver) ResolveInteractively(err OutputError, interaction UserInteraction) error {
	m.resolveCallCount++
	return m.resolveError
}

func TestResolutionChoice_String(t *testing.T) {
	tests := []struct {
		choice   ResolutionChoice
		expected string
	}{
		{ResolutionIgnore, "Ignore"},
		{ResolutionRetry, "Retry"},
		{ResolutionFix, "Fix"},
		{ResolutionConfigure, "Configure"},
		{ResolutionAbort, "Abort"},
		{ResolutionChoice(999), "Unknown"},
	}

	for _, test := range tests {
		if result := test.choice.String(); result != test.expected {
			t.Errorf("ResolutionChoice.String() = %q, want %q", result, test.expected)
		}
	}
}

func TestFixSuggestion(t *testing.T) {
	applyCalled := false
	undoCalled := false

	fix := NewFixSuggestion("Test fix", func() error {
		applyCalled = true
		return nil
	})

	if fix.Description() != "Test fix" {
		t.Errorf("Description() = %q, want %q", fix.Description(), "Test fix")
	}

	if fix.IsReversible() {
		t.Error("IsReversible() = true, want false")
	}

	if err := fix.Apply(); err != nil {
		t.Errorf("Apply() returned error: %v", err)
	}

	if !applyCalled {
		t.Error("Apply() did not call the fix function")
	}

	if err := fix.Undo(); err == nil {
		t.Error("Undo() should return error for non-reversible fix")
	}

	// Test reversible fix
	reversibleFix := NewReversibleFixSuggestion("Reversible fix",
		func() error {
			applyCalled = true
			return nil
		},
		func() error {
			undoCalled = true
			return nil
		})

	if !reversibleFix.IsReversible() {
		t.Error("IsReversible() = false, want true for reversible fix")
	}

	if err := reversibleFix.Undo(); err != nil {
		t.Errorf("Undo() returned error: %v", err)
	}

	if !undoCalled {
		t.Error("Undo() did not call the undo function")
	}
}

func TestMockAutoFix(t *testing.T) {
	fix := NewMockAutoFix("Mock fix", true)

	if fix.Description() != "Mock fix" {
		t.Errorf("Description() = %q, want %q", fix.Description(), "Mock fix")
	}

	if !fix.IsReversible() {
		t.Error("IsReversible() = false, want true")
	}

	if err := fix.Apply(); err != nil {
		t.Errorf("Apply() returned error: %v", err)
	}

	if fix.applyCount != 1 {
		t.Errorf("Apply() call count = %d, want 1", fix.applyCount)
	}

	if err := fix.Undo(); err != nil {
		t.Errorf("Undo() returned error: %v", err)
	}

	if fix.undoCount != 1 {
		t.Errorf("Undo() call count = %d, want 1", fix.undoCount)
	}

	// Test non-reversible fix
	nonReversibleFix := NewMockAutoFix("Non-reversible", false)
	if err := nonReversibleFix.Undo(); err == nil {
		t.Error("Undo() should return error for non-reversible fix")
	}
}

func TestNoInteraction(t *testing.T) {
	noInteraction := NewNoInteraction()
	err := NewError(ErrInvalidFormat, "test error")

	if noInteraction.IsInteractionPossible() {
		t.Error("IsInteractionPossible() = true, want false")
	}

	if choice := noInteraction.PromptForResolution(err); choice != ResolutionAbort {
		t.Errorf("PromptForResolution() = %v, want ResolutionAbort", choice)
	}

	fix := NewMockAutoFix("test fix", false)
	if confirm := noInteraction.ConfirmAutoFix(err, fix); confirm {
		t.Error("ConfirmAutoFix() = true, want false")
	}

	if retry := noInteraction.PromptForRetry(err, 1); retry {
		t.Error("PromptForRetry() = true, want false")
	}

	value, ok := noInteraction.PromptForConfiguration(err, "field", []string{"suggestion"})
	if value != "" || ok {
		t.Errorf("PromptForConfiguration() = (%q, %v), want (\"\", false)", value, ok)
	}
}

func TestInteractiveErrorHandler_NewHandlers(t *testing.T) {
	handler := NewInteractiveErrorHandler()
	if handler.DefaultErrorHandler == nil {
		t.Error("DefaultErrorHandler is nil")
	}
	if handler.userInteraction == nil {
		t.Error("userInteraction is nil")
	}
	if handler.maxRetryCount != 3 {
		t.Errorf("maxRetryCount = %d, want 3", handler.maxRetryCount)
	}

	mockInteraction := NewMockUserInteraction()
	customHandler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	if customHandler.userInteraction != mockInteraction {
		t.Error("Custom interaction not set correctly")
	}
}

func TestInteractiveErrorHandler_Configuration(t *testing.T) {
	handler := NewInteractiveErrorHandler()

	// Test AddResolver
	resolver := NewMockInteractiveErrorResolver()
	handler.AddResolver(resolver)
	if len(handler.resolvers) != 1 {
		t.Errorf("Resolver count = %d, want 1", len(handler.resolvers))
	}

	// Test SetMaxRetryCount
	handler.SetMaxRetryCount(5)
	if handler.maxRetryCount != 5 {
		t.Errorf("maxRetryCount = %d, want 5", handler.maxRetryCount)
	}
}

func TestInteractiveErrorHandler_NonInteractiveMode(t *testing.T) {
	handler := NewInteractiveErrorHandler()
	handler.SetMode(ErrorModeStrict)

	err := NewError(ErrInvalidFormat, "test error")
	result := handler.HandleError(err)

	// Should delegate to DefaultErrorHandler in non-interactive mode
	if result == nil {
		t.Error("Expected error to be returned in strict mode")
	}
}

func TestInteractiveErrorHandler_NonInteractiveFallback(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.interactionPossible = false

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	err := NewError(ErrInvalidFormat, "test error")
	result := handler.HandleError(err)

	// Should fall back to lenient mode when interaction not possible
	if result != nil {
		t.Error("Expected nil result (lenient mode behavior) when interaction not possible")
	}

	errors := handler.Errors()
	if len(errors) != 1 {
		t.Errorf("Error count = %d, want 1", len(errors))
	}
}

func TestInteractiveErrorHandler_FatalErrors(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	err := NewError(ErrInvalidFormat, "fatal error").WithSeverity(SeverityFatal)
	result := handler.HandleError(err)

	// Fatal errors should always be returned immediately
	if result == nil {
		t.Error("Expected fatal error to be returned immediately")
	}

	if mockInteraction.promptCount > 0 {
		t.Error("Should not prompt for fatal errors")
	}
}

func TestInteractiveErrorHandler_WarningsAndInfo(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	// Test warning
	warning := NewError(ErrInvalidFormat, "warning").WithSeverity(SeverityWarning)
	result := handler.HandleError(warning)
	if result != nil {
		t.Error("Expected warning to be handled gracefully")
	}

	// Test info
	info := NewError(ErrInvalidFormat, "info").WithSeverity(SeverityInfo)
	result = handler.HandleError(info)
	if result != nil {
		t.Error("Expected info to be handled gracefully")
	}

	errors := handler.Errors()
	if len(errors) != 2 {
		t.Errorf("Error count = %d, want 2", len(errors))
	}

	if mockInteraction.promptCount > 0 {
		t.Error("Should not prompt for warnings and info")
	}
}

func TestInteractiveErrorHandler_ResolutionIgnore(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionIgnore

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	err := NewError(ErrInvalidFormat, "test error")
	result := handler.HandleError(err)

	if result != nil {
		t.Error("Expected ignored error to return nil")
	}

	errors := handler.Errors()
	if len(errors) != 1 {
		t.Errorf("Error count = %d, want 1", len(errors))
	}

	if mockInteraction.promptCount != 1 {
		t.Errorf("Prompt count = %d, want 1", mockInteraction.promptCount)
	}
}

func TestInteractiveErrorHandler_ResolutionAbort(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionAbort

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	err := NewError(ErrInvalidFormat, "test error")
	result := handler.HandleError(err)

	if result == nil {
		t.Error("Expected aborted error to be returned")
	}

	if mockInteraction.promptCount != 1 {
		t.Errorf("Prompt count = %d, want 1", mockInteraction.promptCount)
	}
}

func TestInteractiveErrorHandler_ResolutionFix(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionFix
	mockInteraction.confirmAutoFix = true

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	// Add resolver with auto-fix
	resolver := NewMockInteractiveErrorResolver()
	resolver.resolveError = errors.New("resolver failed") // Force resolver to fail so user gets prompted
	fix := NewMockAutoFix("Test fix", false)
	resolver.autoFixes = []AutoFix{fix}
	handler.AddResolver(resolver)

	err := NewError(ErrInvalidFormat, "test error")
	result := handler.HandleError(err)

	if result != nil {
		t.Error("Expected fixed error to return nil")
	}

	if fix.applyCount != 1 {
		t.Errorf("Fix apply count = %d, want 1", fix.applyCount)
	}

	if mockInteraction.fixConfirmationCount != 1 {
		t.Errorf("Fix confirmation count = %d, want 1", mockInteraction.fixConfirmationCount)
	}
}

func TestInteractiveErrorHandler_ResolutionFixDeclined(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionFix
	mockInteraction.confirmAutoFix = false

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)
	handler.SetMaxRetryCount(0) // Prevent retry loop

	// Add resolver with auto-fix
	resolver := NewMockInteractiveErrorResolver()
	resolver.resolveError = errors.New("resolver failed") // Force resolver to fail so user gets prompted
	fix := NewMockAutoFix("Test fix", false)
	resolver.autoFixes = []AutoFix{fix}
	handler.AddResolver(resolver)

	err := NewError(ErrInvalidFormat, "test error")
	result := handler.HandleError(err)

	// Should return error when fix is declined and no retries allowed
	if result == nil {
		t.Error("Expected error to be returned when fix declined")
	}

	if fix.applyCount != 0 {
		t.Errorf("Fix apply count = %d, want 0", fix.applyCount)
	}
}

func TestInteractiveErrorHandler_ResolutionConfigure(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionConfigure
	mockInteraction.configurationValue = "test-value"
	mockInteraction.configurationProvided = true

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	err := NewError(ErrMissingRequired, "missing field").
		WithContext(ErrorContext{Field: "test-field"})
	result := handler.HandleError(err)

	if result != nil {
		t.Error("Expected configured error to return nil")
	}

	if mockInteraction.configurationPromptCount != 1 {
		t.Errorf("Configuration prompt count = %d, want 1", mockInteraction.configurationPromptCount)
	}
}

func TestInteractiveErrorHandler_ResolutionConfigureNoField(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionConfigure

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)
	handler.SetMaxRetryCount(0) // Prevent retry loop

	err := NewError(ErrMissingRequired, "missing field") // No context field
	result := handler.HandleError(err)

	// Should return error when no field context available
	if result == nil {
		t.Error("Expected error to be returned when no field context")
	}
}

func TestInteractiveErrorHandler_ResolverIntegration(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	resolver := NewMockInteractiveErrorResolver()
	resolver.resolveError = nil // Successful resolution
	handler.AddResolver(resolver)

	err := NewError(ErrInvalidFormat, "test error")
	result := handler.HandleError(err)

	if result != nil {
		t.Error("Expected resolver to handle error successfully")
	}

	if resolver.resolveCallCount != 1 {
		t.Errorf("Resolver resolve call count = %d, want 1", resolver.resolveCallCount)
	}

	// Resolver success should prevent user prompting
	if mockInteraction.promptCount > 0 {
		t.Error("Should not prompt when resolver succeeds")
	}
}

func TestInteractiveErrorHandler_RetryLogic(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionRetry

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)
	handler.SetMaxRetryCount(2)

	err := NewError(ErrInvalidFormat, "test error")
	result := handler.HandleError(err)

	// Should return error after exceeding max retries
	if result == nil {
		t.Error("Expected error after exceeding max retries")
	}

	// Should prompt multiple times (initial + retries)
	if mockInteraction.promptCount < 3 {
		t.Errorf("Prompt count = %d, want at least 3", mockInteraction.promptCount)
	}
}

func TestInteractiveErrorHandler_NilError(t *testing.T) {
	handler := NewInteractiveErrorHandler()
	result := handler.HandleError(nil)

	if result != nil {
		t.Error("Expected nil error to return nil")
	}
}

func TestInteractiveErrorHandler_RegularError(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionIgnore

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	regularErr := errors.New("regular error")
	result := handler.HandleError(regularErr)

	if result != nil {
		t.Error("Expected wrapped regular error to be handled")
	}

	errors := handler.Errors()
	if len(errors) != 1 {
		t.Errorf("Error count = %d, want 1", len(errors))
	}
}

func TestDefaultUserInteraction_IsInteractionPossible(t *testing.T) {
	ui := NewDefaultUserInteraction()
	// Note: This test may vary depending on test environment
	// We're just testing that the method doesn't panic
	_ = ui.IsInteractionPossible()
}

func TestFixSuggestion_EdgeCases(t *testing.T) {
	// Test fix with nil function
	fix := &FixSuggestion{description: "nil fix"}
	if err := fix.Apply(); err == nil {
		t.Error("Expected error when fix function is nil")
	}

	// Test non-reversible undo
	nonReversible := NewFixSuggestion("non-reversible", func() error { return nil })
	if err := nonReversible.Undo(); err == nil {
		t.Error("Expected error when undoing non-reversible fix")
	}

	// Test reversible with nil undo function
	reversible := &FixSuggestion{
		description: "bad reversible",
		fixFunc:     func() error { return nil },
		reversible:  true,
		undoFunc:    nil,
	}
	if reversible.IsReversible() {
		t.Error("Should not be reversible with nil undo function")
	}
}

// Integration test combining multiple components
func TestInteractiveErrorHandling_Integration(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	// Test sequence: Fix -> Configure -> Ignore
	testErrors := []OutputError{
		NewError(ErrInvalidFormat, "format error"),
		NewError(ErrMissingRequired, "missing config").WithContext(ErrorContext{Field: "output-format"}),
		NewError(ErrInvalidDataType, "type error"),
	}

	choices := []ResolutionChoice{ResolutionFix, ResolutionConfigure, ResolutionIgnore}
	mockInteraction.confirmAutoFix = true
	mockInteraction.configurationValue = "json"
	mockInteraction.configurationProvided = true

	// Add resolver with fixes
	resolver := NewMockInteractiveErrorResolver()
	resolver.resolveError = errors.New("resolver failed") // Force resolver to fail so user gets prompted
	fix := NewMockAutoFix("Format fix", false)
	resolver.autoFixes = []AutoFix{fix}
	handler.AddResolver(resolver)

	for i, err := range testErrors {
		mockInteraction.resolutionChoice = choices[i]
		result := handler.HandleError(err)

		if result != nil {
			t.Errorf("Error %d: Expected nil result, got %v", i, result)
		}
	}

	// Verify all interactions occurred
	if mockInteraction.promptCount != 3 {
		t.Errorf("Prompt count = %d, want 3", mockInteraction.promptCount)
	}

	if mockInteraction.fixConfirmationCount != 1 {
		t.Errorf("Fix confirmation count = %d, want 1", mockInteraction.fixConfirmationCount)
	}

	if mockInteraction.configurationPromptCount != 1 {
		t.Errorf("Configuration prompt count = %d, want 1", mockInteraction.configurationPromptCount)
	}

	// Check final error collection
	// Only the last error (ResolutionIgnore) should be in the collection
	// Fix and Configure should have resolved their errors successfully
	errors := handler.Errors()
	if len(errors) != 1 {
		t.Errorf("Final error count = %d, want 1", len(errors))
	}
}

// Benchmark interactive error handling performance
func BenchmarkInteractiveErrorHandler(b *testing.B) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionIgnore

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	err := NewError(ErrInvalidFormat, "benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandleError(err)
		handler.Clear() // Reset for next iteration
	}
}

func TestInteractiveErrorHandler_CallbackIntegration(t *testing.T) {
	mockInteraction := NewMockUserInteraction()
	mockInteraction.resolutionChoice = ResolutionIgnore

	handler := NewInteractiveErrorHandlerWithInteraction(mockInteraction)
	handler.SetMode(ErrorModeInteractive)

	warningCalled := false
	infoCalled := false

	handler.SetWarningHandler(func(err error) {
		warningCalled = true
	})

	handler.SetInfoHandler(func(err error) {
		infoCalled = true
	})

	// Test warning callback
	warning := NewError(ErrInvalidFormat, "warning").WithSeverity(SeverityWarning)
	handler.HandleError(warning)

	if !warningCalled {
		t.Error("Warning handler was not called")
	}

	// Test info callback
	info := NewError(ErrInvalidFormat, "info").WithSeverity(SeverityInfo)
	handler.HandleError(info)

	if !infoCalled {
		t.Error("Info handler was not called")
	}
}
