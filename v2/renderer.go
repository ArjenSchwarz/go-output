package output

import (
	"context"
	"fmt"
	"io"
)

// Format name constants
const (
	FormatJSON     = "json"
	FormatYAML     = "yaml"
	FormatMarkdown = "markdown"
	FormatTable    = "table"
	FormatCSV      = "csv"
	FormatHTML     = "html"
	FormatText     = "text"
	FormatDOT      = "dot"
	FormatMermaid  = "mermaid"
	FormatDrawIO   = "drawio"
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
	JSON         = Format{Name: FormatJSON, Renderer: &jsonRenderer{}}
	YAML         = Format{Name: FormatYAML, Renderer: &yamlRenderer{}}
	CSV          = Format{Name: FormatCSV, Renderer: &csvRenderer{}}
	HTML         = Format{Name: FormatHTML, Renderer: &htmlRenderer{useTemplate: true, template: DefaultHTMLTemplate}}
	HTMLFragment = Format{Name: FormatHTML, Renderer: &htmlRenderer{useTemplate: false}}
	Table        = Format{Name: FormatTable, Renderer: &tableRenderer{}}
	Markdown     = Format{Name: FormatMarkdown, Renderer: &markdownRenderer{headingLevel: 1}}
	DOT          = Format{Name: FormatDOT, Renderer: &dotRenderer{}}
	Mermaid      = Format{Name: FormatMermaid, Renderer: &mermaidRenderer{}}
	DrawIO       = Format{Name: FormatDrawIO, Renderer: &drawioRenderer{}}
)

// Table style format constants for v1 compatibility
var (
	TableDefault       = Format{Name: FormatTable, Renderer: NewTableRendererWithStyle("Default")}
	TableBold          = Format{Name: FormatTable, Renderer: NewTableRendererWithStyle("Bold")}
	TableColoredBright = Format{Name: FormatTable, Renderer: NewTableRendererWithStyle("ColoredBright")}
	TableLight         = Format{Name: FormatTable, Renderer: NewTableRendererWithStyle("Light")}
	TableRounded       = Format{Name: FormatTable, Renderer: NewTableRendererWithStyle("Rounded")}
)

// TableWithStyle creates a table format with the specified style for v1 compatibility
func TableWithStyle(styleName string) Format {
	return Format{
		Name:     FormatTable,
		Renderer: NewTableRendererWithStyle(styleName),
	}
}

// TableWithMaxColumnWidth creates a table format with maximum column width
func TableWithMaxColumnWidth(maxColumnWidth int) Format {
	return Format{
		Name:     FormatTable,
		Renderer: NewTableRendererWithStyleAndWidth("Default", maxColumnWidth),
	}
}

// TableWithStyleAndMaxColumnWidth creates a table format with specified style and maximum column width
func TableWithStyleAndMaxColumnWidth(styleName string, maxColumnWidth int) Format {
	return Format{
		Name:     FormatTable,
		Renderer: NewTableRendererWithStyleAndWidth(styleName, maxColumnWidth),
	}
}

// MarkdownWithToC creates a markdown format with table of contents for v1 compatibility
func MarkdownWithToC(enabled bool) Format {
	return Format{
		Name:     FormatMarkdown,
		Renderer: NewMarkdownRendererWithToC(enabled),
	}
}

// MarkdownWithFrontMatter creates a markdown format with front matter for v1 compatibility
func MarkdownWithFrontMatter(frontMatter map[string]string) Format {
	return Format{
		Name:     FormatMarkdown,
		Renderer: NewMarkdownRendererWithFrontMatter(frontMatter),
	}
}

// MarkdownWithOptions creates a markdown format with ToC and front matter for v1 compatibility
func MarkdownWithOptions(includeToC bool, frontMatter map[string]string) Format {
	return Format{
		Name:     FormatMarkdown,
		Renderer: NewMarkdownRendererWithOptions(includeToC, frontMatter),
	}
}

// applyContentTransformations applies transformations to content during rendering
// This is a helper function used by renderers to execute per-content transformations
func applyContentTransformations(ctx context.Context, content Content) (Content, error) {
	transformations := content.GetTransformations()
	if len(transformations) == 0 {
		return content, nil // No transformations
	}

	// Start with a clone to preserve immutability
	current := content.Clone()

	// Apply each transformation in sequence
	for i, op := range transformations {
		// Check context cancellation before operation
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("content %s transformation cancelled: %w",
				content.ID(), err)
		}

		// Validate operation configuration
		if err := op.Validate(); err != nil {
			return nil, fmt.Errorf("content %s transformation %d (%s) invalid: %w",
				content.ID(), i, op.Name(), err)
		}

		// Apply the operation
		result, err := op.Apply(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("content %s transformation %d (%s) failed: %w",
				content.ID(), i, op.Name(), err)
		}
		current = result
	}

	return current, nil
}

// HTMLWithTemplate creates an HTML format with a custom template
// If template is nil, produces fragment output (no template wrapping)
// If template is provided, produces complete HTML document with the template
func HTMLWithTemplate(template *HTMLTemplate) Format {
	return Format{
		Name:     FormatHTML,
		Renderer: &htmlRenderer{useTemplate: template != nil, template: template},
	}
}
