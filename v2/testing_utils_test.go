package output

import (
	"context"
	"fmt"
	"testing"
)

// statefulOperation is a test operation that maintains mutable state
type statefulOperation struct {
	name       string
	callCount  int
	modifyData bool
}

func (s *statefulOperation) Name() string {
	return s.name
}

func (s *statefulOperation) Validate() error {
	return nil
}

func (s *statefulOperation) Apply(ctx context.Context, content Content) (Content, error) {
	s.callCount++ // UNSAFE: Mutating state during Apply()

	// If this is a table, modify it differently based on call count
	if tc, ok := content.(*TableContent); ok {
		clone := tc.Clone().(*TableContent)
		if s.modifyData && s.callCount > 1 {
			// Non-deterministic: different results on different calls
			clone.records = append(clone.records, Record{"added": s.callCount})
		}
		return clone, nil
	}

	return content, nil
}

func (s *statefulOperation) CanOptimize(with Operation) bool {
	return false
}

// statelessOperation is a proper stateless operation
type statelessOperation struct {
	name   string
	suffix string
}

func (s *statelessOperation) Name() string {
	return s.name
}

func (s *statelessOperation) Validate() error {
	return nil
}

func (s *statelessOperation) Apply(ctx context.Context, content Content) (Content, error) {
	// Stateless: always produces same output for same input
	if tc, ok := content.(*TableContent); ok {
		clone := tc.Clone().(*TableContent)
		// Deterministic modification
		for i := range clone.records {
			if name, ok := clone.records[i]["name"]; ok {
				clone.records[i]["name"] = fmt.Sprintf("%v%s", name, s.suffix)
			}
		}
		return clone, nil
	}

	return content, nil
}

func (s *statelessOperation) CanOptimize(with Operation) bool {
	return false
}

// TestValidateStatelessOperation_DetectsNonDeterministic tests that the utility detects non-deterministic operations
func TestValidateStatelessOperation_DetectsNonDeterministic(t *testing.T) {
	tests := map[string]struct {
		createOp      func() Operation
		createContent func() Content
		expectError   bool
	}{
		"stateful operation with call counter": {
			createOp: func() Operation {
				return &statefulOperation{
					name:       "stateful-counter",
					modifyData: true,
				}
			},
			createContent: func() Content {
				return &TableContent{
					id:    "test",
					title: "Test",
					records: []Record{
						{"name": "Alice", "age": 30},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
				}
			},
			expectError: true,
		},
		"stateless operation with deterministic output": {
			createOp: func() Operation {
				return &statelessOperation{
					name:   "stateless-append",
					suffix: "-modified",
				}
			},
			createContent: func() Content {
				return &TableContent{
					id:    "test",
					title: "Test",
					records: []Record{
						{"name": "Alice", "age": 30},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
				}
			},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			op := tc.createOp()
			content := tc.createContent()

			// Call the testing utility
			err := ValidateStatelessOperation(t, op, content)

			if tc.expectError && err == nil {
				t.Error("Expected ValidateStatelessOperation to return error for non-deterministic operation")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected ValidateStatelessOperation to pass, got error: %v", err)
			}
		})
	}
}

// TestValidateStatelessOperation_PassesDeterministicOperations tests that deterministic operations pass validation
func TestValidateStatelessOperation_PassesDeterministicOperations(t *testing.T) {
	tests := map[string]struct {
		createOp      func() Operation
		createContent func() Content
	}{
		"filter operation": {
			createOp: func() Operation {
				return NewFilterOp(func(r Record) bool {
					return r["age"].(int) >= 30
				})
			},
			createContent: func() Content {
				return &TableContent{
					id:    "test",
					title: "Test",
					records: []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
				}
			},
		},
		"sort operation": {
			createOp: func() Operation {
				return NewSortOp(SortKey{Column: "age", Direction: Ascending})
			},
			createContent: func() Content {
				return &TableContent{
					id:    "test",
					title: "Test",
					records: []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
				}
			},
		},
		"limit operation": {
			createOp: func() Operation {
				return NewLimitOp(5)
			},
			createContent: func() Content {
				return &TableContent{
					id:    "test",
					title: "Test",
					records: []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			op := tc.createOp()
			content := tc.createContent()

			// Call the testing utility
			err := ValidateStatelessOperation(t, op, content)

			if err != nil {
				t.Errorf("Expected operation to be stateless, got error: %v", err)
			}
		})
	}
}

// TestValidateStatelessOperation_UsageExample demonstrates how to use the utility
func TestValidateStatelessOperation_UsageExample(t *testing.T) {
	// Example: Testing a custom operation for statelessness
	customOp := &statelessOperation{
		name:   "append-suffix",
		suffix: "-processed",
	}

	testData := &TableContent{
		id:    "example",
		title: "Example Data",
		records: []Record{
			{"name": "Alice", "age": 30},
			{"name": "Bob", "age": 25},
		},
		schema: &Schema{
			Fields:   []Field{{Name: "name"}, {Name: "age"}},
			keyOrder: []string{"name", "age"},
		},
	}

	// Validate that the operation is stateless
	if err := ValidateStatelessOperation(t, customOp, testData); err != nil {
		t.Fatalf("Custom operation failed statelessness check: %v", err)
	}

	// If we get here, the operation is stateless and safe for concurrent use
	t.Log("Custom operation passed statelessness validation")
}
