# go-output v2 Library Documentation

A comprehensive Go library for outputting structured data in multiple formats. Version 2 provides a complete redesign with a Document-Builder pattern, thread-safe operations, and guaranteed key order preservation.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Architecture](#core-architecture)
- [Document-Builder Pattern](#document-builder-pattern)
- [Content Types](#content-types)
- [Supported Output Formats](#supported-output-formats)
- [Configuration Options](#configuration-options)
- [Advanced Features](#advanced-features)
- [Migration from v1](#migration-from-v1)
- [API Reference](#api-reference)

## Installation

```bash
go get github.com/ArjenSchwarz/go-output/v2@latest
```

Requirements:
- Go 1.24 or later
- No global state dependencies
- Thread-safe by design

## Quick Start

Here's a simple example demonstrating the v2 Document-Builder pattern:

```go
package main

import (
    "fmt"
    output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
    // Create a builder
    builder := output.NewBuilder()
    
    // Add a table with preserved key ordering
    builder.AddTable(
        []map[string]any{
            {"Name": "Alice", "Age": 30, "City": "New York"},
            {"Name": "Bob", "Age": 25, "City": "London"},
        },
        output.WithKeys("Name", "Age", "City"), // Exact order preserved
    )
    
    // Build immutable document
    doc := builder.Build()
    
    // Render to different formats
    json, _ := doc.Render("json")
    fmt.Println(json)
    
    table, _ := doc.Render("table")
    fmt.Println(table)
}
```

## Core Architecture

### Document-Builder Pattern

v2 introduces a clean separation of concerns:

1. **Builder Phase**: Accumulate content using the Builder
2. **Document Phase**: Create an immutable Document from the Builder
3. **Render Phase**: Generate output in various formats from the Document

```go
// Phase 1: Building
builder := output.NewBuilder()
builder.AddTable(data)
builder.AddText("Summary text")

// Phase 2: Document creation (immutable)
doc := builder.Build()

// Phase 3: Rendering (repeatable, thread-safe)
json, _ := doc.Render("json")
yaml, _ := doc.Render("yaml")
```

### Thread Safety

All v2 operations are thread-safe:

```go
builder := output.NewBuilder()

// Safe concurrent building
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        builder.AddText(fmt.Sprintf("Worker %d", id))
    }(i)
}
wg.Wait()

// Document is immutable and safe to share
doc := builder.Build()

// Safe concurrent rendering
go func() { json, _ := doc.Render("json") }()
go func() { yaml, _ := doc.Render("yaml") }()
```

### Key Order Preservation

v2 guarantees exact preservation of user-specified key ordering:

```go
// Keys will ALWAYS appear in this exact order
builder.AddTable(data, output.WithKeys("ID", "Name", "Email", "Status"))

// Or use a schema for more control
schema := output.NewSchema(
    output.Field{Key: "ID", Hidden: false},
    output.Field{Key: "Name", Formatter: output.UppercaseFormatter},
    output.Field{Key: "Email"},
    output.Field{Key: "Status", Hidden: true}, // Hidden from output
)
builder.AddTable(data, output.WithSchema(schema))
```

## Document-Builder Pattern

### Builder

The Builder accumulates content before creating an immutable Document:

```go
type Builder struct {
    contents []Content     // Thread-safe content list
    metadata Metadata      // Document metadata
    mu       sync.Mutex    // Ensures thread safety
}

// Create a new builder
builder := output.NewBuilder()

// Set metadata
builder.SetTitle("Monthly Report")
builder.SetMetadata("author", "Alice")
builder.SetMetadata("version", "2.0")

// Add content
builder.AddTable(data)
builder.AddText("Summary")
builder.AddSection("Details", sectionContent)

// Build immutable document
doc := builder.Build()
```

### Document

Documents are immutable containers for content and metadata:

```go
type Document struct {
    contents []Content         // Immutable content list
    metadata Metadata          // Immutable metadata
    mu       sync.RWMutex      // Read-write lock for safe access
}

// Documents are immutable after creation
doc := builder.Build()

// Safe to share across goroutines
ch := make(chan *Document)
go func() { ch <- doc }()

// Multiple renders are safe
json, _ := doc.Render("json")
yaml, _ := doc.Render("yaml")
```

## Content Types

### Table Content

Tables with schema-driven key ordering:

```go
// Simple key ordering
builder.AddTable(
    []map[string]any{
        {"Name": "Alice", "Score": 95},
        {"Name": "Bob", "Score": 87},
    },
    output.WithKeys("Name", "Score"),
)

// Schema with formatters
schema := output.NewSchema(
    output.Field{Key: "Name", Formatter: output.UppercaseFormatter},
    output.Field{Key: "Score", Formatter: output.PercentFormatter},
)
builder.AddTable(data, output.WithSchema(schema))

// Auto-detect schema (Note: key order may vary)
builder.AddTable(data, output.WithAutoSchema())
```

### Text Content

Unstructured text with optional styling:

```go
// Simple text
builder.AddText("This is a plain text message")

// Styled text (when supported by renderer)
builder.AddText(
    "Important message",
    output.WithTextStyle(output.TextStyleBold),
)

// Multi-line text
builder.AddText(`Line 1
Line 2
Line 3`)
```

### Section Content

Hierarchical content grouping:

```go
// Create a section with nested content
sectionBuilder := output.NewBuilder()
sectionBuilder.AddText("Section introduction")
sectionBuilder.AddTable(sectionData)

builder.AddSection(
    "Section Title",
    sectionBuilder.Build(),
    output.WithSectionLevel(2), // H2 level
)
```

### Raw Content

Format-specific content:

```go
// Add raw HTML (only rendered in HTML output)
builder.AddRaw(
    "<div class='custom'>Custom HTML</div>",
    output.ContentTypeHTML,
)

// Add raw Markdown (only rendered in Markdown output)
builder.AddRaw(
    "## Custom Markdown\n\n- Item 1\n- Item 2",
    output.ContentTypeMarkdown,
)
```

## Supported Output Formats

### JSON Format

Standard JSON with preserved key ordering:

```go
json, err := doc.Render("json")
// Output: [{"Name":"Alice","Age":30},{"Name":"Bob","Age":25}]
```

### YAML Format

YAML with maintained structure:

```go
yaml, err := doc.Render("yaml")
// Output:
// - Name: Alice
//   Age: 30
// - Name: Bob
//   Age: 25
```

### CSV Format

Comma-separated values with headers:

```go
csv, err := doc.Render("csv")
// Output:
// Name,Age
// Alice,30
// Bob,25
```

### Table Format

Console-friendly tables with styling:

```go
table, err := doc.Render("table")
// Output:
// ┌───────┬─────┐
// │ Name  │ Age │
// ├───────┼─────┤
// │ Alice │ 30  │
// │ Bob   │ 25  │
// └───────┴─────┘
```

### HTML Format

Complete HTML documents:

```go
html, err := doc.Render("html")
// Generates full HTML with <table> elements and styling
```

### Markdown Format

GitHub-flavored Markdown:

```go
markdown, err := doc.Render("markdown")
// Output:
// | Name | Age |
// |------|-----|
// | Alice | 30 |
// | Bob | 25 |
```

### Mermaid Format

Mermaid diagram syntax:

```go
// For flowcharts
builder.AddTable(
    []map[string]any{
        {"From": "Start", "To": "Process"},
        {"From": "Process", "To": "End"},
    },
    output.WithKeys("From", "To"),
)
mermaid, _ := doc.Render("mermaid")
```

### DOT Format

GraphViz DOT syntax:

```go
// For graph visualization
builder.AddTable(
    []map[string]any{
        {"Source": "A", "Target": "B"},
        {"Source": "B", "Target": "C"},
    },
    output.WithKeys("Source", "Target"),
)
dot, _ := doc.Render("dot")
```

## Configuration Options

### Table Options

```go
// Simple key ordering
output.WithKeys("col1", "col2", "col3")

// Full schema control
output.WithSchema(schema)

// Auto-detect from data
output.WithAutoSchema()

// Custom table title
output.WithTableTitle("User Data")
```

### Field Options

```go
type Field struct {
    Key       string           // Column key
    Header    string           // Display header (optional)
    Formatter FieldFormatter   // Value formatter (optional)
    Hidden    bool            // Hide from output
}

// Built-in formatters
output.UppercaseFormatter
output.LowercaseFormatter
output.PercentFormatter
output.CurrencyFormatter
output.DateFormatter
output.BooleanFormatter

// Custom formatter
type CustomFormatter struct{}
func (f CustomFormatter) Format(value any) string {
    return fmt.Sprintf("Custom: %v", value)
}
```

### Rendering Options

```go
// With specific renderer configuration
opts := output.RenderOptions{
    Format: "table",
    TableStyle: "ascii",
    ColorOutput: true,
    MaxColumnWidth: 50,
}
result, err := doc.RenderWithOptions(opts)
```

## Advanced Features

### Multi-Format Output

```go
// Build once, render many
doc := builder.Build()

formats := []string{"json", "yaml", "csv", "table"}
results := make(map[string]string)

for _, format := range formats {
    result, _ := doc.Render(format)
    results[format] = result
}
```

### Custom Transformations

```go
// Apply transformations before rendering
doc := builder.Build()

// Transform for emoji support
transformed := doc.Transform(output.EmojiTransformer{})

// Render transformed document
result, _ := transformed.Render("table")
```

### Concurrent Processing

```go
// Process large datasets concurrently
func processConcurrently(data [][]map[string]any) *output.Document {
    builder := output.NewBuilder()
    
    var wg sync.WaitGroup
    for _, chunk := range data {
        wg.Add(1)
        go func(c []map[string]any) {
            defer wg.Done()
            builder.AddTable(c, output.WithAutoSchema())
        }(chunk)
    }
    
    wg.Wait()
    return builder.Build()
}
```

### Schema Validation

```go
// Define schema with validation
schema := output.NewSchema(
    output.Field{
        Key: "Email",
        Validator: func(v any) error {
            str, ok := v.(string)
            if !ok || !strings.Contains(str, "@") {
                return fmt.Errorf("invalid email")
            }
            return nil
        },
    },
)

// Validation happens during table addition
err := builder.AddTable(data, output.WithSchema(schema))
if err != nil {
    log.Printf("Validation failed: %v", err)
}
```

## Migration from v1

### Key Differences

| Feature | v1 | v2 |
|---------|----|----|
| Pattern | Direct output | Document-Builder |
| State | Global settings | No global state |
| Thread Safety | Limited | Full thread safety |
| Key Order | Unpredictable | Guaranteed preservation |
| Documents | N/A | Immutable |
| API Style | Procedural | Functional options |

### Migration Examples

```go
// v1 Code
settings := format.NewOutputSettings()
settings.SetOutputFormat("json")
output := format.OutputArray{
    Settings: settings,
    Keys: []string{"Name", "Age"},
}
output.AddContents(map[string]interface{}{
    "Name": "Alice",
    "Age": 30,
})
output.Write()

// v2 Equivalent
builder := output.NewBuilder()
builder.AddTable(
    []map[string]any{
        {"Name": "Alice", "Age": 30},
    },
    output.WithKeys("Name", "Age"),
)
doc := builder.Build()
json, _ := doc.Render("json")
fmt.Print(json)
```

## API Reference

### Builder Methods

```go
// Creation
func NewBuilder() *Builder

// Metadata
func (b *Builder) SetTitle(title string)
func (b *Builder) SetMetadata(key string, value any)

// Content addition
func (b *Builder) AddTable(data []map[string]any, opts ...TableOption) error
func (b *Builder) AddText(text string, opts ...TextOption)
func (b *Builder) AddSection(title string, content *Document, opts ...SectionOption)
func (b *Builder) AddRaw(content string, format ContentFormat)

// Document creation
func (b *Builder) Build() *Document
```

### Document Methods

```go
// Rendering
func (d *Document) Render(format string) (string, error)
func (d *Document) RenderWithOptions(opts RenderOptions) (string, error)

// Metadata access
func (d *Document) GetTitle() string
func (d *Document) GetMetadata(key string) (any, bool)

// Content access (read-only)
func (d *Document) GetContents() []Content
func (d *Document) ContentCount() int

// Transformation
func (d *Document) Transform(t Transformer) *Document
```

### Table Options

```go
func WithKeys(keys ...string) TableOption
func WithSchema(schema *Schema) TableOption
func WithAutoSchema() TableOption
func WithTableTitle(title string) TableOption
```

### Schema Methods

```go
func NewSchema(fields ...Field) *Schema
func (s *Schema) AddField(field Field)
func (s *Schema) GetFieldOrder() []string
func (s *Schema) ValidateData(data map[string]any) error
```

## Best Practices

1. **Use WithKeys() for predictable ordering**: Always specify key order explicitly
2. **Build once, render many**: Create the Document once and render multiple formats
3. **Leverage thread safety**: Use concurrent operations for large datasets
4. **Validate early**: Use schema validation to catch data issues early
5. **Prefer immutability**: Don't try to modify Documents after building
6. **Use appropriate content types**: Choose the right content type for your data
7. **Handle errors**: Always check errors from Render operations
8. **Test key ordering**: Verify that key order is preserved in your tests

## Dependencies

The v2 library has minimal, well-maintained dependencies:
- Standard library for core functionality
- `github.com/jedib0t/go-pretty/v6` - Table formatting
- `gopkg.in/yaml.v3` - YAML output
- Additional format-specific dependencies as needed

## License

This library is provided as-is. Check the repository for specific license terms.

---

This documentation covers the complete functionality of go-output v2. For additional examples, migration guides, or specific use cases, refer to the examples directory or the design documents in agents/v2-redesign/.