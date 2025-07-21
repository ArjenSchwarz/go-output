package output

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Transformer modifies content or output
type Transformer interface {
	// Name returns the transformer name
	Name() string

	// Transform modifies the input bytes
	Transform(ctx context.Context, input []byte, format string) ([]byte, error)

	// CanTransform checks if this transformer applies to the given format
	CanTransform(format string) bool

	// Priority determines transform order (lower = earlier)
	Priority() int
}

// TransformPipeline manages a collection of transformers with priority-based ordering
type TransformPipeline struct {
	mu           sync.RWMutex
	transformers []Transformer
	sorted       bool
}

// NewTransformPipeline creates a new transform pipeline
func NewTransformPipeline() *TransformPipeline {
	return &TransformPipeline{
		transformers: make([]Transformer, 0),
		sorted:       true, // Empty pipeline is considered sorted
	}
}

// Add adds a transformer to the pipeline
func (tp *TransformPipeline) Add(transformer Transformer) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.transformers = append(tp.transformers, transformer)
	tp.sorted = false
}

// Remove removes a transformer from the pipeline by name
func (tp *TransformPipeline) Remove(name string) bool {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	for i, t := range tp.transformers {
		if t.Name() == name {
			tp.transformers = append(tp.transformers[:i], tp.transformers[i+1:]...)
			return true
		}
	}
	return false
}

// Has checks if a transformer with the given name exists in the pipeline
func (tp *TransformPipeline) Has(name string) bool {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	for _, t := range tp.transformers {
		if t.Name() == name {
			return true
		}
	}
	return false
}

// Get returns a transformer by name, or nil if not found
func (tp *TransformPipeline) Get(name string) Transformer {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	for _, t := range tp.transformers {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

// Clear removes all transformers from the pipeline
func (tp *TransformPipeline) Clear() {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.transformers = tp.transformers[:0]
	tp.sorted = true
}

// Count returns the number of transformers in the pipeline
func (tp *TransformPipeline) Count() int {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	return len(tp.transformers)
}

// sortTransformers ensures transformers are sorted by priority (lower numbers first)
func (tp *TransformPipeline) sortTransformers() {
	if tp.sorted {
		return
	}

	sort.Slice(tp.transformers, func(i, j int) bool {
		return tp.transformers[i].Priority() < tp.transformers[j].Priority()
	})
	tp.sorted = true
}

// Transform applies all applicable transformers to the input in priority order
func (tp *TransformPipeline) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	tp.mu.Lock()
	tp.sortTransformers()

	// Create a copy of transformers to avoid holding the lock during transformation
	applicable := make([]Transformer, 0, len(tp.transformers))
	for _, t := range tp.transformers {
		if t.CanTransform(format) {
			applicable = append(applicable, t)
		}
	}
	tp.mu.Unlock()

	// Apply transformers in order
	output := input
	for _, transformer := range applicable {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var err error
		output, err = transformer.Transform(ctx, output, format)
		if err != nil {
			return nil, NewTransformError(transformer.Name(), format, output, err)
		}
	}

	return output, nil
}

// TransformInfo provides information about transformers in the pipeline
type TransformInfo struct {
	Name     string
	Priority int
	Formats  []string // Formats this transformer can handle
}

// Info returns information about all transformers in the pipeline, ordered by priority
func (tp *TransformPipeline) Info() []TransformInfo {
	tp.mu.Lock()
	tp.sortTransformers()

	info := make([]TransformInfo, 0, len(tp.transformers))
	for _, t := range tp.transformers {
		// Test common formats to see which ones this transformer supports
		formats := make([]string, 0)
		testFormats := []string{"json", "yaml", "csv", "html", "table", "markdown", "dot", "mermaid", "drawio"}
		for _, format := range testFormats {
			if t.CanTransform(format) {
				formats = append(formats, format)
			}
		}

		info = append(info, TransformInfo{
			Name:     t.Name(),
			Priority: t.Priority(),
			Formats:  formats,
		})
	}
	tp.mu.Unlock()

	return info
}

// TransformError represents an error that occurred during transformation
type TransformError struct {
	Transformer string
	Input       []byte
	Format      string
	Cause       error
}

func (e *TransformError) Error() string {
	return fmt.Sprintf("transformer %s failed for format %s: %v", e.Transformer, e.Format, e.Cause)
}

func (e *TransformError) Unwrap() error {
	return e.Cause
}

// NewTransformError creates a new transform error
func NewTransformError(transformer, format string, input []byte, cause error) *TransformError {
	return &TransformError{
		Transformer: transformer,
		Input:       input,
		Format:      format,
		Cause:       cause,
	}
}
