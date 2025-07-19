package output

import (
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
	doc *Document
	mu  sync.Mutex // Ensure thread-safe building
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
