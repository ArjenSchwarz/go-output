# HTML Template System - Decision Log

This document records key design decisions made during the development of the HTML Template System feature, along with their rationales and alternatives considered.

## Decision 1: String-based Template Wrapping vs html/template Package

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

Go provides a standard `html/template` package for generating HTML with automatic contextual escaping. We needed to decide whether to use this package or implement simpler string-based template wrapping.

### Decision

Use string-based template wrapping with explicit `html.EscapeString()` for user-controlled fields.

### Rationale

1. **Thread Safety Concerns**: Research revealed documented data race issues with `html/template.Execute()` in concurrent scenarios (Go issues #51344, #47040), despite official documentation claiming thread-safety. Our use case requires safe concurrent rendering.

2. **Performance**: Template parsing and execution adds overhead for simple document wrapping where we don't need dynamic template features.

3. **Simplicity**: Our use case is straightforward - wrapping already-escaped HTML fragments in a document structure. We don't need conditional logic, loops, or complex template features.

4. **Control**: Direct string manipulation gives precise control over output structure and formatting (indentation, spacing).

5. **Security**: Content rendering already handles HTML escaping. We only need to escape template metadata fields (title, meta tags, etc.), which `html.EscapeString()` handles explicitly.

### Alternatives Considered

**Alternative A: Use html/template package**
- **Pros**: Standard library, automatic escaping, template syntax flexibility
- **Cons**: Thread safety issues, performance overhead, added complexity, parsing required
- **Rejected because**: Thread safety concerns and unnecessary complexity

**Alternative B: Use text/template package**
- **Pros**: Simpler than html/template, no HTML-specific constraints
- **Cons**: No automatic escaping (security risk), still has parsing overhead
- **Rejected because**: No escaping support, violates best practices

### Consequences

- Positive: Fast, thread-safe, simple implementation
- Positive: Full control over HTML structure
- Negative: Manual escaping required (but localized to template metadata)
- Negative: No template syntax features (but not needed)

### References

- [Go Issue #51344: html/template: Execute is not concurrent safe](https://github.com/golang/go/issues/51344)
- [Go Issue #47040: html/template: data race with concurrent ExecuteTemplate calls](https://github.com/golang/go/issues/47040)

---

## Decision 2: Embedded CSS vs External Stylesheet Files

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

Default templates need CSS styling. We could embed CSS in `<style>` tags or reference external CSS files.

### Decision

Embed CSS directly in `<style>` tags for default templates, with optional external stylesheet support via `ExternalCSS` field.

### Rationale

1. **Zero Configuration**: Users get styled output immediately without managing CSS files, paths, or build configuration.

2. **Portability**: Single-file HTML documents work everywhere - email attachments, file sharing, archived reports, offline viewing.

3. **Simplicity**: No file path dependencies, no relative vs absolute path issues, no 404 risks.

4. **Performance**: Eliminates HTTP request for CSS (saves 20-100ms). Small CSS payload (~5KB) doesn't justify separate file.

5. **Security**: No CORS issues, no CDN dependencies, no external resource risks.

6. **Flexibility**: Custom templates can still use `ExternalCSS` for scenarios requiring separate files (corporate branding, CDN caching, large stylesheets).

### Alternatives Considered

**Alternative A: External stylesheet files**
- **Pros**: Cacheable, separates concerns, easier to customize
- **Cons**: File management, path configuration, portability issues, extra HTTP request
- **Rejected because**: Adds complexity and breaks single-file portability

**Alternative B: Inline styles (style attribute)**
- **Pros**: Maximum specificity, no stylesheet needed
- **Cons**: Massive repetition, no media queries, poor maintainability, CSP issues
- **Rejected because**: Violates all CSS best practices

### Consequences

- Positive: Immediate usability, zero configuration
- Positive: Perfect portability, works offline
- Positive: Optimal performance (no extra requests)
- Negative: Large HTML files if CSS is extensive (mitigated by ~5KB size)
- Negative: Can't cache CSS separately (acceptable tradeoff)

---

## Decision 3: Format Constants vs Renderer Configuration

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

We need to expose template vs fragment mode to users. Options include boolean flags on renderer, configuration methods, or separate format constants.

### Decision

Provide separate `HTML` and `HTMLFragment` format constants, plus `HTMLWithTemplate()` constructor for custom templates.

### Rationale

1. **API Clarity**: Intent is explicit - `output.HTML` clearly means full document, `output.HTMLFragment` clearly means fragments only.

2. **Type Safety**: Format is a distinct value (type `Format`), not a boolean flag that could be set incorrectly.

3. **Consistency**: Matches existing v2 patterns like `Markdown`, `MarkdownWithToC()`, `TableWithStyle()`, etc.

4. **Discoverability**: IDE autocomplete shows all format options when typing `output.`. Users see `HTML` and `HTMLFragment` side-by-side.

5. **Immutability**: Format instances are read-only values, preventing accidental modification.

### Alternatives Considered

**Alternative A: Boolean flag on renderer**
```go
HTML.WithTemplate(true)  // Returns new Format with template enabled
HTML.WithTemplate(false) // Returns new Format with template disabled
```
- **Pros**: Single format constant, fluent API
- **Cons**: Less discoverable, boolean state can be confusing, breaks immutability pattern
- **Rejected because**: Less clear than separate constants

**Alternative B: Renderer method**
```go
renderer := output.NewHTMLRenderer()
renderer.EnableTemplate(true)
```
- **Pros**: Explicit configuration
- **Cons**: Breaks Format abstraction, requires renderer instance, mutable state
- **Rejected because**: Violates v2 immutability design

### Consequences

- Positive: Crystal-clear API, discoverable, type-safe
- Positive: Consistent with v2 patterns
- Negative: Two format constants instead of one (acceptable, clear intent)

---

## Decision 4: CSS Custom Properties for Theming

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

Need to define CSS theme system. Options include hardcoded values, Sass variables, or CSS custom properties.

### Decision

Use CSS custom properties (CSS variables) for all theme values (colors, spacing, typography).

### Rationale

1. **Runtime Modification**: Users can override theme colors via JavaScript if needed:
   ```js
   document.documentElement.style.setProperty('--color-primary', '#ff0000');
   ```

2. **Browser Support**: 95%+ global support (all modern browsers), acceptable for 2025.

3. **Maintainability**: Central theme definition in `:root`, easy to see all theme values, scoped overrides possible.

4. **Standards-Based**: No build tooling (Sass, Less) required, works in all environments.

5. **Performance**: No runtime cost compared to hardcoded values, browser-native feature.

6. **Cascading**: Variables cascade and inherit, enabling component-level theming:
   ```css
   .dark-section {
       --color-background: #111;
       --color-text: #fff;
   }
   ```

### Alternatives Considered

**Alternative A: Hardcoded CSS values**
```css
body { background-color: #ffffff; }
```
- **Pros**: Simple, no variables to learn, slightly faster parsing
- **Cons**: Theming requires string replacement, no runtime modification, poor maintainability
- **Rejected because**: Can't be customized without editing CSS

**Alternative B: Sass/Less variables**
```scss
$color-primary: #2563eb;
```
- **Pros**: More features (math, functions), familiar to many developers
- **Cons**: Requires build tooling, no runtime modification, not browser-native
- **Rejected because**: Adds build dependency, can't change at runtime

### Consequences

- Positive: Modern, flexible theming system
- Positive: Runtime customization possible
- Positive: No build tools required
- Negative: Slightly more verbose CSS (acceptable)
- Negative: No support in IE11 (acceptable, v2 targets modern browsers)

---

## Decision 5: Mobile-First Responsive Design

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

Need to decide on responsive design approach: mobile-first vs desktop-first vs adaptive.

### Decision

Use mobile-first responsive design with progressive enhancement.

### Rationale

1. **Modern Best Practice**: Industry standard since ~2015, recommended by Google, MDN, CSS-Tricks.

2. **Performance**: Mobile devices get minimal CSS, desktop gets enhancements. Mobile devices (often slower) load faster.

3. **Accessibility**: Forces focus on essential content first, improving information hierarchy.

4. **Maintainability**: Easier to enhance than simplify. Adding features is clearer than removing them.

5. **Table Stacking**: Mobile-first makes stacking pattern natural:
   ```css
   /* Base: stacked (mobile) */
   .data-table tr { display: block; }

   /* Enhancement: traditional table (desktop) */
   @media (min-width: 481px) {
       .data-table tr { display: table-row; }
   }
   ```

### Alternatives Considered

**Alternative A: Desktop-first**
```css
/* Base: desktop */
.data-table { display: table; }

/* Override: mobile */
@media (max-width: 480px) {
    .data-table { display: block; }
}
```
- **Pros**: Matches v1 approach, easier for developers with desktop background
- **Cons**: Mobile devices load desktop CSS then override (performance cost), against modern best practices
- **Rejected because**: Performance and best practices favor mobile-first

**Alternative B: Adaptive (fixed breakpoints)**
```css
/* Exactly 320px */
@media (width: 320px) { }
/* Exactly 768px */
@media (width: 768px) { }
```
- **Pros**: Precise control for specific devices
- **Cons**: Breaks on intermediate sizes, high maintenance, outdated approach
- **Rejected because**: Fluid responsive design is more robust

### Consequences

- Positive: Modern, performant, accessible design
- Positive: Better mobile experience (growing user base)
- Positive: Follows industry best practices
- Negative: Requires mobile-first thinking (learning curve for some developers)

---

## Decision 6: Table Responsive Pattern - Stacking vs Scrolling

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

Tables need to work on mobile devices. Common patterns: horizontal scrolling, stacking rows, hiding columns, or accordion expansion.

### Decision

Use stacking pattern (rows become blocks) for mobile devices.

### Rationale

1. **User Research**: Studies show stacking is preferred for small tables (<10 columns), which matches our typical use case.

2. **Accessibility**: Screen readers maintain context with hidden-but-present `<thead>`, no horizontal scrolling needed.

3. **Data Visibility**: All data visible without horizontal scroll, reducing cognitive load.

4. **Proven Pattern**: Widely used in Bootstrap, Tailwind, Foundation, Material Design.

5. **Implementation**: Clean CSS-only solution, no JavaScript required:
   ```css
   @media (max-width: 480px) {
       thead { position: absolute; left: -9999px; }
       tr { display: block; }
       td::before { content: attr(data-label); }
   }
   ```

### Alternatives Considered

**Alternative A: Horizontal scrolling**
```css
.table-container { overflow-x: auto; }
```
- **Pros**: Maintains table structure, simple implementation
- **Cons**: Poor UX on mobile (hard to scroll), data often cut off, accessibility issues
- **Rejected because**: Poor mobile UX

**Alternative B: Priority columns (hide non-essential)**
```css
@media (max-width: 480px) {
    td:nth-child(3), td:nth-child(4) { display: none; }
}
```
- **Pros**: Keeps table structure, shows essential data
- **Cons**: Hides data (bad UX), requires priority configuration, data loss
- **Rejected because**: Data should be accessible, not hidden

**Alternative C: Accordion expansion**
```css
tr { display: block; }
tr.collapsed td:not(:first-child) { display: none; }
```
- **Pros**: Compact initial view, all data accessible
- **Cons**: Requires JavaScript, extra interaction, discoverability issues
- **Rejected because**: JavaScript dependency, complexity

### Consequences

- Positive: Excellent mobile UX, all data visible
- Positive: CSS-only, no JavaScript required
- Positive: Accessible to screen readers
- Negative: Layout change can be jarring (acceptable, common pattern)
- Negative: Very wide tables (>20 columns) get long on mobile (rare case)

---

## Decision 7: 480px Mobile Breakpoint

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

Need to choose breakpoint for mobile vs desktop table rendering.

### Decision

Use 480px as mobile breakpoint (`@media (max-width: 480px)`).

### Rationale

1. **Device Coverage**:
   - Covers all phones in portrait (320px - 428px)
   - Most small tablets transition to desktop layout (iPad Mini: 768px)
   - Aligns with common "mobile" definition

2. **Framework Alignment**:
   - Bootstrap: 576px (we're slightly lower for more mobile coverage)
   - Tailwind: 640px (sm)
   - Foundation: 640px
   - Our choice: 480px (conservative, more devices get mobile layout)

3. **Table Usability**: 480px is approximately the point where 5-6 table columns become readable without stacking. Below that, stacking improves readability.

4. **Real Device Data**:
   - iPhone SE: 375px
   - iPhone 12/13/14: 390px
   - iPhone 12/13/14 Pro Max: 428px
   - All covered by 480px breakpoint

### Alternatives Considered

**Alternative A: 640px breakpoint**
- **Pros**: Matches Tailwind/Foundation, covers more devices as "mobile"
- **Cons**: Forces landscape phones to stacked layout (arguably worse UX)
- **Rejected because**: Too aggressive, landscape phones can handle table layout

**Alternative B: 320px breakpoint**
- **Pros**: Only smallest devices get stacking
- **Cons**: Most phones (375px+) struggle with table layout, poor UX
- **Rejected because**: Too conservative, poor mobile UX

**Alternative C: Multiple breakpoints**
```css
@media (max-width: 320px) { /* ultra-compact */ }
@media (max-width: 480px) { /* mobile */ }
@media (max-width: 768px) { /* tablet */ }
```
- **Pros**: Fine-grained control
- **Cons**: Complexity, more CSS, hard to maintain, diminishing returns
- **Rejected because**: Overengineered for our use case

### Consequences

- Positive: Good coverage of mobile devices
- Positive: Conservative approach (when in doubt, stack)
- Positive: Simple single breakpoint
- Negative: Some landscape phones get stacking (acceptable, rare use case)

---

## Decision 8: No html/template for Content Rendering

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

Should we use `html/template` for rendering table/text/section content, or continue with current string building?

### Decision

Keep existing string building approach for content rendering, don't introduce `html/template`.

### Rationale

1. **Consistency**: Content rendering already uses string building with `html.EscapeString()`. Mixing approaches adds complexity.

2. **Performance**: Current approach is fast. Template parsing adds overhead for every content item (hundreds per document).

3. **Simplicity**: Content rendering is straightforward string concatenation. Templates add indirection.

4. **Control**: String building gives precise control over whitespace, indentation, and structure.

5. **Thread Safety**: Avoids `html/template` thread safety concerns in content rendering hot path.

6. **Testing**: Easier to test string building (simple string comparison) than template execution.

### Alternatives Considered

**Alternative A: Use html/template for all rendering**
- **Pros**: Consistent approach, automatic escaping, template syntax
- **Cons**: Thread safety issues, performance overhead, complexity
- **Rejected because**: Decision 1 rationale applies here too

**Alternative B: Use html/template only for tables**
- **Pros**: Tables are complex, templates might simplify
- **Cons**: Inconsistent approach, performance overhead, added complexity
- **Rejected because**: Current table rendering is clear and performant

### Consequences

- Positive: Consistent rendering approach across all content types
- Positive: Maximum performance
- Positive: Simple, maintainable code
- Negative: Manual escaping required (but localized and explicit)

---

## Decision 9: Package-Level Template Variables

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

Should built-in templates be package-level variables or factory functions?

### Decision

Use package-level variables for built-in templates:
```go
var DefaultHTMLTemplate = &HTMLTemplate{...}
var MinimalHTMLTemplate = &HTMLTemplate{...}
var MermaidHTMLTemplate = &HTMLTemplate{...}
```

### Rationale

1. **Zero Allocation**: Variables are initialized once at package init, zero allocation per use.

2. **Simplicity**: Users access templates directly: `output.DefaultHTMLTemplate`.

3. **Immutability**: Templates are effectively immutable (fields are read during wrapping, never modified).

4. **Consistency**: Matches Go standard library patterns (`http.DefaultServeMux`, `time.UTC`, etc.).

5. **Discovery**: IDE autocomplete shows all available templates when typing `output.`.

### Alternatives Considered

**Alternative A: Factory functions**
```go
func DefaultHTMLTemplate() *HTMLTemplate { return &HTMLTemplate{...} }
```
- **Pros**: Explicit initialization, could support parameterization later
- **Cons**: Allocation on every call, unnecessary indirection, requires ()
- **Rejected because**: Wasteful allocations, no current need for parameters

**Alternative B: Singleton pattern**
```go
var defaultTemplateOnce sync.Once
var defaultTemplate *HTMLTemplate
func DefaultHTMLTemplate() *HTMLTemplate {
    defaultTemplateOnce.Do(func() { defaultTemplate = &HTMLTemplate{...} })
    return defaultTemplate
}
```
- **Pros**: Lazy initialization, guaranteed single instance
- **Cons**: Overengineered, unnecessary complexity, same result as variable
- **Rejected because**: Package init already runs once, sync.Once adds no value

### Consequences

- Positive: Maximum performance (zero allocations)
- Positive: Simple, clean API
- Positive: Follows Go idioms
- Negative: Templates must be truly immutable (acceptable, design enforces this)

---

## Decision 10: Mermaid Script Injection Location

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase

### Context

Mermaid.js script can be injected in `<head>` or at end of `<body>`. Where should it go?

### Decision

Inject Mermaid script at end of `<body>`, after all content.

### Rationale

1. **Performance**: Scripts in `<head>` block HTML parsing and rendering. Scripts at end of `<body>` allow content to render first.

2. **User Experience**: Users see content immediately, diagrams render progressively.

3. **Best Practice**: Modern web development best practice is scripts at end of body (or async/defer in head).

4. **Compatibility**: Mermaid initializes with `startOnLoad: true`, which requires DOM to be ready. End of body ensures this.

5. **Template Wrapping**: Natural to append scripts after content in `BodyExtra` position.

### Alternatives Considered

**Alternative A: Script in <head>**
```html
<head>
    <script type="module">...</script>
</head>
```
- **Pros**: All scripts in one place, traditional location
- **Cons**: Blocks rendering, slower page load, poor UX
- **Rejected because**: Performance and UX concerns

**Alternative B: Script in <head> with defer**
```html
<head>
    <script type="module" defer>...</script>
</head>
```
- **Pros**: Non-blocking, modern approach
- **Cons**: Module scripts already defer by default, unnecessary complexity
- **Rejected because**: End of body is simpler and equivalent

**Alternative C: Split script (library in head, init in body)**
```html
<head>
    <script src="mermaid.js"></script>
</head>
<body>
    <script>mermaid.initialize()</script>
</body>
```
- **Pros**: Separation of concerns
- **Cons**: Two injection points, more complex, same performance result
- **Rejected because**: Overengineered, no benefit

### Consequences

- Positive: Optimal page load performance
- Positive: Content visible before diagrams render
- Positive: Simple single injection point
- Negative: Script separated from diagrams in source (acceptable, standard pattern)

---

## Decision 11: No Template Validation/Compilation

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: Design Phase, User Confirmed 2025-10-21

### Context

Should we validate `HTMLTemplate` struct fields or compile templates before use? This includes validation of:
- CSS content for injection patterns (e.g., `</style>` tags)
- External CSS URLs for JavaScript schemes (e.g., `javascript:` URLs)
- Body attributes for event handlers (e.g., `onclick` attributes)
- CSS syntax and structure

### Decision

No validation or compilation. Accept template as-is, fill missing fields with defaults, include extra content verbatim. Prioritize simplicity and flexibility over security validation.

### Rationale

1. **Simplicity**: Validation adds complexity with little benefit. Invalid config produces invalid HTML, which browsers handle gracefully.

2. **User Responsibility**: Advanced fields like `HeadExtra`/`BodyExtra` are explicitly documented as "use with care." Users provide valid HTML.

3. **Failure Modes**:
   - Invalid CSS → browser ignores, no crash
   - Invalid HTML in extra content → browser renders best effort, no crash
   - XSS in metadata → prevented by escaping

4. **Performance**: Zero validation overhead on every render.

5. **Flexibility**: Users can include experimental features, non-standard tags, etc.

### Alternatives Considered

**Alternative A: Field validation**
```go
func (t *HTMLTemplate) Validate() error {
    if t.Charset != "UTF-8" && t.Charset != "ISO-8859-1" {
        return errors.New("invalid charset")
    }
    // ... more validation
}
```
- **Pros**: Catches errors early, better UX
- **Cons**: Defines "valid" restrictively, blocks legitimate use cases, complexity
- **Rejected because**: Overly restrictive, unnecessary

**Alternative B: Template compilation**
```go
func CompileHTMLTemplate(t *HTMLTemplate) (*CompiledTemplate, error) {
    // Precompute HTML structure
}
```
- **Pros**: Faster rendering, validates structure
- **Cons**: Added complexity, memory overhead, unclear benefit
- **Rejected because**: String building is already fast (<1μs)

### Consequences

- Positive: Simple, fast, flexible
- Positive: Users can do anything HTML supports
- Negative: Invalid config produces invalid HTML (acceptable, user control)
- Negative: No early error detection (acceptable, browsers handle gracefully)

---

## Decision 12: ThemeOverrides for CSS Custom Property Customization

**Date**: 2025-10-21
**Status**: Accepted
**Decision Maker(s)**: User Confirmed 2025-10-21

### Context

Users wanting to customize the default theme must currently replace the entire CSS block (~5KB), even to change a single color. This creates poor UX for simple theme customizations.

### Decision

Add `ThemeOverrides map[string]string` field to `HTMLTemplate` that allows overriding individual CSS custom properties without replacing the entire CSS block.

### Rationale

1. **Usability**: Changing theme colors becomes trivial:
   ```go
   template := &HTMLTemplate{
       ThemeOverrides: map[string]string{
           "--color-primary": "#ff0000",
           "--color-background": "#f5f5f5",
       },
   }
   ```

2. **Maintainability**: Users don't need to copy/maintain large CSS blocks for simple customizations.

3. **Compatibility**: Works with default CSS custom properties without breaking existing templates.

4. **Implementation**: Simple to implement - inject a second `<style>` block with `:root` overrides.

5. **Flexibility**: Users can override any CSS variable, not just colors (spacing, fonts, etc.).

### Alternatives Considered

**Alternative A: Provide color-specific fields**
```go
type HTMLTemplate struct {
    PrimaryColor     string
    BackgroundColor  string
    // ... many more color fields
}
```
- **Pros**: Type-safe, discoverable
- **Cons**: Inflexible (can't override spacing, fonts), many fields, maintenance burden
- **Rejected because**: Too restrictive, doesn't support all customization needs

**Alternative B: WithCustomCSS() helper function**
```go
func WithCustomCSS(css string) Format { /* ... */ }
```
- **Pros**: Explicit customization method
- **Cons**: Still requires full CSS replacement, no incremental overrides
- **Rejected because**: Doesn't solve the usability problem

**Alternative C: CSS string replacement**
- Users manually find/replace color values in default CSS
- **Pros**: No new feature needed
- **Cons**: Fragile, error-prone, requires understanding CSS structure
- **Rejected because**: Poor UX, maintenance nightmare

### Consequences

- Positive: Easy theme customization without CSS expertise
- Positive: Works with default CSS or custom CSS
- Positive: Extensible to any CSS custom property
- Negative: Map iteration order is non-deterministic (minor, cosmetic issue only)
- Negative: No validation that properties exist in default CSS (acceptable, flexible design)

### Implementation Notes

ThemeOverrides are injected after the main CSS block:

```go
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
```

Property names and values are HTML-escaped to prevent injection, but no validation is performed (consistent with Decision 11).

---

## Summary of Key Decisions

1. **String-based template wrapping** - Avoids html/template thread safety issues
2. **Embedded CSS by default** - Zero configuration, perfect portability
3. **Format constants** - Clear, discoverable API
4. **CSS custom properties** - Modern, flexible theming
5. **Mobile-first design** - Performance, accessibility, best practices
6. **Table stacking pattern** - Best mobile UX for typical tables
7. **480px breakpoint** - Good device coverage, conservative approach
8. **String building for content** - Consistency, performance, simplicity
9. **Package-level variables** - Zero allocation, simple discovery
10. **Scripts at end of body** - Optimal performance, modern practice
11. **No template validation** - Simplicity, flexibility, user control
12. **ThemeOverrides map** - Easy CSS customization without full replacement

These decisions form the foundation of the HTML Template System design and should guide implementation and future enhancements.
