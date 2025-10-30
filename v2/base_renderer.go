// Package output provides a flexible output generation library for Go applications.
// It supports multiple output formats (JSON, YAML(), CSV(), HTML(), Markdown(), etc.)
// with a document-builder pattern that eliminates global state and provides
// thread-safe operations.
package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// RendererConfig provides collapsible-specific configuration for renderers
type RendererConfig struct {
	// Global expansion override (Requirement 13: Global Expansion Control)
	ForceExpansion bool

	// Character limits for detail truncation (Requirement 10: configurable with 500 default)
	MaxDetailLength   int
	TruncateIndicator string

	// Format-specific settings (Requirement 14: Configurable Renderer Settings)
	TableHiddenIndicator string
	HTMLCSSClasses       map[string]string

	// Transformer configuration (Task 10.2: Dual transformer system)
	DataTransformers []*TransformerAdapter
	ByteTransformers *TransformPipeline
}

// DefaultRendererConfig provides sensible default configuration values
var DefaultRendererConfig = RendererConfig{
	ForceExpansion:       false,
	MaxDetailLength:      500,
	TruncateIndicator:    "[...truncated]",
	TableHiddenIndicator: "[details hidden - use --expand for full view]",
	HTMLCSSClasses: map[string]string{
		"details": "collapsible-cell",
		"summary": "collapsible-summary",
		"content": "collapsible-details",
	},
	DataTransformers: make([]*TransformerAdapter, 0),
	ByteTransformers: NewTransformPipeline(),
}

// baseRenderer provides common functionality shared by all renderer implementations
type baseRenderer struct {
	mu     sync.RWMutex
	config RendererConfig
}

// renderDocument handles the core document rendering logic with context cancellation
func (b *baseRenderer) renderDocument(ctx context.Context, doc *Document, renderFunc func(Content) ([]byte, error)) ([]byte, error) {
	return b.renderDocumentWithFormat(ctx, doc, renderFunc, "")
}

// renderDocumentWithFormat handles the dual transformer system
func (b *baseRenderer) renderDocumentWithFormat(ctx context.Context, doc *Document, renderFunc func(Content) ([]byte, error), format string) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	b.mu.RLock()
	config := b.config
	b.mu.RUnlock()

	// Step 1: Apply data transformers to the document before rendering
	transformedDoc, err := b.applyDataTransformers(ctx, doc, format, config.DataTransformers)
	if err != nil {
		return nil, fmt.Errorf("data transformation failed: %w", err)
	}

	// Step 2: Render the transformed document
	renderedBytes, err := b.renderTransformedDocument(ctx, transformedDoc, renderFunc)
	if err != nil {
		return nil, fmt.Errorf("rendering failed: %w", err)
	}

	// Step 3: Apply byte transformers to the rendered output
	finalBytes, err := b.applyByteTransformers(ctx, renderedBytes, format, config.ByteTransformers)
	if err != nil {
		return nil, fmt.Errorf("byte transformation failed: %w", err)
	}

	return finalBytes, nil
}

// renderTransformedDocument renders the document after data transformations
func (b *baseRenderer) renderTransformedDocument(ctx context.Context, doc *Document, renderFunc func(Content) ([]byte, error)) ([]byte, error) {
	var result bytes.Buffer
	contents := doc.GetContents()

	for i, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply per-content transformations before rendering
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}

		contentBytes, err := renderFunc(transformed)
		if err != nil {
			return nil, fmt.Errorf("failed to render content %s: %w", content.ID(), err)
		}

		if i > 0 && len(contentBytes) > 0 {
			result.WriteByte('\n')
		}

		result.Write(contentBytes)
	}

	return result.Bytes(), nil
}

// applyDataTransformers applies data transformations before rendering
func (b *baseRenderer) applyDataTransformers(ctx context.Context, doc *Document, format string, transformers []*TransformerAdapter) (*Document, error) {
	if len(transformers) == 0 {
		return doc, nil
	}

	// Filter and sort applicable transformers by priority
	var applicable []DataTransformer
	for _, adapter := range transformers {
		if !adapter.IsDataTransformer() {
			continue
		}

		dataTransformer := adapter.AsDataTransformer()
		if dataTransformer == nil {
			continue
		}

		// Check if transformer applies to any content in the document
		hasApplicableContent := false
		for _, content := range doc.GetContents() {
			if dataTransformer.CanTransform(content, format) {
				hasApplicableContent = true
				break
			}
		}

		if hasApplicableContent {
			applicable = append(applicable, dataTransformer)
		}
	}

	if len(applicable) == 0 {
		return doc, nil
	}

	// Sort by priority (lower numbers = higher priority)
	sort.Slice(applicable, func(i, j int) bool {
		return applicable[i].Priority() < applicable[j].Priority()
	})

	// Apply transformers in sequence
	currentDoc := doc
	for _, transformer := range applicable {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		transformedContents := make([]Content, 0, len(currentDoc.GetContents()))

		for _, content := range currentDoc.GetContents() {
			if transformer.CanTransform(content, format) {
				transformed, err := transformer.TransformData(ctx, content, format)
				if err != nil {
					return nil, fmt.Errorf("transformer %s failed: %w", transformer.Name(), err)
				}
				transformedContents = append(transformedContents, transformed)
			} else {
				// Pass through unchanged content
				transformedContents = append(transformedContents, content)
			}
		}

		// Create new document with transformed contents
		currentDoc = &Document{
			contents: transformedContents,
			metadata: currentDoc.GetMetadata(),
		}
	}

	return currentDoc, nil
}

// applyByteTransformers applies byte transformations after rendering
func (b *baseRenderer) applyByteTransformers(ctx context.Context, input []byte, format string, pipeline *TransformPipeline) ([]byte, error) {
	if pipeline == nil {
		return input, nil
	}

	return pipeline.Transform(ctx, input, format)
}

// renderDocumentTo handles streaming document rendering with context cancellation
func (b *baseRenderer) renderDocumentTo(ctx context.Context, doc *Document, w io.Writer, renderFunc func(Content, io.Writer) error) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	contents := doc.GetContents()

	for i, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if i > 0 {
			if _, err := w.Write([]byte{'\n'}); err != nil {
				return fmt.Errorf("failed to write separator: %w", err)
			}
		}

		// Apply per-content transformations before rendering
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return err
		}

		if err := renderFunc(transformed, w); err != nil {
			return fmt.Errorf("failed to render content %s: %w", content.ID(), err)
		}
	}

	return nil
}

// renderContent provides a default content rendering implementation
func (b *baseRenderer) renderContent(content Content) ([]byte, error) {
	if content == nil {
		return nil, fmt.Errorf("content cannot be nil")
	}

	// Use the content's own AppendText method for basic rendering
	return content.AppendText(nil)
}

// renderContentTo provides a default streaming content rendering implementation
func (b *baseRenderer) renderContentTo(content Content, w io.Writer) error {
	if content == nil {
		return fmt.Errorf("content cannot be nil")
	}
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}

	contentBytes, err := content.AppendText(nil)
	if err != nil {
		return err
	}

	_, err = w.Write(contentBytes)
	return err
}

// processFieldValue applies Field.Formatter and detects CollapsibleValue interface
// This method provides collapsible value detection for all renderers while maintaining
// backward compatibility with existing formatters.
func (b *baseRenderer) processFieldValue(val any, field *Field) any {
	if field != nil && field.Formatter != nil {
		// Apply enhanced formatter (returns any, could be CollapsibleValue)
		return field.Formatter(val)
	}
	return val
}

// ProcessedValue represents a value that has been processed through field formatting
// with cached type information to minimize type assertions (Requirement 10.1)
type ProcessedValue struct {
	Value            any
	IsCollapsible    bool
	CollapsibleValue CollapsibleValue
	detailsAccessed  bool
	cachedDetails    any
	formattedString  string
	stringFormatted  bool
}

// processFieldValueOptimized applies Field.Formatter and caches CollapsibleValue detection
// This optimized version performs type assertion only once per value (Requirement 10.1)
func (b *baseRenderer) processFieldValueOptimized(val any, field *Field) *ProcessedValue {
	processed := b.processFieldValue(val, field)

	result := &ProcessedValue{
		Value: processed,
	}

	// Perform single type assertion for CollapsibleValue detection (Requirement 10.1)
	if cv, ok := processed.(CollapsibleValue); ok {
		result.IsCollapsible = true
		result.CollapsibleValue = cv
	}

	return result
}

// GetDetails returns the details from a CollapsibleValue with lazy evaluation
// This avoids unnecessary processing when details are not accessed (Requirement 10.3)
func (pv *ProcessedValue) GetDetails() any {
	if !pv.IsCollapsible {
		return nil
	}

	// Cache details on first access to avoid redundant processing (Requirement 10.3)
	if !pv.detailsAccessed {
		pv.cachedDetails = pv.CollapsibleValue.Details()
		pv.detailsAccessed = true
	}

	return pv.cachedDetails
}

// String returns the string representation with caching to avoid redundant conversions
func (pv *ProcessedValue) String() string {
	if !pv.stringFormatted {
		if pv.IsCollapsible {
			pv.formattedString = pv.CollapsibleValue.Summary()
		} else {
			pv.formattedString = fmt.Sprint(pv.Value)
		}
		pv.stringFormatted = true
	}
	return pv.formattedString
}

// StreamingValueProcessor handles large dataset processing with minimal memory footprint
// This maintains streaming capabilities for large datasets (Requirement 10.2)
type StreamingValueProcessor struct {
	config RendererConfig
}

// NewStreamingValueProcessor creates a processor optimized for large datasets
func NewStreamingValueProcessor(config RendererConfig) *StreamingValueProcessor {
	return &StreamingValueProcessor{
		config: config,
	}
}

// ProcessBatch processes a batch of values efficiently without loading all details into memory
// This avoids unnecessary memory allocation when processing large datasets (Requirement 10.2)
func (svp *StreamingValueProcessor) ProcessBatch(values []any, fields []*Field, processor func(*ProcessedValue) error) error {
	for i, val := range values {
		// Get field for this value (may be nil)
		var field *Field
		if i < len(fields) {
			field = fields[i]
		}

		// Create processed value with lazy evaluation
		processed := &ProcessedValue{
			Value: val,
		}

		// Apply field formatter if available
		if field != nil && field.Formatter != nil {
			processed.Value = field.Formatter(val)
		}

		// Perform single type assertion for CollapsibleValue detection
		if cv, ok := processed.Value.(CollapsibleValue); ok {
			processed.IsCollapsible = true
			processed.CollapsibleValue = cv
		}

		// Process without forcing details evaluation unless needed
		if err := processor(processed); err != nil {
			return fmt.Errorf("failed to process value at index %d: %w", i, err)
		}

		// Clear processed value to free memory (streaming optimization)
		processed = nil
	}

	return nil
}

// MemoryOptimizedProcessor handles large content processing with memory pooling
// This implements memory-efficient processing for large content (Requirement 10.4, 10.6, 10.7)
type MemoryOptimizedProcessor struct {
	config      RendererConfig
	bufferPool  sync.Pool
	stringPool  sync.Pool
	maxPoolSize int
}

// NewMemoryOptimizedProcessor creates a processor with memory pooling for large datasets
func NewMemoryOptimizedProcessor(config RendererConfig) *MemoryOptimizedProcessor {
	processor := &MemoryOptimizedProcessor{
		config:      config,
		maxPoolSize: 100, // Limit pool size to prevent unbounded memory growth
	}

	// Initialize buffer pool for efficient string building
	processor.bufferPool = sync.Pool{
		New: func() any {
			return &strings.Builder{}
		},
	}

	// Initialize string slice pool for efficient array handling
	processor.stringPool = sync.Pool{
		New: func() any {
			slice := make([]string, 0, 10) // Start with capacity of 10
			return &slice
		},
	}

	return processor
}

// GetBuffer retrieves a buffer from the pool for efficient string building
func (mop *MemoryOptimizedProcessor) GetBuffer() *strings.Builder {
	buf := mop.bufferPool.Get().(*strings.Builder)
	buf.Reset()
	return buf
}

// ReturnBuffer returns a buffer to the pool for reuse
func (mop *MemoryOptimizedProcessor) ReturnBuffer(buf *strings.Builder) {
	// Only return reasonably sized buffers to prevent memory bloat
	if buf.Cap() < 4096 { // 4KB limit
		mop.bufferPool.Put(buf)
	}
}

// GetStringSlice retrieves a string slice from the pool
func (mop *MemoryOptimizedProcessor) GetStringSlice() []string {
	slice := mop.stringPool.Get().(*[]string)
	*slice = (*slice)[:0] // Reset length but keep capacity
	return *slice
}

// ReturnStringSlice returns a string slice to the pool for reuse
func (mop *MemoryOptimizedProcessor) ReturnStringSlice(slice []string) {
	// Only return reasonably sized slices to prevent memory bloat
	if cap(slice) < 100 {
		// Clear the slice before returning to pool
		slice = slice[:0]
		mop.stringPool.Put(&slice)
	}
}

// ProcessLargeDetails efficiently processes large detail content with memory management
// This avoids redundant transformations for large content (Requirement 10.4)
func (mop *MemoryOptimizedProcessor) ProcessLargeDetails(details any, maxLength int) (any, error) {
	if details == nil {
		return nil, nil
	}

	// Handle different detail types efficiently
	switch d := details.(type) {
	case string:
		return mop.processLargeString(d, maxLength), nil

	case []string:
		return mop.processStringArray(d, maxLength), nil

	case map[string]any:
		return mop.processMap(d, maxLength), nil

	default:
		// For other types, convert to string and process
		str := fmt.Sprint(details)
		return mop.processLargeString(str, maxLength), nil
	}
}

// processLargeString handles large string content with efficient truncation
func (mop *MemoryOptimizedProcessor) processLargeString(content string, maxLength int) string {
	if maxLength <= 0 || len(content) <= maxLength {
		return content
	}

	// Efficient truncation with configured indicator
	truncateIndicator := mop.config.TruncateIndicator
	if truncateIndicator == "" {
		truncateIndicator = "[...truncated]"
	}

	if len(content) > maxLength {
		return content[:maxLength] + truncateIndicator
	}

	return content
}

// processStringArray handles string arrays with memory pooling
func (mop *MemoryOptimizedProcessor) processStringArray(content []string, maxLength int) []string {
	if len(content) == 0 {
		return content
	}

	// Use pooled slice for memory efficiency
	result := mop.GetStringSlice()
	defer mop.ReturnStringSlice(result)

	totalLength := 0
	for _, str := range content {
		if maxLength > 0 && totalLength+len(str) > maxLength {
			// Truncate remaining content
			remaining := maxLength - totalLength
			if remaining > 0 {
				result = append(result, str[:remaining]+mop.config.TruncateIndicator)
			}
			break
		}

		result = append(result, str)
		totalLength += len(str)
	}

	// Return a copy to avoid issues with pooled slice reuse
	finalResult := make([]string, len(result))
	copy(finalResult, result)
	return finalResult
}

// processMap handles map content with efficient string building
func (mop *MemoryOptimizedProcessor) processMap(content map[string]any, maxLength int) map[string]any {
	if len(content) == 0 {
		return content
	}

	// For maps, we'll limit individual values but preserve structure
	result := make(map[string]any, len(content))

	for key, value := range content {
		if valueStr, ok := value.(string); ok && maxLength > 0 {
			result[key] = mop.processLargeString(valueStr, maxLength/len(content)) // Distribute limit
		} else {
			result[key] = value
		}
	}

	return result
}

// NewMarkdownRendererWithCollapsible creates a markdown renderer with collapsible configuration
func NewMarkdownRendererWithCollapsible(config RendererConfig) Renderer {
	return &markdownRenderer{
		baseRenderer:      baseRenderer{},
		includeToC:        false,
		headingLevel:      1,
		collapsibleConfig: config,
	}
}

// NewTableRendererWithCollapsible creates a table renderer with collapsible configuration
func NewTableRendererWithCollapsible(styleName string, config RendererConfig) Renderer {
	return &tableRenderer{
		styleName:         styleName,
		collapsibleConfig: config,
	}
}

// NewHTMLRendererWithCollapsible creates an HTML renderer with collapsible configuration
func NewHTMLRendererWithCollapsible(config RendererConfig) Renderer {
	return &htmlRenderer{
		baseRenderer:      baseRenderer{},
		collapsibleConfig: config,
	}
}

// NewCSVRendererWithCollapsible creates a CSV renderer with collapsible configuration
func NewCSVRendererWithCollapsible(config RendererConfig) Renderer {
	return &csvRenderer{
		collapsibleConfig: config,
	}
}
