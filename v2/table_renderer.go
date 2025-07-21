package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// tableRenderer implements console table output format
type tableRenderer struct {
	styleName string
}

func (t *tableRenderer) Format() string {
	return FormatTable
}

func (tr *tableRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return tr.renderDocumentTable(ctx, doc)
}

func (tr *tableRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	data, err := tr.renderDocumentTable(ctx, doc)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (t *tableRenderer) SupportsStreaming() bool {
	return true
}

// renderDocumentTable renders entire document as formatted console tables
func (tr *tableRenderer) renderDocumentTable(ctx context.Context, doc *Document) ([]byte, error) {
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

			tableWriter := tr.renderTable(c)
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
					tableWriter := tr.renderTable(subTable)
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
func (tr *tableRenderer) renderTable(tableContent *TableContent) table.Writer {
	t := table.NewWriter()
	t.SetStyle(tr.getTableStyle())

	// Add title if present
	if tableContent.Title() != "" {
		t.SetTitle(tableContent.Title())
	}

	// Get key order from schema
	keyOrder := tableContent.Schema().GetKeyOrder()
	if len(keyOrder) == 0 {
		return t // Return empty table
	}

	// Set headers with proper order
	headerRow := make(table.Row, len(keyOrder))
	for i, key := range keyOrder {
		headerRow[i] = key
	}
	t.AppendHeader(headerRow)

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
		t.AppendRow(row)
	}

	return t
}

// getTableStyle returns the table style configuration
func (tr *tableRenderer) getTableStyle() table.Style {
	switch tr.styleName {
	case "ColoredBright":
		return table.Style{
			Name: "ColoredBright",
			Box: table.BoxStyle{
				BottomLeft:       "└",
				BottomRight:      "┘",
				BottomSeparator:  "┴",
				Left:             "│",
				LeftSeparator:    "├",
				MiddleHorizontal: "─",
				MiddleSeparator:  "┼",
				MiddleVertical:   "│",
				PaddingLeft:      " ",
				PaddingRight:     " ",
				Right:            "│",
				RightSeparator:   "┤",
				TopLeft:          "┌",
				TopRight:         "┐",
				TopSeparator:     "┬",
				UnfinishedRow:    " ~~",
			},
			Color: table.ColorOptions{
				Footer:       text.Colors{text.BgBlack, text.FgRed},
				Header:       text.Colors{text.BgHiBlack, text.FgHiWhite, text.Bold},
				Row:          text.Colors{text.BgHiBlack, text.FgWhite},
				RowAlternate: text.Colors{text.BgBlack, text.FgHiWhite},
			},
			Format: table.FormatOptions{
				Footer: text.FormatDefault,
				Header: text.FormatDefault,
				Row:    text.FormatDefault,
			},
			Options: table.Options{
				DrawBorder:      true,
				SeparateColumns: true,
				SeparateFooter:  true,
				SeparateHeader:  true,
				SeparateRows:    false,
			},
		}

	case "ColoredDark":
		return table.Style{
			Name: "ColoredDark",
			Box: table.BoxStyle{
				BottomLeft:       "└",
				BottomRight:      "┘",
				BottomSeparator:  "┴",
				Left:             "│",
				LeftSeparator:    "├",
				MiddleHorizontal: "─",
				MiddleSeparator:  "┼",
				MiddleVertical:   "│",
				PaddingLeft:      " ",
				PaddingRight:     " ",
				Right:            "│",
				RightSeparator:   "┤",
				TopLeft:          "┌",
				TopRight:         "┐",
				TopSeparator:     "┬",
				UnfinishedRow:    " ~~",
			},
			Color: table.ColorOptions{
				Footer:       text.Colors{text.BgHiBlack, text.FgHiRed},
				Header:       text.Colors{text.BgHiRed, text.FgHiWhite, text.Bold},
				Row:          text.Colors{text.BgBlack, text.FgHiWhite},
				RowAlternate: text.Colors{text.BgHiBlack, text.FgWhite},
			},
			Format: table.FormatOptions{
				Footer: text.FormatDefault,
				Header: text.FormatDefault,
				Row:    text.FormatDefault,
			},
			Options: table.Options{
				DrawBorder:      true,
				SeparateColumns: true,
				SeparateFooter:  true,
				SeparateHeader:  true,
				SeparateRows:    false,
			},
		}

	case "Light":
		return table.StyleLight

	case "Bold":
		return table.StyleBold

	case "Double":
		return table.StyleDouble

	case "Rounded":
		return table.StyleRounded

	default:
		return table.StyleDefault
	}
}

// NewTableRendererWithStyle creates a table renderer with specific style
func NewTableRendererWithStyle(styleName string) *tableRenderer {
	return &tableRenderer{
		styleName: styleName,
	}
}
