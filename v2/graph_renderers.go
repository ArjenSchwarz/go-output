package output

import (
	"context"
	"io"
)

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
	return false
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
	return false
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
	return false
}
