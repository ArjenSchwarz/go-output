package output

import (
	"context"
	"fmt"
	"sync"
)

// Output manages rendering and writing documents to multiple formats and destinations
type Output struct {
	formats      []Format
	transformers []Transformer
	writers      []Writer
	progress     Progress

	// v1 compatibility features
	tableStyle  string
	hasTOC      bool
	frontMatter map[string]string
	metadata    map[string]any

	mu sync.RWMutex
}

// OutputOption configures Output instances
type OutputOption func(*Output)

// NewOutput creates a new Output instance with the given options
func NewOutput(opts ...OutputOption) *Output {
	output := &Output{
		metadata: make(map[string]any),
	}

	// Apply all options
	for _, opt := range opts {
		opt(output)
	}

	// Default to no-op progress if none specified
	if output.progress == nil {
		output.progress = NewNoOpProgress()
	}

	return output
}

// WithFormat adds an output format to the Output
func WithFormat(format Format) OutputOption {
	return func(o *Output) {
		o.formats = append(o.formats, format)
	}
}

// WithFormats adds multiple output formats to the Output
func WithFormats(formats ...Format) OutputOption {
	return func(o *Output) {
		o.formats = append(o.formats, formats...)
	}
}

// WithTransformer adds a transformer to the Output pipeline
func WithTransformer(transformer Transformer) OutputOption {
	return func(o *Output) {
		o.transformers = append(o.transformers, transformer)
	}
}

// WithTransformers adds multiple transformers to the Output pipeline
func WithTransformers(transformers ...Transformer) OutputOption {
	return func(o *Output) {
		o.transformers = append(o.transformers, transformers...)
	}
}

// WithWriter adds a writer to the Output
func WithWriter(writer Writer) OutputOption {
	return func(o *Output) {
		o.writers = append(o.writers, writer)
	}
}

// WithWriters adds multiple writers to the Output
func WithWriters(writers ...Writer) OutputOption {
	return func(o *Output) {
		o.writers = append(o.writers, writers...)
	}
}

// WithProgress sets the progress indicator for the Output
func WithProgress(progress Progress) OutputOption {
	return func(o *Output) {
		o.progress = progress
	}
}

// WithTableStyle sets the table style for v1 compatibility
func WithTableStyle(style string) OutputOption {
	return func(o *Output) {
		o.tableStyle = style
	}
}

// WithTOC enables or disables table of contents generation for v1 compatibility
func WithTOC(enabled bool) OutputOption {
	return func(o *Output) {
		o.hasTOC = enabled
	}
}

// WithFrontMatter sets markdown front matter for v1 compatibility
func WithFrontMatter(fm map[string]string) OutputOption {
	return func(o *Output) {
		if o.frontMatter == nil {
			o.frontMatter = make(map[string]string)
		}
		for k, v := range fm {
			o.frontMatter[k] = v
		}
	}
}

// WithMetadata sets metadata for the output
func WithMetadata(key string, value any) OutputOption {
	return func(o *Output) {
		o.metadata[key] = value
	}
}

// Render processes a document through all configured formats, transformers, and writers
func (o *Output) Render(ctx context.Context, doc *Document) error {
	o.mu.RLock()
	formats := make([]Format, len(o.formats))
	copy(formats, o.formats)
	writers := make([]Writer, len(o.writers))
	copy(writers, o.writers)
	transformers := make([]Transformer, len(o.transformers))
	copy(transformers, o.transformers)
	progress := o.progress
	o.mu.RUnlock()

	if len(formats) == 0 {
		return fmt.Errorf("no output formats configured")
	}

	if len(writers) == 0 {
		return fmt.Errorf("no writers configured")
	}

	// Calculate total work units for progress tracking
	totalWork := len(formats) * len(writers)
	progress.SetTotal(totalWork)
	progress.SetStatus("Starting render process")

	var wg sync.WaitGroup
	errChan := make(chan error, totalWork)
	workDone := 0
	workMu := sync.Mutex{}

	// Process each format concurrently
	for _, format := range formats {
		wg.Add(1)
		go func(f Format) {
			defer wg.Done()

			// Check for cancellation
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			progress.SetStatus(fmt.Sprintf("Rendering %s format", f.Name))

			// Render the document in this format
			data, err := f.Renderer.Render(ctx, doc)
			if err != nil {
				errChan <- fmt.Errorf("failed to render %s format: %w", f.Name, err)
				return
			}

			// Apply transformers to the rendered data
			transformedData := data
			for _, transformer := range transformers {
				if transformer.CanTransform(f.Name) {
					progress.SetStatus(fmt.Sprintf("Applying %s transformer to %s", transformer.Name(), f.Name))
					transformedData, err = transformer.Transform(ctx, transformedData, f.Name)
					if err != nil {
						errChan <- fmt.Errorf("failed to transform %s with %s: %w", f.Name, transformer.Name(), err)
						return
					}
				}
			}

			// Write to all configured writers
			for _, writer := range writers {
				// Check for cancellation before each write
				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				default:
				}

				progress.SetStatus(fmt.Sprintf("Writing %s format", f.Name))
				err := writer.Write(ctx, f.Name, transformedData)
				if err != nil {
					errChan <- fmt.Errorf("failed to write %s format: %w", f.Name, err)
					return
				}

				// Update progress
				workMu.Lock()
				workDone++
				progress.SetCurrent(workDone)
				workMu.Unlock()
			}
		}(format)
	}

	// Wait for all renders to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// Report the first error to progress and return it
		progress.Fail(errors[0])
		return errors[0]
	}

	progress.Complete()
	return nil
}

// RenderTo processes a document and writes all formats to their respective writers
// This is a convenience method that calls Render with a background context
func (o *Output) RenderTo(doc *Document) error {
	return o.Render(context.Background(), doc)
}

// GetFormats returns a copy of the configured formats
func (o *Output) GetFormats() []Format {
	o.mu.RLock()
	defer o.mu.RUnlock()

	formats := make([]Format, len(o.formats))
	copy(formats, o.formats)
	return formats
}

// GetWriters returns a copy of the configured writers
func (o *Output) GetWriters() []Writer {
	o.mu.RLock()
	defer o.mu.RUnlock()

	writers := make([]Writer, len(o.writers))
	copy(writers, o.writers)
	return writers
}

// GetTransformers returns a copy of the configured transformers
func (o *Output) GetTransformers() []Transformer {
	o.mu.RLock()
	defer o.mu.RUnlock()

	transformers := make([]Transformer, len(o.transformers))
	copy(transformers, o.transformers)
	return transformers
}

// GetProgress returns the configured progress indicator
func (o *Output) GetProgress() Progress {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.progress
}

// Close cleans up resources used by the Output
func (o *Output) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.progress != nil {
		return o.progress.Close()
	}
	return nil
}
