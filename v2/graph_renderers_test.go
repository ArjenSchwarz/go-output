package output

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestGraphRenderers_NilDocument verifies that the DOT, Mermaid, and Draw.io
// renderers return a "document cannot be nil" error instead of panicking when
// Render or RenderTo is called directly with a nil document (T-1092). Other
// renderers (csv_renderer.go, base_renderer.go, table_renderer.go) already
// guard against nil documents, so these graph renderers should behave the same.
func TestGraphRenderers_NilDocument(t *testing.T) {
	renderers := map[string]Renderer{
		"DOT":     DOT().Renderer,
		"Mermaid": Mermaid().Renderer,
		"DrawIO":  DrawIO().Renderer,
	}

	ctx := context.Background()

	for name, renderer := range renderers {
		t.Run(name+"/Render", func(t *testing.T) {
			_, err := renderer.Render(ctx, nil)
			if err == nil {
				t.Fatalf("%s.Render(ctx, nil) returned no error, want \"document cannot be nil\"", name)
			}
			if !strings.Contains(err.Error(), "document cannot be nil") {
				t.Errorf("%s.Render(ctx, nil) error = %q, want it to contain \"document cannot be nil\"", name, err.Error())
			}
		})

		t.Run(name+"/RenderTo", func(t *testing.T) {
			var buf bytes.Buffer
			err := renderer.RenderTo(ctx, nil, &buf)
			if err == nil {
				t.Fatalf("%s.RenderTo(ctx, nil, w) returned no error, want \"document cannot be nil\"", name)
			}
			if !strings.Contains(err.Error(), "document cannot be nil") {
				t.Errorf("%s.RenderTo(ctx, nil, w) error = %q, want it to contain \"document cannot be nil\"", name, err.Error())
			}
		})
	}
}

func TestDOTRenderer_Render(t *testing.T) {
	tests := map[string]struct {
		doc     *Document
		want    string
		wantErr bool
	}{"empty graph": {

		doc: New().
			Graph("Empty", []Edge{}).
			Build(),
		want: `digraph {
  label="Empty";
}
`,
	}, "graph with labels": {

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
	}, "graph with special characters": {

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
	}, "multiple graphs": {

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
	}, "simple graph": {

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
	}, "table with Source/Target columns": {

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
	}, "table with from/to columns": {

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
	}, "table without graph columns": {

		doc: New().
			Table("Users", []map[string]any{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
			}).
			Build(),
		want: `digraph {
}
`,
	}}

	ctx := context.Background()
	renderer := &dotRenderer{}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
	tests := map[string]struct {
		doc     *Document
		want    string
		wantErr bool
	}{"empty graph": {

		doc: New().
			Graph("Empty", []Edge{}).
			Build(),
		want: `graph TD
  % Empty
`,
	}, "graph with labels": {

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
	}, "graph with special characters": {

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
	}, "multiple graphs": {

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
	}, "simple graph": {

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
	}, "table with Source/Target columns": {

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
	}, "table with from/to columns": {

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
	}, "table without graph columns": {

		doc: New().
			Table("Users", []map[string]any{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
			}).
			Build(),
		want: `graph TD
`,
	}}

	ctx := context.Background()
	renderer := &mermaidRenderer{}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
	for i := range 1000 {
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
		{"with\"quote", "[\"with&quot;quote\"]"},
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

// TestDOTRenderer_EscapesLabels is a regression test for T-1292.
//
// Bug: the DOT renderer wrote user-controlled title and edge label text
// directly into label="%s" via Fprintf without escaping. Labels containing
// double quotes, backslashes, or newlines broke the generated DOT syntax
// (e.g. a label of `say "hi"` produced label="say "hi"" which is invalid).
//
// Expected: special characters are escaped per DOT string rules — `"` becomes
// `\"`, `\` becomes `\\`, and newlines become the literal `\n` sequence.
func TestDOTRenderer_EscapesLabels(t *testing.T) {
	tests := map[string]struct {
		doc  *Document
		want string
	}{
		"title with quotes": {
			doc: New().
				Graph(`say "hi"`, []Edge{
					{From: "A", To: "B"},
				}).
				Build(),
			want: `digraph {
  label="say \"hi\"";
  A -> B;
}
`,
		},
		"edge label with quotes": {
			doc: New().
				Graph("", []Edge{
					{From: "A", To: "B", Label: `weight "5"`},
				}).
				Build(),
			want: `digraph {
  A -> B [label="weight \"5\""];
}
`,
		},
		"edge label with backslash": {
			doc: New().
				Graph("", []Edge{
					{From: "A", To: "B", Label: `path\to`},
				}).
				Build(),
			want: `digraph {
  A -> B [label="path\\to"];
}
`,
		},
		"edge label with newline": {
			doc: New().
				Graph("", []Edge{
					{From: "A", To: "B", Label: "line1\nline2"},
				}).
				Build(),
			want: `digraph {
  A -> B [label="line1\nline2"];
}
`,
		},
	}

	ctx := context.Background()
	renderer := &dotRenderer{}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := renderer.Render(ctx, tt.doc)
			if err != nil {
				t.Fatalf("DOTRenderer.Render() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("DOTRenderer.Render() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

// TestMermaidRenderer_EscapesLabels is a regression test for T-1292.
//
// Bug: the Mermaid renderer wrote user-controlled edge label text directly into
// -->|%s| and node text into [%s] without escaping. Labels containing double
// quotes, pipes, or newlines broke the generated Mermaid syntax (e.g. a label
// containing `|` prematurely closed the edge-label delimiter).
//
// Expected: label text is wrapped in double quotes and quote characters are
// HTML-entity-encoded (&quot;) so Mermaid renders the literal text safely.
func TestMermaidRenderer_EscapesLabels(t *testing.T) {
	tests := map[string]struct {
		doc  *Document
		want string
	}{
		"edge label with quotes": {
			doc: New().
				Graph("", []Edge{
					{From: "A", To: "B", Label: `weight "5"`},
				}).
				Build(),
			want: `graph TD
  A -->|"weight &quot;5&quot;"| B
`,
		},
		"edge label with pipe": {
			doc: New().
				Graph("", []Edge{
					{From: "A", To: "B", Label: "a|b"},
				}).
				Build(),
			want: `graph TD
  A -->|"a|b"| B
`,
		},
		"edge label with newline": {
			doc: New().
				Graph("", []Edge{
					{From: "A", To: "B", Label: "line1\nline2"},
				}).
				Build(),
			want: `graph TD
  A -->|"line1<br/>line2"| B
`,
		},
		"node text with quotes": {
			doc: New().
				Graph("", []Edge{
					{From: `node "x"`, To: "B"},
				}).
				Build(),
			want: `graph TD
  ["node &quot;x&quot;"] --> B
`,
		},
	}

	ctx := context.Background()
	renderer := &mermaidRenderer{}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := renderer.Render(ctx, tt.doc)
			if err != nil {
				t.Fatalf("MermaidRenderer.Render() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MermaidRenderer.Render() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestDrawioRenderer_Render(t *testing.T) {
	tests := map[string]struct {
		doc          *Document
		wantContains []string // Check for key parts of CSV output
		wantErr      bool
	}{"empty graph": {

		doc: New().
			Graph("Empty", []Edge{}).
			Build(),
		wantContains: []string{
			"# label: %Name%",
			"# layout: auto",
			"Name,From,To,Label",
		},
	}, "graph with labels": {

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
	}, "graph with special CSV characters": {

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
	}, "simple graph": {

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
	}, "table with from/to columns": {

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
	}}

	ctx := context.Background()
	renderer := &drawioRenderer{}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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

// graphTransformationDoc builds a document with a single edge table that has
// from/to/label columns and a filter + limit transformation attached.
//
// Source rows (in order):
//   - keep1 -> keep1b   (keep = true)
//   - drop1 -> drop1b   (keep = false, filtered out)
//   - keep2 -> keep2b   (keep = true)
//   - keep3 -> keep3b   (keep = true, removed by limit 2)
//
// After the filter (keep == true) and limit (2), only the first two kept rows
// survive: keep1 -> keep1b and keep2 -> keep2b.
func graphTransformationDoc() *Document {
	return New().
		Table("Edges", []map[string]any{
			{"from": "keep1", "to": "keep1b", "label": "L1", "keep": true},
			{"from": "drop1", "to": "drop1b", "label": "L2", "keep": false},
			{"from": "keep2", "to": "keep2b", "label": "L3", "keep": true},
			{"from": "keep3", "to": "keep3b", "label": "L4", "keep": true},
		},
			WithKeys("from", "to", "label", "keep"),
			WithTransformations(
				NewFilterOp(func(r Record) bool {
					kept, _ := r["keep"].(bool)
					return kept
				}),
				NewLimitOp(2),
			),
		).
		Build()
}

// TestDOTRenderer_AppliesTableTransformations is a regression test for T-1091:
// the DOT renderer must apply per-content transformations (filter/limit) before
// extracting graph data from a table, so filtered/limited rows do not appear.
func TestDOTRenderer_AppliesTableTransformations(t *testing.T) {
	ctx := context.Background()
	renderer := &dotRenderer{}

	got, err := renderer.Render(ctx, graphTransformationDoc())
	if err != nil {
		t.Fatalf("DOTRenderer.Render() error = %v", err)
	}
	gotStr := string(got)

	wantPresent := []string{"keep1 -> keep1b", "keep2 -> keep2b"}
	for _, want := range wantPresent {
		if !strings.Contains(gotStr, want) {
			t.Errorf("DOTRenderer.Render() missing expected edge %q\nGot:\n%s", want, gotStr)
		}
	}

	wantAbsent := []string{"drop1", "keep3"}
	for _, absent := range wantAbsent {
		if strings.Contains(gotStr, absent) {
			t.Errorf("DOTRenderer.Render() included transformed-out row %q\nGot:\n%s", absent, gotStr)
		}
	}
}

// TestMermaidRenderer_AppliesTableTransformations is a regression test for
// T-1091: the Mermaid renderer must apply per-content transformations before
// extracting graph data from a table.
func TestMermaidRenderer_AppliesTableTransformations(t *testing.T) {
	ctx := context.Background()
	renderer := &mermaidRenderer{}

	got, err := renderer.Render(ctx, graphTransformationDoc())
	if err != nil {
		t.Fatalf("MermaidRenderer.Render() error = %v", err)
	}
	gotStr := string(got)

	wantPresent := []string{"keep1 -->|L1| keep1b", "keep2 -->|L3| keep2b"}
	for _, want := range wantPresent {
		if !strings.Contains(gotStr, want) {
			t.Errorf("MermaidRenderer.Render() missing expected edge %q\nGot:\n%s", want, gotStr)
		}
	}

	wantAbsent := []string{"drop1", "keep3"}
	for _, absent := range wantAbsent {
		if strings.Contains(gotStr, absent) {
			t.Errorf("MermaidRenderer.Render() included transformed-out row %q\nGot:\n%s", absent, gotStr)
		}
	}
}

// TestDrawioRenderer_AppliesTableTransformations is a regression test for
// T-1091: the Draw.io renderer must apply per-content transformations before
// emitting table rows as CSV.
func TestDrawioRenderer_AppliesTableTransformations(t *testing.T) {
	ctx := context.Background()
	renderer := &drawioRenderer{}

	got, err := renderer.Render(ctx, graphTransformationDoc())
	if err != nil {
		t.Fatalf("DrawioRenderer.Render() error = %v", err)
	}
	gotStr := string(got)

	wantPresent := []string{"keep1", "keep2"}
	for _, want := range wantPresent {
		if !strings.Contains(gotStr, want) {
			t.Errorf("DrawioRenderer.Render() missing expected row %q\nGot:\n%s", want, gotStr)
		}
	}

	wantAbsent := []string{"drop1", "keep3"}
	for _, absent := range wantAbsent {
		if strings.Contains(gotStr, absent) {
			t.Errorf("DrawioRenderer.Render() included transformed-out row %q\nGot:\n%s", absent, gotStr)
		}
	}
}
