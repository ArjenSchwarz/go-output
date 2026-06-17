package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Capitalised column-header tokens reused across the graph renderers.
const (
	colNameFrom        = "From"
	colNameTo          = "To"
	colNameLabel       = "Label"
	colNameName        = "Name"
	colNameDescription = "Description"
)

// Common column-name candidates used when extracting graph data from tables.
// Defined once and shared by the DOT and Mermaid renderers to avoid repeating
// the same string literals across functions.
var (
	graphFromColumns  = []string{"from", colNameFrom, "source", "Source", "start", "Start"}
	graphToColumns    = []string{"to", colNameTo, "target", "Target", "end", "End", "dest", "Dest"}
	graphLabelColumns = []string{"label", colNameLabel, "name", colNameName, "description", colNameDescription}

	// Draw.io column-detection candidates extend the common candidates with a
	// few additional names recognised by the Draw.io renderer.
	drawioFromColumns  = append(append([]string{}, graphFromColumns...), "parent", "Parent")
	drawioToColumns    = append(append([]string{}, graphToColumns...), "name", colNameName)
	drawioLabelColumns = []string{"label", colNameLabel, "description", colNameDescription, "title", "Title"}

	// drawioCSVColumnNames is the header row written for Draw.io CSV output.
	drawioCSVColumnNames = []string{colNameName, colNameFrom, colNameTo, colNameLabel}
)

// dotRenderer implements DOT (Graphviz) output format
type dotRenderer struct {
	baseRenderer
}

func (d *dotRenderer) Format() string {
	return FormatDOT
}

func (d *dotRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	var buf bytes.Buffer

	// Write DOT header
	buf.WriteString("digraph {\n")

	// Process each content item
	contents := doc.GetContents()
	for _, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply per-content transformations (filter/sort/limit) before
		// extracting graph data, so DOT output matches the other formats (T-1091).
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}

		// Handle different content types
		switch c := transformed.(type) {
		case *GraphContent:
			d.renderGraphContent(&buf, c)
		case *TableContent:
			// Try to extract graph from table if it has from/to columns
			if graph := d.extractGraphFromTable(c); graph != nil {
				d.renderGraphContent(&buf, graph)
			}
		}
	}

	// Write DOT footer
	buf.WriteString("}\n")

	return buf.Bytes(), nil
}

func (d *dotRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	data, err := d.Render(ctx, doc)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (d *dotRenderer) SupportsStreaming() bool {
	return false
}

// renderGraphContent renders a GraphContent as DOT format
func (d *dotRenderer) renderGraphContent(buf *bytes.Buffer, graph *GraphContent) {
	// Add title as graph label if present
	if title := graph.GetTitle(); title != "" {
		// Always quote labels in DOT format, escaping special characters so the
		// title cannot break out of the quoted string (T-1292).
		fmt.Fprintf(buf, "  label=\"%s\";\n", escapeDOTLabel(title))
	}

	// Render edges
	for _, edge := range graph.GetEdges() {
		buf.WriteString("  ")
		buf.WriteString(sanitizeDOTID(edge.From))
		buf.WriteString(" -> ")
		buf.WriteString(sanitizeDOTID(edge.To))

		if edge.Label != "" {
			// Always quote edge labels, escaping special characters (T-1292).
			fmt.Fprintf(buf, " [label=\"%s\"]", escapeDOTLabel(edge.Label))
		}

		buf.WriteString(";\n")
	}
}

// extractGraphFromTable attempts to extract graph data from a table
func (d *dotRenderer) extractGraphFromTable(table *TableContent) *GraphContent {
	// Look for common from/to column names
	fromColumns := graphFromColumns
	toColumns := graphToColumns
	labelColumns := graphLabelColumns

	var fromCol, toCol, labelCol string

	// Find from column
	for _, col := range fromColumns {
		if table.schema.HasField(col) {
			fromCol = col
			break
		}
	}

	// Find to column
	for _, col := range toColumns {
		if table.schema.HasField(col) {
			toCol = col
			break
		}
	}

	// Find label column (optional)
	for _, col := range labelColumns {
		if table.schema.HasField(col) {
			labelCol = col
			break
		}
	}

	// If we found both from and to columns, extract graph
	if fromCol != "" && toCol != "" {
		graph, _ := NewGraphContentFromTable(table, fromCol, toCol, labelCol)
		return graph
	}

	return nil
}

// mermaidRenderer implements Mermaid diagram output format
type mermaidRenderer struct {
	baseRenderer
}

func (m *mermaidRenderer) Format() string {
	return FormatMermaid
}

func (m *mermaidRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	var buf bytes.Buffer

	// Process each content item. Apply per-content transformations
	// (filter/sort/limit) up front so both the detection pass and the render
	// pass operate on the transformed content, matching the other formats (T-1091).
	rawContents := doc.GetContents()
	contents := make([]Content, len(rawContents))
	for i, content := range rawContents {
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}
		contents[i] = transformed
	}

	hasFlowchart := false
	specializedCharts := []Content{}
	hasGraphContent := false

	// First pass: determine if we need flowchart header or have specialized charts
	for _, content := range contents {
		switch c := content.(type) {
		case *ChartContent:
			hasGraphContent = true
			// Specialized charts handle their own headers
			switch c.GetChartType() {
			case ChartTypeGantt:
				specializedCharts = append(specializedCharts, c)
			case ChartTypePie:
				specializedCharts = append(specializedCharts, c)
			case ChartTypeFlowchart:
				hasFlowchart = true
			}
		case *GraphContent:
			hasGraphContent = true
			hasFlowchart = true
		case *TableContent:
			hasGraphContent = true
			// Always check tables for potential graph content
			hasFlowchart = true
		}
	}

	// If no graph-related content, fall back to base renderer
	if !hasGraphContent {
		return m.renderDocumentWithFormat(ctx, doc, func(content Content) ([]byte, error) {
			return m.renderContent(content)
		}, FormatMermaid)
	}

	// Render specialized charts first
	for _, chart := range specializedCharts {
		c := chart.(*ChartContent)
		switch c.GetChartType() {
		case ChartTypeGantt:
			m.renderGanttChart(&buf, c)
		case ChartTypePie:
			m.renderPieChart(&buf, c)
		}
	}

	// Handle flowchart content if present
	if hasFlowchart {
		buf.WriteString("graph TD\n")

		for _, content := range contents {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			// Handle different content types for flowcharts
			switch c := content.(type) {
			case *ChartContent:
				if c.GetChartType() == ChartTypeFlowchart {
					m.renderFlowchartContent(&buf, c)
				}
			case *GraphContent:
				m.renderGraphContent(&buf, c)
			case *TableContent:
				// Try to extract graph from table if it has from/to columns
				if graph := m.extractGraphFromTable(c); graph != nil {
					m.renderGraphContent(&buf, graph)
				}
			}
		}
	}

	return buf.Bytes(), nil
}

func (m *mermaidRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	data, err := m.Render(ctx, doc)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (m *mermaidRenderer) SupportsStreaming() bool {
	return false
}

// renderGraphContent renders a GraphContent as Mermaid format
func (m *mermaidRenderer) renderGraphContent(buf *bytes.Buffer, graph *GraphContent) {
	// Add title as a comment if present
	if title := graph.GetTitle(); title != "" {
		fmt.Fprintf(buf, "  %% %s\n", title)
	}

	// Render edges
	for _, edge := range graph.GetEdges() {
		buf.WriteString("  ")
		buf.WriteString(sanitizeMermaidID(edge.From))

		if edge.Label != "" {
			if mermaidLabelNeedsQuoting(edge.Label) {
				// Wrap the edge label in quotes and escape special characters so
				// pipes, quotes, and newlines cannot break the edge syntax (T-1292).
				fmt.Fprintf(buf, " -->|\"%s\"| ", escapeMermaidLabel(edge.Label))
			} else {
				fmt.Fprintf(buf, " -->|%s| ", edge.Label)
			}
		} else {
			buf.WriteString(" --> ")
		}

		buf.WriteString(sanitizeMermaidID(edge.To))
		buf.WriteString("\n")
	}
}

// renderGanttChart renders a ChartContent as Mermaid Gantt chart
func (m *mermaidRenderer) renderGanttChart(buf *bytes.Buffer, chart *ChartContent) {
	ganttData, ok := chart.GetData().(*GanttData)
	if !ok {
		return
	}

	// Write Gantt header
	buf.WriteString("gantt\n")

	// Add title if present
	if title := chart.GetTitle(); title != "" {
		fmt.Fprintf(buf, "    title %s\n", title)
	}

	// Add date format
	fmt.Fprintf(buf, "    dateFormat %s\n", ganttData.DateFormat)

	// Add axis format if different from default
	if ganttData.AxisFormat != "" && ganttData.AxisFormat != "%Y-%m-%d" {
		fmt.Fprintf(buf, "    axisFormat %s\n", ganttData.AxisFormat)
	}

	// Group tasks by section, tracking first-seen section order separately so
	// rendering is deterministic. Ranging the section map directly would expose
	// Go's randomized map iteration order, reordering section blocks even when
	// the input task order is stable.
	sections := make(map[string][]GanttTask)
	sectionOrder := make([]string, 0, len(ganttData.Tasks))
	defaultSection := "Tasks"

	for _, task := range ganttData.Tasks {
		section := task.Section
		if section == "" {
			section = defaultSection
		}
		if _, exists := sections[section]; !exists {
			sectionOrder = append(sectionOrder, section)
		}
		sections[section] = append(sections[section], task)
	}

	// Render sections and tasks in first-seen order
	for _, sectionName := range sectionOrder {
		tasks := sections[sectionName]
		if sectionName != defaultSection || len(sections) > 1 {
			fmt.Fprintf(buf, "    section %s\n", sectionName)
		}

		for _, task := range tasks {
			buf.WriteString("    ")
			buf.WriteString(task.Title)
			buf.WriteString(" :")

			// Add status if present
			if task.Status != "" {
				buf.WriteString(task.Status)
				buf.WriteString(", ")
			}

			// Add task ID if present
			if task.ID != "" {
				buf.WriteString(task.ID)
				buf.WriteString(", ")
			}

			// Add start date
			buf.WriteString(task.StartDate)

			// Add duration or end date
			if task.Duration != "" {
				buf.WriteString(", ")
				buf.WriteString(task.Duration)
			} else if task.EndDate != "" {
				buf.WriteString(", ")
				buf.WriteString(task.EndDate)
			}

			buf.WriteString("\n")
		}
	}
}

// renderPieChart renders a ChartContent as Mermaid pie chart
func (m *mermaidRenderer) renderPieChart(buf *bytes.Buffer, chart *ChartContent) {
	pieData, ok := chart.GetData().(*PieData)
	if !ok {
		return
	}

	// Write pie header
	buf.WriteString("pie")
	if pieData.ShowData {
		buf.WriteString(" showData")
	}
	buf.WriteString("\n")

	// Add title if present
	if title := chart.GetTitle(); title != "" {
		fmt.Fprintf(buf, "    title %s\n", title)
	}

	// Render slices
	for _, slice := range pieData.Slices {
		fmt.Fprintf(buf, "    \"%s\" : %.2f\n", slice.Label, slice.Value)
	}
}

// renderFlowchartContent renders a ChartContent as flowchart (uses GraphContent structure)
func (m *mermaidRenderer) renderFlowchartContent(buf *bytes.Buffer, chart *ChartContent) {
	// For flowchart type, expect the data to be compatible with GraphContent
	// This is for future extensibility when flowchart-specific data structures are needed
	if title := chart.GetTitle(); title != "" {
		fmt.Fprintf(buf, "  %% %s\n", title)
	}
}

// extractGraphFromTable attempts to extract graph data from a table
func (m *mermaidRenderer) extractGraphFromTable(table *TableContent) *GraphContent {
	// Look for common from/to column names
	fromColumns := graphFromColumns
	toColumns := graphToColumns
	labelColumns := graphLabelColumns

	var fromCol, toCol, labelCol string

	// Find from column
	for _, col := range fromColumns {
		if table.schema.HasField(col) {
			fromCol = col
			break
		}
	}

	// Find to column
	for _, col := range toColumns {
		if table.schema.HasField(col) {
			toCol = col
			break
		}
	}

	// Find label column (optional)
	for _, col := range labelColumns {
		if table.schema.HasField(col) {
			labelCol = col
			break
		}
	}

	// If we found both from and to columns, extract graph
	if fromCol != "" && toCol != "" {
		graph, _ := NewGraphContentFromTable(table, fromCol, toCol, labelCol)
		return graph
	}

	return nil
}

// sanitizeMermaidID makes a string safe for use as a Mermaid identifier
func sanitizeMermaidID(s string) string {
	// Mermaid uses brackets for node text with special characters.
	if containsSpecialChars(s) {
		// Characters that would otherwise break out of the [..] node syntax are
		// escaped and the text is wrapped in double quotes (T-1292). Plain
		// special characters such as spaces or dashes keep the simpler [text]
		// form for readability and backwards compatibility.
		if mermaidLabelNeedsQuoting(s) {
			return `["` + escapeMermaidLabel(s) + `"]`
		}
		return fmt.Sprintf("[%s]", s)
	}
	return s
}

// escapeDOTLabel escapes a string for safe inclusion inside a quoted DOT label.
// DOT string rules require escaping the backslash and double-quote characters;
// literal newlines are converted to the "\n" escape sequence understood by
// Graphviz so labels cannot break out of the label="..." syntax (T-1292).
func escapeDOTLabel(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
	)
	return replacer.Replace(s)
}

// mermaidLabelNeedsQuoting reports whether a label contains characters that
// would break out of Mermaid's node ([..]) or edge (|..|) label syntax and
// therefore require the quoted, escaped form (T-1292).
func mermaidLabelNeedsQuoting(s string) bool {
	return strings.ContainsAny(s, "\"[]|<>\n\r")
}

// escapeMermaidLabel escapes a string for safe inclusion inside a quoted Mermaid
// label. Mermaid recommends wrapping label text in double quotes and encoding
// embedded quotes as the &quot; HTML entity; newlines become <br/> so the label
// cannot break out of its delimiter (T-1292). Callers are responsible for
// wrapping the result in the surrounding quotes.
func escapeMermaidLabel(s string) string {
	replacer := strings.NewReplacer(
		`"`, "&quot;",
		"\n", "<br/>",
		"\r", "",
	)
	return replacer.Replace(s)
}

// containsSpecialChars checks if a string contains characters that need escaping in Mermaid
func containsSpecialChars(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '-' || r == ':' || r == ';' || r == ',' || r == '.' ||
			r == '\'' || r == '"' || r == '\\' || r == '/' || r == '<' || r == '>' ||
			r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' ||
			r == '!' || r == '@' || r == '#' || r == '$' || r == '%' || r == '^' ||
			r == '&' || r == '*' || r == '+' || r == '=' || r == '|' || r == '?' ||
			r == '~' || r == '`' {
			return true
		}
	}
	return false
}

// drawioRenderer implements Draw.io CSV output format (v1 compatibility)
type drawioRenderer struct {
	baseRenderer
}

func (d *drawioRenderer) Format() string {
	return FormatDrawIO
}

func (d *drawioRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	var buf bytes.Buffer

	// Process each content item. Apply per-content transformations
	// (filter/sort/limit) up front so both the compatibility-detection pass and
	// the render pass operate on the transformed content, matching the other
	// formats (T-1091).
	rawContents := doc.GetContents()
	contents := make([]Content, len(rawContents))
	for i, content := range rawContents {
		transformed, err := applyContentTransformations(ctx, content)
		if err != nil {
			return nil, err
		}
		contents[i] = transformed
	}

	hasDrawIOContent := false

	// Check if we have any Draw.io-compatible content
	for _, content := range contents {
		switch content.(type) {
		case *DrawIOContent, *GraphContent, *TableContent:
			hasDrawIOContent = true
		}
	}

	// If no Draw.io-compatible content, fall back to base renderer
	if !hasDrawIOContent {
		return d.renderDocumentWithFormat(ctx, doc, func(content Content) ([]byte, error) {
			return d.renderContent(content)
		}, FormatDrawIO)
	}

	for _, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Handle different content types
		switch c := content.(type) {
		case *DrawIOContent:
			d.renderDrawIOContent(&buf, c)
		case *GraphContent:
			d.renderGraphAsDrawIO(&buf, c)
		case *TableContent:
			d.renderTableAsDrawIO(&buf, c)
		}
	}

	return buf.Bytes(), nil
}

func (d *drawioRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	data, err := d.Render(ctx, doc)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (d *drawioRenderer) SupportsStreaming() bool {
	return false
}

// renderDrawIOContent renders DrawIOContent as CSV format
func (d *drawioRenderer) renderDrawIOContent(buf *bytes.Buffer, content *DrawIOContent) {
	header := content.GetHeader()
	records := content.GetRecords()

	// Generate CSV header based on DrawIOHeader configuration
	d.writeDrawIOHeader(buf, header)

	// Use the explicit column order when set; otherwise fall back to
	// alphabetized auto-detection from the records.
	columnNames := content.GetColumns()
	if len(columnNames) == 0 {
		columnNames = d.extractColumnNames(records)
	}

	// Write CSV header row
	d.writeCSVRow(buf, columnNames)

	// Write data rows
	for _, record := range records {
		row := make([]string, len(columnNames))
		for i, col := range columnNames {
			if val, ok := record[col]; ok {
				row[i] = fmt.Sprint(val)
			}
		}
		d.writeCSVRow(buf, row)
	}
}

// renderGraphAsDrawIO renders GraphContent as Draw.io CSV
func (d *drawioRenderer) renderGraphAsDrawIO(buf *bytes.Buffer, graph *GraphContent) {
	// Create a default header for graph content
	header := DefaultDrawIOHeader()
	header.Label = "%Name%"
	header.Connections = []DrawIOConnection{
		{
			From:  drawioCSVColumnNames[1],
			To:    drawioCSVColumnNames[2],
			Label: drawioCSVColumnNames[3],
			Style: DrawIODefaultConnectionStyle,
		},
	}

	d.writeDrawIOHeader(buf, header)

	// Convert graph edges to CSV records
	d.writeCSVRow(buf, drawioCSVColumnNames)

	// Write nodes first
	nodes := graph.GetNodes()
	for _, node := range nodes {
		row := []string{node, "", "", ""}
		d.writeCSVRow(buf, row)
	}

	// Write edges
	for _, edge := range graph.GetEdges() {
		row := []string{"", edge.From, edge.To, edge.Label}
		d.writeCSVRow(buf, row)
	}
}

// renderTableAsDrawIO renders TableContent as Draw.io CSV
func (d *drawioRenderer) renderTableAsDrawIO(buf *bytes.Buffer, table *TableContent) {
	// Create header with auto-detected connections
	header := DefaultDrawIOHeader()

	// Try to detect from/to columns for connections
	fromCol, toCol, labelCol := d.detectConnectionColumns(table)
	if fromCol != "" && toCol != "" {
		connection := DrawIOConnection{
			From:  fromCol,
			To:    toCol,
			Style: DrawIODefaultConnectionStyle,
		}
		if labelCol != "" {
			connection.Label = labelCol
		}
		header.Connections = []DrawIOConnection{connection}
	}

	d.writeDrawIOHeader(buf, header)

	// Get column names from schema to preserve order
	columnNames := table.schema.GetFieldNames()
	d.writeCSVRow(buf, columnNames)

	// Write data rows
	for _, record := range table.records {
		row := make([]string, len(columnNames))
		for i, col := range columnNames {
			if val, ok := record[col]; ok {
				row[i] = fmt.Sprint(val)
			}
		}
		d.writeCSVRow(buf, row)
	}
}

// writeDrawIOHeader writes the Draw.io CSV header comments. All lines go
// through writeDrawIODirective and the drawioKey* constants so the writer and
// the parser share one definition of the directive grammar.
func (d *drawioRenderer) writeDrawIOHeader(buf *bytes.Buffer, header DrawIOHeader) {
	// Label
	if header.Label != "" {
		writeDrawIODirective(buf, drawioKeyLabel, header.Label)
	}

	// Style
	if header.Style != "" {
		writeDrawIODirective(buf, drawioKeyStyle, header.Style)
	}

	// Identity
	if header.Identity != "" {
		writeDrawIODirective(buf, drawioKeyIdentity, header.Identity)
	}

	// Parent
	if header.Parent != "" {
		writeDrawIODirective(buf, drawioKeyParent, header.Parent)
		if header.ParentStyle != "" {
			writeDrawIODirective(buf, drawioKeyParentStyle, header.ParentStyle)
		}
	}

	// Namespace
	if header.Namespace != "" {
		writeDrawIODirective(buf, drawioKeyNamespace, header.Namespace)
	}

	// Connections: encode through the drawioConnectionJSON mirror struct so
	// values are escaped per JSON rules while &, <, and > stay verbatim
	// (requirement 3.5). Encode's trailing newline terminates the line.
	if len(header.Connections) > 0 {
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		for _, conn := range header.Connections {
			buf.WriteString("# " + drawioKeyConnect + ": ")
			// Encoding a struct of strings and a bool cannot fail.
			_ = enc.Encode(drawioConnectionJSON(conn))
		}
	}

	// Dimensions
	if header.Height != "" {
		writeDrawIODirective(buf, drawioKeyHeight, header.Height)
	}
	if header.Width != "" {
		writeDrawIODirective(buf, drawioKeyWidth, header.Width)
	}

	// Ignore
	if header.Ignore != "" {
		writeDrawIODirective(buf, drawioKeyIgnore, header.Ignore)
	}

	// Spacing
	if header.NodeSpacing > 0 {
		writeDrawIODirective(buf, drawioKeyNodeSpacing, strconv.Itoa(header.NodeSpacing))
	}
	if header.LevelSpacing > 0 {
		writeDrawIODirective(buf, drawioKeyLevelSpacing, strconv.Itoa(header.LevelSpacing))
	}
	if header.EdgeSpacing > 0 {
		writeDrawIODirective(buf, drawioKeyEdgeSpacing, strconv.Itoa(header.EdgeSpacing))
	}

	// Padding
	if header.Padding > 0 {
		writeDrawIODirective(buf, drawioKeyPadding, strconv.Itoa(header.Padding))
	}

	// Link
	if header.Link != "" {
		writeDrawIODirective(buf, drawioKeyLink, header.Link)
	}

	// Position columns (only for layout=none)
	if header.Layout == DrawIOLayoutNone {
		if header.Left != "" {
			writeDrawIODirective(buf, drawioKeyLeft, header.Left)
		}
		if header.Top != "" {
			writeDrawIODirective(buf, drawioKeyTop, header.Top)
		}
	}

	// Layout
	if header.Layout != "" {
		writeDrawIODirective(buf, drawioKeyLayout, header.Layout)
	}
}

// writeCSVRow writes a CSV row with proper escaping
func (d *drawioRenderer) writeCSVRow(buf *bytes.Buffer, row []string) {
	for i, field := range row {
		if i > 0 {
			buf.WriteByte(',')
		}

		// Quote fields containing comma, quote, or newline; fields starting
		// with '#' (which would otherwise look like a comment or directive);
		// and a single empty sole field (which would otherwise render as a
		// blank line that CSV readers silently skip). Requirement 3.6.
		needsQuoting := strings.ContainsAny(field, ",\"\n\r") ||
			strings.HasPrefix(field, "#") ||
			(len(row) == 1 && field == "")
		if needsQuoting {
			escaped := strings.ReplaceAll(field, "\"", "\"\"")
			fmt.Fprintf(buf, "\"%s\"", escaped)
		} else {
			buf.WriteString(field)
		}
	}
	buf.WriteByte('\n')
}

// extractColumnNames extracts unique column names from records
func (d *drawioRenderer) extractColumnNames(records []Record) []string {
	columnSet := make(map[string]bool)

	// Collect all unique column names
	for _, record := range records {
		for key := range record {
			columnSet[key] = true
		}
	}

	// Convert to sorted slice for deterministic output
	columns := make([]string, 0, len(columnSet))
	for key := range columnSet {
		columns = append(columns, key)
	}
	sort.Strings(columns)

	return columns
}

// detectConnectionColumns tries to find from/to columns in table schema
func (d *drawioRenderer) detectConnectionColumns(table *TableContent) (string, string, string) {
	fromColumns := drawioFromColumns
	toColumns := drawioToColumns
	labelColumns := drawioLabelColumns

	var fromCol, toCol, labelCol string

	// Find from column
	for _, col := range fromColumns {
		if table.schema.HasField(col) {
			fromCol = col
			break
		}
	}

	// Find to column
	for _, col := range toColumns {
		if table.schema.HasField(col) {
			toCol = col
			break
		}
	}

	// Find label column (optional)
	for _, col := range labelColumns {
		if table.schema.HasField(col) {
			labelCol = col
			break
		}
	}

	return fromCol, toCol, labelCol
}
