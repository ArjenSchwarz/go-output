package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// jsonRenderer implements JSON output format
type jsonRenderer struct {
	baseRenderer
}

func (j *jsonRenderer) Format() string {
	return FormatJSON
}

func (j *jsonRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return j.renderDocumentJSON(ctx, doc)
}

func (j *jsonRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	data, err := j.renderDocumentJSON(ctx, doc)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (j *jsonRenderer) SupportsStreaming() bool {
	return true
}

// renderDocumentJSON renders entire document as a single JSON structure
func (j *jsonRenderer) renderDocumentJSON(ctx context.Context, doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	contents := doc.GetContents()

	// If single content, render it directly
	if len(contents) == 1 {
		return j.renderContent(contents[0])
	}

	// Multiple contents: create a JSON array
	var contentArray []any
	for _, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		contentBytes, err := j.renderContent(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render content %s: %w", content.ID(), err)
		}

		var contentData any
		if err := json.Unmarshal(contentBytes, &contentData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content JSON: %w", err)
		}
		contentArray = append(contentArray, contentData)
	}

	return json.MarshalIndent(contentArray, "", "  ")
}

// renderContent renders content specifically for JSON format
func (j *jsonRenderer) renderContent(content Content) ([]byte, error) {
	switch c := content.(type) {
	case *TableContent:
		return j.renderTableContentJSON(c)
	case *TextContent:
		return j.renderTextContentJSON(c)
	case *RawContent:
		return j.renderRawContentJSON(c)
	case *SectionContent:
		return j.renderSectionContentJSON(c)
	case *ChartContent:
		return j.renderChartContentJSON(c)
	case *GraphContent:
		return j.renderGraphContentJSON(c)
	case *DrawIOContent:
		return j.renderDrawIOContentJSON(c)
	default:
		// Fallback to basic rendering - wrap plain text as JSON string
		textData, err := j.baseRenderer.renderContent(content)
		if err != nil {
			return nil, err
		}
		return json.Marshal(string(textData))
	}
}

// renderContentTo renders content to a writer for JSON format with streaming support
func (j *jsonRenderer) renderContentTo(content Content, w io.Writer) error {
	switch c := content.(type) {
	case *TableContent:
		return j.renderTableContentJSONStream(c, w)
	case *TextContent:
		return j.renderTextContentJSONStream(c, w)
	case *RawContent:
		return j.renderRawContentJSONStream(c, w)
	case *SectionContent:
		return j.renderSectionContentJSONStream(c, w)
	case *ChartContent, *GraphContent, *DrawIOContent:
		// These complex types fall back to buffered rendering
		data, err := j.renderContent(content)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		// Fallback to buffered rendering
		data, err := j.renderContent(content)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
}

// renderTableContentJSON renders table content as JSON with key order preservation
func (j *jsonRenderer) renderTableContentJSON(table *TableContent) ([]byte, error) {
	result := make(map[string]any)

	if table.Title() != "" {
		result["title"] = table.Title()
	}

	// Convert records to ordered map preserving key order
	keyOrder := table.Schema().GetKeyOrder()
	var tableData []map[string]any

	for _, record := range table.Records() {
		// Create ordered map that preserves key order for JSON marshaling
		orderedRecord := make(map[string]any)

		// Add keys in the specified order
		for _, key := range keyOrder {
			if val, exists := record[key]; exists {
				// Find field for this key to apply formatter
				field := table.Schema().FindField(key)
				// Process field value and handle CollapsibleValue
				processedVal := j.formatValueForJSON(val, field)
				orderedRecord[key] = processedVal
			}
		}

		tableData = append(tableData, orderedRecord)
	}

	result["data"] = tableData
	result["schema"] = map[string]any{
		"keys":   keyOrder,
		"fields": j.convertFieldsToJSON(table.Schema()),
	}

	return json.MarshalIndent(result, "", "  ")
}

// formatValueForJSON processes field values and handles CollapsibleValue interface
func (j *jsonRenderer) formatValueForJSON(val any, field *Field) any {
	// Apply field formatter if present
	processed := j.processFieldValue(val, field)

	// Check if result is CollapsibleValue (Requirement 4.1)
	if cv, ok := processed.(CollapsibleValue); ok {
		result := map[string]any{
			"type":     "collapsible",   // Requirement 4.1: type indication
			"summary":  cv.Summary(),    // Requirement 4.2: include summary
			"details":  cv.Details(),    // Requirement 4.2: include details
			"expanded": cv.IsExpanded(), // Requirement 4.2: include expanded
		}

		// Add format-specific hints (Requirement 4.3)
		if hints := cv.FormatHint(FormatJSON); hints != nil {
			for k, v := range hints {
				result[k] = v
			}
		}

		return result
	}

	return processed
}

// renderTextContentJSON renders text content as JSON
func (j *jsonRenderer) renderTextContentJSON(text *TextContent) ([]byte, error) {
	result := map[string]any{
		"type":    FormatText,
		"content": text.Text(),
	}

	style := text.Style()
	if style.Bold || style.Italic || style.Color != "" || style.Size > 0 || style.Header {
		result["style"] = map[string]any{
			"bold":   style.Bold,
			"italic": style.Italic,
			"color":  style.Color,
			"size":   style.Size,
			"header": style.Header,
		}
	}

	return json.MarshalIndent(result, "", "  ")
}

// renderRawContentJSON renders raw content as JSON
func (j *jsonRenderer) renderRawContentJSON(raw *RawContent) ([]byte, error) {
	result := map[string]any{
		"type":   "raw",
		"format": raw.Format(),
		"data":   string(raw.Data()),
	}

	return json.MarshalIndent(result, "", "  ")
}

// renderSectionContentJSON renders section content as JSON with nested content
func (j *jsonRenderer) renderSectionContentJSON(section *SectionContent) ([]byte, error) {
	result := map[string]any{
		"type":  "section",
		"title": section.Title(),
		"level": section.Level(),
	}

	var contents []any
	for _, content := range section.Contents() {
		contentJSON, err := j.renderContent(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render nested content: %w", err)
		}

		var contentData any
		if err := json.Unmarshal(contentJSON, &contentData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content JSON: %w", err)
		}
		contents = append(contents, contentData)
	}

	result["contents"] = contents

	return json.MarshalIndent(result, "", "  ")
}

// convertFieldsToJSON converts schema fields to JSON representation
func (j *jsonRenderer) convertFieldsToJSON(schema *Schema) []map[string]any {
	var fields []map[string]any

	for _, field := range schema.Fields {
		fieldMap := map[string]any{
			"name":   field.Name,
			"type":   field.Type,
			"hidden": field.Hidden,
		}
		fields = append(fields, fieldMap)
	}

	return fields
}

// Streaming implementations for large datasets

// renderTableContentJSONStream renders table content as JSON directly to writer
func (j *jsonRenderer) renderTableContentJSONStream(table *TableContent, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	// Create structure step by step to avoid loading all data in memory
	result := make(map[string]any)

	if table.Title() != "" {
		result["title"] = table.Title()
	}

	// Add schema information
	keyOrder := table.Schema().GetKeyOrder()
	result["schema"] = map[string]any{
		"keys":   keyOrder,
		"fields": j.convertFieldsToJSON(table.Schema()),
	}

	// For streaming, we need to handle data differently
	// First write the opening structure
	if _, err := w.Write([]byte("{\n")); err != nil {
		return fmt.Errorf("failed to write opening brace: %w", err)
	}

	if table.Title() != "" {
		if _, err := fmt.Fprintf(w, "  \"title\": %q,\n", table.Title()); err != nil {
			return fmt.Errorf("failed to write title: %w", err)
		}
	}

	// Write schema
	if _, err := w.Write([]byte("  \"schema\": {\n")); err != nil {
		return fmt.Errorf("failed to write schema header: %w", err)
	}

	// Write keys
	if _, err := w.Write([]byte("    \"keys\": [\n")); err != nil {
		return fmt.Errorf("failed to write keys header: %w", err)
	}
	for i, key := range keyOrder {
		if i > 0 {
			if _, err := w.Write([]byte(",\n")); err != nil {
				return fmt.Errorf("failed to write key separator: %w", err)
			}
		}
		if _, err := fmt.Fprintf(w, "      %q", key); err != nil {
			return fmt.Errorf("failed to write key: %w", err)
		}
	}
	if _, err := w.Write([]byte("\n    ],\n")); err != nil {
		return fmt.Errorf("failed to write keys footer: %w", err)
	}

	// Write fields
	if _, err := w.Write([]byte("    \"fields\": [\n")); err != nil {
		return fmt.Errorf("failed to write fields header: %w", err)
	}
	fields := j.convertFieldsToJSON(table.Schema())
	for i, field := range fields {
		if i > 0 {
			if _, err := w.Write([]byte(",\n")); err != nil {
				return fmt.Errorf("failed to write field separator: %w", err)
			}
		}
		fieldJSON, err := json.MarshalIndent(field, "      ", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal field: %w", err)
		}
		if _, err := w.Write([]byte("      ")); err != nil {
			return fmt.Errorf("failed to write field indent: %w", err)
		}
		if _, err := w.Write(fieldJSON); err != nil {
			return fmt.Errorf("failed to write field JSON: %w", err)
		}
	}
	if _, err := w.Write([]byte("\n    ]\n")); err != nil {
		return fmt.Errorf("failed to write fields footer: %w", err)
	}
	if _, err := w.Write([]byte("  },\n")); err != nil {
		return fmt.Errorf("failed to write schema footer: %w", err)
	}

	// Stream data records
	if _, err := w.Write([]byte("  \"data\": [\n")); err != nil {
		return fmt.Errorf("failed to write data header: %w", err)
	}
	records := table.Records()
	for i, record := range records {
		if i > 0 {
			if _, err := w.Write([]byte(",\n")); err != nil {
				return fmt.Errorf("failed to write record separator: %w", err)
			}
		}

		// Create ordered record preserving key order
		orderedRecord := make(map[string]any)
		for _, key := range keyOrder {
			if val, exists := record[key]; exists {
				// Find field for this key to apply formatter
				field := table.Schema().FindField(key)
				// Process field value and handle CollapsibleValue
				processedVal := j.formatValueForJSON(val, field)
				orderedRecord[key] = processedVal
			}
		}

		recordJSON, err := json.MarshalIndent(orderedRecord, "    ", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal record: %w", err)
		}
		if _, err := w.Write([]byte("    ")); err != nil {
			return fmt.Errorf("failed to write record indent: %w", err)
		}
		if _, err := w.Write(recordJSON); err != nil {
			return fmt.Errorf("failed to write record JSON: %w", err)
		}
	}
	if _, err := w.Write([]byte("\n  ]\n")); err != nil {
		return fmt.Errorf("failed to write data footer: %w", err)
	}
	if _, err := w.Write([]byte("}\n")); err != nil {
		return fmt.Errorf("failed to write closing brace: %w", err)
	}

	return nil
}

// renderTextContentJSONStream renders text content as JSON to writer
func (j *jsonRenderer) renderTextContentJSONStream(text *TextContent, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	result := map[string]any{
		"type":    FormatText,
		"content": text.Text(),
	}

	style := text.Style()
	if style.Bold || style.Italic || style.Color != "" || style.Size > 0 || style.Header {
		result["style"] = map[string]any{
			"bold":   style.Bold,
			"italic": style.Italic,
			"color":  style.Color,
			"size":   style.Size,
			"header": style.Header,
		}
	}

	return encoder.Encode(result)
}

// renderRawContentJSONStream renders raw content as JSON to writer
func (j *jsonRenderer) renderRawContentJSONStream(raw *RawContent, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	result := map[string]any{
		"type":   "raw",
		"format": raw.Format(),
		"data":   string(raw.Data()),
	}

	return encoder.Encode(result)
}

// renderSectionContentJSONStream renders section content as JSON to writer
func (j *jsonRenderer) renderSectionContentJSONStream(section *SectionContent, w io.Writer) error {
	// For sections with nested content, we need a more complex streaming approach
	if _, err := w.Write([]byte("{\n")); err != nil {
		return fmt.Errorf("failed to write opening brace: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  \"type\": \"section\",\n"); err != nil {
		return fmt.Errorf("failed to write type: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  \"title\": %q,\n", section.Title()); err != nil {
		return fmt.Errorf("failed to write title: %w", err)
	}
	if _, err := fmt.Fprintf(w, "  \"level\": %d,\n", section.Level()); err != nil {
		return fmt.Errorf("failed to write level: %w", err)
	}
	if _, err := w.Write([]byte("  \"contents\": [\n")); err != nil {
		return fmt.Errorf("failed to write contents header: %w", err)
	}

	contents := section.Contents()
	for i, content := range contents {
		if i > 0 {
			if _, err := w.Write([]byte(",\n")); err != nil {
				return fmt.Errorf("failed to write content separator: %w", err)
			}
		}

		// Create a buffer for the nested content
		var buf bytes.Buffer
		if err := j.renderContentTo(content, &buf); err != nil {
			return fmt.Errorf("failed to render nested content: %w", err)
		}

		// Indent the nested content
		lines := bytes.Split(buf.Bytes(), []byte("\n"))
		for k, line := range lines {
			if k > 0 {
				if _, err := w.Write([]byte("\n")); err != nil {
					return fmt.Errorf("failed to write line break: %w", err)
				}
			}
			if len(line) > 0 {
				if _, err := w.Write([]byte("    ")); err != nil {
					return fmt.Errorf("failed to write indent: %w", err)
				}
				if _, err := w.Write(line); err != nil {
					return fmt.Errorf("failed to write line: %w", err)
				}
			}
		}
	}

	if _, err := w.Write([]byte("\n  ]\n")); err != nil {
		return fmt.Errorf("failed to write contents footer: %w", err)
	}
	if _, err := w.Write([]byte("}\n")); err != nil {
		return fmt.Errorf("failed to write closing brace: %w", err)
	}

	return nil
}

// renderChartContentJSON renders ChartContent as JSON
func (j *jsonRenderer) renderChartContentJSON(content *ChartContent) ([]byte, error) {
	chartData := map[string]any{
		"type":       content.Type(),
		"title":      content.GetTitle(),
		"chart_type": content.GetChartType(),
		"data":       content.GetData(),
	}
	return json.MarshalIndent(chartData, "", "  ")
}

// renderGraphContentJSON renders GraphContent as JSON
func (j *jsonRenderer) renderGraphContentJSON(content *GraphContent) ([]byte, error) {
	graphData := map[string]any{
		"type":  content.Type(),
		"title": content.GetTitle(),
		"nodes": content.GetNodes(),
		"edges": content.GetEdges(),
	}
	return json.MarshalIndent(graphData, "", "  ")
}

// renderDrawIOContentJSON renders DrawIOContent as JSON
func (j *jsonRenderer) renderDrawIOContentJSON(content *DrawIOContent) ([]byte, error) {
	drawioData := map[string]any{
		"type":    content.Type(),
		"title":   content.GetTitle(),
		"records": content.GetRecords(),
		"header":  content.GetHeader(),
	}
	return json.MarshalIndent(drawioData, "", "  ")
}

// yamlRenderer implements YAML output format
type yamlRenderer struct {
	baseRenderer
}

func (y *yamlRenderer) Format() string {
	return FormatYAML
}

func (y *yamlRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return y.renderDocumentYAML(ctx, doc)
}

func (y *yamlRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	data, err := y.renderDocumentYAML(ctx, doc)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (y *yamlRenderer) SupportsStreaming() bool {
	return true
}

// renderDocumentYAML renders entire document as a single YAML structure
func (y *yamlRenderer) renderDocumentYAML(ctx context.Context, doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	contents := doc.GetContents()

	// If single content, render it directly
	if len(contents) == 1 {
		return y.renderContent(contents[0])
	}

	// Multiple contents: create a YAML array
	var contentArray []any
	for _, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		contentBytes, err := y.renderContent(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render content %s: %w", content.ID(), err)
		}

		var contentData any
		if err := yaml.Unmarshal(contentBytes, &contentData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content YAML: %w", err)
		}
		contentArray = append(contentArray, contentData)
	}

	return yaml.Marshal(contentArray)
}

// renderContent renders content specifically for YAML format
func (y *yamlRenderer) renderContent(content Content) ([]byte, error) {
	switch c := content.(type) {
	case *TableContent:
		return y.renderTableContentYAML(c)
	case *TextContent:
		return y.renderTextContentYAML(c)
	case *RawContent:
		return y.renderRawContentYAML(c)
	case *SectionContent:
		return y.renderSectionContentYAML(c)
	case *ChartContent:
		return y.renderChartContentYAML(c)
	case *GraphContent:
		return y.renderGraphContentYAML(c)
	case *DrawIOContent:
		return y.renderDrawIOContentYAML(c)
	default:
		// Fallback to basic rendering - wrap plain text as YAML string
		textData, err := y.baseRenderer.renderContent(content)
		if err != nil {
			return nil, err
		}
		return yaml.Marshal(string(textData))
	}
}

// renderContentTo renders content to a writer for YAML format with streaming support
func (y *yamlRenderer) renderContentTo(content Content, w io.Writer) error {
	switch c := content.(type) {
	case *TableContent:
		return y.renderTableContentYAMLStream(c, w)
	case *TextContent:
		return y.renderTextContentYAMLStream(c, w)
	case *RawContent:
		return y.renderRawContentYAMLStream(c, w)
	case *SectionContent:
		return y.renderSectionContentYAMLStream(c, w)
	case *ChartContent, *GraphContent, *DrawIOContent:
		// These complex types fall back to buffered rendering
		data, err := y.renderContent(content)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		// Fallback to buffered rendering
		data, err := y.renderContent(content)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
}

// renderTableContentYAML renders table content as YAML with key order preservation
func (y *yamlRenderer) renderTableContentYAML(table *TableContent) ([]byte, error) {
	result := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	// Add title if present
	if table.Title() != "" {
		result.Content = append(result.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "title"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: table.Title()},
		)
	}

	// Create schema node with key order preservation
	schemaNode := &yaml.Node{Kind: yaml.MappingNode}

	// Add keys array
	keyOrder := table.Schema().GetKeyOrder()
	keysArrayNode := &yaml.Node{Kind: yaml.SequenceNode}
	for _, key := range keyOrder {
		keysArrayNode.Content = append(keysArrayNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		)
	}

	schemaNode.Content = append(schemaNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "keys"},
		keysArrayNode,
	)

	// Add fields array
	fieldsArrayNode := &yaml.Node{Kind: yaml.SequenceNode}
	for _, field := range table.Schema().Fields {
		fieldNode := &yaml.Node{Kind: yaml.MappingNode}
		fieldNode.Content = append(fieldNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "name"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: field.Name},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "type"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: field.Type},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "hidden"},
			y.createYAMLValueNode(field.Hidden),
		)
		fieldsArrayNode.Content = append(fieldsArrayNode.Content, fieldNode)
	}

	schemaNode.Content = append(schemaNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "fields"},
		fieldsArrayNode,
	)

	result.Content = append(result.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "schema"},
		schemaNode,
	)

	// Create data array with preserved key order
	dataArrayNode := &yaml.Node{Kind: yaml.SequenceNode}
	for _, record := range table.Records() {
		recordNode := &yaml.Node{Kind: yaml.MappingNode}

		// Add keys in the specified order
		for _, key := range keyOrder {
			if val, exists := record[key]; exists {
				// Find field for this key to apply formatter
				field := table.Schema().FindField(key)
				// Process field value and handle CollapsibleValue
				processedVal := y.formatValueForYAML(val, field)
				recordNode.Content = append(recordNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: key},
					y.createYAMLValueNode(processedVal),
				)
			}
		}

		dataArrayNode.Content = append(dataArrayNode.Content, recordNode)
	}

	result.Content = append(result.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "data"},
		dataArrayNode,
	)

	return yaml.Marshal(result)
}

// renderTextContentYAML renders text content as YAML
func (y *yamlRenderer) renderTextContentYAML(text *TextContent) ([]byte, error) {
	result := map[string]any{
		"type":    FormatText,
		"content": text.Text(),
	}

	style := text.Style()
	if style.Bold || style.Italic || style.Color != "" || style.Size > 0 || style.Header {
		result["style"] = map[string]any{
			"bold":   style.Bold,
			"italic": style.Italic,
			"color":  style.Color,
			"size":   style.Size,
			"header": style.Header,
		}
	}

	return yaml.Marshal(result)
}

// renderRawContentYAML renders raw content as YAML
func (y *yamlRenderer) renderRawContentYAML(raw *RawContent) ([]byte, error) {
	result := map[string]any{
		"type":   "raw",
		"format": raw.Format(),
		"data":   string(raw.Data()),
	}

	return yaml.Marshal(result)
}

// renderSectionContentYAML renders section content as YAML with nested content
func (y *yamlRenderer) renderSectionContentYAML(section *SectionContent) ([]byte, error) {
	result := map[string]any{
		"type":  "section",
		"title": section.Title(),
		"level": section.Level(),
	}

	var contents []any
	for _, content := range section.Contents() {
		contentYAML, err := y.renderContent(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render nested content: %w", err)
		}

		var contentData any
		if err := yaml.Unmarshal(contentYAML, &contentData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content YAML: %w", err)
		}
		contents = append(contents, contentData)
	}

	result["contents"] = contents

	return yaml.Marshal(result)
}

// createYAMLValueNode creates a yaml.Node for any value type
func (y *yamlRenderer) createYAMLValueNode(val any) *yaml.Node {
	switch v := val.(type) {
	case string:
		// For empty strings, we need to ensure they are preserved as empty strings
		// and not interpreted as null by YAML parsers
		if v == "" {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"}
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v}
	case bool:
		if v {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: "true"}
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "false"}
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	case map[string]any:
		// Handle map structures (like CollapsibleValue results)
		mapNode := &yaml.Node{Kind: yaml.MappingNode}
		for key, value := range v {
			mapNode.Content = append(mapNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: key},
				y.createYAMLValueNode(value),
			)
		}
		return mapNode
	case []any:
		// Handle array structures
		arrayNode := &yaml.Node{Kind: yaml.SequenceNode}
		for _, item := range v {
			arrayNode.Content = append(arrayNode.Content, y.createYAMLValueNode(item))
		}
		return arrayNode
	case []string:
		// Handle string array structures
		arrayNode := &yaml.Node{Kind: yaml.SequenceNode}
		for _, item := range v {
			arrayNode.Content = append(arrayNode.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: item})
		}
		return arrayNode
	default:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", v)}
	}
}

// formatValueForYAML processes field values and handles CollapsibleValue interface
func (y *yamlRenderer) formatValueForYAML(val any, field *Field) any {
	// Apply field formatter if present
	processed := y.processFieldValue(val, field)

	// Check if result is CollapsibleValue (Requirement 5.1)
	if cv, ok := processed.(CollapsibleValue); ok {
		result := map[string]any{
			"summary":  cv.Summary(), // Requirement 5.1: YAML mapping
			"details":  cv.Details(), // Requirement 5.1: with these fields
			"expanded": cv.IsExpanded(),
		}

		// YAML-specific formatting hints (Requirement 5.2)
		if hints := cv.FormatHint(FormatYAML); hints != nil {
			for k, v := range hints {
				result[k] = v
			}
		}

		return result
	}

	return processed
}

// Streaming implementations for large datasets

// renderTableContentYAMLStream renders table content as YAML directly to writer
func (y *yamlRenderer) renderTableContentYAMLStream(table *TableContent, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func() {
		_ = encoder.Close()
	}()

	// Create structure manually to preserve key order
	result := make(map[string]any)

	if table.Title() != "" {
		result["title"] = table.Title()
	}

	// Add schema information
	keyOrder := table.Schema().GetKeyOrder()
	fields := make([]map[string]any, 0, len(table.Schema().Fields))
	for _, field := range table.Schema().Fields {
		fieldMap := map[string]any{
			"name":   field.Name,
			"type":   field.Type,
			"hidden": field.Hidden,
		}
		fields = append(fields, fieldMap)
	}

	result["schema"] = map[string]any{
		"keys":   keyOrder,
		"fields": fields,
	}

	// Convert records to ordered maps preserving key order
	var tableData []map[string]any
	for _, record := range table.Records() {
		orderedRecord := make(map[string]any)

		// Add keys in the specified order
		for _, key := range keyOrder {
			if val, exists := record[key]; exists {
				// Find field for this key to apply formatter
				field := table.Schema().FindField(key)
				// Process field value and handle CollapsibleValue
				processedVal := y.formatValueForYAML(val, field)
				orderedRecord[key] = processedVal
			}
		}

		tableData = append(tableData, orderedRecord)
	}

	result["data"] = tableData

	return encoder.Encode(result)
}

// renderTextContentYAMLStream renders text content as YAML to writer
func (y *yamlRenderer) renderTextContentYAMLStream(text *TextContent, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func() {
		_ = encoder.Close()
	}()

	result := map[string]any{
		"type":    FormatText,
		"content": text.Text(),
	}

	style := text.Style()
	if style.Bold || style.Italic || style.Color != "" || style.Size > 0 || style.Header {
		result["style"] = map[string]any{
			"bold":   style.Bold,
			"italic": style.Italic,
			"color":  style.Color,
			"size":   style.Size,
			"header": style.Header,
		}
	}

	return encoder.Encode(result)
}

// renderRawContentYAMLStream renders raw content as YAML to writer
func (y *yamlRenderer) renderRawContentYAMLStream(raw *RawContent, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func() {
		_ = encoder.Close()
	}()

	result := map[string]any{
		"type":   "raw",
		"format": raw.Format(),
		"data":   string(raw.Data()),
	}

	return encoder.Encode(result)
}

// renderSectionContentYAMLStream renders section content as YAML to writer
func (y *yamlRenderer) renderSectionContentYAMLStream(section *SectionContent, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func() {
		_ = encoder.Close()
	}()

	result := map[string]any{
		"type":  "section",
		"title": section.Title(),
		"level": section.Level(),
	}

	var contents []any
	for _, content := range section.Contents() {
		contentYAML, err := y.renderContent(content)
		if err != nil {
			return fmt.Errorf("failed to render nested content: %w", err)
		}

		var contentData any
		if err := yaml.Unmarshal(contentYAML, &contentData); err != nil {
			return fmt.Errorf("failed to unmarshal content YAML: %w", err)
		}
		contents = append(contents, contentData)
	}

	result["contents"] = contents

	return encoder.Encode(result)
}

// renderChartContentYAML renders ChartContent as YAML
func (y *yamlRenderer) renderChartContentYAML(content *ChartContent) ([]byte, error) {
	chartData := map[string]any{
		"type":       content.Type(),
		"title":      content.GetTitle(),
		"chart_type": content.GetChartType(),
		"data":       content.GetData(),
	}
	return yaml.Marshal(chartData)
}

// renderGraphContentYAML renders GraphContent as YAML
func (y *yamlRenderer) renderGraphContentYAML(content *GraphContent) ([]byte, error) {
	graphData := map[string]any{
		"type":  content.Type(),
		"title": content.GetTitle(),
		"nodes": content.GetNodes(),
		"edges": content.GetEdges(),
	}
	return yaml.Marshal(graphData)
}

// renderDrawIOContentYAML renders DrawIOContent as YAML
func (y *yamlRenderer) renderDrawIOContentYAML(content *DrawIOContent) ([]byte, error) {
	drawioData := map[string]any{
		"type":    content.Type(),
		"title":   content.GetTitle(),
		"records": content.GetRecords(),
		"header":  content.GetHeader(),
	}
	return yaml.Marshal(drawioData)
}
