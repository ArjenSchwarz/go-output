# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Renderer interface with Format(), Render(), RenderTo(), and SupportsStreaming() methods
- Format struct for output format configuration with Name, Renderer, and Options fields
- Built-in format constants for all v1 formats: JSON, YAML, CSV, HTML, Table, Markdown, DOT, Mermaid, DrawIO
- BaseRenderer struct with common functionality for thread-safe, context-aware rendering
- Context cancellation support for all rendering operations
- Memory-efficient rendering patterns using bytes.Buffer and streaming approaches
- Comprehensive test suite covering interface compliance, context cancellation, error handling, and thread safety
- Streaming support categorization (JSON, YAML, CSV, HTML, Table, Markdown support streaming; DOT, Mermaid, DrawIO do not)
- Error handling with proper error wrapping and validation for nil inputs
- Thread-safe concurrent rendering operations using sync.RWMutex

### Changed
- Updated tasks.md to mark rendering pipeline foundation tasks (5.1, 5.2) as completed