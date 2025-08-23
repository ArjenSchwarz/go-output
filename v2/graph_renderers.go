package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
)

// dotRenderer implements DOT (Graphviz) output format
type dotRenderer struct {
	baseRenderer
}

func (d *dotRenderer) Format() string {
	return "dot"
}

func (d *dotRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
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

		// Handle different content types
		switch c := content.(type) {
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
		// Always quote labels in DOT format
		fmt.Fprintf(buf, "  label=\"%s\";\n", title)
	}

	// Render edges
	for _, edge := range graph.GetEdges() {
		buf.WriteString("  ")
		buf.WriteString(sanitizeDOTID(edge.From))
		buf.WriteString(" -> ")
		buf.WriteString(sanitizeDOTID(edge.To))

		if edge.Label != "" {
			// Always quote edge labels
			fmt.Fprintf(buf, " [label=\"%s\"]", edge.Label)
		}

		buf.WriteString(";\n")
	}
}

// extractGraphFromTable attempts to extract graph data from a table
func (d *dotRenderer) extractGraphFromTable(table *TableContent) *GraphContent {
	// Look for common from/to column names
	fromColumns := []string{"from", "From", "source", "Source", "start", "Start"}
	toColumns := []string{"to", "To", "target", "Target", "end", "End", "dest", "Dest"}
	labelColumns := []string{"label", "Label", "name", "Name", "description", "Description"}

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
	return "mermaid"
}

func (m *mermaidRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	var buf bytes.Buffer

	// Process each content item
	contents := doc.GetContents()
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
			fmt.Fprintf(buf, " -->|%s| ", edge.Label)
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

	// Group tasks by section
	sections := make(map[string][]GanttTask)
	defaultSection := "Tasks"

	for _, task := range ganttData.Tasks {
		section := task.Section
		if section == "" {
			section = defaultSection
		}
		sections[section] = append(sections[section], task)
	}

	// Render sections and tasks
	for sectionName, tasks := range sections {
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
	fromColumns := []string{"from", "From", "source", "Source", "start", "Start"}
	toColumns := []string{"to", "To", "target", "Target", "end", "End", "dest", "Dest"}
	labelColumns := []string{"label", "Label", "name", "Name", "description", "Description"}

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
	// Mermaid uses brackets for node text with special characters
	if containsSpecialChars(s) {
		return fmt.Sprintf("[%s]", s)
	}
	return s
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
	return "drawio"
}

func (d *drawioRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	var buf bytes.Buffer

	// Process each content item
	contents := doc.GetContents()
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

	// Extract column names from records
	columnNames := d.extractColumnNames(records)

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
		{From: "From", To: "To", Label: "Label", Style: DrawIODefaultConnectionStyle},
	}

	d.writeDrawIOHeader(buf, header)

	// Convert graph edges to CSV records
	columnNames := []string{"Name", "From", "To", "Label"}
	d.writeCSVRow(buf, columnNames)

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

// writeDrawIOHeader writes the Draw.io CSV header comments
func (d *drawioRenderer) writeDrawIOHeader(buf *bytes.Buffer, header DrawIOHeader) {
	// Label
	if header.Label != "" {
		fmt.Fprintf(buf, "# label: %s\n", header.Label)
	}

	// Style
	if header.Style != "" {
		fmt.Fprintf(buf, "# style: %s\n", header.Style)
	}

	// Identity
	if header.Identity != "" {
		fmt.Fprintf(buf, "# identity: %s\n", header.Identity)
	}

	// Parent
	if header.Parent != "" {
		fmt.Fprintf(buf, "# parent: %s\n", header.Parent)
		if header.ParentStyle != "" {
			fmt.Fprintf(buf, "# parentstyle: %s\n", header.ParentStyle)
		}
	}

	// Namespace
	if header.Namespace != "" {
		fmt.Fprintf(buf, "# namespace: %s\n", header.Namespace)
	}

	// Connections
	for _, conn := range header.Connections {
		connJSON := fmt.Sprintf(`{"from":"%s","to":"%s","invert":%t,"label":"%s","style":"%s"}`,
			conn.From, conn.To, conn.Invert, conn.Label, conn.Style)
		fmt.Fprintf(buf, "# connect: %s\n", connJSON)
	}

	// Dimensions
	if header.Height != "" {
		fmt.Fprintf(buf, "# height: %s\n", header.Height)
	}
	if header.Width != "" {
		fmt.Fprintf(buf, "# width: %s\n", header.Width)
	}

	// Ignore
	if header.Ignore != "" {
		fmt.Fprintf(buf, "# ignore: %s\n", header.Ignore)
	}

	// Spacing
	if header.NodeSpacing > 0 {
		fmt.Fprintf(buf, "# nodespacing: %d\n", header.NodeSpacing)
	}
	if header.LevelSpacing > 0 {
		fmt.Fprintf(buf, "# levelspacing: %d\n", header.LevelSpacing)
	}
	if header.EdgeSpacing > 0 {
		fmt.Fprintf(buf, "# edgespacing: %d\n", header.EdgeSpacing)
	}

	// Padding
	if header.Padding > 0 {
		fmt.Fprintf(buf, "# padding: %d\n", header.Padding)
	}

	// Link
	if header.Link != "" {
		fmt.Fprintf(buf, "# link: %s\n", header.Link)
	}

	// Position columns (only for layout=none)
	if header.Layout == DrawIOLayoutNone {
		if header.Left != "" {
			fmt.Fprintf(buf, "# left: %s\n", header.Left)
		}
		if header.Top != "" {
			fmt.Fprintf(buf, "# top: %s\n", header.Top)
		}
	}

	// Layout
	if header.Layout != "" {
		fmt.Fprintf(buf, "# layout: %s\n", header.Layout)
	}
}

// writeCSVRow writes a CSV row with proper escaping
func (d *drawioRenderer) writeCSVRow(buf *bytes.Buffer, row []string) {
	for i, field := range row {
		if i > 0 {
			buf.WriteByte(',')
		}

		// Escape field if it contains comma, quote, or newline
		if strings.ContainsAny(field, ",\"\n\r") {
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
	fromColumns := []string{"from", "From", "source", "Source", "start", "Start", "parent", "Parent"}
	toColumns := []string{"to", "To", "target", "Target", "end", "End", "dest", "Dest", "name", "Name"}
	labelColumns := []string{"label", "Label", "description", "Description", "title", "Title"}

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
