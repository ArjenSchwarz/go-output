package output

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Mock transformers for testing

type mockTransformer struct {
	name     string
	priority int
	formats  []string
	output   string
	err      error
	calls    int
	mu       sync.Mutex
}

func newMockTransformer(name string, priority int, formats []string, output string) *mockTransformer {
	return &mockTransformer{
		name:     name,
		priority: priority,
		formats:  formats,
		output:   output,
	}
}

func (m *mockTransformer) Name() string {
	return m.name
}

func (m *mockTransformer) Priority() int {
	return m.priority
}

func (m *mockTransformer) CanTransform(format string) bool {
	for _, f := range m.formats {
		if f == format {
			return true
		}
	}
	return false
}

func (m *mockTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls++

	if m.err != nil {
		return nil, m.err
	}

	if m.output != "" {
		return []byte(fmt.Sprintf("%s[%s]", string(input), m.output)), nil
	}

	return input, nil
}

func (m *mockTransformer) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *mockTransformer) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Test basic pipeline operations

func TestNewTransformPipeline(t *testing.T) {
	pipeline := NewTransformPipeline()

	if pipeline == nil {
		t.Fatal("NewTransformPipeline() returned nil")
	}

	if pipeline.Count() != 0 {
		t.Errorf("new pipeline should be empty, got count = %d", pipeline.Count())
	}
}

func TestTransformPipeline_Add(t *testing.T) {
	pipeline := NewTransformPipeline()
	transformer := newMockTransformer("test", 100, []string{"json"}, "")

	pipeline.Add(transformer)

	if pipeline.Count() != 1 {
		t.Errorf("pipeline count = %d, want 1", pipeline.Count())
	}

	if !pipeline.Has("test") {
		t.Error("pipeline should contain transformer 'test'")
	}
}

func TestTransformPipeline_Remove(t *testing.T) {
	pipeline := NewTransformPipeline()
	transformer := newMockTransformer("test", 100, []string{"json"}, "")

	pipeline.Add(transformer)

	removed := pipeline.Remove("test")
	if !removed {
		t.Error("Remove() should return true when transformer exists")
	}

	if pipeline.Count() != 0 {
		t.Errorf("pipeline count = %d, want 0", pipeline.Count())
	}

	if pipeline.Has("test") {
		t.Error("pipeline should not contain transformer 'test' after removal")
	}

	// Test removing non-existent transformer
	removed = pipeline.Remove("nonexistent")
	if removed {
		t.Error("Remove() should return false when transformer doesn't exist")
	}
}

func TestTransformPipeline_Get(t *testing.T) {
	pipeline := NewTransformPipeline()
	transformer := newMockTransformer("test", 100, []string{"json"}, "")

	pipeline.Add(transformer)

	retrieved := pipeline.Get("test")
	if retrieved == nil {
		t.Error("Get() should return transformer when it exists")
	}

	if retrieved.Name() != "test" {
		t.Errorf("retrieved transformer name = %s, want test", retrieved.Name())
	}

	// Test getting non-existent transformer
	retrieved = pipeline.Get("nonexistent")
	if retrieved != nil {
		t.Error("Get() should return nil when transformer doesn't exist")
	}
}

func TestTransformPipeline_Clear(t *testing.T) {
	pipeline := NewTransformPipeline()
	pipeline.Add(newMockTransformer("test1", 100, []string{"json"}, ""))
	pipeline.Add(newMockTransformer("test2", 200, []string{"yaml"}, ""))

	pipeline.Clear()

	if pipeline.Count() != 0 {
		t.Errorf("pipeline count after clear = %d, want 0", pipeline.Count())
	}
}

// Test priority-based ordering

func TestTransformPipeline_PriorityOrdering(t *testing.T) {
	pipeline := NewTransformPipeline()

	// Add transformers in reverse priority order
	high := newMockTransformer("high", 300, []string{"json"}, "HIGH")
	medium := newMockTransformer("medium", 200, []string{"json"}, "MEDIUM")
	low := newMockTransformer("low", 100, []string{"json"}, "LOW")

	pipeline.Add(high)
	pipeline.Add(medium)
	pipeline.Add(low)

	// Transform should apply in priority order (low to high)
	result, err := pipeline.Transform(context.Background(), []byte("input"), "json")
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	expected := "input[LOW][MEDIUM][HIGH]"
	if string(result) != expected {
		t.Errorf("Transform() result = %s, want %s", string(result), expected)
	}
}

func TestTransformPipeline_FormatFiltering(t *testing.T) {
	pipeline := NewTransformPipeline()

	jsonTransformer := newMockTransformer("json", 100, []string{"json"}, "JSON")
	yamlTransformer := newMockTransformer("yaml", 200, []string{"yaml"}, "YAML")
	universalTransformer := newMockTransformer("universal", 300, []string{"json", "yaml", "html"}, "UNIVERSAL")

	pipeline.Add(jsonTransformer)
	pipeline.Add(yamlTransformer)
	pipeline.Add(universalTransformer)

	// Test JSON format - should only apply json and universal transformers
	result, err := pipeline.Transform(context.Background(), []byte("input"), "json")
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	expected := "input[JSON][UNIVERSAL]"
	if string(result) != expected {
		t.Errorf("JSON transform result = %s, want %s", string(result), expected)
	}

	// Test YAML format - should only apply yaml and universal transformers
	result, err = pipeline.Transform(context.Background(), []byte("input"), "yaml")
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	expected = "input[YAML][UNIVERSAL]"
	if string(result) != expected {
		t.Errorf("YAML transform result = %s, want %s", string(result), expected)
	}

	// Test HTML format - should only apply universal transformer
	result, err = pipeline.Transform(context.Background(), []byte("input"), "html")
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	expected = "input[UNIVERSAL]"
	if string(result) != expected {
		t.Errorf("HTML transform result = %s, want %s", string(result), expected)
	}

	// Test unsupported format - should return input unchanged
	result, err = pipeline.Transform(context.Background(), []byte("input"), "unsupported")
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	expected = "input"
	if string(result) != expected {
		t.Errorf("unsupported format result = %s, want %s", string(result), expected)
	}
}

// Test error handling

func TestTransformPipeline_TransformError(t *testing.T) {
	pipeline := NewTransformPipeline()

	goodTransformer := newMockTransformer("good", 100, []string{"json"}, "GOOD")
	badTransformer := newMockTransformer("bad", 200, []string{"json"}, "BAD")

	badTransformer.SetError(errors.New("transform failed"))

	pipeline.Add(goodTransformer)
	pipeline.Add(badTransformer)

	_, err := pipeline.Transform(context.Background(), []byte("input"), "json")
	if err == nil {
		t.Error("Transform() should return error when transformer fails")
	}

	if err.Error() != `transformer "bad" failed: transform failed` {
		t.Errorf("Transform() error should wrap transformer error, got: %v", err)
	}

	// Verify the good transformer was called but bad one caused failure
	if goodTransformer.CallCount() != 1 {
		t.Errorf("good transformer call count = %d, want 1", goodTransformer.CallCount())
	}

	if badTransformer.CallCount() != 1 {
		t.Errorf("bad transformer call count = %d, want 1", badTransformer.CallCount())
	}
}

func TestTransformPipeline_ContextCancellation(t *testing.T) {
	pipeline := NewTransformPipeline()

	// Create a slow transformer that checks context
	slowTransformer := &slowMockTransformer{
		mockTransformer: mockTransformer{
			name:     "slow",
			priority: 100,
			formats:  []string{"json"},
		},
	}

	pipeline.Add(slowTransformer)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := pipeline.Transform(ctx, []byte("input"), "json")
	if err == nil {
		t.Error("Transform() should return error when context is cancelled")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Transform() should return context.Canceled, got: %v", err)
	}
}

// slowMockTransformer simulates a slow operation that respects context cancellation
type slowMockTransformer struct {
	mockTransformer
}

func (s *slowMockTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++

	select {
	case <-time.After(100 * time.Millisecond):
		return input, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Test Info method

func TestTransformPipeline_Info(t *testing.T) {
	pipeline := NewTransformPipeline()

	// Add transformers with different priorities and format support
	pipeline.Add(newMockTransformer("high", 300, []string{"json", "yaml"}, ""))
	pipeline.Add(newMockTransformer("low", 100, []string{"json"}, ""))
	pipeline.Add(newMockTransformer("medium", 200, []string{"html", "markdown"}, ""))

	info := pipeline.Info()

	if len(info) != 3 {
		t.Errorf("Info() returned %d transformers, want 3", len(info))
	}

	// Should be ordered by priority
	if info[0].Name != "low" || info[0].Priority != 100 {
		t.Errorf("first transformer should be 'low' with priority 100, got %s with priority %d", info[0].Name, info[0].Priority)
	}

	if info[1].Name != "medium" || info[1].Priority != 200 {
		t.Errorf("second transformer should be 'medium' with priority 200, got %s with priority %d", info[1].Name, info[1].Priority)
	}

	if info[2].Name != "high" || info[2].Priority != 300 {
		t.Errorf("third transformer should be 'high' with priority 300, got %s with priority %d", info[2].Name, info[2].Priority)
	}

	// Check format support
	if len(info[0].Formats) != 1 || info[0].Formats[0] != "json" {
		t.Errorf("low transformer should support [json], got %v", info[0].Formats)
	}

	if len(info[2].Formats) != 2 {
		t.Errorf("high transformer should support 2 formats, got %d", len(info[2].Formats))
	}
}

// Test concurrent access

func TestTransformPipeline_ConcurrentAccess(t *testing.T) {
	pipeline := NewTransformPipeline()

	// Start multiple goroutines adding and removing transformers
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)

		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("transformer-%d", i)
			pipeline.Add(newMockTransformer(name, i*100, []string{"json"}, ""))
		}(i)

		go func(i int) {
			defer wg.Done()
			// Transform with some transformers
			ctx := context.Background()
			_, _ = pipeline.Transform(ctx, []byte("test"), "json")
		}(i)
	}

	wg.Wait()

	// Pipeline should be in a consistent state
	if pipeline.Count() != 10 {
		t.Errorf("expected 10 transformers after concurrent operations, got %d", pipeline.Count())
	}
}

// Test TransformError

func TestTransformError(t *testing.T) {
	originalErr := errors.New("original error")
	transformErr := NewTransformError("test-transformer", "json", []byte("input"), originalErr)

	if transformErr.Transformer != "test-transformer" {
		t.Errorf("TransformError.Transformer = %s, want test-transformer", transformErr.Transformer)
	}

	if transformErr.Format != "json" {
		t.Errorf("TransformError.Format = %s, want json", transformErr.Format)
	}

	if string(transformErr.Input) != "input" {
		t.Errorf("TransformError.Input = %s, want input", string(transformErr.Input))
	}

	if !errors.Is(transformErr, originalErr) {
		t.Error("TransformError should wrap original error")
	}

	errorStr := transformErr.Error()
	expectedStr := "transformer test-transformer failed for format json: original error"
	if errorStr != expectedStr {
		t.Errorf("TransformError.Error() = %s, want %s", errorStr, expectedStr)
	}
}

// Benchmark tests

func BenchmarkTransformPipeline_Transform(b *testing.B) {
	pipeline := NewTransformPipeline()

	// Add several transformers
	for i := 0; i < 5; i++ {
		transformer := newMockTransformer(fmt.Sprintf("bench-%d", i), i*100, []string{"json"}, "")
		pipeline.Add(transformer)
	}

	input := []byte("benchmark input data")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.Transform(ctx, input, "json")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTransformPipeline_PrioritySort(b *testing.B) {
	pipeline := NewTransformPipeline()

	// Add many transformers in random order
	for i := 0; i < 100; i++ {
		priority := (i * 37) % 1000 // Pseudo-random priorities
		transformer := newMockTransformer(fmt.Sprintf("bench-%d", i), priority, []string{"json"}, "")
		pipeline.Add(transformer)
	}

	input := []byte("benchmark input")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.Transform(ctx, input, "json")
		if err != nil {
			b.Fatal(err)
		}
	}
}
