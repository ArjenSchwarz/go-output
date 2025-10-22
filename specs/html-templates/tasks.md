---
references:
    - specs/html-templates/requirements.md
    - specs/html-templates/design.md
    - specs/html-templates/decision_log.md
---
# HTML Template System - Implementation Tasks (TDD)

## Phase 1: HTMLTemplate Core (TDD)

- [x] 1. Create HTMLTemplate struct (stub)
  - Create v2/html_template.go file
  - Define empty HTMLTemplate struct
  - Define package-level variables for DefaultHTMLTemplate, MinimalHTMLTemplate, MermaidHTMLTemplate (initialized with empty structs for now)

- [x] 2. Write tests for HTMLTemplate struct fields and defaults
  - Create v2/html_template_test.go file
  - Write test for default Title field value (should be "Output Report")
  - Write test for default Charset field value (should be "UTF-8")
  - Write test for default Language field value (should be "en")
  - Write test for default Viewport field value
  - Write tests for custom field overrides
  - Write tests for empty/nil field handling
  - Run tests (should fail)
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6), [2.7](requirements.md#2.7), [11.2](requirements.md#11.2), [11.3](requirements.md#11.3), [11.4](requirements.md#11.4), [13.1](requirements.md#13.1)

- [x] 3. Implement HTMLTemplate struct fields
  - Add all fields to HTMLTemplate struct (Title, Language, Charset, Viewport, Description, Author, MetaTags, CSS, ExternalCSS, ThemeOverrides, HeadExtra, BodyClass, BodyAttrs, BodyExtra)
  - Add godoc comments to struct and all fields
  - Add security warnings to CSS, HeadExtra, BodyExtra, ExternalCSS, BodyAttrs fields
  - Implement DefaultHTMLTemplate with default values (Title="Output Report", Language="en", Charset="UTF-8", Viewport with responsive settings)
  - Implement MinimalHTMLTemplate with defaults but empty CSS
  - Implement MermaidHTMLTemplate with defaults
  - Run tests (should pass)
  - Run golangci-lint and go fmt
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6), [2.7](requirements.md#2.7), [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [3.4](requirements.md#3.4), [3.5](requirements.md#3.5), [16.5](requirements.md#16.5), [16.6](requirements.md#16.6)

## Phase 2: Responsive CSS (TDD)

- [x] 4. Create CSS constants file (stub)
  - Create v2/html_css.go file
  - Define empty string constants for defaultResponsiveCSS and mermaidOptimizedCSS

- [x] 5. Write tests for CSS structure and requirements
  - Add tests to v2/html_template_test.go or create new test file
  - Write test verifying defaultResponsiveCSS contains CSS custom properties (check for ":root" and "--color-" variables)
  - Write test verifying mobile breakpoint exists (@media max-width: 480px)
  - Write test verifying table stacking styles exist (.data-table rules)
  - Write test verifying system font stack is used
  - Write test verifying WCAG AA color contrast (check color values)
  - Write test verifying mermaidOptimizedCSS contains mermaid-specific styles
  - Run tests (should fail)
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [6.4](requirements.md#6.4), [6.9](requirements.md#6.9), [6.10](requirements.md#6.10)

- [x] 6. Implement responsive CSS constants
  - Implement defaultResponsiveCSS constant with CSS custom properties in :root
  - Include mobile-first responsive design with 480px breakpoint
  - Implement table stacking pattern for mobile devices
  - Include WCAG AA compliant color contrast values
  - Use system font stack
  - Include styling for tables, sections, text content, mermaid containers
  - Implement mermaidOptimizedCSS constant for diagram rendering
  - Update DefaultHTMLTemplate to reference defaultResponsiveCSS
  - Update MermaidHTMLTemplate to reference mermaidOptimizedCSS
  - Run tests (should pass)
  - Run golangci-lint and go fmt
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [6.4](requirements.md#6.4), [6.5](requirements.md#6.5), [6.6](requirements.md#6.6), [6.7](requirements.md#6.7), [6.8](requirements.md#6.8), [6.9](requirements.md#6.9), [6.10](requirements.md#6.10), [3.6](requirements.md#3.6)

## Phase 3: Template Wrapping (TDD)

- [x] 7. Write tests for template wrapping function
  - Create test file or add to existing html_template_test.go
  - Write test for DOCTYPE declaration in output
  - Write test for html element with lang attribute
  - Write test for head section with charset meta tag
  - Write test for viewport meta tag
  - Write test for title tag with escaped content
  - Write test for description and author meta tags
  - Write test for custom MetaTags map rendering
  - Write test for ExternalCSS link tags with HTML escaping
  - Write test for embedded CSS style tag
  - Write test for ThemeOverrides as separate :root style block
  - Write test for HeadExtra content injection (unescaped)
  - Write test for body element with BodyClass
  - Write test for BodyAttrs rendering
  - Write test for fragment content injection
  - Write test for BodyExtra content injection (unescaped)
  - Write test for nil template using DefaultHTMLTemplate
  - Write test for empty fields producing valid output
  - Run tests (should fail)
  - Requirements: [1.2](requirements.md#1.2), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [1.7](requirements.md#1.7), [1.8](requirements.md#1.8), [2.8](requirements.md#2.8), [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [7.4](requirements.md#7.4), [7.5](requirements.md#7.5), [7.6](requirements.md#7.6), [11.1](requirements.md#11.1), [11.5](requirements.md#11.5), [11.6](requirements.md#11.6)

- [x] 8. Write tests for HTML escaping in template fields
  - Write test for XSS prevention in Title field (include <script> tag in input)
  - Write test for XSS prevention in Description and Author fields
  - Write test for XSS prevention in MetaTags names and values
  - Write test for XSS prevention in BodyClass
  - Write test for XSS prevention in BodyAttrs names and values
  - Write test for XSS prevention in ThemeOverrides names and values
  - Write test verifying CSS field is NOT escaped
  - Write test verifying HeadExtra field is NOT escaped
  - Write test verifying BodyExtra field is NOT escaped
  - Write test for special characters in all fields
  - Run tests (should fail)
  - Requirements: [2.8](requirements.md#2.8), [11.7](requirements.md#11.7), [13.2](requirements.md#13.2)

- [x] 9. Implement wrapInTemplate function
  - Add wrapInTemplate() function to v2/html_renderer.go
  - Use strings.Builder for efficient string concatenation
  - Implement DOCTYPE and html element with escaped lang attribute
  - Implement head section with meta tags (charset, viewport, description, author)
  - Implement custom MetaTags map iteration and injection with escaping
  - Implement ExternalCSS link tags with html.EscapeString()
  - Implement embedded CSS style tag (no escaping)
  - Implement ThemeOverrides as separate style block with :root and escaped properties
  - Implement HeadExtra content injection (no escaping)
  - Implement body element with escaped BodyClass and escaped BodyAttrs
  - Inject fragment content parameter (already escaped by caller)
  - Implement BodyExtra content injection (no escaping)
  - Handle nil template by using DefaultHTMLTemplate
  - Run tests (should pass)
  - Run golangci-lint and go fmt
  - Requirements: [1.2](requirements.md#1.2), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [1.7](requirements.md#1.7), [1.8](requirements.md#1.8), [2.8](requirements.md#2.8), [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [7.4](requirements.md#7.4), [7.5](requirements.md#7.5), [7.6](requirements.md#7.6)

## Phase 4: Renderer Integration (TDD)

- [x] 10. Write tests for htmlRenderer template fields and integration
  - Write test for htmlRenderer with useTemplate=true calling wrapInTemplate
  - Write test for htmlRenderer with useTemplate=false skipping template wrapping
  - Write test verifying Mermaid script injection happens before template wrapping
  - Write test for template wrapping as final step in pipeline
  - Write test verifying Mermaid script appears at end of body in wrapped output
  - Run tests (should fail)
  - Requirements: [1.1](requirements.md#1.1), [7.7](requirements.md#7.7), [8.5](requirements.md#8.5), [8.6](requirements.md#8.6)

- [x] 11. Add template fields to htmlRenderer and integrate wrapping
  - Add useTemplate bool field to htmlRenderer struct
  - Add template *HTMLTemplate field to htmlRenderer struct
  - Update htmlRenderer.Render() to call wrapInTemplate() when useTemplate is true
  - Ensure Mermaid script injection happens before template wrapping
  - Preserve existing fragment rendering logic when useTemplate is false
  - Ensure template wrapping is the final step in render pipeline
  - Run tests (should pass)
  - Run golangci-lint and go fmt
  - Requirements: [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.3](requirements.md#8.3), [8.4](requirements.md#8.4), [8.5](requirements.md#8.5), [8.6](requirements.md#8.6), [1.1](requirements.md#1.1), [7.7](requirements.md#7.7)

## Phase 5: Format API (TDD)

- [x] 12. Write tests for format constants and constructors
  - Write test verifying HTML format uses template by default (check useTemplate=true)
  - Write test verifying HTMLFragment format skips template wrapping (check useTemplate=false)
  - Write test for HTMLWithTemplate(nil) enabling fragment mode
  - Write test for HTMLWithTemplate(custom) using custom template
  - Write test verifying format constructor creates new renderer instance (not shared)
  - Run tests (should fail)
  - Requirements: [1.1](requirements.md#1.1), [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [13.4](requirements.md#13.4), [13.5](requirements.md#13.5), [13.6](requirements.md#13.6)

- [x] 13. Implement format constants and constructors
  - Update HTML format constant in v2/renderer.go to use htmlRenderer with useTemplate=true and template=DefaultHTMLTemplate
  - Create HTMLFragment format constant with useTemplate=false
  - Implement HTMLWithTemplate(template *HTMLTemplate) constructor function
  - Handle nil template in HTMLWithTemplate to set useTemplate=false
  - Ensure format constructors return new renderer instances (not reusing package-level renderers)
  - Add godoc comment to HTMLWithTemplate() function
  - Run tests (should pass)
  - Run golangci-lint and go fmt
  - Requirements: [1.1](requirements.md#1.1), [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5), [4.6](requirements.md#4.6), [4.7](requirements.md#4.7), [4.8](requirements.md#4.8), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [16.5](requirements.md#16.5), [16.6](requirements.md#16.6)

## Phase 6: Integration Testing

- [x] 14. Write integration tests for full document generation
  - Create v2/html_integration_test.go file
  - Write test for document with table content producing valid HTML5 structure
  - Write test for document with multiple content types (tables, text, sections)
  - Write test for document with nested sections
  - Write test verifying DOCTYPE declaration presence in output
  - Write test verifying html, head, body tag structure
  - Write test verifying meta tags are properly injected
  - Write test for complete end-to-end rendering pipeline
  - Run tests (should pass since implementation is complete)
  - Requirements: [13.7](requirements.md#13.7), [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4)

- [x] 15. Write integration tests for Mermaid chart handling
  - Add tests to v2/html_integration_test.go
  - Write test verifying Mermaid chart with template includes script at end of body
  - Write test verifying Mermaid chart in fragment mode includes script
  - Write test for multiple charts in same document
  - Write test verifying script injection order (content, then scripts, then closing tags)
  - Run tests (should pass)
  - Requirements: [13.8](requirements.md#13.8), [5.6](requirements.md#5.6), [7.7](requirements.md#7.7)

- [x] 16. Write integration tests for template customization
  - Add tests to v2/html_integration_test.go
  - Write test verifying custom title appears in output
  - Write test verifying custom CSS overrides defaults
  - Write test verifying external stylesheet links are rendered
  - Write test verifying meta tag injection (description, author, custom)
  - Write test verifying HeadExtra content injection
  - Write test verifying BodyExtra content injection
  - Write test verifying ThemeOverrides generate separate style block
  - Write test verifying BodyClass and BodyAttrs are applied to body element
  - Run tests (should pass)
  - Requirements: [13.4](requirements.md#13.4), [1.7](requirements.md#1.7)

- [x] 17. Write tests for edge cases
  - Write test for empty document producing valid HTML structure
  - Write test for empty CSS field producing template without style tags
  - Write test for empty ExternalCSS slice producing template without link tags
  - Write test for missing optional fields using defaults
  - Run tests (should pass)
  - Requirements: [13.9](requirements.md#13.9), [11.1](requirements.md#11.1), [11.5](requirements.md#11.5), [11.6](requirements.md#11.6)

- [x] 18. Write thread safety tests
  - Write test for multiple goroutines rendering same template concurrently
  - Write test for template instance reuse across renders
  - Write test verifying no shared mutable state in rendering path
  - Run tests with -race flag to detect data races
  - Verify all tests pass with race detector enabled
  - Requirements: [13.10](requirements.md#13.10), [9.1](requirements.md#9.1), [9.2](requirements.md#9.2), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4)

## Phase 7: Final Validation

- [x] 19. Final code quality validation
  - Run golangci-lint on all new code and fix any issues
  - Run go fmt on all files
  - Run modernize tool validation and apply fixes
  - Verify all code follows project Go language rules
  - Run all tests (unit + integration) with -race flag
  - Verify test coverage meets project standards
  - Requirements: [16.1](requirements.md#16.1), [16.2](requirements.md#16.2), [16.3](requirements.md#16.3), [16.4](requirements.md#16.4)
