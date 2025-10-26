# go-output v2.4.0 Release Notes

## Breaking Changes

⚠️ **Pipeline API Removed** - The document-level Pipeline API has been removed in favor of the more flexible per-content transformations system introduced in v2.2.0.

**Migration Required:**
- Replace `doc.Pipeline()` with `WithTransformations()` on individual content items
- Each table, text, section, or raw content can now have its own transformations
- See [v2/docs/PIPELINE_MIGRATION.md](v2/docs/PIPELINE_MIGRATION.md) for detailed migration patterns

**Why this change:**
The Pipeline API was too limiting - it applied transformations uniformly across all content in a document. Per-content transformations allow different operations on different tables in the same document, which is the primary use case.

## Major Features

### 🎯 Per-Content Transformations

Apply transformations to individual content items at creation time, enabling different operations on different content:

```go
doc := output.New().
    Table("High Earners", employees,
        output.WithKeys("Name", "Department", "Salary"),
        output.WithTransformations(
            output.NewFilterOp(func(r output.Record) bool {
                return r["Salary"].(float64) > 100000
            }),
            output.NewSortOp(output.SortKey{Column: "Salary", Direction: output.Descending}),
        ),
    ).
    Table("Active Projects", projects,
        output.WithKeys("Project", "Status", "Priority"),
        output.WithTransformations(
            output.NewFilterOp(func(r output.Record) bool {
                return r["Status"] == "Active"
            }),
            output.NewLimitOp(10),
        ),
    ).
    Build()
```

**Benefits:**
- Different transformations per table in the same document
- More intuitive - transformations defined where content is created
- Better performance - only transforms what needs transforming
- Full renderer integration across all formats

### 📝 File Append Mode

Append content to existing files instead of replacing them:

```go
fw, _ := output.NewFileWriterWithOptions(
    "./logs",
    "app.{ext}",
    output.WithAppendMode(),
)
```

**Format-Specific Behavior:**
- **JSON/YAML**: Byte-level appending (perfect for NDJSON logging)
- **CSV**: Automatically strips duplicate headers when appending
- **HTML**: Marker-based insertion using `<!-- go-output-append -->` with atomic writes for crash safety

**Options:**
- `WithAppendMode()` - Enable append mode
- `WithPermissions(mode)` - Set custom file permissions (default 0644)
- `WithDisallowUnsafeAppend()` - Prevent JSON/YAML appends

See [v2/examples/append_mode/](v2/examples/append_mode/) for practical examples.

### ☁️ S3 Append Support

Extend append functionality to S3 objects with conflict detection:

```go
s3Writer, _ := output.NewS3WriterWithOptions(
    s3Client,
    "bucket-name",
    "logs/{date}.{ext}",
    output.WithS3AppendMode(),
    output.WithMaxAppendSize(100 * 1024 * 1024), // 100MB limit
)
```

**Features:**
- Download-modify-upload pattern for infrequent appends
- ETag-based optimistic locking for concurrent modification detection
- Format-aware data combining (HTML markers, CSV header stripping)
- Configurable size limits to prevent appending to large objects

### 🎨 HTML Template System

Generate complete HTML documents with responsive styling:

```go
htmlFormat := output.HTML.WithOptions(
    output.WithHTMLTemplate(output.DefaultHTMLTemplate),
)
```

**Three Built-in Templates:**
- `DefaultHTMLTemplate` - Modern responsive design with mobile-first CSS
- `MinimalHTMLTemplate` - Clean HTML with no styling
- `MermaidHTMLTemplate` - Optimized for diagram rendering

**Features:**
- Mobile-first responsive CSS with WCAG AA compliant colors
- Customizable via `HTMLTemplate` struct (title, CSS, meta tags, head/body injection)
- CSS custom properties for easy theme customization
- System font stack for optimal performance
- Automatic fragment mode for append operations

## Critical Bug Fixes

### 🔧 Transformation Rendering Consistency

Fixed critical bugs where `RenderTo()` methods bypassed per-content transformations:

- All renderers now apply transformations consistently in both `Render()` and `RenderTo()` paths
- Fixed transformations being ignored for content nested within sections
- Streaming render now produces identical output to buffered render
- Added comprehensive tests to prevent regression

### 🔐 HTML Append File Permissions

Fixed atomic write operations losing original file permissions:

- HTML append uses write-to-temp-and-rename for crash safety
- Now preserves original file permissions (e.g., 0644, 0755) after atomic rename
- Added `os.Chmod()` to restore permissions after atomic operations

### 🏁 Race Condition Fixes

Eliminated race detector warnings:

- Removed all `t.Parallel()` calls that caused concurrent access to shared global state
- Tests now run sequentially to prevent race conditions
- All race detector warnings resolved: `go test -race ./...` passes cleanly

## Code Quality Improvements

- **Modernized to Go 1.24+** - Using `maps.Copy()` and other modern patterns
- **Consolidated Clone Implementation** - Complete field coverage for all content types
- **Testing Best Practices** - All tests use map-based table-driven patterns per Go 2025 guidelines

## Upgrade Instructions

1. **For Pipeline API users**: Review [v2/docs/PIPELINE_MIGRATION.md](v2/docs/PIPELINE_MIGRATION.md) and migrate to `WithTransformations()`
2. **Update your import**: Ensure you're importing `github.com/ArjenSchwarz/go-output/v2`
3. **Run tests**: Verify your code with `go test ./...` and `go test -race ./...`
4. **Check linting**: Run `golangci-lint run` to catch any issues

## Documentation

- 📚 [Getting Started Guide](v2/docs/GETTING-STARTED.md)
- 📖 [API Reference](v2/docs/API.md)
- 🔄 [Migration Guide](v2/docs/MIGRATION.md)
- 🚀 [Pipeline Migration](v2/docs/PIPELINE_MIGRATION.md)
- ✅ [Best Practices](v2/docs/BEST_PRACTICES.md)

## Examples

- [Per-Content Transformations](v2/examples/transformations/)
- [Append Mode (File & S3)](v2/examples/append_mode/)
- [HTML Templates](v2/examples/) (in standard examples)

## Contributors

This release represents significant architectural improvements focused on flexibility, correctness, and user experience. Thank you to everyone who provided feedback and helped identify issues!

## Full Changelog

See [CHANGELOG.md](CHANGELOG.md#240--2025-10-26) for complete details.
