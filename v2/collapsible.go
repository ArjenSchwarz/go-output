package output

import (
	"fmt"
)

// Constants for repeated strings
const (
	defaultSummaryPlaceholder = "[no summary]"
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

	// Code fence configuration for wrapping details in code blocks
	codeLanguage  string // Language for syntax highlighting (e.g., "json", "yaml", "go")
	useCodeFences bool   // Whether to wrap details in code fences

	// Performance optimization fields for lazy evaluation (Requirement 10.3, 10.5)
	processedDetails any
	detailsProcessed bool
	hintsAccessed    map[string]bool
	memoryProcessor  *MemoryOptimizedProcessor
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
		// Initialize performance optimization fields (Requirements 10.3, 10.5)
		detailsProcessed: false,
		hintsAccessed:    nil, // Lazy initialized when first accessed
	}

	for _, opt := range opts {
		opt(cv)
	}

	return cv
}

// WithCollapsibleExpanded sets whether the collapsible value should be expanded by default
func WithCollapsibleExpanded(expanded bool) CollapsibleOption {
	return func(cv *DefaultCollapsibleValue) {
		cv.defaultExpanded = expanded
	}
}

// WithExpanded sets whether the collapsible value should be expanded by default
// Deprecated: Use WithCollapsibleExpanded instead for consistency with section options
func WithExpanded(expanded bool) CollapsibleOption {
	return WithCollapsibleExpanded(expanded)
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

// WithCodeFences enables wrapping details content in code fences
func WithCodeFences(language string) CollapsibleOption {
	return func(cv *DefaultCollapsibleValue) {
		cv.useCodeFences = true
		cv.codeLanguage = language
	}
}

// WithoutCodeFences explicitly disables code fence wrapping
func WithoutCodeFences() CollapsibleOption {
	return func(cv *DefaultCollapsibleValue) {
		cv.useCodeFences = false
		cv.codeLanguage = ""
	}
}

// Summary returns the collapsed view with fallback handling
func (d *DefaultCollapsibleValue) Summary() string {
	if d.summary == "" {
		return defaultSummaryPlaceholder // Requirement: default placeholder
	}
	return d.summary
}

// Details returns the expanded content with character limit truncation
// Implements lazy evaluation to avoid unnecessary processing (Requirement 10.3)
func (d *DefaultCollapsibleValue) Details() any {
	if d.details == nil {
		return d.summary // Fallback for nil details
	}

	// Use cached processed details if available (Requirement 10.3)
	if d.detailsProcessed {
		return d.processedDetails
	}

	// Initialize memory processor if needed for large content processing (Requirement 10.4)
	if d.memoryProcessor == nil && d.needsMemoryOptimization() {
		config := RendererConfig{
			MaxDetailLength:   d.maxDetailLength,
			TruncateIndicator: d.truncateIndicator,
		}
		d.memoryProcessor = NewMemoryOptimizedProcessor(config)
	}

	// Process details once and cache result
	var result any
	var err error

	// Use memory-optimized processing for large content (Requirement 10.4)
	if d.memoryProcessor != nil {
		result, err = d.memoryProcessor.ProcessLargeDetails(d.details, d.maxDetailLength)
		if err != nil {
			// Fallback to simple processing on error
			result = d.processDetailsSimple()
		}
	} else {
		result = d.processDetailsSimple()
	}

	// Cache the processed result to avoid redundant processing (Requirement 10.3)
	d.processedDetails = result
	d.detailsProcessed = true

	return result
}

// needsMemoryOptimization determines if memory optimization is beneficial
func (d *DefaultCollapsibleValue) needsMemoryOptimization() bool {
	if d.details == nil {
		return false
	}

	// Use memory optimization for large strings, arrays, or maps
	switch details := d.details.(type) {
	case string:
		return len(details) > 1000 // Optimize strings over 1KB
	case []string:
		return len(details) > 10 // Optimize arrays with many elements
	case map[string]any:
		return len(details) > 5 // Optimize maps with many keys
	default:
		return false
	}
}

// processDetailsSimple provides simple detail processing without memory optimization
func (d *DefaultCollapsibleValue) processDetailsSimple() any {
	result := d.details

	// Apply character limit truncation if configured
	if d.maxDetailLength > 0 {
		if detailStr, ok := d.details.(string); ok && len(detailStr) > d.maxDetailLength {
			result = detailStr[:d.maxDetailLength] + d.truncateIndicator
		}
	}

	return result
}

// IsExpanded returns whether this should be expanded by default
func (d *DefaultCollapsibleValue) IsExpanded() bool {
	return d.defaultExpanded
}

// FormatHint returns renderer-specific hints for the given format
// Implements lazy evaluation to avoid processing hints when not used (Requirement 10.5)
func (d *DefaultCollapsibleValue) FormatHint(format string) map[string]any {
	// Initialize hintsAccessed map if not already done
	if d.hintsAccessed == nil {
		d.hintsAccessed = make(map[string]bool)
	}

	// Mark this format as accessed for performance tracking
	d.hintsAccessed[format] = true

	if hints, exists := d.formatHints[format]; exists {
		return hints
	}
	return nil
}

// UseCodeFences returns whether details should be wrapped in code fences
func (d *DefaultCollapsibleValue) UseCodeFences() bool {
	return d.useCodeFences
}

// CodeLanguage returns the language to use for syntax highlighting in code fences
func (d *DefaultCollapsibleValue) CodeLanguage() string {
	return d.codeLanguage
}

// String implements the Stringer interface for debugging
func (d *DefaultCollapsibleValue) String() string {
	return fmt.Sprintf("CollapsibleValue{summary: %q, expanded: %t}", d.summary, d.defaultExpanded)
}
