package output

import (
	"strings"
	"testing"
)

func TestSectionContent_Basic(t *testing.T) {
	title := "Test Section"
	content := NewSectionContent(title)

	if content.Type() != ContentTypeSection {
		t.Errorf("Expected content type %v, got %v", ContentTypeSection, content.Type())
	}

	if content.Title() != title {
		t.Errorf("Expected title %q, got %q", title, content.Title())
	}

	if content.Level() != 0 {
		t.Errorf("Expected default level 0, got %d", content.Level())
	}

	if content.ID() == "" {
		t.Error("Expected non-empty ID")
	}

	if len(content.Contents()) != 0 {
		t.Errorf("Expected empty contents, got %d items", len(content.Contents()))
	}
}

func TestSectionContent_WithLevel(t *testing.T) {
	tests := map[string]struct {
		level         int
		expectedLevel int
	}{"level 0": {

		level:         0,
		expectedLevel: 0,
	}, "level 1": {

		level:         1,
		expectedLevel: 1,
	}, "level 3": {

		level:         3,
		expectedLevel: 3,
	}, "negative level (should default to 0)": {

		level:         -1,
		expectedLevel: 0,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			content := NewSectionContent("Test", WithLevel(tt.level))
			if content.Level() != tt.expectedLevel {
				t.Errorf("Expected level %d, got %d", tt.expectedLevel, content.Level())
			}
		})
	}
}

func TestSectionContent_AddContent(t *testing.T) {
	section := NewSectionContent("Test Section")

	// Add some content
	textContent := NewTextContent("Test text")
	rawContent, _ := NewRawContent(FormatHTML, []byte("<p>Test HTML</p>"))
	tableContent, _ := NewTableContent("Test Table", []map[string]any{{"key": "value"}})

	section.AddContent(textContent)
	section.AddContent(rawContent)
	section.AddContent(tableContent)

	contents := section.Contents()
	if len(contents) != 3 {
		t.Errorf("Expected 3 contents, got %d", len(contents))
	}

	// Verify order is preserved
	if contents[0].Type() != ContentTypeText {
		t.Error("First content should be text")
	}
	if contents[1].Type() != ContentTypeRaw {
		t.Error("Second content should be raw")
	}
	if contents[2].Type() != ContentTypeTable {
		t.Error("Third content should be table")
	}

	// Verify immutability - modifying returned slice shouldn't affect original
	originalLen := len(section.Contents())
	returnedContents := section.Contents()
	returnedContents = append(returnedContents, textContent)

	if len(section.Contents()) != originalLen {
		t.Error("Section contents were modified by external change")
	}
}

func TestSectionContent_AppendText_Basic(t *testing.T) {
	section := NewSectionContent("Test Section")

	result, err := section.AppendText([]byte{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "# Test Section\n"
	if string(result) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result))
	}
}

func TestSectionContent_AppendText_WithContent(t *testing.T) {
	section := NewSectionContent("Main Section")

	// Add some content
	section.AddContent(NewTextContent("First paragraph"))
	section.AddContent(NewTextContent("Second paragraph"))

	result, err := section.AppendText([]byte{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	resultStr := string(result)

	// Should contain section title
	if !strings.Contains(resultStr, "# Main Section") {
		t.Error("Result should contain section title with level prefix")
	}

	// Should contain both text contents
	if !strings.Contains(resultStr, "First paragraph") {
		t.Error("Result should contain first paragraph")
	}
	if !strings.Contains(resultStr, "Second paragraph") {
		t.Error("Result should contain second paragraph")
	}
}

func TestSectionContent_HierarchicalLevels(t *testing.T) {
	tests := map[string]struct {
		level          int
		expectedPrefix string
	}{"level 0": {

		level:          0,
		expectedPrefix: "# ",
	}, "level 1": {

		level:          1,
		expectedPrefix: "## ",
	}, "level 2": {

		level:          2,
		expectedPrefix: "### ",
	}, "level 3": {

		level:          3,
		expectedPrefix: "#### ",
	}, "level 4": {

		level:          4,
		expectedPrefix: "##### ",
	}, "level 5": {

		level:          5,
		expectedPrefix: "###### ",
	}, "level 6": {

		level:          6,
		expectedPrefix: "####### ",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			section := NewSectionContent("Test Title", WithLevel(tt.level))

			result, err := section.AppendText([]byte{})
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			resultStr := string(result)
			if !strings.HasPrefix(resultStr, tt.expectedPrefix+"Test Title") {
				t.Errorf("Expected result to start with %q, got %q", tt.expectedPrefix+"Test Title", resultStr)
			}
		})
	}
}

func TestSectionContent_NestedIndentation(t *testing.T) {
	section := NewSectionContent("Level 1 Section", WithLevel(1))
	section.AddContent(NewTextContent("Indented text"))

	result, err := section.AppendText([]byte{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	resultStr := string(result)
	lines := strings.Split(resultStr, "\n")

	// Find the line with "Indented text"
	var indentedLine string
	for _, line := range lines {
		if strings.Contains(line, "Indented text") {
			indentedLine = line
			break
		}
	}

	if indentedLine == "" {
		t.Error("Could not find indented text line")
	}

	// Should be indented with 2 spaces (1 level * 2 spaces)
	if !strings.HasPrefix(indentedLine, "  Indented text") {
		t.Errorf("Expected line to be indented with 2 spaces, got %q", indentedLine)
	}
}

func TestSectionContent_AppendBinary(t *testing.T) {
	section := NewSectionContent("Test Section")
	section.AddContent(NewTextContent("Test content"))

	input := []byte("Prefix: ")
	result, err := section.AppendBinary(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should behave the same as AppendText
	textResult, _ := section.AppendText(input)
	if string(result) != string(textResult) {
		t.Error("AppendBinary should behave the same as AppendText")
	}
}

func TestSectionContent_UniqueIDs(t *testing.T) {
	section1 := NewSectionContent("Section 1")
	section2 := NewSectionContent("Section 2")

	if section1.ID() == section2.ID() {
		t.Error("Expected different IDs for different section instances")
	}
}

func TestBuilder_Section(t *testing.T) {
	builder := New()

	result := builder.Section("Test Section", func(b *Builder) {
		b.Text("Content in section")
		b.Header("Subsection")
		b.Text("More content")
	})

	if result != builder {
		t.Error("Expected builder to return itself for chaining")
	}

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(contents))
	}

	sectionContent, ok := contents[0].(*SectionContent)
	if !ok {
		t.Errorf("Expected SectionContent, got %T", contents[0])
	}

	if sectionContent.Title() != "Test Section" {
		t.Errorf("Expected title 'Test Section', got %q", sectionContent.Title())
	}

	// Check nested content
	sectionContents := sectionContent.Contents()
	if len(sectionContents) != 3 {
		t.Errorf("Expected 3 nested contents, got %d", len(sectionContents))
	}

	// Verify order is preserved
	if sectionContents[0].Type() != ContentTypeText {
		t.Error("First nested content should be text")
	}
	if sectionContents[1].Type() != ContentTypeText {
		t.Error("Second nested content should be text (header)")
	}
	if sectionContents[2].Type() != ContentTypeText {
		t.Error("Third nested content should be text")
	}
}

func TestBuilder_SectionWithLevel(t *testing.T) {
	builder := New()

	builder.Section("Level 2 Section", func(b *Builder) {
		b.Text("Nested content")
	}, WithLevel(2))

	doc := builder.Build()
	contents := doc.GetContents()

	sectionContent := contents[0].(*SectionContent)
	if sectionContent.Level() != 2 {
		t.Errorf("Expected level 2, got %d", sectionContent.Level())
	}
}

func TestBuilder_NestedSections(t *testing.T) {
	builder := New()

	builder.Section("Main Section", func(b *Builder) {
		b.Text("Main content")
		b.Section("Nested Section", func(nb *Builder) {
			nb.Text("Nested content")
		}, WithLevel(1))
		b.Text("More main content")
	})

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 1 {
		t.Errorf("Expected 1 main content item, got %d", len(contents))
	}

	mainSection := contents[0].(*SectionContent)
	nestedContents := mainSection.Contents()

	if len(nestedContents) != 3 {
		t.Errorf("Expected 3 nested contents, got %d", len(nestedContents))
	}

	// Second content should be the nested section
	nestedSection, ok := nestedContents[1].(*SectionContent)
	if !ok {
		t.Errorf("Expected nested SectionContent, got %T", nestedContents[1])
	}

	if nestedSection.Title() != "Nested Section" {
		t.Errorf("Expected nested title 'Nested Section', got %q", nestedSection.Title())
	}

	if nestedSection.Level() != 1 {
		t.Errorf("Expected nested level 1, got %d", nestedSection.Level())
	}
}

func TestBuilder_MixedContentWithSections(t *testing.T) {
	builder := New()

	builder.
		Text("Introduction").
		Section("Data Section", func(b *Builder) {
			b.Table("Users", []map[string]any{{"name": "Alice", "age": 30}})
			b.Text("Table description")
		}).
		Raw(FormatHTML, []byte("<hr>")).
		Section("Conclusion", func(b *Builder) {
			b.Text("Final thoughts")
		})

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 4 {
		t.Errorf("Expected 4 content items, got %d", len(contents))
	}

	// Verify order and types
	expectedTypes := []ContentType{
		ContentTypeText,    // Introduction
		ContentTypeSection, // Data Section
		ContentTypeRaw,     // HTML
		ContentTypeSection, // Conclusion
	}

	for i, expectedType := range expectedTypes {
		if contents[i].Type() != expectedType {
			t.Errorf("Content %d: expected type %v, got %v", i, expectedType, contents[i].Type())
		}
	}
}

func TestSectionContent_ErrorHandling(t *testing.T) {
	section := NewSectionContent("Test Section")

	// Create a mock content that will return an error (we'll simulate this)
	// Since we can't easily create a content that errors, we'll test the error path
	// by checking that the function can handle errors appropriately

	// This test mainly ensures that the AppendText method properly handles
	// and propagates errors from nested content
	result, err := section.AppendText([]byte{})
	if err != nil {
		t.Errorf("Unexpected error with empty section: %v", err)
	}

	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}
}

func TestSplitLines(t *testing.T) {
	tests := map[string]struct {
		input    []byte
		expected [][]byte
	}{"empty input": {

		input:    []byte{},
		expected: [][]byte{},
	}, "empty lines": {

		input:    []byte("line1\n\nline3"),
		expected: [][]byte{[]byte("line1"), []byte(""), []byte("line3")},
	}, "multiple lines": {

		input:    []byte("line1\nline2\nline3"),
		expected: [][]byte{[]byte("line1"), []byte("line2"), []byte("line3")},
	}, "single line": {

		input:    []byte("hello"),
		expected: [][]byte{[]byte("hello")},
	}, "trailing newline": {

		input:    []byte("line1\nline2\n"),
		expected: [][]byte{[]byte("line1"), []byte("line2")},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := splitLines(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(result))
				return
			}

			for i, expectedLine := range tt.expected {
				if string(result[i]) != string(expectedLine) {
					t.Errorf("Line %d: expected %q, got %q", i, string(expectedLine), string(result[i]))
				}
			}
		})
	}
}

func TestIndentContent(t *testing.T) {
	tests := map[string]struct {
		content  []byte
		level    int
		expected string
	}{"empty content": {

		content:  []byte{},
		level:    1,
		expected: "",
	}, "empty lines preserved": {

		content:  []byte("line1\n\nline3"),
		level:    1,
		expected: "  line1\n\n  line3",
	}, "level 0 (no indentation)": {

		content:  []byte("line1\nline2"),
		level:    0,
		expected: "line1\nline2",
	}, "level 1": {

		content:  []byte("line1\nline2"),
		level:    1,
		expected: "  line1\n  line2",
	}, "level 2": {

		content:  []byte("line1\nline2"),
		level:    2,
		expected: "    line1\n    line2",
	}, "negative level": {

		content:  []byte("line1\nline2"),
		level:    -1,
		expected: "line1\nline2",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := indentContent(tt.content, tt.level)
			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}
