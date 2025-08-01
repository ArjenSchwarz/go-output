package output

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// markdownRenderer implements Markdown output format
type markdownRenderer struct {
	baseRenderer
	includeToC        bool
	frontMatter       map[string]string
	headingLevel      int
	collapsibleConfig RendererConfig
}

func (m *markdownRenderer) Format() string {
	return FormatMarkdown
}

func (m *markdownRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return m.renderDocumentMarkdown(ctx, doc)
}

func (m *markdownRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	data, err := m.renderDocumentMarkdown(ctx, doc)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (m *markdownRenderer) SupportsStreaming() bool {
	return true
}

// renderDocumentMarkdown renders the entire document as Markdown with front matter and ToC support
func (m *markdownRenderer) renderDocumentMarkdown(ctx context.Context, doc *Document) ([]byte, error) {
	var result strings.Builder

	// Add front matter if configured
	if len(m.frontMatter) > 0 {
		result.WriteString("---\n")
		for key, value := range m.frontMatter {
			result.WriteString(fmt.Sprintf("%s: %s\n", key, m.escapeYAMLValue(value)))
		}
		result.WriteString("---\n\n")
	}

	// Generate table of contents if enabled
	if m.includeToC {
		toc := m.generateTableOfContents(doc)
		if toc != "" {
			result.WriteString("## Table of Contents\n\n")
			result.WriteString(toc)
			result.WriteString("\n")
		}
	}

	// Render content
	for i, content := range doc.GetContents() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if i > 0 {
			result.WriteString("\n")
		}

		contentMD, err := m.renderContent(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render content %s: %w", content.ID(), err)
		}

		result.Write(contentMD)
	}

	return []byte(result.String()), nil
}

// generateTableOfContents creates a markdown table of contents from document sections
func (m *markdownRenderer) generateTableOfContents(doc *Document) string {
	var toc strings.Builder

	for _, content := range doc.GetContents() {
		switch c := content.(type) {
		case *SectionContent:
			// Add section to ToC
			level := c.Level()
			if level < 1 {
				level = 1
			}
			indent := strings.Repeat("  ", level-1)
			anchor := m.createMarkdownAnchor(c.Title())
			fmt.Fprintf(&toc, "%s- [%s](#%s)\n", indent, m.escapeMarkdown(c.Title()), anchor)

			// Recursively add subsections
			m.addSubsectionsToToC(&toc, c.Contents(), c.Level())

		case *TextContent:
			// Check if it's a header
			if c.Style().Header {
				anchor := m.createMarkdownAnchor(c.Text())
				fmt.Fprintf(&toc, "- [%s](#%s)\n", m.escapeMarkdown(c.Text()), anchor)
			}
		}
	}

	return toc.String()
}

// addSubsectionsToToC recursively adds subsections to the table of contents
func (m *markdownRenderer) addSubsectionsToToC(toc *strings.Builder, contents []Content, parentLevel int) {
	for _, content := range contents {
		switch c := content.(type) {
		case *SectionContent:
			level := parentLevel + 1
			if level < 1 {
				level = 1
			}
			indent := strings.Repeat("  ", level-1)
			anchor := m.createMarkdownAnchor(c.Title())
			fmt.Fprintf(toc, "%s- [%s](#%s)\n", indent, m.escapeMarkdown(c.Title()), anchor)
			m.addSubsectionsToToC(toc, c.Contents(), level)
		case *TextContent:
			if c.Style().Header {
				level := parentLevel + 1
				if level < 1 {
					level = 1
				}
				indent := strings.Repeat("  ", level-1)
				anchor := m.createMarkdownAnchor(c.Text())
				fmt.Fprintf(toc, "%s- [%s](#%s)\n", indent, m.escapeMarkdown(c.Text()), anchor)
			}
		}
	}
}

// renderContent renders content specifically for Markdown format
func (m *markdownRenderer) renderContent(content Content) ([]byte, error) {
	switch c := content.(type) {
	case *TableContent:
		return m.renderTableContentMarkdown(c)
	case *TextContent:
		return m.renderTextContentMarkdown(c)
	case *RawContent:
		return m.renderRawContentMarkdown(c)
	case *SectionContent:
		return m.renderSectionContentMarkdown(c)
	case *DefaultCollapsibleSection:
		return m.renderCollapsibleSection(c)
	default:
		// Fallback to basic rendering with markdown escaping
		data, err := m.baseRenderer.renderContent(content)
		if err != nil {
			return nil, err
		}
		escaped := m.escapeMarkdown(string(data))
		return []byte(escaped), nil
	}
}

// renderTableContentMarkdown renders table content as Markdown table
func (m *markdownRenderer) renderTableContentMarkdown(table *TableContent) ([]byte, error) {
	var result strings.Builder

	// Add title if present
	if table.Title() != "" {
		result.WriteString(fmt.Sprintf("### %s\n\n", m.escapeMarkdown(table.Title())))
	}

	// Get key order from schema
	keyOrder := table.Schema().GetKeyOrder()
	if len(keyOrder) == 0 {
		return []byte(""), nil // No columns to render
	}

	// Write header row
	result.WriteString("|")
	for _, key := range keyOrder {
		result.WriteString(fmt.Sprintf(" %s |", m.escapeMarkdown(key)))
	}
	result.WriteString("\n")

	// Write separator row
	result.WriteString("|")
	for range keyOrder {
		result.WriteString(" --- |")
	}
	result.WriteString("\n")

	// Write data rows
	for _, record := range table.Records() {
		result.WriteString("|")
		for _, key := range keyOrder {
			var cellValue string
			if val, exists := record[key]; exists {
				// Apply field formatter if available
				field := table.Schema().FindField(key)
				cellValue = m.formatCellValue(val, field)
			}
			// Escape markdown and handle newlines in table cells
			cellValue = m.escapeMarkdownTableCell(cellValue)
			result.WriteString(fmt.Sprintf(" %s |", cellValue))
		}
		result.WriteString("\n")
	}

	result.WriteString("\n")
	return []byte(result.String()), nil
}

// renderTextContentMarkdown renders text content as Markdown with styling
func (m *markdownRenderer) renderTextContentMarkdown(text *TextContent) ([]byte, error) {
	content := text.Text()
	style := text.Style()

	// Apply markdown formatting based on style
	if style.Header {
		// Use ## for headers (assuming main document title would be #)
		content = fmt.Sprintf("## %s\n\n", m.escapeMarkdown(content))
	} else {
		// Escape markdown characters first
		escapedContent := m.escapeMarkdown(content)

		// Apply inline formatting
		switch {
		case style.Bold:
			content = fmt.Sprintf("**%s**", escapedContent)
		case style.Italic:
			content = fmt.Sprintf("*%s*", escapedContent)
		default:
			content = escapedContent
		}

		// Color and size don't have standard markdown equivalents
		// Add as HTML if needed for enhanced markdown
		if style.Color != "" || style.Size > 0 {
			var htmlAttrs []string
			if style.Color != "" {
				htmlAttrs = append(htmlAttrs, fmt.Sprintf("color: %s", style.Color))
			}
			if style.Size > 0 {
				htmlAttrs = append(htmlAttrs, fmt.Sprintf("font-size: %dpx", style.Size))
			}
			styleAttr := strings.Join(htmlAttrs, "; ")
			content = fmt.Sprintf(`<span style="%s">%s</span>`, styleAttr, content)
		}

		content += "\n\n"
	}

	return []byte(content), nil
}

// renderRawContentMarkdown renders raw content for Markdown
func (m *markdownRenderer) renderRawContentMarkdown(raw *RawContent) ([]byte, error) {
	if raw.Format() == FormatMarkdown || raw.Format() == "md" {
		// Include raw markdown directly
		return raw.Data(), nil
	} else {
		// Escape other formats as code block
		escaped := m.escapeMarkdown(string(raw.Data()))
		return []byte(fmt.Sprintf("```\n%s\n```\n\n", escaped)), nil
	}
}

// renderSectionContentMarkdown renders section content with proper heading levels
func (m *markdownRenderer) renderSectionContentMarkdown(section *SectionContent) ([]byte, error) {
	return m.renderSectionContentMarkdownWithDepth(section, m.headingLevel)
}

// renderSectionContentMarkdownWithDepth renders section content with explicit depth tracking
func (m *markdownRenderer) renderSectionContentMarkdownWithDepth(section *SectionContent, depth int) ([]byte, error) {
	var result strings.Builder

	// Use depth as the heading level, clamped to valid range
	headingLevel := min(max(depth, 1), 6)

	headingPrefix := strings.Repeat("#", headingLevel)
	result.WriteString(fmt.Sprintf("%s %s\n\n", headingPrefix, m.escapeMarkdown(section.Title())))

	// Render nested content with increased depth for nested sections
	for _, content := range section.Contents() {
		var contentMD []byte
		var err error

		if nestedSection, ok := content.(*SectionContent); ok {
			// Increase depth for nested sections
			contentMD, err = m.renderSectionContentMarkdownWithDepth(nestedSection, depth+1)
		} else {
			contentMD, err = m.renderContent(content)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to render nested content: %w", err)
		}
		result.Write(contentMD)
	}

	return []byte(result.String()), nil
}

// escapeMarkdown escapes special markdown characters
func (m *markdownRenderer) escapeMarkdown(text string) string {
	// Escape markdown special characters
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		".", "\\.",
		"!", "\\!",
		"|", "\\|",
	)
	return replacer.Replace(text)
}

// escapeMarkdownTableCell escapes content for use in markdown table cells
func (m *markdownRenderer) escapeMarkdownTableCell(text string) string {
	// First escape all markdown special characters using the general escaper
	text = m.escapeMarkdown(text)
	
	// Then handle table-specific requirements
	text = strings.ReplaceAll(text, "\n", "<br>")
	text = strings.ReplaceAll(text, "\r", "")
	return text
}

// createMarkdownAnchor creates a markdown-compatible anchor from text
func (m *markdownRenderer) createMarkdownAnchor(text string) string {
	// Convert to lowercase and replace spaces/special chars with hyphens
	anchor := strings.ToLower(text)
	// Remove special characters and replace spaces with hyphens
	anchor = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(anchor, "")
	anchor = regexp.MustCompile(`\s+`).ReplaceAllString(anchor, "-")
	anchor = strings.Trim(anchor, "-")
	return anchor
}

// escapeYAMLValue escapes a value for use in YAML front matter
func (m *markdownRenderer) escapeYAMLValue(value string) string {
	// Simple YAML escaping - quote if contains special characters, but not dates
	// Don't quote simple dates in YYYY-MM-DD format
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, value); matched {
		return value
	}

	if strings.ContainsAny(value, ":{}[],&*#?|-<>=!%@`") || strings.Contains(value, " ") {
		// Escape quotes within the value
		value = strings.ReplaceAll(value, "\"", "\\\"")
		return fmt.Sprintf("\"%s\"", value)
	}
	return value
}

// NewMarkdownRendererWithToC creates a markdown renderer with table of contents enabled
func NewMarkdownRendererWithToC(enabled bool) *markdownRenderer {
	return &markdownRenderer{
		includeToC:   enabled,
		headingLevel: 1,
	}
}

// NewMarkdownRendererWithFrontMatter creates a markdown renderer with front matter
func NewMarkdownRendererWithFrontMatter(frontMatter map[string]string) *markdownRenderer {
	return &markdownRenderer{
		frontMatter:  frontMatter,
		headingLevel: 1,
	}
}

// NewMarkdownRendererWithOptions creates a markdown renderer with both ToC and front matter
func NewMarkdownRendererWithOptions(includeToC bool, frontMatter map[string]string) *markdownRenderer {
	return &markdownRenderer{
		includeToC:   includeToC,
		frontMatter:  frontMatter,
		headingLevel: 1,
	}
}

// formatCellValue processes field values and handles CollapsibleValue interface
func (m *markdownRenderer) formatCellValue(val any, field *Field) string {
	// Apply field formatter first using base renderer method
	processed := m.processFieldValue(val, field)

	// Check if result is CollapsibleValue (Requirement 3.1)
	if cv, ok := processed.(CollapsibleValue); ok {
		return m.renderCollapsibleValue(cv)
	}

	// Handle regular values (maintain backward compatibility)
	return fmt.Sprint(processed)
}

// renderCollapsibleValue renders a CollapsibleValue as HTML details element with error recovery
func (m *markdownRenderer) renderCollapsibleValue(cv CollapsibleValue) string {
	// Add panic recovery for error handling (Requirement 11.3)
	defer func() {
		if r := recover(); r != nil {
			// Log error and fall back to summary only
			fmt.Fprintf(os.Stderr, "Error rendering collapsible value in markdown: %v\n", r)
		}
	}()

	// Validate CollapsibleValue to prevent nil pointer issues
	if cv == nil {
		return "[invalid collapsible value]"
	}

	// Check global expansion override (Requirement 13.1)
	expanded := cv.IsExpanded() || m.collapsibleConfig.ForceExpansion

	openAttr := ""
	if expanded {
		openAttr = openAttribute // Requirement 3.2: add open attribute
	}

	// Get summary with error handling (Requirement 11.2)
	summary := m.getSafeSummary(cv)

	// Get and format details with error handling (Requirement 11.3)
	details := m.getSafeDetails(cv)

	// Use GitHub's native <details> support (Requirement 3.1)
	return fmt.Sprintf("<details%s><summary>%s</summary><br/>%s</details>",
		openAttr,
		m.escapeMarkdownTableCell(summary),
		m.escapeMarkdownTableCell(details))
}

// getSafeSummary gets the summary with error handling and fallbacks
func (m *markdownRenderer) getSafeSummary(cv CollapsibleValue) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error getting summary from collapsible value: %v\n", r)
		}
	}()

	summary := cv.Summary()
	if summary == "" {
		return defaultSummaryPlaceholder // Requirement 11.2: default placeholder
	}
	return summary
}

// getSafeDetails gets and formats details with comprehensive error handling
func (m *markdownRenderer) getSafeDetails(cv CollapsibleValue) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error getting details from collapsible value: %v\n", r)
		}
	}()

	details := cv.Details()
	if details == nil {
		// Requirement 11.1: treat nil details as non-collapsible, fall back to summary
		return m.getSafeSummary(cv)
	}

	// Format details with error recovery
	return m.formatDetailsForMarkdownSafe(details)
}

// formatDetailsForMarkdownSafe formats details content with comprehensive error handling
func (m *markdownRenderer) formatDetailsForMarkdownSafe(details any) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error formatting details for markdown: %v\n", r)
		}
	}()

	// Check for nested CollapsibleValue and prevent recursion (Requirement 11.5)
	if cv, ok := details.(CollapsibleValue); ok {
		// Treat nested CollapsibleValues as regular content to prevent infinite loops
		return fmt.Sprintf("[nested collapsible: %s]", cv.Summary())
	}

	switch d := details.(type) {
	case string:
		// Apply character limits if configured (Requirement 11.6)
		if m.collapsibleConfig.MaxDetailLength > 0 && len(d) > m.collapsibleConfig.MaxDetailLength {
			return d[:m.collapsibleConfig.MaxDetailLength] + m.collapsibleConfig.TruncateIndicator
		}
		return d
	case []string:
		if len(d) == 0 {
			return "[empty list]"
		}
		return strings.Join(d, "<br/>") // Requirement 3.4
	case map[string]any:
		if len(d) == 0 {
			return "[empty map]"
		}
		// Requirement 3.5: format as key-value pairs
		var parts []string
		for k, v := range d {
			// Handle potential nil values in map
			if v == nil {
				parts = append(parts, fmt.Sprintf("<strong>%s:</strong> [nil]", k))
			} else {
				parts = append(parts, fmt.Sprintf("<strong>%s:</strong> %v", k, v))
			}
		}
		result := strings.Join(parts, "<br/>")
		// Apply character limits if configured (Requirement 11.6)
		if m.collapsibleConfig.MaxDetailLength > 0 && len(result) > m.collapsibleConfig.MaxDetailLength {
			return result[:m.collapsibleConfig.MaxDetailLength] + m.collapsibleConfig.TruncateIndicator
		}
		return result
	case nil:
		return "[nil details]"
	default:
		// Fallback to string representation for unknown types (Requirement 11.3)
		result := fmt.Sprint(details)
		// Apply character limits if configured (Requirement 11.6)
		if m.collapsibleConfig.MaxDetailLength > 0 && len(result) > m.collapsibleConfig.MaxDetailLength {
			return result[:m.collapsibleConfig.MaxDetailLength] + m.collapsibleConfig.TruncateIndicator
		}
		return result
	}
}

// formatDetailsForMarkdown formats details content based on type (Requirements 3.4, 3.5)
// This method is kept for backward compatibility but now uses the safe version
func (m *markdownRenderer) formatDetailsForMarkdown(details any) string {
	return m.formatDetailsForMarkdownSafe(details)
}

// renderCollapsibleSection renders a CollapsibleSection as nested HTML details structure (Requirement 15.4)
func (m *markdownRenderer) renderCollapsibleSection(section *DefaultCollapsibleSection) ([]byte, error) {
	var result strings.Builder

	// Create nested details structure
	openAttr := ""
	if section.IsExpanded() || m.collapsibleConfig.ForceExpansion {
		openAttr = openAttribute
	}

	// Use nested details with section title (Requirement 15.4)
	result.WriteString(fmt.Sprintf("<details%s>\n", openAttr))
	result.WriteString(fmt.Sprintf("<summary>%s</summary>\n\n", m.escapeMarkdown(section.Title())))

	// Render all nested content within the collapsible section (Requirement 15.4)
	for i, content := range section.Content() {
		if i > 0 {
			result.WriteString("\n")
		}

		contentMD, err := m.renderContent(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render section content: %w", err)
		}

		result.Write(contentMD)
	}

	result.WriteString("\n</details>\n\n")

	return []byte(result.String()), nil
}
