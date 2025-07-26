package output

import (
	"fmt"
)

// CollapsibleValue represents a value that can be expanded/collapsed across formats
// This interface enables table cells and content to display summary information
// with expandable details, working consistently across all output formats.
type CollapsibleValue interface {
	// Summary returns the collapsed view (what users see initially)
	Summary() string

	// Details returns the expanded content of any type to support structured data
	Details() any

	// IsExpanded returns whether this should be expanded by default
	IsExpanded() bool

	// FormatHint provides renderer-specific hints
	FormatHint(format string) map[string]any
}

// DefaultCollapsibleValue provides a standard implementation of CollapsibleValue
type DefaultCollapsibleValue struct {
	summary         string
	details         any
	defaultExpanded bool
	formatHints     map[string]map[string]any

	// Configuration for truncation (requirements: configurable with 500 default)
	maxDetailLength   int
	truncateIndicator string
}

// CollapsibleOption configures a DefaultCollapsibleValue
type CollapsibleOption func(*DefaultCollapsibleValue)

// NewCollapsibleValue creates a new collapsible value with configuration options
func NewCollapsibleValue(summary string, details any, opts ...CollapsibleOption) *DefaultCollapsibleValue {
	cv := &DefaultCollapsibleValue{
		summary:           summary,
		details:           details,
		defaultExpanded:   false,
		maxDetailLength:   500, // Default from requirements
		truncateIndicator: "[...truncated]",
		formatHints:       make(map[string]map[string]any),
	}

	for _, opt := range opts {
		opt(cv)
	}

	return cv
}

// WithExpanded sets whether the collapsible value should be expanded by default
func WithExpanded(expanded bool) CollapsibleOption {
	return func(cv *DefaultCollapsibleValue) {
		cv.defaultExpanded = expanded
	}
}

// WithMaxLength sets the maximum character length for details before truncation
func WithMaxLength(length int) CollapsibleOption {
	return func(cv *DefaultCollapsibleValue) {
		cv.maxDetailLength = length
	}
}

// WithTruncateIndicator sets the indicator text used when content is truncated
func WithTruncateIndicator(indicator string) CollapsibleOption {
	return func(cv *DefaultCollapsibleValue) {
		cv.truncateIndicator = indicator
	}
}

// WithFormatHint adds format-specific rendering hints
func WithFormatHint(format string, hints map[string]any) CollapsibleOption {
	return func(cv *DefaultCollapsibleValue) {
		if cv.formatHints == nil {
			cv.formatHints = make(map[string]map[string]any)
		}
		cv.formatHints[format] = hints
	}
}

// Summary returns the collapsed view with fallback handling
func (d *DefaultCollapsibleValue) Summary() string {
	if d.summary == "" {
		return "[no summary]" // Requirement: default placeholder
	}
	return d.summary
}

// Details returns the expanded content with character limit truncation
func (d *DefaultCollapsibleValue) Details() any {
	if d.details == nil {
		return d.summary // Fallback for nil details
	}

	// Apply character limit truncation if configured
	if d.maxDetailLength > 0 {
		if detailStr, ok := d.details.(string); ok && len(detailStr) > d.maxDetailLength {
			return detailStr[:d.maxDetailLength] + d.truncateIndicator
		}
	}

	return d.details
}

// IsExpanded returns whether this should be expanded by default
func (d *DefaultCollapsibleValue) IsExpanded() bool {
	return d.defaultExpanded
}

// FormatHint returns renderer-specific hints for the given format
func (d *DefaultCollapsibleValue) FormatHint(format string) map[string]any {
	if hints, exists := d.formatHints[format]; exists {
		return hints
	}
	return nil
}

// String implements the Stringer interface for debugging
func (d *DefaultCollapsibleValue) String() string {
	return fmt.Sprintf("CollapsibleValue{summary: %q, expanded: %t}", d.summary, d.defaultExpanded)
}
