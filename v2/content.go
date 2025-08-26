package output

import (
	"crypto/rand"
	"encoding"
	"fmt"
	"maps"
)

const (
	// unknownValue is the string representation for unknown/default values
	unknownValue = "unknown"
)

// ContentType identifies the type of content
type ContentType int

const (
	// ContentTypeTable represents tabular data content
	ContentTypeTable ContentType = iota
	// ContentTypeText represents unstructured text content
	ContentTypeText
	// ContentTypeRaw represents format-specific raw content
	ContentTypeRaw
	// ContentTypeSection represents grouped content with a heading
	ContentTypeSection
)

// String returns the string representation of the ContentType
func (ct ContentType) String() string {
	switch ct {
	case ContentTypeTable:
		return "table"
	case ContentTypeText:
		return "text"
	case ContentTypeRaw:
		return "raw"
	case ContentTypeSection:
		return "section"
	default:
		return unknownValue
	}
}

// Content is the core interface all content must implement
type Content interface {
	// Type returns the content type
	Type() ContentType

	// ID returns a unique identifier for this content
	ID() string

	// Encoding interfaces for efficient serialization
	encoding.TextAppender
	encoding.BinaryAppender
}

// GenerateID creates a unique identifier for content
func GenerateID() string {
	// Generate 8 random bytes
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple counter if crypto/rand fails
		return fmt.Sprintf("content-%d", len(bytes))
	}
	return fmt.Sprintf("content-%x", bytes)
}

// Record is a single table row
type Record map[string]any

// TableContent represents tabular data with preserved key ordering
type TableContent struct {
	id      string
	title   string
	schema  *Schema
	records []Record
}

// NewTableContent creates a new table content with the given data and options
func NewTableContent(title string, data any, opts ...TableOption) (*TableContent, error) {
	tc := ApplyTableOptions(opts...)

	table := &TableContent{
		id:    GenerateID(),
		title: title,
	}

	// Determine schema based on options
	switch {
	case tc.schema != nil:
		table.schema = tc.schema
	case len(tc.keys) > 0:
		table.schema = NewSchemaFromKeys(tc.keys)
	case tc.autoSchema:
		table.schema = DetectSchemaFromData(data)
		if len(tc.keys) > 0 {
			table.schema.SetKeyOrder(tc.keys)
		}
	default:
		table.schema = DetectSchemaFromData(data)
	}

	// Convert data to records
	records, err := convertToRecords(data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert data to records: %w", err)
	}
	table.records = records

	return table, nil
}

// Type returns the content type
func (t *TableContent) Type() ContentType {
	return ContentTypeTable
}

// ID returns the unique identifier for this content
func (t *TableContent) ID() string {
	return t.id
}

// Title returns the table title
func (t *TableContent) Title() string {
	return t.title
}

// Schema returns the table schema
func (t *TableContent) Schema() *Schema {
	return t.schema
}

// Records returns the table records
func (t *TableContent) Records() []Record {
	// Return a copy to prevent external modification
	records := make([]Record, len(t.records))
	for i, record := range t.records {
		newRecord := make(Record, len(record))
		maps.Copy(newRecord, record)
		records[i] = newRecord
	}
	return records
}

// AppendText implements encoding.TextAppender preserving key order
func (t *TableContent) AppendText(b []byte) ([]byte, error) {
	keyOrder := t.schema.GetKeyOrder()

	// Add title if present
	if t.title != "" {
		b = append(b, t.title...)
		b = append(b, '\n')
	}

	// Headers in exact order from schema
	for i, key := range keyOrder {
		if i > 0 {
			b = append(b, '\t')
		}
		b = append(b, key...)
	}
	b = append(b, '\n')

	// Records with values in same key order
	for _, record := range t.records {
		for i, key := range keyOrder {
			if i > 0 {
				b = append(b, '\t')
			}
			if val, ok := record[key]; ok {
				field := t.schema.FindField(key)
				if field != nil && field.Formatter != nil {
					formatted := field.Formatter(val)
					b = append(b, formatValue(formatted)...)
				} else {
					b = append(b, formatValue(val)...)
				}
			}
		}
		b = append(b, '\n')
	}

	return b, nil
}

// AppendBinary implements encoding.BinaryAppender
func (t *TableContent) AppendBinary(b []byte) ([]byte, error) {
	// For binary append, we'll use the same text representation
	return t.AppendText(b)
}

// formatValue converts a value to its string representation
func formatValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprint(val)
}

// convertToRecords converts various data types to records
func convertToRecords(data any) ([]Record, error) {
	switch v := data.(type) {
	case []Record:
		return v, nil
	case []map[string]any:
		records := make([]Record, len(v))
		for i, m := range v {
			records[i] = Record(m)
		}
		return records, nil
	case map[string]any:
		return []Record{Record(v)}, nil
	case []any:
		records := make([]Record, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				records = append(records, Record(m))
			} else {
				return nil, fmt.Errorf("unsupported item type in slice: %T", item)
			}
		}
		return records, nil
	default:
		return nil, fmt.Errorf("unsupported data type: %T", data)
	}
}

// TextStyle defines text formatting options
type TextStyle struct {
	Bold   bool   // Bold text
	Italic bool   // Italic text
	Color  string // Text color (format-specific interpretation)
	Size   int    // Text size (format-specific interpretation)
	Header bool   // Whether this text is a header (for v1 AddHeader compatibility)
}

// TextContent represents unstructured text with styling options
type TextContent struct {
	id    string
	text  string
	style TextStyle
}

// NewTextContent creates a new text content with the given text and options
func NewTextContent(text string, opts ...TextOption) *TextContent {
	tc := ApplyTextOptions(opts...)

	return &TextContent{
		id:    GenerateID(),
		text:  text,
		style: tc.style,
	}
}

// Type returns the content type
func (t *TextContent) Type() ContentType {
	return ContentTypeText
}

// ID returns the unique identifier for this content
func (t *TextContent) ID() string {
	return t.id
}

// Text returns the text content
func (t *TextContent) Text() string {
	return t.text
}

// Style returns the text style
func (t *TextContent) Style() TextStyle {
	return t.style
}

// AppendText implements encoding.TextAppender
func (t *TextContent) AppendText(b []byte) ([]byte, error) {
	// For text content, we simply append the text
	// Styling will be handled by format-specific renderers
	b = append(b, t.text...)

	// Add newline if this is not already ending with one
	if len(b) > 0 && b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}

	return b, nil
}

// AppendBinary implements encoding.BinaryAppender
func (t *TextContent) AppendBinary(b []byte) ([]byte, error) {
	// For binary append, we'll use the same text representation
	return t.AppendText(b)
}

// RawContent represents format-specific content
type RawContent struct {
	id     string
	format string
	data   []byte
}

// NewRawContent creates a new raw content with the given format and data
func NewRawContent(format string, data []byte, opts ...RawOption) (*RawContent, error) {
	rc := ApplyRawOptions(opts...)

	// Validate format if validation is enabled
	if rc.validateFormat && !isValidFormat(format) {
		return nil, fmt.Errorf("invalid format: %s", format)
	}

	// Create a copy of the data to ensure immutability
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	return &RawContent{
		id:     GenerateID(),
		format: format,
		data:   dataCopy,
	}, nil
}

// Type returns the content type
func (r *RawContent) Type() ContentType {
	return ContentTypeRaw
}

// ID returns the unique identifier for this content
func (r *RawContent) ID() string {
	return r.id
}

// Format returns the format of the raw content
func (r *RawContent) Format() string {
	return r.format
}

// Data returns a copy of the raw data to prevent external modification
func (r *RawContent) Data() []byte {
	dataCopy := make([]byte, len(r.data))
	copy(dataCopy, r.data)
	return dataCopy
}

// AppendText implements encoding.TextAppender
func (r *RawContent) AppendText(b []byte) ([]byte, error) {
	// For raw content, we append the data as-is
	// Format-specific renderers will handle the content appropriately
	b = append(b, r.data...)
	return b, nil
}

// AppendBinary implements encoding.BinaryAppender
func (r *RawContent) AppendBinary(b []byte) ([]byte, error) {
	// For binary append, we append the raw data as-is
	b = append(b, r.data...)
	return b, nil
}

// isValidFormat checks if a format string is valid
func isValidFormat(format string) bool {
	// Define known valid formats
	validFormats := map[string]bool{
		"html":     true,
		"css":      true,
		"js":       true,
		"json":     true,
		"xml":      true,
		"yaml":     true,
		"markdown": true,
		"text":     true,
		"csv":      true,
		"dot":      true,
		"mermaid":  true,
		"drawio":   true,
		"svg":      true,
	}

	return validFormats[format]
}

// SectionContent represents grouped content with a hierarchical structure
type SectionContent struct {
	id       string
	title    string
	level    int
	contents []Content
}

// NewSectionContent creates a new section content with the given title and options
func NewSectionContent(title string, opts ...SectionOption) *SectionContent {
	sc := ApplySectionOptions(opts...)

	return &SectionContent{
		id:       GenerateID(),
		title:    title,
		level:    sc.level,
		contents: make([]Content, 0),
	}
}

// Type returns the content type
func (s *SectionContent) Type() ContentType {
	return ContentTypeSection
}

// ID returns the unique identifier for this content
func (s *SectionContent) ID() string {
	return s.id
}

// Title returns the section title
func (s *SectionContent) Title() string {
	return s.title
}

// Level returns the section level (for hierarchical rendering)
func (s *SectionContent) Level() int {
	return s.level
}

// Contents returns a copy of the section contents to prevent external modification
func (s *SectionContent) Contents() []Content {
	contents := make([]Content, len(s.contents))
	copy(contents, s.contents)
	return contents
}

// AddContent adds content to this section
func (s *SectionContent) AddContent(content Content) {
	s.contents = append(s.contents, content)
}

// AppendText implements encoding.TextAppender with hierarchical rendering
func (s *SectionContent) AppendText(b []byte) ([]byte, error) {
	// Add section title with appropriate level indicators
	levelPrefix := generateLevelPrefix(s.level)
	if s.title != "" {
		b = append(b, levelPrefix...)
		b = append(b, s.title...)
		b = append(b, '\n')

		// Add separator line for sections with content
		if len(s.contents) > 0 {
			b = append(b, '\n')
		}
	}

	// Render all nested content
	for i, content := range s.contents {
		if i > 0 {
			b = append(b, '\n')
		}

		contentBytes, err := content.AppendText(nil)
		if err != nil {
			return b, fmt.Errorf("failed to render nested content %s: %w", content.ID(), err)
		}

		// Indent nested content if this is a subsection
		if s.level > 0 {
			indentedBytes := indentContent(contentBytes, s.level)
			b = append(b, indentedBytes...)
		} else {
			b = append(b, contentBytes...)
		}
	}

	return b, nil
}

// AppendBinary implements encoding.BinaryAppender
func (s *SectionContent) AppendBinary(b []byte) ([]byte, error) {
	// For binary append, we'll use the same text representation
	return s.AppendText(b)
}

// generateLevelPrefix creates appropriate prefixes for different section levels
func generateLevelPrefix(level int) string {
	switch level {
	case 0:
		return "# "
	case 1:
		return "## "
	case 2:
		return "### "
	case 3:
		return "#### "
	case 4:
		return "##### "
	case 5:
		return "###### "
	default:
		// For levels beyond 6, use repeated hashes
		prefix := ""
		for i := 0; i <= level; i++ {
			prefix += "#"
		}
		return prefix + " "
	}
}

// indentContent adds indentation to content based on section level
func indentContent(content []byte, level int) []byte {
	if level <= 0 || len(content) == 0 {
		return content
	}

	// Create indentation (2 spaces per level)
	indent := make([]byte, level*2)
	for i := range indent {
		indent[i] = ' '
	}

	var result []byte
	lines := splitLines(content)

	for i, line := range lines {
		if i > 0 {
			result = append(result, '\n')
		}

		// Only indent non-empty lines
		if len(line) > 0 {
			result = append(result, indent...)
		}
		result = append(result, line...)
	}

	return result
}

// splitLines splits content into lines while preserving line endings
func splitLines(content []byte) [][]byte {
	if len(content) == 0 {
		return [][]byte{}
	}

	var lines [][]byte
	start := 0

	for i := range content {
		if content[i] == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
		}
	}

	// Add the last line if it doesn't end with newline
	if start < len(content) {
		lines = append(lines, content[start:])
	}

	return lines
}
