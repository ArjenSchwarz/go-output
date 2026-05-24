package output

import (
	"context"
	"strings"
	"testing"
)

// TestOperationsApplyNilContent is a regression test for T-1142.
//
// Bug: FilterOp, SortOp, LimitOp, GroupByOp, and AddColumnOp all called
// content.Type().String() when building the validation error after a failed
// type assertion. Passing nil content makes the type assertion fail, so the
// error path then called Type() on a nil Content interface and panicked.
//
// Expected: Apply must return a normal validation error (not panic) when
// content is nil, matching the behaviour for other invalid content.
func TestOperationsApplyNilContent(t *testing.T) {
	type opCase struct {
		op      Operation
		wantMsg string
	}

	cases := map[string]opCase{
		"filter": {
			op:      &FilterOp{predicate: func(Record) bool { return true }},
			wantMsg: "filter operation requires table content",
		},
		"sort": {
			op:      &SortOp{keys: []SortKey{{Column: "id", Direction: Ascending}}},
			wantMsg: "sort operation requires table content",
		},
		"limit": {
			op:      &LimitOp{count: 5},
			wantMsg: "limit operation requires table content",
		},
		"groupBy": {
			op:      &GroupByOp{groupBy: []string{"id"}, aggregates: map[string]AggregateFunc{"count": CountAggregate()}},
			wantMsg: "groupBy operation requires table content",
		},
		"addColumn": {
			op:      &AddColumnOp{name: "calc", fn: func(Record) any { return 1 }},
			wantMsg: "addColumn operation requires table content",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Apply must not panic on nil content.
			_, err := tc.op.Apply(context.Background(), nil)
			if err == nil {
				t.Fatalf("expected validation error when applying %s op to nil content, got nil", name)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("expected error containing %q, got %q", tc.wantMsg, err.Error())
			}
		})
	}
}

// TestOperationsCanTransformNilContent is a regression test for T-1142.
//
// Bug: every operation's CanTransform called content.Type() directly, which
// panics when content is nil.
//
// Expected: CanTransform must return false for nil content instead of
// panicking.
func TestOperationsCanTransformNilContent(t *testing.T) {
	ops := map[string]Operation{
		"filter":    &FilterOp{predicate: func(Record) bool { return true }},
		"sort":      &SortOp{keys: []SortKey{{Column: "id", Direction: Ascending}}},
		"limit":     &LimitOp{count: 5},
		"groupBy":   &GroupByOp{groupBy: []string{"id"}, aggregates: map[string]AggregateFunc{"count": CountAggregate()}},
		"addColumn": &AddColumnOp{name: "calc", fn: func(Record) any { return 1 }},
	}

	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			transformer, ok := op.(interface {
				CanTransform(content Content, format string) bool
			})
			if !ok {
				t.Fatalf("%s op does not implement CanTransform", name)
			}
			// CanTransform must not panic and must report it cannot transform
			// nil content.
			if transformer.CanTransform(nil, "json") {
				t.Errorf("expected CanTransform to return false for nil content on %s op", name)
			}
		})
	}
}
