package output

import (
	"context"
	"fmt"
	"html"
	"io"
	"strings"
)

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

// renderContent renders content specifically for HTML format
func (h *htmlRenderer) renderContent(content Content) ([]byte, error) {
	switch c := content.(type) {
	case *TableContent:
		return h.renderTableContentHTML(c)
	case *TextContent:
		return h.renderTextContentHTML(c)
	case *RawContent:
		return h.renderRawContentHTML(c)
	case *SectionContent:
		return h.renderSectionContentHTML(c)
	default:
		// Fallback to basic rendering with HTML escaping
		data, err := h.baseRenderer.renderContent(content)
		if err != nil {
			return nil, err
		}
		escaped := html.EscapeString(string(data))
		return []byte(fmt.Sprintf("<pre>%s</pre>\n", escaped)), nil
	}
}

// renderContentTo renders content to a writer for HTML format
func (h *htmlRenderer) renderContentTo(content Content, w io.Writer) error {
	data, err := h.renderContent(content)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// renderTableContentHTML renders table content as HTML with proper escaping
func (h *htmlRenderer) renderTableContentHTML(table *TableContent) ([]byte, error) {
	var result strings.Builder

	// Add title if present
	if table.Title() != "" {
		result.WriteString(fmt.Sprintf("<h3>%s</h3>\n", html.EscapeString(table.Title())))
	}

	result.WriteString("<div class=\"table-container\">\n")
	result.WriteString("  <table class=\"data-table\">\n")

	// Get key order from schema
	keyOrder := table.Schema().GetKeyOrder()
	if len(keyOrder) == 0 {
		result.WriteString("  </table>\n</div>\n")
		return []byte(result.String()), nil // No columns to render
	}

	// Write header
	result.WriteString("    <thead>\n")
	result.WriteString("      <tr>\n")
	for _, key := range keyOrder {
		result.WriteString(fmt.Sprintf("        <th>%s</th>\n", html.EscapeString(key)))
	}
	result.WriteString("      </tr>\n")
	result.WriteString("    </thead>\n")

	// Write body
	result.WriteString("    <tbody>\n")
	for _, record := range table.Records() {
		result.WriteString("      <tr>\n")
		for _, key := range keyOrder {
			var cellValue string
			if val, exists := record[key]; exists {
				// Apply field formatter if available
				field := table.Schema().FindField(key)
				if field != nil && field.Formatter != nil {
					cellValue = field.Formatter(val)
				} else {
					cellValue = fmt.Sprint(val)
				}
			}
			result.WriteString(fmt.Sprintf("        <td>%s</td>\n", html.EscapeString(cellValue)))
		}
		result.WriteString("      </tr>\n")
	}
	result.WriteString("    </tbody>\n")

	result.WriteString("  </table>\n</div>\n")

	return []byte(result.String()), nil
}

// renderTextContentHTML renders text content as HTML with proper escaping and styling
func (h *htmlRenderer) renderTextContentHTML(text *TextContent) ([]byte, error) {
	var result strings.Builder

	style := text.Style()

	// Determine the appropriate HTML tag based on style
	var tag string
	var cssClass string
	var styleAttr string

	if style.Header {
		tag = "h2"
		cssClass = "text-header"
	} else {
		tag = "p"
		cssClass = "text-content"
	}

	// Build inline styles
	var styles []string
	if style.Bold {
		styles = append(styles, "font-weight: bold")
	}
	if style.Italic {
		styles = append(styles, "font-style: italic")
	}
	if style.Color != "" {
		styles = append(styles, fmt.Sprintf("color: %s", html.EscapeString(style.Color)))
	}
	if style.Size > 0 {
		styles = append(styles, fmt.Sprintf("font-size: %dpx", style.Size))
	}

	if len(styles) > 0 {
		styleAttr = fmt.Sprintf(" style=\"%s\"", strings.Join(styles, "; "))
	}

	// Create the HTML element
	result.WriteString(fmt.Sprintf("<%s class=\"%s\"%s>%s</%s>\n",
		tag, cssClass, styleAttr, html.EscapeString(text.Text()), tag))

	return []byte(result.String()), nil
}

// renderRawContentHTML renders raw content as HTML (with caution for security)
func (h *htmlRenderer) renderRawContentHTML(raw *RawContent) ([]byte, error) {
	// If the raw content is HTML format, include it directly (with warning)
	// Otherwise, escape it for safety
	if raw.Format() == "html" {
		// Include raw HTML directly - this is a security risk and should be documented
		return raw.Data(), nil
	} else {
		// Escape non-HTML raw content
		escaped := html.EscapeString(string(raw.Data()))
		return []byte(fmt.Sprintf("<pre class=\"raw-content\">%s</pre>\n", escaped)), nil
	}
}

// renderSectionContentHTML renders section content as HTML with nested content
func (h *htmlRenderer) renderSectionContentHTML(section *SectionContent) ([]byte, error) {
	var result strings.Builder

	// Create section with appropriate heading level
	headingLevel := section.Level()
	if headingLevel < 1 {
		headingLevel = 1
	}
	if headingLevel > 6 {
		headingLevel = 6
	}

	result.WriteString("<section class=\"content-section\">\n")
	result.WriteString(fmt.Sprintf("  <h%d>%s</h%d>\n", headingLevel, html.EscapeString(section.Title()), headingLevel))
	result.WriteString("  <div class=\"section-content\">\n")

	// Render nested content
	for _, content := range section.Contents() {
		contentHTML, err := h.renderContent(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render nested content: %w", err)
		}

		// Indent the content
		lines := strings.Split(string(contentHTML), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				result.WriteString("    ")
				result.WriteString(line)
				result.WriteString("\n")
			}
		}
	}

	result.WriteString("  </div>\n")
	result.WriteString("</section>\n")

	return []byte(result.String()), nil
}
