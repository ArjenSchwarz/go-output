package output

import (
	"testing"
)

func TestBuilder_HeaderMethod(t *testing.T) {
	tests := map[string]struct {
		text     string
		expected string
	}{"empty header": {

		text:     "",
		expected: "",
	}, "header with special characters": {

		text:     "Header: Test & Examples",
		expected: "Header: Test & Examples",
	}, "simple header": {

		text:     "Test Header",
		expected: "Test Header",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			builder := New()
			result := builder.Header(tt.text)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("Header() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a TextContent with header style
			textContent, ok := doc.contents[0].(*TextContent)
			if !ok {
				t.Fatalf("Expected TextContent, got %T", doc.contents[0])
			}

			if textContent.text != tt.expected {
				t.Errorf("text = %q, want %q", textContent.text, tt.expected)
			}

			if !textContent.style.Header {
				t.Error("Header style should be true")
			}
		})
	}
}

// TestBuilder_SectionMethod tests the Section method
