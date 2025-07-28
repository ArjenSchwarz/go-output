package output

import (
	"context"
	"strings"
	"testing"
)

func TestDOTRenderer_Render(t *testing.T) {
	tests := []struct {
		name    string
		doc     *Document
		want    string
		wantErr bool
	}{
		{
			name: "simple graph",
			doc: New().
				Graph("Test Graph", []Edge{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
				}).
				Build(),
			want: `digraph {
  label="Test Graph";
  A -> B;
  B -> C;
}
`,
		},
		{
			name: "graph with labels",
			doc: New().
				Graph("Workflow", []Edge{
					{From: "Start", To: "Process", Label: "begin"},
					{From: "Process", To: "End", Label: "complete"},
				}).
				Build(),
			want: `digraph {
  label="Workflow";
  Start -> Process [label="begin"];
  Process -> End [label="complete"];
}
`,
		},
		{
			name: "graph with special characters",
			doc: New().
				Graph("", []Edge{
					{From: "Node A", To: "Node-B"},
					{From: "Node-B", To: "Node:C"},
					{From: "Node:C", To: "Node/D"},
				}).
				Build(),
			want: `digraph {
  "Node A" -> "Node-B";
  "Node-B" -> "Node:C";
  "Node:C" -> "Node/D";
}
`,
		},
		{
			name: "table with from/to columns",
			doc: New().
				Table("Dependencies", []map[string]any{
					{"from": "package-a", "to": "package-b", "label": "depends"},
					{"from": "package-b", "to": "package-c", "label": "requires"},
				}).
				Build(),
			want: `digraph {
  label="Dependencies";
  "package-a" -> "package-b" [label="depends"];
  "package-b" -> "package-c" [label="requires"];
}
`,
		},
		{
			name: "table with Source/Target columns",
			doc: New().
				Table("Network", []map[string]any{
					{"Source": "Server1", "Target": "Server2", "Name": "HTTP"},
					{"Source": "Server2", "Target": "Server3", "Name": "HTTPS"},
				}).
				Build(),
			want: `digraph {
  label="Network";
  Server1 -> Server2 [label="HTTP"];
  Server2 -> Server3 [label="HTTPS"];
}
`,
		},
		{
			name: "multiple graphs",
			doc: New().
				Graph("Graph 1", []Edge{
					{From: "A", To: "B"},
				}).
				Graph("Graph 2", []Edge{
					{From: "X", To: "Y"},
				}).
				Build(),
			want: `digraph {
  label="Graph 1";
  A -> B;
  label="Graph 2";
  X -> Y;
}
`,
		},
		{
			name: "empty graph",
			doc: New().
				Graph("Empty", []Edge{}).
				Build(),
			want: `digraph {
  label="Empty";
}
`,
		},
		{
			name: "table without graph columns",
			doc: New().
				Table("Users", []map[string]any{
					{"name": "Alice", "age": 30},
					{"name": "Bob", "age": 25},
				}).
				Build(),
			want: `digraph {
}
`,
		},
	}

	ctx := context.Background()
	renderer := &dotRenderer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderer.Render(ctx, tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("DOTRenderer.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("DOTRenderer.Render() = %q, want %q", gotStr, tt.want)
			}
		})
	}
}

func TestMermaidRenderer_Render(t *testing.T) {
	tests := []struct {
		name    string
		doc     *Document
		want    string
		wantErr bool
	}{
		{
			name: "simple graph",
			doc: New().
				Graph("Test Graph", []Edge{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
				}).
				Build(),
			want: `graph TD
  % Test Graph
  A --> B
  B --> C
`,
		},
		{
			name: "graph with labels",
			doc: New().
				Graph("Workflow", []Edge{
					{From: "Start", To: "Process", Label: "begin"},
					{From: "Process", To: "End", Label: "complete"},
				}).
				Build(),
			want: `graph TD
  % Workflow
  Start -->|begin| Process
  Process -->|complete| End
`,
		},
		{
			name: "graph with special characters",
			doc: New().
				Graph("", []Edge{
					{From: "Node A", To: "Node-B"},
					{From: "Node-B", To: "Node:C"},
					{From: "Node:C", To: "Node/D"},
				}).
				Build(),
			want: `graph TD
  [Node A] --> [Node-B]
  [Node-B] --> [Node:C]
  [Node:C] --> [Node/D]
`,
		},
		{
			name: "table with from/to columns",
			doc: New().
				Table("Dependencies", []map[string]any{
					{"from": "package-a", "to": "package-b", "label": "depends"},
					{"from": "package-b", "to": "package-c", "label": "requires"},
				}).
				Build(),
			want: `graph TD
  % Dependencies
  [package-a] -->|depends| [package-b]
  [package-b] -->|requires| [package-c]
`,
		},
		{
			name: "table with Source/Target columns",
			doc: New().
				Table("Network", []map[string]any{
					{"Source": "Server1", "Target": "Server2", "Name": "HTTP"},
					{"Source": "Server2", "Target": "Server3", "Name": "HTTPS"},
				}).
				Build(),
			want: `graph TD
  % Network
  Server1 -->|HTTP| Server2
  Server2 -->|HTTPS| Server3
`,
		},
		{
			name: "multiple graphs",
			doc: New().
				Graph("Graph 1", []Edge{
					{From: "A", To: "B"},
				}).
				Graph("Graph 2", []Edge{
					{From: "X", To: "Y"},
				}).
				Build(),
			want: `graph TD
  % Graph 1
  A --> B
  % Graph 2
  X --> Y
`,
		},
		{
			name: "empty graph",
			doc: New().
				Graph("Empty", []Edge{}).
				Build(),
			want: `graph TD
  % Empty
`,
		},
		{
			name: "table without graph columns",
			doc: New().
				Table("Users", []map[string]any{
					{"name": "Alice", "age": 30},
					{"name": "Bob", "age": 25},
				}).
				Build(),
			want: `graph TD
`,
		},
	}

	ctx := context.Background()
	renderer := &mermaidRenderer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderer.Render(ctx, tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("MermaidRenderer.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("MermaidRenderer.Render() = %q, want %q", gotStr, tt.want)
			}
		})
	}
}

func TestDOTRenderer_ContextCancellation(t *testing.T) {
	// Create a large document
	var edges []Edge
	for i := 0; i < 1000; i++ {
		edges = append(edges, Edge{
			From: "Node" + string(rune(i)),
			To:   "Node" + string(rune(i+1)),
		})
	}
	doc := New().Graph("Large Graph", edges).Build()

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	renderer := &dotRenderer{}
	_, err := renderer.Render(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

func TestSanitizeDOTID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with space", `"with space"`},
		{"with-dash", `"with-dash"`},
		{"with:colon", `"with:colon"`},
		{"with/slash", `"with/slash"`},
		{`with"quote`, `"with\"quote"`},
		{"a.b.c", `"a.b.c"`},
		{"normal_underscore", "normal_underscore"},
		{"CamelCase", "CamelCase"},
		{"123numeric", "123numeric"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeDOTID(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeDOTID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeMermaidID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with space", "[with space]"},
		{"with-dash", "[with-dash]"},
		{"with:colon", "[with:colon]"},
		{"with/slash", "[with/slash]"},
		{"with\"quote", "[with\"quote]"},
		{"a.b.c", "[a.b.c]"},
		{"normal_underscore", "normal_underscore"},
		{"CamelCase", "CamelCase"},
		{"123numeric", "123numeric"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeMermaidID(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeMermaidID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDrawioRenderer_Render(t *testing.T) {
	tests := []struct {
		name         string
		doc          *Document
		wantContains []string // Check for key parts of CSV output
		wantErr      bool
	}{
		{
			name: "simple graph",
			doc: New().
				Graph("Test Graph", []Edge{
					{From: "A", To: "B"},
					{From: "B", To: "C"},
				}).
				Build(),
			wantContains: []string{
				"# label: %Name%",
				"# style: %Image%",
				"# layout: auto",
				"Name,From,To,Label",
				"A,,,",
				"B,,,",
				"C,,,",
				",A,B,",
				",B,C,",
			},
		},
		{
			name: "graph with labels",
			doc: New().
				Graph("Workflow", []Edge{
					{From: "Start", To: "Process", Label: "begin"},
					{From: "Process", To: "End", Label: "complete"},
				}).
				Build(),
			wantContains: []string{
				"# label: %Name%",
				"Name,From,To,Label",
				"Start,,,",
				"Process,,,",
				"End,,,",
				",Start,Process,begin",
				",Process,End,complete",
			},
		},
		{
			name: "graph with special CSV characters",
			doc: New().
				Graph("", []Edge{
					{From: "A,B", To: "C\"D"},
					{From: "C\"D", To: "E\nF"},
				}).
				Build(),
			wantContains: []string{
				"\"A,B\",,,",
				"\"C\"\"D\",,,",
				"\"E\nF\",,,",
				",\"A,B\",\"C\"\"D\",",
				",\"C\"\"D\",\"E\nF\",",
			},
		},
		{
			name: "empty graph",
			doc: New().
				Graph("Empty", []Edge{}).
				Build(),
			wantContains: []string{
				"# label: %Name%",
				"# layout: auto",
				"Name,From,To,Label",
			},
		},
		{
			name: "table with from/to columns",
			doc: New().
				Table("Dependencies", []map[string]any{
					{"from": "package-a", "to": "package-b", "label": "depends"},
				}).
				Build(),
			wantContains: []string{
				"# label: %Name%",
				"# connect:",
				"from,label,to",               // Alphabetically sorted column order
				"package-a,depends,package-b", // Data in alphabetically sorted column order
			},
		},
	}

	ctx := context.Background()
	renderer := &drawioRenderer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderer.Render(ctx, tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("DrawioRenderer.Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotStr := string(got)
			for _, want := range tt.wantContains {
				if !strings.Contains(gotStr, want) {
					t.Errorf("DrawioRenderer.Render() output missing %q\nGot:\n%s", want, gotStr)
				}
			}
		})
	}
}
