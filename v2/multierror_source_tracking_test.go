package output

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Regression tests for T-1508: MultiError source tracking panics on
// non-comparable errors.
//
// AddWithSource stored errors as keys in SourceMap (map[error]ErrorSource),
// and Error() looked errors up in that map. Both operations hash the error
// interface value, which panics with "hash of unhashable type" when the
// error's dynamic type is not comparable (e.g. a value-receiver struct
// containing a slice or map field). This is public API surface and also
// affects render error aggregation when a custom renderer, transformer, or
// writer returns such an error.

// sliceFieldError is a non-comparable error type: a value-receiver struct
// containing a slice field.
type sliceFieldError struct {
	parts []string
}

func (e sliceFieldError) Error() string { return strings.Join(e.parts, ": ") }

// mapFieldError is a non-comparable error type: a value-receiver struct
// containing a map field.
type mapFieldError struct {
	fields map[string]string
}

func (e mapFieldError) Error() string { return fmt.Sprintf("map error with %d fields", len(e.fields)) }

// dynamicPayloadError is statically comparable (its only field is an
// interface), but hashing it panics at runtime when the interface holds a
// non-comparable value. reflect.Type.Comparable reports true for this type;
// only a dynamic check (reflect.Value.Comparable) catches it.
type dynamicPayloadError struct {
	payload any
}

func (e dynamicPayloadError) Error() string { return fmt.Sprintf("payload: %v", e.payload) }

// TestMultiErrorAddWithSourceNonComparable verifies that AddWithSource does
// not panic for valid error values whose dynamic type cannot be used as a
// map key, and that their source metadata still appears in Error() output.
//
// Expected: no panic; source info rendered for every error.
// Actual (before fix): panic "hash of unhashable type" on AddWithSource.
func TestMultiErrorAddWithSourceNonComparable(t *testing.T) {
	tests := map[string]struct {
		err error
	}{
		"struct with slice field": {
			err: sliceFieldError{parts: []string{"render", "failed"}},
		},
		"struct with map field": {
			err: mapFieldError{fields: map[string]string{"key": "value"}},
		},
		"comparable struct holding non-comparable payload": {
			err: dynamicPayloadError{payload: []int{1, 2, 3}},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			multiErr := NewMultiError("render")
			multiErr.AddWithSource(tt.err, "renderer", map[string]any{
				"format": "json",
			})
			// Add a second error so Error() takes the multi-error path,
			// which performs the source lookup.
			multiErr.AddWithSource(errors.New("second failure"), "writer", map[string]any{
				"format": "csv",
			})

			got := multiErr.Error()

			if !strings.Contains(got, tt.err.Error()) {
				t.Errorf("Error() = %q, want it to contain %q", got, tt.err.Error())
			}
			if !strings.Contains(got, "component=renderer") || !strings.Contains(got, "format=json") {
				t.Errorf("Error() = %q, want source info component=renderer format=json for non-comparable error", got)
			}
			if !strings.Contains(got, "component=writer") {
				t.Errorf("Error() = %q, want source info component=writer for second error", got)
			}
			if len(multiErr.Errors) != 2 {
				t.Errorf("len(Errors) = %d, want 2", len(multiErr.Errors))
			}
		})
	}
}

// TestMultiErrorErrorFormattingNonComparableViaAdd verifies that Error()
// does not panic when a non-comparable error was added via plain Add().
// NewMultiError initializes SourceMap, and since Go 1.12 map lookups hash
// the key even on empty maps, so formatting alone panicked before the fix.
//
// Expected: no panic; both errors formatted.
// Actual (before fix): panic "hash of unhashable type" in Error().
func TestMultiErrorErrorFormattingNonComparableViaAdd(t *testing.T) {
	multiErr := NewMultiError("render")
	multiErr.Add(sliceFieldError{parts: []string{"boom"}})
	multiErr.Add(errors.New("other failure"))

	got := multiErr.Error()

	if !strings.Contains(got, "boom") {
		t.Errorf("Error() = %q, want it to contain %q", got, "boom")
	}
	if !strings.Contains(got, "other failure") {
		t.Errorf("Error() = %q, want it to contain %q", got, "other failure")
	}
}

// TestMultiErrorSourceMapCompatibility verifies the exported SourceMap field
// keeps its existing behaviour for comparable errors: AddWithSource still
// populates it, and entries placed directly into SourceMap by callers are
// still rendered by Error().
func TestMultiErrorSourceMapCompatibility(t *testing.T) {
	t.Run("AddWithSource populates SourceMap for comparable errors", func(t *testing.T) {
		multiErr := NewMultiError("render")
		err := errors.New("comparable failure")
		multiErr.AddWithSource(err, "renderer", map[string]any{"format": "json"})

		source, ok := multiErr.SourceMap[err]
		if !ok {
			t.Fatalf("SourceMap missing entry for comparable error added via AddWithSource")
		}
		if source.Component != "renderer" {
			t.Errorf("SourceMap[err].Component = %q, want %q", source.Component, "renderer")
		}
	})

	t.Run("directly populated SourceMap entries are rendered", func(t *testing.T) {
		errA := errors.New("first failure")
		errB := errors.New("second failure")
		multiErr := &MultiError{
			Operation: "render",
			Errors:    []error{errA, errB},
			SourceMap: map[error]ErrorSource{
				errA: {Component: "renderer", Details: map[string]any{"format": "json"}},
			},
		}

		got := multiErr.Error()
		if !strings.Contains(got, "component=renderer") || !strings.Contains(got, "format=json") {
			t.Errorf("Error() = %q, want source info from directly populated SourceMap", got)
		}
	})
}
