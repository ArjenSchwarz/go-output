package errors

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const unknownString = "Unknown"

// UserInteraction handles user interaction for error resolution
type UserInteraction interface {
	// PromptForResolution asks the user how to handle an error
	PromptForResolution(err OutputError) ResolutionChoice
	// ConfirmAutoFix asks the user to confirm an automatic fix
	ConfirmAutoFix(err OutputError, fix AutoFix) bool
	// PromptForRetry asks the user if they want to retry an operation
	PromptForRetry(err OutputError, attemptCount int) bool
	// PromptForConfiguration asks the user to provide missing configuration
	PromptForConfiguration(err OutputError, field string, suggestions []string) (string, bool)
	// IsInteractionPossible checks if user interaction is available
	IsInteractionPossible() bool
}

// ResolutionChoice represents the user's choice for error resolution
type ResolutionChoice int

// User resolution choices
const (
	ResolutionIgnore    ResolutionChoice = iota // Ignore the error and continue
	ResolutionRetry                             // Retry the operation
	ResolutionFix                               // Apply an automatic fix
	ResolutionConfigure                         // Provide missing configuration
	ResolutionAbort                             // Abort the operation
)

// String returns the string representation of ResolutionChoice
func (r ResolutionChoice) String() string {
	switch r {
	case ResolutionIgnore:
		return "Ignore"
	case ResolutionRetry:
		return "Retry"
	case ResolutionFix:
		return "Fix"
	case ResolutionConfigure:
		return "Configure"
	case ResolutionAbort:
		return "Abort"
	default:
		return unknownString
	}
}

// AutoFix represents an automatic fix that can be applied to resolve an error
type AutoFix interface {
	// Description returns a human-readable description of the fix
	Description() string
	// Apply applies the fix and returns any error
	Apply() error
	// IsReversible returns true if the fix can be undone
	IsReversible() bool
	// Undo reverses the fix if it's reversible
	Undo() error
}

// FixSuggestion represents a suggested fix for an error
type FixSuggestion struct {
	description string
	fixFunc     func() error
	undoFunc    func() error
	reversible  bool
}

// NewFixSuggestion creates a new fix suggestion
func NewFixSuggestion(description string, fixFunc func() error) *FixSuggestion {
	return &FixSuggestion{
		description: description,
		fixFunc:     fixFunc,
		reversible:  false,
	}
}

// NewReversibleFixSuggestion creates a new reversible fix suggestion
func NewReversibleFixSuggestion(description string, fixFunc, undoFunc func() error) *FixSuggestion {
	return &FixSuggestion{
		description: description,
		fixFunc:     fixFunc,
		undoFunc:    undoFunc,
		reversible:  true,
	}
}

// Description returns a human-readable description of the fix
func (f *FixSuggestion) Description() string {
	return f.description
}

// Apply applies the fix and returns any error
func (f *FixSuggestion) Apply() error {
	if f.fixFunc == nil {
		return fmt.Errorf("no fix function available")
	}
	return f.fixFunc()
}

// IsReversible returns true if the fix can be undone
func (f *FixSuggestion) IsReversible() bool {
	return f.reversible && f.undoFunc != nil
}

// Undo reverses the fix if it's reversible
func (f *FixSuggestion) Undo() error {
	if !f.IsReversible() {
		return fmt.Errorf("fix is not reversible")
	}
	return f.undoFunc()
}

// InteractiveErrorResolver manages interactive error resolution
type InteractiveErrorResolver interface {
	// CanResolve checks if this resolver can handle the given error
	CanResolve(err OutputError) bool
	// GetAutoFixes returns automatic fixes available for the error
	GetAutoFixes(err OutputError) []AutoFix
	// ResolveInteractively handles the error with user interaction
	ResolveInteractively(err OutputError, interaction UserInteraction) error
}

// DefaultUserInteraction provides console-based user interaction
type DefaultUserInteraction struct {
	reader *bufio.Reader
}

// NewDefaultUserInteraction creates a new console-based user interaction
func NewDefaultUserInteraction() *DefaultUserInteraction {
	return &DefaultUserInteraction{
		reader: bufio.NewReader(os.Stdin),
	}
}

// PromptForResolution asks the user how to handle an error
func (ui *DefaultUserInteraction) PromptForResolution(err OutputError) ResolutionChoice {
	if !ui.IsInteractionPossible() {
		return ResolutionAbort
	}

	fmt.Printf("\nError encountered:\n")
	fmt.Printf("Code: %s\n", err.Code())
	fmt.Printf("Message: %s\n", err.Error())

	if context := err.Context(); context.Operation != "" || context.Field != "" {
		fmt.Printf("Context: Operation=%s, Field=%s\n", context.Operation, context.Field)
	}

	suggestions := err.Suggestions()
	if len(suggestions) > 0 {
		fmt.Printf("Suggestions:\n")
		for i, suggestion := range suggestions {
			fmt.Printf("  %d. %s\n", i+1, suggestion)
		}
	}

	fmt.Printf("\nHow would you like to proceed?\n")
	fmt.Printf("1. Ignore this error and continue\n")
	fmt.Printf("2. Retry the operation\n")
	fmt.Printf("3. Apply automatic fix (if available)\n")
	fmt.Printf("4. Provide configuration\n")
	fmt.Printf("5. Abort\n")
	fmt.Printf("Choice (1-5): ")

	for {
		input, readErr := ui.reader.ReadString('\n')
		if readErr != nil {
			return ResolutionAbort
		}

		input = strings.TrimSpace(input)
		switch input {
		case "1":
			return ResolutionIgnore
		case "2":
			return ResolutionRetry
		case "3":
			return ResolutionFix
		case "4":
			return ResolutionConfigure
		case "5":
			return ResolutionAbort
		default:
			fmt.Printf("Invalid choice. Please enter 1-5: ")
		}
	}
}

// ConfirmAutoFix asks the user to confirm an automatic fix
func (ui *DefaultUserInteraction) ConfirmAutoFix(err OutputError, fix AutoFix) bool {
	if !ui.IsInteractionPossible() {
		return false
	}

	fmt.Printf("\nAutomatic fix available:\n")
	fmt.Printf("Fix: %s\n", fix.Description())
	if fix.IsReversible() {
		fmt.Printf("This fix is reversible.\n")
	} else {
		fmt.Printf("WARNING: This fix is NOT reversible.\n")
	}
	fmt.Printf("Apply this fix? (y/n): ")

	for {
		input, readErr := ui.reader.ReadString('\n')
		if readErr != nil {
			return false
		}

		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Printf("Please enter y or n: ")
		}
	}
}

// PromptForRetry asks the user if they want to retry an operation
func (ui *DefaultUserInteraction) PromptForRetry(err OutputError, attemptCount int) bool {
	if !ui.IsInteractionPossible() {
		return false
	}

	fmt.Printf("\nOperation failed (attempt %d):\n", attemptCount)
	fmt.Printf("Error: %s\n", err.Error())
	fmt.Printf("Retry? (y/n): ")

	for {
		input, readErr := ui.reader.ReadString('\n')
		if readErr != nil {
			return false
		}

		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Printf("Please enter y or n: ")
		}
	}
}

// PromptForConfiguration asks the user to provide missing configuration
func (ui *DefaultUserInteraction) PromptForConfiguration(err OutputError, field string, suggestions []string) (string, bool) {
	if !ui.IsInteractionPossible() {
		return "", false
	}

	fmt.Printf("\nMissing configuration for field: %s\n", field)
	if len(suggestions) > 0 {
		fmt.Printf("Suggested values:\n")
		for i, suggestion := range suggestions {
			fmt.Printf("  %d. %s\n", i+1, suggestion)
		}
		fmt.Printf("Enter a number to select, or type a custom value: ")
	} else {
		fmt.Printf("Please provide a value: ")
	}

	input, readErr := ui.reader.ReadString('\n')
	if readErr != nil {
		return "", false
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return "", false
	}

	// Check if user selected a suggestion by number
	if len(suggestions) > 0 {
		if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(suggestions) {
			return suggestions[num-1], true
		}
	}

	return input, true
}

// IsInteractionPossible checks if user interaction is available
func (ui *DefaultUserInteraction) IsInteractionPossible() bool {
	// Check if we're running in an interactive terminal
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// NoInteraction provides a non-interactive implementation for automated environments
type NoInteraction struct{}

// NewNoInteraction creates a new non-interactive user interaction
func NewNoInteraction() *NoInteraction {
	return &NoInteraction{}
}

// PromptForResolution always returns ResolutionAbort for non-interactive environments
func (ni *NoInteraction) PromptForResolution(err OutputError) ResolutionChoice {
	return ResolutionAbort
}

// ConfirmAutoFix always returns false for non-interactive environments
func (ni *NoInteraction) ConfirmAutoFix(err OutputError, fix AutoFix) bool {
	return false
}

// PromptForRetry always returns false for non-interactive environments
func (ni *NoInteraction) PromptForRetry(err OutputError, attemptCount int) bool {
	return false
}

// PromptForConfiguration always returns empty values for non-interactive environments
func (ni *NoInteraction) PromptForConfiguration(err OutputError, field string, suggestions []string) (string, bool) {
	return "", false
}

// IsInteractionPossible always returns false for non-interactive environments
func (ni *NoInteraction) IsInteractionPossible() bool {
	return false
}

// InteractiveErrorHandler extends DefaultErrorHandler with interactive capabilities
type InteractiveErrorHandler struct {
	*DefaultErrorHandler
	userInteraction UserInteraction
	resolvers       []InteractiveErrorResolver
	maxRetryCount   int
}

// NewInteractiveErrorHandler creates a new interactive error handler
func NewInteractiveErrorHandler() *InteractiveErrorHandler {
	return &InteractiveErrorHandler{
		DefaultErrorHandler: NewDefaultErrorHandler(),
		userInteraction:     NewDefaultUserInteraction(),
		resolvers:           make([]InteractiveErrorResolver, 0),
		maxRetryCount:       3,
	}
}

// NewInteractiveErrorHandlerWithInteraction creates an interactive error handler with custom interaction
func NewInteractiveErrorHandlerWithInteraction(interaction UserInteraction) *InteractiveErrorHandler {
	return &InteractiveErrorHandler{
		DefaultErrorHandler: NewDefaultErrorHandler(),
		userInteraction:     interaction,
		resolvers:           make([]InteractiveErrorResolver, 0),
		maxRetryCount:       3,
	}
}

// AddResolver adds an interactive error resolver
func (h *InteractiveErrorHandler) AddResolver(resolver InteractiveErrorResolver) {
	h.resolvers = append(h.resolvers, resolver)
}

// SetMaxRetryCount sets the maximum number of retry attempts
func (h *InteractiveErrorHandler) SetMaxRetryCount(count int) {
	h.maxRetryCount = count
}

// HandleError processes an error with interactive resolution
func (h *InteractiveErrorHandler) HandleError(err error) error {
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

	// If not in interactive mode, delegate to parent handler
	if h.Mode() != ErrorModeInteractive {
		return h.DefaultErrorHandler.HandleError(outputErr)
	}

	return h.handleInteractiveError(outputErr)
}

// handleInteractiveError implements the interactive error handling logic
func (h *InteractiveErrorHandler) handleInteractiveError(err OutputError) error {
	// Handle non-interactive environment
	if !h.userInteraction.IsInteractionPossible() {
		// Fall back to lenient mode in non-interactive environments
		return h.handleLenient(err)
	}

	// For fatal errors, always return immediately
	if err.Severity() == SeverityFatal {
		h.mutex.Lock()
		h.errors = append(h.errors, err)
		h.mutex.Unlock()
		return err
	}

	// Handle warnings and info messages with callbacks first
	if err.Severity() == SeverityWarning && h.warningHandler != nil {
		h.warningHandler(err)
	}
	if err.Severity() == SeverityInfo && h.infoHandler != nil {
		h.infoHandler(err)
	}

	// For warnings and info, just collect and continue
	if err.Severity() >= SeverityWarning {
		h.mutex.Lock()
		h.errors = append(h.errors, err)
		h.mutex.Unlock()
		return nil
	}

	// For errors, try interactive resolution
	retryCount := 0
	for retryCount <= h.maxRetryCount {
		// Check if any resolver can handle this error
		var resolver InteractiveErrorResolver
		for _, r := range h.resolvers {
			if r.CanResolve(err) {
				resolver = r
				break
			}
		}

		if resolver != nil {
			// Try resolver-specific interactive handling
			if resolverErr := resolver.ResolveInteractively(err, h.userInteraction); resolverErr == nil {
				// Successfully resolved
				return nil
			}
		}

		// Get user choice for resolution
		choice := h.userInteraction.PromptForResolution(err)

		switch choice {
		case ResolutionIgnore:
			// Collect error and continue
			h.mutex.Lock()
			h.errors = append(h.errors, err)
			h.mutex.Unlock()
			return nil

		case ResolutionRetry:
			retryCount++
			if retryCount > h.maxRetryCount {
				fmt.Printf("Maximum retry attempts (%d) exceeded.\n", h.maxRetryCount)
				break
			}
			// Continue loop to retry
			continue

		case ResolutionFix:
			if applied := h.tryAutoFix(err); applied {
				// Auto-fix was applied successfully
				return nil
			}
			// If auto-fix failed, check retry count
			retryCount++
			if retryCount > h.maxRetryCount {
				fmt.Printf("Maximum retry attempts (%d) exceeded.\n", h.maxRetryCount)
				return err
			}
			// Continue to ask for next action
			continue

		case ResolutionConfigure:
			if configured := h.tryConfiguration(err); configured {
				// Configuration was provided successfully
				return nil
			}
			// If configuration failed, check retry count
			retryCount++
			if retryCount > h.maxRetryCount {
				fmt.Printf("Maximum retry attempts (%d) exceeded.\n", h.maxRetryCount)
				return err
			}
			// Continue to ask for next action
			continue

		case ResolutionAbort:
			// Return the error to abort
			return err

		default:
			// unknownString choice, treat as abort
			return err
		}
	}

	// If we get here, all retry attempts failed
	return err
}

// tryAutoFix attempts to apply automatic fixes for the error
func (h *InteractiveErrorHandler) tryAutoFix(err OutputError) bool {
	for _, resolver := range h.resolvers {
		if !resolver.CanResolve(err) {
			continue
		}

		fixes := resolver.GetAutoFixes(err)
		for _, fix := range fixes {
			if h.userInteraction.ConfirmAutoFix(err, fix) {
				if applyErr := fix.Apply(); applyErr == nil {
					fmt.Printf("Fix applied successfully: %s\n", fix.Description())
					return true
				} else {
					fmt.Printf("Fix failed: %v\n", applyErr)
				}
			}
		}
	}

	fmt.Printf("No automatic fixes available or user declined all fixes.\n")
	return false
}

// tryConfiguration attempts to get missing configuration from the user
func (h *InteractiveErrorHandler) tryConfiguration(err OutputError) bool {
	context := err.Context()
	if context.Field == "" {
		fmt.Printf("Cannot determine which configuration field is missing.\n")
		return false
	}

	suggestions := err.Suggestions()
	value, ok := h.userInteraction.PromptForConfiguration(err, context.Field, suggestions)
	if !ok || value == "" {
		fmt.Printf("No configuration provided.\n")
		return false
	}

	fmt.Printf("Configuration received: %s = %s\n", context.Field, value)
	// Note: In a real implementation, this would update the actual configuration
	// For now, we just simulate success
	return true
}
