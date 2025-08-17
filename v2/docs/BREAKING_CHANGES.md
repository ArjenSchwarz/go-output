# Breaking Changes: go-output v1 to v2

This document provides detailed before/after examples for all breaking changes between go-output v1 and v2.

## Table of Contents

1. [Import Path Changes](#1-import-path-changes)
2. [OutputArray Struct Removal](#2-outputarray-struct-removal)
3. [OutputSettings Struct Removal](#3-outputsettings-struct-removal)
4. [Keys Field Changes](#4-keys-field-changes)
5. [AddContents Method Changes](#5-addcontents-method-changes)
6. [Write Method Changes](#6-write-method-changes)
7. [Global State Elimination](#7-global-state-elimination)
8. [String-based Configuration Changes](#8-string-based-configuration-changes)

## 1. Import Path Changes

The import path has changed to include `/v2` for the new version.

### Before (v1)
```go
import (
    "github.com/ArjenSchwarz/go-output/format"
    "github.com/ArjenSchwarz/go-output/drawio"
    "github.com/ArjenSchwarz/go-output/mermaid"
)
```

### After (v2)
```go
import (
    "github.com/ArjenSchwarz/go-output/v2"
)
```

**Impact**: All imports must be updated. The v2 package consolidates all functionality into a single import.

## 2. OutputArray Struct Removal

The `OutputArray` struct has been replaced with a Document Builder pattern.

### Before (v1)
```go
// Direct struct initialization
output := &format.OutputArray{
    Keys:     []string{"Name", "Age"},
    Settings: settings,
}

// Or empty initialization
output := &format.OutputArray{}
output.Keys = []string{"Name", "Age"}
```

### After (v2)
```go
// Builder pattern
doc := output.New().
    Table("", data, output.WithKeys("Name", "Age")).
    Build()

// Document is immutable after Build()
```

**Impact**:
- No more direct struct manipulation
- Fluent API for building documents
- Clear separation between building and rendering

## 3. OutputSettings Struct Removal

The `OutputSettings` struct has been replaced with functional options.

### Before (v1)
```go
settings := format.NewOutputSettings()
settings.OutputFormat = "json"
settings.OutputFile = "report.json"
settings.UseEmoji = true
settings.UseColors = true
settings.SortKey = "Name"
settings.TableStyle = "ColoredBright"
settings.HasTOC = true

output := &format.OutputArray{
    Settings: settings,
}
```

### After (v2)
```go
out := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithWriter(output.NewFileWriter(".", "report.json")),
    output.WithTransformer(&output.EmojiTransformer{}),
    output.WithTransformer(&output.ColorTransformer{}),
    output.WithTransformer(&output.SortTransformer{Key: "Name", Ascending: true}),
    output.WithTableStyle("ColoredBright"),
    output.WithTOC(true),
)
```

**Impact**:
- Settings are now configured through option functions
- More type-safe configuration
- Composable options

## 4. Keys Field Changes

The `Keys` field has been replaced with per-table schemas that preserve order.

### Before (v1)
```go
// Global keys for all content
output.Keys = []string{"ID", "Name", "Status"}
output.AddContents(data1)

// Changing keys affects subsequent content
output.Keys = []string{"Date", "Event", "Count"}
output.AddContents(data2)
```

### After (v2)
```go
// Per-table keys
doc := output.New().
    Table("Users", data1, output.WithKeys("ID", "Name", "Status")).
    Table("Events", data2, output.WithKeys("Date", "Event", "Count")).
    Build()

// Or with schema for more control
doc := output.New().
    Table("Users", data1, output.WithSchema(
        output.Field{Name: "ID", Type: "int"},
        output.Field{Name: "Name", Type: "string"},
        output.Field{Name: "Status", Type: "string"},
    )).
    Build()
```

**Impact**:
- Each table has its own independent key ordering
- No global key state to manage
- More flexible schema definition

## 5. AddContents Method Changes

The `AddContents` method has been replaced with type-specific builder methods.

### Before (v1)
```go
output := &format.OutputArray{}

// Adding table data
output.AddContents(tableData)

// Adding headers
output.AddHeader("Section Title")

// Buffering multiple tables
output.AddContents(data1)
output.AddToBuffer()
output.AddContents(data2)
output.AddToBuffer()
```

### After (v2)
```go
doc := output.New().
    // Adding table data
    Table("", tableData).

    // Adding headers
    Header("Section Title").

    // Multiple tables (no buffering needed)
    Table("First", data1).
    Table("Second", data2).

    Build()
```

**Impact**:
- Clearer API with specific methods for each content type
- No need for manual buffering
- Chainable builder pattern

## 6. Write Method Changes

The `Write` method has been replaced with `Render` that requires context.

### Before (v1)
```go
output := &format.OutputArray{}
output.AddContents(data)

// Simple write
output.Write()

// Or with error handling
err := output.Write()
```

### After (v2)
```go
// Build document
doc := output.New().
    Table("", data).
    Build()

// Create output configuration
out := output.NewOutput(output.WithFormat(output.Table))

// Render with context
ctx := context.Background()
err := out.Render(ctx, doc)

// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
err := out.Render(ctx, doc)
```

**Impact**:
- Context is required for cancellation support
- Separation of document building and rendering
- Better resource management

## 7. Global State Elimination

All global state has been eliminated in v2.

### Before (v1)
```go
// Global settings affected all outputs
format.SetGlobalOption("emoji", true)

// DrawIO global header
drawio.SetHeaderValues(drawio.Header{
    Label: "%Name%",
    Layout: "auto",
})

// Shared state between instances
output1 := &format.OutputArray{}
output2 := &format.OutputArray{}
// Both could interfere with each other
```

### After (v2)
```go
// Each instance is independent
out1 := output.NewOutput(
    output.WithTransformer(&output.EmojiTransformer{}),
)

out2 := output.NewOutput(
    // Different configuration, no interference
    output.WithFormat(output.JSON),
)

// DrawIO configuration is per-document
doc := output.New().
    DrawIO("Diagram", data,
        output.WithDrawIOLabel("%Name%"),
        output.WithDrawIOLayout("auto"),
    ).
    Build()
```

**Impact**:
- Thread-safe by design
- No surprising side effects
- Each instance is completely independent

## 8. String-based Configuration Changes

String-based configuration has been replaced with type-safe options.

### Before (v1)
```go
settings := format.NewOutputSettings()

// String-based format selection
settings.OutputFormat = "json"  // Could typo as "josn"
settings.OutputFileFormat = "html"

// String-based options
settings.SetOption("color", "true")  // String "true" instead of boolean
settings.TableStyle = "ColoredBright"  // Magic string

// Progress color as constant
p.SetColor(format.ProgressColorGreen)
```

### After (v2)
```go
// Type-safe format selection
out := output.NewOutput(
    output.WithFormat(output.JSON),    // Compile-time checked
    output.WithFormat(output.HTML),
)

// Boolean options
output.WithTransformer(&output.ColorTransformer{})  // No string parsing

// Table style (still string but validated)
output.WithTableStyle("ColoredBright")

// Progress color as type-safe constant
p.SetColor(output.ProgressColorGreen)
```

**Impact**:
- Compile-time type checking
- IDE autocompletion support
- Fewer runtime errors

## Migration Strategy

1. **Update imports**: Change all imports to use `/v2`
2. **Replace OutputArray**: Convert to Document Builder pattern
3. **Replace OutputSettings**: Convert to functional options
4. **Update method calls**: Use new builder methods
5. **Add context**: All rendering now requires context
6. **Test thoroughly**: Ensure output matches expectations

## Automated Migration

Use the migration tool for automated conversion:

```bash
# Install migration tool
go install github.com/ArjenSchwarz/go-output/v2/migrate/cmd/migrate@latest

# Run migration
migrate -source ./myproject -verbose
```

The tool handles most common patterns but manual review is recommended.

## Additional Resources

- [Full Migration Guide](./MIGRATION.md)
- [API Documentation](https://pkg.go.dev/github.com/ArjenSchwarz/go-output/v2)
- [Examples](./examples/)