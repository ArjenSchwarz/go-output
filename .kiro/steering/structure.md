# Project Structure

## Root Level Files
- `output.go` - Core OutputArray struct and main formatting logic
- `outputsettings.go` - Configuration struct and table styles
- `outputcolors.go` - Color handling utilities
- `progress.go` - Progress indicator interface and factory
- `progress_*.go` - Progress implementations (pretty, noop)
- `*_test.go` - Unit tests for corresponding modules

## Package Organization

### `/mermaid/` - Mermaid Diagram Generation
- `mermaid.go` - Base Mermaid functionality
- `flowchart.go` - Flowchart diagram implementation
- `piechart.go` - Pie chart implementation  
- `ganttchart.go` - Gantt chart implementation
- `settings.go` - Mermaid-specific configuration

### `/drawio/` - Draw.io Integration
- `output.go` - Draw.io CSV export functionality
- `header.go` - Draw.io import header configuration
- `/shapes/` - Shape definitions and conversion utilities

### `/templates/` - HTML Templates
- `html.go` - HTML template definitions for output formatting

### `/examples/` - Usage Examples
- `basic_usage.go` - Comprehensive examples for all formats
- `/progress/` - Progress indicator examples
- `README.md` - Example documentation

### `/docs/` - Documentation
- `/plans/` - Development plans and specifications

## Key Architectural Patterns

### Core Types
- `OutputArray` - Main data container with settings and contents
- `OutputHolder` - Individual record container
- `OutputSettings` - Configuration object with format-specific options

### Extension Points
- Format-specific rendering methods (`toJSON()`, `toHTML()`, etc.)
- Progress interface implementations
- Table style configurations
- Template system for HTML output

### File Naming Conventions
- Core functionality: `output*.go`
- Format-specific: `format_name.go` in subpackages
- Tests: `*_test.go` alongside source files
- Examples: Descriptive names in `/examples/`

### Import Structure
- Main package: `github.com/ArjenSchwarz/go-output`
- Subpackages: `github.com/ArjenSchwarz/go-output/mermaid`
- Import alias: `format "github.com/ArjenSchwarz/go-output"`