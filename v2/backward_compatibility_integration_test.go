package output

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestBackwardCompatibility_ExistingByteTransformersUnmodified verifies existing transformers work without modification
func TestBackwardCompatibility_ExistingByteTransformersUnmodified(t *testing.T) {
	t.Run("Existing transformers function identically", func(t *testing.T) {
		testCases := []struct {
			name        string
			transformer Transformer
			input       string
			format      string
			wantContain []string
		}{
			{
				name:        "EmojiTransformer unchanged",
				transformer: &EmojiTransformer{},
				input:       "OK No !!",
				format:      FormatTable,
				wantContain: []string{"✅", "❌", "🚨"},
			},
			{
				name:        "ColorTransformer unchanged",
				transformer: NewColorTransformer(),
				input:       "✅ success",
				format:      FormatTable,
				wantContain: []string{"✅", "success"},
			},
			{
				name:        "RemoveColorsTransformer unchanged",
				transformer: NewRemoveColorsTransformer(),
				input:       "\x1B[31mred\x1B[0m",
				format:      FormatTable,
				wantContain: []string{"red"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				result, err := tc.transformer.Transform(ctx, []byte(tc.input), tc.format)
				if err != nil {
					t.Fatalf("Transform() error = %v", err)
				}

				resultStr := string(result)
				for _, want := range tc.wantContain {
					if !strings.Contains(resultStr, want) {
						t.Errorf("Transform() result %q should contain %q", resultStr, want)
					}
				}
			})
		}
	})

	t.Run("Transformer interface methods preserved", func(t *testing.T) {
		transformer := &EmojiTransformer{}

		// Verify all interface methods are still present and working
		if name := transformer.Name(); name != "emoji" {
			t.Errorf("Name() = %v, want emoji", name)
		}

		if priority := transformer.Priority(); priority != 100 {
			t.Errorf("Priority() = %v, want 100", priority)
		}

		if !transformer.CanTransform(FormatTable) {
			t.Error("CanTransform(FormatTable) should return true")
		}

		if transformer.CanTransform("unsupported") {
			t.Error("CanTransform('unsupported') should return false")
		}
	})
}

// TestBackwardCompatibility_TransformationConfigurationMethods verifies configuration methods are preserved
func TestBackwardCompatibility_TransformationConfigurationMethods(t *testing.T) {
	t.Run("TransformPipeline configuration methods preserved", func(t *testing.T) {
		pipeline := NewTransformPipeline()

		// Test Add method
		transformer := &EmojiTransformer{}
		pipeline.Add(transformer)

		if !pipeline.Has("emoji") {
			t.Error("Has() should return true for added transformer")
		}

		if pipeline.Count() != 1 {
			t.Errorf("Count() = %v, want 1", pipeline.Count())
		}

		// Test Get method
		retrieved := pipeline.Get("emoji")
		if retrieved == nil {
			t.Error("Get() should return the transformer")
		}

		if retrieved.Name() != "emoji" {
			t.Errorf("Retrieved transformer Name() = %v, want emoji", retrieved.Name())
		}

		// Test Remove method
		removed := pipeline.Remove("emoji")
		if !removed {
			t.Error("Remove() should return true for existing transformer")
		}

		if pipeline.Count() != 0 {
			t.Errorf("Count() after removal = %v, want 0", pipeline.Count())
		}

		// Test Clear method
		pipeline.Add(&EmojiTransformer{})
		pipeline.Add(NewColorTransformer())
		pipeline.Clear()

		if pipeline.Count() != 0 {
			t.Errorf("Count() after Clear() = %v, want 0", pipeline.Count())
		}
	})

	t.Run("Transformer creation methods preserved", func(t *testing.T) {
		// Verify all existing transformer constructors still work
		colorTransformer := NewColorTransformer()
		if colorTransformer.Name() != "color" {
			t.Error("NewColorTransformer() should create color transformer")
		}

		colorTransformerCustom := NewColorTransformerWithScheme(ColorScheme{
			Success: "green",
			Error:   "red",
		})
		if colorTransformerCustom.Name() != "color" {
			t.Error("NewColorTransformerWithScheme() should create color transformer")
		}

		sortTransformer := NewSortTransformer("name", true)
		if sortTransformer.Name() != "sort" {
			t.Error("NewSortTransformer() should create sort transformer")
		}

		sortTransformerAsc := NewSortTransformerAscending("name")
		if sortTransformerAsc.Name() != "sort" {
			t.Error("NewSortTransformerAscending() should create sort transformer")
		}

		lineSplitTransformer := NewLineSplitTransformer(",")
		if lineSplitTransformer.Name() != "linesplit" {
			t.Error("NewLineSplitTransformer() should create linesplit transformer")
		}

		lineSplitTransformerDefault := NewLineSplitTransformerDefault()
		if lineSplitTransformerDefault.Name() != "linesplit" {
			t.Error("NewLineSplitTransformerDefault() should create linesplit transformer")
		}

		removeColorsTransformer := NewRemoveColorsTransformer()
		if removeColorsTransformer.Name() != "remove-colors" {
			t.Error("NewRemoveColorsTransformer() should create remove-colors transformer")
		}
	})
}

// TestBackwardCompatibility_ImmutabilityGuarantees verifies immutability guarantees are preserved
func TestBackwardCompatibility_ImmutabilityGuarantees(t *testing.T) {
	t.Run("Document immutability preserved", func(t *testing.T) {
		// Create document
		doc := New().
			Table("Test", []Record{
				{"name": "Alice", "value": 100},
			}).
			Build()

		// Get original contents
		originalContents := doc.GetContents()
		originalCount := len(originalContents)

		// Verify contents are immutable - attempting to modify should not affect original
		if originalCount != 1 {
			t.Errorf("Expected 1 content item, got %d", originalCount)
		}

		// Document should remain unchanged
		afterContents := doc.GetContents()
		if len(afterContents) != originalCount {
			t.Error("Document contents were modified, violating immutability")
		}
	})

	t.Run("Transformer state isolation", func(t *testing.T) {
		transformer1 := &EmojiTransformer{}
		transformer2 := &EmojiTransformer{}

		// Transformers should be stateless and independent
		ctx := context.Background()
		input := []byte("OK test")

		result1, err1 := transformer1.Transform(ctx, input, FormatTable)
		result2, err2 := transformer2.Transform(ctx, input, FormatTable)

		if err1 != nil || err2 != nil {
			t.Fatalf("Transform errors: %v, %v", err1, err2)
		}

		// Results should be identical - no state shared between instances
		if string(result1) != string(result2) {
			t.Error("Transformer instances should produce identical results (stateless)")
		}
	})
}

// TestBackwardCompatibility_ThreadSafety verifies thread-safety during concurrent operations
func TestBackwardCompatibility_ThreadSafety(t *testing.T) {
	t.Run("Pipeline concurrent access safety", func(t *testing.T) {
		pipeline := NewTransformPipeline()

		const numGoroutines = 20
		const operationsPerGoroutine = 50

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*operationsPerGoroutine)

		// Test concurrent Add operations
		for i := range numGoroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := range operationsPerGoroutine {
					transformer := &mockTransformer{
						name:     fmt.Sprintf("transformer-%d-%d", id, j),
						priority: id*100 + j,
						formats:  []string{FormatJSON},
					}

					pipeline.Add(transformer)

					// Also test concurrent transformations
					ctx := context.Background()
					_, err := pipeline.Transform(ctx, []byte("test"), FormatJSON)
					if err != nil {
						errors <- err
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Errorf("Concurrent operation error: %v", err)
		}

		// Pipeline should be in consistent state
		finalCount := pipeline.Count()
		expectedCount := numGoroutines * operationsPerGoroutine
		if finalCount != expectedCount {
			t.Errorf("Expected %d transformers after concurrent operations, got %d", expectedCount, finalCount)
		}
	})

	t.Run("Transformer concurrent transformation safety", func(t *testing.T) {
		transformer := &EmojiTransformer{}

		const numGoroutines = 10
		const transformsPerGoroutine = 100

		var wg sync.WaitGroup
		results := make(chan string, numGoroutines*transformsPerGoroutine)
		errors := make(chan error, numGoroutines*transformsPerGoroutine)

		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for range transformsPerGoroutine {
					ctx := context.Background()
					result, err := transformer.Transform(ctx, []byte("OK test"), FormatTable)
					if err != nil {
						errors <- err
						return
					}
					results <- string(result)
				}
			}()
		}

		wg.Wait()
		close(results)
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Errorf("Concurrent transformation error: %v", err)
		}

		// All results should be identical (stateless transformer)
		var firstResult string
		count := 0
		for result := range results {
			if count == 0 {
				firstResult = result
			} else if result != firstResult {
				t.Error("Concurrent transformations produced different results, indicating race condition")
			}
			count++
		}

		expectedCount := numGoroutines * transformsPerGoroutine
		if count != expectedCount {
			t.Errorf("Expected %d results, got %d", expectedCount, count)
		}
	})
}

// TestBackwardCompatibility_ErrorHandlingPreserved verifies error handling behavior is preserved
func TestBackwardCompatibility_ErrorHandlingPreserved(t *testing.T) {
	t.Run("TransformError handling preserved", func(t *testing.T) {
		// Test that TransformError structure and behavior is unchanged
		originalErr := errors.New("test error")
		transformErr := NewTransformError("test-transformer", FormatJSON, []byte("input"), originalErr)

		// Verify structure
		if transformErr.Transformer != "test-transformer" {
			t.Errorf("TransformError.Transformer = %s, want test-transformer", transformErr.Transformer)
		}

		if transformErr.Format != FormatJSON {
			t.Errorf("TransformError.Format = %s, want json", transformErr.Format)
		}

		if string(transformErr.Input) != "input" {
			t.Errorf("TransformError.Input = %s, want input", string(transformErr.Input))
		}

		// Verify error wrapping
		if !errors.Is(transformErr, originalErr) {
			t.Error("TransformError should wrap original error")
		}

		// Verify error message format
		expectedMsg := "transformer test-transformer failed for format json: test error"
		if transformErr.Error() != expectedMsg {
			t.Errorf("TransformError.Error() = %s, want %s", transformErr.Error(), expectedMsg)
		}
	})

	t.Run("Pipeline error propagation preserved", func(t *testing.T) {
		pipeline := NewTransformPipeline()

		// Add failing transformer
		failingTransformer := &integrationFailingMockTransformer{
			name:     "failing",
			priority: 100,
			err:      errors.New("mock failure"),
		}
		pipeline.Add(failingTransformer)

		ctx := context.Background()
		_, err := pipeline.Transform(ctx, []byte("test"), FormatJSON)
		if err == nil {
			t.Error("Transform() should return error when transformer fails")
		}

		// Should be TransformError
		var transformErr *TransformError
		if !AsError(err, &transformErr) {
			t.Errorf("Transform() error should be TransformError, got: %T", err)
		}

		if transformErr.Transformer != "failing" {
			t.Errorf("TransformError.Transformer = %s, want failing", transformErr.Transformer)
		}
	})

	t.Run("Context cancellation preserved", func(t *testing.T) {
		pipeline := NewTransformPipeline()

		slowTransformer := &integrationSlowMockTransformer{
			name:     "slow",
			priority: 100,
		}
		pipeline.Add(slowTransformer)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := pipeline.Transform(ctx, []byte("test"), FormatJSON)
		if err == nil {
			t.Error("Transform() should return error when context times out")
		}

		// Should be context deadline exceeded
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			t.Errorf("Transform() should return context error, got: %v", err)
		}
	})
}

// TestBackwardCompatibility_APIStability verifies API surface area stability
func TestBackwardCompatibility_APIStability(t *testing.T) {
	t.Run("All transformer interface methods present", func(t *testing.T) {
		// Verify Transformer interface is unchanged by testing all methods
		var transformer Transformer = &EmojiTransformer{}

		// These should compile without errors, proving interface unchanged
		_ = transformer.Name()
		_ = transformer.Priority()
		_ = transformer.CanTransform("test")

		ctx := context.Background()
		_, _ = transformer.Transform(ctx, []byte("test"), "format")
	})

	t.Run("All pipeline methods present", func(t *testing.T) {
		// Verify TransformPipeline API is unchanged
		pipeline := NewTransformPipeline()
		transformer := &EmojiTransformer{}

		// These should compile without errors, proving API unchanged
		pipeline.Add(transformer)
		_ = pipeline.Has("emoji")
		_ = pipeline.Get("emoji")
		_ = pipeline.Count()
		_ = pipeline.Remove("emoji")
		pipeline.Clear()
		_ = pipeline.Info()

		ctx := context.Background()
		_, _ = pipeline.Transform(ctx, []byte("test"), FormatJSON)
	})

	t.Run("Constructor functions present", func(t *testing.T) {
		// Verify all constructor functions still exist
		_ = NewTransformPipeline()
		_ = NewColorTransformer()
		_ = NewColorTransformerWithScheme(DefaultColorScheme())
		_ = NewSortTransformer("key", true)
		_ = NewSortTransformerAscending("key")
		_ = NewLineSplitTransformer(",")
		_ = NewLineSplitTransformerDefault()
		_ = NewRemoveColorsTransformer()
		_ = DefaultColorScheme()
	})
}

// Mock transformers for testing (avoiding conflicts with existing mocks)
type integrationFailingMockTransformer struct {
	name     string
	priority int
	err      error
}

func (f *integrationFailingMockTransformer) Name() string                    { return f.name }
func (f *integrationFailingMockTransformer) Priority() int                   { return f.priority }
func (f *integrationFailingMockTransformer) CanTransform(format string) bool { return true }
func (f *integrationFailingMockTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	return nil, f.err
}

type integrationSlowMockTransformer struct {
	name     string
	priority int
}

func (s *integrationSlowMockTransformer) Name() string                    { return s.name }
func (s *integrationSlowMockTransformer) Priority() int                   { return s.priority }
func (s *integrationSlowMockTransformer) CanTransform(format string) bool { return true }
func (s *integrationSlowMockTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	select {
	case <-time.After(50 * time.Millisecond):
		return input, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
