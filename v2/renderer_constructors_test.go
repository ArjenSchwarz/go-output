package output

import (
	"reflect"
	"testing"
)

// TestNewMarkdownRendererWithToC tests the markdown renderer with TOC constructor
func TestNewMarkdownRendererWithToC(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		wantToC bool
	}{
		{
			name:    "TOC enabled",
			enabled: true,
			wantToC: true,
		},
		{
			name:    "TOC disabled",
			enabled: false,
			wantToC: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	tests := []struct {
		name        string
		frontMatter map[string]string
	}{
		{
			name: "with front matter",
			frontMatter: map[string]string{
				"title":  "Test Document",
				"author": "Test Author",
				"date":   "2024-01-01",
			},
		},
		{
			name:        "empty front matter",
			frontMatter: map[string]string{},
		},
		{
			name:        "nil front matter",
			frontMatter: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	tests := []struct {
		name        string
		includeToC  bool
		frontMatter map[string]string
	}{
		{
			name:       "with both TOC and front matter",
			includeToC: true,
			frontMatter: map[string]string{
				"title": "Full Document",
				"toc":   "true",
			},
		},
		{
			name:        "TOC only",
			includeToC:  true,
			frontMatter: nil,
		},
		{
			name:       "front matter only",
			includeToC: false,
			frontMatter: map[string]string{
				"author": "Test",
			},
		},
		{
			name:        "neither TOC nor front matter",
			includeToC:  false,
			frontMatter: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	tests := []struct {
		name      string
		styleName string
	}{
		{
			name:      "default style",
			styleName: "Default",
		},
		{
			name:      "light style",
			styleName: "Light",
		},
		{
			name:      "bold style",
			styleName: "Bold",
		},
		{
			name:      "colored bright style",
			styleName: "ColoredBright",
		},
		{
			name:      "empty style",
			styleName: "",
		},
		{
			name:      "custom style",
			styleName: "CustomStyle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewTableRendererWithStyle(tt.styleName)

			if renderer == nil {
				t.Fatal("NewTableRendererWithStyle returned nil")
			}

			if renderer.styleName != tt.styleName {
				t.Errorf("styleName = %q, want %q", renderer.styleName, tt.styleName)
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
			if renderer.styleName != style {
				t.Errorf("Expected style %q, got %q", style, renderer.styleName)
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
