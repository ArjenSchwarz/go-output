package output

import (
	"fmt"
	"strings"
)

// GraphContent represents graph/diagram data
type GraphContent struct {
	id    string
	title string
	edges []Edge
}

// Edge represents a connection between two nodes
type Edge struct {
	From  string
	To    string
	Label string
}

// NewGraphContent creates a new graph content
func NewGraphContent(title string, edges []Edge) *GraphContent {
	return &GraphContent{
		id:    GenerateID(),
		title: title,
		edges: edges,
	}
}

// NewGraphContentFromTable extracts graph data from table content using from/to columns
func NewGraphContentFromTable(table *TableContent, fromColumn, toColumn string, labelColumn string) (*GraphContent, error) {
	if table == nil {
		return nil, fmt.Errorf("table content cannot be nil")
	}

	graph := &GraphContent{
		id:    GenerateID(),
		title: table.title,
		edges: make([]Edge, 0, len(table.records)),
	}

	// Extract edges from table records
	for _, record := range table.records {
		edge := Edge{}

		// Get From value
		if fromVal, ok := record[fromColumn]; ok {
			edge.From = fmt.Sprint(fromVal)
		} else {
			continue // Skip records without from value
		}

		// Get To value
		if toVal, ok := record[toColumn]; ok {
			edge.To = fmt.Sprint(toVal)
		} else {
			continue // Skip records without to value
		}

		// Get optional Label value
		if labelColumn != "" {
			if labelVal, ok := record[labelColumn]; ok {
				edge.Label = fmt.Sprint(labelVal)
			}
		}

		graph.edges = append(graph.edges, edge)
	}

	return graph, nil
}

// Type returns the content type
func (g *GraphContent) Type() ContentType {
	// For now, treat as raw content type since graphs are format-specific
	return ContentTypeRaw
}

// ID returns the unique identifier
func (g *GraphContent) ID() string {
	return g.id
}

// AppendText implements encoding.TextAppender
func (g *GraphContent) AppendText(b []byte) ([]byte, error) {
	// Simple text representation
	if g.title != "" {
		b = append(b, g.title...)
		b = append(b, '\n')
	}

	for _, edge := range g.edges {
		b = append(b, edge.From...)
		b = append(b, " -> "...)
		b = append(b, edge.To...)
		if edge.Label != "" {
			b = append(b, " ["...)
			b = append(b, edge.Label...)
			b = append(b, ']')
		}
		b = append(b, '\n')
	}

	return b, nil
}

// AppendBinary implements encoding.BinaryAppender
func (g *GraphContent) AppendBinary(b []byte) ([]byte, error) {
	// For now, just use text representation
	return g.AppendText(b)
}

// GetEdges returns the graph edges
func (g *GraphContent) GetEdges() []Edge {
	return g.edges
}

// GetTitle returns the graph title
func (g *GraphContent) GetTitle() string {
	return g.title
}

// GetNodes returns unique nodes from all edges
func (g *GraphContent) GetNodes() []string {
	nodeMap := make(map[string]bool)
	for _, edge := range g.edges {
		nodeMap[edge.From] = true
		nodeMap[edge.To] = true
	}

	nodes := make([]string, 0, len(nodeMap))
	for node := range nodeMap {
		nodes = append(nodes, node)
	}
	return nodes
}

// ChartContent represents specialized chart data for Gantt, pie charts, etc.
type ChartContent struct {
	id        string
	title     string
	chartType string // "gantt", "pie", "flowchart"
	data      any    // Chart-specific data structure
}

// ChartType constants for different chart types
const (
	ChartTypeGantt     = "gantt"
	ChartTypePie       = "pie"
	ChartTypeFlowchart = "flowchart"
)

// GanttTask represents a task in a Gantt chart
type GanttTask struct {
	ID           string
	Title        string
	StartDate    string
	EndDate      string
	Duration     string
	Dependencies []string
	Status       string // "active", "done", "crit"
	Section      string // Section name for grouping
}

// PieSlice represents a slice in a pie chart
type PieSlice struct {
	Label string
	Value float64
}

// GanttData represents data for a Gantt chart
type GanttData struct {
	DateFormat string
	AxisFormat string
	Tasks      []GanttTask
}

// PieData represents data for a pie chart
type PieData struct {
	ShowData bool
	Slices   []PieSlice
}

// NewChartContent creates a new chart content
func NewChartContent(title, chartType string, data any) *ChartContent {
	return &ChartContent{
		id:        GenerateID(),
		title:     title,
		chartType: chartType,
		data:      data,
	}
}

// NewGanttChart creates a new Gantt chart content
func NewGanttChart(title string, tasks []GanttTask) *ChartContent {
	data := &GanttData{
		DateFormat: "YYYY-MM-DD",
		AxisFormat: "%Y-%m-%d",
		Tasks:      tasks,
	}
	return NewChartContent(title, ChartTypeGantt, data)
}

// NewPieChart creates a new pie chart content
func NewPieChart(title string, slices []PieSlice, showData bool) *ChartContent {
	data := &PieData{
		ShowData: showData,
		Slices:   slices,
	}
	return NewChartContent(title, ChartTypePie, data)
}

// Type returns the content type
func (c *ChartContent) Type() ContentType {
	return ContentTypeRaw // Charts are format-specific
}

// ID returns the unique identifier
func (c *ChartContent) ID() string {
	return c.id
}

// GetChartType returns the chart type
func (c *ChartContent) GetChartType() string {
	return c.chartType
}

// GetTitle returns the chart title
func (c *ChartContent) GetTitle() string {
	return c.title
}

// GetData returns the chart data
func (c *ChartContent) GetData() any {
	return c.data
}

// AppendText implements encoding.TextAppender
func (c *ChartContent) AppendText(b []byte) ([]byte, error) {
	// Simple text representation
	if c.title != "" {
		b = append(b, c.title...)
		b = append(b, '\n')
	}

	b = append(b, "Chart Type: "...)
	b = append(b, c.chartType...)
	b = append(b, '\n')

	switch c.chartType {
	case ChartTypeGantt:
		if ganttData, ok := c.data.(*GanttData); ok {
			for _, task := range ganttData.Tasks {
				b = append(b, task.Title...)
				b = append(b, " ("...)
				b = append(b, task.StartDate...)
				if task.Duration != "" {
					b = append(b, ", "...)
					b = append(b, task.Duration...)
				}
				b = append(b, ")\n"...)
			}
		}
	case ChartTypePie:
		if pieData, ok := c.data.(*PieData); ok {
			for _, slice := range pieData.Slices {
				b = append(b, slice.Label...)
				b = append(b, ": "...)
				b = append(b, fmt.Sprintf("%.2f", slice.Value)...)
				b = append(b, '\n')
			}
		}
	}

	return b, nil
}

// AppendBinary implements encoding.BinaryAppender
func (c *ChartContent) AppendBinary(b []byte) ([]byte, error) {
	// For now, just use text representation
	return c.AppendText(b)
}

// DrawIOContent represents Draw.io diagram data for CSV export
type DrawIOContent struct {
	id      string
	title   string
	header  DrawIOHeader
	records []Record
}

// DrawIOHeader configures Draw.io CSV import behavior (v1 compatibility)
type DrawIOHeader struct {
	Label        string             // Node label with placeholders (%Name%)
	Style        string             // Node style with placeholders (%Image%)
	Ignore       string             // Columns to ignore in metadata
	Connections  []DrawIOConnection // Connection definitions
	Link         string             // Link column
	Layout       string             // Layout type (auto, horizontalflow, etc.)
	NodeSpacing  int                // Spacing between nodes
	LevelSpacing int                // Spacing between levels
	EdgeSpacing  int                // Spacing between edges
	Parent       string             // Parent column for hierarchical diagrams
	ParentStyle  string             // Style for parent nodes
	Height       string             // Node height
	Width        string             // Node width
	Padding      int                // Padding when auto-sizing
	Left         string             // X coordinate column
	Top          string             // Y coordinate column
	Identity     string             // Identity column
	Namespace    string             // Namespace prefix
}

// DrawIOConnection defines relationships between nodes
type DrawIOConnection struct {
	From   string // Source column
	To     string // Target column
	Invert bool   // Invert direction
	Label  string // Connection label
	Style  string // Connection style (curved, straight, etc.)
}

// NewDrawIOContent creates a new Draw.io content
func NewDrawIOContent(title string, records []Record, header DrawIOHeader) *DrawIOContent {
	return &DrawIOContent{
		id:      GenerateID(),
		title:   title,
		header:  header,
		records: records,
	}
}

// NewDrawIOContentFromTable creates Draw.io content from table data
func NewDrawIOContentFromTable(table *TableContent, header DrawIOHeader) *DrawIOContent {
	return &DrawIOContent{
		id:      GenerateID(),
		title:   table.title,
		header:  header,
		records: table.records,
	}
}

// Type returns the content type
func (d *DrawIOContent) Type() ContentType {
	return ContentTypeRaw // Draw.io CSV is format-specific
}

// ID returns the unique identifier
func (d *DrawIOContent) ID() string {
	return d.id
}

// GetTitle returns the Draw.io content title
func (d *DrawIOContent) GetTitle() string {
	return d.title
}

// GetHeader returns the Draw.io header configuration
func (d *DrawIOContent) GetHeader() DrawIOHeader {
	return d.header
}

// GetRecords returns the data records
func (d *DrawIOContent) GetRecords() []Record {
	return d.records
}

// AppendText implements encoding.TextAppender
func (d *DrawIOContent) AppendText(b []byte) ([]byte, error) {
	// Simple text representation
	if d.title != "" {
		b = append(b, d.title...)
		b = append(b, '\n')
	}

	b = append(b, "Draw.io CSV with "...)
	b = append(b, fmt.Sprintf("%d records", len(d.records))...)
	b = append(b, '\n')

	if d.header.Layout != "" {
		b = append(b, "Layout: "...)
		b = append(b, d.header.Layout...)
		b = append(b, '\n')
	}

	return b, nil
}

// AppendBinary implements encoding.BinaryAppender
func (d *DrawIOContent) AppendBinary(b []byte) ([]byte, error) {
	// For now, just use text representation
	return d.AppendText(b)
}

// DefaultDrawIOHeader returns a header with default Draw.io settings
func DefaultDrawIOHeader() DrawIOHeader {
	return DrawIOHeader{
		Label:        "%Name%",
		Style:        "%Image%",
		Ignore:       "Image",
		Layout:       "auto",
		NodeSpacing:  40,
		LevelSpacing: 100,
		EdgeSpacing:  40,
		Height:       "auto",
		Width:        "auto",
		Namespace:    "csvimport-",
	}
}

// Layout constants for Draw.io
const (
	DrawIOLayoutAuto           = "auto"
	DrawIOLayoutNone           = "none"
	DrawIOLayoutHorizontalFlow = "horizontalflow"
	DrawIOLayoutVerticalFlow   = "verticalflow"
	DrawIOLayoutHorizontalTree = "horizontaltree"
	DrawIOLayoutVerticalTree   = "verticaltree"
	DrawIOLayoutOrganic        = "organic"
	DrawIOLayoutCircle         = "circle"
)

// Connection style constants
const (
	DrawIODefaultConnectionStyle       = "curved=1;endArrow=blockThin;endFill=1;fontSize=11;"
	DrawIOBidirectionalConnectionStyle = "curved=1;endArrow=blockThin;endFill=1;fontSize=11;startArrow=blockThin;startFill=1;"
	DrawIODefaultParentStyle           = "swimlane;whiteSpace=wrap;html=1;childLayout=stackLayout;horizontal=1;horizontalStack=0;resizeParent=1;resizeLast=0;collapsible=1;"
)

// sanitizeDOTID makes a string safe for use as a DOT identifier
func sanitizeDOTID(s string) string {
	// If string contains special characters, quote it
	if strings.ContainsAny(s, " -:;,.'\"\\/<>()[]{}!@#$%^&*+=|?~`") {
		// Escape quotes in the string
		s = strings.ReplaceAll(s, `"`, `\"`)
		return `"` + s + `"`
	}
	return s
}
