package output

import (
	"encoding/json"
	"fmt"
)

// CollapsibleFormatter creates a field formatter that produces collapsible values
// Takes summaryTemplate and detailFunc parameters (Requirement 2.4)
// Returns field formatter that produces CollapsibleValue instances
func CollapsibleFormatter(summaryTemplate string, detailFunc func(any) any, opts ...CollapsibleOption) func(any) any {
	return func(val any) any {
		// Prevent nested CollapsibleValues to avoid infinite loops (Requirement 11.5)
		if _, ok := val.(CollapsibleValue); ok {
			return val // Return CollapsibleValue as-is to prevent nesting
		}

		if detailFunc == nil {
			return val // Return original value unchanged (Requirement 2.5)
		}

		detail := detailFunc(val)
		if detail == nil || detail == val {
			return val // No collapsible needed
		}

		// Prevent creating CollapsibleValue with CollapsibleValue details (Requirement 11.5)
		if _, ok := detail.(CollapsibleValue); ok {
			return val // Return original value to prevent nesting
		}

		summary := fmt.Sprintf(summaryTemplate, val)
		return NewCollapsibleValue(summary, detail, opts...)
	}
}

// ErrorListFormatter creates a formatter for error arrays (Requirement 9.1, 9.2)
func ErrorListFormatter(opts ...CollapsibleOption) func(any) any {
	return func(val any) any {
		// Prevent nested CollapsibleValues to avoid infinite loops (Requirement 11.5)
		if _, ok := val.(CollapsibleValue); ok {
			return val // Return CollapsibleValue as-is to prevent nesting
		}

		var errors []string
		var count int

		switch v := val.(type) {
		case []string:
			if len(v) == 0 {
				return val // No collapsible needed for empty
			}
			errors = v
			count = len(v)
		case []error:
			if len(v) == 0 {
				return val
			}
			errors = make([]string, len(v))
			for i, err := range v {
				errors[i] = err.Error()
			}
			count = len(v)
		default:
			return val // Graceful fallback for incompatible data types (Requirement 9.6)
		}

		summary := fmt.Sprintf("%d errors (click to expand)", count)
		return NewCollapsibleValue(summary, errors, opts...)
	}
}

// FilePathFormatter creates a formatter for long file paths (Requirement 9.3, 9.4)
func FilePathFormatter(maxLength int, opts ...CollapsibleOption) func(any) any {
	return func(val any) any {
		// Prevent nested CollapsibleValues to avoid infinite loops (Requirement 11.5)
		if _, ok := val.(CollapsibleValue); ok {
			return val // Return CollapsibleValue as-is to prevent nesting
		}

		path, ok := val.(string)
		if !ok || len(path) <= maxLength {
			return val // No collapsible needed for short paths or non-strings
		}

		// Create abbreviated path for summary
		var summary string
		if len(path) > 20 {
			summary = fmt.Sprintf("...%s (show full path)", path[len(path)-20:])
		} else {
			summary = fmt.Sprintf("...%s (show full path)", path)
		}

		return NewCollapsibleValue(summary, path, opts...)
	}
}

// JSONFormatter creates a formatter for complex data structures (Requirement 9.5)
func JSONFormatter(maxLength int, opts ...CollapsibleOption) func(any) any {
	return func(val any) any {
		// Prevent nested CollapsibleValues to avoid infinite loops (Requirement 11.5)
		if _, ok := val.(CollapsibleValue); ok {
			return val // Return CollapsibleValue as-is to prevent nesting
		}

		jsonBytes, err := json.MarshalIndent(val, "", "  ")
		if err != nil || len(jsonBytes) <= maxLength {
			return val // No collapsible needed for small data or marshal errors
		}

		summary := fmt.Sprintf("JSON data (%d bytes)", len(jsonBytes))
		return NewCollapsibleValue(summary, string(jsonBytes), opts...)
	}
}
