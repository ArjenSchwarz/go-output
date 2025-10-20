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
			level := max(c.Level(), 1)
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
			level := max(parentLevel+1, 1)
			indent := strings.Repeat("  ", level-1)
			anchor := m.createMarkdownAnchor(c.Title())
			fmt.Fprintf(toc, "%s- [%s](#%s)\n", indent, m.escapeMarkdown(c.Title()), anchor)
			m.addSubsectionsToToC(toc, c.Contents(), level)
		case *TextContent:
			if c.Style().Header {
				level := max(parentLevel+1, 1)
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
	case *ChartContent:
		return m.renderChartContentMarkdown(c)
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
			// Skip escaping if this is a collapsible value (starts with <details)
			if !strings.HasPrefix(cellValue, "<details") {
				cellValue = m.escapeMarkdownTableCell(cellValue)
			} else {
				// For collapsible values, only replace newlines with <br>
				cellValue = strings.ReplaceAll(cellValue, "\n", "<br>")
			}
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
		return fmt.Appendf(nil, "```\n%s\n```\n\n", escaped), nil
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

// renderChartContentMarkdown renders chart content as Markdown with mermaid code fence
func (m *markdownRenderer) renderChartContentMarkdown(chart *ChartContent) ([]byte, error) {
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

	// Wrap in ```mermaid code fence
	var result strings.Builder
	result.WriteString("```mermaid\n")
	result.Write(mermaidData)
	// Ensure there's no trailing newline before closing fence
	mermaidStr := strings.TrimRight(string(mermaidData), "\n")
	result.Reset()
	result.WriteString("```mermaid\n")
	result.WriteString(mermaidStr)
	result.WriteString("\n```\n")

	return []byte(result.String()), nil
}

// escapeMarkdown escapes special markdown characters for general markdown content
func (m *markdownRenderer) escapeMarkdown(text string) string {
	// Only escape characters that would actually be interpreted as markdown
	// in the specific context where this is used
	replacer := strings.NewReplacer(
		"\\", "\\\\", // Backslash must be escaped
		"`", "\\`", // Backticks for code
		"*", "\\*", // Asterisks for emphasis
		"_", "\\_", // Underscores for emphasis
		"[", "\\[", // Square brackets for links
		"]", "\\]", // Square brackets for links
		"#", "\\#", // Hash for headers (only at start of line in practice)
		"|", "\\|", // Pipe for tables
	)
	return replacer.Replace(text)
}

// escapeMarkdownTableCell escapes content for use in markdown table cells
func (m *markdownRenderer) escapeMarkdownTableCell(text string) string {
	// In table cells, markdown formatting is still interpreted, so we need to escape:
	// - Pipes (|) which break table structure
	// - Asterisks (*) which create emphasis/bold formatting
	// - Underscores (_) which create emphasis/italic formatting
	// - Backticks (`) which create code formatting
	// - Square brackets ([]) which create links
	text = strings.ReplaceAll(text, "|", "\\|")
	text = strings.ReplaceAll(text, "*", "\\*")
	text = strings.ReplaceAll(text, "_", "\\_")
	text = strings.ReplaceAll(text, "`", "\\`")
	text = strings.ReplaceAll(text, "[", "\\[")
	text = strings.ReplaceAll(text, "]", "\\]")

	// Replace newlines with <br> for table cell compatibility
	text = strings.ReplaceAll(text, "\n", "<br>")

	text = strings.ReplaceAll(text, "\r", "")
	return text
}

// preserveCodeFenceNewlines replaces newlines with <br> in table cells
// Since this is called from escapeMarkdownTableCell, we know we're in a table context
// and need to use <br> tags to avoid breaking the table structure
func (m *markdownRenderer) preserveCodeFenceNewlines(text string) string {
	// In table cells, we must use <br> tags to preserve table structure
	// The code fence will still provide syntax highlighting, just with <br> instead of newlines
	var result strings.Builder
	inCodeFence := false
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		// Check if this line starts or ends a code fence
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "```") {
			inCodeFence = !inCodeFence
		}

		result.WriteString(line)

		// Don't add line separator after the last line
		if i < len(lines)-1 {
			// Always use <br> in table context to maintain table structure
			// The code fence syntax highlighting will still work with <br> tags
			result.WriteString("<br>")
		}
	}

	return result.String()
}

// escapeHTMLContent escapes markdown characters that would be interpreted inside HTML tags
// This is needed for content inside <details>, <summary>, etc where GitHub still processes markdown
func (m *markdownRenderer) escapeHTMLContent(text string) string {
	// Only escape the most critical markdown characters that cause issues in HTML content
	replacer := strings.NewReplacer(
		"*", "\\*", // Asterisks for emphasis
		"_", "\\_", // Underscores for emphasis
		"`", "\\`", // Backticks for code
		"[", "\\[", // Square brackets for links
		"]", "\\]", // Square brackets for links
	)
	return replacer.Replace(text)
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

	// Handle array values by joining with <br/> for markdown table cells
	switch v := processed.(type) {
	case []string:
		if len(v) == 0 {
			return ""
		}
		// Escape each item and join with <br/> for proper rendering in markdown tables
		escaped := make([]string, len(v))
		for i, item := range v {
			escaped[i] = m.escapeMarkdown(item)
		}
		return strings.Join(escaped, "<br/>")
	case []any:
		if len(v) == 0 {
			return ""
		}
		// Convert to strings, escape, and join
		strs := make([]string, len(v))
		for i, item := range v {
			strs[i] = m.escapeMarkdown(fmt.Sprint(item))
		}
		return strings.Join(strs, "<br/>")
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

	// Check if the value implements code fence configuration
	var useCodeFences bool
	var codeLanguage string
	if dcv, ok := cv.(*DefaultCollapsibleValue); ok {
		useCodeFences = dcv.UseCodeFences()
		codeLanguage = dcv.CodeLanguage()
	}

	// Get details with potential code fence wrapping
	var details string
	if useCodeFences {
		details = m.getSafeDetailsWithCodeFences(cv, codeLanguage)
	} else {
		// Get and format details with error handling (Requirement 11.3)
		details = m.getSafeDetails(cv)
	}

	// Use GitHub's native <details> support (Requirement 3.1)
	// Only escape if not one of our default placeholders
	if summary != defaultSummaryPlaceholder {
		summary = m.escapeHTMLContent(summary)
	}
	// Details have already been processed by formatDetailsForMarkdownSafe
	return fmt.Sprintf("<details%s><summary>%s</summary><br/>%s</details>",
		openAttr,
		summary,
		details)
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
		// Don't escape the square brackets here - they're part of our error message
		return fmt.Sprintf("[nested collapsible: %s]", m.escapeHTMLContent(cv.Summary()))
	}

	switch d := details.(type) {
	case string:
		// Apply character limits if configured (Requirement 11.6)
		if m.collapsibleConfig.MaxDetailLength > 0 && len(d) > m.collapsibleConfig.MaxDetailLength {
			d = d[:m.collapsibleConfig.MaxDetailLength] + m.collapsibleConfig.TruncateIndicator
		}
		// Replace newlines with <br> for proper HTML rendering
		d = strings.ReplaceAll(d, "\n", "<br>")
		d = strings.ReplaceAll(d, "\r", "")
		return d
	case []string:
		if len(d) == 0 {
			return "[empty list]"
		}
		// Escape each string item before joining
		escaped := make([]string, len(d))
		for i, item := range d {
			escaped[i] = m.escapeHTMLContent(item)
		}
		return strings.Join(escaped, "<br/>") // Requirement 3.4
	case map[string]any:
		if len(d) == 0 {
			return "[empty map]"
		}
		// Requirement 3.5: format as key-value pairs
		var parts []string
		for k, v := range d {
			// Escape the key
			escapedKey := m.escapeHTMLContent(k)
			// Handle potential nil values in map
			if v == nil {
				parts = append(parts, fmt.Sprintf("<strong>%s:</strong> [nil]", escapedKey))
			} else {
				escapedValue := m.escapeHTMLContent(fmt.Sprint(v))
				parts = append(parts, fmt.Sprintf("<strong>%s:</strong> %s", escapedKey, escapedValue))
			}
		}
		result := strings.Join(parts, "<br/>")
		// Apply character limits if configured (Requirement 11.6)
		if m.collapsibleConfig.MaxDetailLength > 0 && len(result) > m.collapsibleConfig.MaxDetailLength {
			return result[:m.collapsibleConfig.MaxDetailLength] + m.collapsibleConfig.TruncateIndicator
		}
		return result
	case nil:
		// Don't escape square brackets - they're part of our error message
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

// getSafeDetailsWithCodeFences gets and formats details wrapped in code fences with error handling
func (m *markdownRenderer) getSafeDetailsWithCodeFences(cv CollapsibleValue, language string) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error getting details with code fences from collapsible value: %v\n", r)
		}
	}()

	details := cv.Details()
	if details == nil {
		// Requirement 11.1: treat nil details as non-collapsible, fall back to summary
		return m.getSafeSummary(cv)
	}

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

	// Apply character limits if configured
	if m.collapsibleConfig.MaxDetailLength > 0 && len(content) > m.collapsibleConfig.MaxDetailLength {
		content = content[:m.collapsibleConfig.MaxDetailLength] + m.collapsibleConfig.TruncateIndicator
	}

	// Wrap in markdown code fences
	// Note: We don't add leading/trailing newlines here because
	// they would be converted to <br/> tags in table cells
	if language != "" {
		return fmt.Sprintf("```%s\n%s\n```", language, content)
	}
	return fmt.Sprintf("```\n%s\n```", content)
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

	// Render all nested content within the collapsible section with indentation (Requirement 15.4)
	for i, content := range section.Content() {
		if i > 0 {
			result.WriteString("\n")
		}

		contentMD, err := m.renderContent(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render section content: %w", err)
		}

		// Indent the content for better visual hierarchy in markdown
		lines := strings.SplitSeq(string(contentMD), "\n")
		for line := range lines {
			if strings.TrimSpace(line) != "" {
				result.WriteString("  ") // Add 2 spaces for indentation
				result.WriteString(line)
				result.WriteString("\n")
			} else if line == "" {
				result.WriteString("\n") // Preserve empty lines
			}
		}
	}

	result.WriteString("\n</details>\n\n")

	return []byte(result.String()), nil
}
