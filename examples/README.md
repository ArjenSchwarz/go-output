# go-output Examples

This directory contains practical examples demonstrating how to use the go-output library.

## Running the Examples

1. Navigate to the examples directory:
```bash
cd examples
```

2. Run the basic usage example:
```bash
go run basic_usage.go
```

## Example Files

### basic_usage.go

Demonstrates the core functionality of the library including:

- **JSON Output**: Basic structured data output in JSON format
- **Table Output**: Formatted console tables with styling and colors
- **HTML Output**: Multi-section HTML reports with table of contents
- **Mermaid Diagrams**: Flowchart generation from relationship data
- **CSV File Output**: Writing data to CSV files with array handling

## What Each Example Shows

### 1. Basic JSON Example
Shows how to create a simple OutputArray and output employee data as JSON.

### 2. Table Example
Demonstrates table formatting with:
- Custom titles
- Sorting by field
- Color support
- Custom table styles

### 3. HTML Example
Creates a multi-section HTML report featuring:
- Table of contents generation
- Section headers
- Emoji support for boolean values
- File output

### 4. Mermaid Example
Generates a flowchart showing organizational structure using:
- Manager-employee relationships
- Automatic node creation
- Edge connections

### 5. CSV File Example
Exports employee data to CSV with:
- Array field handling (skills)
- Department-based sorting
- File output

## Sample Output

When you run `basic_usage.go`, you'll see:

1. JSON output printed to console
2. A formatted table printed to console
3. An HTML file created (`employee_report.html`)
4. Mermaid diagram syntax printed to console
5. A CSV file created (`employees.csv`)

## Customizing Examples

You can modify the examples to:
- Change output formats
- Add more data fields
- Experiment with different styling options
- Try different Mermaid chart types
- Add S3 output destinations

Refer to the main documentation for complete configuration options.
