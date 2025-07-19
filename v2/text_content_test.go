package output

import (
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
	tests := []struct {
		name     string
		text     string
		options  []TextOption
		expected TextStyle
	}{
		{
			name:    "with bold",
			text:    "Bold text",
			options: []TextOption{WithBold(true)},
			expected: TextStyle{
				Bold:   true,
				Italic: false,
				Color:  "",
				Size:   0,
				Header: false,
			},
		},
		{
			name:    "with italic",
			text:    "Italic text",
			options: []TextOption{WithItalic(true)},
			expected: TextStyle{
				Bold:   false,
				Italic: true,
				Color:  "",
				Size:   0,
				Header: false,
			},
		},
		{
			name:    "with color",
			text:    "Colored text",
			options: []TextOption{WithColor("red")},
			expected: TextStyle{
				Bold:   false,
				Italic: false,
				Color:  "red",
				Size:   0,
				Header: false,
			},
		},
		{
			name:    "with size",
			text:    "Sized text",
			options: []TextOption{WithSize(14)},
			expected: TextStyle{
				Bold:   false,
				Italic: false,
				Color:  "",
				Size:   14,
				Header: false,
			},
		},
		{
			name:    "with header",
			text:    "Header text",
			options: []TextOption{WithHeader(true)},
			expected: TextStyle{
				Bold:   false,
				Italic: false,
				Color:  "",
				Size:   0,
				Header: true,
			},
		},
		{
			name: "with multiple styles",
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
		},
		{
			name:    "with complete style",
			text:    "Complete style text",
			options: []TextOption{WithTextStyle(TextStyle{Bold: true, Italic: true, Color: "green", Size: 18, Header: true})},
			expected: TextStyle{
				Bold:   true,
				Italic: true,
				Color:  "green",
				Size:   18,
				Header: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	tests := []struct {
		name     string
		text     string
		input    []byte
		expected string
	}{
		{
			name:     "empty input",
			text:     "Hello",
			input:    []byte{},
			expected: "Hello\n",
		},
		{
			name:     "with existing content",
			text:     "World",
			input:    []byte("Hello "),
			expected: "Hello World\n",
		},
		{
			name:     "multiline text",
			text:     "Line 1\nLine 2",
			input:    []byte{},
			expected: "Line 1\nLine 2\n",
		},
		{
			name:     "text already ending with newline",
			text:     "Text with newline\n",
			input:    []byte{},
			expected: "Text with newline\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
