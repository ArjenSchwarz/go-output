package output

// HTMLTemplate defines the structure for HTML document templates with metadata,
// styling options, and customization points for full HTML document generation.
//
// All string fields containing user-controlled content are HTML-escaped before
// inclusion in the final document to prevent XSS injection. Fields containing
// pre-validated content (CSS, scripts) are included as-is and are the user's
// responsibility to validate for safety.
type HTMLTemplate struct {
	// Document metadata
	//
	// Title is the page title displayed in the browser tab and document head.
	// Defaults to "Output Report" if empty. This field is HTML-escaped to prevent XSS.
	Title string

	// Language is the HTML lang attribute value that specifies the page language.
	// Defaults to "en" if empty. This field is HTML-escaped to prevent XSS.
	Language string

	// Charset is the character encoding for the HTML document.
	// Defaults to "UTF-8" if empty. This field is HTML-escaped to prevent XSS.
	Charset string

	// Meta tags
	//
	// Viewport is the viewport meta tag content controlling device rendering.
	// Defaults to "width=device-width, initial-scale=1.0" if empty.
	// This field is HTML-escaped to prevent XSS.
	Viewport string

	// Description is the page description meta tag for search engines and social media.
	// Optional. This field is HTML-escaped to prevent XSS.
	Description string

	// Author is the page author meta tag.
	// Optional. This field is HTML-escaped to prevent XSS.
	Author string

	// MetaTags contains additional custom meta tags as name → content pairs.
	// Optional. Both names and values are HTML-escaped to prevent XSS.
	MetaTags map[string]string

	// Styling
	//
	// CSS contains embedded CSS content included directly in a <style> tag.
	// Optional. WARNING: This content is NOT escaped or validated. Ensure this contains
	// only trusted CSS from reliable sources. Malicious CSS can break document structure
	// via </style> injection or potentially leak data.
	CSS string

	// ExternalCSS contains URLs to external stylesheets to be linked in the <head>.
	// Optional. WARNING: URLs are HTML-escaped but NOT validated for safety. Ensure URLs
	// use http:// or https:// schemes. JavaScript URLs (javascript:) are possible and
	// create XSS risks.
	ExternalCSS []string

	// ThemeOverrides contains CSS custom property overrides applied after the main CSS.
	// Use this to customize theme colors and other CSS variables without replacing
	// the entire CSS block. Example: map[string]string{"--color-primary": "#ff0000"}.
	// Optional. Property names and values are HTML-escaped to prevent injection,
	// but no validation is performed on the CSS content itself.
	ThemeOverrides map[string]string

	// Additional content
	//
	// HeadExtra contains additional HTML content injected into the <head> section.
	// Optional. WARNING: This content is NOT escaped or validated. Only use with content
	// from trusted sources. Including user-provided content creates XSS vulnerabilities.
	// Common use cases: custom fonts, analytics scripts, meta tags.
	HeadExtra string

	// BodyClass is the CSS class attribute value for the <body> element.
	// Optional. This field is HTML-escaped to prevent XSS.
	BodyClass string

	// BodyAttrs contains additional attributes for the <body> element as name → value pairs.
	// Optional. WARNING: Attribute names and values are HTML-escaped but event handlers
	// (onclick, onload, etc.) are allowed. Ensure this map contains only safe attributes.
	BodyAttrs map[string]string

	// BodyExtra contains additional HTML content injected before the closing </body> tag.
	// Optional. WARNING: This content is NOT escaped or validated. Only use with content
	// from trusted sources. Including user-provided content creates XSS vulnerabilities.
	// Common use cases: analytics scripts, tracking pixels, footer content.
	BodyExtra string
}

// DefaultHTMLTemplate provides modern responsive styling with sensible defaults.
// It includes embedded CSS with mobile-first responsive design, modern CSS features
// (flexbox, grid), and accessibility support. The CSS uses custom properties for theming.
// This is initialized in html_css.go's init() function to ensure CSS constants are set.
var DefaultHTMLTemplate *HTMLTemplate

// MinimalHTMLTemplate provides basic HTML structure with no styling.
// Use this template when you want to apply your own CSS or embed in pages with
// existing styling.
// This is initialized in html_css.go's init() function.
var MinimalHTMLTemplate *HTMLTemplate

// MermaidHTMLTemplate is optimized for diagram rendering with Mermaid.js integration.
// It includes CSS optimizations for mermaid diagram display and all features of DefaultHTMLTemplate.
// This is initialized in html_css.go's init() function to ensure CSS constants are set.
var MermaidHTMLTemplate *HTMLTemplate

// CSS constants will be defined in html_css.go
// These are referenced here and will be populated by that file.
var (
	defaultResponsiveCSS string
	mermaidOptimizedCSS  string
)
