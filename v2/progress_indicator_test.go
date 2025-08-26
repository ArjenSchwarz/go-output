package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTextProgress_ProgressDisplay(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0), // No throttling for tests
		WithWidth(10),
		WithPercentage(true),
	)

	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()

	// Check for progress bar elements
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Error("output should contain progress bar brackets")
	}

	if !strings.Contains(output, "100.0%") {
		t.Error("output should contain percentage when completed")
	}

	if !strings.Contains(output, "(100/100)") {
		t.Error("output should contain count display")
	}
}

func TestTextProgress_WithPrefix(t *testing.T) {
	var buf bytes.Buffer
	prefix := "Processing files:"
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
		WithPrefix(prefix),
	)

	progress.SetTotal(10)
	progress.SetCurrent(5)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, prefix) {
		t.Errorf("output should contain prefix %q", prefix)
	}
}

func TestTextProgress_WithSuffix(t *testing.T) {
	var buf bytes.Buffer
	suffix := "files processed"
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
		WithSuffix(suffix),
	)

	progress.SetTotal(10)
	progress.SetCurrent(5)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, suffix) {
		t.Errorf("output should contain suffix %q", suffix)
	}
}
