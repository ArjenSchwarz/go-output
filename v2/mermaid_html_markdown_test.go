package output

import (
	"context"
	"strings"
	"testing"
)

func TestHTMLRenderer_ChartContent(t *testing.T) {
	tests := map[string]struct {
		chart       *ChartContent
		mustContain []string
	}{
		"gantt chart in HTML": {
			chart: NewGanttChart("Project Timeline", []GanttTask{
				{
					ID:        "task1",
					Title:     "Design Phase",
					StartDate: "2024-01-01",
					Duration:  "5d",
					Status:    "done",
					Section:   "Planning",
				},
			}),
			mustContain: []string{
				`<pre class="mermaid">`,
				"gantt",
				"title Project Timeline",
				"dateFormat YYYY-MM-DD",
				"section Planning",
				"Design Phase :done, task1, 2024-01-01, 5d",
				"</pre>",
			},
		},
		"pie chart in HTML": {
			chart: NewPieChart("Distribution", []PieSlice{
				{Label: "A", Value: 60},
				{Label: "B", Value: 40},
			}, true),
			mustContain: []string{
				`<pre class="mermaid">`,
				"pie showData",
				"title Distribution",
				// Double quotes around pie labels are HTML-escaped (T-1293).
				`&#34;A&#34; : 60.00`,
				`&#34;B&#34; : 40.00`,
				"</pre>",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().AddContent(tc.chart).Build()
			renderer := &htmlRenderer{}

			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			output := string(result)

			for _, expected := range tc.mustContain {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

// TestHTMLRenderer_ChartContent_EscapesUserText is a regression test for T-1293.
//
// renderChartContentHTML wrote the raw Mermaid bytes directly inside
// <pre class="mermaid"> without HTML escaping. Chart titles, task names,
// section names, and pie labels are user-controlled and the Mermaid renderer
// emits them as raw text, so values containing "<", "</pre>", or "<script>"
// could break out of the pre block or be interpreted as HTML/script (an XSS
// injection path).
//
// Expected behaviour: HTML metacharacters in user-controlled chart fields are
// HTML-escaped in the output. The browser un-escapes the text content of the
// pre block before Mermaid.js parses it, so this preserves valid Mermaid syntax.
func TestHTMLRenderer_ChartContent_EscapesUserText(t *testing.T) {
	tests := map[string]struct {
		chart        *ChartContent
		mustContain  []string
		mustNotMatch []string
	}{
		"pie chart with injection in title and labels": {
			chart: NewPieChart(`<script>alert('xss')</script>`, []PieSlice{
				{Label: `</pre><script>alert(1)</script>`, Value: 60},
				{Label: "Safe", Value: 40},
			}, true),
			mustContain: []string{
				`<pre class="mermaid">`,
				"</pre>",
				// The injected markup must be escaped, not emitted verbatim.
				"&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
				"&lt;/pre&gt;&lt;script&gt;alert(1)&lt;/script&gt;",
			},
			mustNotMatch: []string{
				// A literal opening script tag would execute in the browser.
				"<script>alert('xss')</script>",
				"<script>alert(1)</script>",
				// A literal closing pre breaks out of the mermaid block.
				"</pre><script>",
			},
		},
		"gantt chart with injection in title, section and task name": {
			chart: NewGanttChart(`<script>evil()</script>`, []GanttTask{
				{
					ID:        "task1",
					Title:     `</pre><img src=x onerror=alert(1)>`,
					StartDate: "2024-01-01",
					Duration:  "5d",
					Status:    "done",
					Section:   `<script>section()</script>`,
				},
			}),
			mustContain: []string{
				`<pre class="mermaid">`,
				"</pre>",
				"&lt;script&gt;evil()&lt;/script&gt;",
				"&lt;script&gt;section()&lt;/script&gt;",
				"&lt;/pre&gt;&lt;img src=x onerror=alert(1)&gt;",
			},
			mustNotMatch: []string{
				"<script>evil()</script>",
				"<script>section()</script>",
				"</pre><img src=x onerror=alert(1)>",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().AddContent(tc.chart).Build()
			renderer := &htmlRenderer{}

			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			output := string(result)

			for _, expected := range tc.mustContain {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got:\n%s", expected, output)
				}
			}

			// Isolate the chart block so the mermaid script (which legitimately
			// contains a <script> tag) does not produce false positives.
			start := strings.Index(output, `<pre class="mermaid">`)
			if start < 0 {
				t.Fatalf("Output missing mermaid pre block, got:\n%s", output)
			}
			end := strings.Index(output[start:], "</pre>")
			if end < 0 {
				t.Fatalf("Output missing closing </pre>, got:\n%s", output)
			}
			chartBlock := output[start : start+end]

			for _, forbidden := range tc.mustNotMatch {
				if strings.Contains(chartBlock, forbidden) {
					t.Errorf("Chart block should not contain unescaped %q, got:\n%s", forbidden, chartBlock)
				}
			}
		})
	}
}

func TestMarkdownRenderer_ChartContent(t *testing.T) {
	tests := map[string]struct {
		chart       *ChartContent
		mustContain []string
	}{
		"gantt chart in Markdown": {
			chart: NewGanttChart("Project Timeline", []GanttTask{
				{
					ID:        "task1",
					Title:     "Design Phase",
					StartDate: "2024-01-01",
					Duration:  "5d",
					Status:    "done",
					Section:   "Planning",
				},
			}),
			mustContain: []string{
				"```mermaid",
				"gantt",
				"title Project Timeline",
				"dateFormat YYYY-MM-DD",
				"section Planning",
				"Design Phase :done, task1, 2024-01-01, 5d",
				"```",
			},
		},
		"pie chart in Markdown": {
			chart: NewPieChart("Distribution", []PieSlice{
				{Label: "A", Value: 60},
				{Label: "B", Value: 40},
			}, true),
			mustContain: []string{
				"```mermaid",
				"pie showData",
				"title Distribution",
				`"A" : 60.00`,
				`"B" : 40.00`,
				"```",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().AddContent(tc.chart).Build()
			renderer := &markdownRenderer{headingLevel: 1}

			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			output := string(result)

			for _, expected := range tc.mustContain {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestMarkdownRenderer_ChartCodeFenceFormat(t *testing.T) {
	// Test that the code fence is properly formatted with no extra spaces
	chart := NewGanttChart("Test", []GanttTask{
		{Title: "Task", StartDate: "2024-01-01", Duration: "1d"},
	})

	doc := New().AddContent(chart).Build()
	renderer := &markdownRenderer{headingLevel: 1}

	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Check code fence starts correctly
	if !strings.HasPrefix(strings.TrimSpace(output), "```mermaid") {
		t.Errorf("Output should start with ```mermaid, got:\n%s", output)
	}

	// Check code fence ends correctly (no space before closing backticks)
	if !strings.HasSuffix(strings.TrimSpace(output), "```") {
		t.Errorf("Output should end with ```, got:\n%s", output)
	}
}

func TestHTMLRenderer_ChartPreClass(t *testing.T) {
	// Test that HTML output has correct class attribute
	chart := NewPieChart("Test", []PieSlice{
		{Label: "X", Value: 100},
	}, false)

	doc := New().AddContent(chart).Build()
	renderer := &htmlRenderer{}

	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Check for exact class attribute
	if !strings.Contains(output, `class="mermaid"`) {
		t.Errorf("Output should contain class=\"mermaid\", got:\n%s", output)
	}

	// Ensure it's within a pre tag
	if !strings.Contains(output, "<pre class=\"mermaid\">") {
		t.Errorf("Output should contain <pre class=\"mermaid\">, got:\n%s", output)
	}
}

func TestHTMLRenderer_MermaidScriptInjection(t *testing.T) {
	tests := map[string]struct {
		doc              *Document
		shouldHaveScript bool
	}{
		"document with chart content": {
			doc: New().
				GanttChart("Test", []GanttTask{
					{Title: "Task", StartDate: "2024-01-01", Duration: "1d"},
				}).
				Build(),
			shouldHaveScript: true,
		},
		"document without chart content": {
			doc: New().
				Text("Plain text").
				Table("Data", []map[string]any{{"key": "value"}}).
				Build(),
			shouldHaveScript: false,
		},
		"document with nested chart in section": {
			doc: func() *Document {
				section := NewSectionContent("Charts", WithLevel(1))
				section.AddContent(NewPieChart("Test", []PieSlice{{Label: "A", Value: 100}}, false))
				return New().AddContent(section).Build()
			}(),
			shouldHaveScript: true,
		},
		"mixed content with chart": {
			doc: New().
				Text("Header").
				GanttChart("Timeline", []GanttTask{
					{Title: "Task", StartDate: "2024-01-01", Duration: "1d"},
				}).
				Table("Data", []map[string]any{{"key": "value"}}).
				Build(),
			shouldHaveScript: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			renderer := &htmlRenderer{}
			result, err := renderer.Render(context.Background(), tc.doc)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			output := string(result)

			// Check for mermaid.js script
			hasScript := strings.Contains(output, `<script type="module">`) &&
				strings.Contains(output, `import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs'`) &&
				strings.Contains(output, `mermaid.initialize({ startOnLoad: true })`)

			if tc.shouldHaveScript && !hasScript {
				t.Errorf("Output should contain mermaid.js script, got:\n%s", output)
			}

			if !tc.shouldHaveScript && hasScript {
				t.Errorf("Output should NOT contain mermaid.js script for non-chart content, got:\n%s", output)
			}
		})
	}
}

func TestHTMLRenderer_MermaidScriptFormat(t *testing.T) {
	// Test the exact format of the injected script
	chart := NewPieChart("Test", []PieSlice{{Label: "A", Value: 100}}, false)
	doc := New().AddContent(chart).Build()
	renderer := &htmlRenderer{}

	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Verify the script is at the end
	scriptStart := strings.Index(output, "<script type=\"module\">")
	if scriptStart == -1 {
		t.Fatal("Script tag not found in output")
	}

	// Verify it's after the chart content
	chartPos := strings.Index(output, `<pre class="mermaid">`)
	if chartPos == -1 {
		t.Fatal("Chart content not found in output")
	}

	if scriptStart < chartPos {
		t.Error("Script should appear after chart content")
	}

	// Verify script contains required elements
	requiredElements := []string{
		`<script type="module">`,
		`import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs'`,
		`mermaid.initialize({ startOnLoad: true })`,
		`</script>`,
	}

	for _, element := range requiredElements {
		if !strings.Contains(output, element) {
			t.Errorf("Script should contain %q", element)
		}
	}
}
