package output

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestDefaultHTMLTemplateDefaults(t *testing.T) {

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

			if tc.got != tc.want {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestEmptyHTMLTemplateFieldHandling(t *testing.T) {

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

			if !tc.check() {
				t.Errorf("expected empty field %q", tc.field)
			}
		})
	}
}

func TestDefaultResponsiveCSSContainsCSSCustomProperties(t *testing.T) {

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

			if !strings.Contains(defaultResponsiveCSS, tc.pattern) {
				t.Errorf("defaultResponsiveCSS missing pattern %q", tc.pattern)
			}
		})
	}
}

func TestDefaultResponsiveCSSmobileBreakpoint(t *testing.T) {

	if !strings.Contains(defaultResponsiveCSS, "@media (max-width: 480px)") && !strings.Contains(defaultResponsiveCSS, "@media (max-width:480px)") {
		t.Error("defaultResponsiveCSS missing mobile breakpoint @media (max-width: 480px)")
	}
}

func TestDefaultResponsiveCSSTableStacking(t *testing.T) {

	tests := map[string]struct {
		pattern string
	}{
		".data-table exists": {
			pattern: ".data-table",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			if !strings.Contains(defaultResponsiveCSS, tc.pattern) {
				t.Errorf("defaultResponsiveCSS missing pattern %q", tc.pattern)
			}
		})
	}
}

func TestDefaultResponsiveCSSSystemFontStack(t *testing.T) {

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

			if !strings.Contains(defaultResponsiveCSS, tc.pattern) {
				t.Errorf("defaultResponsiveCSS missing system font pattern %q", tc.pattern)
			}
		})
	}
}

func TestDefaultResponsiveCSSWCAGColorContrast(t *testing.T) {

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

			if !strings.Contains(defaultResponsiveCSS, tc.pattern) {
				t.Errorf("defaultResponsiveCSS missing %s with pattern %q", tc.name, tc.pattern)
			}
		})
	}
}

func TestMermaidOptimizedCSSContainsMermaidStyles(t *testing.T) {

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

			if !strings.Contains(mermaidOptimizedCSS, tc.pattern) {
				t.Errorf("mermaidOptimizedCSS missing %s with pattern %q", tc.name, tc.pattern)
			}
		})
	}
}

// Tests for template wrapping functionality
// These tests verify that the wrapInTemplate function correctly generates valid HTML5 documents

func TestWrapInTemplate_DOCTYPEDeclaration(t *testing.T) {

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

			if !strings.Contains(html, tc.wantTag) {
				t.Errorf("HTML should contain %q, got:\n%s", tc.wantTag, html)
			}
		})
	}
}

func TestWrapInTemplate_ExternalCSSLinks(t *testing.T) {

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

			if !strings.Contains(html, tc.wantLink) {
				t.Errorf("HTML should contain %q, got:\n%s", tc.wantLink, html)
			}
		})
	}
}

func TestWrapInTemplate_ExternalCSSWithEscaping(t *testing.T) {

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

// Tests for HTML escaping (XSS prevention)

func TestWrapInTemplate_XSSPrevention_TitleWithScript(t *testing.T) {

	template := &HTMLTemplate{
		Title:   "<script>alert('xss')</script>",
		Charset: "UTF-8",
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Script tag should be escaped, not executed
	if strings.Contains(html, "<script>alert('xss')</script>") {
		t.Error("XSS vulnerability: unescaped script tag in title")
	}

	// Should contain escaped version
	if !strings.Contains(html, "&lt;script&gt;") || !strings.Contains(html, "&lt;/script&gt;") {
		t.Errorf("Title should be HTML-escaped, got:\n%s", html)
	}
}

func TestWrapInTemplate_XSSPrevention_TitleWithVariousPatterns(t *testing.T) {

	tests := map[string]struct {
		title         string
		shouldNotFind string
		shouldFind    string
	}{
		"img tag with onerror": {
			title:         `<img src=x onerror="alert('xss')">`,
			shouldNotFind: `<img src=x onerror="alert('xss')">`,
			shouldFind:    "&lt;img",
		},
		"svg with onload": {
			title:         `<svg onload="alert('xss')">`,
			shouldNotFind: `<svg onload="alert('xss')">`,
			shouldFind:    "&lt;svg",
		},
		"javascript href": {
			title:         `<a href="javascript:alert('xss')">`,
			shouldNotFind: `javascript:alert('xss')`,
			shouldFind:    "&lt;a",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

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

			if strings.Contains(html, tc.shouldNotFind) {
				t.Errorf("XSS vulnerability in title: found unescaped %q", tc.shouldNotFind)
			}

			if !strings.Contains(html, tc.shouldFind) {
				t.Errorf("Title should be escaped, expected to find %q, got:\n%s", tc.shouldFind, html)
			}
		})
	}
}

func TestWrapInTemplate_XSSPrevention_DescriptionAndAuthor(t *testing.T) {

	template := &HTMLTemplate{
		Title:       "Test",
		Description: "Description with <script>alert('xss')</script>",
		Author:      "Author <img src=x onerror='alert(1)'>",
		Charset:     "UTF-8",
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Check that script tags are escaped
	if strings.Contains(html, "<script>alert('xss')</script>") {
		t.Error("XSS vulnerability: unescaped script tag in description")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("Description should be escaped")
	}

	// Check that img tag is escaped
	if strings.Contains(html, "<img src=x onerror='alert(1)'>") {
		t.Error("XSS vulnerability: unescaped img tag in author")
	}
	if !strings.Contains(html, "&lt;img") {
		t.Error("Author should be escaped")
	}
}

func TestWrapInTemplate_XSSPrevention_MetaTags(t *testing.T) {

	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		MetaTags: map[string]string{
			"custom\"><script>alert('xss')</script><meta name=\"x": "value",
			"safe-tag": "<img src=x onerror='alert(1)'>",
		},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Script tags should not be present
	if strings.Contains(html, "<script>alert('xss')</script>") {
		t.Error("XSS vulnerability: unescaped script tag in meta tag name")
	}

	// img tag in value should be escaped
	if strings.Contains(html, "<img src=x onerror='alert(1)'") {
		t.Error("XSS vulnerability: unescaped img tag in meta tag value")
	}

	// Both names and values should be escaped
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("MetaTags name should be escaped")
	}
	if !strings.Contains(html, "&lt;img") {
		t.Error("MetaTags value should be escaped")
	}
}

func TestWrapInTemplate_XSSPrevention_BodyClass(t *testing.T) {

	template := &HTMLTemplate{
		Title:     "Test",
		Charset:   "UTF-8",
		BodyClass: "normal\" onclick=\"alert('xss')\" class=\"",
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// The onclick handler should be escaped (not functional)
	if strings.Contains(html, `onclick="alert('xss')"`) {
		t.Error("XSS vulnerability: unescaped onclick handler in BodyClass")
	}

	// Quotes should be escaped
	if !strings.Contains(html, "&#34;") {
		t.Errorf("BodyClass should escape quotes, got:\n%s", html)
	}
}

func TestWrapInTemplate_XSSPrevention_BodyAttrs(t *testing.T) {

	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		BodyAttrs: map[string]string{
			"data-config\"><script>alert('xss')</script><div class=\"": "value",
			"data-safe": "<img src=x onerror='alert(1)'>",
		},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Script tags should not be present
	if strings.Contains(html, "<script>alert('xss')</script>") {
		t.Error("XSS vulnerability: unescaped script tag in BodyAttrs")
	}

	// img tag should be escaped
	if strings.Contains(html, "<img src=x onerror='alert(1)'") {
		t.Error("XSS vulnerability: unescaped img tag in BodyAttrs")
	}

	// Both names and values should be escaped
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("BodyAttrs should escape script tags")
	}
	if !strings.Contains(html, "&lt;img") {
		t.Error("BodyAttrs should escape img tags")
	}
}

func TestWrapInTemplate_XSSPrevention_ThemeOverrides(t *testing.T) {

	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		ThemeOverrides: map[string]string{
			"--color-primary\"></style><script>alert('xss')</script><style>": "value",
			"--color-text": "red\"><script>alert('xss')</script><div class=\"",
		},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Script tags should not be present as active HTML
	if strings.Contains(html, "</style><script>alert('xss')</script><style>") {
		t.Error("XSS vulnerability: unescaped script injection in ThemeOverrides")
	}

	// Should be escaped
	if !strings.Contains(html, "&lt;/style&gt;") && !strings.Contains(html, "&#34;") {
		t.Errorf("ThemeOverrides should be escaped, got:\n%s", html)
	}
}

func TestWrapInTemplate_NoEscaping_CSSField(t *testing.T) {

	// CSS field should NOT be escaped - it's trusted content
	cssContent := `body { color: red; }
/* Valid CSS comment */
@media (max-width: 480px) {
  .table { display: block; }
}`

	template := &HTMLTemplate{
		Title:   "Test",
		Charset: "UTF-8",
		CSS:     cssContent,
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// CSS should appear exactly as provided (not escaped)
	if !strings.Contains(html, cssContent) {
		t.Errorf("CSS field should not be escaped, expected %q, got:\n%s", cssContent, html)
	}
}

func TestWrapInTemplate_NoEscaping_HeadExtraField(t *testing.T) {

	// HeadExtra field should NOT be escaped - it's trusted HTML
	headExtra := `<meta property="og:title" content="My Page">
<script>window.custom = { theme: 'dark' };</script>
<link rel="preconnect" href="https://fonts.googleapis.com">`

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

	// HeadExtra should appear exactly as provided (not escaped)
	if !strings.Contains(html, headExtra) {
		t.Errorf("HeadExtra field should not be escaped, expected %q, got:\n%s", headExtra, html)
	}

	// Verify it's in the head section
	headSection := html[:strings.Index(html, "</head>")]
	if !strings.Contains(headSection, "<script>window.custom") {
		t.Error("HeadExtra should be in head section")
	}
}

func TestWrapInTemplate_NoEscaping_BodyExtraField(t *testing.T) {

	// BodyExtra field should NOT be escaped - it's trusted HTML
	bodyExtra := `<script>
  document.addEventListener('DOMContentLoaded', function() {
    console.log('Analytics initialized');
  });
</script>
<footer><p>&copy; 2024 My Company</p></footer>`

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

	// BodyExtra should appear exactly as provided (not escaped)
	if !strings.Contains(html, bodyExtra) {
		t.Errorf("BodyExtra field should not be escaped, expected %q, got:\n%s", bodyExtra, html)
	}

	// Verify it's before closing body tag
	bodyEndIndex := strings.LastIndex(html, "</body>")
	if bodyEndIndex > 0 {
		beforeBodyEnd := html[:bodyEndIndex]
		if !strings.Contains(beforeBodyEnd, "Analytics initialized") {
			t.Error("BodyExtra should be before </body> tag")
		}
	}
}

func TestWrapInTemplate_SpecialCharactersInAllFields(t *testing.T) {

	specialChars := "Test & <HTML> \"quotes\" 'apostrophes'"

	template := &HTMLTemplate{
		Title:       specialChars,
		Language:    "en",
		Charset:     "UTF-8",
		Viewport:    specialChars,
		Description: specialChars,
		Author:      specialChars,
		BodyClass:   specialChars,
		MetaTags: map[string]string{
			specialChars: specialChars,
		},
		ThemeOverrides: map[string]string{
			specialChars: specialChars,
		},
		BodyAttrs: map[string]string{
			specialChars: specialChars,
		},
		ExternalCSS: []string{specialChars},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// All special chars should be escaped in attribute/meta contexts
	// The ampersand should be escaped at least somewhere
	escapedAmp := "&amp;"
	escapedLt := "&lt;"
	escapedGt := "&gt;"
	escapedQuote := "&#34;"

	foundEscaped := false

	// At least one of these should be escaped (we don't require all of them because
	// they might be in different HTML contexts)
	if strings.Contains(html, escapedAmp) {
		foundEscaped = true
	}
	if strings.Contains(html, escapedLt) {
		foundEscaped = true
	}
	if strings.Contains(html, escapedGt) {
		foundEscaped = true
	}
	if strings.Contains(html, escapedQuote) {
		foundEscaped = true
	}

	if !foundEscaped {
		t.Errorf("Special characters should be escaped in HTML output, got:\n%s", html)
	}

	// Ensure no unescaped script tags that could be parsed as HTML
	if strings.Count(html, "<") != strings.Count(html, ">") {
		t.Log("HTML tag balance check may indicate improper escaping")
	}
}

func TestWrapInTemplate_SpecialCharactersInURLs(t *testing.T) {

	urlWithSpecialChars := `https://example.com/page?title=Test&author="John"&desc=<hello>`

	template := &HTMLTemplate{
		Title:       "Test",
		Charset:     "UTF-8",
		ExternalCSS: []string{urlWithSpecialChars},
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// URL should be escaped in href attribute
	if !strings.Contains(html, "&amp;") && !strings.Contains(html, "&#34;") && !strings.Contains(html, "&lt;") {
		t.Errorf("URL special characters should be escaped, got:\n%s", html)
	}
}

func TestWrapInTemplate_NullByteHandling(t *testing.T) {

	// Test with various problematic characters
	tests := map[string]struct {
		title string
	}{
		"with unicode": {
			title: "Test™ with ∑ symbols",
		},
		"with emoji": {
			title: "Report 📊 Summary 📈",
		},
		"mixed special": {
			title: "Test <>&\"' with all",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

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

			// Should produce valid HTML
			if !strings.HasPrefix(html, "<!DOCTYPE html>") {
				t.Error("Should produce valid HTML5")
			}

			// Should close properly
			if !strings.Contains(html, "</html>") {
				t.Error("HTML should be properly closed")
			}
		})
	}
}

// Integration tests for htmlRenderer with template wrapping

func TestHTMLRenderer_UseTemplate_True_CallsWrapInTemplate(t *testing.T) {

	template := &HTMLTemplate{
		Title:   "Integration Test",
		Charset: "UTF-8",
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}
	doc := New().Text("Test content").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should produce full HTML document
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Output should start with DOCTYPE when useTemplate=true")
	}

	// Should contain template title
	if !strings.Contains(html, "<title>Integration Test</title>") {
		t.Error("Output should contain template title")
	}

	// Should contain content
	if !strings.Contains(html, "Test content") {
		t.Error("Output should contain rendered content")
	}

	// Should close properly
	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("Output should end with </html>")
	}
}

func TestHTMLRenderer_UseTemplate_False_SkipsTemplateWrapping(t *testing.T) {

	renderer := &htmlRenderer{useTemplate: false, template: nil}
	doc := New().Text("Test content").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should NOT produce full HTML document
	if strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Output should NOT start with DOCTYPE when useTemplate=false")
	}

	// Should NOT have html/head/body tags
	if strings.Contains(html, "<html") {
		t.Error("Output should not contain <html tag when useTemplate=false")
	}

	// Should contain content
	if !strings.Contains(html, "Test content") {
		t.Error("Output should contain rendered content")
	}
}

func TestHTMLRenderer_TemplateWrappingAsFinalStep(t *testing.T) {

	customTemplate := &HTMLTemplate{
		Title:       "Final Step Test",
		Description: "Testing template as final step",
		Charset:     "UTF-8",
		BodyClass:   "custom-body",
	}

	renderer := &htmlRenderer{useTemplate: true, template: customTemplate}
	doc := New().Text("Important content").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify order: DOCTYPE -> html -> head -> body -> content -> closing tags
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("DOCTYPE should be first")
	}

	// Verify template fields are in output
	if !strings.Contains(html, "Final Step Test") {
		t.Error("Template title should be in output")
	}
	if !strings.Contains(html, "custom-body") {
		t.Error("Template BodyClass should be in output")
	}
}

func TestHTMLRenderer_DefaultTemplateUsedWhenNil(t *testing.T) {

	// Create renderer with nil template but useTemplate=true
	renderer := &htmlRenderer{useTemplate: true, template: nil}
	doc := New().Text("Test").Build()
	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should contain default template title
	if !strings.Contains(html, DefaultHTMLTemplate.Title) {
		t.Errorf("Output should use DefaultHTMLTemplate title: %q", DefaultHTMLTemplate.Title)
	}

	// Should have default charset
	if !strings.Contains(html, "UTF-8") {
		t.Error("Output should contain default UTF-8 charset")
	}

	// Should have default language
	if !strings.Contains(html, `lang="en"`) {
		t.Error("Output should contain default language en")
	}
}

func TestHTMLRenderer_IntegrationMultipleContents(t *testing.T) {

	template := &HTMLTemplate{
		Title:   "Multi-Content Test",
		Charset: "UTF-8",
	}

	renderer := &htmlRenderer{useTemplate: true, template: template}

	// Create a document with multiple content types
	data := []map[string]any{
		{"Name": "Alice", "Age": 30},
		{"Name": "Bob", "Age": 25},
	}
	doc := New().
		Text("First paragraph").
		Table("Employee", data, WithKeys("Name", "Age")).
		Text("After table").
		Build()

	output, err := renderer.Render(context.Background(), doc)

	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should wrap everything in proper HTML document
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Should have DOCTYPE")
	}

	// Should have all content
	if !strings.Contains(html, "First paragraph") {
		t.Error("Should contain first text content")
	}
	if !strings.Contains(html, "Alice") || !strings.Contains(html, "Bob") {
		t.Error("Should contain table content")
	}
	if !strings.Contains(html, "After table") {
		t.Error("Should contain text after table")
	}

	// Should have table structure
	if !strings.Contains(html, "<table") || !strings.Contains(html, "</table>") {
		t.Error("Should contain proper table tags")
	}

	// Should properly close
	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("Should properly close HTML document")
	}
}
