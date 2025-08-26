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

// FormatAwareOperation extends Operation with format awareness
type FormatAwareOperation interface {
	Operation

	// ApplyWithFormat applies the operation with format context
	ApplyWithFormat(ctx context.Context, content Content, format string) (Content, error)

	// CanTransform checks if this operation applies to the given content and format
	CanTransform(content Content, format string) bool
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

// AddColumn adds a calculated field to the table
// The calculation function receives the full record and should return the calculated value
func (p *Pipeline) AddColumn(name string, fn func(Record) any) *Pipeline {
	addColumnOp := NewAddColumnOp(name, fn, nil) // nil position = append to end
	p.operations = append(p.operations, addColumnOp)
	return p
}

// AddColumnAt adds a calculated field at a specific position in the table
// The calculation function receives the full record and should return the calculated value
// Position 0 inserts at the beginning, position >= field count appends to end
func (p *Pipeline) AddColumnAt(name string, fn func(Record) any, position int) *Pipeline {
	addColumnOp := NewAddColumnOp(name, fn, &position)
	p.operations = append(p.operations, addColumnOp)
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
			validationErr := NewPipelineError("Validate", i, err)
			validationErr.AddContext("operation_name", op.Name())
			validationErr.AddContext("operation_type", fmt.Sprintf("%T", op))
			validationErr.AddPipelineContext("total_operations", len(p.operations))

			// Add specific validation context based on operation type
			switch op.(type) {
			case *FilterOp:
				validationErr.AddContext("validation_issue", "invalid predicate function")
			case *SortOp:
				validationErr.AddContext("validation_issue", "invalid sort configuration")
			case *LimitOp:
				validationErr.AddContext("validation_issue", "invalid limit value")
			case *GroupByOp:
				validationErr.AddContext("validation_issue", "invalid groupBy configuration")
			case *AddColumnOp:
				validationErr.AddContext("validation_issue", "invalid column configuration")
			default:
				validationErr.AddContext("validation_issue", "unknown operation validation failure")
			}

			return validationErr
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

// ExecuteWithFormat executes the pipeline transformations with format context
func (p *Pipeline) ExecuteWithFormat(ctx context.Context, format string) (*Document, error) {
	return p.ExecuteWithFormatContext(ctx, format)
}

// ExecuteWithFormatContext executes the pipeline with format context
func (p *Pipeline) ExecuteWithFormatContext(ctx context.Context, format string) (*Document, error) {
	// Validate format parameter
	if format == "" {
		return nil, NewPipelineError("ExecuteWithFormat", -1, fmt.Errorf("format cannot be empty"))
	}

	if !isValidFormat(format) {
		return nil, NewPipelineError("ExecuteWithFormat", -1, fmt.Errorf("invalid format: %s", format))
	}

	// Apply timeout if configured
	if p.options.MaxExecutionTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.options.MaxExecutionTime)
		defer cancel()
	}

	// Validate first
	if err := p.Validate(); err != nil {
		pipelineErr := NewPipelineError("ExecuteWithFormat", -1, err)
		pipelineErr.AddPipelineContext("operation_count", len(p.operations))
		pipelineErr.AddPipelineContext("max_operations", p.options.MaxOperations)
		pipelineErr.AddPipelineContext("max_execution_time", p.options.MaxExecutionTime)
		pipelineErr.AddPipelineContext("format", format)
		return nil, pipelineErr
	}

	// Optimize operations before execution
	p.optimize()

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
			pipelineErr := NewPipelineError("ExecuteWithFormat", contentIndex, ctx.Err())
			pipelineErr.AddPipelineContext("cancelled_at_content", contentIndex)
			pipelineErr.AddPipelineContext("total_contents", len(p.document.GetContents()))
			pipelineErr.AddPipelineContext("format", format)
			return nil, pipelineErr
		default:
		}

		if content.Type() == ContentTypeTable {
			// Count input records
			if tableContent, ok := content.(*TableContent); ok {
				stats.InputRecords += len(tableContent.records)
			}

			// Apply transformations to table content with format context
			transformed, err := p.applyOperationsWithFormat(ctx, content, format, &stats)
			if err != nil {
				// Check if it's already a PipelineError, if so enhance it, otherwise wrap it
				var pipelineErr *PipelineError
				if AsError(err, &pipelineErr) {
					// Already a pipeline error, add additional context
					pipelineErr.AddPipelineContext("content_index", contentIndex)
					pipelineErr.AddPipelineContext("content_type", content.Type().String())
					pipelineErr.AddPipelineContext("content_id", content.ID())
					pipelineErr.AddPipelineContext("format", format)
					return nil, pipelineErr
				} else {
					// Wrap in new pipeline error
					newPipelineErr := NewPipelineError("ContentTransformation", contentIndex, err)
					newPipelineErr.AddPipelineContext("content_type", content.Type().String())
					newPipelineErr.AddPipelineContext("content_id", content.ID())
					newPipelineErr.AddPipelineContext("format", format)
					return nil, newPipelineErr
				}
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

	// Create new document with transformed contents and stats
	originalMetadata := p.document.GetMetadata()
	newMetadata := make(map[string]any, len(originalMetadata)+1)

	// Copy original metadata
	maps.Copy(newMetadata, originalMetadata)

	// Add transformation stats
	newMetadata["transform_stats"] = stats

	newDoc := createDocumentWithContents(newContents, newMetadata)

	return newDoc, nil
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
		pipelineErr := NewPipelineError("Execute", -1, err)
		pipelineErr.AddPipelineContext("operation_count", len(p.operations))
		pipelineErr.AddPipelineContext("max_operations", p.options.MaxOperations)
		pipelineErr.AddPipelineContext("max_execution_time", p.options.MaxExecutionTime)
		return nil, pipelineErr
	}

	// Optimize operations before execution
	p.optimize()

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
			pipelineErr := NewPipelineError("Execute", contentIndex, ctx.Err())
			pipelineErr.AddPipelineContext("cancelled_at_content", contentIndex)
			pipelineErr.AddPipelineContext("total_contents", len(p.document.GetContents()))
			return nil, pipelineErr
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
				// Check if it's already a PipelineError, if so enhance it, otherwise wrap it
				var pipelineErr *PipelineError
				if AsError(err, &pipelineErr) {
					// Already a pipeline error, add additional context
					pipelineErr.AddPipelineContext("content_index", contentIndex)
					pipelineErr.AddPipelineContext("content_type", content.Type().String())
					pipelineErr.AddPipelineContext("content_id", content.ID())
					return nil, pipelineErr
				} else {
					// Wrap in new pipeline error
					newPipelineErr := NewPipelineError("ContentTransformation", contentIndex, err)
					newPipelineErr.AddPipelineContext("content_type", content.Type().String())
					newPipelineErr.AddPipelineContext("content_id", content.ID())
					return nil, newPipelineErr
				}
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

	// Create new document with transformed contents and stats
	originalMetadata := p.document.GetMetadata()
	newMetadata := make(map[string]any, len(originalMetadata)+1)

	// Copy original metadata
	maps.Copy(newMetadata, originalMetadata)

	// Add transformation stats
	newMetadata["transform_stats"] = stats

	newDoc := createDocumentWithContents(newContents, newMetadata)

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
			pipelineErr := NewPipelineError("Execute", i, ctx.Err())
			pipelineErr.AddContext("operation_name", op.Name())
			pipelineErr.AddContext("operation_index", i)
			return nil, pipelineErr
		default:
		}

		opStart := time.Now()

		// Apply the operation
		result, err := op.Apply(ctx, current)
		if err != nil {
			// Create detailed pipeline error with context
			pipelineErr := NewPipelineError(op.Name(), i, err)
			pipelineErr.AddContext("operation_type", fmt.Sprintf("%T", op))
			pipelineErr.AddContext("content_type", current.Type().String())

			// Add input sample for debugging (limit size to avoid huge errors)
			if tableContent, ok := current.(*TableContent); ok {
				if len(tableContent.records) > 0 {
					// Include just the first record as a sample
					pipelineErr.Input = tableContent.records[0]
				}
				pipelineErr.AddContext("input_record_count", len(tableContent.records))
			}

			return nil, pipelineErr
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

// applyOperationsWithFormat applies operations with format context
func (p *Pipeline) applyOperationsWithFormat(ctx context.Context, content Content, format string, stats *TransformStats) (Content, error) {
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
			pipelineErr := NewPipelineError("ExecuteWithFormat", i, ctx.Err())
			pipelineErr.AddContext("operation_name", op.Name())
			pipelineErr.AddContext("operation_index", i)
			pipelineErr.AddContext("format", format)
			return nil, pipelineErr
		default:
		}

		// Check if operation supports format awareness and applies to this format
		if formatAwareOp, ok := op.(FormatAwareOperation); ok {
			if !formatAwareOp.CanTransform(current, format) {
				// Skip this operation for this format
				continue
			}
		}

		opStart := time.Now()

		var result Content
		var err error

		// Use format-aware operation if available
		if formatAwareOp, ok := op.(FormatAwareOperation); ok {
			result, err = formatAwareOp.ApplyWithFormat(ctx, current, format)
		} else {
			// Fall back to regular Apply method for backward compatibility
			result, err = op.Apply(ctx, current)
		}

		if err != nil {
			// Create detailed pipeline error with context
			pipelineErr := NewPipelineError(op.Name(), i, err)
			pipelineErr.AddContext("operation_type", fmt.Sprintf("%T", op))
			pipelineErr.AddContext("content_type", current.Type().String())
			pipelineErr.AddContext("format", format)

			// Add input sample for debugging (limit size to avoid huge errors)
			if tableContent, ok := current.(*TableContent); ok {
				if len(tableContent.records) > 0 {
					// Include just the first record as a sample
					pipelineErr.Input = tableContent.records[0]
				}
				pipelineErr.AddContext("input_record_count", len(tableContent.records))
			}

			return nil, pipelineErr
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
// Implements operation reordering for performance improvements
func (p *Pipeline) optimize() {
	if len(p.operations) <= 1 {
		return // Nothing to optimize
	}

	// Separate operations by type for optimal ordering
	var filters []Operation
	var sorts []Operation
	var limits []Operation
	var addColumns []Operation
	var groupBys []Operation
	var others []Operation

	for _, op := range p.operations {
		switch op.Name() {
		case "Filter":
			filters = append(filters, op)
		case "Sort":
			sorts = append(sorts, op)
		case "Limit":
			limits = append(limits, op)
		case "AddColumn":
			addColumns = append(addColumns, op)
		case "GroupBy":
			groupBys = append(groupBys, op)
		default:
			others = append(others, op)
		}
	}

	// Optimal ordering strategy:
	// 1. Apply filters first (reduces data size for subsequent operations)
	// 2. Add calculated columns (may be needed for sorting/grouping)
	// 3. Group operations (further reduces data size)
	// 4. Sort operations (expensive, better with smaller datasets)
	// 5. Limit operations (should be last to get top N results)
	// 6. Other operations

	optimized := make([]Operation, 0, len(p.operations))

	// Apply filters first to reduce data size
	optimized = append(optimized, filters...)

	// Add calculated columns next (may be needed for sorting/grouping)
	optimized = append(optimized, addColumns...)

	// Apply grouping operations (reduces data further)
	optimized = append(optimized, groupBys...)

	// Sort operations (expensive, better after data reduction)
	optimized = append(optimized, sorts...)

	// Apply limits last to get top N of final results
	optimized = append(optimized, limits...)

	// Any other operations
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
