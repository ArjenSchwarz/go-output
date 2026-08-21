package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"

	"gopkg.in/yaml.v3"
)

// marshalFunc is a function that marshals data to bytes
type marshalFunc func(v any) ([]byte, error)

// wrapContentFunc converts rendered content bytes into a value that can be
// embedded in a larger structure without losing information. Implementations
// must preserve key order (e.g. json.RawMessage for JSON, *yaml.Node for
// YAML) — unmarshaling into maps would destroy it.
type wrapContentFunc func(data []byte) (any, error)

// renderDocumentGeneric is a shared implementation for rendering documents in different formats
func renderDocumentGeneric(
	ctx context.Context,
	doc *Document,
	format string,
	renderContent func(Content) ([]byte, error),
	wrap wrapContentFunc,
	marshal marshalFunc,
) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	contents := doc.GetContents()

	// If single content, apply transformations and render it directly
	if len(contents) == 1 {
		transformed, err := applyContentTransformations(ctx, contents[0])
		if err != nil {
			return nil, err
		}
		return renderContent(transformed)
	}

	// Multiple contents: create an array
	var contentArray []any
	for _, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply transformations before rendering
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}

		contentBytes, err := renderContent(transformed)
		if err != nil {
			return nil, fmt.Errorf("failed to render content %s: %w", content.ID(), err)
		}

		contentData, err := wrap(contentBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to wrap %s content: %w", format, err)
		}
		contentArray = append(contentArray, contentData)
	}

	return marshal(contentArray)
}

// jsonMember is a single key/value pair of an orderedJSONObject.
type jsonMember struct {
	key   string
	value any
}

// orderedJSONObject marshals as a JSON object whose members appear in slice
// order. encoding/json sorts map keys alphabetically, so key order
// preservation — the library's core guarantee — requires this custom
// marshaler. Indentation is applied by the caller (json.MarshalIndent and
// json.Encoder re-indent marshaler output).
type orderedJSONObject []jsonMember

func (o orderedJSONObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, member := range o {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(member.key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		value, err := json.Marshal(member.value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal value for key %q: %w", member.key, err)
		}
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// yamlContentNode parses rendered YAML content into a *yaml.Node so nested
// structures keep their key order when re-marshaled as part of a larger
// document (yaml.Unmarshal into any would produce order-destroying maps).
func yamlContentNode(data []byte) (any, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0], nil
	}
	if node.Kind == 0 {
		// Empty document (no content) marshals as null.
		return nil, nil
	}
	return &node, nil
}

// buildTextContentData creates a generic representation of text content
func buildTextContentData(text *TextContent) map[string]any {
	result := map[string]any{
		keyType:    FormatText,
		keyContent: text.Text(),
	}

	style := text.Style()
	if style.Bold || style.Italic || style.Color != "" || style.Size > 0 || style.Header {
		result["style"] = map[string]any{
			keyBold:   style.Bold,
			keyItalic: style.Italic,
			keyColor:  style.Color,
			keySize:   style.Size,
			keyHeader: style.Header,
		}
	}

	return result
}

// buildRawContentData creates a generic representation of raw content
func buildRawContentData(raw *RawContent) map[string]any {
	return map[string]any{
		keyType:   contentTypeNameRaw,
		keyFormat: raw.Format(),
		keyData:   string(raw.Data()),
	}
}

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
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
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
	return renderDocumentGeneric(ctx, doc, "JSON", func(content Content) ([]byte, error) {
		return j.renderContent(ctx, content)
	}, func(data []byte) (any, error) {
		// Embed the already-rendered JSON verbatim; unmarshaling into maps
		// would destroy key order.
		return json.RawMessage(data), nil
	}, func(v any) ([]byte, error) {
		return json.MarshalIndent(v, "", "  ")
	})
}

// renderContent renders content specifically for JSON format
func (j *jsonRenderer) renderContent(ctx context.Context, content Content) ([]byte, error) {
	switch c := content.(type) {
	case *TableContent:
		return j.renderTableContentJSON(c)
	case *TextContent:
		return j.renderTextContentJSON(c)
	case *RawContent:
		return j.renderRawContentJSON(c)
	case *SectionContent:
		return j.renderSectionContentJSON(ctx, c)
	case *DefaultCollapsibleSection:
		return j.renderCollapsibleSectionJSON(ctx, c)
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
func (j *jsonRenderer) renderContentTo(ctx context.Context, content Content, w io.Writer) error {
	switch c := content.(type) {
	case *TableContent:
		return j.renderTableContentJSONStream(c, w)
	case *TextContent:
		return j.renderTextContentJSONStream(c, w)
	case *RawContent:
		return j.renderRawContentJSONStream(c, w)
	case *SectionContent:
		return j.renderSectionContentJSONStream(ctx, c, w)
	case *ChartContent, *GraphContent, *DrawIOContent:
		// These complex types fall back to buffered rendering
		data, err := j.renderContent(ctx, content)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		// Fallback to buffered rendering
		data, err := j.renderContent(ctx, content)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
}

// renderTableContentJSON renders table content as JSON with key order preservation
func (j *jsonRenderer) renderTableContentJSON(table *TableContent) ([]byte, error) {
	return json.MarshalIndent(j.buildTableContentJSON(table), "", "  ")
}

// buildTableContentJSON builds the ordered JSON structure for table content.
// Record objects serialize their keys in the user-specified order (the
// library's core guarantee); the envelope is title, schema, data.
func (j *jsonRenderer) buildTableContentJSON(table *TableContent) orderedJSONObject {
	var result orderedJSONObject

	if table.Title() != "" {
		result = append(result, jsonMember{keyTitle, table.Title()})
	}

	keyOrder := table.getSchema().GetKeyOrder()

	var tableData []any
	for _, record := range table.Records() {
		// Build an ordered record preserving key order
		var orderedRecord orderedJSONObject
		for _, key := range keyOrder {
			if val, exists := record[key]; exists {
				// Find field for this key to apply formatter
				field := table.getSchema().FindField(key)
				// Process field value and handle CollapsibleValue
				orderedRecord = append(orderedRecord, jsonMember{key, j.formatValueForJSON(val, field)})
			}
		}
		tableData = append(tableData, orderedRecord)
	}

	return append(result,
		jsonMember{"schema", orderedJSONObject{
			{keyKeys, keyOrder},
			{keyFields, j.convertFieldsToJSON(table.getSchema())},
		}},
		jsonMember{keyData, tableData},
	)
}

// formatValueForJSON processes field values and handles CollapsibleValue interface
func (j *jsonRenderer) formatValueForJSON(val any, field *Field) any {
	// Apply field formatter if present
	processed := j.processFieldValue(val, field)

	// Check if result is CollapsibleValue (Requirement 4.1)
	if cv, ok := processed.(CollapsibleValue); ok {
		result := map[string]any{
			keyType:     "collapsible",   // Requirement 4.1: type indication
			keySummary:  cv.Summary(),    // Requirement 4.2: include summary
			keyDetails:  cv.Details(),    // Requirement 4.2: include details
			keyExpanded: cv.IsExpanded(), // Requirement 4.2: include expanded
		}

		// Add format-specific hints (Requirement 4.3)
		if hints := cv.FormatHint(FormatJSON); hints != nil {
			maps.Copy(result, hints)
		}

		return result
	}

	return processed
}

// renderTextContentJSON renders text content as JSON
func (j *jsonRenderer) renderTextContentJSON(text *TextContent) ([]byte, error) {
	result := buildTextContentData(text)
	return json.MarshalIndent(result, "", "  ")
}

// renderRawContentJSON renders raw content as JSON
func (j *jsonRenderer) renderRawContentJSON(raw *RawContent) ([]byte, error) {
	result := buildRawContentData(raw)
	return json.MarshalIndent(result, "", "  ")
}

// renderSectionContentJSON renders section content as JSON with nested content
func (j *jsonRenderer) renderSectionContentJSON(ctx context.Context, section *SectionContent) ([]byte, error) {
	result, err := j.buildSectionContentJSON(ctx, section)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(result, "", "  ")
}

// buildSectionContentJSON builds the ordered JSON structure for section
// content, embedding each nested content's rendered JSON verbatim to keep
// nested key order intact.
func (j *jsonRenderer) buildSectionContentJSON(ctx context.Context, section *SectionContent) (orderedJSONObject, error) {
	var contents []any
	for _, content := range section.Contents() {
		// Apply per-content transformations before rendering
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}

		contentJSON, err := j.renderContent(ctx, transformed)
		if err != nil {
			return nil, fmt.Errorf("failed to render nested content: %w", err)
		}

		contents = append(contents, json.RawMessage(contentJSON))
	}

	return orderedJSONObject{
		{keyType, contentTypeNameSection},
		{keyTitle, section.Title()},
		{keyLevel, section.Level()},
		{"contents", contents},
	}, nil
}

// convertFieldsToJSON converts schema fields to JSON representation
func (j *jsonRenderer) convertFieldsToJSON(schema *Schema) []map[string]any {
	var fields []map[string]any

	for _, field := range schema.Fields {
		fieldMap := map[string]any{
			keyName:   field.Name,
			keyType:   field.Type,
			keyHidden: field.Hidden,
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
	return encoder.Encode(j.buildTableContentJSON(table))
}

// renderTextContentJSONStream renders text content as JSON to writer
func (j *jsonRenderer) renderTextContentJSONStream(text *TextContent, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(buildTextContentData(text))
}

// renderRawContentJSONStream renders raw content as JSON to writer
func (j *jsonRenderer) renderRawContentJSONStream(raw *RawContent, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(buildRawContentData(raw))
}

// renderSectionContentJSONStream renders section content as JSON to writer
func (j *jsonRenderer) renderSectionContentJSONStream(ctx context.Context, section *SectionContent, w io.Writer) error {
	result, err := j.buildSectionContentJSON(ctx, section)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// renderChartContentJSON renders ChartContent as JSON
func (j *jsonRenderer) renderChartContentJSON(content *ChartContent) ([]byte, error) {
	chartData := map[string]any{
		keyType:      content.Type(),
		keyTitle:     content.GetTitle(),
		"chart_type": content.GetChartType(),
		keyData:      content.GetData(),
	}
	return json.MarshalIndent(chartData, "", "  ")
}

// renderGraphContentJSON renders GraphContent as JSON
func (j *jsonRenderer) renderGraphContentJSON(content *GraphContent) ([]byte, error) {
	graphData := map[string]any{
		keyType:  content.Type(),
		keyTitle: content.GetTitle(),
		"nodes":  content.GetNodes(),
		"edges":  content.GetEdges(),
	}
	return json.MarshalIndent(graphData, "", "  ")
}

// renderDrawIOContentJSON renders DrawIOContent as JSON
func (j *jsonRenderer) renderDrawIOContentJSON(content *DrawIOContent) ([]byte, error) {
	drawioData := map[string]any{
		keyType:   content.Type(),
		keyTitle:  content.GetTitle(),
		"records": content.GetRecords(),
		keyHeader: content.GetHeader(),
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
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
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
	return renderDocumentGeneric(ctx, doc, "YAML", func(content Content) ([]byte, error) {
		return y.renderContent(ctx, content)
	}, yamlContentNode, yaml.Marshal)
}

// renderContent renders content specifically for YAML format
func (y *yamlRenderer) renderContent(ctx context.Context, content Content) ([]byte, error) {
	switch c := content.(type) {
	case *TableContent:
		return y.renderTableContentYAML(c)
	case *TextContent:
		return y.renderTextContentYAML(c)
	case *RawContent:
		return y.renderRawContentYAML(c)
	case *SectionContent:
		return y.renderSectionContentYAML(ctx, c)
	case *DefaultCollapsibleSection:
		return y.renderCollapsibleSectionYAML(ctx, c)
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
func (y *yamlRenderer) renderContentTo(ctx context.Context, content Content, w io.Writer) error {
	switch c := content.(type) {
	case *TableContent:
		return y.renderTableContentYAMLStream(c, w)
	case *TextContent:
		return y.renderTextContentYAMLStream(c, w)
	case *RawContent:
		return y.renderRawContentYAMLStream(c, w)
	case *SectionContent:
		return y.renderSectionContentYAMLStream(ctx, c, w)
	case *ChartContent, *GraphContent, *DrawIOContent:
		// These complex types fall back to buffered rendering
		data, err := y.renderContent(ctx, content)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	default:
		// Fallback to buffered rendering
		data, err := y.renderContent(ctx, content)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
}

// renderTableContentYAML renders table content as YAML with key order preservation
func (y *yamlRenderer) renderTableContentYAML(table *TableContent) ([]byte, error) {
	return yaml.Marshal(y.buildTableContentYAMLNode(table))
}

// buildTableContentYAMLNode builds the yaml.Node structure for table content,
// preserving user-specified key order in each record mapping.
func (y *yamlRenderer) buildTableContentYAMLNode(table *TableContent) *yaml.Node {
	result := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	// Add title if present
	if table.Title() != "" {
		result.Content = append(result.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: keyTitle},
			&yaml.Node{Kind: yaml.ScalarNode, Value: table.Title()},
		)
	}

	// Create schema node with key order preservation
	schemaNode := &yaml.Node{Kind: yaml.MappingNode}

	// Add keys array
	keyOrder := table.getSchema().GetKeyOrder()
	keysArrayNode := &yaml.Node{Kind: yaml.SequenceNode}
	for _, key := range keyOrder {
		keysArrayNode.Content = append(keysArrayNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		)
	}

	schemaNode.Content = append(schemaNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: keyKeys},
		keysArrayNode,
	)

	// Add fields array
	fieldsArrayNode := &yaml.Node{Kind: yaml.SequenceNode}
	for _, field := range table.getSchema().Fields {
		fieldNode := &yaml.Node{Kind: yaml.MappingNode}
		fieldNode.Content = append(fieldNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: keyName},
			&yaml.Node{Kind: yaml.ScalarNode, Value: field.Name},
			&yaml.Node{Kind: yaml.ScalarNode, Value: keyType},
			&yaml.Node{Kind: yaml.ScalarNode, Value: field.Type},
			&yaml.Node{Kind: yaml.ScalarNode, Value: keyHidden},
			y.createYAMLValueNode(field.Hidden),
		)
		fieldsArrayNode.Content = append(fieldsArrayNode.Content, fieldNode)
	}

	schemaNode.Content = append(schemaNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: keyFields},
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
				field := table.getSchema().FindField(key)
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
		&yaml.Node{Kind: yaml.ScalarNode, Value: keyData},
		dataArrayNode,
	)

	return result
}

// renderTextContentYAML renders text content as YAML
func (y *yamlRenderer) renderTextContentYAML(text *TextContent) ([]byte, error) {
	result := buildTextContentData(text)
	return yaml.Marshal(result)
}

// renderRawContentYAML renders raw content as YAML
func (y *yamlRenderer) renderRawContentYAML(raw *RawContent) ([]byte, error) {
	result := buildRawContentData(raw)
	return yaml.Marshal(result)
}

// renderSectionContentYAML renders section content as YAML with nested content
func (y *yamlRenderer) renderSectionContentYAML(ctx context.Context, section *SectionContent) ([]byte, error) {
	result := map[string]any{
		keyType:  contentTypeNameSection,
		keyTitle: section.Title(),
		keyLevel: section.Level(),
	}

	var contents []any
	for _, content := range section.Contents() {
		// Apply per-content transformations before rendering
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}

		contentYAML, err := y.renderContent(ctx, transformed)
		if err != nil {
			return nil, fmt.Errorf("failed to render nested content: %w", err)
		}

		// Re-parse as yaml.Node so nested key order survives re-marshaling
		contentData, err := yamlContentNode(contentYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to parse content YAML: %w", err)
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
			keySummary:  cv.Summary(), // Requirement 5.1: YAML mapping
			keyDetails:  cv.Details(), // Requirement 5.1: with these fields
			keyExpanded: cv.IsExpanded(),
		}

		// YAML-specific formatting hints (Requirement 5.2)
		if hints := cv.FormatHint(FormatYAML); hints != nil {
			maps.Copy(result, hints)
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

	return encoder.Encode(y.buildTableContentYAMLNode(table))
}

// renderTextContentYAMLStream renders text content as YAML to writer
func (y *yamlRenderer) renderTextContentYAMLStream(text *TextContent, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func() {
		_ = encoder.Close()
	}()

	return encoder.Encode(buildTextContentData(text))
}

// renderRawContentYAMLStream renders raw content as YAML to writer
func (y *yamlRenderer) renderRawContentYAMLStream(raw *RawContent, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func() {
		_ = encoder.Close()
	}()

	return encoder.Encode(buildRawContentData(raw))
}

// renderSectionContentYAMLStream renders section content as YAML to writer
func (y *yamlRenderer) renderSectionContentYAMLStream(ctx context.Context, section *SectionContent, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func() {
		_ = encoder.Close()
	}()

	result := map[string]any{
		keyType:  contentTypeNameSection,
		keyTitle: section.Title(),
		keyLevel: section.Level(),
	}

	var contents []any
	for _, content := range section.Contents() {
		// Apply per-content transformations before rendering
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return err
		}

		contentYAML, err := y.renderContent(ctx, transformed)
		if err != nil {
			return fmt.Errorf("failed to render nested content: %w", err)
		}

		// Re-parse as yaml.Node so nested key order survives re-marshaling
		contentData, err := yamlContentNode(contentYAML)
		if err != nil {
			return fmt.Errorf("failed to parse content YAML: %w", err)
		}
		contents = append(contents, contentData)
	}

	result["contents"] = contents

	return encoder.Encode(result)
}

// renderChartContentYAML renders ChartContent as YAML
func (y *yamlRenderer) renderChartContentYAML(content *ChartContent) ([]byte, error) {
	chartData := map[string]any{
		keyType:      content.Type(),
		keyTitle:     content.GetTitle(),
		"chart_type": content.GetChartType(),
		keyData:      content.GetData(),
	}
	return yaml.Marshal(chartData)
}

// renderGraphContentYAML renders GraphContent as YAML
func (y *yamlRenderer) renderGraphContentYAML(content *GraphContent) ([]byte, error) {
	graphData := map[string]any{
		keyType:  content.Type(),
		keyTitle: content.GetTitle(),
		"nodes":  content.GetNodes(),
		"edges":  content.GetEdges(),
	}
	return yaml.Marshal(graphData)
}

// renderDrawIOContentYAML renders DrawIOContent as YAML
func (y *yamlRenderer) renderDrawIOContentYAML(content *DrawIOContent) ([]byte, error) {
	drawioData := map[string]any{
		keyType:   content.Type(),
		keyTitle:  content.GetTitle(),
		"records": content.GetRecords(),
		keyHeader: content.GetHeader(),
	}
	return yaml.Marshal(drawioData)
}

// renderCollapsibleSectionJSON renders a CollapsibleSection as structured JSON (Requirement 15.5)
func (j *jsonRenderer) renderCollapsibleSectionJSON(ctx context.Context, section *DefaultCollapsibleSection) ([]byte, error) {
	result := map[string]any{
		keyType:     "collapsible_section", // Requirement 15.5: type indication
		keyTitle:    section.Title(),       // Requirement 15.5: section metadata
		keyLevel:    section.Level(),       // Requirement 15.5: section metadata
		keyExpanded: section.IsExpanded(),  // Requirement 15.5: section metadata
	}

	// Render nested content (Requirement 15.5: nested content)
	var contentArray []any
	for _, content := range section.Content() {
		// Skip nil entries defensively: NewCollapsibleSection filters them,
		// but a malformed section must degrade gracefully instead of
		// panicking the public render path (T-1472).
		if content == nil {
			continue
		}

		// Apply per-content transformations before rendering so nested
		// content observes the caller's context (cancellation/deadlines).
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}

		contentJSON, err := j.renderContent(ctx, transformed)
		if err != nil {
			return nil, fmt.Errorf("failed to render section content: %w", err)
		}

		// Embed the rendered JSON verbatim to keep nested key order intact
		contentArray = append(contentArray, json.RawMessage(contentJSON))
	}

	result[keyContent] = contentArray // Requirement 15.5: nested content array

	// Add format-specific hints (Requirement 15.5)
	if hints := section.FormatHint(FormatJSON); hints != nil {
		maps.Copy(result, hints)
	}

	return json.MarshalIndent(result, "", "  ")
}

// renderCollapsibleSectionYAML renders a CollapsibleSection as structured YAML (Requirement 15.5)
func (y *yamlRenderer) renderCollapsibleSectionYAML(ctx context.Context, section *DefaultCollapsibleSection) ([]byte, error) {
	result := map[string]any{
		keyType:     "collapsible_section", // Requirement 15.5: type indication
		keyTitle:    section.Title(),       // Requirement 15.5: section metadata
		keyLevel:    section.Level(),       // Requirement 15.5: section metadata
		keyExpanded: section.IsExpanded(),  // Requirement 15.5: section metadata
	}

	// Render nested content as YAML structures (Requirement 15.5: nested content)
	var contentArray []any
	for _, content := range section.Content() {
		// Skip nil entries defensively: NewCollapsibleSection filters them,
		// but a malformed section must degrade gracefully instead of
		// panicking the public render path (T-1472).
		if content == nil {
			continue
		}

		// Apply per-content transformations before rendering so nested
		// content observes the caller's context (cancellation/deadlines).
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}

		contentYAML, err := y.renderContent(ctx, transformed)
		if err != nil {
			return nil, fmt.Errorf("failed to render section content: %w", err)
		}

		// Re-parse as yaml.Node so nested key order survives re-marshaling
		contentData, err := yamlContentNode(contentYAML)
		if err != nil {
			return nil, fmt.Errorf("failed to parse section content: %w", err)
		}
		contentArray = append(contentArray, contentData)
	}

	result[keyContent] = contentArray // Requirement 15.5: nested content array

	// Add format-specific hints for YAML structure (Requirement 15.5)
	if hints := section.FormatHint(FormatYAML); hints != nil {
		maps.Copy(result, hints)
	}

	return yaml.Marshal(result)
}
