package output

import (
	"context"
)

// Operation represents a pipeline operation
type Operation interface {
	Name() string
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
