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
	collapsibleConfig RendererConfig
}

func (c *csvRenderer) Format() string {
	return FormatCSV
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
func (c *csvRenderer) renderDocumentCSV(ctx context.Context, doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	var buf bytes.Buffer
	err := c.renderDocumentCSVTo(ctx, doc, &buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderDocumentCSVTo streams CSV output with proper key order preservation
func (c *csvRenderer) renderDocumentCSVTo(ctx context.Context, doc *Document, w io.Writer) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}

	contents := doc.GetContents()
	csvWriter := csv.NewWriter(w)

	// flushCSV flushes any buffered rows and reports a write error surfaced by
	// the underlying io.Writer. csv.Writer.Write buffers data, so an underlying
	// failure may only be reported here via csvWriter.Error() (T-1186).
	flushCSV := func() error {
		csvWriter.Flush()
		return csvWriter.Error()
	}

	// Track the last written header schema to detect schema changes
	var lastKeyOrder []string

	for i, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			// Flush already-buffered rows; prefer a write error over the
			// context error so a failed destination is not masked.
			if flushErr := flushCSV(); flushErr != nil {
				return flushErr
			}
			return ctx.Err()
		default:
		}

		// Apply per-content transformations before rendering
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			// Flush already-buffered rows; prefer a write error over the
			// transformation error so a failed destination is not masked.
			if flushErr := flushCSV(); flushErr != nil {
				return flushErr
			}
			return err
		}

		// Handle different content types for CSV output
		switch content := transformed.(type) {
		case *TableContent:
			// Add a blank line between tables (except for the first table)
			if i > 0 && lastKeyOrder != nil {
				if err := csvWriter.Write([]string{}); err != nil {
					return fmt.Errorf("failed to write separator row: %w", err)
				}
			}

			// Skip table title for CSV as it breaks parsing
			// CSV format doesn't support comments in a standard way

			// Write headers for first table or when schema differs from previous table
			writeHeaders := !keyOrdersEqual(lastKeyOrder, content.getSchema().GetKeyOrder())
			if err := c.renderTableContentCSV(content, csvWriter, writeHeaders); err != nil {
				return fmt.Errorf("failed to render table %s: %w", content.ID(), err)
			}

			lastKeyOrder = content.getSchema().GetKeyOrder()

		case *SectionContent:
			// Extract and render tables from sections. CSV is a flat format,
			// so we recursively flatten the section hierarchy to any depth
			// (T-1239). renderSectionTablesCSV applies per-content
			// transformations at every level and renders each nested table.
			if err := c.renderSectionTablesCSV(ctx, content, csvWriter, &lastKeyOrder, flushCSV); err != nil {
				return err
			}

		case *DefaultCollapsibleSection:
			// Handle CollapsibleSection with metadata comments (Requirement 15.8).
			// lastKeyOrder is shared so header re-writes and separators stay
			// consistent with surrounding top-level tables, and so tables inside
			// the section get their own headers whenever the schema changes
			// (T-1315).
			if err := c.renderCollapsibleSectionCSV(content, csvWriter, &lastKeyOrder); err != nil {
				return fmt.Errorf("failed to render collapsible section %s: %w", content.ID(), err)
			}

		default:
			if lastKeyOrder == nil {
				// For non-table content, write as a single-column CSV row
				// Only if no tables have been written yet (to avoid mixing formats)
				contentText, err := content.AppendText(nil)
				if err == nil && len(contentText) > 0 {
					// Write simple header and content as CSV
					if err := csvWriter.Write([]string{keyContent}); err != nil {
						return fmt.Errorf("failed to write content header: %w", err)
					}
					if err := csvWriter.Write([]string{c.formatValueForCSV(string(contentText))}); err != nil {
						return fmt.Errorf("failed to write content row: %w", err)
					}
					lastKeyOrder = []string{keyContent}
				}
			}
		}
	}

	// Flush buffered rows and report any error from the underlying writer
	// that was deferred until flush time (T-1186).
	return flushCSV()
}

// renderSectionTablesCSV recursively flattens a section hierarchy and renders
// every nested table to CSV (T-1239). It walks the section's contents,
// applying per-content transformations at each level, rendering any
// TableContent it finds, and recursing into any nested SectionContent. This
// replaces an earlier hand-written loop that only reached tables one section
// level deep, silently dropping tables nested deeper.
//
// lastKeyOrder is shared across the whole document so separators and header
// re-writes behave consistently with top-level tables. flushCSV is the
// document-level flush used to surface deferred writer errors before returning
// a transformation error (T-1186).
func (c *csvRenderer) renderSectionTablesCSV(ctx context.Context, section *SectionContent, csvWriter *csv.Writer, lastKeyOrder *[]string, flushCSV func() error) error {
	for _, nestedContent := range section.Contents() {
		// Apply transformations to nested content at this level.
		transformed, err := applyContentTransformations(ctx, nestedContent)
		if err != nil {
			// Flush already-buffered rows; prefer a write error over the
			// transformation error so a failed destination is not masked.
			if flushErr := flushCSV(); flushErr != nil {
				return flushErr
			}
			return err
		}

		switch nested := transformed.(type) {
		case *TableContent:
			// Add separator between tables (matches top-level behaviour).
			if *lastKeyOrder != nil {
				if err := csvWriter.Write([]string{}); err != nil {
					return fmt.Errorf("failed to write separator row: %w", err)
				}
			}

			writeHeaders := !keyOrdersEqual(*lastKeyOrder, nested.getSchema().GetKeyOrder())
			if err := c.renderTableContentCSV(nested, csvWriter, writeHeaders); err != nil {
				return fmt.Errorf("failed to render table %s: %w", nested.ID(), err)
			}
			*lastKeyOrder = nested.getSchema().GetKeyOrder()

		case *SectionContent:
			// Recurse into deeper sections to any depth.
			if err := c.renderSectionTablesCSV(ctx, nested, csvWriter, lastKeyOrder, flushCSV); err != nil {
				return err
			}
		}
	}

	return nil
}

// renderTableContentCSV renders table content to CSV with key order preservation
func (c *csvRenderer) renderTableContentCSV(table *TableContent, csvWriter *csv.Writer, writeHeaders bool) error {
	// Handle collapsible fields by creating extended schema and records (Requirement 8.1)
	enhancedTable, err := c.handleCollapsibleFields(table)
	if err != nil {
		return fmt.Errorf("failed to process collapsible fields: %w", err)
	}

	keyOrder := enhancedTable.getSchema().GetKeyOrder()
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
	for _, record := range enhancedTable.Records() {
		row := make([]string, len(keyOrder))
		for i, key := range keyOrder {
			if val, exists := record[key]; exists {
				row[i] = c.formatValueForCSV(val)
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
func (c *csvRenderer) formatValueForCSV(val any) string {
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

// handleCollapsibleFields analyzes table schema and creates additional "_details" columns
// for fields that produce CollapsibleValue content (Requirement 8.1)
func (c *csvRenderer) handleCollapsibleFields(table *TableContent) (*TableContent, error) {
	originalFields := table.getSchema().Fields
	originalKeyOrder := table.getSchema().GetKeyOrder()
	originalRecords := table.Records()

	// Analyze which fields contain CollapsibleValue content
	collapsibleFields := c.detectCollapsibleFields(table)
	if len(collapsibleFields) == 0 {
		// No collapsible content, return original table
		return table, nil
	}

	// Create new fields and key order with detail columns (Requirement 8.4)
	newFields := []Field{}
	newKeyOrder := []string{}

	for _, field := range originalFields {
		// Add original field
		newFields = append(newFields, field)
		newKeyOrder = append(newKeyOrder, field.Name)

		// Check if this field produces collapsible content
		if collapsibleFields[field.Name] {
			// Add detail column adjacent to source column (Requirement 8.4)
			detailField := Field{
				Name: field.Name + "_details",
				Type: "string",
				// No formatter for detail columns
			}
			newFields = append(newFields, detailField)
			newKeyOrder = append(newKeyOrder, detailField.Name)
		}
	}

	// Create new schema with enhanced fields
	newSchema := &Schema{
		Fields:   newFields,
		keyOrder: newKeyOrder,
	}

	// Transform records to include detail columns
	newRecords := make([]Record, len(originalRecords))
	for i, record := range originalRecords {
		newRecord := make(Record)

		// Process each original field
		for _, key := range originalKeyOrder {
			val := record[key]
			field := table.getSchema().FindField(key)

			// Apply field formatter if present
			if field != nil && field.Formatter != nil {
				processed := field.Formatter(val)

				// Check if result is CollapsibleValue (Requirement 8.2)
				if cv, ok := processed.(CollapsibleValue); ok {
					newRecord[key] = cv.Summary()                              // Summary in original column
					newRecord[key+"_details"] = c.flattenDetails(cv.Details()) // Details in new column
				} else {
					newRecord[key] = processed
					// Leave detail column empty for non-collapsible (Requirement 8.3)
					if collapsibleFields[key] {
						newRecord[key+"_details"] = ""
					}
				}
			} else {
				newRecord[key] = val
				// Leave detail column empty for non-collapsible (Requirement 8.3)
				if collapsibleFields[key] {
					newRecord[key+"_details"] = ""
				}
			}
		}

		newRecords[i] = newRecord
	}

	// Create new table with enhanced schema and records
	enhancedTable := &TableContent{
		id:      table.ID(),
		title:   table.Title(),
		schema:  newSchema,
		records: newRecords,
	}

	return enhancedTable, nil
}

// detectCollapsibleFields analyzes table data to identify fields that may produce CollapsibleValue content
func (c *csvRenderer) detectCollapsibleFields(table *TableContent) map[string]bool {
	collapsibleFields := make(map[string]bool)
	records := table.Records()

	if len(records) == 0 {
		return collapsibleFields
	}

	// Check each field by applying its formatter to sample data
	for _, field := range table.getSchema().Fields {
		if field.Formatter == nil {
			continue
		}

		// Test formatter with first non-nil value from records
		for _, record := range records {
			if val, exists := record[field.Name]; exists && val != nil {
				processed := field.Formatter(val)
				if _, ok := processed.(CollapsibleValue); ok {
					collapsibleFields[field.Name] = true
					break
				}
			}
		}
	}

	return collapsibleFields
}

// flattenDetails converts complex detail structures to appropriate string representations for CSV (Requirement 8.5)
func (c *csvRenderer) flattenDetails(details any) string {
	if details == nil {
		return ""
	}

	switch d := details.(type) {
	case string:
		// Handle newlines and tabs for CSV compatibility
		str := strings.ReplaceAll(d, "\n", " ")
		str = strings.ReplaceAll(str, "\r", " ")
		str = strings.ReplaceAll(str, "\t", " ")
		return str
	case []string:
		// Join string arrays with semicolon separator
		return strings.Join(d, "; ")
	case map[string]any:
		// Convert maps to key-value pairs
		var parts []string
		for k, v := range d {
			parts = append(parts, fmt.Sprintf("%s: %v", k, v))
		}
		return strings.Join(parts, "; ")
	case []any:
		// Convert generic arrays to string
		var parts []string
		for _, item := range d {
			parts = append(parts, fmt.Sprintf("%v", item))
		}
		return strings.Join(parts, "; ")
	default:
		// For other complex types, convert to string and clean for CSV
		str := fmt.Sprintf("%v", details)
		str = strings.ReplaceAll(str, "\n", " ")
		str = strings.ReplaceAll(str, "\r", " ")
		str = strings.ReplaceAll(str, "\t", " ")
		return str
	}
}

// renderCollapsibleSectionCSV renders a CollapsibleSection with metadata comments (Requirement 15.8).
//
// lastKeyOrder is shared with the document-level renderer so that headers are
// re-written and separators inserted whenever a table's key order changes,
// mirroring the normal CSV path and renderSectionTablesCSV (T-1315). Tracking a
// single boolean previously suppressed headers for every table after the first,
// producing malformed CSV when a section held tables with differing schemas.
func (c *csvRenderer) renderCollapsibleSectionCSV(section *DefaultCollapsibleSection, csvWriter *csv.Writer, lastKeyOrder *[]string) error {
	// Add section metadata as CSV comments or special rows (Requirement 15.8)

	// Since CSV doesn't support comments officially, we'll use a special metadata row format
	// This creates a recognizable pattern that can be parsed later if needed
	metadataRow := []string{
		fmt.Sprintf("# Section: %s", section.Title()),
		fmt.Sprintf("Level: %d", section.Level()),
		fmt.Sprintf("Expanded: %t", section.IsExpanded()),
		"Type: collapsible_section",
	}

	if err := csvWriter.Write(metadataRow); err != nil {
		return fmt.Errorf("failed to write section metadata: %w", err)
	}

	// Process each content item in the section (Requirement 15.8)
	for i, content := range section.Content() {
		switch contentItem := content.(type) {
		case *TableContent:
			// Add a blank separator row between tables (matches top-level
			// behaviour) when a previous table has already been written.
			if *lastKeyOrder != nil {
				if err := csvWriter.Write([]string{}); err != nil {
					return fmt.Errorf("failed to write separator row: %w", err)
				}
			}

			// For table content, add section context to CSV (Requirement 15.8)
			if contentItem.Title() != "" {
				// Add table title with section context
				titleRow := []string{fmt.Sprintf("# %s - %s", section.Title(), contentItem.Title())}
				if err := csvWriter.Write(titleRow); err != nil {
					return fmt.Errorf("failed to write table title: %w", err)
				}
			}

			// Write headers for the first table or whenever the schema/key
			// order differs from the previously rendered table (T-1315).
			keyOrder := contentItem.getSchema().GetKeyOrder()
			writeHeaders := !keyOrdersEqual(*lastKeyOrder, keyOrder)
			if err := c.renderTableContentCSV(contentItem, csvWriter, writeHeaders); err != nil {
				return fmt.Errorf("failed to render section table: %w", err)
			}
			*lastKeyOrder = keyOrder

		default:
			// For non-table content, add as metadata (Requirement 15.8)
			metadataRow := []string{fmt.Sprintf("# Content %d: %s", i+1, content.Type())}
			if err := csvWriter.Write(metadataRow); err != nil {
				return fmt.Errorf("failed to write content metadata: %w", err)
			}

			// Try to get text representation
			if contentText, err := content.AppendText(nil); err == nil && len(contentText) > 0 {
				contentRow := []string{c.formatValueForCSV(string(contentText))}
				if err := csvWriter.Write(contentRow); err != nil {
					return fmt.Errorf("failed to write content: %w", err)
				}
			}
		}
	}

	return nil
}

// keyOrdersEqual returns true if two key order slices are identical
func keyOrdersEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
