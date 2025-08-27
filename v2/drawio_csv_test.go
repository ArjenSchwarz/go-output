package output

import (
	"context"
	"strings"
	"testing"
)

func TestDrawIOContent_Creation(t *testing.T) {
	header := DefaultDrawIOHeader()
	header.Layout = DrawIOLayoutHorizontalFlow
	header.NodeSpacing = 50

	records := []Record{
		{"Name": "Server1", "Type": "Web Server", "Location": "US-East"},
		{"Name": "Server2", "Type": "Database", "Location": "US-West"},
	}

	content := NewDrawIOContent("Infrastructure", records, header)

	if content.GetTitle() != "Infrastructure" {
		t.Errorf("GetTitle() = %v, want Infrastructure", content.GetTitle())
	}

	if content.Type() != ContentTypeRaw {
		t.Errorf("Type() = %v, want %v", content.Type(), ContentTypeRaw)
	}

	if len(content.GetRecords()) != 2 {
		t.Errorf("len(GetRecords()) = %v, want 2", len(content.GetRecords()))
	}

	gotHeader := content.GetHeader()
	if gotHeader.Layout != DrawIOLayoutHorizontalFlow {
		t.Errorf("Header.Layout = %v, want %v", gotHeader.Layout, DrawIOLayoutHorizontalFlow)
	}

	if gotHeader.NodeSpacing != 50 {
		t.Errorf("Header.NodeSpacing = %v, want 50", gotHeader.NodeSpacing)
	}
}

func TestDrawIORenderer_CSVGeneration(t *testing.T) {
	tests := map[string]struct {
		content  Content
		contains []string
	}{"drawio content with connections": {

		content: NewDrawIOContent("Service Map", []Record{
			{"Service": "Frontend", "Port": "80", "Backend": "API"},
			{"Service": "API", "Port": "8080", "Backend": "Database"},
		}, DrawIOHeader{
			Label: "%Service%",
			Style: "rounded=1;whiteSpace=wrap;html=1;",
			Connections: []DrawIOConnection{
				{From: "Service", To: "Backend", Label: "Port", Style: DrawIODefaultConnectionStyle},
			},
			Layout: DrawIOLayoutAuto,
		}),
		contains: []string{
			"# label: %Service%",
			"# style: rounded=1;whiteSpace=wrap;html=1;",
			`# connect: {"from":"Service","to":"Backend","invert":false,"label":"Port","style":"curved=1;endArrow=blockThin;endFill=1;fontSize=11;"}`,
			"# layout: auto",
			"Frontend",
			"80",
			"API",
			"8080",
			"Database",
		},
	}, "drawio content with custom header": {

		content: NewDrawIOContent("Network Diagram", []Record{
			{"Name": "Router", "IP": "192.168.1.1", "Type": "Gateway"},
			{"Name": "Switch", "IP": "192.168.1.2", "Type": "Switch"},
		}, DrawIOHeader{
			Label:        "%Name%",
			Style:        "shape=%Type%;fillColor=lightblue",
			Layout:       DrawIOLayoutVerticalFlow,
			NodeSpacing:  60,
			LevelSpacing: 120,
			EdgeSpacing:  50,
			Height:       "80",
			Width:        "120",
			Ignore:       "Type",
		}),
		contains: []string{
			"# label: %Name%",
			"# style: shape=%Type%;fillColor=lightblue",
			"# height: 80",
			"# width: 120",
			"# ignore: Type",
			"# nodespacing: 60",
			"# levelspacing: 120",
			"# edgespacing: 50",
			"# layout: verticalflow",
			"Router",
			"192.168.1.1",
			"Gateway",
			"Switch",
			"192.168.1.2",
		},
	}, "drawio content with hierarchy": {

		content: NewDrawIOContent("Org Chart", []Record{
			{"Name": "CEO", "Title": "Chief Executive", "Manager": "", "ID": "1"},
			{"Name": "CTO", "Title": "Chief Technology", "Manager": "CEO", "ID": "2"},
			{"Name": "Developer", "Title": "Software Engineer", "Manager": "CTO", "ID": "3"},
		}, DrawIOHeader{
			Label:       "%Name%",
			Style:       "rounded=1;fillColor=lightgreen;",
			Identity:    "ID",
			Parent:      "Manager",
			ParentStyle: DrawIODefaultParentStyle,
			Layout:      DrawIOLayoutVerticalTree,
			Namespace:   "org-chart-",
		}),
		contains: []string{
			"# label: %Name%",
			"# style: rounded=1;fillColor=lightgreen;",
			"# identity: ID",
			"# parent: Manager",
			"# parentstyle: " + DrawIODefaultParentStyle,
			"# namespace: org-chart-",
			"# layout: verticaltree",
			"CEO",
			"Chief Executive",
			"CTO",
			"Chief Technology",
			"Developer",
			"Software Engineer",
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().AddContent(tt.content).Build()
			renderer := &drawioRenderer{}

			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			output := string(result)
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestDrawIORenderer_TableAutoDetection(t *testing.T) {
	// Test automatic connection detection from table data
	doc := New().
		Table("AWS Infrastructure", []map[string]any{
			{"from": "LoadBalancer", "to": "WebServer", "description": "HTTP Traffic", "port": "80"},
			{"from": "WebServer", "to": "Database", "description": "DB Connection", "port": "3306"},
		}).
		Build()

	renderer := &drawioRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Should detect from/to columns and create connections
	expectedParts := []string{
		"# label: %Name%",
		"# connect:",
		`"from":"from"`,
		`"to":"to"`,
		`"label":"description"`, // Should auto-detect description as label
		"LoadBalancer",
		"WebServer",
		"HTTP Traffic",
		"80",
		"Database",
		"DB Connection",
		"3306",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain %q, got:\n%s", expected, output)
		}
	}
}

func TestDrawIORenderer_CSVEscaping(t *testing.T) {
	// Test proper CSV escaping for special characters
	doc := New().
		DrawIO("Test Escaping", []Record{
			{"Name": "Item, with comma", "Description": "Value \"with quotes\"", "Notes": "Line\nbreak"},
			{"Name": "Normal", "Description": "No special chars", "Notes": "Regular text"},
		}, DefaultDrawIOHeader()).
		Build()

	renderer := &drawioRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Check for proper CSV escaping
	expectedEscaping := []string{
		`"Item, with comma"`,      // Comma escaping
		`"Value ""with quotes"""`, // Quote escaping
		`"Line`,                   // Newline escaping start
		`break"`,                  // Newline escaping end
		"Normal",                  // No escaping needed
		"No special chars",        // No escaping needed
		"Regular text",            // No escaping needed
	}

	for _, expected := range expectedEscaping {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain escaped CSV %q, got:\n%s", expected, output)
		}
	}
}

func TestDrawIOBuilder_Integration(t *testing.T) {
	// Test the Builder.DrawIO method
	header := DefaultDrawIOHeader()
	header.Layout = DrawIOLayoutCircle
	header.Connections = []DrawIOConnection{
		{From: "Name", To: "ConnectsTo", Label: "Relationship", Style: DrawIOBidirectionalConnectionStyle},
	}

	doc := New().
		DrawIO("Service Dependencies", []Record{
			{"Name": "Frontend", "ConnectsTo": "API", "Relationship": "calls"},
			{"Name": "API", "ConnectsTo": "Database", "Relationship": "queries"},
		}, header).
		Build()

	contents := doc.GetContents()
	if len(contents) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(contents))
	}

	drawioContent, ok := contents[0].(*DrawIOContent)
	if !ok {
		t.Fatal("Content should be DrawIOContent")
	}

	if drawioContent.GetTitle() != "Service Dependencies" {
		t.Errorf("Title = %v, want Service Dependencies", drawioContent.GetTitle())
	}

	if len(drawioContent.GetRecords()) != 2 {
		t.Errorf("Records count = %v, want 2", len(drawioContent.GetRecords()))
	}

	gotHeader := drawioContent.GetHeader()
	if gotHeader.Layout != DrawIOLayoutCircle {
		t.Errorf("Layout = %v, want %v", gotHeader.Layout, DrawIOLayoutCircle)
	}

	if len(gotHeader.Connections) != 1 {
		t.Fatalf("Connections count = %v, want 1", len(gotHeader.Connections))
	}

	conn := gotHeader.Connections[0]
	if conn.Style != DrawIOBidirectionalConnectionStyle {
		t.Errorf("Connection style = %v, want %v", conn.Style, DrawIOBidirectionalConnectionStyle)
	}
}

func TestDefaultDrawIOHeader(t *testing.T) {
	header := DefaultDrawIOHeader()

	expected := DrawIOHeader{
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

	if header.Label != expected.Label {
		t.Errorf("Label = %v, want %v", header.Label, expected.Label)
	}
	if header.Style != expected.Style {
		t.Errorf("Style = %v, want %v", header.Style, expected.Style)
	}
	if header.Layout != expected.Layout {
		t.Errorf("Layout = %v, want %v", header.Layout, expected.Layout)
	}
	if header.NodeSpacing != expected.NodeSpacing {
		t.Errorf("NodeSpacing = %v, want %v", header.NodeSpacing, expected.NodeSpacing)
	}
}
