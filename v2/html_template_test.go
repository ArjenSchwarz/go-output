package output

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestDefaultHTMLTemplateDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template *HTMLTemplate
		field    string
		want     any
	}{
		"default Title": {
			template: DefaultHTMLTemplate,
			field:    "Title",
			want:     "Output Report",
		},
		"default Charset": {
			template: DefaultHTMLTemplate,
			field:    "Charset",
			want:     "UTF-8",
		},
		"default Language": {
			template: DefaultHTMLTemplate,
			field:    "Language",
			want:     "en",
		},
		"default Viewport": {
			template: DefaultHTMLTemplate,
			field:    "Viewport",
			want:     "width=device-width, initial-scale=1.0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got any
			switch tc.field {
			case "Title":
				got = tc.template.Title
			case "Charset":
				got = tc.template.Charset
			case "Language":
				got = tc.template.Language
			case "Viewport":
				got = tc.template.Viewport
			}

			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMinimalHTMLTemplateDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		field string
		want  any
	}{
		"Title": {
			field: "Title",
			want:  "Output Report",
		},
		"Charset": {
			field: "Charset",
			want:  "UTF-8",
		},
		"Language": {
			field: "Language",
			want:  "en",
		},
		"Viewport": {
			field: "Viewport",
			want:  "width=device-width, initial-scale=1.0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got any
			switch tc.field {
			case "Title":
				got = MinimalHTMLTemplate.Title
			case "Charset":
				got = MinimalHTMLTemplate.Charset
			case "Language":
				got = MinimalHTMLTemplate.Language
			case "Viewport":
				got = MinimalHTMLTemplate.Viewport
			}

			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMermaidHTMLTemplateDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		field string
		want  any
	}{
		"Title": {
			field: "Title",
			want:  "Diagram Output",
		},
		"Charset": {
			field: "Charset",
			want:  "UTF-8",
		},
		"Language": {
			field: "Language",
			want:  "en",
		},
		"Viewport": {
			field: "Viewport",
			want:  "width=device-width, initial-scale=1.0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got any
			switch tc.field {
			case "Title":
				got = MermaidHTMLTemplate.Title
			case "Charset":
				got = MermaidHTMLTemplate.Charset
			case "Language":
				got = MermaidHTMLTemplate.Language
			case "Viewport":
				got = MermaidHTMLTemplate.Viewport
			}

			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCustomHTMLTemplateOverrides(t *testing.T) {
	t.Parallel()

	custom := &HTMLTemplate{
		Title:    "Custom Title",
		Charset:  "ISO-8859-1",
		Language: "de",
		Viewport: "width=800px",
	}

	tests := map[string]struct {
		got  string
		want string
	}{
		"custom Title": {
			got:  custom.Title,
			want: "Custom Title",
		},
		"custom Charset": {
			got:  custom.Charset,
			want: "ISO-8859-1",
		},
		"custom Language": {
			got:  custom.Language,
			want: "de",
		},
		"custom Viewport": {
			got:  custom.Viewport,
			want: "width=800px",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.got != tc.want {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestEmptyHTMLTemplateFieldHandling(t *testing.T) {
	t.Parallel()

	empty := &HTMLTemplate{}

	tests := map[string]struct {
		field string
		check func() bool
	}{
		"empty Title": {
			field: "Title",
			check: func() bool { return empty.Title == "" },
		},
		"empty Charset": {
			field: "Charset",
			check: func() bool { return empty.Charset == "" },
		},
		"empty Language": {
			field: "Language",
			check: func() bool { return empty.Language == "" },
		},
		"empty Viewport": {
			field: "Viewport",
			check: func() bool { return empty.Viewport == "" },
		},
		"empty CSS": {
			field: "CSS",
			check: func() bool { return empty.CSS == "" },
		},
		"nil ExternalCSS": {
			field: "ExternalCSS",
			check: func() bool { return empty.ExternalCSS == nil },
		},
		"nil MetaTags": {
			field: "MetaTags",
			check: func() bool { return empty.MetaTags == nil },
		},
		"nil ThemeOverrides": {
			field: "ThemeOverrides",
			check: func() bool { return empty.ThemeOverrides == nil },
		},
		"nil BodyAttrs": {
			field: "BodyAttrs",
			check: func() bool { return empty.BodyAttrs == nil },
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if !tc.check() {
				t.Errorf("expected empty field %q", tc.field)
			}
		})
	}
}

func TestDefaultResponsiveCSSContainsCSSCustomProperties(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pattern string
	}{
		":root selector exists": {
			pattern: ":root",
		},
		"--color- variables exist": {
			pattern: "--color-",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(defaultResponsiveCSS, tc.pattern) {
				t.Errorf("defaultResponsiveCSS missing pattern %q", tc.pattern)
			}
		})
	}
}

func TestDefaultResponsiveCSSmobileBreakpoint(t *testing.T) {
	t.Parallel()

	if !strings.Contains(defaultResponsiveCSS, "@media (max-width: 480px)") && !strings.Contains(defaultResponsiveCSS, "@media (max-width:480px)") {
		t.Error("defaultResponsiveCSS missing mobile breakpoint @media (max-width: 480px)")
	}
}

func TestDefaultResponsiveCSSTableStacking(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pattern string
	}{
		".data-table exists": {
			pattern: ".data-table",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(defaultResponsiveCSS, tc.pattern) {
				t.Errorf("defaultResponsiveCSS missing pattern %q", tc.pattern)
			}
		})
	}
}

func TestDefaultResponsiveCSSSystemFontStack(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pattern string
	}{
		"system font stack 1": {
			pattern: "-apple-system",
		},
		"system font stack 2": {
			pattern: "BlinkMacSystemFont",
		},
		"system font stack 3": {
			pattern: "Segoe UI",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(defaultResponsiveCSS, tc.pattern) {
				t.Errorf("defaultResponsiveCSS missing system font pattern %q", tc.pattern)
			}
		})
	}
}

func TestDefaultResponsiveCSSWCAGColorContrast(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pattern string
		name    string
	}{
		"primary color": {
			pattern: "--color-primary:",
			name:    "primary color variable",
		},
		"background color": {
			pattern: "--color-background:",
			name:    "background color variable",
		},
		"text color": {
			pattern: "--color-text:",
			name:    "text color variable",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(defaultResponsiveCSS, tc.pattern) {
				t.Errorf("defaultResponsiveCSS missing %s with pattern %q", tc.name, tc.pattern)
			}
		})
	}
}

func TestMermaidOptimizedCSSContainsMermaidStyles(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pattern string
		name    string
	}{
		"mermaid class": {
			pattern: ".mermaid",
			name:    "mermaid class selector",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(mermaidOptimizedCSS, tc.pattern) {
				t.Errorf("mermaidOptimizedCSS missing %s with pattern %q", tc.name, tc.pattern)
			}
		})
	}
}

// Tests for template wrapping functionality
// These tests verify that the wrapInTemplate function correctly generates valid HTML5 documents

func TestWrapInTemplate_DOCTYPEDeclaration(t *testing.T) {
	t.Parallel()

	renderer := &htmlRenderer{useTemplate: true, template: DefaultHTMLTemplate}
	doc := New().Text("Test content").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Errorf("HTML output should start with <!DOCTYPE html>, got: %q", html[:50])
	}
}

func TestWrapInTemplate_HTMLElementWithLangAttribute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template *HTMLTemplate
		wantLang string
	}{
		"default lang": {
			template: DefaultHTMLTemplate,
			wantLang: "en",
		},
		"custom lang": {
			template: &HTMLTemplate{
				Title:    "Test",
				Language: "de",
				Charset:  "UTF-8",
			},
			wantLang: "de",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			renderer := &htmlRenderer{useTemplate: true, template: tc.template}
			doc := New().Text("Test").Build()
			output, err := renderer.Render(context.Background(), doc)

			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			html := string(output)
			expectedTag := `<html lang="` + tc.wantLang + `">`
			if !strings.Contains(html, expectedTag) {
				t.Errorf("HTML should contain %q, but got:\n%s", expectedTag, html[:200])
			}
		})
	}
}

func TestWrapInTemplate_HeadSectionWithCharsetMeta(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template    *HTMLTemplate
		wantCharset string
	}{
		"default charset": {
			template:    DefaultHTMLTemplate,
			wantCharset: "UTF-8",
		},
		"custom charset": {
			template: &HTMLTemplate{
				Title:   "Test",
				Charset: "ISO-8859-1",
			},
			wantCharset: "ISO-8859-1",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			renderer := &htmlRenderer{useTemplate: true, template: tc.template}
			doc := New().Text("Test").Build()
			output, err := renderer.Render(context.Background(), doc)

			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			html := string(output)
			expectedMeta := `<meta charset="` + tc.wantCharset + `">`
			if !strings.Contains(html, expectedMeta) {
				t.Errorf("HTML should contain %q, got:\n%s", expectedMeta, html)
			}
		})
	}
}

func TestWrapInTemplate_ViewportMetaTag(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template     *HTMLTemplate
		wantViewport string
	}{
		"default viewport": {
			template:     DefaultHTMLTemplate,
			wantViewport: "width=device-width, initial-scale=1.0",
		},
		"custom viewport": {
			template: &HTMLTemplate{
				Title:    "Test",
				Viewport: "width=800px",
			},
			wantViewport: "width=800px",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			renderer := &htmlRenderer{useTemplate: true, template: tc.template}
			doc := New().Text("Test").Build()
			output, err := renderer.Render(context.Background(), doc)

			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			html := string(output)
			expectedMeta := `<meta name="viewport" content="` + tc.wantViewport + `">`
			if !strings.Contains(html, expectedMeta) {
				t.Errorf("HTML should contain %q, got:\n%s", expectedMeta, html)
			}
		})
	}
}

func TestWrapInTemplate_TitleTagWithEscaping(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		title     string
		wantTitle string
	}{
		"simple title": {
			title:     "My Report",
			wantTitle: "My Report",
		},
		"title with special chars": {
			title:     "Report & Analysis <test>",
			wantTitle: "Report &amp; Analysis &lt;test&gt;",
		},
		"title with quotes": {
			title:     `My "Report"`,
			wantTitle: `My &#34;Report&#34;`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			template := &HTMLTemplate{
				Title:   tc.title,
				Charset: "UTF-8",
			}
			renderer := &htmlRenderer{useTemplate: true, template: template}
			doc := New().Text("Test").Build()
			output, err := renderer.Render(context.Background(), doc)

			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			html := string(output)
			expectedTitle := `<title>` + tc.wantTitle + `</title>`
			if !strings.Contains(html, expectedTitle) {
				t.Errorf("HTML should contain %q, got:\n%s", expectedTitle, html)
			}
		})
	}
}

func TestWrapInTemplate_DescriptionAndAuthorMetaTags(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name               string
		description        string
		author             string
		wantDescription    bool
		wantAuthor         bool
		wantDescriptionTag string
		wantAuthorTag      string
	}{
		"with description and author": {
			name:               "with both",
			description:        "Test Report",
			author:             "John Doe",
			wantDescription:    true,
			wantAuthor:         true,
			wantDescriptionTag: `<meta name="description" content="Test Report">`,
			wantAuthorTag:      `<meta name="author" content="John Doe">`,
		},
		"only description": {
			name:               "only description",
			description:        "Test Report",
			author:             "",
			wantDescription:    true,
			wantAuthor:         false,
			wantDescriptionTag: `<meta name="description" content="Test Report">`,
		},
		"only author": {
			name:            "only author",
			description:     "",
			author:          "Jane Smith",
			wantDescription: false,
			wantAuthor:      true,
			wantAuthorTag:   `<meta name="author" content="Jane Smith">`,
		},
		"with escaping": {
			name:               "with escaping",
			description:        "Test & <Report>",
			author:             "John \"Doe\"",
			wantDescription:    true,
			wantAuthor:         true,
			wantDescriptionTag: `<meta name="description" content="Test &amp; &lt;Report&gt;">`,
			wantAuthorTag:      `<meta name="author" content="John &#34;Doe&#34;">`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			template := &HTMLTemplate{
				Title:       "Test",
				Description: tc.description,
				Author:      tc.author,
				Charset:     "UTF-8",
			}
			renderer := &htmlRenderer{useTemplate: true, template: template}
			doc := New().Text("Test").Build()
			output, err := renderer.Render(context.Background(), doc)

			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			html := string(output)

			if tc.wantDescription && !strings.Contains(html, tc.wantDescriptionTag) {
				t.Errorf("HTML should contain %q, got:\n%s", tc.wantDescriptionTag, html)
			}

			if tc.wantAuthor && !strings.Contains(html, tc.wantAuthorTag) {
				t.Errorf("HTML should contain %q, got:\n%s", tc.wantAuthorTag, html)
			}

			if !tc.wantDescription && strings.Contains(html, "description") {
				t.Errorf("HTML should not contain description meta tag")
			}

			if !tc.wantAuthor && strings.Contains(html, `<meta name="author"`) {
				t.Errorf("HTML should not contain author meta tag")
			}
		})
	}
}

func TestWrapInTemplate_CustomMetaTags(t *testing.T) {
	t.Parallel()

	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		MetaTags: map[string]string{
			"theme-color":                  "#ff0000",
			"apple-mobile-web-app-capable": "yes",
		},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	tests := map[string]struct {
		wantTag string
	}{
		"theme-color": {
			wantTag: `<meta name="theme-color" content="#ff0000">`,
		},
		"apple-mobile-web-app-capable": {
			wantTag: `<meta name="apple-mobile-web-app-capable" content="yes">`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(html, tc.wantTag) {
				t.Errorf("HTML should contain %q, got:\n%s", tc.wantTag, html)
			}
		})
	}
}

func TestWrapInTemplate_ExternalCSSLinks(t *testing.T) {
	t.Parallel()

	template := &HTMLTemplate{
		Title:       "Test",
		Charset:     "UTF-8",
		ExternalCSS: []string{"https://example.com/style1.css", "https://example.com/style2.css"},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	tests := map[string]struct {
		wantLink string
	}{
		"first stylesheet": {
			wantLink: `<link rel="stylesheet" href="https://example.com/style1.css">`,
		},
		"second stylesheet": {
			wantLink: `<link rel="stylesheet" href="https://example.com/style2.css">`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(html, tc.wantLink) {
				t.Errorf("HTML should contain %q, got:\n%s", tc.wantLink, html)
			}
		})
	}
}

func TestWrapInTemplate_ExternalCSSWithEscaping(t *testing.T) {
	t.Parallel()

	template := &HTMLTemplate{
		Title:       "Test",
		Charset:     "UTF-8",
		ExternalCSS: []string{`https://example.com/style.css?v="1.0"`},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)
	expectedLink := `<link rel="stylesheet" href="https://example.com/style.css?v=&#34;1.0&#34;">`
	if !strings.Contains(html, expectedLink) {
		t.Errorf("HTML should contain escaped URL %q, got:\n%s", expectedLink, html)
	}
}

func TestWrapInTemplate_EmbeddedCSSStyleTag(t *testing.T) {
	t.Parallel()

	css := "body { color: red; }"
	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		CSS:     css,
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)
	expectedStyle := "<style>\n" + css + "\n  </style>"

	if !strings.Contains(html, expectedStyle) {
		t.Errorf("HTML should contain embedded CSS, got:\n%s", html)
	}
}

func TestWrapInTemplate_ThemeOverrides(t *testing.T) {
	t.Parallel()

	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		ThemeOverrides: map[string]string{
			"--color-primary":   "#ff0000",
			"--color-secondary": "#00ff00",
		},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Check for :root style block
	if !strings.Contains(html, ":root") {
		t.Error("HTML should contain :root CSS selector for theme overrides")
	}

	// Check for color overrides (order is not guaranteed in maps)
	if !strings.Contains(html, "--color-primary") || !strings.Contains(html, "#ff0000") {
		t.Errorf("HTML should contain primary color override, got:\n%s", html)
	}
	if !strings.Contains(html, "--color-secondary") || !strings.Contains(html, "#00ff00") {
		t.Errorf("HTML should contain secondary color override, got:\n%s", html)
	}
}

func TestWrapInTemplate_ThemeOverridesWithEscaping(t *testing.T) {
	t.Parallel()

	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		ThemeOverrides: map[string]string{
			"--color-value": `rgb(255, 0, "red")`,
		},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)
	// Quotes should be escaped
	if !strings.Contains(html, `&#34;`) {
		t.Errorf("HTML should contain escaped quotes in theme overrides, got:\n%s", html)
	}
}

func TestWrapInTemplate_HeadExtraContent(t *testing.T) {
	t.Parallel()

	headExtra := `<link rel="preconnect" href="https://fonts.googleapis.com">`
	template := &HTMLTemplate{
		Title:     "Test",
		Charset:   "UTF-8",
		HeadExtra: headExtra,
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)
	if !strings.Contains(html, headExtra) {
		t.Errorf("HTML should contain HeadExtra content, got:\n%s", html)
	}

	// Verify it's in the head section (before </head>)
	headSection := html[:strings.Index(html, "</head>")]
	if !strings.Contains(headSection, headExtra) {
		t.Error("HeadExtra should be in head section")
	}
}

func TestWrapInTemplate_BodyWithClass(t *testing.T) {
	t.Parallel()

	template := &HTMLTemplate{
		Title:     "Test",
		Charset:   "UTF-8",
		BodyClass: "dark-mode custom-theme",
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)
	expectedBodyTag := `<body class="dark-mode custom-theme">`
	if !strings.Contains(html, expectedBodyTag) {
		t.Errorf("HTML should contain body with class, got:\n%s", html)
	}
}

func TestWrapInTemplate_BodyAttrs(t *testing.T) {
	t.Parallel()

	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		BodyAttrs: map[string]string{
			"data-theme":   "light",
			"data-version": "1.0",
		},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Check for attributes (order not guaranteed)
	if !strings.Contains(html, `data-theme="light"`) {
		t.Errorf("HTML should contain data-theme attribute, got:\n%s", html)
	}
	if !strings.Contains(html, `data-version="1.0"`) {
		t.Errorf("HTML should contain data-version attribute, got:\n%s", html)
	}
}

func TestWrapInTemplate_BodyAttrsWithEscaping(t *testing.T) {
	t.Parallel()

	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		BodyAttrs: map[string]string{
			"data-config": `{"theme": "dark"}`,
		},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)
	// Quotes should be escaped
	if !strings.Contains(html, `&#34;`) {
		t.Errorf("HTML should contain escaped quotes in body attributes, got:\n%s", html)
	}
}

func TestWrapInTemplate_FragmentContentInjection(t *testing.T) {
	t.Parallel()

	template := DefaultHTMLTemplate
	renderer := &htmlRenderer{useTemplate: true, template: template}

	doc := New().Text("Hello World").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)
	if !strings.Contains(html, "Hello World") {
		t.Errorf("HTML should contain fragment content, got:\n%s", html)
	}

	// Verify content is in body
	bodyStart := strings.Index(html, "<body")
	bodyEnd := strings.Index(html, "</body>")
	bodyContent := html[bodyStart:bodyEnd]
	if !strings.Contains(bodyContent, "Hello World") {
		t.Error("Fragment content should be in body section")
	}
}

func TestWrapInTemplate_BodyExtra(t *testing.T) {
	t.Parallel()

	bodyExtra := `<script>console.log('Analytics');</script>`
	template := &HTMLTemplate{
		Title:     "Test",
		Charset:   "UTF-8",
		BodyExtra: bodyExtra,
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)
	if !strings.Contains(html, bodyExtra) {
		t.Errorf("HTML should contain BodyExtra content, got:\n%s", html)
	}

	// Verify it's before </body> tag
	bodyEndIndex := strings.LastIndex(html, "</body>")
	if bodyEndIndex > 0 {
		beforeBodyEnd := html[:bodyEndIndex]
		if !strings.Contains(beforeBodyEnd, bodyExtra) {
			t.Error("BodyExtra should be before </body> tag")
		}
	}
}

func TestWrapInTemplate_NilTemplate(t *testing.T) {
	t.Parallel()

	renderer := &htmlRenderer{useTemplate: true, template: nil}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should use DefaultHTMLTemplate
	if !strings.Contains(html, DefaultHTMLTemplate.Title) {
		t.Errorf("HTML should use DefaultHTMLTemplate title, got:\n%s", html)
	}
	if !strings.Contains(html, `lang="en"`) {
		t.Errorf("HTML should have default lang attribute, got:\n%s", html)
	}
}

func TestWrapInTemplate_EmptyFieldsProducingValidOutput(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name     string
		template *HTMLTemplate
		check    func(string) error
	}{
		"all empty fields": {
			name:     "empty fields",
			template: &HTMLTemplate{},
			check: func(html string) error {
				// Should still have DOCTYPE and closing tags even with empty fields
				if !strings.HasPrefix(html, "<!DOCTYPE html>") {
					return fmt.Errorf("missing DOCTYPE")
				}
				if !strings.Contains(html, "</html>") {
					return fmt.Errorf("missing closing html tag")
				}
				if !strings.Contains(html, "</head>") {
					return fmt.Errorf("missing closing head tag")
				}
				if !strings.Contains(html, "</body>") {
					return fmt.Errorf("missing closing body tag")
				}
				return nil
			},
		},
		"only title set": {
			name: "only title",
			template: &HTMLTemplate{
				Title: "Custom Title",
			},
			check: func(html string) error {
				if !strings.Contains(html, "<title>Custom Title</title>") {
					return fmt.Errorf("missing title")
				}
				return nil
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			renderer := &htmlRenderer{useTemplate: true, template: tc.template}
			doc := New().Text("Content").Build()
			output, err := renderer.Render(context.Background(), doc)

			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			html := string(output)
			if err := tc.check(html); err != nil {
				t.Errorf("Validation failed: %v\nHTML:\n%s", err, html)
			}
		})
	}
}

func TestWrapInTemplate_HTMLStructureOrder(t *testing.T) {
	t.Parallel()

	template := &HTMLTemplate{
		Title:       "Test Report",
		Language:    "en",
		Charset:     "UTF-8",
		Description: "A test report",
		CSS:         "body { color: blue; }",
		HeadExtra:   "<meta name=\"custom\" content=\"value\">",
		BodyClass:   "main",
		BodyExtra:   "<footer>Footer</footer>",
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Main content").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify basic structure and order
	// Find the head and body sections to avoid confusion with CSS content
	headStart := strings.Index(html, "<head>")
	headEnd := strings.Index(html, "</head>")
	bodyStart := strings.Index(html, "<body")
	bodyEnd := strings.LastIndex(html, "</body>")

	if headStart == -1 || headEnd == -1 || bodyStart == -1 || bodyEnd == -1 {
		t.Fatal("Missing head or body sections")
	}

	// Verify DOCTYPE comes first
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("HTML should start with DOCTYPE")
	}

	// Verify html tag with lang
	if !strings.Contains(html[:headStart], `<html lang="en">`) {
		t.Error("HTML should have lang attribute")
	}

	// Extract head content
	headContent := html[headStart : headEnd+7]

	// Verify order within head
	charsetIdx := strings.Index(headContent, `<meta charset="UTF-8">`)
	descIdx := strings.Index(headContent, `<meta name="description"`)
	titleIdx := strings.Index(headContent, `<title>Test Report</title>`)
	styleIdx := strings.Index(headContent, `<style>`)

	if charsetIdx == -1 {
		t.Error("Missing charset meta tag")
	}
	if descIdx == -1 {
		t.Error("Missing description meta tag")
	}
	if titleIdx == -1 {
		t.Error("Missing title tag")
	}
	if styleIdx == -1 {
		t.Error("Missing style tag")
	}

	// Verify relative order
	if charsetIdx > titleIdx {
		t.Error("charset should come before title")
	}
	if titleIdx > descIdx {
		t.Error("title should come before description")
	}

	// Extract body content
	bodyContent := html[bodyStart:bodyEnd]

	// Verify body has class
	bodyStartContent := bodyContent
	if len(bodyContent) > 100 {
		bodyStartContent = bodyContent[:100]
	}
	if !strings.Contains(bodyStartContent, `class="main"`) {
		t.Error("body should have class attribute")
	}

	// Verify content is in body
	if !strings.Contains(bodyContent, "Main content") {
		t.Error("Main content should be in body")
	}

	// Verify BodyExtra is near end of body
	if !strings.Contains(bodyContent, "<footer>Footer</footer>") {
		t.Error("BodyExtra should be in body")
	}

	// Verify closing tags
	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("HTML should end with </html>")
	}
}
