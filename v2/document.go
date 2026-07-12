package output

import (
	"fmt"
	"maps"
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
	maps.Copy(metadata, d.metadata)
	return metadata
}

// Builder constructs documents with a fluent API.
//
// Fluent methods never return errors mid-chain. Failures — such as a Table or
// Raw constructor error, or any mutation after Build() — are recorded on the
// builder and the offending content is omitted from the document. A document
// with recorded errors still builds and renders successfully with that
// content missing, so check HasErrors or Errors after building:
//
//	builder := output.New().Table("data", rows).Text("note")
//	doc := builder.Build()
//	if builder.HasErrors() {
//		// handle builder.Errors()
//	}
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

// Build finalizes and returns the document.
//
// Build is one-shot: it clears the builder's document reference to enforce
// immutability. Any later mutation (SetMetadata, AddContent, or the content
// helpers that route through them) records a builder error and leaves the
// built document unchanged; a second Build call returns nil. Errors
// accumulated during building remain available via HasErrors and Errors —
// check them after Build, because content that failed to construct is
// silently absent from the returned document.
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

// SetMetadata sets a metadata key-value pair.
//
// Calling SetMetadata after Build() records a builder error and leaves the
// built document unchanged (T-1689).
func (b *Builder) SetMetadata(key string, value any) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.doc == nil {
		b.addError(fmt.Errorf("SetMetadata %q: cannot set metadata after Build() has been called", key))
		return b
	}
	b.doc.metadata[key] = value
	return b
}

// AddContent is a helper method to safely add content.
//
// Nil content is rejected: it is not appended to the document and a builder
// error is recorded instead (consistent with Table and Raw). This prevents nil
// entries in the document's contents, which would otherwise cause nil
// dereferences during rendering and transformation.
//
// Calling AddContent after Build() records a builder error and leaves the
// built document unchanged (T-1689).
func (b *Builder) AddContent(content Content) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()

	if content == nil {
		b.addError(fmt.Errorf("AddContent: cannot add nil content"))
		return b
	}

	if b.doc == nil {
		b.addError(fmt.Errorf("AddContent: cannot add content after Build() has been called"))
		return b
	}

	// The document shares these exact content pointers and must remain
	// immutable after Build(), so seal the content against in-place
	// mutation through Transform (T-1677).
	sealContents(content)

	b.doc.contents = append(b.doc.contents, content)
	return b
}

// Table adds a table with preserved key ordering.
//
// If table construction fails, the error is recorded on the builder, no
// content is added, and the chain continues; the built document renders
// without the failed table. Check HasErrors or Errors after building.
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

// Raw adds format-specific raw content.
//
// If content construction fails, the error is recorded on the builder, no
// content is added, and the chain continues; the built document renders
// without the failed content. Check HasErrors or Errors after building.
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

	// A nil callback is invalid input: record a builder error and skip invoking
	// it. The section is still added as an empty section, keeping the fluent API
	// non-panicking and consistent with the builder's error-accumulation pattern.
	if fn == nil {
		b.mu.Lock()
		b.addError(fmt.Errorf("Section %q: callback function is nil", title))
		b.mu.Unlock()
	} else {
		fn(subBuilder)
	}

	if subErrors := subBuilder.Errors(); len(subErrors) > 0 {
		b.mu.Lock()
		for _, err := range subErrors {
			b.addError(err)
		}
		b.mu.Unlock()
	}

	// Add all contents from sub-builder to this section.
	//
	// The callback may have already finalized the sub-builder by calling its
	// Build() method, in which case the sub-builder's document is nil and our
	// Build() call returns nil. Guard against that to avoid a nil dereference,
	// recording a builder error so callers can detect the misuse. The section is
	// still added (as an empty section), keeping the fluent API non-panicking.
	subDoc := subBuilder.Build()
	if subDoc == nil {
		b.mu.Lock()
		b.addError(fmt.Errorf("Section %q: callback finalized the sub-builder by calling Build()", title))
		b.mu.Unlock()
	} else {
		for _, content := range subDoc.GetContents() {
			section.AddContent(content)
		}
	}

	return b.AddContent(section)
}

// Graph adds graph content with edges
func (b *Builder) Graph(title string, edges []Edge) *Builder {
	graphContent := NewGraphContent(title, edges)
	return b.AddContent(graphContent)
}

// Chart adds a generic chart content
func (b *Builder) Chart(title, chartType string, data any) *Builder {
	chartContent := NewChartContent(title, chartType, data)
	return b.AddContent(chartContent)
}

// GanttChart adds a Gantt chart with tasks
func (b *Builder) GanttChart(title string, tasks []GanttTask) *Builder {
	chartContent := NewGanttChart(title, tasks)
	return b.AddContent(chartContent)
}

// PieChart adds a pie chart with slices
func (b *Builder) PieChart(title string, slices []PieSlice, showData bool) *Builder {
	chartContent := NewPieChart(title, slices, showData)
	return b.AddContent(chartContent)
}

// DrawIO adds Draw.io diagram content with CSV configuration
func (b *Builder) DrawIO(title string, records []Record, header DrawIOHeader, opts ...DrawIOOption) *Builder {
	drawioContent := NewDrawIOContent(title, records, header, opts...)
	return b.AddContent(drawioContent)
}

// AddCollapsibleSection adds a collapsible section containing the provided content
func (b *Builder) AddCollapsibleSection(title string, content []Content, opts ...CollapsibleSectionOption) *Builder {
	section := NewCollapsibleSection(title, content, opts...)
	return b.AddContent(section)
}

// AddCollapsibleTable adds a collapsible section containing a single table
func (b *Builder) AddCollapsibleTable(title string, table *TableContent, opts ...CollapsibleSectionOption) *Builder {
	section := NewCollapsibleTable(title, table, opts...)
	return b.AddContent(section)
}

// CollapsibleSection groups content under an expandable/collapsible section with hierarchical structure
func (b *Builder) CollapsibleSection(title string, fn func(*Builder), opts ...CollapsibleSectionOption) *Builder {
	// Create sub-builder for section contents
	subBuilder := &Builder{doc: &Document{metadata: make(map[string]any)}}

	// A nil callback is invalid input: record a builder error and skip invoking
	// it. The section is still added as an empty section, keeping the fluent API
	// non-panicking and consistent with the builder's error-accumulation pattern.
	if fn == nil {
		b.mu.Lock()
		b.addError(fmt.Errorf("CollapsibleSection %q: callback function is nil", title))
		b.mu.Unlock()
	} else {
		fn(subBuilder)
	}

	if subErrors := subBuilder.Errors(); len(subErrors) > 0 {
		b.mu.Lock()
		for _, err := range subErrors {
			b.addError(err)
		}
		b.mu.Unlock()
	}

	// Add all contents from sub-builder to collapsible section.
	//
	// The callback may have already finalized the sub-builder by calling its
	// Build() method, in which case the sub-builder's document is nil and our
	// Build() call returns nil. Guard against that to avoid a nil dereference,
	// recording a builder error so callers can detect the misuse. The section is
	// still added (as an empty section), keeping the fluent API non-panicking.
	var contents []Content
	if subDoc := subBuilder.Build(); subDoc != nil {
		contents = subDoc.GetContents()
	} else {
		b.mu.Lock()
		b.addError(fmt.Errorf("CollapsibleSection %q: callback finalized the sub-builder by calling Build()", title))
		b.mu.Unlock()
	}

	section := NewCollapsibleSection(title, contents, opts...)
	return b.AddContent(section)
}
