package output

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Regression tests for T-1515: AsError and IsCancelled have weaker semantics
// than stdlib errors.As/errors.Is, and the error types' Error() methods
// iterate context maps in random order, producing non-deterministic messages.

// asHookError implements the As(any) bool hook that errors.As honors.
// A hand-rolled unwrap loop that only follows Unwrap() error ignores it.
type asHookError struct {
	code string
}

func (e *asHookError) Error() string {
	return "as hook error: " + e.code
}

func (e *asHookError) As(target any) bool {
	if t, ok := target.(**asHookTarget); ok {
		*t = &asHookTarget{code: e.code}
		return true
	}
	return false
}

// asHookTarget is the type asHookError converts itself into via its As hook.
type asHookTarget struct {
	code string
}

func (e *asHookTarget) Error() string {
	return "as hook target: " + e.code
}

func TestIsCancelledWrappedContextErrors(t *testing.T) {
	tests := map[string]struct {
		err      error
		expected bool
	}{"wrapped context.Canceled": {
		// fmt.Errorf("%w", ctx.Err()) is how cancellations surface
		// throughout the render path; == comparison misses it.
		err:      fmt.Errorf("render json: %w", context.Canceled),
		expected: true,
	}, "wrapped context.DeadlineExceeded": {
		err:      fmt.Errorf("render json: %w", context.DeadlineExceeded),
		expected: true,
	}, "deeply wrapped context.Canceled": {
		err:      fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", context.Canceled)),
		expected: true,
	}, "ContextError wrapping context.Canceled": {
		// ContextError is not inherently a cancellation, but its cause
		// chain must be traversed.
		err:      NewContextError("render", context.Canceled),
		expected: true,
	}, "ContextError wrapping a non-cancellation": {
		err:      NewContextError("render", errors.New("disk full")),
		expected: false,
	}, "multi-error tree containing context.Canceled": {
		err:      &MultiWriteError{Errors: []error{errors.New("disk full"), context.Canceled}},
		expected: true,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := IsCancelled(tt.err); got != tt.expected {
				t.Errorf("IsCancelled(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestAsErrorMultiErrorTree(t *testing.T) {
	// MultiWriteError implements Unwrap() []error, not Unwrap() error.
	// errors.As traverses the whole tree; a manual loop following only
	// Unwrap() error finds nothing inside it.
	writerErr := NewWriterError("FileWriter", "json", errors.New("disk full"))
	multiErr := &MultiWriteError{Errors: []error{
		errors.New("unrelated failure"),
		writerErr,
	}}

	var target *WriterError
	if !AsError(error(multiErr), &target) {
		t.Fatalf("AsError should find *WriterError inside a MultiWriteError (Unwrap() []error) tree")
	}
	if target != writerErr {
		t.Errorf("AsError returned %v, want the original *WriterError instance", target)
	}

	// The same tree wrapped once more via fmt.Errorf must also be traversed.
	wrapped := fmt.Errorf("write phase: %w", multiErr)
	target = nil
	if !AsError(wrapped, &target) {
		t.Fatalf("AsError should find *WriterError inside a wrapped MultiWriteError tree")
	}
	if target != writerErr {
		t.Errorf("AsError returned %v, want the original *WriterError instance", target)
	}
}

func TestAsErrorAsMethodHook(t *testing.T) {
	err := fmt.Errorf("wrapper: %w", &asHookError{code: "E42"})

	var target *asHookTarget
	if !AsError(err, &target) {
		t.Fatalf("AsError should honor the As(any) bool hook like errors.As does")
	}
	if target == nil || target.code != "E42" {
		t.Errorf("AsError As-hook conversion produced %+v, want code E42", target)
	}
}

// deterministicContext returns a context map with enough keys that an
// unsorted map iteration has a negligible (1/8! ≈ 0.0025%) chance of
// accidentally producing the sorted order.
func deterministicContext() map[string]any {
	return map[string]any{
		"alpha": 1, "bravo": 2, "charlie": 3, "delta": 4,
		"echo": 5, "foxtrot": 6, "golf": 7, "hotel": 8,
	}
}

const sortedContextPairs = "alpha=1, bravo=2, charlie=3, delta=4, echo=5, foxtrot=6, golf=7, hotel=8"

func TestErrorMessagesDeterministicContext(t *testing.T) {
	tests := map[string]struct {
		err  error
		want string
	}{"RenderError": {
		err: &RenderError{
			Format:  "json",
			Context: deterministicContext(),
			Cause:   errors.New("boom"),
		},
		want: "context=[" + sortedContextPairs + "]",
	}, "WriterError": {
		err: &WriterError{
			Writer:  "FileWriter",
			Format:  "json",
			Context: deterministicContext(),
			Cause:   errors.New("boom"),
		},
		want: "context=[" + sortedContextPairs + "]",
	}, "ContextError": {
		err: &ContextError{
			Operation: "render",
			Context:   deterministicContext(),
			Cause:     errors.New("boom"),
		},
		want: "context: " + sortedContextPairs,
	}, "PipelineError operation context": {
		err: &PipelineError{
			Operation: "Filter",
			Stage:     1,
			Context:   deterministicContext(),
			Cause:     errors.New("boom"),
		},
		want: "operation_context=[" + sortedContextPairs + "]",
	}, "PipelineError pipeline context": {
		err: &PipelineError{
			Operation:   "Filter",
			Stage:       1,
			PipelineCtx: deterministicContext(),
			Cause:       errors.New("boom"),
		},
		want: "pipeline_context=[" + sortedContextPairs + "]",
	}, "MultiError single error": {
		err: &MultiError{
			Operation: "render",
			Errors:    []error{errors.New("boom")},
			Context:   deterministicContext(),
		},
		want: "[" + sortedContextPairs + "]",
	}, "MultiError multiple errors": {
		err: &MultiError{
			Operation: "render",
			Errors:    []error{errors.New("boom"), errors.New("bang")},
			Context:   deterministicContext(),
		},
		want: "Context: " + sortedContextPairs,
	}, "StructuredError": {
		// Already sorted before this fix; must stay sorted.
		err: &StructuredError{
			Code:    "TEST_001",
			Context: deterministicContext(),
		},
		want: "context=[" + sortedContextPairs + "]",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.err.Error()
			if !strings.Contains(got, tt.want) {
				t.Errorf("Error() = %q, want it to contain sorted context %q", got, tt.want)
			}
		})
	}
}

func TestMultiErrorSourceDetailsDeterministic(t *testing.T) {
	multiErr := NewMultiError("render")
	multiErr.AddWithSource(errors.New("boom"), "writer", deterministicContext())
	multiErr.Add(errors.New("bang")) // force the multi-error branch

	want := "[component=writer, " + sortedContextPairs + "]"
	got := multiErr.Error()
	if !strings.Contains(got, want) {
		t.Errorf("Error() = %q, want it to contain sorted source details %q", got, want)
	}
}
