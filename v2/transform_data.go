package output

import (
	"context"
	"fmt"
	"time"
)

// DataTransformer operates on structured data before rendering
type DataTransformer interface {
	// Name returns the transformer name for identification
	Name() string

	// TransformData modifies structured content data
	TransformData(ctx context.Context, content Content, format string) (Content, error)

	// CanTransform checks if this transformer applies to the given content and format
	CanTransform(content Content, format string) bool

	// Priority determines transform order (lower = earlier)
	Priority() int

	// Describe returns a human-readable description for debugging
	Describe() string
}

// TransformContext carries metadata through the transformation pipeline
type TransformContext struct {
	Format   string
	Document *Document
	Metadata map[string]any
	Stats    TransformStats
}

// TransformStats tracks transformation metrics
type TransformStats struct {
	InputRecords  int
	OutputRecords int
	FilteredCount int
	Duration      time.Duration
	Operations    []OperationStat
}

// OperationStat tracks individual operation metrics
type OperationStat struct {
	Name             string
	Duration         time.Duration
	RecordsProcessed int
}

// TransformerAdapter wraps transformers for unified handling
type TransformerAdapter struct {
	transformer any
}

// NewTransformerAdapter creates a new adapter for any transformer type
func NewTransformerAdapter(transformer any) *TransformerAdapter {
	return &TransformerAdapter{transformer: transformer}
}

// IsDataTransformer checks if the wrapped transformer is a DataTransformer
func (ta *TransformerAdapter) IsDataTransformer() bool {
	_, ok := ta.transformer.(DataTransformer)
	return ok
}

// AsDataTransformer returns the transformer as a DataTransformer, or nil
func (ta *TransformerAdapter) AsDataTransformer() DataTransformer {
	if dt, ok := ta.transformer.(DataTransformer); ok {
		return dt
	}
	return nil
}

// AsByteTransformer returns the transformer as a byte Transformer, or nil
func (ta *TransformerAdapter) AsByteTransformer() Transformer {
	if bt, ok := ta.transformer.(Transformer); ok {
		return bt
	}
	return nil
}

// TransformableContent extends Content with transformation support
type TransformableContent interface {
	Content

	// Clone creates a deep copy for transformation
	Clone() Content

	// Transform applies a transformation function
	Transform(fn TransformFunc) error
}

// TransformFunc defines the transformation function signature
type TransformFunc func(data any) (any, error)

// Ensure TableContent implements TransformableContent
var _ TransformableContent = (*TableContent)(nil)

// Transform applies a transformation function to the table's records.
//
// The function receives a deep copy of the records, so mutations of the
// passed-in data never alias the table's internal state and a failed
// transformation leaves the table unchanged; the table is only updated
// when fn succeeds and returns a []Record.
//
// Tables attached to a document are sealed and cannot be transformed,
// because documents are immutable after Build() (T-1677). Call Clone()
// and transform the copy instead.
func (tc *TableContent) Transform(fn TransformFunc) error {
	if tc.sealed.Load() {
		return fmt.Errorf("cannot transform table content %q: it belongs to a document and documents are immutable after Build(); call Clone() and transform the copy instead", tc.title)
	}

	result, err := fn(tc.Records())
	if err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}

	records, ok := result.([]Record)
	if !ok {
		return fmt.Errorf("transformation must return []Record, got %T", result)
	}

	tc.records = records
	return nil
}

// sealContents recursively seals every TableContent reachable from content.
// The builder calls this when content is attached to a document: from that
// moment the document shares the exact content pointers and must remain
// immutable after Build(), so in-place mutation through Transform is
// refused (T-1677). Recursion covers tables nested in sections and
// collapsible sections, including sections the caller constructed directly
// (e.g. via NewCollapsibleSection or NewCollapsibleTable).
func sealContents(content Content) {
	switch c := content.(type) {
	case *TableContent:
		c.seal()
	case *SectionContent:
		for _, child := range c.contents {
			sealContents(child)
		}
	case *DefaultCollapsibleSection:
		for _, child := range c.content {
			sealContents(child)
		}
	}
}
