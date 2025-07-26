package output

import (
	"context"
	"fmt"
	"io"
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
				if field != nil && field.Formatter != nil {
					formatted := field.Formatter(val)
					cellValue = fmt.Sprint(formatted)
				} else {
					cellValue = fmt.Sprint(val)
				}
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
	// For table cells, we need to escape pipes and newlines specially
	text = strings.ReplaceAll(text, "|", "\\|")
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
