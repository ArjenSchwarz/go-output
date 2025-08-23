package output

import (
	"context"
	"fmt"
	"maps"
	"time"
)

// Pipeline provides fluent API for data transformations
type Pipeline struct {
	document   *Document
	operations []Operation
	options    PipelineOptions
}

// Operation represents a pipeline operation
type Operation interface {
	Name() string
	Apply(ctx context.Context, content Content) (Content, error)
	CanOptimize(with Operation) bool
	Validate() error
}

// PipelineOptions configures pipeline behavior
type PipelineOptions struct {
	MaxOperations    int           // Maximum number of operations allowed (default: 100)
	MaxExecutionTime time.Duration // Maximum execution time (default: 30s)
}

// DefaultPipelineOptions returns default pipeline options
func DefaultPipelineOptions() PipelineOptions {
	return PipelineOptions{
		MaxOperations:    100,
		MaxExecutionTime: 30 * time.Second,
	}
}

// Pipeline creates a new pipeline for transforming this document
func (d *Document) Pipeline() *Pipeline {
	return &Pipeline{
		document:   d,
		operations: make([]Operation, 0),
		options:    DefaultPipelineOptions(),
	}
}

// WithOptions sets custom pipeline options
func (p *Pipeline) WithOptions(opts PipelineOptions) *Pipeline {
	p.options = opts
	return p
}

// Filter adds a filter operation to the pipeline
// The predicate function should return true for records to keep
func (p *Pipeline) Filter(predicate func(Record) bool) *Pipeline {
	filterOp := NewFilterOp(predicate)
	p.operations = append(p.operations, filterOp)
	return p
}

// Sort adds a sort operation to the pipeline
// Accepts one or more sort keys with column names and directions
func (p *Pipeline) Sort(keys ...SortKey) *Pipeline {
	sortOp := NewSortOp(keys...)
	p.operations = append(p.operations, sortOp)
	return p
}

// SortBy adds a sort operation to the pipeline with a single column
// This is a convenience method for sorting by one column
func (p *Pipeline) SortBy(column string, direction SortDirection) *Pipeline {
	return p.Sort(SortKey{Column: column, Direction: direction})
}

// SortWith adds a sort operation using a custom comparator function
// The comparator should return -1 if a < b, 0 if a == b, 1 if a > b
func (p *Pipeline) SortWith(comparator func(a, b Record) int) *Pipeline {
	sortOp := NewSortOpWithComparator(comparator)
	p.operations = append(p.operations, sortOp)
	return p
}

// Limit adds a limit operation to the pipeline
// Returns the first N records from the data
func (p *Pipeline) Limit(count int) *Pipeline {
	limitOp := NewLimitOp(count)
	p.operations = append(p.operations, limitOp)
	return p
}

// GroupBy adds a groupBy operation to the pipeline
// Groups records by the specified columns and applies aggregate functions
func (p *Pipeline) GroupBy(columns []string, aggregates map[string]AggregateFunc) *Pipeline {
	groupByOp := NewGroupByOp(columns, aggregates)
	p.operations = append(p.operations, groupByOp)
	return p
}

// Validate checks if pipeline operations can be applied
func (p *Pipeline) Validate() error {
	// Check if document has transformable content
	hasTableContent := false
	for _, content := range p.document.GetContents() {
		if content.Type() == ContentTypeTable {
			hasTableContent = true
			break
		}
	}
	if !hasTableContent {
		return NewValidationError("document", p.document, "pipeline operations require table content")
	}

	// Check operation count limit
	if p.options.MaxOperations > 0 && len(p.operations) > p.options.MaxOperations {
		return NewValidationError("operations", len(p.operations),
			fmt.Sprintf("pipeline exceeds maximum operations limit: %d > %d",
				len(p.operations), p.options.MaxOperations))
	}

	// Validate operation compatibility
	for i, op := range p.operations {
		if err := op.Validate(); err != nil {
			return NewContextError("operation_validation", err).
				AddContext("operation_name", op.Name()).
				AddContext("operation_index", i).
				AddContext("operation_type", fmt.Sprintf("%T", op))
		}

		// Check if operation can be optimized with previous operation
		// Note: actual optimization would happen during execution
		// This is just validation that operations are compatible
	}

	return nil
}

// Execute runs the pipeline and returns a new transformed document
func (p *Pipeline) Execute() (*Document, error) {
	return p.ExecuteContext(context.Background())
}

// ExecuteContext runs the pipeline with a context and returns a new transformed document
func (p *Pipeline) ExecuteContext(ctx context.Context) (*Document, error) {
	// Apply timeout if configured
	if p.options.MaxExecutionTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.options.MaxExecutionTime)
		defer cancel()
	}

	// Validate first
	if err := p.Validate(); err != nil {
		return nil, NewContextError("pipeline_execution", err).
			AddContext("operation_count", len(p.operations)).
			AddContext("max_operations", p.options.MaxOperations)
	}

	// Start tracking stats
	stats := TransformStats{
		InputRecords: 0,
		Operations:   make([]OperationStat, 0, len(p.operations)),
	}
	startTime := time.Now()

	// Process contents
	newContents := make([]Content, 0, len(p.document.GetContents()))

	for contentIndex, content := range p.document.GetContents() {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, NewCancelledError("pipeline_execution", ctx.Err())
		default:
		}

		if content.Type() == ContentTypeTable {
			// Count input records
			if tableContent, ok := content.(*TableContent); ok {
				stats.InputRecords += len(tableContent.records)
			}

			// Apply transformations to table content
			transformed, err := p.applyOperations(ctx, content, &stats)
			if err != nil {
				// Wrap error with pipeline context
				return nil, NewContextError("content_transformation", err).
					AddContext("content_index", contentIndex).
					AddContext("content_type", content.Type().String()).
					AddContext("content_id", content.ID())
			}

			// Count output records
			if tableContent, ok := transformed.(*TableContent); ok {
				stats.OutputRecords += len(tableContent.records)
			}

			newContents = append(newContents, transformed)
		} else {
			// Pass through non-table content unchanged
			newContents = append(newContents, content)
		}
	}

	// Calculate final stats
	stats.Duration = time.Since(startTime)
	stats.FilteredCount = stats.InputRecords - stats.OutputRecords

	// Create new document with transformed contents
	newDoc := createDocumentWithContents(newContents, p.document.GetMetadata())

	// Add transformation stats to metadata
	metadata := newDoc.GetMetadata()
	metadata["transform_stats"] = stats

	return newDoc, nil
}

// applyOperations applies all pipeline operations to the content
func (p *Pipeline) applyOperations(ctx context.Context, content Content, stats *TransformStats) (Content, error) {
	// Start with a clone to preserve immutability
	current := content
	if transformable, ok := content.(TransformableContent); ok {
		current = transformable.Clone()
	}

	// Apply each operation in sequence
	for i, op := range p.operations {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, NewCancelledError("pipeline_execution", ctx.Err())
		default:
		}

		opStart := time.Now()

		// Apply the operation
		result, err := op.Apply(ctx, current)
		if err != nil {
			// Create detailed error with context
			return nil, NewContextError("operation_execution", err).
				AddContext("operation_name", op.Name()).
				AddContext("operation_index", i).
				AddContext("operation_type", fmt.Sprintf("%T", op)).
				AddContext("content_type", current.Type().String())
		}

		// Track operation stats
		opStat := OperationStat{
			Name:     op.Name(),
			Duration: time.Since(opStart),
		}
		if tableContent, ok := result.(*TableContent); ok {
			opStat.RecordsProcessed = len(tableContent.records)
		}
		stats.Operations = append(stats.Operations, opStat)

		current = result
	}

	return current, nil
}

// createDocumentWithContents creates a new document with the given contents and metadata
func createDocumentWithContents(contents []Content, metadata map[string]any) *Document {
	doc := &Document{
		contents: contents,
		metadata: make(map[string]any),
	}

	// Copy metadata
	if len(metadata) > 0 {
		maps.Copy(doc.metadata, metadata)
	}

	return doc
}

// optimize attempts to optimize the pipeline operations before execution
// This is currently unused but reserved for future optimization implementation
// nolint:unused // Reserved for future use
func (p *Pipeline) optimize() {
	// This is a placeholder for future optimization logic
	// For now, we'll implement basic filter-before-sort optimization

	optimized := make([]Operation, 0, len(p.operations))
	filters := make([]Operation, 0)
	others := make([]Operation, 0)

	// Separate filters from other operations
	for _, op := range p.operations {
		// Check if this is a filter operation by name (type-agnostic approach)
		if op.Name() == "Filter" {
			filters = append(filters, op)
		} else {
			others = append(others, op)
		}
	}

	// Apply filters first (they reduce data size)
	optimized = append(optimized, filters...)
	optimized = append(optimized, others...)

	p.operations = optimized
}

// PipelineResult encapsulates transformation results
type PipelineResult struct {
	Document *Document
	Stats    TransformStats
	Errors   []error
}

// GetTransformStats returns the transformation statistics from the document
func (d *Document) GetTransformStats() (TransformStats, bool) {
	metadata := d.GetMetadata()
	if stats, ok := metadata["transform_stats"].(TransformStats); ok {
		return stats, true
	}
	return TransformStats{}, false
}
