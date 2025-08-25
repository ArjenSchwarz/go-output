package output

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"
)

// SortDirection defines the direction for sorting operations
type SortDirection int

const (
	// Ascending sort direction
	Ascending SortDirection = iota
	// Descending sort direction
	Descending
)

// String returns the string representation of the sort direction
func (d SortDirection) String() string {
	switch d {
	case Ascending:
		return "asc"
	case Descending:
		return "desc"
	default:
		return unknownValue
	}
}

// SortKey defines a column and direction for sorting
type SortKey struct {
	Column    string        // Column name to sort by
	Direction SortDirection // Sort direction
}

// FilterOp implements filtering operation using predicate functions
type FilterOp struct {
	predicate func(Record) bool
}

// Name returns the operation name
func (o *FilterOp) Name() string {
	return "Filter"
}

// Apply filters table records based on the predicate
func (o *FilterOp) Apply(ctx context.Context, content Content) (Content, error) {
	// Type check
	tableContent, ok := content.(*TableContent)
	if !ok {
		return nil, NewValidationError("content_type", content.Type().String(),
			"filter operation requires table content")
	}

	// Clone the content to preserve immutability
	cloned := tableContent.Clone().(*TableContent)

	// Apply filter predicate
	filtered := make([]Record, 0)
	for _, record := range cloned.records {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if o.predicate(record) {
			filtered = append(filtered, record)
		}
	}

	cloned.records = filtered
	return cloned, nil
}

// CanOptimize returns true if this filter can be optimized with another operation
func (o *FilterOp) CanOptimize(with Operation) bool {
	// Filters can be combined with other filters
	_, isFilter := with.(*FilterOp)
	return isFilter
}

// Validate checks if the filter operation is valid
func (o *FilterOp) Validate() error {
	if o.predicate == nil {
		return NewValidationError("predicate", nil, "filter predicate function is required and cannot be nil")
	}
	return nil
}

// ApplyWithFormat applies the filter operation with format context
func (o *FilterOp) ApplyWithFormat(ctx context.Context, content Content, format string) (Content, error) {
	// Filter operations are format-agnostic, so delegate to Apply
	return o.Apply(ctx, content)
}

// CanTransform checks if filter operation applies to the given content and format
func (o *FilterOp) CanTransform(content Content, format string) bool {
	// Filter works with table content in any format
	return content.Type() == ContentTypeTable
}

// SortOp implements sorting operation using keys or custom comparators
type SortOp struct {
	keys       []SortKey             // Sort keys (column and direction)
	comparator func(a, b Record) int // Custom comparator function (optional)
}

// Name returns the operation name
func (o *SortOp) Name() string {
	return "Sort"
}

// Apply sorts table records based on keys or comparator
func (o *SortOp) Apply(ctx context.Context, content Content) (Content, error) {
	// Type check
	tableContent, ok := content.(*TableContent)
	if !ok {
		return nil, NewValidationError("content_type", content.Type().String(),
			"sort operation requires table content")
	}

	// Clone the content to preserve immutability
	cloned := tableContent.Clone().(*TableContent)

	// Validate that sort columns exist in the data (only if we have records and are using keys)
	if len(cloned.records) > 0 && o.comparator == nil && len(o.keys) > 0 {
		// Check first record for column existence
		// Note: Missing columns in other records will be treated as nil values during comparison
		firstRecord := cloned.records[0]
		availableColumns := make([]string, 0, len(firstRecord))
		for col := range firstRecord {
			availableColumns = append(availableColumns, col)
		}

		for _, key := range o.keys {
			if _, exists := firstRecord[key.Column]; !exists {
				return nil, NewValidationError("sort_column", key.Column,
					fmt.Sprintf("sort column '%s' does not exist in table data. Available columns: %v",
						key.Column, availableColumns))
			}
		}
	}

	// Create a copy of records for sorting
	records := make([]Record, len(cloned.records))
	copy(records, cloned.records)

	// Sort using either custom comparator or keys
	if o.comparator != nil {
		sort.Slice(records, func(i, j int) bool {
			// Check context cancellation during sorting
			select {
			case <-ctx.Done():
				return false // Stop sorting
			default:
			}
			return o.comparator(records[i], records[j]) < 0
		})
	} else {
		sort.SliceStable(records, func(i, j int) bool {
			// Check context cancellation during sorting
			select {
			case <-ctx.Done():
				return false // Stop sorting
			default:
			}
			return o.compareRecords(records[i], records[j]) < 0
		})
	}

	// Check if context was cancelled during sorting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	cloned.records = records
	return cloned, nil
}

// compareRecords compares two records based on sort keys
func (o *SortOp) compareRecords(a, b Record) int {
	for _, key := range o.keys {
		valA := a[key.Column]
		valB := b[key.Column]

		cmp := o.compareValues(valA, valB)
		if cmp != 0 {
			if key.Direction == Descending {
				return -cmp
			}
			return cmp
		}
	}
	return 0 // Equal
}

// compareValues compares two values of potentially different types
func (o *SortOp) compareValues(a, b any) int {
	// Handle nil values
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Use reflection to handle different types
	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)

	// If types don't match, compare as strings
	if aVal.Type() != bVal.Type() {
		aStr := fmt.Sprintf("%v", a)
		bStr := fmt.Sprintf("%v", b)
		if aStr < bStr {
			return -1
		} else if aStr > bStr {
			return 1
		}
		return 0
	}

	// Type-specific comparisons
	switch a := a.(type) {
	case string:
		b := b.(string)
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0

	case int:
		b := b.(int)
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0

	case int64:
		b := b.(int64)
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0

	case float64:
		b := b.(float64)
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0

	case bool:
		b := b.(bool)
		if !a && b {
			return -1
		} else if a && !b {
			return 1
		}
		return 0

	case time.Time:
		b := b.(time.Time)
		if a.Before(b) {
			return -1
		} else if a.After(b) {
			return 1
		}
		return 0

	default:
		// Fallback to string comparison
		aStr := fmt.Sprintf("%v", a)
		bStr := fmt.Sprintf("%v", b)
		if aStr < bStr {
			return -1
		} else if aStr > bStr {
			return 1
		}
		return 0
	}
}

// CanOptimize returns true if this sort can be optimized with another operation
func (o *SortOp) CanOptimize(with Operation) bool {
	// Sorts generally cannot be combined with other operations
	return false
}

// Validate checks if the sort operation is valid
func (o *SortOp) Validate() error {
	if len(o.keys) == 0 && o.comparator == nil {
		return NewValidationError("sort_configuration", nil, "sort operation requires either sort keys or a custom comparator function")
	}
	// Validate individual sort keys
	for i, key := range o.keys {
		if key.Column == "" {
			return NewValidationError("sort_key_column", key, fmt.Sprintf("sort key at index %d has empty column name", i))
		}
		if key.Direction != Ascending && key.Direction != Descending {
			return NewValidationError("sort_key_direction", key.Direction, fmt.Sprintf("sort key at index %d has invalid direction", i))
		}
	}
	return nil
}

// ApplyWithFormat applies the sort operation with format context
func (o *SortOp) ApplyWithFormat(ctx context.Context, content Content, format string) (Content, error) {
	// Sort operations are format-agnostic, so delegate to Apply
	return o.Apply(ctx, content)
}

// CanTransform checks if sort operation applies to the given content and format
func (o *SortOp) CanTransform(content Content, format string) bool {
	// Sort works with table content in any format
	return content.Type() == ContentTypeTable
}

// LimitOp implements limit operation to restrict the number of records
type LimitOp struct {
	count int // Number of records to keep
}

// Name returns the operation name
func (o *LimitOp) Name() string {
	return "Limit"
}

// Apply limits the number of records in the table
func (o *LimitOp) Apply(ctx context.Context, content Content) (Content, error) {
	// Type check
	tableContent, ok := content.(*TableContent)
	if !ok {
		return nil, NewValidationError("content_type", content.Type().String(),
			"limit operation requires table content")
	}

	// Clone the content to preserve immutability
	cloned := tableContent.Clone().(*TableContent)

	// Apply limit
	if o.count < len(cloned.records) {
		cloned.records = cloned.records[:o.count]
	}

	return cloned, nil
}

// CanOptimize returns true if this limit can be optimized with another operation
func (o *LimitOp) CanOptimize(with Operation) bool {
	// Limits generally cannot be combined with other operations
	return false
}

// Validate checks if the limit operation is valid
func (o *LimitOp) Validate() error {
	if o.count < 0 {
		return NewValidationError("count", o.count, "limit count must be non-negative (>= 0)")
	}
	return nil
}

// ApplyWithFormat applies the limit operation with format context
func (o *LimitOp) ApplyWithFormat(ctx context.Context, content Content, format string) (Content, error) {
	// Limit operations are format-agnostic, so delegate to Apply
	return o.Apply(ctx, content)
}

// CanTransform checks if limit operation applies to the given content and format
func (o *LimitOp) CanTransform(content Content, format string) bool {
	// Limit works with table content in any format
	return content.Type() == ContentTypeTable
}

// NewFilterOp creates a new filter operation with the given predicate
func NewFilterOp(predicate func(Record) bool) *FilterOp {
	return &FilterOp{
		predicate: predicate,
	}
}

// NewSortOp creates a new sort operation with the given sort keys
func NewSortOp(keys ...SortKey) *SortOp {
	return &SortOp{
		keys: keys,
	}
}

// NewSortOpWithComparator creates a new sort operation with a custom comparator
func NewSortOpWithComparator(comparator func(a, b Record) int) *SortOp {
	return &SortOp{
		comparator: comparator,
	}
}

// NewLimitOp creates a new limit operation with the given count
func NewLimitOp(count int) *LimitOp {
	return &LimitOp{
		count: count,
	}
}

// AggregateFunc defines the signature for aggregate functions
type AggregateFunc func(records []Record, field string) any

// GroupByOp implements grouping and aggregation operations
type GroupByOp struct {
	groupBy    []string                 // Columns to group by
	aggregates map[string]AggregateFunc // Map of result column name to aggregate function
}

// Name returns the operation name
func (o *GroupByOp) Name() string {
	return "GroupBy"
}

// Apply groups table records and applies aggregate functions
func (o *GroupByOp) Apply(ctx context.Context, content Content) (Content, error) {
	// Type check
	tableContent, ok := content.(*TableContent)
	if !ok {
		return nil, NewValidationError("content_type", content.Type().String(),
			"groupBy operation requires table content")
	}

	// Clone the content to preserve immutability
	cloned := tableContent.Clone().(*TableContent)

	// Group records by groupBy columns
	groups := make(map[string][]Record)
	groupKeys := make([]string, 0) // To preserve order

	for _, record := range cloned.records {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Create group key from groupBy columns
		groupKey := o.createGroupKey(record)

		if _, exists := groups[groupKey]; !exists {
			groupKeys = append(groupKeys, groupKey)
			groups[groupKey] = make([]Record, 0)
		}
		groups[groupKey] = append(groups[groupKey], record)
	}

	// Create result records with aggregated data
	resultRecords := make([]Record, 0, len(groups))

	for _, groupKey := range groupKeys {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		groupRecords := groups[groupKey]
		resultRecord := make(Record)

		// Add groupBy column values
		sampleRecord := groupRecords[0] // Use first record for group column values
		for _, column := range o.groupBy {
			resultRecord[column] = sampleRecord[column]
		}

		// Apply aggregate functions
		for aggName, aggFunc := range o.aggregates {
			// For count aggregate, field parameter is ignored
			field := ""
			if aggName != "count" {
				// Try to infer field from aggregate name patterns
				field = o.inferFieldFromAggregateName(aggName)
			}
			resultRecord[aggName] = aggFunc(groupRecords, field)
		}

		resultRecords = append(resultRecords, resultRecord)
	}

	// Create new schema with preserved key order
	newSchema := o.createAggregatedSchema(cloned.schema)

	// Update the cloned content
	cloned.records = resultRecords
	cloned.schema = newSchema

	return cloned, nil
}

// createGroupKey creates a string key from groupBy column values
func (o *GroupByOp) createGroupKey(record Record) string {
	key := ""
	for i, column := range o.groupBy {
		if i > 0 {
			key += "||" // Separator to avoid collisions
		}
		if val, exists := record[column]; exists {
			key += fmt.Sprintf("%v", val)
		} else {
			key += "<nil>"
		}
	}
	return key
}

// inferFieldFromAggregateName tries to infer field name from aggregate name
func (o *GroupByOp) inferFieldFromAggregateName(aggName string) string {
	// Common patterns: total_salary -> salary, sum_amount -> amount, etc.
	if len(aggName) > 4 && aggName[:4] == "sum_" {
		return aggName[4:]
	}
	if len(aggName) > 6 && aggName[:6] == "total_" {
		return aggName[6:]
	}
	if len(aggName) > 4 && aggName[:4] == "avg_" {
		return aggName[4:]
	}
	if len(aggName) > 8 && aggName[:8] == "average_" {
		return aggName[8:]
	}
	if len(aggName) > 4 && aggName[:4] == "min_" {
		return aggName[4:]
	}
	if len(aggName) > 4 && aggName[:4] == "max_" {
		return aggName[4:]
	}

	// Special case: if aggregate name ends with 's', try singular form
	// This handles cases like "names" -> "name"
	if len(aggName) > 1 && aggName[len(aggName)-1] == 's' {
		return aggName[:len(aggName)-1]
	}

	// Default: return the aggregate name itself as field name
	return aggName
}

// createAggregatedSchema creates a schema for aggregated results
func (o *GroupByOp) createAggregatedSchema(originalSchema *Schema) *Schema {
	// Build key order: groupBy columns first, then aggregate columns
	keyOrder := make([]string, 0, len(o.groupBy)+len(o.aggregates))

	// Add groupBy columns
	keyOrder = append(keyOrder, o.groupBy...)

	// Add aggregate columns (order may vary due to map iteration)
	for aggName := range o.aggregates {
		keyOrder = append(keyOrder, aggName)
	}

	// Create fields for the new schema
	fields := make([]Field, 0, len(keyOrder))

	// Add fields for groupBy columns (preserve from original schema if available)
	for _, column := range o.groupBy {
		if originalSchema != nil {
			if originalField := originalSchema.FindField(column); originalField != nil {
				fields = append(fields, *originalField)
				continue
			}
		}
		// Default field for groupBy column
		fields = append(fields, Field{Name: column})
	}

	// Add fields for aggregate columns
	for aggName := range o.aggregates {
		fields = append(fields, Field{Name: aggName})
	}

	return &Schema{
		Fields:   fields,
		keyOrder: keyOrder,
	}
}

// CanOptimize returns true if this groupBy can be optimized with another operation
func (o *GroupByOp) CanOptimize(with Operation) bool {
	// GroupBy generally cannot be combined with other operations
	return false
}

// Validate checks if the groupBy operation is valid
func (o *GroupByOp) Validate() error {
	if len(o.groupBy) == 0 {
		return NewValidationError("group_columns", o.groupBy, "groupBy operation requires at least one grouping column")
	}
	// Validate grouping columns
	for i, column := range o.groupBy {
		if column == "" {
			return NewValidationError("group_column", column, fmt.Sprintf("groupBy column at index %d cannot be empty", i))
		}
	}

	if len(o.aggregates) == 0 {
		return NewValidationError("aggregates", o.aggregates, "groupBy operation requires at least one aggregate function")
	}

	// Validate aggregate function names
	for aggName := range o.aggregates {
		if aggName == "" {
			return NewValidationError("aggregate_name", aggName, "aggregate function name cannot be empty")
		}
	}

	return nil
}

// ApplyWithFormat applies the groupBy operation with format context
func (o *GroupByOp) ApplyWithFormat(ctx context.Context, content Content, format string) (Content, error) {
	// GroupBy operations are format-agnostic, so delegate to Apply
	return o.Apply(ctx, content)
}

// CanTransform checks if groupBy operation applies to the given content and format
func (o *GroupByOp) CanTransform(content Content, format string) bool {
	// GroupBy works with table content in any format
	return content.Type() == ContentTypeTable
}

// Built-in aggregate functions

// CountAggregate returns an aggregate function that counts records
func CountAggregate() AggregateFunc {
	return func(records []Record, field string) any {
		return len(records)
	}
}

// SumAggregate returns an aggregate function that sums numeric values
func SumAggregate(field string) AggregateFunc {
	return func(records []Record, _ string) any {
		sum := float64(0)
		for _, record := range records {
			if val, exists := record[field]; exists {
				switch v := val.(type) {
				case int:
					sum += float64(v)
				case int64:
					sum += float64(v)
				case float64:
					sum += v
				case float32:
					sum += float64(v)
					// Ignore non-numeric values
				}
			}
		}
		return sum
	}
}

// AverageAggregate returns an aggregate function that calculates the average
func AverageAggregate(field string) AggregateFunc {
	return func(records []Record, _ string) any {
		sum := float64(0)
		count := 0
		for _, record := range records {
			if val, exists := record[field]; exists {
				switch v := val.(type) {
				case int:
					sum += float64(v)
					count++
				case int64:
					sum += float64(v)
					count++
				case float64:
					sum += v
					count++
				case float32:
					sum += float64(v)
					count++
					// Ignore non-numeric values
				}
			}
		}
		if count == 0 {
			return float64(0)
		}
		return sum / float64(count)
	}
}

// MinAggregate returns an aggregate function that finds the minimum value
func MinAggregate(field string) AggregateFunc {
	return func(records []Record, _ string) any {
		var min *float64
		for _, record := range records {
			if val, exists := record[field]; exists {
				var floatVal float64
				switch v := val.(type) {
				case int:
					floatVal = float64(v)
				case int64:
					floatVal = float64(v)
				case float64:
					floatVal = v
				case float32:
					floatVal = float64(v)
				default:
					continue // Ignore non-numeric values
				}

				if min == nil || floatVal < *min {
					min = &floatVal
				}
			}
		}
		if min == nil {
			return float64(0)
		}
		return *min
	}
}

// MaxAggregate returns an aggregate function that finds the maximum value
func MaxAggregate(field string) AggregateFunc {
	return func(records []Record, _ string) any {
		var max *float64
		for _, record := range records {
			if val, exists := record[field]; exists {
				var floatVal float64
				switch v := val.(type) {
				case int:
					floatVal = float64(v)
				case int64:
					floatVal = float64(v)
				case float64:
					floatVal = v
				case float32:
					floatVal = float64(v)
				default:
					continue // Ignore non-numeric values
				}

				if max == nil || floatVal > *max {
					max = &floatVal
				}
			}
		}
		if max == nil {
			return float64(0)
		}
		return *max
	}
}

// NewGroupByOp creates a new groupBy operation
func NewGroupByOp(groupBy []string, aggregates map[string]AggregateFunc) *GroupByOp {
	return &GroupByOp{
		groupBy:    groupBy,
		aggregates: aggregates,
	}
}

// AddColumnOp implements operation to add calculated fields
type AddColumnOp struct {
	name     string           // Name of the new column
	fn       func(Record) any // Function to calculate the field value
	position *int             // Optional position to insert the column (nil = append)
}

// Name returns the operation name
func (o *AddColumnOp) Name() string {
	return "AddColumn"
}

// Apply adds a calculated column to the table
func (o *AddColumnOp) Apply(ctx context.Context, content Content) (Content, error) {
	// Type check
	tableContent, ok := content.(*TableContent)
	if !ok {
		return nil, NewValidationError("content_type", content.Type().String(),
			"addColumn operation requires table content")
	}

	// Clone the content to preserve immutability
	cloned := tableContent.Clone().(*TableContent)

	// Add the new column to each record
	for i, record := range cloned.records {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calculate the new field value
		value := o.fn(record)

		// Add the new field to the record
		cloned.records[i][o.name] = value
	}

	// Update the schema with the new field
	newSchema := o.evolveSchema(cloned.schema)
	cloned.schema = newSchema

	return cloned, nil
}

// evolveSchema creates a new schema with the added field
func (o *AddColumnOp) evolveSchema(originalSchema *Schema) *Schema {
	if originalSchema == nil {
		// Create a simple schema if none exists
		return &Schema{
			Fields:   []Field{{Name: o.name}},
			keyOrder: []string{o.name},
		}
	}

	// Get the original key order - this is critical for preserving field order
	originalKeyOrder := originalSchema.GetKeyOrder()

	// Determine position for insertion based on the key order, not the fields array
	targetPosition := len(originalKeyOrder) // Default: append to end
	if o.position != nil {
		if *o.position >= 0 && *o.position <= len(originalKeyOrder) {
			targetPosition = *o.position
		}
		// If position is beyond bounds, it defaults to append
	}

	// Create new key order with the field inserted at the appropriate position
	newKeyOrder := make([]string, len(originalKeyOrder)+1)

	if targetPosition >= len(originalKeyOrder) {
		// Append to end
		copy(newKeyOrder, originalKeyOrder)
		newKeyOrder[len(originalKeyOrder)] = o.name
	} else {
		// Insert at position
		copy(newKeyOrder[:targetPosition], originalKeyOrder[:targetPosition])
		newKeyOrder[targetPosition] = o.name
		copy(newKeyOrder[targetPosition+1:], originalKeyOrder[targetPosition:])
	}

	// Create fields array that matches the new key order
	newFields := make([]Field, len(newKeyOrder))

	// Copy existing fields, preserving their definitions
	for i, key := range newKeyOrder {
		if key == o.name {
			// This is our new field
			newFields[i] = Field{Name: o.name}
		} else {
			// Find the existing field definition
			if existingField := originalSchema.FindField(key); existingField != nil {
				newFields[i] = *existingField
			} else {
				// Fallback: create a basic field
				newFields[i] = Field{Name: key}
			}
		}
	}

	return &Schema{
		Fields:   newFields,
		keyOrder: newKeyOrder,
	}
}

// CanOptimize returns true if this addColumn can be optimized with another operation
func (o *AddColumnOp) CanOptimize(with Operation) bool {
	// AddColumn generally cannot be combined with other operations
	return false
}

// Validate checks if the addColumn operation is valid
func (o *AddColumnOp) Validate() error {
	if o.name == "" {
		return NewValidationError("column_name", o.name, "addColumn operation requires a non-empty column name")
	}
	if o.fn == nil {
		return NewValidationError("calculation_function", nil, "addColumn operation requires a calculation function and cannot be nil")
	}
	if o.position != nil && *o.position < 0 {
		return NewValidationError("column_position", *o.position, "addColumn position must be non-negative (>= 0)")
	}
	return nil
}

// ApplyWithFormat applies the addColumn operation with format context
func (o *AddColumnOp) ApplyWithFormat(ctx context.Context, content Content, format string) (Content, error) {
	// AddColumn operations are format-agnostic, so delegate to Apply
	return o.Apply(ctx, content)
}

// CanTransform checks if addColumn operation applies to the given content and format
func (o *AddColumnOp) CanTransform(content Content, format string) bool {
	// AddColumn works with table content in any format
	return content.Type() == ContentTypeTable
}

// NewAddColumnOp creates a new addColumn operation
func NewAddColumnOp(name string, fn func(Record) any, position *int) *AddColumnOp {
	return &AddColumnOp{
		name:     name,
		fn:       fn,
		position: position,
	}
}
