# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go library for outputting structured data in multiple formats. It provides a unified interface to convert data into JSON, YAML, CSV, HTML, tables, markdown, DOT graphs, Mermaid diagrams, and Draw.io files.

## Development Commands

### Core Commands
```bash
# Format all Go files
go fmt

# Run all tests
go test ./...

# Run tests for specific packages
go test ./errors
go test ./validators
go test ./mermaid
go test ./drawio

# Run examples
cd examples
go run basic_usage.go
```

### Examples
The `examples/` directory contains working examples demonstrating library features. Each example can be run with `go run`.

## Architecture

### Core Components

1. **OutputArray**: Main struct that holds data and settings for output generation
2. **OutputSettings**: Configuration for output format, styling, and destinations
3. **OutputHolder**: Individual data records within an OutputArray

### Key Packages

- **Root package** (`format`): Main output functionality in `output.go` and `outputsettings.go`
- **errors/**: Comprehensive error handling system with validation, recovery, and interactive features
- **validators/**: Data and configuration validation
- **mermaid/**: Mermaid diagram generation (flowcharts, pie charts, Gantt charts)
- **drawio/**: Draw.io CSV export functionality
- **templates/**: HTML template definitions

### Output Formats

The library supports 9 output formats:
- **json**: Standard JSON output
- **yaml**: YAML format 
- **csv**: Comma-separated values
- **html**: Full HTML pages with styling
- **table**: Console tables with various styles
- **markdown**: GitHub-flavored markdown
- **dot**: GraphViz DOT format
- **mermaid**: Mermaid diagrams
- **drawio**: Draw.io/Diagrams.net CSV import

### Key Features

- **Progress Indicators**: Visual progress bars for long-running operations
- **Error Handling**: Comprehensive validation and error reporting system
- **Cloud Output**: S3 bucket integration for cloud storage
- **Flexible Configuration**: Extensive customization options via OutputSettings

### Dependencies

The library uses external packages for specialized functionality:
- `github.com/jedib0t/go-pretty/v6` for table formatting
- `github.com/emicklei/dot` for DOT graph generation
- `github.com/aws/aws-sdk-go-v2` for S3 integration
- `gopkg.in/yaml.v3` for YAML processing

## Testing

Tests are organized by package and follow Go testing conventions. The project includes:
- Unit tests for all major components
- Integration tests for error handling
- Validation tests for OutputSettings
- Example tests in the examples directory

## Error Handling

The library implements a sophisticated error handling system with:
- Validation errors with context and suggestions
- Recovery mechanisms for common failures
- Interactive error handling for user prompts
- Comprehensive error reporting and metrics