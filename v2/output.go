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
	return SafeExecuteWithTracer(GetGlobalDebugTracer(), "render", func() error {
		// Validate inputs early
		if err := FailFast(
			ValidateNonNil("context", ctx),
			ValidateNonNil("document", doc),
		); err != nil {
			return err
		}

		GlobalTrace("render", "starting document render process")

		o.mu.RLock()
		formats := make([]Format, len(o.formats))
		copy(formats, o.formats)
		writers := make([]Writer, len(o.writers))
		copy(writers, o.writers)
		transformers := make([]Transformer, len(o.transformers))
		copy(transformers, o.transformers)
		progress := o.progress
		o.mu.RUnlock()

		GlobalTrace("render", "loaded configuration: %d formats, %d writers, %d transformers",
			len(formats), len(writers), len(transformers))

		// Validate configuration
		if err := FailFast(
			ValidateSliceNonEmpty("formats", formats),
			ValidateSliceNonEmpty("writers", writers),
		); err != nil {
			return err
		}

		return o.renderWithConfig(ctx, doc, formats, writers, transformers, progress)
	})
}

// renderWithConfig performs the actual rendering with the given configuration
func (o *Output) renderWithConfig(ctx context.Context, doc *Document, formats []Format, writers []Writer, transformers []Transformer, progress Progress) error {
	// Check for cancellation early
	if IsCancelled(ctx.Err()) {
		return NewCancelledError("render", ctx.Err())
	}

	// Calculate total work units for progress tracking
	totalWork := len(formats) * len(writers)
	progress.SetTotal(totalWork)
	progress.SetStatus("Starting render process")

	GlobalTrace("render", "starting concurrent processing of %d format(s) to %d writer(s)", len(formats), len(writers))

	var wg sync.WaitGroup
	errChan := make(chan error, totalWork)
	workDone := 0
	workMu := sync.Mutex{}

	// Process each format concurrently
	for _, format := range formats {
		wg.Add(1)
		go func(f Format) {
			defer wg.Done()

			// Use safe execution with panic recovery for each format
			err := SafeExecuteWithTracer(GetGlobalDebugTracer(), fmt.Sprintf("render-%s", f.Name), func() error {
				// Check for cancellation
				if IsCancelled(ctx.Err()) {
					return NewCancelledError(fmt.Sprintf("render-%s", f.Name), ctx.Err())
				}

				GlobalTrace("render", "starting render for format: %s", f.Name)
				progress.SetStatus(fmt.Sprintf("Rendering %s format", f.Name))

				// Render the document in this format
				data, err := f.Renderer.Render(ctx, doc)
				if err != nil {
					// Create a detailed render error with enhanced context
					renderErr := NewRenderErrorWithDetails(f.Name, fmt.Sprintf("%T", f.Renderer), "render", nil, err)
					renderErr.AddContext("renderer_type", fmt.Sprintf("%T", f.Renderer))
					if data != nil {
						renderErr.AddContext("data_size", len(data))
					}
					return renderErr
				}

				GlobalTrace("render", "rendered %s format successfully, %d bytes", f.Name, len(data))
				return o.processFormatData(ctx, f, data, transformers, writers, progress, &workDone, &workMu)
			})

			if err != nil {
				errChan <- err
			}
		}(format)
	}

	// Wait for all renders to complete
	GlobalTrace("render", "waiting for all format processing to complete")
	wg.Wait()
	close(errChan)

	// Collect all errors using the enhanced error handling system
	multiErr := NewMultiError("render")
	multiErr.AddContext("total_formats", len(o.formats))
	multiErr.AddContext("document_contents", len(doc.GetContents()))
	for err := range errChan {
		// Add error with source tracking - determine source component from error type
		component := unknownValue
		details := make(map[string]any)

		var renderErr *RenderError
		if AsError(err, &renderErr) {
			component = "renderer"
			details["format"] = renderErr.Format
			details["renderer"] = renderErr.Renderer
		} else {
			var transformErr *TransformError
			if AsError(err, &transformErr) {
				component = "transformer"
				details["transformer"] = transformErr.Transformer
				details["format"] = transformErr.Format
			} else {
				var writerErr *WriterError
				if AsError(err, &writerErr) {
					component = "writer"
					details["writer"] = writerErr.Writer
					details["format"] = writerErr.Format
					details["operation"] = writerErr.Operation
				} else {
					var contextErr *ContextError
					if AsError(err, &contextErr) {
						component = contextErr.Operation
						for k, v := range contextErr.Context {
							details[k] = v
						}
					}
				}
			}
		}

		multiErr.AddWithSource(err, component, details)
	}

	if multiErr.HasErrors() {
		GlobalError("render", "render process failed with %d error(s)", len(multiErr.Errors))
		progress.Fail(multiErr)
		return multiErr
	}

	GlobalTrace("render", "all format processing completed successfully")
	progress.Complete()
	return nil
}

// processFormatData applies transformers and writes the data to all configured writers
func (o *Output) processFormatData(ctx context.Context, format Format, data []byte, transformers []Transformer, writers []Writer, progress Progress, workDone *int, workMu *sync.Mutex) error {
	// Apply transformers to the rendered data
	transformedData := data
	for _, transformer := range transformers {
		if transformer.CanTransform(format.Name) {
			GlobalTrace("transform", "applying %s transformer to %s format", transformer.Name(), format.Name)
			progress.SetStatus(fmt.Sprintf("Applying %s transformer to %s", transformer.Name(), format.Name))

			var err error
			transformedData, err = transformer.Transform(ctx, transformedData, format.Name)
			if err != nil {
				return ErrorWithContext("transform", err,
					"format", format.Name,
					"transformer", transformer.Name(),
					"input_size", len(data))
			}

			GlobalTrace("transform", "applied %s transformer to %s format, %d -> %d bytes",
				transformer.Name(), format.Name, len(data), len(transformedData))
		}
	}

	// Write to all configured writers
	for _, writer := range writers {
		// Check for cancellation before each write
		if IsCancelled(ctx.Err()) {
			return NewCancelledError(fmt.Sprintf("write-%s", format.Name), ctx.Err())
		}

		GlobalTrace("write", "writing %s format using %T writer", format.Name, writer)
		progress.SetStatus(fmt.Sprintf("Writing %s format", format.Name))

		err := writer.Write(ctx, format.Name, transformedData)
		if err != nil {
			// Create a detailed writer error with enhanced context
			writerErr := NewWriterErrorWithDetails(fmt.Sprintf("%T", writer), format.Name, "write", err)
			writerErr.AddContext("data_size", len(transformedData))
			writerErr.AddContext("writer_type", fmt.Sprintf("%T", writer))
			return writerErr
		}

		// Update progress safely
		workMu.Lock()
		*workDone++
		progress.SetCurrent(*workDone)
		workMu.Unlock()

		GlobalTrace("write", "successfully wrote %s format using %T writer", format.Name, writer)
	}

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
