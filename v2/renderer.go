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

// Built-in format constructor functions for common output formats
// These functions return new Format instances with fresh renderers to ensure thread safety.
// Each call creates a new renderer instance, preventing shared mutable state issues.
//
// IMPORTANT: Do not store these Format values in global variables and mutate their Renderer field.
// Always call these functions when you need a Format to get a fresh instance.

// JSON returns a Format configured for JSON output
func JSON() Format {
	return Format{Name: FormatJSON, Renderer: &jsonRenderer{}}
}

// YAML returns a Format configured for YAML output
func YAML() Format {
	return Format{Name: FormatYAML, Renderer: &yamlRenderer{}}
}

// CSV returns a Format configured for CSV output
func CSV() Format {
	return Format{Name: FormatCSV, Renderer: &csvRenderer{}}
}

// HTML returns a Format configured for complete HTML documents with the default template
func HTML() Format {
	return Format{Name: FormatHTML, Renderer: &htmlRenderer{useTemplate: true, template: DefaultHTMLTemplate}}
}

// HTMLFragment returns a Format configured for HTML fragments without template wrapping
func HTMLFragment() Format {
	return Format{Name: FormatHTML, Renderer: &htmlRenderer{useTemplate: false}}
}

// Table returns a Format configured for terminal table output with default style
func Table() Format {
	return Format{Name: FormatTable, Renderer: &tableRenderer{}}
}

// Markdown returns a Format configured for Markdown output
func Markdown() Format {
	return Format{Name: FormatMarkdown, Renderer: &markdownRenderer{headingLevel: 1}}
}

// DOT returns a Format configured for GraphViz DOT output
func DOT() Format {
	return Format{Name: FormatDOT, Renderer: &dotRenderer{}}
}

// Mermaid returns a Format configured for Mermaid diagram output
func Mermaid() Format {
	return Format{Name: FormatMermaid, Renderer: &mermaidRenderer{}}
}

// DrawIO returns a Format configured for Draw.io XML output
func DrawIO() Format {
	return Format{Name: FormatDrawIO, Renderer: &drawioRenderer{}}
}

// Table style format constructors for v1 compatibility

// TableDefault returns a Format configured for terminal table output with Default style
func TableDefault() Format {
	return TableWithStyle("Default")
}

// TableBold returns a Format configured for terminal table output with Bold style
func TableBold() Format {
	return TableWithStyle("Bold")
}

// TableColoredBright returns a Format configured for terminal table output with ColoredBright style
func TableColoredBright() Format {
	return TableWithStyle("ColoredBright")
}

// TableLight returns a Format configured for terminal table output with Light style
func TableLight() Format {
	return TableWithStyle("Light")
}

// TableRounded returns a Format configured for terminal table output with Rounded style
func TableRounded() Format {
	return TableWithStyle("Rounded")
}

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
