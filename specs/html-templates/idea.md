# HTML Template System - Design Document

## Overview

Add full HTML document template support to the v2 HTML renderer, making it output complete, browser-ready HTML documents by default instead of fragments. This addresses the breaking change from v1 which outputted full documents.

## Current State

**v2 HTML Renderer** currently outputs HTML fragments:
```html
<p class="text-content">Sample Report</p>
<div class="table-container">
  <table class="data-table">...</table>
</div>
<pre class="mermaid">
gantt
  ...
</pre>
<script type="module">
  import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
  mermaid.initialize({ startOnLoad: true });
</script>
```

**v1 Template** (templates/html.go):
- Full HTML document with `{{.Settings.Title}}` placeholder
- Embedded responsive table CSS (RESPONSTABLE 2.0 by jordyvanraaij)
- Content replacement using `<div id='end'></div>` placeholder
- Go's `text/template` based

## Design Decisions

### 1. Default Behavior
**Decision**: HTML renderer outputs **full HTML documents** by default
- Restores v1 behavior (non-breaking for v1→v2 migration)
- More immediately useful (can open in browser)
- Fragments available via opt-out when needed

### 2. API Design
**Decision**: Use `HTMLWithTemplate(template)` constructor pattern
```go
// Full document with default template (new default)
output.HTML.Renderer.Render(ctx, doc)

// Full document with custom template
output.HTMLWithTemplate(customTemplate).Renderer.Render(ctx, doc)

// Fragment output (opt-out)
output.HTMLWithTemplate(nil).Renderer.Render(ctx, doc)
// OR
output.HTMLFragment.Renderer.Render(ctx, doc)
```

### 3. CSS Modernization
**Decision**: Modernize the v1 CSS
- Replace old responsive table hacks with modern CSS (flexbox/grid)
- Keep mobile-first approach
- Maintain visual compatibility with v1 output
- Update color scheme if needed
- Add support for mermaid diagrams styling

### 4. Mermaid Script Placement
**Decision**: Keep script at end of `<body>` (current implementation)
- Aligns with mermaid.js documentation recommendations
- Faster initial page render
- Script loads after DOM content available

### 5. Scope
**Decision**: v2 only
- No backport to v1
- v1 remains stable with existing template
- v2 gets modern implementation

## Architecture

### Template Structure

```go
// HTMLTemplate defines the structure for HTML document templates
type HTMLTemplate struct {
    // Document metadata
    Title       string   // Page title (default: "Output Report")
    Language    string   // HTML lang attribute (default: "en")
    Charset     string   // Character encoding (default: "UTF-8")

    // Styling
    CSS         string          // Embedded CSS in <style> tag
    ExternalCSS []string        // Links to external stylesheets

    // Customization
    HeadExtra   string          // Additional content in <head>
    BodyClass   string          // CSS class for <body> tag
    BodyAttrs   map[string]string  // Additional <body> attributes
    BodyExtra   string          // Additional content before </body>

    // Metadata
    Viewport    string          // Viewport meta tag (default: responsive)
    Description string          // Meta description
    Author      string          // Meta author
}
```

### Built-in Templates

```go
// DefaultHTMLTemplate - Modern responsive template with embedded CSS
var DefaultHTMLTemplate = &HTMLTemplate{
    Title:    "Output Report",
    Charset:  "UTF-8",
    Language: "en",
    Viewport: "width=device-width, initial-scale=1.0",
    CSS:      modernResponsiveCSS,
}

// MinimalHTMLTemplate - Bare HTML5 structure, no styling
var MinimalHTMLTemplate = &HTMLTemplate{
    Title:    "Report",
    Charset:  "UTF-8",
    Language: "en",
}

// MermaidHTMLTemplate - Optimized for mermaid diagrams
var MermaidHTMLTemplate = &HTMLTemplate{
    Title:    "Diagram Report",
    Charset:  "UTF-8",
    Language: "en",
    Viewport: "width=device-width, initial-scale=1.0",
    CSS:      mermaidOptimizedCSS,
}
```

### Renderer Changes

```go
type htmlRenderer struct {
    baseRenderer
    collapsibleConfig RendererConfig
    template         *HTMLTemplate  // nil = use DefaultHTMLTemplate
    useTemplate      bool           // false = fragment mode only
}

// Constructor for custom template
func HTMLWithTemplate(template *HTMLTemplate) Format {
    if template == nil {
        // nil means fragment mode
        return Format{
            Name: FormatHTML,
            Renderer: &htmlRenderer{useTemplate: false},
        }
    }
    return Format{
        Name: FormatHTML,
        Renderer: &htmlRenderer{
            template: template,
            useTemplate: true,
        },
    }
}

// Fragment-only format
var HTMLFragment = Format{
    Name: FormatHTML,
    Renderer: &htmlRenderer{useTemplate: false},
}
```

### Render Flow

```go
func (h *htmlRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
    // 1. Render content fragments
    content, err := h.renderDocumentWithFormat(ctx, doc, h.renderContent, FormatHTML)
    if err != nil {
        return nil, err
    }

    // 2. Inject mermaid script if needed (already implemented)
    if h.documentContainsMermaidCharts(doc) {
        content = h.injectMermaidScript(content)
    }

    // 3. Wrap in template if enabled
    if h.useTemplate {
        template := h.template
        if template == nil {
            template = DefaultHTMLTemplate
        }
        content = h.wrapInTemplate(content, template)
    }

    return content, nil
}
```

## Template Generation

### HTML Document Structure

```html
<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
    <meta charset="{{.Charset}}">
    {{if .Viewport}}<meta name="viewport" content="{{.Viewport}}">{{end}}
    {{if .Description}}<meta name="description" content="{{.Description}}">{{end}}
    {{if .Author}}<meta name="author" content="{{.Author}}">{{end}}
    <title>{{.Title}}</title>

    {{if .CSS}}
    <style>
        {{.CSS}}
    </style>
    {{end}}

    {{range .ExternalCSS}}
    <link rel="stylesheet" href="{{.}}">
    {{end}}

    {{.HeadExtra}}
</head>
<body{{if .BodyClass}} class="{{.BodyClass}}"{{end}}{{range $key, $val := .BodyAttrs}} {{$key}}="{{$val}}"{{end}}>
    {{.Content}}
    {{.BodyExtra}}
</body>
</html>
```

### Modern CSS (Modernized from v1)

Key improvements over v1 RESPONSTABLE CSS:
1. **CSS Grid/Flexbox** instead of display hacks
2. **CSS Custom Properties** for theming
3. **Modern responsive breakpoints**
4. **Improved accessibility** (focus states, contrast)
5. **Mermaid diagram support** (proper spacing, backgrounds)
6. **Dark mode ready** (CSS variables for colors)

Base structure:
```css
:root {
    /* Color palette */
    --primary-color: #167f92;
    --primary-dark: #024457;
    --background: #f2f2f2;
    --surface: #fff;
    --border: #d9e4e6;
    --text: #024457;
    --text-light: #666;

    /* Spacing */
    --spacing-sm: 0.5em;
    --spacing-md: 1em;
    --spacing-lg: 2em;

    /* Breakpoints (using container queries where supported) */
    --breakpoint-sm: 480px;
    --breakpoint-md: 768px;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    color: var(--text);
    background: var(--background);
    padding: var(--spacing-lg);
    line-height: 1.6;
    max-width: 1200px;
    margin: 0 auto;
}

/* Table styles - modernized responsive approach */
.data-table {
    width: 100%;
    border-collapse: collapse;
    background: var(--surface);
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

@media (max-width: 480px) {
    /* Stack table for mobile */
    .data-table thead {
        display: none;
    }

    .data-table tbody tr {
        display: block;
        margin-bottom: var(--spacing-md);
        border: 1px solid var(--border);
        border-radius: 4px;
    }

    .data-table td {
        display: grid;
        grid-template-columns: 120px 1fr;
        gap: var(--spacing-sm);
        padding: var(--spacing-sm);
        border-bottom: 1px solid var(--border);
    }

    .data-table td::before {
        content: attr(data-label);
        font-weight: bold;
        color: var(--primary-color);
    }
}

/* Mermaid diagram styles */
.mermaid {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: var(--spacing-lg);
    margin: var(--spacing-md) 0;
    overflow-x: auto;
}

/* Text content */
.text-content,
.text-header {
    margin: var(--spacing-md) 0;
}

.text-header {
    color: var(--primary-color);
    border-bottom: 2px solid var(--primary-color);
    padding-bottom: var(--spacing-sm);
}

/* Sections */
.content-section {
    margin: var(--spacing-lg) 0;
}

.section-content {
    padding-left: var(--spacing-md);
}
```

## Implementation Plan

### Phase 1: Core Template System
1. Create `HTMLTemplate` struct in new file `v2/html_template.go`
2. Define built-in templates (Default, Minimal, Mermaid)
3. Create modern CSS constants
4. Add `template` and `useTemplate` fields to `htmlRenderer`
5. Implement `HTMLWithTemplate()` constructor
6. Add `HTMLFragment` format constant

### Phase 2: Template Rendering
1. Implement `wrapInTemplate()` method
2. Update `Render()` to use template by default
3. Ensure mermaid script injection works with templates
4. Handle edge cases (empty content, nil template, etc.)

### Phase 3: CSS Modernization
1. Port v1 responsive table CSS
2. Modernize with flexbox/grid
3. Add CSS custom properties
4. Add mermaid-specific styles
5. Test responsive behavior at various breakpoints

### Phase 4: Testing
1. Unit tests for template wrapping
2. Tests for each built-in template
3. Tests for custom templates
4. Tests for fragment mode
5. Visual regression tests (if applicable)
6. Integration tests with mermaid charts

### Phase 5: Documentation
1. Update README with template examples
2. Add template customization guide
3. Document migration from v1
4. Add examples for common use cases

## API Examples

### Basic Usage (Default Template)
```go
doc := output.New().
    Text("Sales Report").
    Table("Q4 Results", salesData).
    Build()

// Outputs full HTML document with default template
html, _ := output.HTML.Renderer.Render(ctx, doc)
```

### Custom Title
```go
template := &output.HTMLTemplate{
    Title: "Q4 2024 Sales Report",
}
html, _ := output.HTMLWithTemplate(template).Renderer.Render(ctx, doc)
```

### Custom CSS
```go
template := &output.HTMLTemplate{
    Title: "Branded Report",
    CSS: myCompanyCSS,
    ExternalCSS: []string{
        "https://fonts.googleapis.com/css2?family=Roboto",
    },
}
html, _ := output.HTMLWithTemplate(template).Renderer.Render(ctx, doc)
```

### Analytics Integration
```go
template := &output.HTMLTemplate{
    Title: "Public Report",
    HeadExtra: `<script async src="https://www.googletagmanager.com/gtag/js?id=UA-XXXXX"></script>`,
    BodyExtra: `<script>
        window.dataLayer = window.dataLayer || [];
        function gtag(){dataLayer.push(arguments);}
        gtag('js', new Date());
        gtag('config', 'UA-XXXXX');
    </script>`,
}
html, _ := output.HTMLWithTemplate(template).Renderer.Render(ctx, doc)
```

### Fragment Mode (Embedding)
```go
// For embedding in existing HTML pages
fragment, _ := output.HTMLFragment.Renderer.Render(ctx, doc)
// OR
fragment, _ := output.HTMLWithTemplate(nil).Renderer.Render(ctx, doc)
```

### Mermaid-Optimized Template
```go
doc := output.New().
    GanttChart("Project Timeline", tasks).
    PieChart("Resource Distribution", resources, true).
    Build()

html, _ := output.HTMLWithTemplate(output.MermaidHTMLTemplate).Renderer.Render(ctx, doc)
```

## Backward Compatibility

### Migration from v1
v1 code that used `ToHTML()`:
```go
// v1
html := output.ToHTML()
```

v2 equivalent (identical output):
```go
// v2 - outputs full HTML document by default
html, _ := output.HTML.Renderer.Render(ctx, doc)
```

### Migration from v2 (fragments)
Existing v2 code expecting fragments:
```go
// Old v2 (was fragments)
html, _ := output.HTML.Renderer.Render(ctx, doc)

// New v2 (get fragments)
html, _ := output.HTMLFragment.Renderer.Render(ctx, doc)
```

## Testing Strategy

### Unit Tests
- Template field population
- CSS injection
- Meta tag generation
- Script placement with templates
- Fragment mode vs template mode
- Custom template properties
- Nil template handling

### Integration Tests
- Full document generation
- Mermaid charts with templates
- Multiple content types
- Nested sections with templates
- Large documents (performance)

### Visual Tests
- Responsive behavior at breakpoints
- Table rendering on mobile
- Mermaid diagram display
- Cross-browser compatibility
- Dark mode (if implemented)

## Edge Cases

1. **Empty Content**: Template should still generate valid HTML
2. **Nil Template**: Should output fragments (no template)
3. **Missing Title**: Use default "Output Report"
4. **Invalid CSS**: Should not break HTML structure
5. **Special Characters**: Proper HTML escaping in all template fields
6. **Large Documents**: Memory-efficient template application
7. **Concurrent Rendering**: Thread-safe template usage

## Future Enhancements

### Possible Future Features (Not in Initial Scope)
1. **Dark mode** toggle in template
2. **Print stylesheets** for PDF generation
3. **Multiple theme presets** (professional, minimal, colorful)
4. **Template inheritance** for custom template creation
5. **JavaScript framework integration** (React/Vue component output)
6. **Accessibility helpers** (ARIA labels, screen reader text)
7. **Export options** (add "save as" button in template)
8. **Table of contents** generation from sections
9. **Collapsible sections** in HTML output
10. **Search functionality** in template

## File Structure

```
v2/
├── html_renderer.go          # Modified: add template support
├── html_template.go          # New: template definitions
├── html_css.go              # New: CSS constants
├── html_template_test.go    # New: template tests
└── html_integration_test.go # New: full document tests
```

## Success Criteria

1. ✅ HTML renderer outputs full documents by default
2. ✅ v1 users can migrate with minimal changes
3. ✅ Fragment mode available for embedding
4. ✅ Modern, responsive CSS that works on mobile
5. ✅ Mermaid charts render correctly in templates
6. ✅ Custom templates easy to create and use
7. ✅ All tests pass (unit + integration)
8. ✅ Documentation complete with examples
9. ✅ No performance regression
10. ✅ Zero linter issues
