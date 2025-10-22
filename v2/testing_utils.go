package output

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

// ValidateStatelessOperation validates that an operation is stateless by applying it twice
// to cloned content and comparing the results using reflect.DeepEqual().
//
// This utility helps detect non-deterministic operations that might cause issues in
// concurrent environments. An operation is considered stateless if:
//   - It produces the same output when given the same input
//   - It doesn't modify shared mutable state
//   - It doesn't depend on external factors like time or random numbers
//
// Limitations:
//   - Only detects output non-determinism, not hidden state mutations
//   - Cannot detect external side effects (file writes, network calls)
//   - Cannot detect non-deterministic operations that happen to match twice
//
// Usage Example:
//
//	func TestMyOperation_IsStateless(t *testing.T) {
//	    op := NewMyOperation(config)
//	    testContent := &TableContent{...}
//
//	    if err := ValidateStatelessOperation(t, op, testContent); err != nil {
//	        t.Fatalf("Operation is not stateless: %v", err)
//	    }
//	}
//
// Thread Safety:
// Operations that pass this validation are safe for concurrent use with per-content
// transformations, as they don't maintain mutable state that could cause data races.
func ValidateStatelessOperation(t *testing.T, op Operation, testContent Content) error {
	t.Helper()

	if op == nil {
		return fmt.Errorf("operation is nil")
	}
	if testContent == nil {
		return fmt.Errorf("test content is nil")
	}

	ctx := context.Background()

	// Clone the content twice to ensure independence
	clone1 := testContent.Clone()
	clone2 := testContent.Clone()

	// Apply operation to first clone
	result1, err1 := op.Apply(ctx, clone1)
	if err1 != nil {
		return fmt.Errorf("first application failed: %w", err1)
	}

	// Apply operation to second clone
	result2, err2 := op.Apply(ctx, clone2)
	if err2 != nil {
		return fmt.Errorf("second application failed: %w", err2)
	}

	// Compare results using deep equality
	if !reflect.DeepEqual(result1, result2) {
		return fmt.Errorf(
			"operation %q is not stateless: produced different results on repeated calls\n"+
				"This indicates the operation maintains mutable state or depends on external factors.\n"+
				"First result: %+v\n"+
				"Second result: %+v",
			op.Name(),
			result1,
			result2,
		)
	}

	return nil
}
