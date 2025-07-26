package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

// tableRenderer implements console table output format
type tableRenderer struct {
	styleName         string
	collapsibleConfig RendererConfig
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

		switch c := content.(type) {
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

// formatCellValue applies field formatter and handles CollapsibleValue rendering for table output
// This implements Requirements 6.1-6.7 for table renderer collapsible support
func (t *tableRenderer) formatCellValue(val any, field *Field) string {
	// Apply field formatter using base renderer functionality
	processed := t.processFieldValue(val, field)

	// Check if result is CollapsibleValue (Requirement 6.1)
	if cv, ok := processed.(CollapsibleValue); ok {
		// Check for global expansion override (Requirement 6.7, 13.1)
		expanded := cv.IsExpanded() || t.collapsibleConfig.ForceExpansion

		if expanded {
			// Show both summary and details (Requirement 6.2)
			details := t.formatDetailsForTable(cv.Details())
			return fmt.Sprintf("%s\n%s", cv.Summary(), details)
		}

		// Show summary with configurable indicator (Requirements 6.1, 6.6)
		indicator := t.collapsibleConfig.TableHiddenIndicator
		if indicator == "" {
			indicator = DefaultRendererConfig.TableHiddenIndicator
		}
		return fmt.Sprintf("%s %s", cv.Summary(), indicator)
	}

	// Handle regular values (maintain backward compatibility)
	return fmt.Sprint(processed)
}

// formatDetailsForTable formats details content with proper indentation for table display
// This implements Requirement 6.3 for appropriate spacing and readability
func (t *tableRenderer) formatDetailsForTable(details any) string {
	switch d := details.(type) {
	case string:
		return t.indentText(d)
	case []string:
		return t.indentText(strings.Join(d, "\n"))
	default:
		return t.indentText(fmt.Sprint(d))
	}
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
