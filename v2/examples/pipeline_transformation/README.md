# Per-Content Transformations Example

This example demonstrates the Per-Content Transformations system in Go-Output v2.4+. This approach enables complex data operations like filtering, sorting, aggregation, and calculated fields to be attached directly to individual tables at creation time.

## What This Example Shows

### 1. Basic Per-Content Transformations
- **Filtering**: Select records based on conditions
- **Sorting**: Order data by column values
- **Limiting**: Restrict output to top N results
- **Attached to content**: Transformations defined where tables are created

### 2. Advanced Analytics
- **Calculated Fields**: Add computed columns based on existing data
- **Multi-column operations**: Complex transformations with multiple steps
- **Type-safe operations**: Proper type handling with Go type assertions
- **Lazy execution**: Transformations apply during rendering

### 3. Aggregation and Reporting
- **GroupBy operations**: Group records by column values
- **Built-in aggregates**: Sum, Count, Average, Min, Max
- **Custom aggregates**: User-defined aggregation functions
- **Multi-level grouping**: Complex analytical queries

### 4. Combined Transformations
- **Per-content transformations**: Structured data manipulation per table
- **Byte transformers**: Visual styling and formatting
- **Best practices**: When to use each transformation type

### 5. Multiple Tables with Different Transformations
- **Independent transformations**: Each table can have its own operations
- **Flexible document composition**: Mix tables with different analyses
- **Key advantage**: Not possible with the old Pipeline API

## Key Features Demonstrated

### Per-Content Transformation
```go
doc := output.New().
    Table("Top Sales", salesData,
        output.WithKeys("id", "salesperson", "amount"),
        output.WithTransformations(
            output.NewFilterOp(func(r output.Record) bool {
                return r["status"] == "completed" && r["amount"].(float64) > 20000
            }),
            output.NewAddColumnOp("commission", func(r output.Record) any {
                return r["amount"].(float64) * 0.05
            }, nil),
            output.NewSortOp(output.SortKey{Column: "amount", Direction: output.Descending}),
            output.NewLimitOp(10),
        ),
    ).
    Build()
```

### Aggregation and Analytics
```go
builder.Table("Regional Report", salesData,
    output.WithKeys("region", "total_sales", "avg_sale", "sale_count"),
    output.WithTransformations(
        output.NewGroupByOp(
            []string{"region"},
            map[string]output.AggregateFunc{
                "total_sales": output.SumAggregate("amount"),
                "avg_sale":    output.AverageAggregate("amount"),
                "sale_count":  output.CountAggregate(),
            },
        ),
        output.NewSortOp(output.SortKey{Column: "total_sales", Direction: output.Descending}),
    ),
)
```

### Multiple Tables with Different Transformations
```go
builder := output.New()

// Table 1: Top performers
builder.Table("Top Performers", salesData,
    output.WithTransformations(
        output.NewFilterOp(func(r output.Record) bool {
            return r["status"] == "completed" && r["amount"].(float64) > 30000
        }),
        output.NewSortOp(output.SortKey{Column: "amount", Direction: output.Descending}),
        output.NewLimitOp(5),
    ),
)

// Table 2: Pending sales (different transformations)
builder.Table("Pending Sales", salesData,
    output.WithTransformations(
        output.NewFilterOp(func(r output.Record) bool {
            return r["status"] == "pending"
        }),
        output.NewSortOp(output.SortKey{Column: "date", Direction: output.Descending}),
    ),
)

doc := builder.Build()
```

## Running the Example

```bash
cd examples/pipeline_transformation
go run main.go
```

## Expected Output

The example generates realistic sales data and demonstrates:

1. **Basic Transformations**: Top 10 high-value completed sales
2. **Analytics Report**: Detailed analysis with calculated fields
3. **Aggregation**: Regional and quarterly summaries
4. **Styled Output**: Combined data operations with visual formatting
5. **Multiple Tables**: Different transformations for different tables

## Migration from Pipeline API

The Pipeline API has been removed in v2.4.0. Use per-content transformations instead:

**Old (Pipeline API - REMOVED):**
```go
doc := builder.Build()
transformed, _ := doc.Pipeline().
    Filter(predicate).
    Sort(keys...).
    Execute()
```

**New (Per-Content Transformations):**
```go
doc := output.New().
    Table("data", records,
        output.WithTransformations(
            output.NewFilterOp(predicate),
            output.NewSortOp(keys...),
        ),
    ).
    Build()
```

## Benefits Over Pipeline API

- **Per-table control**: Each table can have unique transformations
- **Clearer intent**: Transformations defined where content is created
- **Better composability**: Build tables with transformations in one place
- **No global state**: Transformations belong to content items
- **Flexible documents**: Mix tables with and without transformations

## When to Use Per-Content Transformations vs Byte Transformers

**Use Per-Content Transformations For:**
- Filtering records based on data values
- Sorting by column values
- Performing aggregations
- Adding calculated fields
- Data structure manipulation

**Use Byte Transformers For:**
- Adding colors to output
- Converting text to emoji
- Format-specific styling
- Post-rendering text modifications

## Learn More

- [Pipeline Migration Guide](../../docs/PIPELINE_MIGRATION.md)
- [API Documentation](../../docs/API.md)
- [Migration Guide](../../docs/MIGRATION.md)
- [Other Examples](../)
