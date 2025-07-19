# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the **v2** directory of the go-output library, representing a complete redesign of the original library. This is a major version bump (v2.0.0) with **no backward compatibility** with v1. The v2 redesign eliminates global state, provides thread-safe operations, and uses modern Go 1.24+ features.

## Development Commands

### Testing
```bash
# Run all tests
go test ./...

# Run specific test
go test -v -run TestSchemaKeyOrderPreservation

# Run tests with coverage
go test -cover ./...
```

### Code Quality
```bash
# Run linter (configured with .golangci.yml)
golangci-lint run

# Format code (automatically converts interface{} to any)
go fmt ./...
```

### Module Management
```bash
# Update dependencies
go mod tidy

# Verify dependencies
go mod verify
```

## Core Architecture

### Document-Builder Pattern
The v2 API centers around a **Document-Builder pattern** that completely eliminates global state:

- **Document**: Immutable container for content and metadata
- **Builder**: Fluent API for constructing documents with thread-safe operations
- **Content Interface**: All content types implement encoding.TextAppender and encoding.BinaryAppender

### Key Order Preservation System
One of the most critical architectural features is **exact key order preservation**:

- **Schema**: Defines table structure with explicit `keyOrder` field that preserves user-specified column ordering
- **Field**: Individual column definitions with optional formatters and hidden flags
- **Table Options**: Functional options pattern (`WithKeys()`, `WithSchema()`, `WithAutoSchema()`) for schema configuration

Key ordering is **never** alphabetized or reordered - it preserves the exact order specified by users, addressing a major limitation of v1.

### Content Type System
Four distinct content types with a unified interface:

- **ContentTypeTable**: Tabular data with schema-driven key ordering
- **ContentTypeText**: Unstructured text with styling options
- **ContentTypeRaw**: Format-specific content (HTML snippets, etc.)
- **ContentTypeSection**: Grouped content with hierarchical structure

### Thread Safety
All operations are thread-safe through careful mutex usage:
- **Document**: Uses sync.RWMutex for safe concurrent reads of contents/metadata
- **Builder**: Uses sync.Mutex for safe concurrent building operations
- **Immutable Design**: Documents become immutable after Build(), preventing modification

## Code Structure Patterns

### Functional Options Pattern
Extensive use of functional options for configuration:
```go
// Table configuration
WithSchema(fields...)     // Explicit schema with preserved field order
WithKeys(keys...)        // Simple key ordering (v1 compatibility)
WithAutoSchema()         // Auto-detect schema from data
```

### Interface-Driven Design
Major components are interfaces to support extensibility:
- **Content**: Core content interface with encoding methods
- **Future interfaces**: Renderer, Transformer, Writer (planned)

### Type Safety with Modern Go
- Uses `any` instead of `interface{}` (enforced by golangci-lint)
- Leverages Go 1.24+ features including new testing.B.Loop for benchmarks
- Proper error handling with wrapped errors and context

## Development Notes

### Key Order Testing
Key order preservation is tested extensively. When adding new table functionality, ensure tests verify that:
1. User-specified key order is preserved exactly
2. Multiple tables can have different key orders
3. Key order remains consistent across multiple operations

### Schema Detection
The `DetectSchemaFromData()` function has limitations due to Go map iteration order. For true order preservation, users should use explicit `WithKeys()` or `WithSchema()` options.

### Migration Context
This v2 is designed to replace v1 completely. The agents/ directory contains:
- **requirements.md**: Complete v2 requirements specification
- **design.md**: Detailed architecture and design decisions  
- **tasks.md**: Implementation task breakdown with progress tracking

The v1 codebase (in parent directory) remains for reference but will not receive updates after v2 release.

### Testing Philosophy
- **Key Order**: Extensive testing of key preservation across scenarios
- **Thread Safety**: Concurrent operation tests with multiple goroutines
- **Immutability**: Tests ensuring documents cannot be modified after Build()
- **Interface Compliance**: All content types must implement the Content interface properly

### Future Implementation Areas
Based on the task breakdown in agents/v2-redesign/tasks.md, the following major components are planned:
- Rendering pipeline with format-specific renderers (JSON, YAML, CSV, HTML, Table, Markdown, DOT, Mermaid, Draw.io)
- Transform system for features like emoji conversion, colors, sorting
- Writer system for multiple output destinations (files, S3, stdout)
- Progress indicators for long-running operations