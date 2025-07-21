# Go-Output v2 Examples

This directory contains working examples demonstrating all major features and use cases of the Go-Output v2 library.

## Running Examples

Each example can be run independently:

```bash
# Run a specific example
go run basic_usage/main.go

# Run with different parameters
go run table_formats/main.go -format table
go run table_formats/main.go -format json
```

## Example Categories

### 1. Basic Usage
- `basic_usage/` - Simple table creation and output
- `multiple_formats/` - Output to multiple formats simultaneously
- `key_ordering/` - Demonstrates key order preservation

### 2. Content Types
- `table_content/` - Advanced table creation and options
- `text_content/` - Text content with styling
- `mixed_content/` - Documents with multiple content types
- `sections/` - Hierarchical content organization

### 3. Output Formats
- `table_formats/` - All table style variations
- `markdown_features/` - Markdown with ToC and front matter
- `json_yaml/` - JSON and YAML output
- `csv_html/` - CSV and HTML generation

### 4. Advanced Features
- `transformers/` - Data transformation pipeline
- `progress_tracking/` - Progress indicators
- `file_output/` - File writing patterns
- `concurrent_rendering/` - Thread-safe operations

### 5. Graph and Charts
- `graphs/` - DOT and Mermaid graph generation
- `charts/` - Gantt and pie charts
- `drawio/` - Draw.io diagram CSV export

### 6. Error Handling
- `error_handling/` - Comprehensive error management
- `validation/` - Input validation patterns

### 7. Performance
- `benchmarks/` - Performance testing examples
- `large_datasets/` - Handling large data efficiently

## Common Patterns

Most examples follow this pattern:

```go
package main

import (
    "context"
    "log"
    
    output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
    // 1. Create document using builder
    doc := output.New().
        // Add content here
        Build()
    
    // 2. Configure output
    out := output.NewOutput(
        // Configure formats, writers, etc.
    )
    
    // 3. Render with error handling
    if err := out.Render(context.Background(), doc); err != nil {
        log.Fatal(err)
    }
}
```

## Key Features Demonstrated

- **Key Order Preservation**: How exact column ordering is maintained
- **Thread Safety**: Concurrent document building and rendering
- **Multiple Formats**: Simultaneous output in different formats
- **Progress Tracking**: Visual feedback for long operations
- **Error Handling**: Comprehensive error management
- **Performance**: Efficient memory usage and streaming
- **Extensibility**: Custom renderers and transformers