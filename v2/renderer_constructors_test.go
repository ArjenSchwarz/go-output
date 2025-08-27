package output

import (
	"reflect"
	"testing"
)

// TestNewMarkdownRendererWithToC tests the markdown renderer with TOC constructor
func TestNewMarkdownRendererWithToC(t *testing.T) {
	tests := map[string]struct {
		enabled bool
		wantToC bool
	}{"TOC disabled": {

		enabled: false,
		wantToC: false,
	}, "TOC enabled": {

		enabled: true,
		wantToC: true,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			renderer := NewMarkdownRendererWithToC(tt.enabled)

			if renderer == nil {
				t.Fatal("NewMarkdownRendererWithToC returned nil")
			}

			if renderer.includeToC != tt.wantToC {
				t.Errorf("includeToC = %v, want %v", renderer.includeToC, tt.wantToC)
			}

			// Should have nil frontMatter when using this constructor
			if renderer.frontMatter != nil {
				t.Error("frontMatter should be nil when using NewMarkdownRendererWithToC")
			}
		})
	}
}

// TestNewMarkdownRendererWithFrontMatter tests the markdown renderer with front matter constructor
func TestNewMarkdownRendererWithFrontMatter(t *testing.T) {
	tests := map[string]struct {
		frontMatter map[string]string
	}{"empty front matter": {

		frontMatter: map[string]string{},
	}, "nil front matter": {

		frontMatter: nil,
	}, "with front matter": {

		frontMatter: map[string]string{
			"title":  "Test Document",
			"author": "Test Author",
			"date":   "2024-01-01",
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			renderer := NewMarkdownRendererWithFrontMatter(tt.frontMatter)

			if renderer == nil {
				t.Fatal("NewMarkdownRendererWithFrontMatter returned nil")
			}

			// Check front matter matches
			if !reflect.DeepEqual(renderer.frontMatter, tt.frontMatter) {
				t.Errorf("frontMatter = %v, want %v", renderer.frontMatter, tt.frontMatter)
			}

			// Should have TOC disabled when using this constructor
			if renderer.includeToC {
				t.Error("includeToC should be false when using NewMarkdownRendererWithFrontMatter")
			}
		})
	}
}

// TestNewMarkdownRendererWithOptions tests the full options constructor
func TestNewMarkdownRendererWithOptions(t *testing.T) {
	tests := map[string]struct {
		includeToC  bool
		frontMatter map[string]string
	}{"TOC only": {

		includeToC:  true,
		frontMatter: nil,
	}, "front matter only": {

		includeToC: false,
		frontMatter: map[string]string{
			"author": "Test",
		},
	}, "neither TOC nor front matter": {

		includeToC:  false,
		frontMatter: nil,
	}, "with both TOC and front matter": {

		includeToC: true,
		frontMatter: map[string]string{
			"title": "Full Document",
			"toc":   "true",
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			renderer := NewMarkdownRendererWithOptions(tt.includeToC, tt.frontMatter)

			if renderer == nil {
				t.Fatal("NewMarkdownRendererWithOptions returned nil")
			}

			if renderer.includeToC != tt.includeToC {
				t.Errorf("includeToC = %v, want %v", renderer.includeToC, tt.includeToC)
			}

			if !reflect.DeepEqual(renderer.frontMatter, tt.frontMatter) {
				t.Errorf("frontMatter = %v, want %v", renderer.frontMatter, tt.frontMatter)
			}
		})
	}
}

// TestNewTableRendererWithStyle tests the table renderer with style constructor
func TestNewTableRendererWithStyle(t *testing.T) {
	tests := map[string]struct {
		styleName string
	}{"bold style": {

		styleName: "Bold",
	}, "colored bright style": {

		styleName: "ColoredBright",
	}, "custom style": {

		styleName: "CustomStyle",
	}, "default style": {

		styleName: "Default",
	}, "empty style": {

		styleName: "",
	}, "light style": {

		styleName: "Light",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			renderer := NewTableRendererWithStyle(tt.styleName)

			if renderer == nil {
				t.Fatal("NewTableRendererWithStyle returned nil")
			}

			tableRenderer := renderer.(*tableRenderer)
			if tableRenderer.StyleName() != tt.styleName {
				t.Errorf("styleName = %q, want %q", tableRenderer.StyleName(), tt.styleName)
			}
		})
	}
}

// TestMarkdownRendererCombinations tests various combinations
func TestMarkdownRendererCombinations(t *testing.T) {
	// Test that different constructors create independent instances
	r1 := NewMarkdownRendererWithToC(true)
	r2 := NewMarkdownRendererWithToC(false)
	r3 := NewMarkdownRendererWithFrontMatter(map[string]string{"key": "value"})

	// Verify they're different instances
	if r1 == r2 || r1 == r3 || r2 == r3 {
		t.Error("Constructors should create independent instances")
	}

	// Verify their settings are independent
	if r1.includeToC == r2.includeToC {
		t.Error("Different TOC settings should be preserved")
	}

	if r3.frontMatter == nil {
		t.Error("Front matter should be preserved in r3")
	}
}

// TestTableRendererStyles verifies common style names work
func TestTableRendererStyles(t *testing.T) {
	// Common style names from go-pretty library
	commonStyles := []string{
		"Default",
		"Light",
		"Bold",
		"ColoredBright",
		"ColoredDark",
		"ColoredBlackOnBlue",
		"ColoredBlackOnCyan",
		"ColoredBlackOnGreen",
		"ColoredBlackOnMagenta",
		"ColoredBlackOnRed",
		"ColoredBlackOnYellow",
		"ColoredBlueOnBlack",
		"ColoredCyanOnBlack",
		"ColoredGreenOnBlack",
		"ColoredMagentaOnBlack",
		"ColoredRedOnBlack",
		"ColoredYellowOnBlack",
		"Double",
		"Rounded",
	}

	for _, style := range commonStyles {
		t.Run(style, func(t *testing.T) {
			renderer := NewTableRendererWithStyle(style)
			if renderer == nil {
				t.Errorf("NewTableRendererWithStyle(%q) returned nil", style)
			}
			tableRenderer := renderer.(*tableRenderer)
			if tableRenderer.StyleName() != style {
				t.Errorf("Expected style %q, got %q", style, tableRenderer.StyleName())
			}
		})
	}
}

// TestRendererInterfaces verifies renderers implement required interfaces
func TestRendererInterfaces(t *testing.T) {
	// Test markdown renderer
	mr1 := NewMarkdownRendererWithToC(true)
	mr2 := NewMarkdownRendererWithFrontMatter(map[string]string{})
	mr3 := NewMarkdownRendererWithOptions(false, nil)

	// Test table renderer
	tr := NewTableRendererWithStyle("Default")

	// All should implement Renderer interface
	renderers := []Renderer{mr1, mr2, mr3, tr}

	for i, r := range renderers {
		if r == nil {
			t.Errorf("Renderer %d is nil", i)
			continue
		}

		// Test basic interface methods
		format := r.Format()
		if format == "" {
			t.Errorf("Renderer %d returned empty format", i)
		}

		// SupportsStreaming should return a boolean
		_ = r.SupportsStreaming()
	}
}
