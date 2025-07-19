package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

// Renderer converts a document to a specific format
type Renderer interface {
	// Format returns the output format name
	Format() string

	// Render converts the document to bytes
	Render(ctx context.Context, doc *Document) ([]byte, error)

	// RenderTo streams output to a writer
	RenderTo(ctx context.Context, doc *Document, w io.Writer) error

	// SupportsStreaming indicates if streaming is supported
	SupportsStreaming() bool
}

// Format represents an output format configuration
type Format struct {
	Name     string
	Renderer Renderer
	Options  map[string]any
}

// baseRenderer provides common functionality for all renderers
type baseRenderer struct {
	mu sync.RWMutex
}

// renderDocument handles the core document rendering logic with context cancellation
func (b *baseRenderer) renderDocument(ctx context.Context, doc *Document, renderFunc func(Content) ([]byte, error)) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	var result bytes.Buffer
	contents := doc.GetContents()

	for i, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		contentBytes, err := renderFunc(content)
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

		if err := renderFunc(content, w); err != nil {
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

// Built-in format constants for common output formats
var (
	JSON     = Format{Name: "json", Renderer: &jsonRenderer{}}
	YAML     = Format{Name: "yaml", Renderer: &yamlRenderer{}}
	CSV      = Format{Name: "csv", Renderer: &csvRenderer{}}
	HTML     = Format{Name: "html", Renderer: &htmlRenderer{}}
	Table    = Format{Name: "table", Renderer: &tableRenderer{}}
	Markdown = Format{Name: "markdown", Renderer: &markdownRenderer{}}
	DOT      = Format{Name: "dot", Renderer: &dotRenderer{}}
	Mermaid  = Format{Name: "mermaid", Renderer: &mermaidRenderer{}}
	DrawIO   = Format{Name: "drawio", Renderer: &drawioRenderer{}}
)

// jsonRenderer implements JSON output format
type jsonRenderer struct {
	baseRenderer
}

func (j *jsonRenderer) Format() string {
	return "json"
}

func (j *jsonRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return j.renderDocument(ctx, doc, j.renderContent)
}

func (j *jsonRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return j.renderDocumentTo(ctx, doc, w, j.renderContentTo)
}

func (j *jsonRenderer) SupportsStreaming() bool {
	return true
}

// yamlRenderer implements YAML output format
type yamlRenderer struct {
	baseRenderer
}

func (y *yamlRenderer) Format() string {
	return "yaml"
}

func (y *yamlRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return y.renderDocument(ctx, doc, y.renderContent)
}

func (y *yamlRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return y.renderDocumentTo(ctx, doc, w, y.renderContentTo)
}

func (y *yamlRenderer) SupportsStreaming() bool {
	return true
}

// csvRenderer implements CSV output format
type csvRenderer struct {
	baseRenderer
}

func (c *csvRenderer) Format() string {
	return "csv"
}

func (c *csvRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return c.renderDocument(ctx, doc, c.renderContent)
}

func (c *csvRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return c.renderDocumentTo(ctx, doc, w, c.renderContentTo)
}

func (c *csvRenderer) SupportsStreaming() bool {
	return true
}

// htmlRenderer implements HTML output format
type htmlRenderer struct {
	baseRenderer
}

func (h *htmlRenderer) Format() string {
	return "html"
}

func (h *htmlRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return h.renderDocument(ctx, doc, h.renderContent)
}

func (h *htmlRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return h.renderDocumentTo(ctx, doc, w, h.renderContentTo)
}

func (h *htmlRenderer) SupportsStreaming() bool {
	return true
}

// tableRenderer implements table output format
type tableRenderer struct {
	baseRenderer
}

func (t *tableRenderer) Format() string {
	return "table"
}

func (t *tableRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return t.renderDocument(ctx, doc, t.renderContent)
}

func (t *tableRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return t.renderDocumentTo(ctx, doc, w, t.renderContentTo)
}

func (t *tableRenderer) SupportsStreaming() bool {
	return true
}

// markdownRenderer implements Markdown output format
type markdownRenderer struct {
	baseRenderer
}

func (m *markdownRenderer) Format() string {
	return "markdown"
}

func (m *markdownRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return m.renderDocument(ctx, doc, m.renderContent)
}

func (m *markdownRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return m.renderDocumentTo(ctx, doc, w, m.renderContentTo)
}

func (m *markdownRenderer) SupportsStreaming() bool {
	return true
}

// dotRenderer implements DOT (Graphviz) output format
type dotRenderer struct {
	baseRenderer
}

func (d *dotRenderer) Format() string {
	return "dot"
}

func (d *dotRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return d.renderDocument(ctx, doc, d.renderContent)
}

func (d *dotRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return d.renderDocumentTo(ctx, doc, w, d.renderContentTo)
}

func (d *dotRenderer) SupportsStreaming() bool {
	return false // DOT format typically doesn't benefit from streaming
}

// mermaidRenderer implements Mermaid diagram output format
type mermaidRenderer struct {
	baseRenderer
}

func (m *mermaidRenderer) Format() string {
	return "mermaid"
}

func (m *mermaidRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return m.renderDocument(ctx, doc, m.renderContent)
}

func (m *mermaidRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return m.renderDocumentTo(ctx, doc, w, m.renderContentTo)
}

func (m *mermaidRenderer) SupportsStreaming() bool {
	return false // Mermaid format typically doesn't benefit from streaming
}

// drawioRenderer implements Draw.io XML output format
type drawioRenderer struct {
	baseRenderer
}

func (d *drawioRenderer) Format() string {
	return "drawio"
}

func (d *drawioRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return d.renderDocument(ctx, doc, d.renderContent)
}

func (d *drawioRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return d.renderDocumentTo(ctx, doc, w, d.renderContentTo)
}

func (d *drawioRenderer) SupportsStreaming() bool {
	return false // Draw.io format typically doesn't benefit from streaming
}
