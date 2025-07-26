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
				// Apply field formatter if available
				field := tableContent.Schema().FindField(key)
				if field != nil && field.Formatter != nil {
					row[i] = field.Formatter(val)
				} else {
					row[i] = val
				}
			} else {
				row[i] = ""
			}
		}
		tw.AppendRow(row)
	}

	return tw
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
