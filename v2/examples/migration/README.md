# Migration Examples from v1 to v2

This directory contains complete, runnable examples showing how to migrate common v1 patterns to v2.

## Running the Examples

Each example can be run independently to see the v2 equivalent:

```bash
cd migration
go run basic_patterns/main.go
go run advanced_patterns/main.go
go run complete_migration/main.go
```

## Example Categories

### 1. Basic Patterns
- Simple table output
- Key ordering preservation  
- Multiple tables
- File output

### 2. Advanced Patterns  
- Output settings migration
- Progress indicators
- Transformers (emoji, colors, sorting)
- Multiple formats

### 3. Complete Migration
- Full v1 → v2 conversion
- Complex document with all features
- Performance comparison

## Key Migration Concepts

### 1. Document-Builder Pattern
**v1**: Global state with OutputArray
**v2**: Immutable documents with Builder pattern

### 2. Key Order Preservation
**v1**: Unpredictable due to map iteration
**v2**: Exact user-specified ordering preserved

### 3. Configuration
**v1**: OutputSettings struct
**v2**: Functional options pattern

### 4. Thread Safety
**v1**: Global state causes race conditions
**v2**: Thread-safe by design

### 5. Error Handling
**v1**: Basic error returns
**v2**: Structured error types with context

## Migration Checklist

When migrating from v1 to v2, follow this checklist:

- [ ] Update import path to `/v2`
- [ ] Replace OutputArray with Document Builder
- [ ] Convert Keys field to WithKeys() option
- [ ] Convert OutputSettings to OutputOption functions
- [ ] Replace AddContents with Table()
- [ ] Replace AddToBuffer with separate Table() calls
- [ ] Replace Write() with Build() and Render()
- [ ] Add context.Context to Render calls
- [ ] Convert file output to FileWriter
- [ ] Update progress creation to format-aware constructors
- [ ] Ensure key orders are explicitly specified