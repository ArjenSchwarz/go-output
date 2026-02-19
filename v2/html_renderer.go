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

	// HTMLAppendMarker is the HTML comment marker used for append mode operations.
	//
	// When a FileWriter or S3Writer operates in append mode with HTML format, this marker
	// indicates the insertion point for new content. New HTML fragments are inserted
	// immediately before this marker, allowing multiple appends while preserving the
	// overall HTML structure.
	//
	// The marker is automatically included when rendering a full HTML page (useTemplate=true).
	// It is positioned near the end of the document, before the closing </body> and </html> tags.
	//
	// HTML fragments (useTemplate=false) do not include the marker, as they are meant to be
	// inserted into existing HTML documents that already contain it.
	//
	// Required for Append Mode: When appending to an existing HTML file, the file MUST
	// contain this marker or an error will be returned. The FileWriter will not attempt
	// to guess an insertion point.
	//
	// Marker Location: The marker should appear only once in the document and should be
	// placed before any closing tags (</body>, </html>). Multiple markers or incorrectly
	// positioned markers will cause undefined behavior.
	HTMLAppendMarker = "<!-- go-output-append -->"
)

// htmlRenderer implements HTML output format
type htmlRenderer struct {
	baseRenderer
	collapsibleConfig RendererConfig
	useTemplate       bool          // Enable/disable template wrapping
	template          *HTMLTemplate // Template configuration (nil = use default)
}

func (h *htmlRenderer) Format() string {
	return FormatHTML
}

func (h *htmlRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	// Render the document
	result, err := h.renderDocumentWithFormat(ctx, doc, h.renderContent, FormatHTML)
	if err != nil {
		return nil, err
	}

	// Check if document contains any ChartContent that would need mermaid.js
	if h.documentContainsMermaidCharts(doc) {
		// Inject mermaid.js script
		result = h.injectMermaidScript(result)
	}

	// Wrap in template if enabled
	if h.useTemplate {
		result = h.wrapInTemplate(result, h.template)
	}

	return result, nil
}

func (h *htmlRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	// Write template header if needed
	if h.useTemplate {
		tmpl := h.template
		if tmpl == nil {
			tmpl = DefaultHTMLTemplate
		}
		header := h.getTemplateHeader(tmpl)
		if _, err := w.Write(header); err != nil {
			return err
		}
	}

	// Render document content
	if err := h.renderDocumentTo(ctx, doc, w, h.renderContentTo); err != nil {
		return err
	}

	// Check if document contains any ChartContent that would need mermaid.js
	if h.documentContainsMermaidCharts(doc) {
		if _, err := w.Write([]byte(h.getMermaidScript())); err != nil {
			return err
		}
	}

	// Write template footer if needed
	if h.useTemplate {
		tmpl := h.template
		if tmpl == nil {
			tmpl = DefaultHTMLTemplate
		}
		footer := h.getTemplateFooter(tmpl)
		if _, err := w.Write(footer); err != nil {
			return err
		}
	}

	return nil
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
	case *ChartContent:
		return h.renderChartContentHTML(c)
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
		fmt.Fprintf(&result, "<h3>%s</h3>\n", html.EscapeString(table.Title()))
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
		fmt.Fprintf(&result, "        <th>%s</th>\n", html.EscapeString(key))
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
			fmt.Fprintf(&result, "        <td>%s</td>\n", cellValue)
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
	fmt.Fprintf(&result, "<%s class=\"%s\"%s>%s</%s>\n",
		tag, cssClass, styleAttr, html.EscapeString(text.Text()), tag)

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
	fmt.Fprintf(&result, "  <h%d>%s</h%d>\n", headingLevel, html.EscapeString(section.Title()), headingLevel)
	result.WriteString("  <div class=\"section-content\">\n")

	// Render nested content
	for _, content := range section.Contents() {
		// Apply per-content transformations before rendering
		transformed, err := applyContentTransformations(context.Background(), content)
		if err != nil {
			return nil, err
		}

		contentHTML, err := h.renderContent(transformed)
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

// renderChartContentHTML renders chart content as HTML with mermaid class
func (h *htmlRenderer) renderChartContentHTML(chart *ChartContent) ([]byte, error) {
	// Use mermaid renderer to generate the chart syntax
	mermaidRenderer := &mermaidRenderer{}

	// Create a temporary document with just this chart
	tempDoc := &Document{
		contents: []Content{chart},
	}

	mermaidData, err := mermaidRenderer.Render(context.Background(), tempDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart as mermaid: %w", err)
	}

	// Wrap in <pre class="mermaid">
	var result strings.Builder
	result.WriteString("<pre class=\"mermaid\">\n")
	result.Write(mermaidData)
	result.WriteString("</pre>\n")

	return []byte(result.String()), nil
}

// documentContainsMermaidCharts checks if a document contains any ChartContent
func (h *htmlRenderer) documentContainsMermaidCharts(doc *Document) bool {
	for _, content := range doc.GetContents() {
		if _, ok := content.(*ChartContent); ok {
			return true
		}
		// Check nested content in sections
		if section, ok := content.(*SectionContent); ok {
			if h.sectionContainsMermaidCharts(section) {
				return true
			}
		}
	}
	return false
}

// sectionContainsMermaidCharts recursively checks if a section contains ChartContent
func (h *htmlRenderer) sectionContainsMermaidCharts(section *SectionContent) bool {
	for _, content := range section.Contents() {
		if _, ok := content.(*ChartContent); ok {
			return true
		}
		if nestedSection, ok := content.(*SectionContent); ok {
			if h.sectionContainsMermaidCharts(nestedSection) {
				return true
			}
		}
	}
	return false
}

// getMermaidScript returns the mermaid.js script as a string
func (h *htmlRenderer) getMermaidScript() string {
	return `<script type="module">
    import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
    mermaid.initialize({ startOnLoad: true });
  </script>
`
}

// injectMermaidScript adds the mermaid.js script to the HTML output
func (h *htmlRenderer) injectMermaidScript(html []byte) []byte {
	// Append the script at the end of the HTML
	var result strings.Builder
	result.Write(html)
	result.WriteString(h.getMermaidScript())
	return []byte(result.String())
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

	fmt.Fprintf(&result, `<section class="%s">`, sectionClass)
	fmt.Fprintf(&result, `<details%s class="%s">`, openAttr, cssClasses["details"])
	fmt.Fprintf(&result, `<summary class="%s">%s</summary>`,
		cssClasses["summary"], html.EscapeString(section.Title()))

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

// getTemplateHeader returns the HTML header portion of the template (up to <body>)
// This is used for streaming output where the header is written before content.
func (h *htmlRenderer) getTemplateHeader(tmpl *HTMLTemplate) []byte {
	if tmpl == nil {
		tmpl = DefaultHTMLTemplate
	}

	var buf strings.Builder

	// DOCTYPE
	buf.WriteString("<!DOCTYPE html>\n")

	// HTML element with lang attribute
	fmt.Fprintf(&buf, "<html lang=\"%s\">\n", html.EscapeString(tmpl.Language))

	// Head section
	buf.WriteString("<head>\n")
	fmt.Fprintf(&buf, "  <meta charset=\"%s\">\n", html.EscapeString(tmpl.Charset))

	if tmpl.Viewport != "" {
		fmt.Fprintf(&buf, "  <meta name=\"viewport\" content=\"%s\">\n",
			html.EscapeString(tmpl.Viewport))
	}

	fmt.Fprintf(&buf, "  <title>%s</title>\n", html.EscapeString(tmpl.Title))

	// Additional meta tags (after title)
	if tmpl.Description != "" {
		fmt.Fprintf(&buf, "  <meta name=\"description\" content=\"%s\">\n",
			html.EscapeString(tmpl.Description))
	}
	if tmpl.Author != "" {
		fmt.Fprintf(&buf, "  <meta name=\"author\" content=\"%s\">\n",
			html.EscapeString(tmpl.Author))
	}

	// Custom meta tags
	for name, content := range tmpl.MetaTags {
		fmt.Fprintf(&buf, "  <meta name=\"%s\" content=\"%s\">\n",
			html.EscapeString(name), html.EscapeString(content))
	}

	// External stylesheets
	for _, href := range tmpl.ExternalCSS {
		fmt.Fprintf(&buf, "  <link rel=\"stylesheet\" href=\"%s\">\n",
			html.EscapeString(href))
	}

	// Embedded CSS
	if tmpl.CSS != "" {
		buf.WriteString("  <style>\n")
		buf.WriteString(tmpl.CSS) // CSS is NOT escaped (assumed safe)
		buf.WriteString("\n  </style>\n")
	}

	// Theme overrides (CSS custom property overrides)
	if len(tmpl.ThemeOverrides) > 0 {
		buf.WriteString("  <style>\n")
		buf.WriteString("    :root {\n")
		for prop, value := range tmpl.ThemeOverrides {
			fmt.Fprintf(&buf, "      %s: %s;\n",
				html.EscapeString(prop), html.EscapeString(value))
		}
		buf.WriteString("    }\n")
		buf.WriteString("  </style>\n")
	}

	// Additional head content
	if tmpl.HeadExtra != "" {
		buf.WriteString(tmpl.HeadExtra) // NOT escaped (assumed safe, user responsibility)
	}

	buf.WriteString("</head>\n")

	// Body section
	bodyTag := "<body"
	if tmpl.BodyClass != "" {
		bodyTag += fmt.Sprintf(" class=\"%s\"", html.EscapeString(tmpl.BodyClass))
	}
	for attr, value := range tmpl.BodyAttrs {
		bodyTag += fmt.Sprintf(" %s=\"%s\"",
			html.EscapeString(attr), html.EscapeString(value))
	}
	bodyTag += ">\n"
	buf.WriteString(bodyTag)

	return []byte(buf.String())
}

// getTemplateFooter returns the HTML footer portion of the template (from </body> to end)
// This is used for streaming output where the footer is written after content.
// When useTemplate is true, includes the HTMLAppendMarker before closing tags.
func (h *htmlRenderer) getTemplateFooter(tmpl *HTMLTemplate) []byte {
	if tmpl == nil {
		tmpl = DefaultHTMLTemplate
	}

	var buf strings.Builder

	// Additional body content
	if tmpl.BodyExtra != "" {
		buf.WriteString(tmpl.BodyExtra) // NOT escaped (scripts, etc.)
	}

	// Include append marker before closing body tag when using template
	if h.useTemplate {
		buf.WriteString("\n")
		buf.WriteString(HTMLAppendMarker)
	}

	buf.WriteString("\n</body>\n</html>\n")

	return []byte(buf.String())
}

// wrapInTemplate wraps rendered HTML fragment in a complete HTML5 document using the provided template.
// All user-controlled fields are HTML-escaped to prevent XSS injection.
// CSS and extra content fields (CSS, HeadExtra, BodyExtra) are included as-is (user responsibility for safety).
// When useTemplate is true, includes the HTMLAppendMarker before closing body tag.
func (h *htmlRenderer) wrapInTemplate(fragmentHTML []byte, tmpl *HTMLTemplate) []byte {
	var result strings.Builder

	// Use header and footer helpers
	result.Write(h.getTemplateHeader(tmpl))
	result.Write(fragmentHTML)
	result.Write(h.getTemplateFooter(tmpl))

	return []byte(result.String())
}
