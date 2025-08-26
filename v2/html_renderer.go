package output

import (
	"context"
	"fmt"
	"html"
	"io"
	"strings"
)

// Constants for repeated strings
const (
	openAttribute = " open"
)

// htmlRenderer implements HTML output format
type htmlRenderer struct {
	baseRenderer
	collapsibleConfig RendererConfig
}

func (h *htmlRenderer) Format() string {
	return FormatHTML
}

func (h *htmlRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return h.renderDocumentWithFormat(ctx, doc, h.renderContent, FormatHTML)
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
	case *DefaultCollapsibleSection:
		return h.renderCollapsibleSection(c)
	default:
		// Fallback to basic rendering with HTML escaping
		data, err := h.baseRenderer.renderContent(content)
		if err != nil {
			return nil, err
		}
		escaped := html.EscapeString(string(data))
		return fmt.Appendf(nil, "<pre>%s</pre>\n", escaped), nil
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
				cellValue = h.formatCellValue(val, field)
			}
			result.WriteString(fmt.Sprintf("        <td>%s</td>\n", cellValue))
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
	if raw.Format() == FormatHTML {
		// Include raw HTML directly - this is a security risk and should be documented
		return raw.Data(), nil
	} else {
		// Escape non-HTML raw content
		escaped := html.EscapeString(string(raw.Data()))
		return fmt.Appendf(nil, "<pre class=\"raw-content\">%s</pre>\n", escaped), nil
	}
}

// renderSectionContentHTML renders section content as HTML with nested content
func (h *htmlRenderer) renderSectionContentHTML(section *SectionContent) ([]byte, error) {
	var result strings.Builder

	// Create section with appropriate heading level
	headingLevel := min(max(section.Level(), 1), 6)

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
		lines := strings.SplitSeq(string(contentHTML), "\n")
		for line := range lines {
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

// formatCellValue processes field values and handles CollapsibleValue interface
func (h *htmlRenderer) formatCellValue(val any, field *Field) string {
	// Apply field formatter first using base renderer method
	processed := h.processFieldValue(val, field)

	// Check if result is CollapsibleValue (Requirement 7.1)
	if cv, ok := processed.(CollapsibleValue); ok {
		return h.renderCollapsibleValue(cv)
	}

	// Handle regular values with HTML escaping (maintain backward compatibility)
	return html.EscapeString(fmt.Sprint(processed))
}

// renderCollapsibleValue renders a CollapsibleValue as HTML5 details element
func (h *htmlRenderer) renderCollapsibleValue(cv CollapsibleValue) string {
	// Check global expansion override
	expanded := cv.IsExpanded() || h.collapsibleConfig.ForceExpansion

	openAttr := ""
	if expanded {
		openAttr = openAttribute // Requirement 7.3: add open attribute
	}

	// Get CSS classes from configuration (Requirement 7.2)
	classes := h.collapsibleConfig.HTMLCSSClasses

	// Check if the value implements code fence configuration
	var useCodeFences bool
	var codeLanguage string
	if dcv, ok := cv.(*DefaultCollapsibleValue); ok {
		useCodeFences = dcv.UseCodeFences()
		codeLanguage = dcv.CodeLanguage()
	}

	// Format details with potential code fence wrapping
	var detailsHTML string
	if useCodeFences {
		detailsHTML = h.formatDetailsAsHTMLWithCodeFences(cv.Details(), codeLanguage)
	} else {
		detailsHTML = h.formatDetailsAsHTML(cv.Details())
	}

	// Use HTML5 details element with semantic classes (Requirements 7.1, 7.2)
	// Add <br/> after summary like markdown renderer does
	return fmt.Sprintf(`<details%s class="%s"><summary class="%s">%s</summary><br/><div class="%s">%s</div></details>`,
		openAttr,
		classes["details"],
		classes["summary"],
		html.EscapeString(cv.Summary()), // Requirement 7.4: escape HTML
		classes["content"],
		detailsHTML)
}

// formatDetailsAsHTML formats details content as appropriate HTML (Requirement 7.5)
func (h *htmlRenderer) formatDetailsAsHTML(details any) string {
	switch d := details.(type) {
	case []string:
		// Format as plain text joined with <br/> tags (consistent with markdown renderer)
		if len(d) == 0 {
			return ""
		}
		escaped := make([]string, len(d))
		for i, item := range d {
			escaped[i] = html.EscapeString(item)
		}
		return strings.Join(escaped, "<br/>")
	case map[string]any:
		// Format as key-value pairs joined with <br/> tags (consistent with markdown renderer)
		if len(d) == 0 {
			return ""
		}
		var parts []string
		for k, v := range d {
			parts = append(parts, fmt.Sprintf("<strong>%s:</strong> %s",
				html.EscapeString(k), html.EscapeString(fmt.Sprint(v))))
		}
		return strings.Join(parts, "<br/>")
	case string:
		// Replace newlines with <br> tags for proper HTML rendering
		escaped := html.EscapeString(d)
		return strings.ReplaceAll(escaped, "\n", "<br>")
	default:
		return html.EscapeString(fmt.Sprint(details))
	}
}

// formatDetailsAsHTMLWithCodeFences formats details content wrapped in HTML code blocks
func (h *htmlRenderer) formatDetailsAsHTMLWithCodeFences(details any, language string) string {
	// Get the raw content as a string
	var content string
	switch d := details.(type) {
	case []string:
		if len(d) == 0 {
			return ""
		}
		content = strings.Join(d, "\n")
	case map[string]any:
		if len(d) == 0 {
			return ""
		}
		// Format map as key: value pairs, one per line
		var lines []string
		for k, v := range d {
			lines = append(lines, fmt.Sprintf("%s: %s", k, v))
		}
		content = strings.Join(lines, "\n")
	case string:
		content = d
	default:
		content = fmt.Sprint(details)
	}

	// Create HTML code block with optional language class
	langClass := ""
	if language != "" {
		langClass = fmt.Sprintf(` class="language-%s"`, html.EscapeString(language))
	}

	// Wrap in pre and code tags for proper HTML code formatting
	return fmt.Sprintf("<pre><code%s>%s</code></pre>", langClass, html.EscapeString(content))
}

// renderCollapsibleSection renders a CollapsibleSection as semantic HTML5 elements (Requirement 15.6)
func (h *htmlRenderer) renderCollapsibleSection(section *DefaultCollapsibleSection) ([]byte, error) {
	var result strings.Builder

	openAttr := ""
	if section.IsExpanded() || h.collapsibleConfig.ForceExpansion {
		openAttr = openAttribute
	}

	// Create semantic section with collapsible behavior (Requirement 15.6)
	cssClasses := h.collapsibleConfig.HTMLCSSClasses
	sectionClass := cssClasses["section"]
	if sectionClass == "" {
		sectionClass = "collapsible-section"
	}

	result.WriteString(fmt.Sprintf(`<section class="%s">`, sectionClass))
	result.WriteString(fmt.Sprintf(`<details%s class="%s">`, openAttr, cssClasses["details"]))
	result.WriteString(fmt.Sprintf(`<summary class="%s">%s</summary>`,
		cssClasses["summary"], html.EscapeString(section.Title())))

	result.WriteString(`<div class="section-content" style="margin-left: 20px; padding-left: 10px;">`)

	// Render all nested content with indentation (Requirement 15.6)
	for _, content := range section.Content() {
		contentHTML, err := h.renderContent(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render section content: %w", err)
		}

		// Indent the content for better visual hierarchy
		lines := strings.SplitSeq(string(contentHTML), "\n")
		for line := range lines {
			if strings.TrimSpace(line) != "" {
				// Use Unicode En spaces for indentation (U+2002) - preserves spacing without HTML escaping issues
				result.WriteString("\u2002\u2002") // Add 2 spaces for indentation
				result.WriteString(line)
				result.WriteString("\n")
			}
		}
	}

	result.WriteString(`</div>`)
	result.WriteString(`</details>`)
	result.WriteString(`</section>`)

	return []byte(result.String()), nil
}
