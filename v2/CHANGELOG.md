# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- JSON renderer support for CollapsibleValue with structured output including type indicators, summary, details, and expanded fields
- YAML renderer support for CollapsibleValue with proper YAML mapping structure and format hints
- Enhanced JSON/YAML table rendering to process field formatters and detect CollapsibleValue interface
- Support for nested data structures in YAML renderer: maps, string arrays, and mixed arrays
- Format-specific hints integration for both JSON and YAML renderers
- Cross-format key order preservation with CollapsibleValue support
- Streaming capabilities for large datasets containing CollapsibleValue objects
- Comprehensive test suite for JSON/YAML collapsible rendering with edge case coverage
- Markdown renderer support for CollapsibleValue with HTML `<details>` elements
- CollapsibleValue detection and rendering in markdown table cells using `formatCellValue` method
- Markdown-specific detail formatting for different data types (strings, arrays, maps)
- Global expansion override support in markdown renderer via `collapsibleConfig.ForceExpansion`
- Proper HTML escaping for collapsible content in markdown table cells
- Comprehensive test coverage for markdown collapsible rendering including edge cases and global expansion
- HTML renderer support for CollapsibleValue with semantic HTML5 `<details>` elements and configurable CSS classes
- HTML-specific detail formatting with proper escaping: string arrays as `<ul>` lists, maps as `<dl>` definition lists
- Support for global expansion control and custom CSS classes in HTML renderer via RendererConfig
- NewHTMLRendererWithCollapsible constructor for configurable HTML renderer with collapsible support
- Comprehensive HTML collapsible test suite with 30+ test cases covering rendering, escaping, and configuration
- Core CollapsibleValue interface for cross-format expandable content
- DefaultCollapsibleValue implementation with functional options pattern
- Support for configurable character limits and truncation indicators
- Format-specific rendering hints system
- Comprehensive error handling with graceful fallbacks for nil details and empty summaries
- Unit tests covering all interface methods, edge cases, and character truncation functionality
- CollapsibleFormatter helper function for creating field formatters with summary templates and detail functions
- Pre-built formatter functions: ErrorListFormatter for error arrays, FilePathFormatter for long paths, JSONFormatter for complex data structures
- Enhanced Field.Formatter signature to return `any` instead of `string` to support CollapsibleValue objects
- Comprehensive unit tests for all collapsible formatter functions with edge case coverage

### Changed
- Field.Formatter signature changed from `func(any) string` to `func(any) any` with full backward compatibility
- Updated all existing renderers (HTML, Markdown, Content) to handle new formatter signature gracefully
- Enhanced markdownRenderer and tableRenderer structs with collapsibleConfig field for expandable content support
- Enhanced htmlRenderer struct with collapsibleConfig field and updated table cell rendering for CollapsibleValue detection

### Added
- Base renderer infrastructure for collapsible content processing
- RendererConfig struct with global expansion control, character limits, and format-specific settings
- DefaultRendererConfig with sensible defaults (500 char limit, configurable indicators)
- processFieldValue method in baseRenderer for CollapsibleValue detection with backward compatibility
- NewMarkdownRendererWithCollapsible constructor for markdown renderer with collapsible configuration
- NewTableRendererWithCollapsible constructor for table renderer with collapsible configuration