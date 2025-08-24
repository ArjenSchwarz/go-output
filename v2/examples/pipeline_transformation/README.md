# Data Transformation Pipeline Example

This example demonstrates the powerful Data Transformation Pipeline system introduced in Go-Output v2. The pipeline enables complex data operations like filtering, sorting, aggregation, and calculated fields directly on structured data before rendering.

## What This Example Shows

### 1. Basic Pipeline Operations
- **Filtering**: Select records based on conditions
- **Sorting**: Order data by column values
- **Limiting**: Restrict output to top N results
- **Method chaining**: Fluent API for readable code

### 2. Advanced Analytics
- **Calculated Fields**: Add computed columns based on existing data
- **Multi-column operations**: Complex transformations with multiple steps
- **Type-safe operations**: Proper type handling with Go type assertions
- **Performance optimization**: Automatic operation reordering

### 3. Aggregation and Reporting
- **GroupBy operations**: Group records by column values
- **Built-in aggregates**: Sum, Count, Average, Min, Max
- **Custom aggregates**: User-defined aggregation functions
- **Multi-level grouping**: Complex analytical queries

### 4. Combined Transformations
- **Data pipeline**: Structured data manipulation
- **Byte transformers**: Visual styling and formatting
- **Best practices**: When to use each transformation type

### 5. Error Handling and Performance
- **Pipeline errors**: Detailed error context and debugging
- **Performance tracking**: Built-in statistics collection
- **Resource management**: Safe operation with limits

## Key Features Demonstrated

### Data-Level Operations
```go
transformedDoc := doc.Pipeline().
    Filter(func(r output.Record) bool {
        return r["status"] == "completed" && r["amount"].(float64) > 20000
    }).
    AddColumn("commission", func(r output.Record) any {
        return r["amount"].(float64) * 0.05
    }).
    SortBy("amount", output.Descending).
    Limit(10).
    Execute()
```

### Aggregation and Analytics
```go
reportDoc := doc.Pipeline().
    GroupBy(
        []string{"region"},
        map[string]output.AggregateFunc{
            "total_sales": output.SumAggregate("amount"),
            "avg_sale":    output.AverageAggregate("amount"),
            "sale_count":  output.CountAggregate,
        },
    ).
    SortBy("total_sales", output.Descending).
    Execute()
```

### Performance Optimization
The pipeline automatically optimizes operations for better performance:
- Filters are applied first to reduce dataset size
- Sort operations work on smaller datasets
- Limits are applied last to get top N of final results

## Running the Example

```bash
cd examples/pipeline_transformation
go run main.go
```

## Expected Output

The example generates realistic sales data and demonstrates:

1. **Filtered Results**: Shows top 10 high-value completed sales
2. **Analytics Report**: Detailed analysis with calculated fields
3. **Regional Summary**: Aggregated statistics by region
4. **Styled Output**: Combined data operations with visual formatting
5. **Performance Stats**: Detailed timing and processing metrics

## Migration Benefits

This example highlights the advantages of the pipeline system over manual data manipulation:

- **50% less code** compared to manual operations
- **Type-safe operations** with clear error messages
- **Format-agnostic** - works with all output formats
- **Automatic optimization** for better performance
- **Built-in error handling** with detailed context
- **Immutable transformations** preserving original data

## When to Use Pipeline vs Byte Transformers

**Use Pipeline For:**
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

- [API Documentation](../../docs/API.md#data-transformation-pipeline-system)
- [Migration Guide](../../docs/MIGRATION.md#data-transformation-pipeline-migration)
- [Other Examples](../)