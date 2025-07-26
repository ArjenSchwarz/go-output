# Migration Examples

This document provides practical examples of migrating from go-output v1 to v2.

## Basic Usage

### Simple Table Output

Basic table output without configuration

**Before (v1):**
```go
output := &format.OutputArray{}
output.AddContents(data)
output.Write()
```

**After (v2):**
```go
ctx := context.Background()
doc := output.New().
    Table("", data).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)
```

**Notes:**
- Context is now required for all operations
- Document building is separate from rendering

### Table with Key Ordering

Preserve specific column order

**Before (v1):**
```go
output := &format.OutputArray{
    Keys: []string{"ID", "Name", "Status"},
}
output.AddContents(data)
output.Write()
```

**After (v2):**
```go
doc := output.New().
    Table("", data, output.WithKeys("ID", "Name", "Status")).
    Build()
```

**Notes:**
- Key order is preserved exactly as specified
- Each table can have different key ordering

## Configuration

### Output Format Configuration

Configure output format and options

**Before (v1):**
```go
settings := format.NewOutputSettings()
settings.OutputFormat = "json"
settings.UseEmoji = true
settings.UseColors = true
```

**After (v2):**
```go
out := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithTransformer(&output.EmojiTransformer{}),
    output.WithTransformer(&output.ColorTransformer{}),
)
```

**Notes:**
- Settings replaced with functional options
- Features are now transformers

### File Output

Write output to file

**Before (v1):**
```go
settings.OutputFile = "report.csv"
settings.OutputFormat = "csv"
```

**After (v2):**
```go
output.WithFormat(output.CSV),
output.WithWriter(output.NewFileWriter(".", "report.csv"))
```

**Notes:**
- File output uses dedicated Writer
- Can output to multiple destinations

## Advanced Features

### Progress Indicators

Show progress during processing

**Before (v1):**
```go
p := format.NewProgress(settings)
p.SetTotal(100)
p.SetColor(format.ProgressColorGreen)
```

**After (v2):**
```go
p := output.NewProgress(output.Table,
    output.WithProgressColor(output.ProgressColorGreen),
)
p.SetTotal(100)
```

**Notes:**
- Progress is format-aware
- Color constants moved to output package

### Multiple Tables

Output multiple tables with different schemas

**Before (v1):**
```go
output.Keys = []string{"Name", "Email"}
output.AddContents(users)
output.AddToBuffer()

output.Keys = []string{"Service", "Status"}
output.AddContents(services)
output.Write()
```

**After (v2):**
```go
doc := output.New().
    Table("Users", users, output.WithKeys("Name", "Email")).
    Table("Services", services, output.WithKeys("Service", "Status")).
    Build()
```

**Notes:**
- No need for AddToBuffer()
- Each table has independent schema