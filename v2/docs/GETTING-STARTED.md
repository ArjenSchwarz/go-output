# Getting Started with go-output v2

Welcome to go-output v2! This guide will get you up and running quickly with this redesigned Go library for outputting structured data in multiple formats.

## What is go-output v2?

go-output v2 is a complete redesign of the original library, providing a thread-safe, immutable Document-Builder pattern to convert your structured data into various output formats including JSON, YAML, CSV, HTML, console tables, Markdown, DOT graphs, Mermaid diagrams, and Draw.io files.

## Quick Installation

```bash
go get github.com/ArjenSchwarz/go-output/v2@latest
```

## 30-Second Example

```go
package main

import (
    "fmt"
    output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
    // Create a builder
    builder := output.NewBuilder()
    
    // Add a table with explicit key ordering
    builder.AddTable(
        []map[string]any{
            {"Name": "Alice", "Age": 30, "Active": true},
            {"Name": "Bob", "Age": 25, "Active": false},
        },
        output.WithKeys("Name", "Age", "Active"), // Preserve exact key order
    )
    
    // Build immutable document
    doc := builder.Build()
    
    // Render as table
    result, _ := doc.Render("table")
    fmt.Print(result)
}
```

## Key Features

✅ **Thread-Safe Operations**: All operations are thread-safe with proper mutex usage  
✅ **Immutable Documents**: Documents become immutable after Build() for safety  
✅ **Key Order Preservation**: Exact preservation of user-specified column ordering  
✅ **Multiple Output Formats**: JSON, YAML, CSV, HTML, Tables, Markdown, DOT, Mermaid, Draw.io  
✅ **Document-Builder Pattern**: Clean separation of content building and rendering  
✅ **Modern Go**: Uses Go 1.24+ features with `any` instead of `interface{}`  
✅ **No Global State**: Complete elimination of global state from v1  

## Common Use Cases

- **CLI Tools**: Add multiple output formats to your command-line applications
- **Reports**: Generate HTML or PDF-ready reports from data
- **Data Export**: Convert database results to CSV, JSON, or other formats
- **Documentation**: Create markdown documentation with embedded data
- **Visualization**: Generate Mermaid diagrams from relationship data
- **Dashboards**: Output data for web dashboards and APIs

## What's New in v2

### Document-Builder Pattern
```go
// v2: Clean separation of building and rendering
builder := output.NewBuilder()
builder.AddTable(data, output.WithKeys("col1", "col2"))
doc := builder.Build()
result, _ := doc.Render("json")

// v1: Mixed concerns with global state
output := format.OutputArray{Settings: settings}
output.AddContents(data)
output.Write()
```

### Key Order Preservation
```go
// v2: Keys are preserved in exact order specified
builder.AddTable(data, output.WithKeys("Name", "Age", "City"))
// Output will ALWAYS be Name, Age, City - never alphabetized

// v1: Keys could be reordered unpredictably
output.Keys = []string{"Name", "Age", "City"}
// Output order was not guaranteed
```

### Thread Safety
```go
// v2: Safe concurrent operations
var wg sync.WaitGroup
builder := output.NewBuilder()

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        builder.AddText(fmt.Sprintf("Worker %d", id))
    }(i)
}
wg.Wait()

doc := builder.Build()
// Document is now immutable and safe to share
```

## Next Steps

1. **📖 Read the Documentation**: See [DOCUMENTATION.md](DOCUMENTATION.md) for complete v2 API details
2. **🔧 Try the Examples**: Run the examples in the [examples/](examples/) directory
3. **🚀 Migrate from v1**: Check the migration guide for upgrading existing code
4. **💡 Explore Features**: Discover advanced features like schemas and transformations

## Documentation Structure

- **[DOCUMENTATION.md](DOCUMENTATION.md)** - Complete v2 library documentation
- **[examples/](examples/)** - Working code examples for all v2 features
- **[CLAUDE.md](CLAUDE.md)** - Development notes and architecture details
- **[../agents/v2-redesign/](../agents/v2-redesign/)** - Design documents and requirements

## Need Help?

- Check the examples directory for working code
- Read the comprehensive documentation
- Review the design documents in agents/v2-redesign/
- Create an issue in the repository for specific questions

## Quick Format Reference

| Format | Use Case | Render Command |
|--------|----------|----------------|
| `json` | APIs, data exchange | `doc.Render("json")` |
| `yaml` | Configuration files | `doc.Render("yaml")` |
| `csv` | Spreadsheets | `doc.Render("csv")` |
| `html` | Web reports | `doc.Render("html")` |
| `table` | Console output | `doc.Render("table")` |
| `markdown` | Documentation | `doc.Render("markdown")` |
| `mermaid` | Diagrams | `doc.Render("mermaid")` |
| `dot` | GraphViz graphs | `doc.Render("dot")` |
| `drawio` | Draw.io import | `doc.Render("drawio")` |

Start with the Document-Builder pattern for clean, maintainable code that leverages v2's thread-safe, immutable design!