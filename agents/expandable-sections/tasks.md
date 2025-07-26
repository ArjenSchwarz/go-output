# Cross-Format Collapsible Content System Implementation Tasks

This document provides an actionable implementation plan for the cross-format collapsible content system based on the requirements and design documents. Each task is designed to be completed incrementally with test-driven development.

## Implementation Tasks

### 1. Core Collapsible Interface and Value Implementation

- [x] 1.1 Create core CollapsibleValue interface in `v2/collapsible.go`
  - Implement Summary() method returning string for collapsed view (Requirement 1.1)
  - Implement Details() method returning any for structured data support (Requirement 1.2)
  - Implement IsExpanded() method returning bool for default expansion state (Requirement 1.3)
  - Implement FormatHint() method returning format-specific rendering hints (Requirement 1.4)

- [x] 1.2 Implement DefaultCollapsibleValue struct with configuration options
  - Add summary, details, defaultExpanded, formatHints fields (Requirement 1.6)
  - Add maxDetailLength and truncateIndicator for character limits (Requirement 10.6, 10.7)
  - Implement all CollapsibleValue interface methods with proper edge case handling (Requirement 11.1-11.6)

- [x] 1.3 Create CollapsibleValue constructor with functional options pattern
  - Implement NewCollapsibleValue with variadic CollapsibleOption parameters
  - Add WithExpanded, WithMaxLength, WithFormatHint option functions
  - Apply default values: maxDetailLength=500, truncateIndicator="[...truncated]" (Requirement 10.6, 10.7)

- [x] 1.4 Write comprehensive unit tests for CollapsibleValue interface
  - Test Summary() with empty string fallback to "[no summary]" (Requirement 11.2)
  - Test Details() with nil handling and character truncation (Requirement 11.1, 11.6)
  - Test IsExpanded() behavior and FormatHint() functionality (Requirement 1.3, 1.4)
  - Test edge cases: nil details, empty summary, character limits

### 2. Field Formatter Integration and Helper Functions

- [x] 2.1 Update Field struct signature to support CollapsibleValue returns
  - Change Field.Formatter from `func(any) string` to `func(any) any` in `v2/schema.go` (Requirement 2.1)
  - Update documentation to reflect CollapsibleValue return capability (Requirement 2.2)
  - Ensure backward compatibility with existing formatters (Requirement 12.1-12.5)

- [x] 2.2 Implement CollapsibleFormatter helper function in `v2/collapsible_formatters.go`
  - Create function that takes summaryTemplate and detailFunc parameters (Requirement 2.4)
  - Return field formatter that produces CollapsibleValue instances
  - Handle nil detailFunc by returning original value unchanged (Requirement 2.5)

- [x] 2.3 Create pre-built formatter functions for common patterns
  - Implement ErrorListFormatter for array of strings/errors (Requirement 9.1, 9.2)
  - Implement FilePathFormatter for long file paths with 50-character threshold (Requirement 9.3, 9.4)
  - Implement JSONFormatter for complex data structures with configurable size limit (Requirement 9.5)
  - Add graceful fallback for incompatible data types (Requirement 9.6)

- [x] 2.4 Write unit tests for formatter helper functions
  - Test CollapsibleFormatter with various template and detail function combinations
  - Test ErrorListFormatter with empty arrays, string arrays, and error arrays
  - Test FilePathFormatter with short paths (no collapsible) and long paths
  - Test JSONFormatter with small and large data structures

### 3. Base Renderer Enhancement for Collapsible Value Processing

- [x] 3.1 Add collapsible value detection to base renderer in `v2/base_renderer.go`
  - Implement processFieldValue method that applies Field.Formatter (Requirement 2.1)
  - Add type assertion to detect CollapsibleValue interface (Requirement 2.1)
  - Maintain backward compatibility for non-CollapsibleValue returns (Requirement 2.2)

- [x] 3.2 Create RendererConfig struct for collapsible-specific configuration
  - Add ForceExpansion field for global expansion control (Requirement 13.1)
  - Add MaxDetailLength and TruncateIndicator for character limits (Requirement 14.2, 14.3)
  - Add TableHiddenIndicator and HTMLCSSClasses for format-specific settings (Requirement 14.1)
  - Implement DefaultRendererConfig with sensible defaults

- [x] 3.3 Add renderer configuration constructors
  - Create NewMarkdownRendererWithCollapsible, NewTableRendererWithCollapsible functions
  - Enable per-renderer configuration of collapsible behavior (Requirement 14.5)
  - Ensure independent configuration across multiple renderer instances (Requirement 14.5)

### 4. Markdown Renderer Collapsible Support

- [x] 4.1 Implement CollapsibleValue detection and rendering in `v2/markdown_renderer.go`
  - Add formatCellValue method with CollapsibleValue type checking (Requirement 3.1)
  - Generate `<details><summary>...</summary>...</details>` HTML structure (Requirement 3.1)
  - Add `open` attribute when IsExpanded() returns true or global expansion enabled (Requirement 3.2, 13.1)

- [x] 4.2 Implement markdown-specific detail formatting
  - Handle string details with proper markdown escaping (Requirement 3.3)
  - Convert string arrays to `<br/>`-separated content (Requirement 3.4)
  - Format map structures as key-value pairs with HTML formatting (Requirement 3.5)
  - Ensure table structure maintenance with collapsible content (Requirement 3.6)

- [x] 4.3 Write tests for markdown collapsible rendering
  - Test `<details>` element generation with proper open attribute handling
  - Test various detail types: strings, arrays, maps, complex structures
  - Test markdown character escaping and table cell compatibility
  - Test global expansion override behavior

### 5. JSON and YAML Renderer Collapsible Support

- [x] 5.1 Implement CollapsibleValue handling in JSON renderer in `v2/json_yaml_renderer.go`
  - Add formatValueForJSON method with CollapsibleValue detection (Requirement 4.1)
  - Generate JSON object with "type": "collapsible" indicator (Requirement 4.1)
  - Include "summary", "details", and "expanded" fields (Requirement 4.2)
  - Incorporate format hints as additional object properties (Requirement 4.3)

- [x] 5.2 Implement CollapsibleValue handling in YAML renderer
  - Add formatValueForYAML method creating YAML mapping (Requirement 5.1)
  - Include summary, details, and expanded fields in YAML structure (Requirement 5.1)
  - Handle nested details content with proper YAML indentation (Requirement 5.3)
  - Convert string arrays to YAML sequences and maps to YAML mappings (Requirement 5.4, 5.5)

- [x] 5.3 Write tests for JSON/YAML collapsible rendering
  - Test JSON structure generation with type indicators and format hints
  - Test YAML mapping creation with proper field inclusion
  - Test key order preservation with collapsible values (Requirement 4.4)
  - Test streaming capabilities with large datasets (Requirement 4.5)

### 6. Table Renderer Collapsible Support with Configuration

- [x] 6.1 Implement CollapsibleValue handling in table renderer in `v2/table_renderer.go`
  - Add formatCellValue method with CollapsibleValue type assertion (Requirement 6.1)
  - Display summary with configurable hidden details indicator when collapsed (Requirement 6.1, 6.6)
  - Show both summary and indented details when expanded (Requirement 6.2)
  - Apply global expansion override when configured (Requirement 6.7, 13.1)

- [x] 6.2 Implement table-specific detail formatting
  - Add formatDetailsForTable method with proper indentation (Requirement 6.3)
  - Handle multi-line details while maintaining table structure (Requirement 6.4)
  - Ensure consistent formatting across multiple collapsible cells (Requirement 6.5)

- [x] 6.3 Write tests for table collapsible rendering
  - Test summary display with configurable expansion indicators
  - Test detail indentation and multi-line content handling
  - Test global expansion override functionality
  - Test custom indicator text configuration

### 7. HTML Renderer Collapsible Support

- [x] 7.1 Implement CollapsibleValue handling in HTML renderer in `v2/html_renderer.go`
  - Add formatCellValue method generating `<details>` elements (Requirement 7.1)
  - Include semantic CSS classes: "collapsible-cell", "collapsible-summary", "collapsible-details" (Requirement 7.2)
  - Add `open` attribute when IsExpanded() returns true (Requirement 7.3)

- [x] 7.2 Implement HTML-specific detail formatting
  - Add formatDetailsAsHTML method with proper HTML escaping (Requirement 7.4)
  - Generate semantic HTML elements for structured data (Requirement 7.5)
  - Convert arrays to `<ul>` lists and maps to `<dl>` definition lists (Requirement 7.5)

- [x] 7.3 Write tests for HTML collapsible rendering
  - Test `<details>` element generation with proper CSS classes
  - Test HTML escaping and semantic element generation
  - Test open attribute handling and accessibility compliance

### 8. CSV Renderer with Detail Column Generation

- [x] 8.1 Implement automatic detail column detection in CSV renderer in `v2/csv_renderer.go`
  - Add handleCollapsibleFields method to analyze table schema (Requirement 8.1)
  - Create additional "_details" columns for collapsible fields (Requirement 8.1)
  - Maintain original column order and append detail columns adjacently (Requirement 8.4)

- [x] 8.2 Implement CollapsibleValue processing for CSV output
  - Place summary content in original column (Requirement 8.2)
  - Place details content in corresponding detail column (Requirement 8.2)
  - Leave detail columns empty for non-collapsible values (Requirement 8.3)

- [x] 8.3 Implement detail content flattening for CSV compatibility
  - Add flattenDetails method for complex structure conversion (Requirement 8.5)
  - Handle strings, arrays, and maps with appropriate separators
  - Ensure CSV compatibility and spreadsheet application support

- [x] 8.4 Write tests for CSV collapsible rendering
  - Test automatic detail column generation and schema modification
  - Test summary/detail column placement and empty cell handling
  - Test complex structure flattening and CSV format compliance

### 9. CollapsibleSection Interface and Implementation

- [x] 9.1 Create CollapsibleSection interface in `v2/collapsible_section.go`
  - Implement Title() method returning section title/summary (Requirement 15.1)
  - Implement Content() method returning []Content for nested items (Requirement 15.3)
  - Implement IsExpanded() method for section-level expansion control (Requirement 15.10)
  - Implement Level() method supporting 0-3 nesting levels (Requirement 15.9)
  - Implement FormatHint() method for section-specific rendering hints

- [x] 9.2 Implement DefaultCollapsibleSection struct
  - Add title, content, defaultExpanded, level, formatHints fields
  - Implement NewCollapsibleSection constructor with functional options
  - Add WithSectionExpanded, WithSectionLevel, WithSectionFormatHint options
  - Ensure content array copying to prevent external modification

- [x] 9.3 Create helper functions for common section patterns
  - Implement NewCollapsibleTable for single table sections (Requirement 15.2)
  - Implement NewCollapsibleMultiTable for multiple table sections
  - Implement NewCollapsibleReport for mixed content types (Requirement 15.3)

- [x] 9.4 Write unit tests for CollapsibleSection functionality
  - Test Title(), Content(), IsExpanded(), Level() methods
  - Test nesting level limits and hierarchical structure support
  - Test helper function creation of various section types

### 10. CollapsibleSection Renderer Integration

- [x] 10.1 Implement CollapsibleSection support in markdown renderer
  - Add renderCollapsibleSection method generating nested `<details>` structure (Requirement 15.4)
  - Render all nested content within collapsible section
  - Handle section title with proper markdown escaping

- [x] 10.2 Implement CollapsibleSection support in JSON/YAML renderers
  - Add renderCollapsibleSection methods creating structured data (Requirement 15.5)
  - Include type, title, level, expanded fields and content array
  - Handle nested content rendering and proper data structure creation

- [x] 10.3 Implement CollapsibleSection support in HTML renderer
  - Add renderCollapsibleSection method with semantic section elements (Requirement 15.6)
  - Generate proper HTML structure with CSS classes and accessibility
  - Handle nested content rendering within section containers

- [x] 10.4 Implement CollapsibleSection support in table renderer
  - Add renderCollapsibleSection method with section headers (Requirement 15.7)
  - Show section title with expansion indicator
  - Indent nested content when expanded, show collapsed indicator when collapsed

- [x] 10.5 Implement CollapsibleSection support in CSV renderer
  - Add renderCollapsibleSection method with metadata comments (Requirement 15.8)
  - Include section information as CSV comments or special rows
  - Handle table content within sections with appropriate context

### 11. Content Type Integration and Builder API Enhancement

- [x] 11.1 Add CollapsibleSection as content type in `v2/content.go`
  - Implement Content interface methods for CollapsibleSection
  - Add CollapsibleSection to content type enumeration
  - Ensure proper integration with existing content processing pipeline

- [x] 11.2 Enhance Builder API for collapsible sections in `v2/builder.go`
  - Add AddCollapsibleSection method to Builder interface
  - Add AddCollapsibleTable method for quick table section creation
  - Maintain fluent API pattern and existing Builder functionality

- [x] 11.3 Write integration tests for Builder API enhancements
  - Test collapsible section creation through Builder API
  - Test mixed content documents with both regular and collapsible content
  - Test Builder method chaining with collapsible sections

### 12. Error Handling and Edge Case Implementation

- [x] 12.1 Implement comprehensive error handling across all renderers
  - Add nil details handling with fallback to summary (Requirement 11.1)
  - Add empty summary handling with "[no summary]" placeholder (Requirement 11.2)
  - Add format error fallback to string representation (Requirement 11.3)
  - Add character limit enforcement with truncation indicators (Requirement 11.6)

- [x] 12.2 Implement nested CollapsibleValue handling prevention
  - Treat inner CollapsibleValues as regular content (Requirement 11.5)
  - Prevent recursive processing and potential infinite loops
  - Test nested collapsible scenarios and ensure stable behavior

- [x] 12.3 Write comprehensive error handling tests
  - Test all edge cases: nil details, empty summaries, format errors
  - Test character limit enforcement and truncation behavior
  - Test nested CollapsibleValue scenarios and recovery mechanisms

### 13. Performance Optimization and Memory Management

- [x] 13.1 Implement performance optimizations across rendering pipeline
  - Minimize type assertions to one per value for format detection (Requirement 10.1)
  - Maintain streaming capabilities for large datasets (Requirement 10.2)
  - Avoid unnecessary processing when details not accessed (Requirement 10.3)

- [x] 13.2 Implement memory-efficient processing for large content
  - Optimize detail content processing to avoid redundant transformations (Requirement 10.4)
  - Implement lazy evaluation for format hints when not used (Requirement 10.5)
  - Add character-based truncation with configurable limits (Requirement 10.6, 10.7)

- [x] 13.3 Write performance benchmarks and memory usage tests
  - Create benchmarks for CollapsibleValue processing overhead
  - Test memory usage with large detail content and many collapsible values
  - Ensure minimal overhead when collapsible features not used (Requirement 12.4)

### 14. Integration Testing and Cross-Format Validation

- [ ] 14.1 Create comprehensive integration tests in `v2/collapsible_integration_test.go`
  - Test real-world scenarios: GitHub PR comments, API responses, terminal output
  - Test cross-format consistency for same data across all renderers
  - Test end-to-end pipeline from data input to formatted output

- [ ] 14.2 Implement backward compatibility validation tests
  - Test existing Field.Formatter functions continue working (Requirement 12.1)
  - Test identical output for non-collapsible content (Requirement 12.3)
  - Test zero overhead when collapsible features unused (Requirement 12.4)

- [ ] 14.3 Create example applications demonstrating feature usage
  - Build GitHub PR comment generation example with collapsible error details
  - Build terminal analysis tool with expandable section reports
  - Build CSV export example with detail columns for spreadsheet analysis

### 15. Documentation and Migration Support

- [ ] 15.1 Update existing documentation for Field.Formatter signature change
  - Document migration from `func(any) string` to `func(any) any`
  - Provide examples of both old and new formatter patterns
  - Document backward compatibility guarantees and migration timeline

- [ ] 15.2 Create comprehensive usage documentation
  - Document CollapsibleValue interface and implementation patterns
  - Document CollapsibleSection usage with practical examples
  - Document renderer configuration options and their effects

- [ ] 15.3 Create migration guide for existing v2 users
  - Provide step-by-step migration instructions for Field.Formatter updates
  - Document new collapsible features and integration approaches
  - Include performance considerations and best practices

## Implementation Priority

**Phase 1 (Core Foundation)**: Tasks 1-3 establish the basic infrastructure
**Phase 2 (Renderer Integration)**: Tasks 4-8 implement format-specific support  
**Phase 3 (Section Support)**: Tasks 9-11 add section-level expandability
**Phase 4 (Polish)**: Tasks 12-15 complete error handling, optimization, and documentation

Each task builds incrementally on previous tasks and includes specific requirement references to ensure complete coverage of the feature specification.