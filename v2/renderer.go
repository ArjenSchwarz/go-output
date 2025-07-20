package output

import (
	"context"
	"io"
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

// Built-in format constants for common output formats
var (
	JSON     = Format{Name: "json", Renderer: &jsonRenderer{}}
	YAML     = Format{Name: "yaml", Renderer: &yamlRenderer{}}
	CSV      = Format{Name: "csv", Renderer: &csvRenderer{}}
	HTML     = Format{Name: "html", Renderer: &htmlRenderer{}}
	Table    = Format{Name: "table", Renderer: &tableRenderer{}}
	Markdown = Format{Name: "markdown", Renderer: &markdownRenderer{headingLevel: 1}}
	DOT      = Format{Name: "dot", Renderer: &dotRenderer{}}
	Mermaid  = Format{Name: "mermaid", Renderer: &mermaidRenderer{}}
	DrawIO   = Format{Name: "drawio", Renderer: &drawioRenderer{}}
)

// Table style format constants for v1 compatibility
var (
	TableDefault       = Format{Name: "table", Renderer: NewTableRendererWithStyle("Default")}
	TableBold          = Format{Name: "table", Renderer: NewTableRendererWithStyle("Bold")}
	TableColoredBright = Format{Name: "table", Renderer: NewTableRendererWithStyle("ColoredBright")}
	TableLight         = Format{Name: "table", Renderer: NewTableRendererWithStyle("Light")}
	TableRounded       = Format{Name: "table", Renderer: NewTableRendererWithStyle("Rounded")}
)

// TableWithStyle creates a table format with the specified style for v1 compatibility
func TableWithStyle(styleName string) Format {
	return Format{
		Name:     "table",
		Renderer: NewTableRendererWithStyle(styleName),
	}
}

// MarkdownWithToC creates a markdown format with table of contents for v1 compatibility
func MarkdownWithToC(enabled bool) Format {
	return Format{
		Name:     "markdown",
		Renderer: NewMarkdownRendererWithToC(enabled),
	}
}

// MarkdownWithFrontMatter creates a markdown format with front matter for v1 compatibility
func MarkdownWithFrontMatter(frontMatter map[string]string) Format {
	return Format{
		Name:     "markdown",
		Renderer: NewMarkdownRendererWithFrontMatter(frontMatter),
	}
}

// MarkdownWithOptions creates a markdown format with ToC and front matter for v1 compatibility
func MarkdownWithOptions(includeToC bool, frontMatter map[string]string) Format {
	return Format{
		Name:     "markdown",
		Renderer: NewMarkdownRendererWithOptions(includeToC, frontMatter),
	}
}
