# Getting Started with go-output

Welcome to go-output! This guide will get you up and running quickly with this versatile Go library for outputting structured data in multiple formats.

## What is go-output?

go-output is a Go library that provides a unified interface to convert your structured data into various output formats including JSON, YAML, CSV, HTML, console tables, Markdown, DOT graphs, Mermaid diagrams, and Draw.io files.

## Quick Installation

```bash
go get github.com/ArjenSchwarz/go-output
```

## 30-Second Example

```go
package main

import format "github.com/ArjenSchwarz/go-output"

func main() {
    settings := format.NewOutputSettings()
    settings.SetOutputFormat("table")
    settings.Title = "My First Report"

    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"Name", "Age", "Active"},
    }

    output.AddContents(map[string]interface{}{
        "Name":   "Alice",
        "Age":    30,
        "Active": true,
    })

    output.Write()
}
```

## Key Features

✅ **Multiple Output Formats**: JSON, YAML, CSV, HTML, Tables, Markdown, DOT, Mermaid, Draw.io
✅ **Easy Configuration**: Simple settings object controls all behavior
✅ **File & Cloud Output**: Write to files or S3 buckets
✅ **Rich Formatting**: Colors, emoji, styling, table of contents
✅ **Graph Generation**: Create flowcharts and diagrams from your data
✅ **Multi-Section Reports**: Combine multiple data sets with headers
✅ **Type Conversion**: Automatic handling of different data types

## Common Use Cases

- **CLI Tools**: Add multiple output formats to your command-line applications
- **Reports**: Generate HTML or PDF-ready reports from data
- **Data Export**: Convert database results to CSV, JSON, or other formats
- **Documentation**: Create markdown documentation with embedded data
- **Visualization**: Generate Mermaid diagrams from relationship data
- **Dashboards**: Output data for web dashboards and APIs

## Next Steps

1. **📖 Read the Documentation**: See [DOCUMENTATION.md](DOCUMENTATION.md) for complete details
2. **🔧 Try the Examples**: Run the examples in the [examples/](examples/) directory
3. **🚀 Integrate**: Add go-output to your existing projects

## Documentation Structure

- **[DOCUMENTATION.md](DOCUMENTATION.md)** - Complete library documentation
- **[examples/](examples/)** - Working code examples for all features
- **[examples/README.md](examples/README.md)** - Guide to running examples

## Need Help?

- Check the examples directory for working code
- Read the comprehensive documentation
- Look at the test files for usage patterns
- Create an issue in the repository for specific questions

## Quick Format Reference

| Format | Use Case | Command |
|--------|----------|---------|
| `json` | APIs, data exchange | `settings.SetOutputFormat("json")` |
| `yaml` | Configuration files | `settings.SetOutputFormat("yaml")` |
| `csv` | Spreadsheets | `settings.SetOutputFormat("csv")` |
| `html` | Web reports | `settings.SetOutputFormat("html")` |
| `table` | Console output | `settings.SetOutputFormat("table")` |
| `markdown` | Documentation | `settings.SetOutputFormat("markdown")` |
| `mermaid` | Diagrams | `settings.SetOutputFormat("mermaid")` |
| `dot` | GraphViz graphs | `settings.SetOutputFormat("dot")` |
| `drawio` | Draw.io import | `settings.SetOutputFormat("drawio")` |

Start with the format you need most, then explore others as your requirements grow!
