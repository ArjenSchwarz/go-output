package output

import (
	"fmt"
)

// CollapsibleSection represents entire content blocks that can be expanded/collapsed (Requirement 15)
type CollapsibleSection interface {
	Content // Implement the Content interface to integrate with the content system

	// Title returns the section title/summary
	Title() string

	// Content returns the nested content items
	Content() []Content

	// IsExpanded returns whether this section should be expanded by default
	IsExpanded() bool

	// Level returns the nesting level (0-3 supported per Requirement 15.9)
	Level() int

	// FormatHint provides renderer-specific hints
	FormatHint(format string) map[string]any
}

// DefaultCollapsibleSection provides a standard implementation for section-level collapsible content
type DefaultCollapsibleSection struct {
	id              string
	title           string
	content         []Content
	defaultExpanded bool
	level           int
	formatHints     map[string]map[string]any
}

// NewCollapsibleSection creates a new collapsible section
func NewCollapsibleSection(title string, content []Content, opts ...CollapsibleSectionOption) *DefaultCollapsibleSection {
	cs := &DefaultCollapsibleSection{
		id:              GenerateID(),
		title:           title,
		content:         content,
		defaultExpanded: false,
		level:           0,
		formatHints:     make(map[string]map[string]any),
	}

	for _, opt := range opts {
		opt(cs)
	}

	return cs
}

// CollapsibleSectionOption is a functional option for CollapsibleSection configuration
type CollapsibleSectionOption func(*DefaultCollapsibleSection)

// WithSectionExpanded sets whether the section should be expanded by default
func WithSectionExpanded(expanded bool) CollapsibleSectionOption {
	return func(cs *DefaultCollapsibleSection) {
		cs.defaultExpanded = expanded
	}
}

// WithSectionLevel sets the nesting level (0-3 supported per Requirement 15.9)
func WithSectionLevel(level int) CollapsibleSectionOption {
	return func(cs *DefaultCollapsibleSection) {
		// Limit to 3 levels per Requirement 15.9
		if level >= 0 && level <= 3 {
			cs.level = level
		}
	}
}

// WithSectionFormatHint adds format-specific hints for the section
func WithSectionFormatHint(format string, hints map[string]any) CollapsibleSectionOption {
	return func(cs *DefaultCollapsibleSection) {
		cs.formatHints[format] = hints
	}
}

// Title returns the section title/summary
func (cs *DefaultCollapsibleSection) Title() string {
	if cs.title == "" {
		return "[untitled section]"
	}
	return cs.title
}

// Content returns the nested content items
func (cs *DefaultCollapsibleSection) Content() []Content {
	// Return copy to prevent external modification
	content := make([]Content, len(cs.content))
	copy(content, cs.content)
	return content
}

// IsExpanded returns whether this section should be expanded by default
func (cs *DefaultCollapsibleSection) IsExpanded() bool {
	return cs.defaultExpanded
}

// Level returns the nesting level (0-3 supported per Requirement 15.9)
func (cs *DefaultCollapsibleSection) Level() int {
	return cs.level
}

// FormatHint provides renderer-specific hints
func (cs *DefaultCollapsibleSection) FormatHint(format string) map[string]any {
	if hints, exists := cs.formatHints[format]; exists {
		return hints
	}
	return nil
}

// Content interface implementation

// Type returns the content type for CollapsibleSection
func (cs *DefaultCollapsibleSection) Type() ContentType {
	return ContentTypeSection // Reuse the existing section content type
}

// ID returns the unique identifier for this content
func (cs *DefaultCollapsibleSection) ID() string {
	return cs.id
}

// AppendText implements encoding.TextAppender with collapsible section rendering
func (cs *DefaultCollapsibleSection) AppendText(b []byte) ([]byte, error) {
	// Basic text representation - this will be overridden by specific renderers
	// Add section title
	if cs.title != "" {
		b = append(b, cs.title...)
		b = append(b, '\n')
		if len(cs.content) > 0 {
			b = append(b, '\n')
		}
	}

	// Render all nested content
	for i, content := range cs.content {
		if i > 0 {
			b = append(b, '\n')
		}

		contentBytes, err := content.AppendText(nil)
		if err != nil {
			return b, fmt.Errorf("failed to render nested content %s: %w", content.ID(), err)
		}

		b = append(b, contentBytes...)
	}

	return b, nil
}

// AppendBinary implements encoding.BinaryAppender
func (cs *DefaultCollapsibleSection) AppendBinary(b []byte) ([]byte, error) {
	// For binary append, we'll use the same text representation
	return cs.AppendText(b)
}

// Helper functions for creating collapsible sections

// NewCollapsibleTable creates a collapsible section containing a single table
func NewCollapsibleTable(title string, table *TableContent, opts ...CollapsibleSectionOption) *DefaultCollapsibleSection {
	return NewCollapsibleSection(title, []Content{table}, opts...)
}

// NewCollapsibleMultiTable creates a collapsible section containing multiple tables
func NewCollapsibleMultiTable(title string, tables []*TableContent, opts ...CollapsibleSectionOption) *DefaultCollapsibleSection {
	content := make([]Content, len(tables))
	for i, table := range tables {
		content[i] = table
	}
	return NewCollapsibleSection(title, content, opts...)
}

// NewCollapsibleReport creates a collapsible section with mixed content types
func NewCollapsibleReport(title string, content []Content, opts ...CollapsibleSectionOption) *DefaultCollapsibleSection {
	return NewCollapsibleSection(title, content, opts...)
}
