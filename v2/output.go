package output

import (
	"context"
	"fmt"
	"maps"
	"sort"
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

// NewOutput creates a new Output instance with the given options.
// Nil options are ignored.
func NewOutput(opts ...OutputOption) *Output {
	output := &Output{
		metadata: make(map[string]any),
	}

	// Apply all options
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(output)
	}

	// Default to no-op progress if none specified. A typed-nil Progress (a nil
	// concrete pointer boxed into the interface) is treated as absent too, so
	// render never calls SetTotal on a value that would panic (T-1649).
	if isNilValue(output.progress) {
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

// WithTableStyle sets the table style for v1 compatibility. During Render the
// style is applied to every configured table format that uses the built-in
// table renderer (Table, TableWithStyle, TableWithMaxColumnWidth, ...); other
// renderer settings such as max column width are preserved. Formats using a
// custom Renderer implementation are unaffected.
func WithTableStyle(style string) OutputOption {
	return func(o *Output) {
		o.tableStyle = style
	}
}

// WithTOC enables or disables table of contents generation for v1
// compatibility. During Render, WithTOC(true) enables ToC generation on every
// configured markdown format that uses the built-in markdown renderer. The
// option is additive: false is the zero value and does not disable a ToC
// enabled through MarkdownWithToC(true). Formats using a custom Renderer
// implementation are unaffected.
func WithTOC(enabled bool) OutputOption {
	return func(o *Output) {
		o.hasTOC = enabled
	}
}

// WithFrontMatter sets markdown front matter for v1 compatibility. During
// Render the entries are applied to every configured markdown format that
// uses the built-in markdown renderer, merging with any front matter supplied
// via MarkdownWithFrontMatter (Output-level keys win on conflict). Formats
// using a custom Renderer implementation are unaffected.
func WithFrontMatter(fm map[string]string) OutputOption {
	return func(o *Output) {
		if o.frontMatter == nil {
			o.frontMatter = make(map[string]string)
		}
		maps.Copy(o.frontMatter, fm)
	}
}

// WithMetadata sets metadata for the output
func WithMetadata(key string, value any) OutputOption {
	return func(o *Output) {
		o.metadata[key] = value
	}
}

// Render processes a document through all configured formats, transformers, and writers.
//
// Formats are rendered and transformed concurrently, but writes happen in
// declared format order: each writer receives the formats in the order they
// were configured (WithFormat/WithFormats), so output sent to a shared writer
// such as a single StdoutWriter is deterministic across runs.
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
		tableStyle := o.tableStyle
		hasTOC := o.hasTOC
		frontMatter := o.frontMatter
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

		// Validate individual configuration entries are non-nil. Without this,
		// a Format with a nil Renderer, a nil transformer, or a nil writer would
		// be dereferenced during rendering and surface as a recovered PanicError
		// instead of a normal validation error.
		if err := validateConfigEntries(formats, transformers, writers); err != nil {
			return err
		}

		// Derive the effective renderers for the Output-level v1 compatibility
		// options (T-1516). This runs after validateConfigEntries so typed-nil
		// renderers have already been rejected.
		formats = applyV1CompatOptions(formats, tableStyle, hasTOC, frontMatter)

		return o.renderWithConfig(ctx, doc, formats, writers, transformers, progress)
	})
}

// validateConfigEntries checks that each configured format has a non-nil
// renderer and that no transformer or writer entry is nil. It returns the first
// validation error encountered, ensuring nil entries are reported as normal
// validation errors rather than being dereferenced during rendering. Checks
// use isNilValue so typed nils (a nil concrete pointer boxed into a non-nil
// interface, e.g. NewFormatAwareTransformer(nil)) are rejected too (T-1649).
func validateConfigEntries(formats []Format, transformers []Transformer, writers []Writer) error {
	for i, f := range formats {
		if isNilValue(f.Renderer) {
			return NewValidationError(
				fmt.Sprintf("formats[%d].renderer", i),
				f.Renderer,
				"cannot be nil",
			)
		}
	}
	for i, transformer := range transformers {
		if isNilValue(transformer) {
			return NewValidationError(
				fmt.Sprintf("transformers[%d]", i),
				transformer,
				"transformer cannot be nil",
			)
		}
	}
	for i, writer := range writers {
		if isNilValue(writer) {
			return NewValidationError(
				fmt.Sprintf("writers[%d]", i),
				writer,
				"writer cannot be nil",
			)
		}
	}
	return nil
}

// applyV1CompatOptions derives the effective formats for the Output-level v1
// compatibility options (WithTableStyle, WithTOC, WithFrontMatter). The
// options target the built-in renderers of their matching formats: tableStyle
// restyles table formats backed by *tableRenderer, and hasTOC/frontMatter
// enable a table of contents and merge front matter (Output-level keys win)
// on markdown formats backed by *markdownRenderer. Renderers are reconfigured
// on copies — the stored formats are never mutated — and settings configured
// through the Format constructors (max column width, collapsible config,
// heading level) are preserved. Custom Renderer implementations and other
// formats are returned unchanged. The options are additive: zero values apply
// no changes, so WithTOC(false) does not disable a ToC enabled through
// MarkdownWithToC(true).
func applyV1CompatOptions(formats []Format, tableStyle string, hasTOC bool, frontMatter map[string]string) []Format {
	if tableStyle == "" && !hasTOC && len(frontMatter) == 0 {
		return formats
	}

	derived := make([]Format, len(formats))
	copy(derived, formats)
	for i, format := range derived {
		switch format.Name {
		case FormatTable:
			if tableStyle == "" {
				continue
			}
			// tableRenderer holds no locks, so a shallow copy is safe and
			// keeps maxColumnWidth and collapsibleConfig intact.
			if tr, ok := format.Renderer.(*tableRenderer); ok && tr != nil {
				clone := *tr
				clone.styleName = tableStyle
				derived[i].Renderer = &clone
			}
		case FormatMarkdown:
			if !hasTOC && len(frontMatter) == 0 {
				continue
			}
			mr, ok := format.Renderer.(*markdownRenderer)
			if !ok || mr == nil {
				continue
			}

			// markdownRenderer embeds baseRenderer, which contains a mutex,
			// so copy fields explicitly instead of copying the struct. The
			// embedded config is read under its own lock; the remaining
			// fields are only written at construction.
			mr.mu.RLock()
			baseConfig := mr.config
			mr.mu.RUnlock()

			merged := mr.frontMatter
			if len(frontMatter) > 0 {
				merged = make(map[string]string, len(mr.frontMatter)+len(frontMatter))
				maps.Copy(merged, mr.frontMatter)
				maps.Copy(merged, frontMatter)
			}

			derived[i].Renderer = &markdownRenderer{
				baseRenderer:      baseRenderer{config: baseConfig},
				includeToC:        mr.includeToC || hasTOC,
				frontMatter:       merged,
				headingLevel:      mr.headingLevel,
				collapsibleConfig: mr.collapsibleConfig,
			}
		}
	}
	return derived
}

// renderWithConfig performs the actual rendering with the given configuration.
//
// Rendering and transformation run concurrently (one goroutine per format),
// but writes are serialized in declared format order: no format's output is
// written until every earlier-declared format has been written. This makes
// output to a shared writer (for example a single StdoutWriter) deterministic
// regardless of how long each format takes to render.
func (o *Output) renderWithConfig(ctx context.Context, doc *Document, formats []Format, writers []Writer, transformers []Transformer, progress Progress) error {
	// Check for cancellation early
	if IsCancelled(ctx.Err()) {
		return NewCancelledError("render", ctx.Err())
	}

	// Calculate total work units for progress tracking
	totalWork := len(formats) * len(writers)
	progress.SetTotal(totalWork)
	progress.SetStatus("Starting render process")

	GlobalTrace("render", "starting concurrent rendering of %d format(s)", len(formats))

	// Phase 1: render and transform each format concurrently. Each goroutine
	// writes only to its own slot of results/renderErrs, so no synchronization
	// beyond the WaitGroup is needed.
	results := make([][]byte, len(formats))
	renderErrs := make([]error, len(formats))

	var wg sync.WaitGroup
	for i, format := range formats {
		wg.Add(1)
		go func(idx int, f Format) {
			defer wg.Done()

			// Use safe execution with panic recovery for each format
			renderErrs[idx] = SafeExecuteWithTracer(GetGlobalDebugTracer(), fmt.Sprintf("render-%s", f.Name), func() error {
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

				transformed, err := o.transformFormatData(ctx, f, data, transformers, progress)
				if err != nil {
					return err
				}
				results[idx] = transformed
				return nil
			})
		}(i, format)
	}

	// Wait for all renders to complete
	GlobalTrace("render", "waiting for all format rendering to complete")
	wg.Wait()

	// Phase 2: write sequentially in declared format order so output sent to
	// shared writers is deterministic. A format that failed to render is
	// reported but does not prevent later formats from being written.
	var errs []error
	workDone := 0
	for i, format := range formats {
		if renderErrs[i] != nil {
			errs = append(errs, renderErrs[i])
			continue
		}

		// Release the slot so earlier formats' buffers can be collected while
		// later formats are still being written (relevant for slow writers).
		data := results[i]
		results[i] = nil
		err := SafeExecuteWithTracer(GetGlobalDebugTracer(), fmt.Sprintf("write-%s", format.Name), func() error {
			return o.writeFormatData(ctx, format, data, writers, progress, &workDone)
		})
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Collect all errors using the enhanced error handling system
	multiErr := NewMultiError("render")
	multiErr.AddContext("total_formats", len(formats))
	multiErr.AddContext("document_contents", len(doc.GetContents()))
	for _, err := range errs {
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
						maps.Copy(details, contextErr.Context)
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

// transformFormatData applies transformers to the rendered data and returns
// the transformed bytes. It runs inside the per-format render goroutines.
func (o *Output) transformFormatData(ctx context.Context, format Format, data []byte, transformers []Transformer, progress Progress) ([]byte, error) {
	// Apply transformers to the rendered data in priority order (lower priority
	// runs first), matching the TransformPipeline contract. Sort a local copy so
	// the caller's slice is not mutated; SliceStable keeps insertion order for
	// transformers that share a priority.
	ordered := make([]Transformer, len(transformers))
	copy(ordered, transformers)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Priority() < ordered[j].Priority()
	})

	transformedData := data
	for _, transformer := range ordered {
		if transformer.CanTransform(format.Name) {
			GlobalTrace("transform", "applying %s transformer to %s format", transformer.Name(), format.Name)
			progress.SetStatus(fmt.Sprintf("Applying %s transformer to %s", transformer.Name(), format.Name))

			var err error
			transformedData, err = transformer.Transform(ctx, transformedData, format.Name)
			if err != nil {
				return nil, ErrorWithContext("transform", err,
					"format", format.Name,
					"transformer", transformer.Name(),
					"input_size", len(data))
			}

			GlobalTrace("transform", "applied %s transformer to %s format, %d -> %d bytes",
				transformer.Name(), format.Name, len(data), len(transformedData))
		}
	}

	return transformedData, nil
}

// writeFormatData writes the transformed data to all configured writers in
// declared writer order. It is called sequentially per format (in declared
// format order) after all renders have completed, so no synchronization is
// needed for the workDone progress counter. A write failure stops the
// remaining writers for this format only.
func (o *Output) writeFormatData(ctx context.Context, format Format, data []byte, writers []Writer, progress Progress, workDone *int) error {
	for _, writer := range writers {
		// Check for cancellation before each write
		if IsCancelled(ctx.Err()) {
			return NewCancelledError(fmt.Sprintf("write-%s", format.Name), ctx.Err())
		}

		GlobalTrace("write", "writing %s format using %T writer", format.Name, writer)
		progress.SetStatus(fmt.Sprintf("Writing %s format", format.Name))

		err := writer.Write(ctx, format.Name, data)
		if err != nil {
			// Create a detailed writer error with enhanced context
			writerErr := NewWriterErrorWithDetails(fmt.Sprintf("%T", writer), format.Name, "write", err)
			writerErr.AddContext("data_size", len(data))
			writerErr.AddContext("writer_type", fmt.Sprintf("%T", writer))
			return writerErr
		}

		*workDone++
		progress.SetCurrent(*workDone)

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
