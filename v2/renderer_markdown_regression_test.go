package output

import (
	"strings"
	"testing"
)

// TestMarkdownRenderer_StyledTextContentEscaping is a regression test for
// T-2030: when Color or Size is set, the Markdown renderer wraps the text in a
// raw <span>. The text content must be HTML-escaped there (matching the HTML
// renderer) so hostile text cannot inject live markup, while the literal text
// is preserved for the reader.
func TestMarkdownRenderer_StyledTextContentEscaping(t *testing.T) {
	tests := map[string]struct {
		text    string
		style   TextStyle
		want    string
		notWant string
	}{
		"script tag with color": {
			text:    `<script>alert(1)</script>`,
			style:   TextStyle{Color: "red"},
			want:    `<span style="color: red">&lt;script&gt;alert(1)&lt;/script&gt;</span>`,
			notWant: `<script>`,
		},
		"element injection with size": {
			text:    `<img src=x onerror="alert(1)">`,
			style:   TextStyle{Size: 12},
			want:    `<span style="font-size: 12px">&lt;img src=x onerror=&#34;alert(1)&#34;&gt;</span>`,
			notWant: `<img`,
		},
		"bold styled text keeps markdown emphasis": {
			text:    `a < b & c`,
			style:   TextStyle{Color: "blue", Bold: true},
			want:    `<span style="color: blue">**a &lt; b &amp; c**</span>`,
			notWant: `a < b`,
		},
		"markdown escapes survive html escaping": {
			text:    `it's *x*`,
			style:   TextStyle{Color: "green"},
			want:    `<span style="color: green">it&#39;s \*x\*</span>`,
			notWant: `&\#39;`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().Text(tc.text, WithTextStyle(tc.style)).Build()

			renderer := &markdownRenderer{headingLevel: 1}
			result, err := renderer.Render(t.Context(), doc)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			got := string(result)
			if strings.Contains(got, tc.notWant) {
				t.Errorf("Render() output contains unescaped text %q:\n%s", tc.notWant, got)
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("Render() output missing %q:\n%s", tc.want, got)
			}
		})
	}
}

// TestMarkdownRenderer_FrontMatterNotAliased is a regression test for T-1407:
// the front matter constructors and format helpers must copy the caller's map
// so later mutations of that map do not change subsequent renders.
func TestMarkdownRenderer_FrontMatterNotAliased(t *testing.T) {
	tests := map[string]struct {
		build func(fm map[string]string) Renderer
	}{
		"NewMarkdownRendererWithFrontMatter": {
			build: func(fm map[string]string) Renderer { return NewMarkdownRendererWithFrontMatter(fm) },
		},
		"NewMarkdownRendererWithOptions": {
			build: func(fm map[string]string) Renderer { return NewMarkdownRendererWithOptions(false, fm) },
		},
		"MarkdownWithFrontMatter": {
			build: func(fm map[string]string) Renderer { return MarkdownWithFrontMatter(fm).Renderer },
		},
		"MarkdownWithOptions": {
			build: func(fm map[string]string) Renderer { return MarkdownWithOptions(false, fm).Renderer },
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fm := map[string]string{"title": "First Report"}
			renderer := tc.build(fm)

			// Reuse the map for another report after constructing the renderer.
			fm["title"] = "Second Report"
			fm["author"] = "someone"
			delete(fm, "title")

			doc := New().Text("body").Build()
			result, err := renderer.Render(t.Context(), doc)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			got := string(result)
			want := "---\ntitle: \"First Report\"\n---\n\n"
			if !strings.HasPrefix(got, want) {
				t.Errorf("Render() front matter changed after mutating caller map\ngot:\n%s\nwant prefix:\n%s", got, want)
			}
		})
	}
}
