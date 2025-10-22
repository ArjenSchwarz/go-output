# HTML Template System Design

## Overview

The HTML Template System transforms the v2 HTML renderer from a fragment-only renderer into a full HTML document generator with modern responsive styling, while maintaining backward compatibility through fragment mode. The system provides three operational modes:

1. **Default mode**: Full HTML5 documents with embedded responsive CSS
2. **Custom template mode**: Full HTML5 documents with user-provided templates
3. **Fragment mode**: HTML fragments for embedding (current behavior)

This design addresses a breaking change from v1 (which output complete documents) while providing a modern, extensible templating system with responsive CSS, customization options, and support for embedded content like Mermaid diagrams.

## Architecture

### High-Level Architecture

The template system integrates with the existing v2 renderer architecture using a wrapper pattern:

```
Document → htmlRenderer.Render() → Fragment Rendering → Template Wrapping → Complete HTML
                                   ↓
                                   Mermaid Script Injection (if needed)
```

### Component Layers

1. **Template Definition Layer** (`html_template.go`)
   - `HTMLTemplate` struct: Document metadata and styling configuration
   - Built-in template variants: `DefaultHTMLTemplate`, `MinimalHTMLTemplate`, `MermaidHTMLTemplate`
   - Template constructor functions

2. **CSS Layer** (`html_css.go`)
   - Responsive CSS constants using CSS custom properties
   - Modern CSS features (flexbox, grid)
   - Mobile-first responsive design patterns

3. **Renderer Integration Layer** (`html_renderer.go`)
   - Template wrapping logic
   - Fragment/document mode switching
   - Script injection coordination

4. **Format Layer** (`renderer.go`)
   - Format constants: `HTML`, `HTMLFragment`
   - Format constructor: `HTMLWithTemplate(template *HTMLTemplate)`

### Design Decisions

**Decision 1: String-based template wrapping instead of html/template package**

While Go's `html/template` package provides robust HTML generation with automatic escaping, we chose a simpler string-based wrapping approach for several reasons:

- **Thread safety concerns**: Research indicates that `html/template.Execute()` has documented data race issues in concurrent scenarios (Go issues #51344, #47040), despite being officially documented as thread-safe
- **Performance**: Template parsing and execution adds overhead for simple document wrapping
- **Simplicity**: Our use case is straightforward - we're wrapping already-escaped HTML fragments in a document structure
- **Control**: Direct string manipulation gives us precise control over the output structure

The content rendering layer already handles HTML escaping for user data, so we only need to escape template metadata fields (title, meta tags, etc.) which we handle explicitly with `html.EscapeString()`.

**Decision 2: Embedded CSS over external stylesheet files**

Default templates embed CSS directly in the `<style>` tag rather than referencing external files:

- **Zero configuration**: Users get styled output immediately without managing CSS files
- **Portability**: Single-file HTML documents work everywhere (email, file sharing, etc.)
- **Simplicity**: No file path dependencies or build configuration required
- **Performance**: Eliminates extra HTTP request for small CSS payloads

Custom templates can still link external stylesheets via the `ExternalCSS` field for scenarios requiring separate files (corporate branding, CDN usage, etc.).

**Decision 3: Format constants over renderer configuration**

We provide separate `HTML` and `HTMLFragment` format constants rather than a single format with boolean configuration:

- **API clarity**: Intent is explicit (`output.HTML` vs `output.HTMLFragment`)
- **Type safety**: Format is a distinct value, not a configuration option
- **Consistency**: Matches existing v2 pattern (see `Markdown`, `MarkdownWithToC`, etc.)
- **Discovery**: IDE autocomplete shows available options

**Decision 4: CSS custom properties for theming**

Modern CSS custom properties (variables) instead of preprocessor variables:

- **Runtime modification**: Users can override theme colors via JavaScript if needed
- **Browser support**: 95%+ global support, acceptable for 2025
- **Maintainability**: Central theme definition with scoped overrides
- **Standards-based**: No build tooling required

## Components and Interfaces

### 1. HTMLTemplate Struct

```go
// HTMLTemplate defines the structure for HTML document templates
type HTMLTemplate struct {
    // Document metadata
    Title    string // Page title, defaults to "Output Report"
    Language string // HTML lang attribute, defaults to "en"
    Charset  string // Character encoding, defaults to "UTF-8"

    // Meta tags
    Viewport    string            // Viewport meta tag, defaults to responsive viewport
    Description string            // Page description meta tag
    Author      string            // Author meta tag
    MetaTags    map[string]string // Additional meta tags (name → content)

    // Styling
    CSS           string            // Embedded CSS content
    ExternalCSS   []string          // External stylesheet URLs
    ThemeOverrides map[string]string // CSS custom property overrides (e.g., "--color-primary": "#ff0000")

    // Additional content
    HeadExtra string            // Additional head content (analytics, fonts, etc.)
    BodyClass string            // CSS class for body element
    BodyAttrs map[string]string // Additional body attributes
    BodyExtra string            // Additional body content (footer scripts, etc.)
}
```

**Field Validation and Sanitization:**

All string fields containing user-controlled content MUST be HTML-escaped before inclusion in the final document to prevent XSS injection. This includes:
- `Title`, `Description`, `Author`
- `MetaTags` names and values
- `BodyClass`, `BodyAttrs` names and values
- `ThemeOverrides` names and values
- `ExternalCSS` URLs (for HTML attribute context)

Fields containing pre-validated content (CSS, scripts) are NOT escaped and are the user's responsibility:
- `CSS` - Raw CSS content, not validated or escaped. Users must ensure this contains only valid CSS.
- `HeadExtra` - Raw HTML, not validated or escaped. Users must ensure this is safe.
- `BodyExtra` - Raw HTML, not validated or escaped. Users must ensure this is safe.

**Security Note**: This design prioritizes simplicity and flexibility. Users providing template content are responsible for ensuring it is safe. The library will escape metadata fields to prevent common XSS vectors, but advanced fields (CSS, HeadExtra, BodyExtra) are provided as-is. See Security Considerations section for guidance.

### 2. Built-in Template Variants

```go
// DefaultHTMLTemplate provides modern responsive styling
var DefaultHTMLTemplate = &HTMLTemplate{
    Title:    "Output Report",
    Language: "en",
    Charset:  "UTF-8",
    Viewport: "width=device-width, initial-scale=1.0",
    CSS:      defaultResponsiveCSS, // From html_css.go
}

// MinimalHTMLTemplate provides basic structure with no styling
var MinimalHTMLTemplate = &HTMLTemplate{
    Title:    "Output Report",
    Language: "en",
    Charset:  "UTF-8",
    Viewport: "width=device-width, initial-scale=1.0",
    CSS:      "", // No styling
}

// MermaidHTMLTemplate optimized for diagram rendering
var MermaidHTMLTemplate = &HTMLTemplate{
    Title:    "Diagram Output",
    Language: "en",
    Charset:  "UTF-8",
    Viewport: "width=device-width, initial-scale=1.0",
    CSS:      mermaidOptimizedCSS, // From html_css.go
}
```

### 3. Format Constructors

```go
// Standard HTML format with default template
var HTML = Format{
    Name:     FormatHTML,
    Renderer: &htmlRenderer{
        useTemplate: true,
        template:    DefaultHTMLTemplate,
    },
}

// HTMLFragment format for embedding
var HTMLFragment = Format{
    Name:     FormatHTML,
    Renderer: &htmlRenderer{
        useTemplate: false,
        template:    nil,
    },
}

// HTMLWithTemplate creates a custom template format
func HTMLWithTemplate(template *HTMLTemplate) Format {
    useTemplate := true
    if template == nil {
        useTemplate = false
        template = nil
    }

    return Format{
        Name:     FormatHTML,
        Renderer: &htmlRenderer{
            useTemplate: useTemplate,
            template:    template,
        },
    }
}
```

### 4. Renderer Structure

```go
type htmlRenderer struct {
    baseRenderer
    collapsibleConfig RendererConfig
    useTemplate       bool          // Enable/disable template wrapping
    template          *HTMLTemplate // Template configuration (nil = use default)
}
```

The renderer's `Render()` method follows this flow:

1. Render document content to HTML fragments (existing logic)
2. Inject Mermaid scripts if charts are present (existing logic)
3. If `useTemplate` is true, wrap fragments in HTML template
4. Return final output

### 5. Template Wrapping Logic

```go
func (h *htmlRenderer) wrapInTemplate(fragmentHTML []byte, tmpl *HTMLTemplate) []byte {
    // Use default template if none provided
    if tmpl == nil {
        tmpl = DefaultHTMLTemplate
    }

    var buf strings.Builder

    // DOCTYPE
    buf.WriteString("<!DOCTYPE html>\n")

    // HTML element with lang attribute
    buf.WriteString(fmt.Sprintf("<html lang=\"%s\">\n", html.EscapeString(tmpl.Language)))

    // Head section
    buf.WriteString("<head>\n")
    buf.WriteString(fmt.Sprintf("  <meta charset=\"%s\">\n", html.EscapeString(tmpl.Charset)))

    if tmpl.Viewport != "" {
        buf.WriteString(fmt.Sprintf("  <meta name=\"viewport\" content=\"%s\">\n",
            html.EscapeString(tmpl.Viewport)))
    }

    buf.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(tmpl.Title)))

    // Additional meta tags
    if tmpl.Description != "" {
        buf.WriteString(fmt.Sprintf("  <meta name=\"description\" content=\"%s\">\n",
            html.EscapeString(tmpl.Description)))
    }
    if tmpl.Author != "" {
        buf.WriteString(fmt.Sprintf("  <meta name=\"author\" content=\"%s\">\n",
            html.EscapeString(tmpl.Author)))
    }

    // Custom meta tags
    for name, content := range tmpl.MetaTags {
        buf.WriteString(fmt.Sprintf("  <meta name=\"%s\" content=\"%s\">\n",
            html.EscapeString(name), html.EscapeString(content)))
    }

    // External stylesheets
    for _, href := range tmpl.ExternalCSS {
        buf.WriteString(fmt.Sprintf("  <link rel=\"stylesheet\" href=\"%s\">\n",
            html.EscapeString(href)))
    }

    // Embedded CSS
    if tmpl.CSS != "" {
        buf.WriteString("  <style>\n")
        buf.WriteString(tmpl.CSS) // CSS is NOT escaped (assumed safe)
        buf.WriteString("\n  </style>\n")
    }

    // Theme overrides (CSS custom property overrides)
    if len(tmpl.ThemeOverrides) > 0 {
        buf.WriteString("  <style>\n")
        buf.WriteString("    :root {\n")
        for prop, value := range tmpl.ThemeOverrides {
            buf.WriteString(fmt.Sprintf("      %s: %s;\n",
                html.EscapeString(prop), html.EscapeString(value)))
        }
        buf.WriteString("    }\n")
        buf.WriteString("  </style>\n")
    }

    // Additional head content
    if tmpl.HeadExtra != "" {
        buf.WriteString(tmpl.HeadExtra) // NOT escaped (assumed safe, user responsibility)
    }

    buf.WriteString("</head>\n")

    // Body section
    bodyTag := "<body"
    if tmpl.BodyClass != "" {
        bodyTag += fmt.Sprintf(" class=\"%s\"", html.EscapeString(tmpl.BodyClass))
    }
    for attr, value := range tmpl.BodyAttrs {
        bodyTag += fmt.Sprintf(" %s=\"%s\"",
            html.EscapeString(attr), html.EscapeString(value))
    }
    bodyTag += ">\n"
    buf.WriteString(bodyTag)

    // Content (already HTML-escaped by fragment rendering)
    buf.Write(fragmentHTML)

    // Additional body content
    if tmpl.BodyExtra != "" {
        buf.WriteString(tmpl.BodyExtra) // NOT escaped (scripts, etc.)
    }

    buf.WriteString("\n</body>\n</html>\n")

    return []byte(buf.String())
}
```

## Data Models

### CSS Theme Variables

The default CSS uses CSS custom properties for theming:

```css
:root {
    /* Color scheme */
    --color-primary: #2563eb;
    --color-background: #ffffff;
    --color-surface: #f9fafb;
    --color-border: #e5e7eb;
    --color-text: #111827;
    --color-text-muted: #6b7280;

    /* Spacing */
    --spacing-xs: 0.25rem;
    --spacing-sm: 0.5rem;
    --spacing-md: 1rem;
    --spacing-lg: 1.5rem;
    --spacing-xl: 2rem;

    /* Typography */
    --font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
                   "Helvetica Neue", Arial, sans-serif;
    --font-size-base: 16px;
    --font-size-small: 14px;
    --font-size-large: 18px;
    --line-height: 1.5;

    /* Layout */
    --border-radius: 0.375rem;
    --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
    --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1);

    /* Breakpoints (reference only, not used in CSS) */
    --breakpoint-mobile: 480px;
}
```

### Responsive Table Pattern

Tables use a stacking pattern on mobile devices:

```css
/* Mobile-first: stack tables */
@media (max-width: 480px) {
    .data-table thead {
        position: absolute;
        left: -9999px; /* Hide visually but keep for screen readers */
    }

    .data-table tr {
        display: block;
        margin-bottom: var(--spacing-md);
        border: 1px solid var(--color-border);
        border-radius: var(--border-radius);
    }

    .data-table td {
        display: block;
        text-align: right;
        padding: var(--spacing-sm) var(--spacing-md);
        border-bottom: 1px solid var(--color-border);
    }

    .data-table td:last-child {
        border-bottom: none;
    }

    .data-table td::before {
        content: attr(data-label);
        float: left;
        font-weight: bold;
        color: var(--color-text-muted);
    }
}
```

## Error Handling

### Template Validation Philosophy

**This design prioritizes simplicity and user control over validation.** Template wrapping does not perform validation or sanitization beyond HTML escaping of metadata fields.

### Validation Strategy

**What IS validated/escaped:**
- Metadata fields (Title, Description, Author, etc.) are HTML-escaped to prevent common XSS
- Map keys and values (MetaTags, BodyAttrs, ThemeOverrides) are HTML-escaped

**What is NOT validated:**
- CSS content - No syntax validation, injection detection, or sanitization
- External CSS URLs - No scheme validation (javascript: URLs are possible)
- HeadExtra/BodyExtra - Raw HTML passed through unchanged
- Attribute names in BodyAttrs - Event handlers (onclick, etc.) are allowed

**Rationale:** Users providing custom templates are assumed to be developers who understand HTML/CSS/JavaScript. Validation would add complexity, restrict legitimate use cases, and provide false security. The library escapes metadata to prevent simple mistakes, but advanced fields are user-controlled by design.

### Error Scenarios

Template wrapping handles edge cases gracefully without returning errors:

1. **Empty content with template**: Produces valid HTML with empty body
2. **Nil template in custom mode**: Uses `DefaultHTMLTemplate` as fallback
3. **Invalid CSS syntax**: Included as-is, browser handles parsing errors
4. **Malformed HeadExtra/BodyExtra**: Included as-is, may break document structure
5. **JavaScript URLs in ExternalCSS**: Passed through (user responsibility)
6. **Event handlers in BodyAttrs**: Passed through (user responsibility)
7. **CSS injection patterns**: Passed through (user responsibility)

**Error handling philosophy**: Template wrapping is the final step after successful fragment rendering. Configuration issues are the user's responsibility. The function does not return errors.

## Testing Strategy

### Unit Tests (`html_template_test.go`)

**Template Field Population:**
- Test default field values (Title, Charset, Language, Viewport)
- Test custom field overrides
- Test empty/nil field handling

**HTML Escaping:**
- Test XSS prevention in Title field (e.g., `<script>alert('xss')</script>`)
- Test XSS prevention in meta tags
- Test XSS prevention in BodyClass and BodyAttrs
- Test that CSS and HeadExtra/BodyExtra are NOT escaped

**Template Variants:**
- Test DefaultHTMLTemplate structure
- Test MinimalHTMLTemplate (no CSS)
- Test MermaidHTMLTemplate (diagram-optimized CSS)

**Format Constructors:**
- Test `HTML` format uses template
- Test `HTMLFragment` format skips template
- Test `HTMLWithTemplate(nil)` enables fragment mode
- Test `HTMLWithTemplate(custom)` uses custom template

### Integration Tests (`html_integration_test.go`)

**Full Document Generation:**
- Test document with table content produces valid HTML5
- Test document with multiple content types
- Test document with sections and nested content
- Validate output with HTML5 validator patterns

**Mermaid Integration:**
- Test Mermaid chart with template includes script
- Test Mermaid script appears at end of body
- Test Mermaid chart in fragment mode includes script

**Template Customization:**
- Test custom title appears in output
- Test custom CSS overrides defaults
- Test external stylesheet links
- Test meta tag injection
- Test HeadExtra/BodyExtra content

**Edge Cases:**
- Test empty document produces valid HTML structure
- Test special characters in all template fields

### Thread Safety Tests

**Concurrent Rendering:**
- Test multiple goroutines rendering same template concurrently
- Verify no data races with race detector (`go test -race`)
- Test template instance reuse across renders

### CSS Validation Tests

**Responsive Behavior:**
- Test CSS contains mobile breakpoint (@media max-width: 480px)
- Test table stacking styles exist
- Test CSS custom properties are defined

**Accessibility:**
- Test focus states exist for interactive elements
- Test color contrast meets WCAG AA standards (4.5:1 for normal text)
- Verify semantic HTML structure (header, main, etc.)

## Testing Infrastructure

### HTML5 Validation

We won't run actual W3C validator calls in unit tests (external dependency), but we'll test for validation markers:

```go
func TestValidHTML5Structure(t *testing.T) {
    tests := map[string]struct{
        doc      *Document
        template *HTMLTemplate
    }{
        "basic document": {
            doc: NewBuilder().AddTable(/* ... */).Build(),
            template: DefaultHTMLTemplate,
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            renderer := &htmlRenderer{useTemplate: true, template: tc.template}
            output, err := renderer.Render(context.Background(), tc.doc)

            require.NoError(t, err)
            html := string(output)

            // Check HTML5 markers
            assert.Contains(t, html, "<!DOCTYPE html>")
            assert.Contains(t, html, "<html lang=")
            assert.Contains(t, html, "<meta charset=")
            assert.Contains(t, html, "<meta name=\"viewport\"")
            assert.Contains(t, html, "<title>")
            assert.Contains(t, html, "</html>")
        })
    }
}
```

### Accessibility Testing Approach

We'll test for the presence of accessibility features but not perform actual WCAG audits:

```go
func TestAccessibilityFeatures(t *testing.T) {
    // Test semantic HTML
    // Test focus state CSS exists
    // Test alt text preservation (future)
    // Test keyboard navigation CSS (details/summary)
}
```

## Implementation Phases

### Phase 1: Core Template Infrastructure
1. Create `html_template.go` with `HTMLTemplate` struct
2. Create `html_css.go` with responsive CSS constants
3. Add `useTemplate` and `template` fields to `htmlRenderer`
4. Implement basic template wrapping logic
5. Unit tests for template field population and escaping

### Phase 2: Format Integration
1. Create format constants (`HTML`, `HTMLFragment`)
2. Implement `HTMLWithTemplate()` constructor
3. Update `htmlRenderer.Render()` to use template wrapping
4. Unit tests for format constructors and mode switching

### Phase 3: Built-in Templates
1. Implement `DefaultHTMLTemplate` with responsive CSS
2. Implement `MinimalHTMLTemplate`
3. Implement `MermaidHTMLTemplate`
4. Unit tests for each template variant
5. Integration tests for template customization

### Phase 4: Testing and Validation
1. Integration tests for full document generation
2. Thread safety tests with race detector
3. Performance benchmarks
4. CSS validation tests
5. Edge case tests

### Phase 5: Documentation
1. Godoc comments for all exported types
2. Usage examples in package docs
3. Migration guide from v1
4. Security documentation for HeadExtra/BodyExtra

## Security Considerations

### Security Philosophy

**This library prioritizes flexibility and simplicity over restrictive validation.** Users are trusted to provide safe template content. The library provides HTML escaping for metadata fields to prevent common mistakes, but does not attempt to validate or sanitize CSS, JavaScript, or HTML content.

### XSS Prevention Strategy

**What the library escapes (automatic protection):**
- `Title`, `Description`, `Author` - HTML-escaped metadata
- `MetaTags` names and values - HTML-escaped
- `BodyClass` - HTML-escaped
- `BodyAttrs` names and values - HTML-escaped (but does not block event handlers)
- `ThemeOverrides` names and values - HTML-escaped
- `ExternalCSS` URLs - HTML-escaped (but does not validate schemes)
- All table/text/section content - HTML-escaped by fragment renderer

**What the library does NOT escape (user responsibility):**
- `CSS` - Raw CSS, passed through unchanged. Can contain `</style>` tags or malicious CSS.
- `HeadExtra` - Raw HTML/JavaScript, passed through unchanged. Direct XSS risk.
- `BodyExtra` - Raw HTML/JavaScript, passed through unchanged. Direct XSS risk.

**Known attack vectors that are NOT prevented:**
1. **CSS Injection**: `CSS: "</style><script>alert(1)</script><style>"`
2. **JavaScript URLs**: `ExternalCSS: []string{"javascript:alert(1)"}`
3. **Event Handlers**: `BodyAttrs: map[string]string{"onload": "alert(1)"}`
4. **Data URIs**: `ExternalCSS: []string{"data:text/css,@import url(javascript:alert(1))"}`

### User Guidance

**Safe usage patterns:**

1. **Use default templates** - Built-in templates are safe and don't require custom content
2. **Only customize metadata** - Changing Title, Description, Author is safe
3. **Validate your own content** - If you must use CSS/HeadExtra/BodyExtra:
   - Only use static content from trusted sources
   - Never include user-provided content in these fields
   - Consider using a CSP header to limit damage if injection occurs

**Unsafe usage patterns:**

```go
// UNSAFE: User-provided CSS
template := &HTMLTemplate{
    CSS: userInput, // XSS risk
}

// UNSAFE: User-provided scripts
template := &HTMLTemplate{
    HeadExtra: fmt.Sprintf("<script>%s</script>", userScript), // XSS risk
}

// UNSAFE: User-provided attributes
template := &HTMLTemplate{
    BodyAttrs: userAttrs, // Can inject event handlers
}
```

### Documentation Requirements

All unescaped fields MUST have godoc warnings:

```go
// CSS contains embedded CSS content included in a <style> tag.
// WARNING: This content is NOT escaped or validated. Ensure this contains only
// trusted CSS from reliable sources. Malicious CSS can break document structure
// (via </style> tags) or potentially leak data.
CSS string

// HeadExtra contains additional HTML content injected into the <head> section.
// WARNING: This content is NOT escaped or validated. Only use with content from
// trusted sources. Including user-provided content creates XSS vulnerabilities.
HeadExtra string

// BodyExtra contains additional HTML content injected before the closing </body> tag.
// WARNING: This content is NOT escaped or validated. Only use with content from
// trusted sources. Including user-provided content creates XSS vulnerabilities.
BodyExtra string

// ExternalCSS contains URLs to external stylesheets.
// WARNING: URLs are HTML-escaped but NOT validated. Ensure URLs use http:// or
// https:// schemes. JavaScript URLs (javascript:) are possible and create XSS risks.
ExternalCSS []string

// BodyAttrs contains additional attributes for the <body> element.
// WARNING: Attribute names and values are HTML-escaped but event handlers
// (onclick, onload, etc.) are allowed. Ensure this map contains only safe attributes.
BodyAttrs map[string]string
```

### Content Security Policy (CSP) Considerations

The default template does not include CSP headers. Users requiring CSP should:
1. Set CSP headers at the HTTP layer (preferred approach)
2. Or use custom template with CSP meta tag in `MetaTags` (less secure)
3. Note: Default template uses inline styles and Mermaid CDN requires CSP adjustments:
   - `style-src 'unsafe-inline'` for embedded CSS
   - `script-src https://cdn.jsdelivr.net` for Mermaid
   - `script-src 'unsafe-inline'` if using HeadExtra/BodyExtra with inline scripts

**Recommended CSP for default template:**
```
Content-Security-Policy: default-src 'self';
  style-src 'self' 'unsafe-inline';
  script-src 'self' https://cdn.jsdelivr.net;
  img-src 'self' data:;
```

## Accessibility Considerations

### Semantic HTML

The template wrapping preserves semantic HTML from fragment rendering:
- Table content uses `<table>`, `<thead>`, `<tbody>`, `<th>`, `<td>`
- Section content uses `<section>` and heading levels
- Text content uses `<h2>`, `<p>` based on styling

### Keyboard Navigation

Existing collapsible sections use `<details>`/`<summary>` which have native keyboard support.

### Screen Reader Support

Responsive table pattern keeps `<thead>` in DOM (visually hidden on mobile) for screen reader context.

### Color Contrast

Default CSS color scheme meets WCAG AA standards:
- Body text: #111827 on #ffffff (12.6:1 ratio)
- Muted text: #6b7280 on #ffffff (4.7:1 ratio)
- Primary color: #2563eb on #ffffff (5.8:1 ratio)

### Focus States

All interactive elements have visible focus states:

```css
details > summary:focus {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
}
```

## Backward Compatibility

### v1 Compatibility

v1 HTML renderer output full documents by default. v2 Fragment mode breaks this. The template system restores v1 behavior:

**v1 code:**
```go
output.HTML.Render(ctx, doc) // Returns full HTML document
```

**v2 equivalent:**
```go
output.HTML.Render(ctx, doc) // Also returns full HTML document (default)
```

**v2 fragment mode:**
```go
output.HTMLFragment.Render(ctx, doc) // Returns fragments for embedding
```

### CSS Visual Compatibility

Default template CSS matches v1 visual design:
- Similar color scheme
- Similar table styling
- Similar spacing and typography
- Responsive enhancements are additions, not breaking changes

### Migration Path

Minimal changes required for v1 users:
1. Replace v1 import with v2 import
2. Update builder pattern (already documented in v2 migration)
3. HTML output works as before (full documents)

## Open Questions for User Feedback

1. **Default behavior**: Should `output.HTML` use templates by default, or should templates be opt-in? (Proposed: default enabled, use `HTMLFragment` to disable)

2. **Template naming**: Are the template names intuitive? (`DefaultHTMLTemplate`, `MinimalHTMLTemplate`, `MermaidHTMLTemplate`)

3. **CSS customization**: Should we provide a `WithCustomCSS(css string)` helper, or is the `HTMLTemplate` struct sufficient?

4. **Viewport configuration**: Should viewport be configurable, or always use the responsive default?

5. **Script injection**: Should Mermaid scripts go in `<head>` or at end of `<body>`? (Proposed: end of body for performance)

## Future Enhancements

These are explicitly out of scope for the initial implementation but noted for future consideration:

1. **Template file loading**: `LoadHTMLTemplateFromFile(path string)` for externalized templates
2. **Template inheritance**: Base template + override mechanism
3. **Syntax highlighting**: Prism.js or Highlight.js integration for code blocks
4. **Dark mode support**: CSS media query `@media (prefers-color-scheme: dark)`
5. **Print stylesheet**: Optimized styles for print media
6. **Custom JavaScript**: Controlled script injection for interactive features
7. **Progressive enhancement**: Lazy loading for large tables
8. **Export helpers**: "Download as HTML" button generation

## Appendix: CSS Architecture

The default CSS follows modern best practices:

### Mobile-First Approach

Base styles target mobile devices, with progressive enhancement for larger screens:

```css
/* Base styles (mobile) */
.data-table { /* stacked layout */ }

/* Desktop enhancement */
@media (min-width: 481px) {
    .data-table { /* traditional table layout */ }
}
```

### Component Organization

CSS is organized by component type:
1. CSS custom properties (`:root`)
2. Global styles (`body`, `html`)
3. Typography (`h1`-`h6`, `p`)
4. Tables (`.data-table`)
5. Sections (`.content-section`)
6. Mermaid (`.mermaid`)
7. Utilities (`.table-container`)

### Performance Optimizations

- System font stack (no web font downloads)
- Minimal CSS payload (~5KB uncompressed)
- No external dependencies
- Efficient selectors (low specificity)

### Browser Support

Target: Last 2 versions of major browsers + Safari 12+

CSS features used:
- CSS custom properties (95%+ support)
- Flexbox (97%+ support)
- CSS Grid (96%+ support)
- `@media` queries (100% support)

No IE11 support required for v2 (modern Go applications).
