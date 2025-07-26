# Cross-Format Collapsible Content System Requirements

## Introduction

This feature implements a cross-format collapsible content system for the go-output v2 library that enables users to create table cells and content with expandable/collapsible details. The system provides consistent behavior across all output formats (Markdown, JSON, YAML, HTML, Table, CSV) while maintaining backward compatibility with existing APIs. The primary use case is analysis tools that generate GitHub PR comments and other outputs where users need quick overviews with access to detailed information on demand.

## Requirements

### 1. Core Collapsible Interface

**User Story**: As a developer using the go-output library, I want to create collapsible content values that work consistently across all output formats, so that I can provide both summary and detailed views of data without overwhelming users.

**Acceptance Criteria**:
1. **WHEN** a user creates a CollapsibleValue, **THEN** the system **SHALL** provide a Summary() method that returns a string representation for collapsed view
2. **WHEN** a user creates a CollapsibleValue, **THEN** the system **SHALL** provide a Details() method that returns the full detailed content of any type
3. **WHEN** a user creates a CollapsibleValue, **THEN** the system **SHALL** provide an IsExpanded() method that indicates whether content should be expanded by default
4. **WHEN** a user creates a CollapsibleValue, **THEN** the system **SHALL** provide a FormatHint() method that returns format-specific rendering hints
5. **WHEN** the system processes a CollapsibleValue, **THEN** it **SHALL** preserve both summary and detail information without data loss
6. **WHEN** a user creates a DefaultCollapsibleValue, **THEN** the system **SHALL** store summary text, details content, expansion state, and format hints

### 2. Field Formatter Integration

**User Story**: As a developer using existing Field.Formatter patterns, I want collapsible functionality to integrate seamlessly with my current code, so that I can add expandable content without rewriting existing table definitions.

**Acceptance Criteria**:
1. **WHEN** a Field.Formatter returns a CollapsibleValue, **THEN** the system **SHALL** treat it as collapsible content across all renderers
2. **WHEN** a Field.Formatter returns a non-CollapsibleValue, **THEN** the system **SHALL** render it normally without any changes
3. **WHEN** the system processes field values, **THEN** it **SHALL** maintain backward compatibility with all existing formatter functions
4. **WHEN** a user applies CollapsibleFormatter helper function, **THEN** the system **SHALL** generate appropriate CollapsibleValue instances
5. **WHEN** the system encounters a nil detail function in CollapsibleFormatter, **THEN** it **SHALL** return the original value unchanged

### 3. Markdown Renderer Support

**User Story**: As a user generating GitHub PR comments, I want collapsible table cells to use HTML `<details>` elements, so that reviewers can expand detailed information on demand without cluttering the initial view.

**Acceptance Criteria**:
1. **WHEN** markdown renderer processes a CollapsibleValue, **THEN** it **SHALL** generate `<details><summary>...</summary>...</details>` HTML structure
2. **WHEN** a CollapsibleValue has IsExpanded() returning true, **THEN** the markdown renderer **SHALL** add the `open` attribute to the details element
3. **WHEN** the markdown renderer formats details content, **THEN** it **SHALL** properly escape markdown characters and use `<br/>` for line breaks in table cells
4. **WHEN** details content is a string array, **THEN** the markdown renderer **SHALL** join elements with `<br/>` separators
5. **WHEN** details content is a map, **THEN** the markdown renderer **SHALL** format it as key-value pairs with proper HTML formatting
6. **WHEN** the markdown renderer processes CollapsibleValue in table cells, **THEN** it **SHALL** ensure proper table structure is maintained

### 4. JSON Renderer Support

**User Story**: As a developer consuming API responses, I want collapsible content to be represented as structured JSON with clear type indication, so that I can programmatically access both summary and detailed information.

**Acceptance Criteria**:
1. **WHEN** JSON renderer processes a CollapsibleValue, **THEN** it **SHALL** create a JSON object with "type": "collapsible"
2. **WHEN** JSON renderer processes a CollapsibleValue, **THEN** it **SHALL** include "summary", "details", and "expanded" fields
3. **WHEN** a CollapsibleValue provides format hints for JSON, **THEN** the JSON renderer **SHALL** include those hints as additional object properties
4. **WHEN** JSON renderer processes table records, **THEN** it **SHALL** maintain key order preservation while handling collapsible values
5. **WHEN** JSON renderer streams large datasets, **THEN** it **SHALL** handle collapsible values without loading all details into memory simultaneously

### 5. YAML Renderer Support

**User Story**: As a user exporting configuration or data files, I want collapsible content to be represented in clean YAML format, so that the files remain human-readable while preserving detailed information.

**Acceptance Criteria**:
1. **WHEN** YAML renderer processes a CollapsibleValue, **THEN** it **SHALL** create a YAML mapping with "summary", "details", and "expanded" fields
2. **WHEN** a CollapsibleValue provides YAML format hints, **THEN** the YAML renderer **SHALL** incorporate those hints into the output structure
3. **WHEN** YAML renderer processes nested details content, **THEN** it **SHALL** maintain proper YAML structure and indentation
4. **WHEN** YAML renderer handles string arrays in details, **THEN** it **SHALL** represent them as proper YAML sequences
5. **WHEN** YAML renderer processes map details, **THEN** it **SHALL** preserve key order and proper YAML mapping syntax

### 6. Table Renderer Support

**User Story**: As a user viewing output in terminal, I want collapsible content to show compact summaries with clear indication of hidden details, so that I can see overview information and know when detailed information is available.

**Acceptance Criteria**:
1. **WHEN** table renderer processes a CollapsibleValue with IsExpanded() false and no global expansion override, **THEN** it **SHALL** display summary text with configurable hidden details indicator
2. **WHEN** table renderer processes a CollapsibleValue with IsExpanded() true or global expansion enabled, **THEN** it **SHALL** display both summary and indented details in the same cell
3. **WHEN** table renderer formats details content, **THEN** it **SHALL** indent each line with appropriate spacing for readability
4. **WHEN** details content spans multiple lines, **THEN** the table renderer **SHALL** maintain table structure while showing all content
5. **WHEN** table renderer processes multiple collapsible cells, **THEN** it **SHALL** maintain consistent formatting and alignment
6. **WHEN** table renderer is configured with custom expansion indicator text, **THEN** it **SHALL** use that text instead of default "[details hidden - use --expand for full view]"
7. **WHEN** global expansion is enabled for table renderer, **THEN** it **SHALL** override individual CollapsibleValue IsExpanded() settings

### 7. HTML Renderer Support

**User Story**: As a user generating HTML reports, I want collapsible content to use semantic HTML5 elements with proper styling classes, so that the output is accessible and can be enhanced with custom CSS or JavaScript.

**Acceptance Criteria**:
1. **WHEN** HTML renderer processes a CollapsibleValue, **THEN** it **SHALL** generate `<details>` elements with semantic CSS classes
2. **WHEN** HTML renderer creates collapsible content, **THEN** it **SHALL** include "collapsible-cell", "collapsible-summary", and "collapsible-details" CSS classes
3. **WHEN** a CollapsibleValue has IsExpanded() true, **THEN** the HTML renderer **SHALL** add the `open` attribute
4. **WHEN** HTML renderer formats details content, **THEN** it **SHALL** properly escape HTML characters and use appropriate markup
5. **WHEN** HTML renderer processes structured details, **THEN** it **SHALL** generate semantic HTML elements (lists, tables, etc.) as appropriate

### 8. CSV Renderer Support

**User Story**: As a user exporting data to spreadsheet applications, I want collapsible content to be split into separate summary and detail columns, so that I can analyze data in Excel or similar tools with access to both views.

**Acceptance Criteria**:
1. **WHEN** CSV renderer detects collapsible fields in table schema, **THEN** it **SHALL** automatically create additional "_details" columns
2. **WHEN** CSV renderer processes a CollapsibleValue, **THEN** it **SHALL** place summary in the original column and details in the corresponding detail column
3. **WHEN** CSV renderer handles non-collapsible values, **THEN** it **SHALL** leave detail columns empty for those rows
4. **WHEN** CSV renderer modifies schema for collapsible fields, **THEN** it **SHALL** maintain original column order and append detail columns adjacent to their source columns
5. **WHEN** details content contains complex structures, **THEN** the CSV renderer **SHALL** flatten them to appropriate string representations

### 9. Helper Functions and Common Patterns

**User Story**: As a developer creating analysis tools, I want pre-built formatter functions for common collapsible patterns, so that I can quickly implement expandable error lists, file paths, and other typical use cases.

**Acceptance Criteria**:
1. **WHEN** a user calls ErrorListFormatter(), **THEN** the system **SHALL** return a formatter that creates collapsible content for error arrays
2. **WHEN** ErrorListFormatter processes an array of strings, **THEN** it **SHALL** create summary showing error count and details showing full error list
3. **WHEN** a user calls FilePathFormatter(), **THEN** the system **SHALL** return a formatter that shortens long file paths in summary view
4. **WHEN** FilePathFormatter processes paths longer than 50 characters, **THEN** it **SHALL** show abbreviated path in summary and full path in details
5. **WHEN** a user calls CollapsibleFormatter with custom template and detail function, **THEN** the system **SHALL** apply the template to create summary and use detail function for expansion content
6. **WHEN** helper formatters encounter incompatible data types, **THEN** they **SHALL** gracefully fall back to displaying the original value

### 10. Performance and Memory Management

**User Story**: As a developer working with large datasets, I want collapsible content to have minimal performance impact and efficient memory usage, so that my applications remain responsive even with extensive detailed information.

**Acceptance Criteria**:
1. **WHEN** the system processes collapsible values, **THEN** it **SHALL** add no more than one type assertion per value for format detection
2. **WHEN** rendering large tables with collapsible content, **THEN** the system **SHALL** maintain streaming capabilities where supported
3. **WHEN** details content is not accessed, **THEN** the system **SHALL** avoid unnecessary processing or memory allocation for details
4. **WHEN** multiple formatters are applied, **THEN** the system **SHALL** process them in single pass without redundant transformations
5. **WHEN** format hints are not used by a renderer, **THEN** the system **SHALL** not process or store them unnecessarily
6. **WHEN** details content exceeds the configured character limit (default 500), **THEN** the system **SHALL** truncate with clear indication of truncation
7. **WHEN** applications configure custom character limits for details truncation, **THEN** the system **SHALL** respect those limits instead of the default

### 11. Error Handling and Edge Cases

**User Story**: As a developer integrating collapsible content, I want the system to handle edge cases gracefully and provide clear error messages, so that my application remains stable even with unexpected input.

**Acceptance Criteria**:
1. **WHEN** a CollapsibleValue has nil details, **THEN** the system **SHALL** treat it as non-collapsible content
2. **WHEN** summary text is empty, **THEN** the system **SHALL** use a default placeholder like "[no summary]"
3. **WHEN** details content cannot be formatted for a specific renderer, **THEN** the system **SHALL** fall back to string representation
4. **WHEN** format hints contain invalid data, **THEN** the system **SHALL** pass them through to renderers as-is without validation
5. **WHEN** nested CollapsibleValues are encountered, **THEN** the system **SHALL** treat inner CollapsibleValues as regular content and not process them recursively
6. **WHEN** details content exceeds configured character limits, **THEN** the system **SHALL** truncate with clear indication like "[...truncated]"

### 12. Backward Compatibility

**User Story**: As a developer with existing go-output v2 code, I want to add collapsible functionality without breaking any current implementations, so that I can upgrade incrementally without rewriting existing code.

**Acceptance Criteria**:
1. **WHEN** existing Field.Formatter functions are used, **THEN** they **SHALL** continue to work exactly as before
2. **WHEN** existing table creation code is executed, **THEN** it **SHALL** produce identical output to current behavior
3. **WHEN** existing renderers process non-collapsible content, **THEN** the performance **SHALL** be identical to current implementation
4. **WHEN** new collapsible features are not used, **THEN** the system **SHALL** have zero overhead compared to current v2 implementation
5. **WHEN** existing APIs are called, **THEN** they **SHALL** maintain the same signatures and return types

### 13. Global Expansion Control

**User Story**: As a developer testing or debugging collapsible content, I want to control expansion behavior globally across all renderers, so that I can override individual CollapsibleValue settings when needed.

**Acceptance Criteria**:
1. **WHEN** a renderer is configured with global expansion enabled, **THEN** it **SHALL** treat all CollapsibleValues as expanded regardless of their IsExpanded() setting
2. **WHEN** a renderer is configured with global expansion disabled, **THEN** it **SHALL** respect individual CollapsibleValue IsExpanded() settings
3. **WHEN** no global expansion setting is configured, **THEN** the renderer **SHALL** default to respecting individual CollapsibleValue settings
4. **WHEN** table renderer has global expansion enabled, **THEN** it **SHALL** show details for all collapsible fields without expansion indicators
5. **WHEN** applications need to override expansion behavior, **THEN** they **SHALL** be able to configure this setting per renderer instance

### 14. Configurable Renderer Settings

**User Story**: As an application developer, I want to configure renderer-specific settings for collapsible content, so that the output matches my application's user interface and conventions.

**Acceptance Criteria**:
1. **WHEN** table renderer is configured with custom expansion indicator text, **THEN** it **SHALL** use that text instead of the default indicator
2. **WHEN** applications configure character limits for detail truncation, **THEN** all renderers **SHALL** respect those limits with default of 500 characters
3. **WHEN** truncation occurs, **THEN** the system **SHALL** append a configurable truncation indicator (default "[...truncated]")
4. **WHEN** renderer settings are not explicitly configured, **THEN** the system **SHALL** use sensible defaults
5. **WHEN** multiple renderer instances are created, **THEN** each **SHALL** maintain its own configuration settings independently

## Implementation Clarifications

Based on feedback, the following design decisions have been made:

1. **Global Expansion Control**: ✅ Implemented via renderer-level configuration settings
2. **Memory Limits**: ✅ Configurable character-based limits with 500 character default
3. **Format Hint Validation**: ✅ Pass-through approach without validation
4. **CLI Integration**: ❌ Not included - left to implementer choice
5. **Nested Collapsible Content**: ❌ Single level only - nested CollapsibleValues treated as regular content

Do the requirements look good or do you want additional changes?