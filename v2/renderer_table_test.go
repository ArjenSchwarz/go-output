package output

import (
	"context"
	"strings"
	"testing"
)

func TestTableRenderer_KeyOrderPreservation(t *testing.T) {
	tests := map[string]struct {
		keys     []string
		data     []map[string]any
		expected []string
	}{"preserve explicit key order": {

		keys: []string{"c", "a", "b"},
		data: []map[string]any{
			{"a": "alpha", "b": "beta", "c": "gamma"},
			{"c": "charlie", "b": "bravo", "a": "alpha"},
		},
		expected: []string{"c", "a", "b"},
	}, "preserve numeric and string keys": {

		keys: []string{"id", "name", "score", "active"},
		data: []map[string]any{
			{"name": "Alice", "id": 1, "active": true, "score": 95.5},
			{"score": 87.2, "id": 2, "name": "Bob", "active": false},
		},
		expected: []string{"id", "name", "score", "active"},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create table with explicit key order
			doc := New().
				Table("Test Table", tt.data, WithKeys(tt.keys...)).
				Build()

			// Test with table renderer
			renderer := &tableRenderer{}
			ctx := context.Background()

			result, err := renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render table: %v", err)
			}

			resultStr := string(result)

			// Verify that the output contains the table title
			if !strings.Contains(resultStr, "Test Table") {
				t.Errorf("Output does not contain table title")
			}

			// Split into lines to check header order
			lines := strings.Split(resultStr, "\n")
			var headerLine string

			// Find the header line (should contain all our keys)
			// Look for uppercase versions since go-pretty converts headers to uppercase
			upperKeys := make([]string, len(tt.keys))
			for i, key := range tt.keys {
				upperKeys[i] = strings.ToUpper(key)
			}

			for _, line := range lines {
				if strings.Contains(line, upperKeys[0]) && strings.Contains(line, upperKeys[1]) {
					headerLine = line
					break
				}
			}

			if headerLine == "" {
				t.Errorf("Could not find header line in output. Full output:\n%s", resultStr)
				return
			}

			// Verify that keys appear in the correct order in the header
			// We check that each key appears before the next one
			// Use uppercase versions for comparison
			for i := 0; i < len(tt.expected)-1; i++ {
				key1 := strings.ToUpper(tt.expected[i])
				key2 := strings.ToUpper(tt.expected[i+1])

				pos1 := strings.Index(headerLine, key1)
				pos2 := strings.Index(headerLine, key2)

				if pos1 == -1 {
					t.Errorf("Key %s not found in header", key1)
				}
				if pos2 == -1 {
					t.Errorf("Key %s not found in header", key2)
				}
				if pos1 >= pos2 {
					t.Errorf("Key %s appears after %s in header, expected before",
						key1, key2)
				}
			}
		})
	}
}

func TestTableRenderer_MixedContent(t *testing.T) {
	// Create a document with mixed content types
	data := []map[string]any{
		{"name": "Alice", "age": 30, "city": "New York"},
		{"name": "Bob", "age": 25, "city": "Los Angeles"},
	}

	doc := New().
		Text("User Report").
		Table("Users", data, WithKeys("name", "age", "city")).
		Text("End of report").
		Build()

	renderer := &tableRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render mixed content: %v", err)
	}

	resultStr := string(result)

	// Should contain text content
	if !strings.Contains(resultStr, "User Report") {
		t.Errorf("Output missing text content 'User Report'")
	}

	if !strings.Contains(resultStr, "End of report") {
		t.Errorf("Output missing text content 'End of report'")
	}

	// Should contain table data
	if !strings.Contains(resultStr, "Alice") {
		t.Errorf("Output missing table data 'Alice'")
	}

	if !strings.Contains(resultStr, "Bob") {
		t.Errorf("Output missing table data 'Bob'")
	}

	// Should contain table title
	if !strings.Contains(resultStr, "Users") {
		t.Errorf("Output missing table title 'Users'")
	}
}

func TestTableRenderer_SectionContent(t *testing.T) {
	// Create a document with section content
	userData := []map[string]any{
		{"name": "Alice", "role": "Admin"},
		{"name": "Bob", "role": "User"},
	}

	doc := New().
		Section("User Management", func(b *Builder) {
			b.Text("This section contains user information").
				Table("Active Users", userData, WithKeys("name", "role"))
		}).
		Build()

	renderer := &tableRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render section content: %v", err)
	}

	resultStr := string(result)

	// Should contain section marker
	if !strings.Contains(resultStr, "=== User Management ===") {
		t.Errorf("Output missing section header")
	}

	// Should contain section text and table data
	if !strings.Contains(resultStr, "This section contains user information") {
		t.Errorf("Output missing section text")
	}

	if !strings.Contains(resultStr, "Alice") || !strings.Contains(resultStr, "Admin") {
		t.Errorf("Output missing table data from section")
	}
}

func TestTableRenderer_StyleConfiguration(t *testing.T) {
	data := []map[string]any{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
	}

	tests := map[string]struct {
		renderer  *tableRenderer
		styleName string
	}{"bold style":

	// default

	{

		renderer:  NewTableRendererWithStyle("Bold").(*tableRenderer),
		styleName: "Bold",
	}, "default style": {

		renderer:  &tableRenderer{},
		styleName: "ColoredBright",
	}, "light style": {

		renderer:  NewTableRendererWithStyle("Light").(*tableRenderer),
		styleName: "Light",
	}, "rounded style": {

		renderer:  NewTableRendererWithStyle("Rounded").(*tableRenderer),
		styleName: "Rounded",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				Table("Styled Table", data, WithKeys("name", "age")).
				Build()

			ctx := context.Background()
			result, err := tt.renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render table with style: %v", err)
			}

			resultStr := string(result)

			// Should contain the data regardless of style
			if !strings.Contains(resultStr, "Alice") {
				t.Errorf("Table data missing from %s style", tt.styleName)
			}

			if !strings.Contains(resultStr, "Bob") {
				t.Errorf("Table data missing from %s style", tt.styleName)
			}

			// Should contain table title (may be split across lines with ANSI codes)
			if !strings.Contains(resultStr, "Styled") || !(strings.Contains(resultStr, "Table") || strings.Contains(resultStr, "Tabl") || strings.Contains(resultStr, "Tab")) {
				t.Errorf("Table title missing from %s style. Full output:\n%s", tt.styleName, resultStr)
			}

			// Different styles should produce different output (basic check)
			if tt.styleName != "ColoredBright" {
				// Create a default renderer for comparison
				defaultRenderer := &tableRenderer{}
				defaultResult, err := defaultRenderer.Render(ctx, doc)
				if err != nil {
					t.Fatalf("Failed to render with default style: %v", err)
				}

				// The styled output should be different from default
				// (This is a basic check - different styles will have different ANSI codes)
				if string(result) == string(defaultResult) {
					t.Errorf("Style %s produced identical output to default", tt.styleName)
				}
			}
		})
	}
}

func TestTableRenderer_PredefinedStyles(t *testing.T) {
	data := []map[string]any{
		{"id": 1, "name": "Test"},
	}

	doc := New().
		Table("Style Test", data, WithKeys("id", "name")).
		Build()

	// Test some of the predefined style formats
	styles := []Format{
		TableDefault(),
		TableBold(),
		TableColoredBright(),
		TableLight(),
		TableRounded(),
	}

	ctx := context.Background()

	for _, style := range styles {
		t.Run(style.Name+"_"+style.Renderer.(*tableRenderer).styleName, func(t *testing.T) {
			result, err := style.Renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render with predefined style: %v", err)
			}

			resultStr := string(result)

			// Should contain the basic data
			if !strings.Contains(resultStr, "Test") {
				t.Errorf("Table data missing from predefined style")
			}

			if !strings.Contains(resultStr, "Style") || !strings.Contains(resultStr, "Test") {
				t.Errorf("Table title missing from predefined style")
			}
		})
	}
}

func TestTableWithStyle_Function(t *testing.T) {
	// Test the TableWithStyle function
	customStyle := TableWithStyle("Double")

	if customStyle.Name != FormatTable {
		t.Errorf("TableWithStyle should have name %q, got %s", FormatTable, customStyle.Name)
	}

	renderer, ok := customStyle.Renderer.(*tableRenderer)
	if !ok {
		t.Fatalf("TableWithStyle should return tableRenderer")
	}

	if renderer.styleName != "Double" {
		t.Errorf("TableWithStyle should set styleName to 'Double', got %s", renderer.styleName)
	}
}

// TestTableRenderer_TransformationIntegration tests that TableRenderer applies transformations
func TestTableRenderer_TransformationIntegration(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	doc := New().
		Table("test", data,
			WithKeys("name", "age"),
			WithTransformations(
				NewFilterOp(func(r Record) bool {
					return r["age"].(int) >= 30
				}),
			),
		).
		Build()

	renderer := &tableRenderer{styleName: "Default"}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	resultStr := string(result)
	// Should contain Alice and Charlie but not Bob
	if !strings.Contains(resultStr, "Alice") {
		t.Error("Missing Alice after filter")
	}
	if !strings.Contains(resultStr, "Charlie") {
		t.Error("Missing Charlie after filter")
	}
	if strings.Contains(resultStr, "Bob") {
		t.Error("Bob should be filtered out")
	}
}

// TestTableRenderer_DeeplyNestedSections is a regression test for T-1522.
// The table renderer unrolled nested sections manually instead of recursing:
// level 1 handled tables/text/sections, level 2 handled only tables. Sections
// nested 3+ levels deep were silently dropped from the output. This test
// builds a table at several nesting depths and verifies its rows (and the
// innermost section header) reach the rendered output.
func TestTableRenderer_DeeplyNestedSections(t *testing.T) {
	// newTable builds a single-row table whose only value identifies the
	// depth at which it lives, so we can assert each table reaches the output.
	newTable := func(t *testing.T, name string) *TableContent {
		t.Helper()
		table, err := NewTableContent(name, []map[string]any{{"name": name}}, WithKeys("name"))
		if err != nil {
			t.Fatalf("failed to create table %q: %v", name, err)
		}
		return table
	}

	tests := map[string]struct {
		// depth is the number of nested sections wrapping the table.
		// depth=1 -> section -> table (already worked before the fix)
		// depth=2 -> outer -> inner -> table (already worked before the fix)
		// depth=3+ -> outer -> ... -> inner -> table (regression)
		depth int
		row   string
	}{
		"one level":    {depth: 1, row: "level-1"},
		"two levels":   {depth: 2, row: "level-2"},
		"three levels": {depth: 3, row: "deep-row"},
		"five levels":  {depth: 5, row: "level-5"},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			// Build the innermost section holding the table, then wrap it in
			// the requested number of outer sections.
			innermost := NewSectionContent("innermost-section")
			innermost.AddContent(newTable(t, tt.row))

			current := innermost
			for level := 1; level < tt.depth; level++ {
				outer := NewSectionContent("section")
				outer.AddContent(current)
				current = outer
			}

			doc := New().AddContent(current).Build()

			result, err := (&tableRenderer{}).Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			out := string(result)
			if !strings.Contains(out, tt.row) {
				t.Errorf("table output missing table nested %d level(s) deep: want it to contain %q, got %q",
					tt.depth, tt.row, out)
			}
			if !strings.Contains(out, "=== innermost-section ===") {
				t.Errorf("table output missing header of section nested %d level(s) deep, got %q",
					tt.depth, out)
			}
		})
	}
}

// TestTableRenderer_NestedSectionTextContent is a regression test for T-1522.
// Text content inside a section nested within another section was silently
// dropped because the hand-unrolled nested-section loop only rendered tables.
func TestTableRenderer_NestedSectionTextContent(t *testing.T) {
	text := NewTextContent("nested text survives")
	table, err := NewTableContent("users", []map[string]any{{"name": "Alice"}}, WithKeys("name"))
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	inner := NewSectionContent("inner")
	inner.AddContent(text)
	inner.AddContent(table)

	outer := NewSectionContent("outer")
	outer.AddContent(inner)

	doc := New().AddContent(outer).Build()

	result, err := (&tableRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := string(result)
	if !strings.Contains(out, "nested text survives") {
		t.Errorf("table output missing text content inside nested section, got %q", out)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("table output missing table content inside nested section, got %q", out)
	}
}

// TestTableRenderer_MultipleTablesAcrossNestingLevels verifies that tables at
// different depths within the same section hierarchy are all rendered, not
// just the shallowest ones. Regression test for T-1522.
func TestTableRenderer_MultipleTablesAcrossNestingLevels(t *testing.T) {
	makeTable := func(t *testing.T, row string) *TableContent {
		t.Helper()
		table, err := NewTableContent(row, []map[string]any{{"name": row}}, WithKeys("name"))
		if err != nil {
			t.Fatalf("failed to create table %q: %v", row, err)
		}
		return table
	}

	// Structure: outer{tableA, middle{tableB, inner{tableC}}}
	inner := NewSectionContent("inner")
	inner.AddContent(makeTable(t, "row-c"))

	middle := NewSectionContent("middle")
	middle.AddContent(makeTable(t, "row-b"))
	middle.AddContent(inner)

	outer := NewSectionContent("outer")
	outer.AddContent(makeTable(t, "row-a"))
	outer.AddContent(middle)

	doc := New().AddContent(outer).Build()

	result, err := (&tableRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := string(result)
	for _, want := range []string{"row-a", "row-b", "row-c"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing table row %q across nesting levels, got %q", want, out)
		}
	}
}

// TestTableRenderer_NestedRawAndCollapsibleContent is a follow-up regression
// test for T-1522. RawContent and DefaultCollapsibleSection nested inside a
// SectionContent were silently dropped because renderSectionTable's switch
// only matched tables, text, and sections, while the top-level switch in
// renderDocumentTable handles all content types.
func TestTableRenderer_NestedRawAndCollapsibleContent(t *testing.T) {
	raw, err := NewRawContent(FormatText, []byte("nested raw survives"))
	if err != nil {
		t.Fatalf("failed to create raw content: %v", err)
	}
	collapsible := NewCollapsibleSection("nested-collapsible", []Content{
		NewTextContent("collapsible body"),
	}, WithSectionExpanded(true))
	table, err := NewTableContent("users", []map[string]any{{"name": "Alice"}}, WithKeys("name"))
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	section := NewSectionContent("outer")
	section.AddContent(raw)
	section.AddContent(collapsible)
	section.AddContent(table)

	doc := New().AddContent(section).Build()

	result, err := (&tableRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := string(result)
	if !strings.Contains(out, "nested raw survives") {
		t.Errorf("table output missing raw content inside section, got %q", out)
	}
	if !strings.Contains(out, "=== nested-collapsible ===") {
		t.Errorf("table output missing collapsible section header inside section, got %q", out)
	}
	if !strings.Contains(out, "collapsible body") {
		t.Errorf("table output missing expanded collapsible body inside section, got %q", out)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("table output missing table row after raw and collapsible content, got %q", out)
	}
}

// TestTableRenderer_NestedHeaderTextContent verifies that header-styled text
// nested inside a section renders with the same upper-cased, underlined format
// the top-level loop applies. renderSectionTable and renderDocumentTable share
// a single writeTextContent helper, so a header TextContent must not render as
// plain text just because it sits inside a section (T-1522 review follow-up).
func TestTableRenderer_NestedHeaderTextContent(t *testing.T) {
	header := NewTextContent("deep heading", WithHeader(true))

	inner := NewSectionContent("inner")
	inner.AddContent(header)
	outer := NewSectionContent("outer")
	outer.AddContent(inner)

	doc := New().AddContent(outer).Build()

	result, err := (&tableRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := string(result)
	if !strings.Contains(out, "DEEP HEADING") {
		t.Errorf("nested header text should be upper-cased like the top level, got %q", out)
	}
	if !strings.Contains(out, strings.Repeat("=", len("deep heading"))) {
		t.Errorf("nested header text should be underlined like the top level, got %q", out)
	}
	if strings.Contains(out, "deep heading") {
		t.Errorf("nested header text should not render verbatim (plain, un-styled), got %q", out)
	}
}

// unknownTransformableContent is a Content implementation whose concrete type
// is not handled by the table renderer's content switches, so rendering it
// falls through to the AppendText fallback. It carries transformations so
// tests can verify the fallback renders the transformed content (T-1448).
type unknownTransformableContent struct {
	id         string
	text       string
	transforms []Operation
}

func (c *unknownTransformableContent) Type() ContentType { return ContentType(99) }

func (c *unknownTransformableContent) ID() string { return c.id }

func (c *unknownTransformableContent) Clone() Content {
	clone := *c
	clone.transforms = append([]Operation(nil), c.transforms...)
	return &clone
}

func (c *unknownTransformableContent) GetTransformations() []Operation { return c.transforms }

func (c *unknownTransformableContent) AppendText(b []byte) ([]byte, error) {
	return append(b, c.text...), nil
}

func (c *unknownTransformableContent) AppendBinary(b []byte) ([]byte, error) {
	return append(b, c.text...), nil
}

// replaceTextOp is a test Operation that rewrites the text of an
// unknownTransformableContent, so pre- and post-transform output differ.
type replaceTextOp struct {
	newText string
}

func (op *replaceTextOp) Name() string { return "replace-text" }

func (op *replaceTextOp) Apply(_ context.Context, content Content) (Content, error) {
	c, ok := content.(*unknownTransformableContent)
	if !ok {
		return content, nil
	}
	clone := c.Clone().(*unknownTransformableContent)
	clone.text = op.newText
	return clone, nil
}

func (op *replaceTextOp) CanOptimize(Operation) bool { return false }

func (op *replaceTextOp) Validate() error { return nil }

// TestTableRenderer_FallbackRendersTransformedContent is a regression test for
// T-1448. renderDocumentTable applies per-content transformations, but its
// default (unknown content type) branch rendered the original pre-transform
// content instead of the transformed result, silently discarding the
// transformation. The fallback must render the transformed content.
func TestTableRenderer_FallbackRendersTransformedContent(t *testing.T) {
	content := &unknownTransformableContent{
		id:         "unknown-1",
		text:       "original text",
		transforms: []Operation{&replaceTextOp{newText: "transformed text"}},
	}

	doc := New().AddContent(content).Build()

	result, err := (&tableRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := string(result)
	if !strings.Contains(out, "transformed text") {
		t.Errorf("fallback should render transformed content, got %q", out)
	}
	if strings.Contains(out, "original text") {
		t.Errorf("fallback must not render pre-transform content, got %q", out)
	}
}

// TestTableRenderer_NestedFallbackRendersTransformedContent verifies that the
// section-level fallback in renderSectionTable renders transformed content the
// same way the top level does, so unknown content types behave consistently at
// any nesting depth (T-1448 consistency guard).
func TestTableRenderer_NestedFallbackRendersTransformedContent(t *testing.T) {
	content := &unknownTransformableContent{
		id:         "unknown-nested",
		text:       "original nested text",
		transforms: []Operation{&replaceTextOp{newText: "transformed nested text"}},
	}

	section := NewSectionContent("outer")
	section.AddContent(content)

	doc := New().AddContent(section).Build()

	result, err := (&tableRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := string(result)
	if !strings.Contains(out, "transformed nested text") {
		t.Errorf("nested fallback should render transformed content, got %q", out)
	}
	if strings.Contains(out, "original nested text") {
		t.Errorf("nested fallback must not render pre-transform content, got %q", out)
	}
}
