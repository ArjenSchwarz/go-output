# go-output v2

A comprehensive Go library for outputting structured data in multiple formats with thread-safe operations and preserved key ordering. This library provides a unified interface to convert your data into JSON, YAML, CSV, HTML, tables, markdown, DOT graphs, Mermaid diagrams, and Draw.io files.

**Version 2.0** represents a complete redesign with no backward compatibility, eliminating global state and providing modern Go 1.24+ features.

## Features

- **Thread-Safe Operations**: All components designed for concurrent use
- **Key Order Preservation**: Maintains exact user-specified column ordering
- **Multiple Output Formats**: Support for 9 different output formats with simultaneous rendering
- **Document-Builder Pattern**: Immutable documents with fluent API construction
- **Rich Content Types**: Tables, text, raw content, and hierarchical sections
- **Transform Pipeline**: Emoji conversion, colors, sorting, and custom transformations
- **Multiple Writers**: Output to stdout, files, S3, or multiple destinations simultaneously
- **Progress Indicators**: Visual progress bars for long-running operations
- **Chart Generation**: Gantt charts, pie charts, and flow diagrams

## Supported Output Formats

- **json** - Standard JSON output with preserved key ordering
- **yaml** - YAML format for configuration files
- **csv** - Comma-separated values for spreadsheets
- **html** - Full HTML pages with styling and navigation
- **table** - Console-friendly tables with various styles and colors
- **markdown** - GitHub-flavored markdown with table support and TOC
- **dot** - GraphViz DOT format for graph visualization
- **mermaid** - Mermaid diagrams (flowcharts, pie charts, Gantt charts)
- **drawio** - Draw.io/Diagrams.net CSV import format

## Quick Start

```bash
go get github.com/ArjenSchwarz/go-output/v2
```

```go
package main

import (
    "context"
    "log"
    
    output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
    // Create document using builder pattern
    doc := output.New().
        Table("Employees", []map[string]any{
            {"Name": "Alice Johnson", "Department": "Engineering", "Active": true},
            {"Name": "Bob Smith", "Department": "Marketing", "Active": false},
        }, output.WithKeys("Name", "Department", "Active")).
        Text("Report generated successfully").
        Build()

    // Configure output with multiple formats and destinations
    out := output.NewOutput(
        output.WithFormats(output.Table, output.JSON),
        output.WithWriter(output.NewStdoutWriter()),
    )

    // Render the document
    if err := out.Render(context.Background(), doc); err != nil {
        log.Fatal(err)
    }
}
```

## Key Order Preservation

One of v2's core features is **exact key order preservation**:

```go
// Keys will appear in exact order: Name, Email, Status, Department
doc := output.New().
    Table("Users", userData, output.WithKeys("Name", "Email", "Status", "Department")).
    Build()
```

## Multiple Formats & Destinations

Output to multiple formats and destinations simultaneously:

```go
fileWriter, _ := output.NewFileWriter("./reports", "report.{format}")

out := output.NewOutput(
    output.WithFormats(output.JSON, output.CSV, output.HTML),
    output.WithWriter(output.NewStdoutWriter()),
    output.WithWriter(fileWriter), // Creates report.json, report.csv, report.html
)
```

## Mixed Content Documents

Create rich documents with multiple content types:

```go
doc := output.New().
    Header("System Report").
    Section("User Statistics", func(b *output.Builder) {
        b.Table("Active Users", activeUsers, output.WithKeys("Name", "LastLogin"))
        b.Table("User Roles", roles, output.WithKeys("Role", "Count"))
    }).
    Text("All systems operational").
    Chart("Resource Usage", "pie", resourceData).
    Build()
```

## Documentation

📖 **[Complete API Documentation](v2/API.md)** - Comprehensive interface reference and examples

🚀 **[Migration Guide](v2/MIGRATION.md)** - Complete migration guide from v1 to v2

💡 **[V1→V2 Migration Examples](v2/V1_V2_MIGRATION_EXAMPLES.md)** - Before/after code examples for v1 to v2 migration

🔄 **[V2 Collapsible Features Migration](v2/V2_UPDATES_MIGRATION_EXAMPLES.md)** - Step-by-step guide for adding collapsible content to existing v2 code  

📋 **[Quick Reference](v2/MIGRATION_QUICK_REFERENCE.md)** - Common patterns lookup table

🔧 **[Working Examples](v2/examples/)** - Runnable examples for all major features

## Migration from v1

**v2 is a complete rewrite with no backward compatibility.** If you're using v1, you'll need to migrate your code.

### Migration Resources

- **[Migration Guide](v2/MIGRATION.md)** - Complete step-by-step migration instructions
- **[V1→V2 Migration Examples](v2/V1_V2_MIGRATION_EXAMPLES.md)** - Real before/after code examples
- **[V2 Collapsible Features Migration](v2/V2_UPDATES_MIGRATION_EXAMPLES.md)** - Step-by-step guide for existing v2 users
- **[Breaking Changes](v2/BREAKING_CHANGES.md)** - Detailed list of all breaking changes


### v1 Documentation

For v1 users who aren't ready to migrate, see **[README-v1.md](README-v1.md)** for the original v1 documentation.

## Dependencies

This library uses several excellent external packages:
- [go-pretty](https://github.com/jedib0t/go-pretty) - Table formatting and styling
- [dot](https://github.com/emicklei/dot) - DOT graph generation  
- [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) - S3 integration
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML processing
- [slug](https://github.com/gosimple/slug) - URL-safe string generation
- [color](https://github.com/fatih/color) - Terminal colors

## Requirements

- **Go 1.24+** - Uses modern Go features including `any` type and latest testing patterns
- **Thread-safe usage** - All components designed for concurrent operations

## Usage in Projects

This library is ideal for CLI tools and applications requiring flexible output formatting with maintained data structure integrity. The v2 architecture supports complex scenarios including:

- Multi-format simultaneous output
- Large dataset processing with streaming
- Complex document hierarchies with sections
- Custom transformation pipelines
- Progress tracking for long operations

## Contributing

Contributions are welcome! Here's how you can help:

### Development Setup
```bash
# Clone the repository
git clone https://github.com/ArjenSchwarz/go-output.git
cd go-output

# Work in v2 directory
cd v2

# Run tests
go test ./...

# Run linter
golangci-lint run

# Run examples
cd examples
go run basic_usage/main.go
```

### Code Contributions
- Fork the repository and create a feature branch
- Follow Go best practices and maintain existing code style
- Add tests for new functionality with thread-safety testing
- Update documentation for new features
- Ensure all tests pass and linting succeeds

### Testing
- Add unit tests with concurrent operation coverage
- Test key order preservation for table functionality
- Include integration tests for complex scenarios
- Test with Go 1.24+ features

## License

This project is open source. See the [LICENSE](LICENSE) file for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.

---

**Need help?** Check the [API documentation](v2/API.md), browse the [examples](v2/examples/), or create an issue for support.

**Still using v1?** See [README-v1.md](README-v1.md) for v1 documentation and migration guidance.