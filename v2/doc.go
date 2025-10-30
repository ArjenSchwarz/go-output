/*
Package output provides a flexible document construction and rendering system for Go.

# Overview

The go-output v2 library is a complete redesign providing a thread-safe, immutable
Document-Builder pattern to convert structured data into various output formats including
JSON, YAML, CSV, HTML, console tables, Markdown, DOT graphs, Mermaid diagrams, and Draw.io files.

Key features:
  - Thread-safe operations with proper mutex usage
  - Immutable documents after Build() for safety
  - Exact preservation of user-specified column ordering
  - Multiple output formats (JSON, YAML, CSV, HTML, Tables, Markdown, DOT, Mermaid, Draw.io)
  - Document-Builder pattern with clean separation of concerns
  - Modern Go 1.24+ features with 'any' instead of 'interface{}'
  - No global state - complete elimination from v1
  - Per-content transformations for filtering, sorting, limiting, and more

# Basic Usage

Create a document using the builder pattern:

	builder := output.New()
	builder.Table("users", userData, output.WithKeys("name", "email", "age"))
	builder.Text("Summary statistics")
	doc := builder.Build()

Render to a specific format:

	out := output.NewOutput(
	    output.WithFormat(output.JSON()),
	    output.WithWriter(output.NewStdoutWriter()),
	)
	err := out.Render(context.Background(), doc)

# Content Types

The library supports multiple content types:

  - TableContent: Tabular data with schema-driven key ordering
  - TextContent: Unstructured text with styling options
  - RawContent: Format-specific content (HTML snippets, etc.)
  - SectionContent: Grouped content with hierarchical structure
  - GraphContent: Network/relationship diagrams
  - ChartContent: Gantt charts, pie charts, and other visualizations
  - CollapsibleSection: Expandable/collapsible sections for HTML/Markdown

# Output Formats

Supported output formats:

  - JSON: Structured data for APIs
  - YAML: Configuration files
  - CSV: Spreadsheet-compatible data
  - HTML: Web reports with templates
  - Table: Console/terminal tables
  - Markdown: Documentation with optional ToC
  - DOT: GraphViz diagrams
  - Mermaid: Mermaid.js diagrams
  - Draw.io: Import into Draw.io

# Per-Content Transformations

The v2 library introduces per-content transformations, allowing you to attach operations
directly to content items at creation time:

	builder.Table("users", userData,
	    output.WithKeys("name", "email", "age"),
	    output.WithTransformations(
	        output.NewFilterOp(func(r output.Record) bool {
	            return r["age"].(int) >= 18
	        }),
	        output.NewSortOp(output.SortKey{Column: "name", Direction: output.Ascending}),
	        output.NewLimitOp(10),
	    ),
	)

Transformations execute during rendering, allowing different tables in the same document
to have different transformation logic.

# Available Operations

The library provides several built-in operations:

- FilterOp: Filter records based on a predicate function
- SortOp: Sort records by one or more columns
- LimitOp: Limit the number of records
- GroupByOp: Group records by columns with aggregation
- AddColumnOp: Add calculated columns

Create operations using constructor functions:

	filter := output.NewFilterOp(predicate)
	sort := output.NewSortOp(keys...)
	limit := output.NewLimitOp(count)
	groupBy := output.NewGroupByOp(columns, aggregates)
	addCol := output.NewAddColumnOp(name, fn, position)

# Thread Safety Requirements

Operations attached to content MUST be thread-safe and stateless:

1. Operations MUST NOT contain mutable state modified during Apply()
2. Operations MUST be safe for concurrent execution from multiple goroutines
3. Operations MUST NOT capture mutable variables in closures

Example of UNSAFE operation (captures loop variable by reference):

	// WRONG - captures loop variable by reference
	for _, threshold := range thresholds {
	    builder.Table(name, data,
	        output.WithTransformations(
	            output.NewFilterOp(func(r Record) bool {
	                return r["value"].(int) > threshold  // BUG!
	            }),
	        ),
	    )
	}

Example of SAFE operation (explicit value capture):

	// CORRECT - captures value explicitly
	for _, threshold := range thresholds {
	    t := threshold  // Capture value
	    builder.Table(name, data,
	        output.WithTransformations(
	            output.NewFilterOp(func(r Record) bool {
	                return r["value"].(int) > t  // Safe
	            }),
	        ),
	    )
	}

# Transformation Execution

Transformations execute lazily during document rendering:

1. Builder constructs content with attached transformations
2. Build() creates an immutable document
3. Renderer applies transformations just before output

This design:
- Preserves original data in the document
- Supports context-aware rendering with cancellation
- Enables efficient rendering with minimal memory overhead

# Performance Characteristics

Per-content transformations have the following performance profile:

- Memory overhead: O(1) per content item (slice of operation references)
- Build time: O(c) where c = number of content items
- Rendering time: O(c × t × r) where c = content items, t = transformations per content, r = records per table
- Each operation clones content during Apply(), chained operations result in multiple clones

The library is designed to handle documents with at least 100 content items, each having
up to 10 transformations, with good performance.

# Context Cancellation

Transformations respect context cancellation:

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out := output.NewOutput(
	    output.WithFormat(output.JSON()),
	    output.WithWriter(output.NewStdoutWriter()),
	)
	err := out.Render(ctx, doc)  // Respects timeout

Context is checked once before each operation in the transformation chain, providing
responsive cancellation without performance overhead in tight loops.

# Error Handling

The library uses fail-fast error handling - rendering stops immediately on the first
transformation error:

	builder.Table("users", data,
	    output.WithTransformations(
	        output.NewSortOp(output.SortKey{Column: "nonexistent"}),  // Error!
	    ),
	)

	doc := builder.Build()
	err := out.Render(ctx, doc)  // Returns error with context about which operation failed

Error messages include:
- Content ID and type
- Operation index and name
- Detailed error description

# Key Order Preservation

A critical feature of v2 is exact key order preservation for tables:

	// Keys appear in specified order, never alphabetized
	builder.Table("users", data, output.WithKeys("name", "age", "email"))

Three ways to specify key order:

1. WithKeys() - explicit key list
2. WithSchema() - full schema with field definitions
3. WithAutoSchema() - auto-detect from data (map iteration order, not recommended)

# Content Types

The library supports multiple content types:

- TableContent: Tabular data with schema-driven key ordering
- TextContent: Unstructured text with styling options
- RawContent: Format-specific content (HTML snippets, etc.)
- SectionContent: Grouped content with hierarchical structure

All content types support transformations through the WithTransformations() option.

# Pipeline API Removal

The document-level Pipeline API has been removed in v2.4.0. Use per-content transformations instead:

	// REMOVED in v2.4.0:
	// doc.Pipeline().Filter(predicate).Sort(keys...).Execute()

	// Use instead:
	builder.Table("data", records,
	    output.WithTransformations(
	        output.NewFilterOp(predicate),
	        output.NewSortOp(keys...),
	    ),
	)

See the migration guide (v2/docs/PIPELINE_MIGRATION.md) for detailed migration examples.

# Best Practices

1. Use explicit WithKeys() or WithSchema() for reliable key ordering
2. Design operations to be stateless and thread-safe
3. Capture values explicitly in closures, not loop variables
4. Use context for cancellation of long-running transformations
5. Test operations with concurrent execution to verify thread safety

For comprehensive best practices, see v2/BEST_PRACTICES.md.

# Example: Complete Workflow

	package main

	import (
	    "context"
	    "log"
	    output "github.com/ArjenSchwarz/go-output/v2"
	)

	func main() {
	    // Sample data
	    users := []output.Record{
	        {"name": "Alice", "email": "alice@example.com", "age": 30},
	        {"name": "Bob", "email": "bob@example.com", "age": 25},
	        {"name": "Charlie", "email": "charlie@example.com", "age": 35},
	    }

	    // Build document with transformations
	    builder := output.New()
	    builder.Table("users", users,
	        output.WithKeys("name", "email", "age"),
	        output.WithTransformations(
	            output.NewFilterOp(func(r output.Record) bool {
	                return r["age"].(int) >= 30
	            }),
	            output.NewSortOp(output.SortKey{
	                Column: "name",
	                Direction: output.Ascending,
	            }),
	        ),
	    )

	    doc := builder.Build()

	    // Render to JSON
	    out := output.NewOutput(
	        output.WithFormat(output.JSON()),
	        output.WithWriter(output.NewStdoutWriter()),
	    )

	    if err := out.Render(context.Background(), doc); err != nil {
	        log.Fatalf("Render error: %v", err)
	    }
	}

# Further Reading

  - Getting Started: v2/docs/GETTING-STARTED.md - Quick start guide
  - Documentation: v2/docs/DOCUMENTATION.md - Complete API documentation
  - Migration from v1: v2/docs/MIGRATION.md - Migrate from v1 to v2
  - Pipeline Migration: v2/docs/PIPELINE_MIGRATION.md - Migrate from Pipeline API to per-content transformations
  - Best Practices: v2/docs/BEST_PRACTICES.md - Safe operation patterns and performance tips
  - API Reference: v2/docs/API.md - Detailed API reference
  - Examples: v2/examples/ - Runnable examples demonstrating features
*/
package output
