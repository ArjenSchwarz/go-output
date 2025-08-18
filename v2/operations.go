package output

import (
	"context"
	"errors"
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
		return "unknown"
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
		return nil, errors.New("filter requires table content")
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
		return errors.New("filter predicate is required")
	}
	return nil
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
		return nil, errors.New("sort requires table content")
	}

	// Clone the content to preserve immutability
	cloned := tableContent.Clone().(*TableContent)

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
		return errors.New("sort requires keys or comparator")
	}
	return nil
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
		return nil, errors.New("limit requires table content")
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
		return errors.New("limit count must be non-negative")
	}
	return nil
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
