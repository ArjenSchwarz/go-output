# Per-Content Transformations Example

This example demonstrates how to use per-content transformations in go-output v2 to apply operations (filtering, sorting, limiting) to individual content items.

## Overview

Per-content transformations allow you to attach transformation operations directly to content items at creation time. This provides a natural model for real-world documents where each table may need different operations applied.

## Features Demonstrated

### 1. Basic Filter + Sort
Shows how to:
- Filter rows based on a predicate function
- Sort results by a specific column
- Chain multiple transformations together

### 2. Multiple Tables with Different Transformations
Demonstrates:
- Different transformations for different tables in the same document
- Using limit operations to get "top N" results
- Combining filtering and sorting

### 3. Dynamic Transformation Construction
Illustrates:
- Building transformation chains dynamically based on runtime conditions
- Conditional transformation application
- Variadic transformation passing

### 4. Error Handling
Shows:
- How transformation errors are reported during rendering
- Validation of operation configuration
- Proper error handling patterns

## Running the Example

```bash
go run main.go
```

## Key Concepts

### Transformation Execution
Transformations execute lazily during rendering, not during document building. This:
- Preserves the original data in the document
- Allows multiple renders with the same transformations
- Supports context-based cancellation

### Operation Types
- **FilterOp**: Filter rows based on a predicate function
- **SortOp**: Sort rows by one or more columns
- **LimitOp**: Limit the number of rows returned
- **GroupByOp**: Group rows and apply aggregations
- **AddColumnOp**: Add calculated columns

### Transformation Order
Transformations are applied in the order they are specified. This is important for operations like filter → sort → limit, where order affects the final result.

## See Also

- [Requirements](../../specs/per-content-transformations/requirements.md)
- [Design Document](../../specs/per-content-transformations/design.md)
- [API Documentation](../../docs/API.md)
