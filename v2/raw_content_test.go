package output

import (
	"testing"
)

func TestRawContent_Basic(t *testing.T) {
	format := FormatHTML
	data := []byte("<div>Hello, World!</div>")

	content, err := NewRawContent(format, data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if content.Type() != ContentTypeRaw {
		t.Errorf("Expected content type %v, got %v", ContentTypeRaw, content.Type())
	}

	if content.Format() != format {
		t.Errorf("Expected format %q, got %q", format, content.Format())
	}

	if content.ID() == "" {
		t.Error("Expected non-empty ID")
	}

	// Test data immutability
	returnedData := content.Data()
	if string(returnedData) != string(data) {
		t.Errorf("Expected data %q, got %q", string(data), string(returnedData))
	}

	// Modify the returned data and ensure original is unchanged
	returnedData[0] = 'X'
	if string(content.Data()) != string(data) {
		t.Error("Original data was modified, immutability broken")
	}
}

func TestRawContent_FormatValidation(t *testing.T) {
	tests := map[string]struct {
		format      string
		data        []byte
		opts        []RawOption
		expectError bool
	}{"default validation (enabled)": {

		format:      "invalid",
		data:        []byte("some data"),
		opts:        []RawOption{},
		expectError: true,
	}, "invalid format with validation": {

		format:      "invalid",
		data:        []byte("some data"),
		opts:        []RawOption{WithFormatValidation(true)},
		expectError: true,
	}, "invalid format without validation": {

		format:      "invalid",
		data:        []byte("some data"),
		opts:        []RawOption{WithFormatValidation(false)},
		expectError: false,
	}, "valid html format": {

		format:      FormatHTML,
		data:        []byte("<p>test</p>"),
		opts:        []RawOption{WithFormatValidation(true)},
		expectError: false,
	}, "valid json format": {

		format:      FormatJSON,
		data:        []byte(`{"key": "value"}`),
		opts:        []RawOption{WithFormatValidation(true)},
		expectError: false,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewRawContent(tt.format, tt.data, tt.opts...)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRawContent_ValidFormats(t *testing.T) {
	validFormats := []string{
		FormatHTML, "css", "js", FormatJSON, "xml", FormatYAML,
		FormatMarkdown, FormatText, FormatCSV, FormatDOT, FormatMermaid, FormatDrawIO, "svg",
	}

	data := []byte("test data")

	for _, format := range validFormats {
		t.Run(format, func(t *testing.T) {
			_, err := NewRawContent(format, data)
			if err != nil {
				t.Errorf("Format %q should be valid, got error: %v", format, err)
			}
		})
	}
}

func TestRawContent_DataPreservation(t *testing.T) {
	originalData := []byte("original data")
	format := FormatText

	// Test with data preservation enabled (default)
	content1, err := NewRawContent(format, originalData, WithDataPreservation(true))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Modify original data
	originalData[0] = 'X'

	// Content should have preserved original data
	if string(content1.Data()) == string(originalData) {
		t.Error("Data preservation failed - original data modification affected content")
	}

	// Reset original data
	originalData[0] = 'o'

	// Test with data preservation disabled
	content2, err := NewRawContent(format, originalData, WithDataPreservation(false))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Even with preservation disabled, data should still be copied for safety
	originalData[0] = 'Y'
	if string(content2.Data()) == string(originalData) {
		t.Error("Data should still be copied even when preservation is disabled")
	}
}

func TestRawContent_AppendText(t *testing.T) {
	tests := map[string]struct {
		format   string
		data     []byte
		input    []byte
		expected string
	}{"empty data": {

		format:   FormatText,
		data:     []byte{},
		input:    []byte("Start: "),
		expected: "Start: ",
	}, "html content": {

		format:   FormatHTML,
		data:     []byte("<div>Hello</div>"),
		input:    []byte("Prefix: "),
		expected: "Prefix: <div>Hello</div>",
	}, "json content": {

		format:   FormatJSON,
		data:     []byte(`{"key": "value"}`),
		input:    []byte{},
		expected: `{"key": "value"}`,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			content, err := NewRawContent(tt.format, tt.data)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

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

func TestRawContent_AppendBinary(t *testing.T) {
	format := "svg"
	data := []byte("<svg>content</svg>")
	content, err := NewRawContent(format, data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	input := []byte("Binary prefix: ")
	result, err := content.AppendBinary(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "Binary prefix: <svg>content</svg>"
	if string(result) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result))
	}
}

func TestRawContent_UniqueIDs(t *testing.T) {
	format := FormatHTML
	data := []byte("<p>test</p>")

	content1, err := NewRawContent(format, data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	content2, err := NewRawContent(format, data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if content1.ID() == content2.ID() {
		t.Error("Expected different IDs for different content instances")
	}
}

func TestBuilder_Raw(t *testing.T) {
	builder := New()
	format := FormatHTML
	data := []byte("<div>Test content</div>")

	result := builder.Raw(format, data)

	if result != builder {
		t.Error("Expected builder to return itself for chaining")
	}

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(contents))
	}

	rawContent, ok := contents[0].(*RawContent)
	if !ok {
		t.Errorf("Expected RawContent, got %T", contents[0])
	}

	if rawContent.Format() != format {
		t.Errorf("Expected format %q, got %q", format, rawContent.Format())
	}

	if string(rawContent.Data()) != string(data) {
		t.Errorf("Expected data %q, got %q", string(data), string(rawContent.Data()))
	}
}

func TestBuilder_RawWithInvalidFormat(t *testing.T) {
	builder := New()
	format := "invalid"
	data := []byte("some data")

	// Should not add invalid content but should not panic
	result := builder.Raw(format, data)

	if result != builder {
		t.Error("Expected builder to return itself for chaining even with invalid content")
	}

	doc := builder.Build()
	contents := doc.GetContents()

	// Should have no contents since invalid format was rejected
	if len(contents) != 0 {
		t.Errorf("Expected 0 content items, got %d", len(contents))
	}
}

func TestBuilder_RawWithOptions(t *testing.T) {
	builder := New()
	format := "custom" // Invalid format
	data := []byte("custom data")

	// Disable validation to allow custom format
	result := builder.Raw(format, data, WithFormatValidation(false))

	if result != builder {
		t.Error("Expected builder to return itself for chaining")
	}

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(contents))
	}

	rawContent, ok := contents[0].(*RawContent)
	if !ok {
		t.Errorf("Expected RawContent, got %T", contents[0])
	}

	if rawContent.Format() != format {
		t.Errorf("Expected format %q, got %q", format, rawContent.Format())
	}
}

func TestBuilder_MixedContentWithRaw(t *testing.T) {
	builder := New()

	builder.
		Text("Introduction").
		Raw(FormatHTML, []byte("<div class='content'>HTML content</div>")).
		Table("Data", []map[string]any{{"key": "value"}}).
		Raw("css", []byte(".content { color: blue; }"))

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 4 {
		t.Errorf("Expected 4 content items, got %d", len(contents))
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

	if contents[3].Type() != ContentTypeRaw {
		t.Error("Fourth content should be raw")
	}

	// Check raw content formats
	htmlContent := contents[1].(*RawContent)
	if htmlContent.Format() != FormatHTML {
		t.Errorf("Expected HTML format, got %q", htmlContent.Format())
	}

	cssContent := contents[3].(*RawContent)
	if cssContent.Format() != "css" {
		t.Errorf("Expected CSS format, got %q", cssContent.Format())
	}
}
