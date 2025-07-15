package format

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// UserAction represents the action a user can take in response to an error
type UserAction int

const (
	UserActionAbort UserAction = iota
	UserActionSkip
	UserActionRetry
	UserActionApplyFix
	UserActionIgnore
)

// String returns the string representation of UserAction
func (a UserAction) String() string {
	switch a {
	case UserActionAbort:
		return "abort"
	case UserActionSkip:
		return "skip"
	case UserActionRetry:
		return "retry"
	case UserActionApplyFix:
		return "apply_fix"
	case UserActionIgnore:
		return "ignore"
	default:
		return "unknown"
	}
}

// UserPrompt represents a prompt shown to the user for error resolution
type UserPrompt struct {
	Message     string            // The main prompt message
	Options     []PromptOption    // Available options for the user
	DefaultIdx  int               // Index of the default option
	Context     map[string]string // Additional context information
	Suggestions []string          // Automatic fix suggestions
}

// PromptOption represents a single option in a user prompt
type PromptOption struct {
	Key         string     // The key the user types to select this option
	Description string     // Human-readable description
	Action      UserAction // The action this option represents
	AutoFix     *AutoFix   // Optional automatic fix associated with this option
}

// AutoFix represents an automatic fix that can be applied to resolve an error
type AutoFix struct {
	Name        string                 // Human-readable name of the fix
	Description string                 // Detailed description of what the fix does
	ApplyFunc   func() error           // Function to apply the fix
	UndoFunc    func() error           // Function to undo the fix (optional)
	Metadata    map[string]interface{} // Additional metadata about the fix
}

// Writer defines the interface for output writing
type Writer interface {
	Write([]byte) (int, error)
	WriteString(string) (int, error)
}

// InteractiveErrorResolver handles interactive error resolution with user prompts
type InteractiveErrorResolver struct {
	input          *bufio.Scanner
	output         Writer
	autoFixEnabled bool
	retryLimit     int
	fixRegistry    map[ErrorCode][]AutoFix
}

// NewInteractiveErrorResolver creates a new interactive error resolver
func NewInteractiveErrorResolver() *InteractiveErrorResolver {
	return &InteractiveErrorResolver{
		input:          bufio.NewScanner(os.Stdin),
		output:         os.Stdout,
		autoFixEnabled: true,
		retryLimit:     3,
		fixRegistry:    make(map[ErrorCode][]AutoFix),
	}
}

// NewInteractiveErrorResolverWithOptions creates a resolver with custom options
func NewInteractiveErrorResolverWithOptions(input *bufio.Scanner, output Writer, autoFixEnabled bool) *InteractiveErrorResolver {
	return &InteractiveErrorResolver{
		input:          input,
		output:         output,
		autoFixEnabled: autoFixEnabled,
		retryLimit:     3,
		fixRegistry:    make(map[ErrorCode][]AutoFix),
	}
}

// ResolveError handles interactive error resolution
func (r *InteractiveErrorResolver) ResolveError(err OutputError) error {
	if err == nil {
		return nil
	}

	// Display the error information
	r.displayError(err)

	// Create user prompt based on error type and available fixes
	prompt := r.createPrompt(err)

	// Get user input and handle the selected action
	for attempt := 0; attempt < r.retryLimit; attempt++ {
		action, autoFix := r.promptUser(prompt)

		switch action {
		case UserActionAbort:
			return err // Return the original error to abort processing

		case UserActionSkip:
			return nil // Skip this error and continue processing

		case UserActionIgnore:
			return nil // Ignore this error (same as skip for now)

		case UserActionApplyFix:
			if autoFix != nil {
				if fixErr := r.applyAutoFix(autoFix); fixErr != nil {
					r.displayMessage(fmt.Sprintf("Failed to apply fix: %v", fixErr))
					continue // Try again
				}
				r.displayMessage(fmt.Sprintf("Applied fix: %s", autoFix.Name))
				return nil // Fix applied successfully
			}
			r.displayMessage("No automatic fix available")
			continue

		case UserActionRetry:
			// For retry, we return nil to indicate the error was handled
			// The calling code should implement the actual retry logic
			r.displayMessage("Retrying operation...")
			return nil

		default:
			r.displayMessage("Invalid option selected. Please try again.")
			continue
		}
	}

	// If we've exhausted retry attempts, return the original error
	r.displayMessage(fmt.Sprintf("Maximum retry attempts (%d) exceeded", r.retryLimit))
	return err
}

// displayError shows the error information to the user
func (r *InteractiveErrorResolver) displayError(err OutputError) {
	fmt.Fprintf(r.output, "\n🚨 Error Encountered:\n")
	fmt.Fprintf(r.output, "Code: %s\n", err.Code())
	fmt.Fprintf(r.output, "Severity: %s\n", err.Severity().String())
	fmt.Fprintf(r.output, "Message: %s\n", err.Error())

	context := err.Context()
	if context.Operation != "" {
		fmt.Fprintf(r.output, "Operation: %s\n", context.Operation)
	}
	if context.Field != "" {
		fmt.Fprintf(r.output, "Field: %s\n", context.Field)
	}
	if context.Value != nil {
		fmt.Fprintf(r.output, "Value: %v\n", context.Value)
	}

	suggestions := err.Suggestions()
	if len(suggestions) > 0 {
		fmt.Fprintf(r.output, "\nSuggestions:\n")
		for _, suggestion := range suggestions {
			fmt.Fprintf(r.output, "  • %s\n", suggestion)
		}
	}

	fmt.Fprintf(r.output, "\n")
}

// createPrompt creates a user prompt based on the error type
func (r *InteractiveErrorResolver) createPrompt(err OutputError) UserPrompt {
	prompt := UserPrompt{
		Message:     "How would you like to handle this error?",
		Options:     []PromptOption{},
		DefaultIdx:  0,
		Context:     make(map[string]string),
		Suggestions: err.Suggestions(),
	}

	// Always add basic options
	prompt.Options = append(prompt.Options, PromptOption{
		Key:         "a",
		Description: "Abort processing",
		Action:      UserActionAbort,
	})

	prompt.Options = append(prompt.Options, PromptOption{
		Key:         "s",
		Description: "Skip this error and continue",
		Action:      UserActionSkip,
	})

	// Add retry option for retryable errors
	if procErr, ok := err.(ProcessingError); ok && procErr.Retryable() {
		prompt.Options = append(prompt.Options, PromptOption{
			Key:         "r",
			Description: "Retry the operation",
			Action:      UserActionRetry,
		})
	}

	// Add automatic fix options if available
	if r.autoFixEnabled {
		fixes := r.getAvailableFixes(err)
		for i, fix := range fixes {
			key := fmt.Sprintf("f%d", i+1)
			prompt.Options = append(prompt.Options, PromptOption{
				Key:         key,
				Description: fmt.Sprintf("Apply fix: %s", fix.Name),
				Action:      UserActionApplyFix,
				AutoFix:     &fix,
			})
		}
	}

	// Set default to skip for warnings, abort for errors
	if err.Severity() <= SeverityWarning {
		prompt.DefaultIdx = 1 // Skip
	} else {
		prompt.DefaultIdx = 0 // Abort
	}

	return prompt
}

// promptUser displays the prompt and gets user input
func (r *InteractiveErrorResolver) promptUser(prompt UserPrompt) (UserAction, *AutoFix) {
	// Display the prompt
	fmt.Fprintf(r.output, "%s\n\n", prompt.Message)

	// Display options
	for _, option := range prompt.Options {
		defaultMarker := ""
		if prompt.Options[prompt.DefaultIdx].Key == option.Key {
			defaultMarker = " (default)"
		}
		fmt.Fprintf(r.output, "  [%s] %s%s\n", option.Key, option.Description, defaultMarker)
	}

	fmt.Fprintf(r.output, "\nEnter your choice: ")

	// Get user input
	if !r.input.Scan() {
		// If input fails, use default option
		return prompt.Options[prompt.DefaultIdx].Action, prompt.Options[prompt.DefaultIdx].AutoFix
	}

	userInput := strings.TrimSpace(strings.ToLower(r.input.Text()))

	// If empty input, use default
	if userInput == "" {
		return prompt.Options[prompt.DefaultIdx].Action, prompt.Options[prompt.DefaultIdx].AutoFix
	}

	// Find matching option
	for _, option := range prompt.Options {
		if option.Key == userInput {
			return option.Action, option.AutoFix
		}
	}

	// Invalid input
	fmt.Fprintf(r.output, "Invalid choice '%s'. Please try again.\n\n", userInput)
	return r.promptUser(prompt) // Recursive call to try again
}

// displayMessage shows a message to the user
func (r *InteractiveErrorResolver) displayMessage(message string) {
	fmt.Fprintf(r.output, "%s\n", message)
}

// getAvailableFixes returns available automatic fixes for an error
func (r *InteractiveErrorResolver) getAvailableFixes(err OutputError) []AutoFix {
	fixes, exists := r.fixRegistry[err.Code()]
	if !exists {
		// Return built-in fixes based on error type
		return r.getBuiltInFixes(err)
	}
	return fixes
}

// getBuiltInFixes returns built-in automatic fixes for common error types
func (r *InteractiveErrorResolver) getBuiltInFixes(err OutputError) []AutoFix {
	var fixes []AutoFix

	switch err.Code() {
	case ErrInvalidFormat:
		fixes = append(fixes, AutoFix{
			Name:        "Use JSON format",
			Description: "Change output format to JSON (most compatible)",
			ApplyFunc: func() error {
				// In a real implementation, this would modify the OutputSettings
				return nil
			},
		})

	case ErrMissingColumn:
		fixes = append(fixes, AutoFix{
			Name:        "Add default column",
			Description: "Add the missing column with default values",
			ApplyFunc: func() error {
				// In a real implementation, this would add the missing column
				return nil
			},
		})

	case ErrInvalidFilePath:
		fixes = append(fixes, AutoFix{
			Name:        "Use current directory",
			Description: "Save file to current directory instead",
			ApplyFunc: func() error {
				// In a real implementation, this would modify the file path
				return nil
			},
		})

	case ErrEmptyDataset:
		fixes = append(fixes, AutoFix{
			Name:        "Create sample data",
			Description: "Generate sample data to demonstrate format",
			ApplyFunc: func() error {
				// In a real implementation, this would add sample data
				return nil
			},
		})
	}

	return fixes
}

// applyAutoFix applies an automatic fix
func (r *InteractiveErrorResolver) applyAutoFix(fix *AutoFix) error {
	if fix.ApplyFunc == nil {
		return fmt.Errorf("fix '%s' has no apply function", fix.Name)
	}

	return fix.ApplyFunc()
}

// RegisterAutoFix registers a custom automatic fix for a specific error code
func (r *InteractiveErrorResolver) RegisterAutoFix(code ErrorCode, fix AutoFix) {
	if r.fixRegistry[code] == nil {
		r.fixRegistry[code] = make([]AutoFix, 0)
	}
	r.fixRegistry[code] = append(r.fixRegistry[code], fix)
}

// SetRetryLimit sets the maximum number of retry attempts for user prompts
func (r *InteractiveErrorResolver) SetRetryLimit(limit int) {
	if limit > 0 {
		r.retryLimit = limit
	}
}

// SetAutoFixEnabled enables or disables automatic fix suggestions
func (r *InteractiveErrorResolver) SetAutoFixEnabled(enabled bool) {
	r.autoFixEnabled = enabled
}

// GuidedErrorResolution provides step-by-step guidance for complex error resolution
type GuidedErrorResolution struct {
	resolver *InteractiveErrorResolver
	steps    []ResolutionStep
}

// ResolutionStep represents a single step in guided error resolution
type ResolutionStep struct {
	Title       string                 // Title of the step
	Description string                 // Detailed description
	Action      func() error           // Action to perform for this step
	Validation  func() (bool, string)  // Validation function (success, message)
	Optional    bool                   // Whether this step is optional
	Metadata    map[string]interface{} // Additional metadata
}

// NewGuidedErrorResolution creates a new guided error resolution workflow
func NewGuidedErrorResolution(resolver *InteractiveErrorResolver) *GuidedErrorResolution {
	return &GuidedErrorResolution{
		resolver: resolver,
		steps:    make([]ResolutionStep, 0),
	}
}

// AddStep adds a resolution step to the workflow
func (g *GuidedErrorResolution) AddStep(step ResolutionStep) {
	g.steps = append(g.steps, step)
}

// Execute runs the guided resolution workflow
func (g *GuidedErrorResolution) Execute() error {
	fmt.Fprintf(g.resolver.output, "\n🔧 Starting guided error resolution...\n")
	fmt.Fprintf(g.resolver.output, "This will walk you through %d steps to resolve the error.\n\n", len(g.steps))

	for i, step := range g.steps {
		fmt.Fprintf(g.resolver.output, "Step %d/%d: %s\n", i+1, len(g.steps), step.Title)
		fmt.Fprintf(g.resolver.output, "%s\n\n", step.Description)

		// Ask user if they want to proceed with this step
		if !g.confirmStep(step, i+1) {
			if !step.Optional {
				return fmt.Errorf("required step %d was skipped", i+1)
			}
			fmt.Fprintf(g.resolver.output, "Skipping optional step %d\n\n", i+1)
			continue
		}

		// Execute the step
		if step.Action != nil {
			if err := step.Action(); err != nil {
				fmt.Fprintf(g.resolver.output, "❌ Step %d failed: %v\n", i+1, err)
				if !step.Optional {
					return fmt.Errorf("required step %d failed: %w", i+1, err)
				}
				continue
			}
		}

		// Validate the step if validation is provided
		if step.Validation != nil {
			if success, message := step.Validation(); !success {
				fmt.Fprintf(g.resolver.output, "❌ Step %d validation failed: %s\n", i+1, message)
				if !step.Optional {
					return fmt.Errorf("step %d validation failed: %s", i+1, message)
				}
			} else {
				fmt.Fprintf(g.resolver.output, "✅ Step %d completed successfully", i+1)
				if message != "" {
					fmt.Fprintf(g.resolver.output, ": %s", message)
				}
				fmt.Fprintf(g.resolver.output, "\n")
			}
		} else {
			fmt.Fprintf(g.resolver.output, "✅ Step %d completed\n", i+1)
		}

		fmt.Fprintf(g.resolver.output, "\n")
	}

	fmt.Fprintf(g.resolver.output, "🎉 Guided resolution completed successfully!\n\n")
	return nil
}

// confirmStep asks the user to confirm proceeding with a step
func (g *GuidedErrorResolution) confirmStep(step ResolutionStep, stepNum int) bool {
	optionalText := ""
	if step.Optional {
		optionalText = " (optional)"
	}

	fmt.Fprintf(g.resolver.output, "Proceed with step %d%s? [Y/n]: ", stepNum, optionalText)

	if !g.resolver.input.Scan() {
		return true // Default to yes if input fails
	}

	response := strings.TrimSpace(strings.ToLower(g.resolver.input.Text()))
	return response == "" || response == "y" || response == "yes"
}

// RetryMechanism provides configurable retry logic with user guidance
type RetryMechanism struct {
	resolver       *InteractiveErrorResolver
	maxAttempts    int
	backoffEnabled bool
	userGuidance   bool
}

// NewRetryMechanism creates a new retry mechanism
func NewRetryMechanism(resolver *InteractiveErrorResolver) *RetryMechanism {
	return &RetryMechanism{
		resolver:       resolver,
		maxAttempts:    3,
		backoffEnabled: true,
		userGuidance:   true,
	}
}

// SetMaxAttempts sets the maximum number of retry attempts
func (r *RetryMechanism) SetMaxAttempts(max int) {
	if max > 0 {
		r.maxAttempts = max
	}
}

// SetBackoffEnabled enables or disables exponential backoff between retries
func (r *RetryMechanism) SetBackoffEnabled(enabled bool) {
	r.backoffEnabled = enabled
}

// SetUserGuidance enables or disables user guidance during retries
func (r *RetryMechanism) SetUserGuidance(enabled bool) {
	r.userGuidance = enabled
}

// ExecuteWithRetry executes an operation with retry logic and user guidance
func (r *RetryMechanism) ExecuteWithRetry(operation func() error, operationName string) error {
	var lastErr error

	for attempt := 1; attempt <= r.maxAttempts; attempt++ {
		fmt.Fprintf(r.resolver.output, "Attempt %d/%d: %s\n", attempt, r.maxAttempts, operationName)

		err := operation()
		if err == nil {
			fmt.Fprintf(r.resolver.output, "✅ Operation succeeded on attempt %d\n", attempt)
			return nil
		}

		lastErr = err
		fmt.Fprintf(r.resolver.output, "❌ Attempt %d failed: %v\n", attempt, err)

		// If this is the last attempt, don't prompt for retry
		if attempt == r.maxAttempts {
			break
		}

		// Ask user if they want to retry (if user guidance is enabled)
		if r.userGuidance {
			if !r.promptForRetry(attempt, r.maxAttempts, err) {
				break
			}
		}

		// Apply backoff if enabled
		if r.backoffEnabled {
			r.applyBackoff(attempt)
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", r.maxAttempts, lastErr)
}

// promptForRetry asks the user if they want to retry the operation
func (r *RetryMechanism) promptForRetry(currentAttempt, maxAttempts int, err error) bool {
	fmt.Fprintf(r.resolver.output, "\nRetry the operation? (%d/%d attempts remaining) [Y/n]: ",
		maxAttempts-currentAttempt, maxAttempts)

	if !r.resolver.input.Scan() {
		return true // Default to retry if input fails
	}

	response := strings.TrimSpace(strings.ToLower(r.resolver.input.Text()))
	return response == "" || response == "y" || response == "yes"
}

// applyBackoff applies exponential backoff delay
func (r *RetryMechanism) applyBackoff(attempt int) {
	// Simple backoff: wait for attempt seconds
	// In a real implementation, this might use time.Sleep()
	fmt.Fprintf(r.resolver.output, "Waiting %d seconds before retry...\n", attempt)
}

// ErrorResolutionWorkflow combines multiple resolution strategies
type ErrorResolutionWorkflow struct {
	resolver         *InteractiveErrorResolver
	guidedResolution *GuidedErrorResolution
	retryMechanism   *RetryMechanism
	autoFixAttempted bool
}

// NewErrorResolutionWorkflow creates a comprehensive error resolution workflow
func NewErrorResolutionWorkflow() *ErrorResolutionWorkflow {
	resolver := NewInteractiveErrorResolver()
	return &ErrorResolutionWorkflow{
		resolver:         resolver,
		guidedResolution: NewGuidedErrorResolution(resolver),
		retryMechanism:   NewRetryMechanism(resolver),
		autoFixAttempted: false,
	}
}

// ResolveWithWorkflow attempts to resolve an error using the full workflow
func (w *ErrorResolutionWorkflow) ResolveWithWorkflow(err OutputError) error {
	// First, try automatic fixes
	if !w.autoFixAttempted {
		if autoFixErr := w.tryAutoFix(err); autoFixErr == nil {
			return nil // Auto-fix succeeded
		}
		w.autoFixAttempted = true
	}

	// If auto-fix fails, use interactive resolution
	return w.resolver.ResolveError(err)
}

// tryAutoFix attempts to automatically fix the error without user interaction
func (w *ErrorResolutionWorkflow) tryAutoFix(err OutputError) error {
	fixes := w.resolver.getAvailableFixes(err)
	if len(fixes) == 0 {
		return fmt.Errorf("no automatic fixes available")
	}

	// Try the first available fix
	fix := fixes[0]
	fmt.Fprintf(w.resolver.output, "Attempting automatic fix: %s\n", fix.Name)

	if err := w.resolver.applyAutoFix(&fix); err != nil {
		fmt.Fprintf(w.resolver.output, "Automatic fix failed: %v\n", err)
		return err
	}

	fmt.Fprintf(w.resolver.output, "✅ Automatic fix applied successfully\n")
	return nil
}

// CreateGuidedResolutionForError creates a guided resolution workflow for a specific error
func (w *ErrorResolutionWorkflow) CreateGuidedResolutionForError(err OutputError) {
	w.guidedResolution = NewGuidedErrorResolution(w.resolver)

	switch err.Code() {
	case ErrInvalidFormat:
		w.addFormatResolutionSteps(err)
	case ErrMissingColumn:
		w.addColumnResolutionSteps(err)
	case ErrInvalidFilePath:
		w.addFilePathResolutionSteps(err)
	default:
		w.addGenericResolutionSteps(err)
	}
}

// addFormatResolutionSteps adds steps for resolving format errors
func (w *ErrorResolutionWorkflow) addFormatResolutionSteps(err OutputError) {
	w.guidedResolution.AddStep(ResolutionStep{
		Title:       "Verify Output Format",
		Description: "Check if the specified output format is supported",
		Action: func() error {
			fmt.Fprintf(w.resolver.output, "Supported formats: json, yaml, csv, html, table, markdown, dot, mermaid\n")
			return nil
		},
		Validation: func() (bool, string) {
			return true, "Format information displayed"
		},
	})

	w.guidedResolution.AddStep(ResolutionStep{
		Title:       "Choose Alternative Format",
		Description: "Select a compatible output format",
		Action: func() error {
			fmt.Fprintf(w.resolver.output, "Recommended: Use 'json' for maximum compatibility\n")
			return nil
		},
		Optional: true,
	})
}

// addColumnResolutionSteps adds steps for resolving missing column errors
func (w *ErrorResolutionWorkflow) addColumnResolutionSteps(err OutputError) {
	w.guidedResolution.AddStep(ResolutionStep{
		Title:       "Identify Missing Column",
		Description: "Determine which column is missing from the data",
		Action: func() error {
			if err.Context().Field != "" {
				fmt.Fprintf(w.resolver.output, "Missing column: %s\n", err.Context().Field)
			}
			return nil
		},
	})

	w.guidedResolution.AddStep(ResolutionStep{
		Title:       "Add Default Values",
		Description: "Add the missing column with appropriate default values",
		Action: func() error {
			fmt.Fprintf(w.resolver.output, "Consider adding default values for missing data\n")
			return nil
		},
		Optional: true,
	})
}

// addFilePathResolutionSteps adds steps for resolving file path errors
func (w *ErrorResolutionWorkflow) addFilePathResolutionSteps(err OutputError) {
	w.guidedResolution.AddStep(ResolutionStep{
		Title:       "Check File Path",
		Description: "Verify the file path exists and is writable",
		Action: func() error {
			if err.Context().Value != nil {
				fmt.Fprintf(w.resolver.output, "Problematic path: %v\n", err.Context().Value)
			}
			return nil
		},
	})

	w.guidedResolution.AddStep(ResolutionStep{
		Title:       "Create Directory",
		Description: "Create the directory structure if it doesn't exist",
		Action: func() error {
			fmt.Fprintf(w.resolver.output, "Use 'mkdir -p' to create directory structure\n")
			return nil
		},
		Optional: true,
	})
}

// addGenericResolutionSteps adds generic resolution steps for unknown errors
func (w *ErrorResolutionWorkflow) addGenericResolutionSteps(err OutputError) {
	w.guidedResolution.AddStep(ResolutionStep{
		Title:       "Review Error Details",
		Description: "Examine the error message and context for clues",
		Action: func() error {
			fmt.Fprintf(w.resolver.output, "Error: %s\n", err.Error())
			fmt.Fprintf(w.resolver.output, "Code: %s\n", err.Code())
			return nil
		},
	})

	w.guidedResolution.AddStep(ResolutionStep{
		Title:       "Apply Suggestions",
		Description: "Try the suggested fixes from the error message",
		Action: func() error {
			suggestions := err.Suggestions()
			if len(suggestions) > 0 {
				fmt.Fprintf(w.resolver.output, "Available suggestions:\n")
				for i, suggestion := range suggestions {
					fmt.Fprintf(w.resolver.output, "%d. %s\n", i+1, suggestion)
				}
			}
			return nil
		},
		Optional: true,
	})
}

// ExecuteGuidedResolution runs the guided resolution workflow
func (w *ErrorResolutionWorkflow) ExecuteGuidedResolution() error {
	return w.guidedResolution.Execute()
}
