package output

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// csvRenderer implements CSV output format
type csvRenderer struct {
	// No base renderer needed for CSV
}

func (c *csvRenderer) Format() string {
	return "csv"
}

func (c *csvRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return c.renderDocumentCSV(ctx, doc)
}

func (c *csvRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	return c.renderDocumentCSVTo(ctx, doc, w)
}

func (c *csvRenderer) SupportsStreaming() bool {
	return true
}

// renderDocumentCSV renders entire document as CSV with proper key order preservation
func (cr *csvRenderer) renderDocumentCSV(ctx context.Context, doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	var buf bytes.Buffer
	err := cr.renderDocumentCSVTo(ctx, doc, &buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderDocumentCSVTo streams CSV output with proper key order preservation
func (cr *csvRenderer) renderDocumentCSVTo(ctx context.Context, doc *Document, w io.Writer) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}

	contents := doc.GetContents()
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Track if we've written any headers to avoid multiple header rows
	hasWrittenHeaders := false

	for i, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Only handle table content for CSV output
		if table, ok := content.(*TableContent); ok {
			// Add a blank line between tables (except for the first table)
			if i > 0 && hasWrittenHeaders {
				if err := csvWriter.Write([]string{}); err != nil {
					return fmt.Errorf("failed to write separator row: %w", err)
				}
			}

			// Skip table title for CSV as it breaks parsing
			// CSV format doesn't support comments in a standard way

			// Write headers for first table or when headers differ
			writeHeaders := !hasWrittenHeaders
			if err := cr.renderTableContentCSV(table, csvWriter, writeHeaders); err != nil {
				return fmt.Errorf("failed to render table %s: %w", content.ID(), err)
			}

			hasWrittenHeaders = true
		} else if !hasWrittenHeaders {
			// For non-table content, write as a single-column CSV row
			// Only if no tables have been written yet (to avoid mixing formats)
			contentText, err := content.AppendText(nil)
			if err == nil && len(contentText) > 0 {
				// Write simple header and content as CSV
				if err := csvWriter.Write([]string{"content"}); err != nil {
					return fmt.Errorf("failed to write content header: %w", err)
				}
				if err := csvWriter.Write([]string{cr.formatValueForCSV(string(contentText))}); err != nil {
					return fmt.Errorf("failed to write content row: %w", err)
				}
				hasWrittenHeaders = true
			}
		}
	}

	return nil
}

// renderTableContentCSV renders table content to CSV with key order preservation
func (cr *csvRenderer) renderTableContentCSV(table *TableContent, csvWriter *csv.Writer, writeHeaders bool) error {
	keyOrder := table.Schema().GetKeyOrder()
	if len(keyOrder) == 0 {
		return nil // No columns to write
	}

	// Write headers if requested
	if writeHeaders {
		if err := csvWriter.Write(keyOrder); err != nil {
			return fmt.Errorf("failed to write CSV headers: %w", err)
		}
	}

	// Write data rows in key order
	for _, record := range table.Records() {
		row := make([]string, len(keyOrder))
		for i, key := range keyOrder {
			if val, exists := record[key]; exists {
				row[i] = cr.formatValueForCSV(val)
			}
			// Empty string for missing values (row[i] is already "")
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// formatValueForCSV converts any value to its CSV string representation
func (cr *csvRenderer) formatValueForCSV(val any) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		// Handle newlines and tabs in strings
		str := strings.ReplaceAll(v, "\n", " ")
		str = strings.ReplaceAll(str, "\r", " ")
		str = strings.ReplaceAll(str, "\t", " ")
		return str
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		// Format float without unnecessary decimal places
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		if v == float32(int32(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case time.Time:
		return v.Format(time.RFC3339)
	case []byte:
		return string(v)
	default:
		// For complex types, convert to string
		str := fmt.Sprintf("%v", v)
		// Handle potential newlines and tabs in data by replacing them with spaces
		str = strings.ReplaceAll(str, "\n", " ")
		str = strings.ReplaceAll(str, "\r", " ")
		str = strings.ReplaceAll(str, "\t", " ")
		return str
	}
}
