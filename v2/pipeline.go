package output

import (
	"context"
)

// Operation represents a pipeline operation
type Operation interface {
	Name() string
	// Apply transforms the given content. The returned Content must be
	// non-nil when the error is nil; a (nil, nil) result is rejected as a
	// transformation error during rendering (T-1601).
	Apply(ctx context.Context, content Content) (Content, error)
	CanOptimize(with Operation) bool
	Validate() error
}

// FormatAwareOperation extends Operation with format awareness
type FormatAwareOperation interface {
	Operation

	// ApplyWithFormat applies the operation with format context
	ApplyWithFormat(ctx context.Context, content Content, format string) (Content, error)

	// CanTransform checks if this operation applies to the given content and format
	CanTransform(content Content, format string) bool
}

// cloneOperations returns a fresh, non-nil slice holding the non-nil entries
// of ops. Every operation slice that crosses the public API boundary passes
// through it: the transformation options copy caller input on the way in and
// GetTransformations copies content state on the way out, so neither side can
// alias the other's backing array and built documents stay immutable
// (T-1378). Nil entries are dropped because a nil Operation panics when its
// methods are called during rendering (T-1208).
func cloneOperations(ops []Operation) []Operation {
	cloned := make([]Operation, 0, len(ops))
	for _, op := range ops {
		if op == nil {
			continue
		}
		cloned = append(cloned, op)
	}
	return cloned
}
