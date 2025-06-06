# go-output

A comprehensive Go library for outputting structured data in multiple formats. This library provides a unified interface to convert your data into JSON, YAML, CSV, HTML, tables, markdown, DOT graphs, Mermaid diagrams, and Draw.io files.

## Features

- **Multiple Output Formats**: Support for 9 different output formats
- **Unified Interface**: Single API for all output types
- **Rich Formatting**: Colors, styling, table of contents, section headers
- **File & Cloud Output**: Write to local files or S3 buckets
- **Graph Generation**: Create flowcharts and diagrams from relationship data
- **CLI Integration**: Perfect for adding multiple output options to command-line tools

## Supported Output Formats

- **json** - Standard JSON output
- **yaml** - YAML format for configuration files
- **csv** - Comma-separated values for spreadsheets
- **html** - Full HTML pages with styling and navigation
- **table** - Console-friendly tables with various styles
- **markdown** - GitHub-flavored markdown with table support
- **dot** - GraphViz DOT format for graph visualization
- **mermaid** - Mermaid diagrams (flowcharts, pie charts, Gantt charts)
- **drawio** - Draw.io/Diagrams.net CSV import format

## Quick Start

```bash
go get github.com/ArjenSchwarz/go-output
```

```go
package main

import format "github.com/ArjenSchwarz/go-output"

func main() {
    settings := format.NewOutputSettings()
    settings.SetOutputFormat("table")
    settings.Title = "Employee Report"

    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"Name", "Department", "Active"},
    }

    output.AddContents(map[string]interface{}{
        "Name":       "Alice Johnson",
        "Department": "Engineering",
        "Active":     true,
    })

    output.Write()
}
```

## Documentation

📖 **[Complete Documentation](DOCUMENTATION.md)** - Comprehensive guide covering all features, configuration options, and API reference

🚀 **[Getting Started Guide](GETTING_STARTED.md)** - Quick introduction and setup instructions

💡 **[Examples](examples/)** - Working code examples demonstrating all features

## Dependencies

This library uses several excellent external packages:
- [go-pretty](https://github.com/jedib0t/go-pretty) - Table formatting and styling
- [dot](https://github.com/emicklei/dot) - DOT graph generation
- [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) - S3 integration
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML processing
- [slug](https://github.com/gosimple/slug) - URL-safe string generation
- [color](https://github.com/fatih/color) - Terminal colors

## Usage in Projects

This library is used by various CLI tools for flexible output formatting. If you only need specific functionality, consider using the underlying packages directly. However, if you need multiple output formats with a consistent interface, go-output provides significant value.

## Contributing

Contributions are welcome! Here's how you can help:

### Reporting Issues
- Use the GitHub issue tracker for bug reports and feature requests
- Provide clear examples and steps to reproduce issues
- Include Go version and operating system information

### Development Setup
```bash
# Clone the repository
git clone https://github.com/ArjenSchwarz/go-output.git
cd go-output

# Run tests
go test ./...

# Run examples
cd examples
go run basic_usage.go
```

### Code Contributions
- Fork the repository and create a feature branch
- Follow Go best practices and maintain existing code style
- Add tests for new functionality
- Update documentation for new features
- Ensure all tests pass before submitting

### Code Style
- Use `gofmt` for formatting
- Follow standard Go naming conventions
- Add comments for exported functions and types
- Keep functions focused and testable

### Testing
- Add unit tests for new features
- Maintain or improve test coverage
- Test with multiple Go versions when possible
- Include integration tests for complex features

## License

This project is open source. See the [LICENSE](LICENSE) file for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.

---

**Need help?** Check the [documentation](DOCUMENTATION.md), browse the [examples](examples/), or create an issue for support.
