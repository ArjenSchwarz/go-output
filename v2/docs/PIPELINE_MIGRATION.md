# Migration Guide: Pipeline API to Per-Content Transformations

This guide helps you migrate from the removed document-level Pipeline API to per-content transformations.

## Overview

The document-level Pipeline API (`Document.Pipeline()`, `Filter()`, `Sort()`, etc.) has been **removed in v2.4.0**. Per-content transformations provide a more flexible model where each table can have its own unique transformation logic.

**Key Benefits:**
- Different transformations for different tables in the same document
- Clearer intent - transformations are defined where content is created
- Better composability - build tables with their transformations in one place
- No global state - transformations belong to content items

## Migration Timeline

- **v2.0-v2.3**: Pipeline API was available but deprecated
- **v2.4.0+**: Pipeline API removed - use per-content transformations instead

## Basic Migration Examples

### Example 1: Simple Filter and Sort

**Before (Pipeline API - REMOVED):**
```go
builder := output.New()
builder.Table("users", users, output.WithKeys("name", "email", "age"))

doc := builder.Build()

// Apply transformations globally
transformed, err := doc.Pipeline().
    Filter(func(r output.Record) bool {
        return r["age"].(int) >= 18
    }).
    Sort(output.SortKey{Column: "name", Direction: output.Ascending}).
    Execute()
```

**After (Per-Content Transformations):**
```go
builder := output.New()

// Attach transformations directly to the table
builder.Table("users", users,
    output.WithKeys("name", "email", "age"),
    output.WithTransformations(
        output.NewFilterOp(func(r output.Record) bool {
            return r["age"].(int) >= 18
        }),
        output.NewSortOp(output.SortKey{Column: "name", Direction: output.Ascending}),
    ),
)

doc := builder.Build()
// Transformations apply automatically during rendering
```

### Example 2: Multiple Tables with Different Transformations

**Before (Pipeline API - REMOVED):**
```go
// Problem: Pipeline applies to ALL tables in the document
builder := output.New()
builder.Table("users", users, output.WithKeys("name", "email", "age"))
builder.Table("products", products, output.WithKeys("id", "name", "price"))

doc := builder.Build()

// This filters BOTH tables - not what we want!
transformed, err := doc.Pipeline().
    Filter(func(r output.Record) bool {
        return r["age"].(int) >= 18  // Error: products don't have "age"
    }).
    Execute()
```

**After (Per-Content Transformations):**
```go
builder := output.New()

// Each table gets its own transformations
builder.Table("users", users,
    output.WithKeys("name", "email", "age"),
    output.WithTransformations(
        output.NewFilterOp(func(r output.Record) bool {
            return r["age"].(int) >= 18
        }),
        output.NewSortOp(output.SortKey{Column: "name", Direction: output.Ascending}),
    ),
)

builder.Table("products", products,
    output.WithKeys("id", "name", "price"),
    output.WithTransformations(
        output.NewSortOp(output.SortKey{Column: "price", Direction: output.Descending}),
        output.NewLimitOp(10),  // Top 10 products by price
    ),
)

doc := builder.Build()
```

### Example 3: Limit Operation

**Before (Pipeline API - deprecated):**
```go
transformed, err := doc.Pipeline().
    Limit(10).
    Execute()
```

**After (Per-Content Transformations):**
```go
builder.Table("data", records,
    output.WithKeys("name", "value"),
    output.WithTransformations(
        output.NewLimitOp(10),
    ),
)
```

### Example 4: GroupBy with Aggregates

**Before (Pipeline API - deprecated):**
```go
transformed, err := doc.Pipeline().
    GroupBy(
        []string{"category"},
        map[string]output.AggregateFunc{
            "total": output.Sum("amount"),
            "count": output.Count(),
        },
    ).
    Execute()
```

**After (Per-Content Transformations):**
```go
builder.Table("summary", records,
    output.WithKeys("category", "total", "count"),
    output.WithTransformations(
        output.NewGroupByOp(
            []string{"category"},
            map[string]output.AggregateFunc{
                "total": output.Sum("amount"),
                "count": output.Count(),
            },
        ),
    ),
)
```

### Example 5: AddColumn Operation

**Before (Pipeline API - deprecated):**
```go
transformed, err := doc.Pipeline().
    AddColumn("full_name", func(r output.Record) any {
        return r["first_name"].(string) + " " + r["last_name"].(string)
    }).
    Execute()
```

**After (Per-Content Transformations):**
```go
builder.Table("people", records,
    output.WithKeys("first_name", "last_name", "full_name"),
    output.WithTransformations(
        output.NewAddColumnOp("full_name", func(r output.Record) any {
            return r["first_name"].(string) + " " + r["last_name"].(string)
        }, nil),  // nil position = append to end
    ),
)
```

### Example 6: AddColumnAt with Position

**Before (Pipeline API - deprecated):**
```go
transformed, err := doc.Pipeline().
    AddColumnAt("id", func(r output.Record) any {
        return uuid.New().String()
    }, 0).  // Insert at beginning
    Execute()
```

**After (Per-Content Transformations):**
```go
pos := 0
builder.Table("data", records,
    output.WithKeys("id", "name", "value"),
    output.WithTransformations(
        output.NewAddColumnOp("id", func(r output.Record) any {
            return uuid.New().String()
        }, &pos),  // Insert at position 0
    ),
)
```

### Example 7: Custom Sort Comparator

**Before (Pipeline API - deprecated):**
```go
transformed, err := doc.Pipeline().
    SortWith(func(a, b output.Record) int {
        aScore := calculateScore(a)
        bScore := calculateScore(b)
        if aScore < bScore {
            return -1
        }
        if aScore > bScore {
            return 1
        }
        return 0
    }).
    Execute()
```

**After (Per-Content Transformations):**
```go
builder.Table("ranked", records,
    output.WithKeys("name", "score"),
    output.WithTransformations(
        output.NewSortOpWithComparator(func(a, b output.Record) int {
            aScore := calculateScore(a)
            bScore := calculateScore(b)
            if aScore < bScore {
                return -1
            }
            if aScore > bScore {
                return 1
            }
            return 0
        }),
    ),
)
```

## Advanced Migration Patterns

### Dynamic Transformation Construction

**Before (Pipeline API - deprecated):**
```go
pipeline := doc.Pipeline()

if userWantsFiltering {
    pipeline = pipeline.Filter(predicate)
}
if userWantsSorting {
    pipeline = pipeline.Sort(keys...)
}
if userWantsLimit {
    pipeline = pipeline.Limit(count)
}

transformed, err := pipeline.Execute()
```

**After (Per-Content Transformations):**
```go
var transformations []output.Operation

if userWantsFiltering {
    transformations = append(transformations, output.NewFilterOp(predicate))
}
if userWantsSorting {
    transformations = append(transformations, output.NewSortOp(keys...))
}
if userWantsLimit {
    transformations = append(transformations, output.NewLimitOp(count))
}

builder.Table("data", records,
    output.WithKeys(...),
    output.WithTransformations(transformations...),
)
```

### Chaining Multiple Operations

**Before (Pipeline API - deprecated):**
```go
transformed, err := doc.Pipeline().
    Filter(activeFilter).
    Filter(dateRangeFilter).
    Sort(
        output.SortKey{Column: "priority", Direction: output.Descending},
        output.SortKey{Column: "created_at", Direction: output.Ascending},
    ).
    Limit(20).
    Execute()
```

**After (Per-Content Transformations):**
```go
builder.Table("tasks", records,
    output.WithKeys("title", "priority", "created_at", "status"),
    output.WithTransformations(
        output.NewFilterOp(activeFilter),
        output.NewFilterOp(dateRangeFilter),
        output.NewSortOp(
            output.SortKey{Column: "priority", Direction: output.Descending},
            output.SortKey{Column: "created_at", Direction: output.Ascending},
        ),
        output.NewLimitOp(20),
    ),
)
```

### Context-Aware Rendering

**Before (Pipeline API - deprecated):**
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

transformed, err := doc.Pipeline().
    Filter(expensiveFilter).
    ExecuteContext(ctx)
```

**After (Per-Content Transformations):**
```go
// Context is passed during rendering
builder.Table("data", records,
    output.WithKeys("name", "value"),
    output.WithTransformations(
        output.NewFilterOp(expensiveFilter),
    ),
)

doc := builder.Build()

// Pass context to renderer
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

renderer := output.NewJSONRenderer()
result, err := renderer.Render(ctx, doc)
```

## Operation Reference

All pipeline operations have corresponding operation constructors:

| Pipeline Method | Per-Content Operation |
|----------------|----------------------|
| `.Filter(predicate)` | `NewFilterOp(predicate)` |
| `.Sort(keys...)` | `NewSortOp(keys...)` |
| `.SortBy(column, direction)` | `NewSortOp(SortKey{Column: column, Direction: direction})` |
| `.SortWith(comparator)` | `NewSortOpWithComparator(comparator)` |
| `.Limit(count)` | `NewLimitOp(count)` |
| `.GroupBy(columns, aggregates)` | `NewGroupByOp(columns, aggregates)` |
| `.AddColumn(name, fn)` | `NewAddColumnOp(name, fn, nil)` |
| `.AddColumnAt(name, fn, position)` | `NewAddColumnOp(name, fn, &position)` |

## Migration Checklist

- [ ] Identify all uses of `doc.Pipeline()`
- [ ] For each pipeline usage:
  - [ ] Identify which table(s) the transformations should apply to
  - [ ] Convert pipeline operations to `NewXxxOp()` constructors
  - [ ] Add `WithTransformations()` option to table creation
  - [ ] Remove `Pipeline().Execute()` calls
- [ ] Test that transformations produce same results
- [ ] Update tests to use per-content transformations
- [ ] Remove any pipeline-related imports if no longer needed

## Common Pitfalls

### Pitfall 1: Forgetting to Remove Execute() Call

**Wrong:**
```go
builder.Table("data", records,
    output.WithTransformations(
        output.NewFilterOp(predicate),
    ),
)

doc := builder.Build()
transformed, err := doc.Pipeline().Execute()  // Don't do this!
```

**Correct:**
```go
builder.Table("data", records,
    output.WithTransformations(
        output.NewFilterOp(predicate),
    ),
)

doc := builder.Build()
// Transformations apply during rendering automatically
```

### Pitfall 2: Applying Same Transformations to Multiple Tables

**Inefficient:**
```go
ops := []output.Operation{
    output.NewFilterOp(predicate),
    output.NewSortOp(keys...),
}

builder.Table("table1", data1, output.WithTransformations(ops...))
builder.Table("table2", data2, output.WithTransformations(ops...))
```

**Better:**
```go
// Create a helper function if transformations are reused
func commonTransformations() []output.Operation {
    return []output.Operation{
        output.NewFilterOp(predicate),
        output.NewSortOp(keys...),
    }
}

builder.Table("table1", data1, output.WithTransformations(commonTransformations()...))
builder.Table("table2", data2, output.WithTransformations(commonTransformations()...))
```

### Pitfall 3: Closure Variable Capture

**Wrong (captures loop variable by reference):**
```go
for _, threshold := range thresholds {
    builder.Table(fmt.Sprintf("data_%d", threshold), records,
        output.WithTransformations(
            output.NewFilterOp(func(r output.Record) bool {
                return r["value"].(int) > threshold  // BUG: captures by reference!
            }),
        ),
    )
}
```

**Correct (explicit parameter):**
```go
for _, threshold := range thresholds {
    t := threshold  // Capture value explicitly
    builder.Table(fmt.Sprintf("data_%d", t), records,
        output.WithTransformations(
            output.NewFilterOp(func(r output.Record) bool {
                return r["value"].(int) > t  // Correct: uses captured value
            }),
        ),
    )
}
```

## Getting Help

If you encounter issues during migration:

1. Check the [Best Practices Guide](BEST_PRACTICES.md) for safe operation patterns
2. Review the [package documentation](https://pkg.go.dev/github.com/yourorg/go-output/v2) for API details
3. Open an issue at https://github.com/yourorg/go-output/issues with your migration question

## Why the Change?

The Pipeline API had limitations:

1. **Global transformations**: All tables in a document shared the same transformations
2. **Complex mental model**: Build document → Transform document → Render
3. **Optimization conflicts**: Per-table optimizations impossible with global pipeline

Per-content transformations solve these issues:

1. **Granular control**: Each table has its own transformations
2. **Clear intent**: Transformations declared with content
3. **Better composability**: Build and transform in one place
4. **Future-proof**: Supports text transformations, section transformations, etc.
