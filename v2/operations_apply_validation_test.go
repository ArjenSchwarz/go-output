package output

import (
	"context"
	"strings"
	"testing"
)

// applyValidationTestTable builds a small table with at least one record so
// that invalid operation configuration (nil predicate, nil calculation
// function, nil aggregate, negative limit) is actually exercised by Apply.
func applyValidationTestTable(t *testing.T) *TableContent {
	t.Helper()
	doc := New().
		Table("test", []Record{
			{"id": 1, "status": "active"},
			{"id": 2, "status": "inactive"},
		}, WithKeys("id", "status")).
		Build()
	return doc.GetContents()[0].(*TableContent)
}

// TestOperationsApplyInvalidConfig is a regression test for T-1502.
//
// Bug: the renderer path (applyContentTransformations) validates operations
// before applying them, but the public Apply methods are also callable
// directly and did not check their own Validate result. Direct calls with
// invalid configuration panicked (nil predicate, nil calculation function,
// nil aggregate function, negative limit slice bound) or silently misbehaved
// (invalid sort direction treated as ascending).
//
// Expected: Apply must return the operation's structured validation error
// instead of panicking or silently ignoring invalid configuration.
func TestOperationsApplyInvalidConfig(t *testing.T) {
	negativePos := -1

	cases := map[string]struct {
		op      Operation
		wantMsg string
	}{
		"filter with nil predicate": {
			op:      NewFilterOp(nil),
			wantMsg: "filter predicate function is required",
		},
		"limit with negative count": {
			op:      NewLimitOp(-1),
			wantMsg: "limit count must be non-negative",
		},
		"addColumn with nil function": {
			op:      NewAddColumnOp("x", nil, nil),
			wantMsg: "addColumn operation requires a calculation function",
		},
		"addColumn with empty name": {
			op:      NewAddColumnOp("", func(Record) any { return 1 }, nil),
			wantMsg: "non-empty column name",
		},
		"addColumn with negative position": {
			op:      NewAddColumnOp("x", func(Record) any { return 1 }, &negativePos),
			wantMsg: "position must be non-negative",
		},
		"groupBy with nil aggregate function": {
			op:      NewGroupByOp([]string{"status"}, map[string]AggregateFunc{"count": nil}),
			wantMsg: "aggregate function for 'count' is required and cannot be nil",
		},
		"groupBy without grouping columns": {
			op:      NewGroupByOp(nil, map[string]AggregateFunc{"count": CountAggregate()}),
			wantMsg: "at least one grouping column",
		},
		"sort with invalid direction": {
			// Column exists in the table, so without the Validate check this
			// silently sorts ascending instead of returning an error.
			op:      NewSortOp(SortKey{Column: "status", Direction: SortDirection(99)}),
			wantMsg: "invalid direction",
		},
		"sort without keys or comparator": {
			op:      NewSortOp(),
			wantMsg: "requires either sort keys or a custom comparator",
		},
		"sort with empty column name": {
			op:      NewSortOp(SortKey{Column: "", Direction: Ascending}),
			wantMsg: "empty column name",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			table := applyValidationTestTable(t)

			// Apply must not panic on invalid configuration.
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Apply panicked with %v — expected validation error", r)
				}
			}()

			_, err := tc.op.Apply(context.Background(), table)
			if err == nil {
				t.Fatal("expected validation error from Apply with invalid config, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("expected error containing %q, got %q", tc.wantMsg, err.Error())
			}
		})
	}
}

// TestOperationsApplyWithFormatInvalidConfig is a regression test for T-1502.
//
// ApplyWithFormat delegates to Apply for every operation, so it must surface
// the same validation errors when called directly with invalid configuration.
func TestOperationsApplyWithFormatInvalidConfig(t *testing.T) {
	cases := map[string]struct {
		op      Operation
		wantMsg string
	}{
		"filter with nil predicate": {
			op:      NewFilterOp(nil),
			wantMsg: "filter predicate function is required",
		},
		"limit with negative count": {
			op:      NewLimitOp(-1),
			wantMsg: "limit count must be non-negative",
		},
		"addColumn with nil function": {
			op:      NewAddColumnOp("x", nil, nil),
			wantMsg: "addColumn operation requires a calculation function",
		},
		"groupBy with nil aggregate function": {
			op:      NewGroupByOp([]string{"status"}, map[string]AggregateFunc{"count": nil}),
			wantMsg: "aggregate function for 'count' is required and cannot be nil",
		},
		"sort with invalid direction": {
			op:      NewSortOp(SortKey{Column: "status", Direction: SortDirection(99)}),
			wantMsg: "invalid direction",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			table := applyValidationTestTable(t)

			formatAware, ok := tc.op.(interface {
				ApplyWithFormat(ctx context.Context, content Content, format string) (Content, error)
			})
			if !ok {
				t.Fatalf("%s does not implement ApplyWithFormat", tc.op.Name())
			}

			// ApplyWithFormat must not panic on invalid configuration.
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ApplyWithFormat panicked with %v — expected validation error", r)
				}
			}()

			_, err := formatAware.ApplyWithFormat(context.Background(), table, "json")
			if err == nil {
				t.Fatal("expected validation error from ApplyWithFormat with invalid config, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("expected error containing %q, got %q", tc.wantMsg, err.Error())
			}
		})
	}
}

// TestOperationsApplyValidConfigStillWorks guards against the Validate wiring
// rejecting valid configuration: a well-formed operation must still apply
// successfully when called directly.
func TestOperationsApplyValidConfigStillWorks(t *testing.T) {
	ops := map[string]Operation{
		"filter":    NewFilterOp(func(r Record) bool { return r["status"] == "active" }),
		"sort":      NewSortOp(SortKey{Column: "id", Direction: Descending}),
		"limit":     NewLimitOp(1),
		"groupBy":   NewGroupByOp([]string{"status"}, map[string]AggregateFunc{"count": CountAggregate()}),
		"addColumn": NewAddColumnOp("calc", func(Record) any { return 1 }, nil),
	}

	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			table := applyValidationTestTable(t)

			result, err := op.Apply(context.Background(), table)
			if err != nil {
				t.Fatalf("Apply() with valid config failed: %v", err)
			}
			if result == nil {
				t.Fatal("Apply() with valid config returned nil content")
			}
		})
	}
}
