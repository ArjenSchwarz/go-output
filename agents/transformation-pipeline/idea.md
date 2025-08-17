# Current v2 Transformation System: Gaps Analysis

## Current State

The v2 transformation system is essentially a **post-rendering text manipulation system**:

```go
Document → Renderer → bytes → Transformer1 → Transformer2 → Final bytes
```

Current transformers:
- **EmojiTransformer**: Text replacement in output
- **ColorTransformer**: Adds ANSI codes
- **SortTransformer**: Would need to parse rendered format to sort
- **LineSplitTransformer**: Splits text lines

## Major Gaps

### 1. **Data-Level Operations Are Impossible**

**Gap**: Cannot filter, aggregate, or transform data after document creation
```go
// Currently impossible:
doc.Filter(records where amount > 100)
doc.GroupBy("category").Sum("amount")
doc.AddCalculatedColumn("tax", amount * 0.15)
```

**Impact**: Users must do ALL data manipulation before calling `output.New()`

### 2. **Format-Specific Transformer Limitations**

**Gap**: Transformers receive bytes without format context
```go
// A SortTransformer would need different logic for:
// - JSON: Parse JSON, sort, re-encode
// - CSV: Parse CSV, sort, re-encode
// - Table: Parse ANSI table, sort, re-render
// Each format needs completely different implementation
```

**Impact**: Complex transformers become nearly impossible to implement correctly

### 3. **No Semantic Understanding**

**Gap**: Transformers work on text, not structured data
```go
// Current: "Find and replace :check: with ✓"
// Impossible: "Sum all values in the Amount column"
// Impossible: "Filter rows where Status = 'Active'"
```

**Impact**: Limited to cosmetic changes only

### 4. **Performance Issues**

**Gap**: Multiple parse/render cycles
```go
// To sort data in current system:
// 1. Render to JSON → 2. Parse JSON → 3. Sort → 4. Re-render to JSON
// This happens for EACH transformer that needs data access
```

**Impact**: Significant performance overhead for data operations

### 5. **No Composability**

**Gap**: Cannot build complex operations from simple ones
```go
// Cannot do:
filter().then(sort()).then(limit())
// Each would need to parse/modify/re-render the entire output
```

**Impact**: Users cannot build reusable transformation logic

## What Needs to Be Done

### Option 1: Enhanced Transformer System (Minimal Change)

**Add format-aware transformers that receive structured data:**

```go
type DataTransformer interface {
    Transformer // Existing interface

    // New method - receives parsed data instead of bytes
    TransformData(ctx context.Context, data []Record, schema *Schema, format string) ([]Record, *Schema, error)
}

// Renderer would check if transformer implements DataTransformer
// If yes: transform data before rendering
// If no: fall back to byte transformation after rendering
```

**Pros**:
- Backward compatible
- Minimal API changes
- Gradual migration path

**Cons**:
- Still limited to table data
- No complex operations like joins
- Two different transformer types

### Option 2: Separate Data Pipeline (Recommended)

**Add a proper data transformation pipeline:**

```go
// New pipeline API - operates on documents before rendering
doc := output.New().
    Table("Sales", data).
    Build()

// Transform the document's data
transformed := doc.Transform(
    output.Filter(func(r Record) bool { return r["amount"].(float64) > 100 }),
    output.Sort("date", true),
    output.Limit(100),
)

// Or use builder pattern
transformed := doc.Pipeline().
    Filter(expr).
    Sort(column).
    Execute()
```

**Pros**:
- Clean separation of concerns
- Powerful data operations
- Type-safe and optimizable
- Composable operations

**Cons**:
- New API to learn
- Larger implementation effort

### Option 3: Hybrid Approach (Best of Both)

**Enhance existing system AND add data pipeline:**

```go
// 1. Enhanced transformers for simple cases
output.WithTransformer(&SortTransformer{Column: "date"}) // Works on data

// 2. Pipeline for complex cases
doc.Pipeline().Filter(...).GroupBy(...).Execute()

// 3. Keep byte transformers for format-specific needs
output.WithTransformer(&ColorTransformer{}) // Still works on bytes
```

## Implementation Roadmap

### Phase 1: Foundation (No Breaking Changes)
1. Add `DataTransformer` interface
2. Update renderers to support data transformers
3. Migrate existing transformers where sensible
4. Add basic data transformers (Sort, Filter)

### Phase 2: Pipeline API (Additive)
1. Add `Pipeline()` method to Document
2. Implement core operations (Filter, Map, Sort, Limit)
3. Add expression system for conditions
4. Create pipeline optimizer

### Phase 3: Advanced Features (Optional)
1. Aggregation operations (GroupBy, Sum, etc.)
2. Join operations between tables
3. Window functions
4. Custom function registry

## Impact on Current Users

### No Breaking Changes

**Current code continues to work:**
```go
// This still works exactly as before
out := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithTransformer(&EmojiTransformer{}),
)
```

### New Capabilities Are Additive

**Users can opt-in to new features:**
```go
// Option 1: Use enhanced transformers
output.WithTransformer(&SortTransformer{
    Column: "date",
    Ascending: true,
})

// Option 2: Use pipeline for complex logic
doc.Pipeline().
    Filter(filterExpr).
    Sort("amount", false).
    Execute()
```

### Migration Path

```go
// Before: Manual data manipulation
filtered := filterData(data)
sorted := sortData(filtered)
doc := output.New().Table("Result", sorted).Build()

// After: Use pipeline
doc := output.New().
    Table("Result", data).
    Build().
    Pipeline().
    Filter(condition).
    Sort(column).
    Execute()
```

### Performance Improvements

Users would see performance benefits:
- Single pass through data instead of multiple parse/render cycles
- Optimized operations (e.g., push filter before sort)
- Parallel execution where applicable

### Clear Documentation

Would need to clearly document:
1. When to use byte transformers (format-specific styling)
2. When to use data transformers (data manipulation)
3. When to use pipeline (complex operations)

## Summary

The current transformation system is limited to post-render text manipulation. To enable real data operations, we need to:

1. **Short term**: Add data-aware transformers that operate before rendering
2. **Medium term**: Implement a proper data pipeline for complex operations
3. **Long term**: Optimize and extend with advanced features

This can be done with **zero breaking changes** - current users' code continues to work while new capabilities are added. The key is making the new features discoverable and providing clear guidance on when to use each approach.