# Code Style and Conventions

## Go Language Conventions

### Modern Go Practices (Go 1.24+)
- **Type Declarations**: Use `any` instead of `interface{}` (enforced by golangci-lint)
- **Error Handling**: Proper error wrapping with context
- **Testing**: Leverage Go 1.24+ features including new testing.B.Loop for benchmarks
- **Generics**: Use when appropriate but avoid over-engineering

### Code Style
- **Formatting**: Use `go fmt` for automatic formatting
- **Naming**: Follow standard Go naming conventions
- **Comments**: Add godoc comments for public APIs
- **Simplicity**: Don't overcomplicate implementations

### Architectural Patterns

#### Document-Builder Pattern
```go
doc := output.New().
    Table("Data", records, options).
    Text("Summary").
    Build()
```

#### Functional Options Pattern
```go
WithSchema(fields...)     // Explicit schema with preserved field order
WithKeys(keys...)        // Simple key ordering
WithAutoSchema()         // Auto-detect schema
```

#### Interface-Driven Design
- Major components implement interfaces for extensibility
- Content types implement Content interface
- Transformers implement Transformer/DataTransformer interfaces

### Thread Safety
- All operations must be thread-safe
- Use sync.RWMutex for safe concurrent reads
- Use sync.Mutex for building operations
- Immutable design prevents modification after Build()

### Key Order Preservation
- **Critical Feature**: Exact key order preservation is fundamental
- Never alphabetize or reorder keys
- Preserve user-specified column ordering
- Extensive testing required for key order functionality

### Testing Philosophy
- **Key Order**: Extensive testing of key preservation
- **Thread Safety**: Concurrent operation tests
- **Immutability**: Tests ensuring documents cannot be modified after Build()
- **Interface Compliance**: All content types must implement interfaces properly
- **Performance**: Benchmark critical paths