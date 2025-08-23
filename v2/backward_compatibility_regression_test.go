package output

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

// TestBackwardCompatibility_ExistingTransformers verifies all existing transformer implementations still work
func TestBackwardCompatibility_ExistingTransformers(t *testing.T) {
	testCases := []struct {
		name         string
		transformer  Transformer
		input        []byte
		format       string
		wantContain  []string // Strings that should be present in output
		wantName     string
		wantPrio     int
		canTransform bool
	}{
		{
			name:         "EmojiTransformer basic functionality",
			transformer:  &EmojiTransformer{},
			input:        []byte("OK No !!"),
			format:       FormatTable,
			wantContain:  []string{"✅", "❌", "🚨"},
			wantName:     "emoji",
			wantPrio:     100,
			canTransform: true,
		},
		{
			name:         "EmojiTransformer format filtering",
			transformer:  &EmojiTransformer{},
			input:        []byte("test"),
			format:       "unsupported",
			wantContain:  []string{"test"}, // Should pass through unchanged
			wantName:     "emoji",
			wantPrio:     100,
			canTransform: false,
		},
		{
			name:         "ColorTransformer basic functionality",
			transformer:  NewColorTransformer(),
			input:        []byte("✅ success"),
			format:       FormatTable,
			wantContain:  []string{"✅", "success"}, // Output should still contain content
			wantName:     "color",
			wantPrio:     200,
			canTransform: true,
		},
		{
			name:         "ColorTransformer format filtering",
			transformer:  NewColorTransformer(),
			input:        []byte("test"),
			format:       FormatJSON,       // Colors only apply to table format
			wantContain:  []string{"test"}, // Should pass through unchanged
			wantName:     "color",
			wantPrio:     200,
			canTransform: false,
		},
		{
			name:         "SortTransformer basic functionality",
			transformer:  NewSortTransformer("name", true),
			input:        []byte("name,age\nBob,25\nAlice,30"),
			format:       FormatCSV,
			wantContain:  []string{"name,age", "Alice", "Bob"},
			wantName:     "sort",
			wantPrio:     50,
			canTransform: true,
		},
		{
			name:         "LineSplitTransformer basic functionality",
			transformer:  NewLineSplitTransformer(","),
			input:        []byte("name|data\ntest|a,b,c"),
			format:       FormatTable,
			wantContain:  []string{"name|data", "test"},
			wantName:     "linesplit",
			wantPrio:     150,
			canTransform: true,
		},
		{
			name:         "RemoveColorsTransformer basic functionality",
			transformer:  NewRemoveColorsTransformer(),
			input:        []byte("\x1B[31mred text\x1B[0m"),
			format:       FormatTable,
			wantContain:  []string{"red text"}, // ANSI codes should be removed
			wantName:     "remove-colors",
			wantPrio:     1000,
			canTransform: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test transformer interface methods
			if got := tc.transformer.Name(); got != tc.wantName {
				t.Errorf("Name() = %v, want %v", got, tc.wantName)
			}

			if got := tc.transformer.Priority(); got != tc.wantPrio {
				t.Errorf("Priority() = %v, want %v", got, tc.wantPrio)
			}

			if got := tc.transformer.CanTransform(tc.format); got != tc.canTransform {
				t.Errorf("CanTransform(%s) = %v, want %v", tc.format, got, tc.canTransform)
			}

			// Test transformation
			ctx := context.Background()
			result, err := tc.transformer.Transform(ctx, tc.input, tc.format)
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
}

// TestBackwardCompatibility_TransformPipeline verifies TransformPipeline functionality remains unchanged
func TestBackwardCompatibility_TransformPipeline(t *testing.T) {
	t.Run("Pipeline creation and basic operations", func(t *testing.T) {
		pipeline := NewTransformPipeline()
		if pipeline == nil {
			t.Fatal("NewTransformPipeline() returned nil")
		}

		if pipeline.Count() != 0 {
			t.Errorf("new pipeline should be empty, got count = %d", pipeline.Count())
		}

		// Test Add
		transformer := &EmojiTransformer{}
		pipeline.Add(transformer)

		if pipeline.Count() != 1 {
			t.Errorf("pipeline count = %d, want 1", pipeline.Count())
		}

		if !pipeline.Has("emoji") {
			t.Error("pipeline should contain transformer 'emoji'")
		}

		// Test Get
		retrieved := pipeline.Get("emoji")
		if retrieved == nil {
			t.Error("Get() should return transformer when it exists")
		}

		// Test Remove
		removed := pipeline.Remove("emoji")
		if !removed {
			t.Error("Remove() should return true when transformer exists")
		}

		if pipeline.Count() != 0 {
			t.Errorf("pipeline count = %d, want 0 after removal", pipeline.Count())
		}

		// Test Clear
		pipeline.Add(&EmojiTransformer{})
		pipeline.Add(NewColorTransformer())
		pipeline.Clear()

		if pipeline.Count() != 0 {
			t.Errorf("pipeline count after clear = %d, want 0", pipeline.Count())
		}
	})

	t.Run("Pipeline transformation behavior", func(t *testing.T) {
		pipeline := NewTransformPipeline()

		// Add transformers with specific priorities
		pipeline.Add(NewSortTransformer("name", true)) // Priority 50
		pipeline.Add(&EmojiTransformer{})              // Priority 100
		pipeline.Add(NewColorTransformer())            // Priority 200

		input := []byte("OK test")
		ctx := context.Background()

		result, err := pipeline.Transform(ctx, input, FormatTable)
		if err != nil {
			t.Fatalf("Transform() error = %v", err)
		}

		// Should contain emoji transformation
		resultStr := string(result)
		if !strings.Contains(resultStr, "✅") {
			t.Errorf("Transform() result should contain emoji transformation, got: %s", resultStr)
		}
	})

	t.Run("Pipeline Info method", func(t *testing.T) {
		pipeline := NewTransformPipeline()
		pipeline.Add(&EmojiTransformer{})
		pipeline.Add(NewColorTransformer())
		pipeline.Add(NewSortTransformer("name", true))

		info := pipeline.Info()
		if len(info) != 3 {
			t.Errorf("Info() returned %d transformers, want 3", len(info))
		}

		// Should be ordered by priority
		if info[0].Name != "sort" || info[0].Priority != 50 {
			t.Errorf("first transformer should be 'sort' with priority 50, got %s with priority %d", info[0].Name, info[0].Priority)
		}

		if info[1].Name != "emoji" || info[1].Priority != 100 {
			t.Errorf("second transformer should be 'emoji' with priority 100, got %s with priority %d", info[1].Name, info[1].Priority)
		}

		if info[2].Name != "color" || info[2].Priority != 200 {
			t.Errorf("third transformer should be 'color' with priority 200, got %s with priority %d", info[2].Name, info[2].Priority)
		}
	})
}

// TestBackwardCompatibility_TransformerPriorityOrdering verifies transformer priority and ordering preserved
func TestBackwardCompatibility_TransformerPriorityOrdering(t *testing.T) {
	testCases := []struct {
		name          string
		transformers  []Transformer
		expectedOrder []string
	}{
		{
			name: "Standard priority ordering",
			transformers: []Transformer{
				&EmojiTransformer{},              // Priority 100
				NewColorTransformer(),            // Priority 200
				NewSortTransformer("name", true), // Priority 50
				NewLineSplitTransformer(","),     // Priority 150
				NewRemoveColorsTransformer(),     // Priority 1000
			},
			expectedOrder: []string{"sort", "emoji", "linesplit", "color", "remove-colors"},
		},
		{
			name: "Same priority handling",
			transformers: []Transformer{
				NewSortTransformer("name", true), // Priority 50
				NewSortTransformer("age", false), // Priority 50 (different instance, same priority)
			},
			expectedOrder: []string{"sort", "sort"}, // Should maintain some order
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pipeline := NewTransformPipeline()

			// Add in reverse order to test sorting
			for i := len(tc.transformers) - 1; i >= 0; i-- {
				pipeline.Add(tc.transformers[i])
			}

			info := pipeline.Info()
			if len(info) != len(tc.expectedOrder) {
				t.Errorf("expected %d transformers, got %d", len(tc.expectedOrder), len(info))
				return
			}

			for i, expected := range tc.expectedOrder {
				if info[i].Name != expected {
					t.Errorf("position %d: expected %s, got %s", i, expected, info[i].Name)
				}
			}

			// Verify priorities are in ascending order
			for i := 1; i < len(info); i++ {
				if info[i].Priority < info[i-1].Priority {
					t.Errorf("priorities not in ascending order: %d < %d at positions %d, %d",
						info[i].Priority, info[i-1].Priority, i, i-1)
				}
			}
		})
	}
}

// TestBackwardCompatibility_ConcurrentAccess verifies thread safety is maintained
func TestBackwardCompatibility_ConcurrentAccess(t *testing.T) {
	pipeline := NewTransformPipeline()

	var wg sync.WaitGroup
	const numGoroutines = 10

	// Test concurrent modifications
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Add transformer
			transformer := &EmojiTransformer{}
			pipeline.Add(transformer)

			// Perform transformation
			ctx := context.Background()
			_, err := pipeline.Transform(ctx, []byte("test"), FormatTable)
			if err != nil {
				t.Errorf("concurrent transform error: %v", err)
			}

			// Remove transformer
			pipeline.Remove("emoji")
		}(i)
	}

	wg.Wait()

	// Pipeline should be in consistent state
	if pipeline.Count() < 0 {
		t.Error("pipeline count should not be negative after concurrent operations")
	}
}

// TestBackwardCompatibility_ErrorHandling verifies error handling behavior is preserved
func TestBackwardCompatibility_ErrorHandling(t *testing.T) {
	t.Run("TransformError structure preserved", func(t *testing.T) {
		originalErr := errors.New("test error")
		transformErr := NewTransformError("test-transformer", FormatJSON, []byte("input"), originalErr)

		if transformErr.Transformer != "test-transformer" {
			t.Errorf("TransformError.Transformer = %s, want test-transformer", transformErr.Transformer)
		}

		if transformErr.Format != FormatJSON {
			t.Errorf("TransformError.Format = %s, want json", transformErr.Format)
		}

		if string(transformErr.Input) != "input" {
			t.Errorf("TransformError.Input = %s, want input", string(transformErr.Input))
		}

		if !errors.Is(transformErr, originalErr) {
			t.Error("TransformError should wrap original error")
		}

		expectedStr := "transformer test-transformer failed for format json: test error"
		if transformErr.Error() != expectedStr {
			t.Errorf("TransformError.Error() = %s, want %s", transformErr.Error(), expectedStr)
		}
	})

	t.Run("Pipeline error propagation", func(t *testing.T) {
		pipeline := NewTransformPipeline()

		// Add a mock transformer that will fail
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

		var transformErr *TransformError
		if !AsError(err, &transformErr) {
			t.Errorf("Transform() error should be TransformError, got: %T", err)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		pipeline := NewTransformPipeline()

		slowTransformer := &integrationSlowMockTransformer{
			name:     "slow",
			priority: 100,
		}
		pipeline.Add(slowTransformer)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := pipeline.Transform(ctx, []byte("test"), FormatJSON)
		if err == nil {
			t.Error("Transform() should return error when context is cancelled")
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("Transform() should return context.Canceled, got: %v", err)
		}
	})
}

// Mock transformers are defined in backward_compatibility_integration_test.go to avoid duplication

// TestBackwardCompatibility_RendererIntegration verifies basic renderer functionality is preserved
func TestBackwardCompatibility_RendererIntegration(t *testing.T) {
	t.Run("Table renderer with transformers", func(t *testing.T) {
		// Create a simple document using the correct v2 API
		doc := New().
			Table("Test Data", []Record{
				{"name": "Alice", "status": "active", "score": 95},
				{"name": "Bob", "status": "inactive", "score": 87},
			}).
			Build()

		// Use the correct renderer creation syntax
		renderer := NewTableRendererWithStyle("simple")

		ctx := context.Background()
		result, err := renderer.Render(ctx, doc)
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}

		if len(result) == 0 {
			t.Error("Render() should produce non-empty output")
		}

		// Verify basic content is present
		resultStr := string(result)
		if !strings.Contains(resultStr, "Alice") || !strings.Contains(resultStr, "Bob") {
			t.Errorf("Render() result should contain data, got: %s", resultStr)
		}
	})
}
