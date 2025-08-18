package output

import (
	"context"
	"fmt"
	"maps"
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

// Clone creates a deep copy of the TableContent
func (tc *TableContent) Clone() Content {
	// Deep copy records
	newRecords := make([]Record, len(tc.records))
	for i, record := range tc.records {
		newRecord := make(Record)
		maps.Copy(newRecord, record)
		newRecords[i] = newRecord
	}

	// Deep copy schema
	var newSchema *Schema
	if tc.schema != nil {
		newFields := make([]Field, len(tc.schema.Fields))
		copy(newFields, tc.schema.Fields)

		newKeyOrder := make([]string, len(tc.schema.keyOrder))
		copy(newKeyOrder, tc.schema.keyOrder)

		newSchema = &Schema{
			Fields:   newFields,
			keyOrder: newKeyOrder,
		}
	}

	return &TableContent{
		records: newRecords,
		schema:  newSchema,
	}
}

// Transform applies a transformation function to the table's records
func (tc *TableContent) Transform(fn TransformFunc) error {
	result, err := fn(tc.records)
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
