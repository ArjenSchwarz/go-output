package output

import (
	"context"
	"testing"
)

func TestTextContent_Basic(t *testing.T) {
	text := "Hello, World!"
	content := NewTextContent(text)

	if content.Type() != ContentTypeText {
		t.Errorf("Expected content type %v, got %v", ContentTypeText, content.Type())
	}

	if content.Text() != text {
		t.Errorf("Expected text %q, got %q", text, content.Text())
	}

	if content.ID() == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestTextContent_WithStyles(t *testing.T) {
	tests := map[string]struct {
		text     string
		options  []TextOption
		expected TextStyle
	}{"with bold": {

		text:    "Bold text",
		options: []TextOption{WithBold(true)},
		expected: TextStyle{
			Bold:   true,
			Italic: false,
			Color:  "",
			Size:   0,
			Header: false,
		},
	}, "with color": {

		text:    "Colored text",
		options: []TextOption{WithColor("red")},
		expected: TextStyle{
			Bold:   false,
			Italic: false,
			Color:  "red",
			Size:   0,
			Header: false,
		},
	}, "with complete style": {

		text:    "Complete style text",
		options: []TextOption{WithTextStyle(TextStyle{Bold: true, Italic: true, Color: "green", Size: 18, Header: true})},
		expected: TextStyle{
			Bold:   true,
			Italic: true,
			Color:  "green",
			Size:   18,
			Header: true,
		},
	}, "with header": {

		text:    "Header text",
		options: []TextOption{WithHeader(true)},
		expected: TextStyle{
			Bold:   false,
			Italic: false,
			Color:  "",
			Size:   0,
			Header: true,
		},
	}, "with italic": {

		text:    "Italic text",
		options: []TextOption{WithItalic(true)},
		expected: TextStyle{
			Bold:   false,
			Italic: true,
			Color:  "",
			Size:   0,
			Header: false,
		},
	}, "with multiple styles": {

		text: "Multi-styled text",
		options: []TextOption{
			WithBold(true),
			WithItalic(true),
			WithColor("blue"),
			WithSize(16),
		},
		expected: TextStyle{
			Bold:   true,
			Italic: true,
			Color:  "blue",
			Size:   16,
			Header: false,
		},
	}, "with size": {

		text:    "Sized text",
		options: []TextOption{WithSize(14)},
		expected: TextStyle{
			Bold:   false,
			Italic: false,
			Color:  "",
			Size:   14,
			Header: false,
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			content := NewTextContent(tt.text, tt.options...)

			style := content.Style()
			if style != tt.expected {
				t.Errorf("Expected style %+v, got %+v", tt.expected, style)
			}

			if content.Text() != tt.text {
				t.Errorf("Expected text %q, got %q", tt.text, content.Text())
			}
		})
	}
}

func TestTextContent_AppendText(t *testing.T) {
	tests := map[string]struct {
		text     string
		input    []byte
		expected string
	}{"empty input": {

		text:     "Hello",
		input:    []byte{},
		expected: "Hello\n",
	}, "multiline text": {

		text:     "Line 1\nLine 2",
		input:    []byte{},
		expected: "Line 1\nLine 2\n",
	}, "text already ending with newline": {

		text:     "Text with newline\n",
		input:    []byte{},
		expected: "Text with newline\n",
	}, "with existing content": {

		text:     "World",
		input:    []byte("Hello "),
		expected: "Hello World\n",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			content := NewTextContent(tt.text)
			result, err := content.AppendText(tt.input)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestTextContent_AppendBinary(t *testing.T) {
	text := "Binary test"
	content := NewTextContent(text)
	input := []byte("Prefix ")

	result, err := content.AppendBinary(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "Prefix Binary test\n"
	if string(result) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result))
	}
}

func TestTextContent_UniqueIDs(t *testing.T) {
	content1 := NewTextContent("Text 1")
	content2 := NewTextContent("Text 2")

	if content1.ID() == content2.ID() {
		t.Error("Expected different IDs for different content instances")
	}
}

func TestBuilder_Text(t *testing.T) {
	builder := New()
	text := "Test text"

	result := builder.Text(text, WithBold(true), WithColor("red"))

	if result != builder {
		t.Error("Expected builder to return itself for chaining")
	}

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(contents))
	}

	textContent, ok := contents[0].(*TextContent)
	if !ok {
		t.Errorf("Expected TextContent, got %T", contents[0])
	}

	if textContent.Text() != text {
		t.Errorf("Expected text %q, got %q", text, textContent.Text())
	}

	style := textContent.Style()
	if !style.Bold {
		t.Error("Expected bold style to be true")
	}

	if style.Color != "red" {
		t.Errorf("Expected color 'red', got %q", style.Color)
	}
}

func TestBuilder_Header(t *testing.T) {
	builder := New()
	headerText := "Header Text"

	result := builder.Header(headerText)

	if result != builder {
		t.Error("Expected builder to return itself for chaining")
	}

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(contents))
	}

	textContent, ok := contents[0].(*TextContent)
	if !ok {
		t.Errorf("Expected TextContent, got %T", contents[0])
	}

	if textContent.Text() != headerText {
		t.Errorf("Expected text %q, got %q", headerText, textContent.Text())
	}

	style := textContent.Style()
	if !style.Header {
		t.Error("Expected header style to be true")
	}
}

func TestBuilder_MultipleTextContents(t *testing.T) {
	builder := New()

	builder.
		Header("Title").
		Text("Normal text").
		Text("Bold text", WithBold(true)).
		Text("Colored text", WithColor("blue"))

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 4 {
		t.Errorf("Expected 4 content items, got %d", len(contents))
	}

	// Check header
	header, ok := contents[0].(*TextContent)
	if !ok || !header.Style().Header {
		t.Error("First content should be a header")
	}

	// Check normal text
	normal, ok := contents[1].(*TextContent)
	if !ok || normal.Text() != "Normal text" {
		t.Error("Second content should be normal text")
	}

	// Check bold text
	bold, ok := contents[2].(*TextContent)
	if !ok || !bold.Style().Bold {
		t.Error("Third content should be bold text")
	}

	// Check colored text
	colored, ok := contents[3].(*TextContent)
	if !ok || colored.Style().Color != "blue" {
		t.Error("Fourth content should be blue colored text")
	}
}

func TestTextOptions_OverrideOrder(t *testing.T) {
	// Test that later options override earlier ones
	content := NewTextContent("Test",
		WithBold(true),
		WithBold(false), // Should override previous
		WithColor("red"),
		WithColor("blue"), // Should override previous
	)

	style := content.Style()
	if style.Bold {
		t.Error("Expected bold to be false (overridden)")
	}

	if style.Color != "blue" {
		t.Errorf("Expected color 'blue' (overridden), got %q", style.Color)
	}
}

// Mock operation for testing transformation storage
type mockOperation struct {
	name string
}

func (m *mockOperation) Name() string {
	return m.name
}

func (m *mockOperation) Apply(ctx context.Context, content Content) (Content, error) {
	return content, nil
}

func (m *mockOperation) CanOptimize(with Operation) bool {
	return false
}

func (m *mockOperation) Validate() error {
	return nil
}

func TestTextContent_WithTransformations(t *testing.T) {
	tests := map[string]struct {
		text            string
		transformations []Operation
		wantCount       int
	}{
		"no transformations": {
			text:            "Test text",
			transformations: nil,
			wantCount:       0,
		},
		"single transformation": {
			text: "Test text",
			transformations: []Operation{
				&mockOperation{name: "transform1"},
			},
			wantCount: 1,
		},
		"multiple transformations": {
			text: "Test text",
			transformations: []Operation{
				&mockOperation{name: "transform1"},
				&mockOperation{name: "transform2"},
				&mockOperation{name: "transform3"},
			},
			wantCount: 3,
		},
		"empty transformations slice": {
			text:            "Test text",
			transformations: []Operation{},
			wantCount:       0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var content *TextContent
			if tc.transformations != nil {
				content = NewTextContent(tc.text, WithTextTransformations(tc.transformations...))
			} else {
				content = NewTextContent(tc.text)
			}

			got := content.GetTransformations()
			if len(got) != tc.wantCount {
				t.Errorf("Expected %d transformations, got %d", tc.wantCount, len(got))
			}
		})
	}
}

func TestTextContent_GetTransformations(t *testing.T) {
	tests := map[string]struct {
		text            string
		transformations []Operation
		wantNames       []string
	}{
		"returns empty slice when no transformations": {
			text:            "Test",
			transformations: nil,
			wantNames:       []string{},
		},
		"returns all transformations in order": {
			text: "Test",
			transformations: []Operation{
				&mockOperation{name: "filter"},
				&mockOperation{name: "sort"},
				&mockOperation{name: "limit"},
			},
			wantNames: []string{"filter", "sort", "limit"},
		},
		"returns single transformation": {
			text: "Test",
			transformations: []Operation{
				&mockOperation{name: "transform"},
			},
			wantNames: []string{"transform"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var content *TextContent
			if tc.transformations != nil {
				content = NewTextContent(tc.text, WithTextTransformations(tc.transformations...))
			} else {
				content = NewTextContent(tc.text)
			}

			got := content.GetTransformations()
			if len(got) != len(tc.wantNames) {
				t.Errorf("Expected %d transformations, got %d", len(tc.wantNames), len(got))
			}

			for i, op := range got {
				if op.Name() != tc.wantNames[i] {
					t.Errorf("Transformation %d: expected name %q, got %q", i, tc.wantNames[i], op.Name())
				}
			}
		})
	}
}

func TestTextContent_Clone_PreservesTransformations(t *testing.T) {
	tests := map[string]struct {
		text            string
		transformations []Operation
		wantCount       int
	}{
		"clone with no transformations": {
			text:            "Test",
			transformations: nil,
			wantCount:       0,
		},
		"clone with single transformation": {
			text: "Test",
			transformations: []Operation{
				&mockOperation{name: "transform1"},
			},
			wantCount: 1,
		},
		"clone with multiple transformations": {
			text: "Test",
			transformations: []Operation{
				&mockOperation{name: "transform1"},
				&mockOperation{name: "transform2"},
				&mockOperation{name: "transform3"},
			},
			wantCount: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var original *TextContent
			if tc.transformations != nil {
				original = NewTextContent(tc.text, WithTextTransformations(tc.transformations...))
			} else {
				original = NewTextContent(tc.text)
			}

			cloned := original.Clone()

			// Verify cloned content has transformations
			clonedTransformations := cloned.GetTransformations()
			if len(clonedTransformations) != tc.wantCount {
				t.Errorf("Cloned content: expected %d transformations, got %d", tc.wantCount, len(clonedTransformations))
			}

			// Verify original still has transformations
			originalTransformations := original.GetTransformations()
			if len(originalTransformations) != tc.wantCount {
				t.Errorf("Original content: expected %d transformations, got %d", tc.wantCount, len(originalTransformations))
			}

			// Verify they reference the same operation instances (shallow copy)
			if tc.wantCount > 0 {
				for i := range originalTransformations {
					if originalTransformations[i] != clonedTransformations[i] {
						t.Errorf("Transformation %d: expected same instance after clone", i)
					}
				}
			}
		})
	}
}

func TestTextContent_TransformationOrderPreservation(t *testing.T) {
	tests := map[string]struct {
		text      string
		ops       []Operation
		wantOrder []string
	}{
		"preserves order of two operations": {
			text: "Test",
			ops: []Operation{
				&mockOperation{name: "first"},
				&mockOperation{name: "second"},
			},
			wantOrder: []string{"first", "second"},
		},
		"preserves order of multiple operations": {
			text: "Test",
			ops: []Operation{
				&mockOperation{name: "alpha"},
				&mockOperation{name: "beta"},
				&mockOperation{name: "gamma"},
				&mockOperation{name: "delta"},
			},
			wantOrder: []string{"alpha", "beta", "gamma", "delta"},
		},
		"preserves order through clone": {
			text: "Test",
			ops: []Operation{
				&mockOperation{name: "op1"},
				&mockOperation{name: "op2"},
				&mockOperation{name: "op3"},
			},
			wantOrder: []string{"op1", "op2", "op3"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			content := NewTextContent(tc.text, WithTextTransformations(tc.ops...))

			// Verify order in original
			got := content.GetTransformations()
			for i, op := range got {
				if op.Name() != tc.wantOrder[i] {
					t.Errorf("Original: position %d expected %q, got %q", i, tc.wantOrder[i], op.Name())
				}
			}

			// Verify order is preserved after clone
			cloned := content.Clone()
			clonedOps := cloned.GetTransformations()
			for i, op := range clonedOps {
				if op.Name() != tc.wantOrder[i] {
					t.Errorf("Cloned: position %d expected %q, got %q", i, tc.wantOrder[i], op.Name())
				}
			}
		})
	}
}
