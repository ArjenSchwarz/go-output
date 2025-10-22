# HTML Template System Requirements

## Introduction

The HTML Template System feature adds full HTML document template support to the v2 HTML renderer, transforming it from outputting HTML fragments to generating complete, browser-ready HTML documents by default. This addresses a breaking change from v1 (which outputted full documents) and provides a modern, extensible templating system with responsive CSS, customization options, and support for embedded content like Mermaid diagrams.

The system provides three operational modes:
1. **Default mode**: Full HTML documents with modern responsive styling
2. **Custom template mode**: Full HTML documents with user-provided templates
3. **Fragment mode**: HTML fragments for embedding in existing pages

## Requirements

### 1. Default HTML Document Output

**User Story:** As a developer migrating from v1 or creating new reports, I want the HTML renderer to output complete HTML documents by default, so that I can immediately open the output in a browser without additional processing.

**Acceptance Criteria:**

1. <a name="1.1"></a>The HTML renderer SHALL output full HTML5 documents by default when `output.HTML.Renderer.Render(ctx, doc)` is called
2. <a name="1.2"></a>The default output SHALL include DOCTYPE declaration, html, head, and body tags
3. <a name="1.3"></a>The default output SHALL include embedded responsive CSS styling
4. <a name="1.4"></a>The default output SHALL be valid HTML5 that passes W3C validation
5. <a name="1.5"></a>The default template SHALL use UTF-8 character encoding
6. <a name="1.6"></a>The default template SHALL include a responsive viewport meta tag
7. <a name="1.7"></a>The default template SHALL have a configurable page title with default value "Output Report"
8. <a name="1.8"></a>The default template SHALL set the HTML lang attribute to "en" by default

### 2. HTMLTemplate Structure

**User Story:** As a developer, I want a structured way to define HTML document templates with metadata and styling options, so that I can customize the output documents for different use cases.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL provide an HTMLTemplate struct with document metadata fields (Title, Language, Charset)
2. <a name="2.2"></a>The HTMLTemplate struct SHALL support embedded CSS via a CSS string field
3. <a name="2.3"></a>The HTMLTemplate struct SHALL support external stylesheet links via an ExternalCSS string slice
4. <a name="2.4"></a>The HTMLTemplate struct SHALL support additional head content via a HeadExtra string field
5. <a name="2.5"></a>The HTMLTemplate struct SHALL support body element customization via BodyClass string and BodyAttrs map fields
6. <a name="2.6"></a>The HTMLTemplate struct SHALL support additional body content via a BodyExtra string field
7. <a name="2.7"></a>The HTMLTemplate struct SHALL support meta tags for viewport, description, and author
8. <a name="2.8"></a>All HTMLTemplate string fields SHALL properly escape HTML special characters to prevent injection

### 3. Built-in Template Variants

**User Story:** As a developer, I want pre-configured template options for common use cases, so that I can quickly select appropriate styling without creating templates from scratch.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL provide a DefaultHTMLTemplate with modern responsive CSS
2. <a name="3.2"></a>The system SHALL provide a MinimalHTMLTemplate with no styling
3. <a name="3.3"></a>The system SHALL provide a MermaidHTMLTemplate optimized for diagram rendering
4. <a name="3.4"></a>The DefaultHTMLTemplate SHALL be used when no template is explicitly specified
5. <a name="3.5"></a>All built-in templates SHALL use sensible defaults for Title, Charset, and Language fields
6. <a name="3.6"></a>The MermaidHTMLTemplate SHALL include CSS optimizations for mermaid diagram display
7. <a name="3.7"></a>Built-in templates SHALL be exported as package-level variables for easy access

### 4. Custom Template Support

**User Story:** As a developer, I want to provide my own HTML templates with custom styling and metadata, so that I can integrate the output with my organization's branding and requirements.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL provide an HTMLWithTemplate(template) constructor function
2. <a name="4.2"></a>The HTMLWithTemplate function SHALL accept a pointer to an HTMLTemplate
3. <a name="4.3"></a>The HTMLWithTemplate function SHALL return a Format that uses the provided template
4. <a name="4.4"></a>Custom templates SHALL support all fields available in the HTMLTemplate struct
5. <a name="4.5"></a>Custom templates SHALL allow partial field specification with sensible defaults for omitted fields
6. <a name="4.6"></a>Custom templates SHALL support embedding custom CSS via the CSS field
7. <a name="4.7"></a>Custom templates SHALL support linking external stylesheets via the ExternalCSS field
8. <a name="4.8"></a>Custom templates SHALL support injecting analytics or other scripts via HeadExtra and BodyExtra fields

### 5. Fragment Mode Support

**User Story:** As a developer embedding HTML output in existing web pages, I want to generate HTML fragments without document structure, so that I can integrate the content into my application's existing HTML.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL provide an HTMLFragment format constant for fragment-only output
2. <a name="5.2"></a>The HTMLFragment format SHALL output content without DOCTYPE, html, head, or body tags
3. <a name="5.3"></a>The HTMLWithTemplate function SHALL accept nil to indicate fragment mode
4. <a name="5.4"></a>Fragment mode output SHALL be identical to the current v2 HTML renderer behavior
5. <a name="5.5"></a>Fragment mode SHALL preserve all content rendering including classes and structure
6. <a name="5.6"></a>Fragment mode SHALL include mermaid script injection if charts are present

### 6. Modern Responsive CSS

**User Story:** As a developer, I want modern, responsive CSS that works across devices, so that my HTML reports display well on desktop, tablet, and mobile devices.

**Acceptance Criteria:**

1. <a name="6.1"></a>The default CSS SHALL use CSS custom properties (variables) for theming
2. <a name="6.2"></a>The default CSS SHALL use modern CSS features (flexbox, grid) instead of legacy display hacks
3. <a name="6.3"></a>The default CSS SHALL implement mobile-first responsive design
4. <a name="6.4"></a>The default CSS SHALL include responsive table styling that stacks on mobile devices
5. <a name="6.5"></a>The default CSS SHALL include styling for mermaid diagram containers
6. <a name="6.6"></a>The default CSS SHALL include styling for text content and headers
7. <a name="6.7"></a>The default CSS SHALL include styling for sections with proper spacing
8. <a name="6.8"></a>The default CSS SHALL use a mobile breakpoint of 480px for table stacking
9. <a name="6.9"></a>The default CSS SHALL use system font stack for performance
10. <a name="6.10"></a>The default CSS SHALL include accessible focus states and sufficient color contrast

### 7. Template Rendering Integration

**User Story:** As a developer, I want template rendering to integrate seamlessly with existing HTML content rendering, so that all content types and features continue to work correctly.

**Acceptance Criteria:**

1. <a name="7.1"></a>The renderer SHALL first render document content to HTML fragments
2. <a name="7.2"></a>The renderer SHALL inject mermaid scripts after fragment rendering if charts are present
3. <a name="7.3"></a>The renderer SHALL wrap fragments in the HTML template as the final step
4. <a name="7.4"></a>The template wrapping SHALL inject content at a designated placeholder location
5. <a name="7.5"></a>The template wrapping SHALL preserve all fragment HTML structure and attributes
6. <a name="7.6"></a>The template wrapping SHALL not modify or escape the already-rendered HTML content
7. <a name="7.7"></a>Mermaid scripts SHALL remain at the end of the body element after template wrapping

### 8. Renderer Configuration

**User Story:** As a developer, I want to configure whether the HTML renderer uses templates, so that I can control the output format based on my use case.

**Acceptance Criteria:**

1. <a name="8.1"></a>The htmlRenderer struct SHALL include a useTemplate boolean field
2. <a name="8.2"></a>The htmlRenderer struct SHALL include a template pointer field for custom templates
3. <a name="8.3"></a>The useTemplate field SHALL default to true for the standard HTML format
4. <a name="8.4"></a>The useTemplate field SHALL be false for the HTMLFragment format
5. <a name="8.5"></a>When useTemplate is true and template is nil, the renderer SHALL use DefaultHTMLTemplate
6. <a name="8.6"></a>When useTemplate is false, the renderer SHALL output fragments regardless of the template field

### 9. Thread Safety

**User Story:** As a developer using concurrent rendering, I want template rendering to be thread-safe, so that I can safely render multiple documents in parallel.

**Acceptance Criteria:**

1. <a name="9.1"></a>Template rendering operations SHALL be safe for concurrent use
2. <a name="9.2"></a>Multiple goroutines SHALL be able to render documents with the same template concurrently
3. <a name="9.3"></a>Template wrapping SHALL not modify shared template state
4. <a name="9.4"></a>Each render operation SHALL use independent buffer allocation

### 10. Error Handling

**User Story:** As a developer, I want clear error messages when template rendering fails, so that I can quickly identify and fix configuration issues.

**Acceptance Criteria:**

1. <a name="10.1"></a>Template rendering errors SHALL be returned with descriptive error messages
2. <a name="10.2"></a>Invalid template syntax SHALL produce errors that identify the specific problem
3. <a name="10.3"></a>Template rendering SHALL not panic under any input conditions
4. <a name="10.4"></a>Errors SHALL wrap underlying template execution errors with context

### 11. Empty and Edge Case Handling

**User Story:** As a developer, I want template rendering to handle edge cases gracefully, so that unusual inputs don't cause failures or invalid output.

**Acceptance Criteria:**

1. <a name="11.1"></a>Empty content SHALL produce valid HTML with just the template structure
2. <a name="11.2"></a>Missing Title field SHALL default to "Output Report"
3. <a name="11.3"></a>Missing Charset field SHALL default to "UTF-8"
4. <a name="11.4"></a>Missing Language field SHALL default to "en"
5. <a name="11.5"></a>Empty CSS field SHALL produce template without style tags
6. <a name="11.6"></a>Empty ExternalCSS slice SHALL produce template without link tags
7. <a name="11.7"></a>Special characters in template fields SHALL be properly HTML-escaped
8. <a name="11.8"></a>Large documents SHALL render without excessive memory allocation

### 12. Backward Compatibility with v1

**User Story:** As a developer migrating from v1 to v2, I want minimal code changes required, so that I can upgrade with confidence and reduced effort.

**Acceptance Criteria:**

1. <a name="12.1"></a>The v2 default HTML output SHALL be visually similar to v1 output
2. <a name="12.2"></a>The v2 default template SHALL include responsive table styling equivalent to v1
3. <a name="12.3"></a>The v2 API SHALL require minimal changes from v1 usage patterns
4. <a name="12.4"></a>The v2 CSS SHALL maintain the v1 color scheme and visual design

### 13. Testing Requirements

**User Story:** As a maintainer, I want test coverage for all template functionality, so that I can confidently make changes without introducing regressions.

**Acceptance Criteria:**

1. <a name="13.1"></a>The system SHALL include unit tests for template field population
2. <a name="13.2"></a>The system SHALL include unit tests for CSS and meta tag injection
3. <a name="13.3"></a>The system SHALL include unit tests for each built-in template
4. <a name="13.4"></a>The system SHALL include unit tests for custom template rendering
5. <a name="13.5"></a>The system SHALL include unit tests for fragment mode
6. <a name="13.6"></a>The system SHALL include unit tests for nil template handling
7. <a name="13.7"></a>The system SHALL include integration tests for full document generation
8. <a name="13.8"></a>The system SHALL include integration tests for mermaid charts with templates
9. <a name="13.9"></a>The system SHALL include tests for empty content edge cases
10. <a name="13.10"></a>The system SHALL include tests for concurrent rendering

### 14. Documentation Requirements

**User Story:** As a developer using this library, I want documentation with clear examples, so that I can quickly understand and implement template customization.

**Acceptance Criteria:**

1. <a name="14.1"></a>Documentation SHALL include examples of default template usage
2. <a name="14.2"></a>Documentation SHALL include examples of custom title configuration
3. <a name="14.3"></a>Documentation SHALL include examples of custom CSS usage
4. <a name="14.4"></a>Documentation SHALL include examples of external stylesheet linking
5. <a name="14.5"></a>Documentation SHALL include examples of analytics integration
6. <a name="14.6"></a>Documentation SHALL include examples of fragment mode usage for embedding in existing pages
7. <a name="14.7"></a>Documentation SHALL include examples of mermaid-optimized templates
8. <a name="14.8"></a>Documentation SHALL include migration guide for v1 users
9. <a name="14.9"></a>Documentation SHALL document all HTMLTemplate struct fields
10. <a name="14.10"></a>Documentation SHALL document all built-in template variants

### 15. File Organization

**User Story:** As a maintainer, I want clear file organization for template-related code, so that the codebase remains maintainable and easy to navigate.

**Acceptance Criteria:**

1. <a name="15.1"></a>Template definitions SHALL be in a new v2/html_template.go file
2. <a name="15.2"></a>CSS constants SHALL be in a new v2/html_css.go file
3. <a name="15.3"></a>Template rendering logic SHALL be added to existing v2/html_renderer.go
4. <a name="15.4"></a>Template tests SHALL be in a new v2/html_template_test.go file
5. <a name="15.5"></a>Integration tests SHALL be in a new v2/html_integration_test.go file

### 16. Code Quality Requirements

**User Story:** As a maintainer, I want all template code to meet project quality standards, so that the codebase remains clean and maintainable.

**Acceptance Criteria:**

1. <a name="16.1"></a>All template code SHALL pass golangci-lint with zero issues
2. <a name="16.2"></a>All template code SHALL be formatted with go fmt
3. <a name="16.3"></a>All template code SHALL pass modernize tool validation
4. <a name="16.4"></a>All template code SHALL follow project Go language rules
5. <a name="16.5"></a>All template code SHALL include appropriate godoc comments
6. <a name="16.6"></a>All exported template types and functions SHALL have documentation
