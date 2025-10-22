package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

// tableRenderer implements console table output format
type tableRenderer struct {
	styleName         string
	collapsibleConfig RendererConfig
	maxColumnWidth    int // Maximum width for table columns (0 = no limit)
}

func (t *tableRenderer) Format() string {
	return FormatTable
}

func (t *tableRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return t.renderDocumentTable(ctx, doc)
}

func (t *tableRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	data, err := t.renderDocumentTable(ctx, doc)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (t *tableRenderer) SupportsStreaming() bool {
	return true
}

// StyleName returns the style name for testing purposes
func (t *tableRenderer) StyleName() string {
	return t.styleName
}

// renderDocumentTable renders entire document as formatted console tables
func (t *tableRenderer) renderDocumentTable(ctx context.Context, doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	var result bytes.Buffer
	contents := doc.GetContents()

	for i, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply per-content transformations before rendering
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}

		switch c := transformed.(type) {
		case *TableContent:
			if i > 0 {
				result.WriteString("\n")
			}

			tableWriter := t.renderTable(c)
			result.WriteString(tableWriter.Render())
			result.WriteString("\n")

		case *DefaultCollapsibleSection:
			if i > 0 {
				result.WriteString("\n")
			}

			sectionOutput, err := t.renderCollapsibleSection(c)
			if err != nil {
				return nil, fmt.Errorf("failed to render collapsible section: %w", err)
			}
			result.Write(sectionOutput)

		case *TextContent:
			if i > 0 {
				result.WriteString("\n")
			}

			style := c.Style()
			text := c.Text()

			if style.Header {
				// Create a simple header format
				result.WriteString(strings.ToUpper(text))
				result.WriteString("\n")
				result.WriteString(strings.Repeat("=", len(text)))
			} else {
				result.WriteString(text)
			}
			result.WriteString("\n")

		case *SectionContent:
			if i > 0 {
				result.WriteString("\n")
			}

			// Render section title with console-friendly formatting
			result.WriteString(fmt.Sprintf("=== %s ===\n\n", c.Title()))

			// Render section contents
			for j, subContent := range c.Contents() {
				if subTable, ok := subContent.(*TableContent); ok {
					if j > 0 {
						result.WriteString("\n")
					}
					tableWriter := t.renderTable(subTable)
					result.WriteString(tableWriter.Render())
					result.WriteString("\n")
				} else if subText, ok := subContent.(*TextContent); ok {
					if j > 0 {
						result.WriteString("\n")
					}
					result.WriteString(subText.Text())
					result.WriteString("\n")
				}
			}

		case *RawContent:
			if i > 0 {
				result.WriteString("\n")
			}
			result.WriteString(string(c.Data()))
			result.WriteString("\n")

		default:
			// Fallback for unknown content types
			if i > 0 {
				result.WriteString("\n")
			}
			contentBytes, err := content.AppendText(nil)
			if err != nil {
				return nil, fmt.Errorf("failed to render content %s: %w", content.ID(), err)
			}
			result.Write(contentBytes)
		}
	}

	return result.Bytes(), nil
}

// renderTable creates a formatted table from TableContent
func (t *tableRenderer) renderTable(tableContent *TableContent) table.Writer {
	tw := table.NewWriter()
	tw.SetStyle(t.getTableStyle())

	// Add title if present
	if tableContent.Title() != "" {
		tw.SetTitle(tableContent.Title())
	}

	// Get key order from schema
	keyOrder := tableContent.Schema().GetKeyOrder()
	if len(keyOrder) == 0 {
		return tw // Return empty table
	}

	// Configure column widths if maxColumnWidth is set
	if t.maxColumnWidth > 0 {
		columnConfigs := make([]table.ColumnConfig, len(keyOrder))
		for i := range keyOrder {
			columnConfigs[i] = table.ColumnConfig{
				Number:   i + 1, // Column numbers are 1-indexed
				WidthMax: t.maxColumnWidth,
			}
		}
		tw.SetColumnConfigs(columnConfigs)
	}

	// Set headers with proper order
	headerRow := make(table.Row, len(keyOrder))
	for i, key := range keyOrder {
		headerRow[i] = key
	}
	tw.AppendHeader(headerRow)

	// Add data rows preserving key order
	for _, record := range tableContent.Records() {
		row := make(table.Row, len(keyOrder))
		for i, key := range keyOrder {
			if val, exists := record[key]; exists {
				// Apply field formatter if available and format cell value
				field := tableContent.Schema().FindField(key)
				row[i] = t.formatCellValue(val, field)
			} else {
				row[i] = ""
			}
		}
		tw.AppendRow(row)
	}

	return tw
}

// formatCellValue applies field formatter and handles CollapsibleValue rendering for table output with error recovery
// This implements Requirements 6.1-6.7 for table renderer collapsible support
func (t *tableRenderer) formatCellValue(val any, field *Field) string {
	// Add panic recovery for error handling (Requirement 11.3)
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error formatting cell value in table renderer: %v\n", r)
		}
	}()

	// Apply field formatter using base renderer functionality
	processed := t.processFieldValue(val, field)

	// Check if result is CollapsibleValue (Requirement 6.1)
	if cv, ok := processed.(CollapsibleValue); ok {
		return t.renderCollapsibleValueSafe(cv)
	}

	// Handle array values by joining with newlines for better table readability
	switch v := processed.(type) {
	case []string:
		if len(v) == 0 {
			return ""
		}
		return strings.Join(v, "\n")
	case []any:
		if len(v) == 0 {
			return ""
		}
		strs := make([]string, len(v))
		for i, item := range v {
			strs[i] = fmt.Sprint(item)
		}
		return strings.Join(strs, "\n")
	}

	// Handle regular values (maintain backward compatibility)
	return fmt.Sprint(processed)
}

// renderCollapsibleValueSafe safely renders a CollapsibleValue with comprehensive error handling
func (t *tableRenderer) renderCollapsibleValueSafe(cv CollapsibleValue) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error rendering collapsible value in table: %v\n", r)
		}
	}()

	// Validate CollapsibleValue to prevent nil pointer issues
	if cv == nil {
		return "[invalid collapsible value]"
	}

	// Get summary with error handling (Requirement 11.2)
	summary := t.getSafeSummary(cv)

	// Check for global expansion override (Requirement 6.7, 13.1)
	expanded := cv.IsExpanded() || t.collapsibleConfig.ForceExpansion

	if expanded {
		// Show both summary and details (Requirement 6.2)
		details := t.getSafeDetails(cv)
		return fmt.Sprintf("%s\n%s", summary, details)
	}

	// Show summary with configurable indicator (Requirements 6.1, 6.6)
	indicator := t.collapsibleConfig.TableHiddenIndicator
	if indicator == "" {
		indicator = DefaultRendererConfig.TableHiddenIndicator
	}
	return fmt.Sprintf("%s %s", summary, indicator)
}

// getSafeSummary gets the summary with error handling and fallbacks for table renderer
func (t *tableRenderer) getSafeSummary(cv CollapsibleValue) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error getting summary from collapsible value in table: %v\n", r)
		}
	}()

	summary := cv.Summary()
	if summary == "" {
		return defaultSummaryPlaceholder // Requirement 11.2: default placeholder
	}
	return summary
}

// getSafeDetails gets and formats details with comprehensive error handling for table renderer
func (t *tableRenderer) getSafeDetails(cv CollapsibleValue) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error getting details from collapsible value in table: %v\n", r)
		}
	}()

	details := cv.Details()
	if details == nil {
		// Requirement 11.1: treat nil details as non-collapsible, fall back to summary
		return t.indentText(t.getSafeSummary(cv))
	}

	// Format details with error recovery
	return t.formatDetailsForTableSafe(details)
}

// formatDetailsForTableSafe formats details content with comprehensive error handling for table display
func (t *tableRenderer) formatDetailsForTableSafe(details any) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Error formatting details for table: %v\n", r)
		}
	}()

	// Check for nested CollapsibleValue and prevent recursion (Requirement 11.5)
	if cv, ok := details.(CollapsibleValue); ok {
		// Treat nested CollapsibleValues as regular content to prevent infinite loops
		return t.indentText(fmt.Sprintf("[nested collapsible: %s]", cv.Summary()))
	}

	switch d := details.(type) {
	case string:
		// Apply character limits if configured (Requirement 11.6)
		if t.collapsibleConfig.MaxDetailLength > 0 && len(d) > t.collapsibleConfig.MaxDetailLength {
			truncated := d[:t.collapsibleConfig.MaxDetailLength] + t.collapsibleConfig.TruncateIndicator
			return t.indentText(truncated)
		}
		return t.indentText(d)
	case []string:
		if len(d) == 0 {
			return t.indentText("[empty list]")
		}
		joined := strings.Join(d, "\n")
		// Apply character limits if configured (Requirement 11.6)
		if t.collapsibleConfig.MaxDetailLength > 0 && len(joined) > t.collapsibleConfig.MaxDetailLength {
			truncated := joined[:t.collapsibleConfig.MaxDetailLength] + t.collapsibleConfig.TruncateIndicator
			return t.indentText(truncated)
		}
		return t.indentText(joined)
	case map[string]any:
		if len(d) == 0 {
			return t.indentText("[empty map]")
		}
		var parts []string
		for k, v := range d {
			// Handle potential nil values in map
			if v == nil {
				parts = append(parts, fmt.Sprintf("%s: [nil]", k))
			} else {
				parts = append(parts, fmt.Sprintf("%s: %v", k, v))
			}
		}
		result := strings.Join(parts, "\n")
		// Apply character limits if configured (Requirement 11.6)
		if t.collapsibleConfig.MaxDetailLength > 0 && len(result) > t.collapsibleConfig.MaxDetailLength {
			truncated := result[:t.collapsibleConfig.MaxDetailLength] + t.collapsibleConfig.TruncateIndicator
			return t.indentText(truncated)
		}
		return t.indentText(result)
	case nil:
		return t.indentText("[nil details]")
	default:
		// Fallback to string representation for unknown types (Requirement 11.3)
		result := fmt.Sprint(details)
		// Apply character limits if configured (Requirement 11.6)
		if t.collapsibleConfig.MaxDetailLength > 0 && len(result) > t.collapsibleConfig.MaxDetailLength {
			truncated := result[:t.collapsibleConfig.MaxDetailLength] + t.collapsibleConfig.TruncateIndicator
			return t.indentText(truncated)
		}
		return t.indentText(result)
	}
}

// formatDetailsForTable formats details content with proper indentation for table display
// This implements Requirement 6.3 for appropriate spacing and readability
// This method is kept for backward compatibility but now uses the safe version
func (t *tableRenderer) formatDetailsForTable(details any) string {
	return t.formatDetailsForTableSafe(details)
}

// indentText adds proper indentation to text lines for table formatting
// This implements Requirement 6.3 for appropriate spacing
func (t *tableRenderer) indentText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "  " + line // Requirement 6.3: appropriate spacing
	}
	return strings.Join(lines, "\n")
}

// processFieldValue applies Field.Formatter and detects CollapsibleValue interface
// This is embedded from baseRenderer to maintain consistency
func (t *tableRenderer) processFieldValue(val any, field *Field) any {
	if field != nil && field.Formatter != nil {
		// Apply enhanced formatter (returns any, could be CollapsibleValue)
		return field.Formatter(val)
	}
	return val
}

// tableStyles is a lookup map for getting the table styles based on a string
var tableStyles = map[string]table.Style{
	"Default":                    table.StyleDefault,
	"Bold":                       table.StyleBold,
	"ColoredBright":              table.StyleColoredBright,
	"ColoredDark":                table.StyleColoredDark,
	"ColoredBlackOnBlueWhite":    table.StyleColoredBlackOnBlueWhite,
	"ColoredBlackOnCyanWhite":    table.StyleColoredBlackOnCyanWhite,
	"ColoredBlackOnGreenWhite":   table.StyleColoredBlackOnGreenWhite,
	"ColoredBlackOnMagentaWhite": table.StyleColoredBlackOnMagentaWhite,
	"ColoredBlackOnYellowWhite":  table.StyleColoredBlackOnYellowWhite,
	"ColoredBlackOnRedWhite":     table.StyleColoredBlackOnRedWhite,
	"ColoredBlueWhiteOnBlack":    table.StyleColoredBlueWhiteOnBlack,
	"ColoredCyanWhiteOnBlack":    table.StyleColoredCyanWhiteOnBlack,
	"ColoredGreenWhiteOnBlack":   table.StyleColoredGreenWhiteOnBlack,
	"ColoredMagentaWhiteOnBlack": table.StyleColoredMagentaWhiteOnBlack,
	"ColoredRedWhiteOnBlack":     table.StyleColoredRedWhiteOnBlack,
	"ColoredYellowWhiteOnBlack":  table.StyleColoredYellowWhiteOnBlack,
	"Double":                     table.StyleDouble,
	"Light":                      table.StyleLight,
	"Rounded":                    table.StyleRounded,
}

// getTableStyle returns the table style configuration
func (t *tableRenderer) getTableStyle() table.Style {
	if style, exists := tableStyles[t.styleName]; exists {
		return style
	}
	return table.StyleDefault
}

// NewTableRendererWithStyle creates a table renderer with specific style
func NewTableRendererWithStyle(styleName string) Renderer {
	return &tableRenderer{
		styleName: styleName,
	}
}

// NewTableRendererWithStyleAndWidth creates a table renderer with specific style and max column width
func NewTableRendererWithStyleAndWidth(styleName string, maxColumnWidth int) Renderer {
	return &tableRenderer{
		styleName:      styleName,
		maxColumnWidth: maxColumnWidth,
	}
}

// renderCollapsibleSection renders a CollapsibleSection for table output (Requirement 15.7)
func (t *tableRenderer) renderCollapsibleSection(section *DefaultCollapsibleSection) ([]byte, error) {
	var result strings.Builder

	// Show section title with expansion indicator (Requirement 15.7)
	expandIndicator := ""
	if !section.IsExpanded() && !t.collapsibleConfig.ForceExpansion {
		expandIndicator = " " + t.collapsibleConfig.TableHiddenIndicator
	}

	// Create section header (Requirement 15.7)
	result.WriteString(fmt.Sprintf("=== %s%s ===\n", section.Title(), expandIndicator))

	if section.IsExpanded() || t.collapsibleConfig.ForceExpansion {
		// Render nested content when expanded (Requirement 15.7)
		for i, content := range section.Content() {
			if i > 0 {
				result.WriteString("\n")
			}

			switch c := content.(type) {
			case *TableContent:
				tableWriter := t.renderTable(c)
				// Indent nested content (Requirement 15.7)
				lines := strings.Split(tableWriter.Render(), "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						result.WriteString("  " + line + "\n")
					}
				}
			case *TextContent:
				style := c.Style()
				text := c.Text()

				if style.Header {
					// Create a simple header format
					indentedText := "  " + strings.ToUpper(text)
					result.WriteString(indentedText + "\n")
					result.WriteString("  " + strings.Repeat("=", len(text)) + "\n")
				} else {
					result.WriteString("  " + text + "\n")
				}
			default:
				// Fallback for other content types
				contentBytes, err := content.AppendText(nil)
				if err != nil {
					return nil, fmt.Errorf("failed to render nested content: %w", err)
				}
				// Indent the content
				lines := strings.Split(string(contentBytes), "\n")
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						result.WriteString("  " + line + "\n")
					}
				}
			}
		}
	} else {
		// Show collapsed indicator (Requirement 15.7)
		result.WriteString(fmt.Sprintf("  [Section collapsed - contains %d item(s)]\n",
			len(section.Content())))
	}

	result.WriteString("\n")
	return []byte(result.String()), nil
}
