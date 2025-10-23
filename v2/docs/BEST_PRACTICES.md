# Best Practices for Per-Content Transformations

This guide provides patterns and recommendations for writing safe, efficient transformations in go-output v2.

## Table of Contents

- [Thread Safety](#thread-safety)
- [Closure Safety](#closure-safety)
- [Performance Optimization](#performance-optimization)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Common Pitfalls](#common-pitfalls)
- [Streaming Render and Nested Content](#streaming-render-and-nested-content)

## Thread Safety

### Requirement: Stateless Operations

Operations MUST be stateless and thread-safe because they may execute concurrently during rendering.

**Rule**: Operations must not modify mutable state during Apply().

#### ✅ Safe: Pure Functions

```go
// SAFE - no mutable state
filter := output.NewFilterOp(func(r output.Record) bool {
    return r["age"].(int) >= 18
})
```

#### ✅ Safe: Immutable Configuration

```go
// SAFE - configuration is immutable after construction
type SafeFilter struct {
    threshold int  // Immutable after construction
}

func (f *SafeFilter) Apply(ctx context.Context, content output.Content) (output.Content, error) {
    // f.threshold is never modified - safe for concurrent use
    ...
}
```

#### ❌ Unsafe: Mutable State

```go
// WRONG - modifies state during Apply()
type UnsafeCounter struct {
    count int  // Modified during Apply()
}

func (c *UnsafeCounter) Apply(ctx context.Context, content output.Content) (output.Content, error) {
    c.count++  // RACE CONDITION!
    ...
}
```

#### ❌ Unsafe: External Side Effects

```go
// WRONG - writes to file during Apply()
filter := output.NewFilterOp(func(r output.Record) bool {
    logFile.WriteString(fmt.Sprintf("Filtering: %v\n", r))  // UNSAFE!
    return r["active"].(bool)
})
```

### Testing for Thread Safety

Use the library's validation utility:

```go
func TestOperationThreadSafety(t *testing.T) {
    op := output.NewFilterOp(func(r output.Record) bool {
        return r["age"].(int) >= 18
    })

    testContent := &output.TableContent{...}

    err := output.ValidateStatelessOperation(t, op, testContent)
    if err != nil {
        t.Errorf("Operation is not stateless: %v", err)
    }
}
```

Run tests with race detector:

```bash
go test -race ./...
```

## Closure Safety

### The Problem: Loop Variable Capture

Go closures capture variables by reference, which can cause subtle bugs.

#### ❌ Wrong: Capturing Loop Variable

```go
// WRONG - captures loop variable by reference
for _, threshold := range thresholds {
    builder.Table(name, data,
        output.WithTransformations(
            output.NewFilterOp(func(r output.Record) bool {
                // BUG: All closures share the same 'threshold' variable
                return r["value"].(int) > threshold
            }),
        ),
    )
}
```

**Problem**: All created filters reference the same `threshold` variable, which will have the last value from the loop.

#### ✅ Correct: Explicit Value Capture

```go
// CORRECT - captures value explicitly
for _, threshold := range thresholds {
    t := threshold  // Create new variable in loop scope
    builder.Table(name, data,
        output.WithTransformations(
            output.NewFilterOp(func(r output.Record) bool {
                // Correct: Each closure has its own 't' variable
                return r["value"].(int) > t
            }),
        ),
    )
}
```

#### ✅ Alternative: Factory Function

```go
// Helper function creates closure with explicit parameter
func createThresholdFilter(threshold int) output.Operation {
    return output.NewFilterOp(func(r output.Record) bool {
        return r["value"].(int) > threshold
    })
}

for _, threshold := range thresholds {
    builder.Table(name, data,
        output.WithTransformations(
            createThresholdFilter(threshold),
        ),
    )
}
```

### Mutable Variable Capture

#### ❌ Wrong: Capturing Mutable Reference

```go
// WRONG - captures mutable reference
config := &FilterConfig{threshold: 10}

filter := output.NewFilterOp(func(r output.Record) bool {
    return r["value"].(int) > config.threshold  // UNSAFE!
})

// Later...
config.threshold = 20  // Modifies filter behavior - race condition!
```

#### ✅ Correct: Capture Immutable Value

```go
// CORRECT - capture immutable value
config := &FilterConfig{threshold: 10}
threshold := config.threshold  // Copy value

filter := output.NewFilterOp(func(r output.Record) bool {
    return r["value"].(int) > threshold  // Safe
})
```

## Performance Optimization

### Transformation Complexity

Each operation clones content during Apply(). Chain length affects memory usage.

#### Optimal Chain Length

**Recommended**: 3-10 transformations per content item

```go
// Good: Reasonable chain length
builder.Table("users", data,
    output.WithTransformations(
        output.NewFilterOp(isActive),
        output.NewFilterOp(isAdult),
        output.NewSortOp(sortKeys...),
        output.NewLimitOp(100),
    ),
)
```

#### When to Optimize

**Consider optimization when**:
- Chain length exceeds 10 operations
- Large datasets (10,000+ records)
- Multiple complex operations

**Optimization techniques**:

1. **Combine filters**:
   ```go
   // Before: Two separate filters
   output.NewFilterOp(isActive),
   output.NewFilterOp(isAdult),

   // After: One combined filter
   output.NewFilterOp(func(r output.Record) bool {
       return isActive(r) && isAdult(r)
   })
   ```

2. **Filter before sort**:
   ```go
   // Good: Filter reduces data before expensive sort
   output.NewFilterOp(predicate),
   output.NewSortOp(keys...),

   // Less efficient: Sort all data first
   output.NewSortOp(keys...),
   output.NewFilterOp(predicate),
   ```

3. **Limit after transformations**:
   ```go
   // Good: Transformations first, then limit
   output.NewFilterOp(predicate),
   output.NewSortOp(keys...),
   output.NewLimitOp(10),
   ```

### Memory Efficiency

#### Record Size Matters

Large records (many columns or large values) increase cloning cost.

**Guideline**: Each operation clones all records
- Small records (10 columns, ~1KB each): Negligible overhead
- Large records (100 columns, ~100KB each): Significant overhead

**Optimization**: Remove unnecessary columns before transformations:

```go
// If you only need 3 columns, project them early
// (Assuming you have a ProjectOp - example pattern)
output.WithTransformations(
    output.NewFilterOp(predicate),
    // ... other transformations on fewer columns
)
```

### Context Cancellation

Long-running transformations should respect context cancellation.

The library checks context **once before each operation**, not during tight loops.

**What this means**:
- Cancellation detected between operations in chain
- CPU-bound operations (sort, filter) complete without interruption
- Responsive enough for typical workloads (< 1 second per operation)

**When to use timeouts**:

```go
// Set timeout for entire rendering
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

out := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithWriter(output.NewStdoutWriter()),
)
err := out.Render(ctx, doc)
```

## Error Handling

### Fail-Fast Philosophy

The library uses fail-fast error handling - rendering stops on first error.

**Benefits**:
- Clear error messages identifying exact failure point
- No partial/corrupt output
- Simpler error handling logic

#### Error Context

Errors include rich context:

```go
// Error example:
// "content users transformation 2 (sort) failed: column 'age' not found"
//  ↑            ↑              ↑      ↑         ↑
//  content ID   index          name   error message
```

#### Handling Errors

```go
err := out.Render(ctx, doc)
if err != nil {
    // Check for specific error types
    var pipelineErr *output.PipelineError
    if errors.As(err, &pipelineErr) {
        log.Printf("Transformation failed at operation %d: %v",
            pipelineErr.Stage, pipelineErr.Cause)
    }

    var renderErr *output.RenderError
    if errors.As(err, &renderErr) {
        log.Printf("Rendering failed for format %s: %v",
            renderErr.Format, renderErr.Cause)
    }

    return err
}
```

### Validation Best Practices

#### Validate Early

```go
// Validate operation configuration
op := output.NewSortOp(output.SortKey{Column: "name", Direction: output.Ascending})

// Validation happens during rendering, but you can test explicitly
if err := op.Validate(); err != nil {
    log.Fatalf("Invalid operation: %v", err)
}
```

#### Test Error Cases

```go
func TestFilterWithInvalidData(t *testing.T) {
    // Test transformation with data that should fail
    builder := output.New()
    builder.Table("test", invalidData,
        output.WithKeys("name"),
        output.WithTransformations(
            output.NewSortOp(output.SortKey{Column: "nonexistent"}),
        ),
    )

    doc := builder.Build()
    out := output.NewOutput(output.WithFormat(output.JSON))

    err := out.Render(context.Background(), doc)
    if err == nil {
        t.Error("Expected error for nonexistent column")
    }
}
```

## Testing

### Unit Testing Operations

#### Test Operation Logic

```go
func TestFilterOperation(t *testing.T) {
    tests := map[string]struct {
        input    []output.Record
        expected []output.Record
    }{
        "filters_adults": {
            input: []output.Record{
                {"name": "Alice", "age": 30},
                {"name": "Bob", "age": 17},
            },
            expected: []output.Record{
                {"name": "Alice", "age": 30},
            },
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            filter := output.NewFilterOp(func(r output.Record) bool {
                return r["age"].(int) >= 18
            })

            content, _ := output.NewTableContent("test", tc.input,
                output.WithKeys("name", "age"))

            result, err := filter.Apply(context.Background(), content)
            if err != nil {
                t.Fatalf("Unexpected error: %v", err)
            }

            // Verify result matches expected
            // ... assertions
        })
    }
}
```

#### Test Thread Safety

```go
func TestConcurrentOperations(t *testing.T) {
    op := output.NewFilterOp(func(r output.Record) bool {
        return r["active"].(bool)
    })

    content, _ := output.NewTableContent("test", testData,
        output.WithKeys("name", "active"))

    // Run operation concurrently
    var wg sync.WaitGroup
    errors := make(chan error, 10)

    for range 10 {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := op.Apply(context.Background(), content)
            if err != nil {
                errors <- err
            }
        }()
    }

    wg.Wait()
    close(errors)

    // Check for race conditions
    for err := range errors {
        t.Errorf("Concurrent operation failed: %v", err)
    }
}
```

### Integration Testing

Test complete workflow with transformations:

```go
func TestDocumentWithTransformations(t *testing.T) {
    builder := output.New()
    builder.Table("users", testUsers,
        output.WithKeys("name", "age"),
        output.WithTransformations(
            output.NewFilterOp(isAdult),
            output.NewSortOp(sortByName),
        ),
    )

    doc := builder.Build()

    out := output.NewOutput(
        output.WithFormat(output.JSON),
        output.WithWriter(output.NewStdoutWriter()),
    )

    err := out.Render(context.Background(), doc)
    if err != nil {
        t.Fatalf("Render failed: %v", err)
    }

    // Verify output matches expected
}
```

## Common Pitfalls

### Pitfall 1: Reusing Operations Across Tables

**Problem**: Same operation instance shared between tables might seem efficient but can cause confusion.

#### ⚠️ Potentially Confusing

```go
filterOp := output.NewFilterOp(predicate)

builder.Table("table1", data1, output.WithTransformations(filterOp))
builder.Table("table2", data2, output.WithTransformations(filterOp))
```

**Better**: Create separate instances for clarity:

```go
builder.Table("table1", data1,
    output.WithTransformations(output.NewFilterOp(predicate)))
builder.Table("table2", data2,
    output.WithTransformations(output.NewFilterOp(predicate)))
```

**Or**: Use helper function:

```go
func activeFilter() output.Operation {
    return output.NewFilterOp(func(r output.Record) bool {
        return r["active"].(bool)
    })
}

builder.Table("table1", data1, output.WithTransformations(activeFilter()))
builder.Table("table2", data2, output.WithTransformations(activeFilter()))
```

### Pitfall 2: Forgetting Context Timeout

**Problem**: Long-running transformations without timeout can hang.

#### ❌ No Timeout

```go
err := out.Render(context.Background(), doc)  // No timeout!
```

#### ✅ With Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := out.Render(ctx, doc)
```

### Pitfall 3: Complex Predicates

**Problem**: Overly complex filter predicates reduce readability and testability.

#### ❌ Complex Inline Predicate

```go
filter := output.NewFilterOp(func(r output.Record) bool {
    age, ok := r["age"].(int)
    if !ok {
        return false
    }
    status, ok := r["status"].(string)
    if !ok {
        return false
    }
    dept, ok := r["department"].(string)
    if !ok {
        return false
    }
    return age >= 18 && age <= 65 &&
           (status == "active" || status == "pending") &&
           (dept == "Engineering" || dept == "Sales")
})
```

#### ✅ Named Helper Function

```go
func isEligibleEmployee(r output.Record) bool {
    age, ok := r["age"].(int)
    if !ok || age < 18 || age > 65 {
        return false
    }

    status, ok := r["status"].(string)
    if !ok {
        return false
    }
    if status != "active" && status != "pending" {
        return false
    }

    dept, ok := r["department"].(string)
    if !ok {
        return false
    }
    return dept == "Engineering" || dept == "Sales"
}

filter := output.NewFilterOp(isEligibleEmployee)
```

**Benefits**:
- Easier to test
- Reusable across tables
- Self-documenting

### Pitfall 4: Incorrect Column Names

**Problem**: Typos in column names cause runtime errors.

#### ❌ Typo in Column Name

```go
sort := output.NewSortOp(output.SortKey{
    Column: "naem",  // Typo!
    Direction: output.Ascending,
})
```

**Error discovered at render time**:
```
transformation 0 (sort) failed: column 'naem' not found
```

#### ✅ Use Constants

```go
const (
    ColumnName  = "name"
    ColumnAge   = "age"
    ColumnEmail = "email"
)

sort := output.NewSortOp(output.SortKey{
    Column: ColumnName,
    Direction: output.Ascending,
})
```

### Pitfall 5: Type Assertions Without Checks

**Problem**: Unchecked type assertions panic on unexpected data.

#### ❌ Unchecked Assertion

```go
filter := output.NewFilterOp(func(r output.Record) bool {
    return r["age"].(int) >= 18  // Panics if age is not int!
})
```

#### ✅ Safe Type Check

```go
filter := output.NewFilterOp(func(r output.Record) bool {
    age, ok := r["age"].(int)
    if !ok {
        return false  // Skip records with invalid age
    }
    return age >= 18
})
```

#### ✅ Alternative: Panic and Recover Pattern

```go
filter := output.NewFilterOp(func(r output.Record) bool {
    defer func() {
        if r := recover(); r != nil {
            // Log or handle panic
            return false
        }
    }()
    return r["age"].(int) >= 18
})
```

## Streaming Render and Nested Content

### Streaming Render Consistency

Both `Render()` and `RenderTo()` methods apply per-content transformations identically. Use `RenderTo()` for large documents to avoid buffering entire output in memory.

```go
table, _ := output.NewTableContent("data", records,
    output.WithKeys("name", "value"),
    output.WithTransformations(
        output.NewSortOp(output.SortKey{Column: "value", Direction: output.Descending}),
        output.NewLimitOp(100),
    ),
)

doc := output.New().AddContent(table).Build()

// Both produce identical output
buffered, _ := renderer.Render(ctx, doc)
renderer.RenderTo(ctx, doc, writer)  // Streams directly to writer
```

### Nested Content Transformations

Transformations are properly applied to content nested within sections and collapsible sections.

```go
// Table with transformations
filteredTable, _ := output.NewTableContent("active-users", users,
    output.WithKeys("name", "email"),
    output.WithTransformations(
        output.NewFilterOp(func(r output.Record) bool {
            return r["active"].(bool)
        }),
    ),
)

// Create section containing the table
section := output.NewSectionContent("User Management")
section.AddContent(filteredTable)

doc := output.New().AddContent(section).Build()

// Transformations are applied during rendering
// regardless of nesting depth
result, _ := renderer.Render(ctx, doc)
```

### Multi-Level Nesting

Transformations work at any nesting depth:

```go
table, _ := output.NewTableContent("data", records,
    output.WithTransformations(output.NewLimitOp(10)),
)

innerSection := output.NewSectionContent("Inner")
innerSection.AddContent(table)

outerSection := output.NewSectionContent("Outer")
outerSection.AddContent(innerSection)

// Transformation executes correctly at any depth
doc := output.New().AddContent(outerSection).Build()
```

### Best Practices for Nested Content

1. **Apply transformations at the content level**, not the section level
2. **Keep nesting shallow** (≤ 3 levels) for maintainability
3. **Test nested scenarios** if using sections with transformations
4. **Use consistent rendering** - both `Render()` and `RenderTo()` work identically

## Summary Checklist

- [ ] Operations are stateless and thread-safe
- [ ] Closures capture values, not loop variables
- [ ] Filter operations come before expensive sorts
- [ ] Context timeouts set for long-running renders
- [ ] Column names use constants to avoid typos
- [ ] Type assertions include safety checks
- [ ] Thread safety validated with race detector
- [ ] Error cases tested with integration tests
- [ ] Complex predicates extracted to named functions
- [ ] Transformation chains kept under 10 operations
- [ ] Nested content transformations tested if using sections
- [ ] Streaming renders (`RenderTo`) tested for large documents

## Further Resources

- [Migration Guide](MIGRATION.md) - Migrate from Pipeline API
- [Package Documentation](https://pkg.go.dev/github.com/ArjenSchwarz/go-output/v2) - API reference
- [Examples](examples/) - Runnable code examples
