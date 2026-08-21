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
