package output

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// BenchmarkBackwardCompatibility_TransformPipeline benchmarks existing transform pipeline performance
func BenchmarkBackwardCompatibility_TransformPipeline(b *testing.B) {
	pipeline := NewTransformPipeline()

	// Add several transformers to test performance
	pipeline.Add(&EmojiTransformer{})
	pipeline.Add(NewColorTransformer())
	pipeline.Add(NewSortTransformer("name", true))
	pipeline.Add(NewLineSplitTransformer(","))

	input := []byte("OK test data\nname,value\nAlice,100\nBob,200")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.Transform(ctx, input, FormatTable)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBackwardCompatibility_EmojiTransformer benchmarks EmojiTransformer performance
func BenchmarkBackwardCompatibility_EmojiTransformer(b *testing.B) {
	transformer := &EmojiTransformer{}
	input := []byte("OK No !! Yes true false")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(ctx, input, FormatTable)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBackwardCompatibility_SortTransformer benchmarks SortTransformer performance
func BenchmarkBackwardCompatibility_SortTransformer(b *testing.B) {
	transformer := NewSortTransformer("name", true)

	// Create larger data for meaningful benchmarking
	var data []string
	data = append(data, "name,age,score")
	for i := 0; i < 100; i++ {
		data = append(data, fmt.Sprintf("Person%d,%d,%d", i, 20+i%40, 50+i%50))
	}
	input := []byte(strings.Join(data, "\n"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(ctx, input, FormatCSV)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBackwardCompatibility_ColorTransformer benchmarks ColorTransformer performance
func BenchmarkBackwardCompatibility_ColorTransformer(b *testing.B) {
	transformer := NewColorTransformer()
	input := []byte("✅ success ❌ failure 🚨 alert")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(ctx, input, FormatTable)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBackwardCompatibility_LineSplitTransformer benchmarks LineSplitTransformer performance
func BenchmarkBackwardCompatibility_LineSplitTransformer(b *testing.B) {
	transformer := NewLineSplitTransformer(",")
	input := []byte("name|data\ntest1|a,b,c\ntest2|d,e,f\ntest3|g,h,i")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(ctx, input, FormatTable)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBackwardCompatibility_RemoveColorsTransformer benchmarks RemoveColorsTransformer performance
func BenchmarkBackwardCompatibility_RemoveColorsTransformer(b *testing.B) {
	transformer := NewRemoveColorsTransformer()
	input := []byte("\x1B[31mred text\x1B[0m\x1B[32mgreen text\x1B[0m\x1B[33myellow text\x1B[0m")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(ctx, input, FormatTable)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBackwardCompatibility_PipelinePrioritySort benchmarks priority sorting performance
func BenchmarkBackwardCompatibility_PipelinePrioritySort(b *testing.B) {
	pipeline := NewTransformPipeline()

	// Add many transformers with random priorities for stress testing
	for i := 0; i < 50; i++ {
		priority := (i * 37) % 1000 // Pseudo-random priorities
		transformer := &mockTransformer{
			name:     fmt.Sprintf("bench-%d", i),
			priority: priority,
			formats:  []string{FormatJSON},
		}
		pipeline.Add(transformer)
	}

	input := []byte("benchmark input")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.Transform(ctx, input, FormatJSON)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBackwardCompatibility_ConcurrentTransforms benchmarks concurrent transform performance
func BenchmarkBackwardCompatibility_ConcurrentTransforms(b *testing.B) {
	pipeline := NewTransformPipeline()
	pipeline.Add(&EmojiTransformer{})
	pipeline.Add(NewColorTransformer())

	input := []byte("OK test concurrent")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := context.Background()
			_, err := pipeline.Transform(ctx, input, FormatTable)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkBackwardCompatibility_LargeDataTransform benchmarks performance with larger datasets
func BenchmarkBackwardCompatibility_LargeDataTransform(b *testing.B) {
	pipeline := NewTransformPipeline()
	pipeline.Add(&EmojiTransformer{})

	// Create large input data
	var data []string
	for i := 0; i < 1000; i++ {
		data = append(data, fmt.Sprintf("Record %d: OK No !!", i))
	}
	input := []byte(strings.Join(data, "\n"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.Transform(ctx, input, FormatTable)
		if err != nil {
			b.Fatal(err)
		}
	}
}
