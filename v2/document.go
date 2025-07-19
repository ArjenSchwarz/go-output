package output

import (
	"fmt"
	"sync"
)

// Document represents a collection of content to be output
type Document struct {
	contents []Content
	metadata map[string]any
	mu       sync.RWMutex // For thread-safety
}

// GetContents returns a copy of the document's contents to prevent external modification
func (d *Document) GetContents() []Content {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy of the slice to prevent external modification
	contents := make([]Content, len(d.contents))
	copy(contents, d.contents)
	return contents
}

// GetMetadata returns a copy of the document's metadata
func (d *Document) GetMetadata() map[string]any {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy of the map to prevent external modification
	metadata := make(map[string]any, len(d.metadata))
	for k, v := range d.metadata {
		metadata[k] = v
	}
	return metadata
}

// Builder constructs documents with a fluent API
type Builder struct {
	doc    *Document
	mu     sync.Mutex // Ensure thread-safe building
	errors []error    // Track errors during building
}

// New creates a new document builder
func New() *Builder {
	return &Builder{
		doc: &Document{
			metadata: make(map[string]any),
		},
	}
}

// Build finalizes and returns the document
func (b *Builder) Build() *Document {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Return the document, preventing further modifications through this builder
	doc := b.doc
	b.doc = nil // Clear the reference to prevent further modifications
	return doc
}

// HasErrors returns true if any errors occurred during building
func (b *Builder) HasErrors() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.errors) > 0
}

// Errors returns a copy of all errors that occurred during building
func (b *Builder) Errors() []error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.errors) == 0 {
		return nil
	}

	errors := make([]error, len(b.errors))
	copy(errors, b.errors)
	return errors
}

// addError safely adds an error to the builder
func (b *Builder) addError(err error) {
	if err != nil {
		b.errors = append(b.errors, err)
	}
}

// SetMetadata sets a metadata key-value pair
func (b *Builder) SetMetadata(key string, value any) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.doc != nil {
		b.doc.metadata[key] = value
	}
	return b
}

// AddContent is a helper method to safely add content
func (b *Builder) AddContent(content Content) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.doc != nil {
		b.doc.contents = append(b.doc.contents, content)
	}
	return b
}

// Table adds a table with preserved key ordering
func (b *Builder) Table(title string, data any, opts ...TableOption) *Builder {
	table, err := NewTableContent(title, data, opts...)
	if err != nil {
		b.mu.Lock()
		b.addError(fmt.Errorf("failed to create table %q: %w", title, err))
		b.mu.Unlock()
		return b
	}
	return b.AddContent(table)
}

// Text adds text content with optional styling
func (b *Builder) Text(text string, opts ...TextOption) *Builder {
	textContent := NewTextContent(text, opts...)
	return b.AddContent(textContent)
}

// Header adds a header text (for v1 compatibility)
func (b *Builder) Header(text string) *Builder {
	return b.Text(text, WithHeader(true))
}

// Raw adds format-specific raw content
func (b *Builder) Raw(format string, data []byte, opts ...RawOption) *Builder {
	rawContent, err := NewRawContent(format, data, opts...)
	if err != nil {
		b.mu.Lock()
		b.addError(fmt.Errorf("failed to create raw content with format %q: %w", format, err))
		b.mu.Unlock()
		return b
	}
	return b.AddContent(rawContent)
}

// Section groups content under a heading with hierarchical structure
func (b *Builder) Section(title string, fn func(*Builder), opts ...SectionOption) *Builder {
	section := NewSectionContent(title, opts...)

	// Create sub-builder for section contents
	subBuilder := &Builder{doc: &Document{metadata: make(map[string]any)}}
	fn(subBuilder)

	// Add all contents from sub-builder to this section
	subDoc := subBuilder.Build()
	for _, content := range subDoc.GetContents() {
		section.AddContent(content)
	}

	return b.AddContent(section)
}
