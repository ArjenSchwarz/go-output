package format

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestPrettyProgressBasics(t *testing.T) {
	settings := NewOutputSettings()
	pp := newPrettyProgress(settings)

	pp.SetTotal(10)
	pp.Increment(3)
	if v := pp.tracker.Value(); v != 3 {
		t.Errorf("expected value 3, got %d", v)
	}

	pp.SetCurrent(5)
	if v := pp.tracker.Value(); v != 5 {
		t.Errorf("expected value 5, got %d", v)
	}

	pp.Complete()
	if pp.IsActive() {
		t.Errorf("progress should not be active after completion")
	}
}

func TestPrettyProgressFail(t *testing.T) {
	settings := NewOutputSettings()
	pp := newPrettyProgress(settings)
	pp.SetTotal(2)
	pp.Fail(assertError("boom"))
	if pp.IsActive() {
		t.Errorf("progress should stop on failure")
	}
}

func TestPrettyProgressContextCancel(t *testing.T) {
	settings := NewOutputSettings()
	pp := newPrettyProgress(settings)
	ctx, cancel := context.WithCancel(context.Background())
	pp.SetContext(ctx)
	cancel()
	time.Sleep(50 * time.Millisecond)
	if pp.IsActive() {
		t.Errorf("progress should stop when context is cancelled")
	}
}

func TestPrettyProgressStatusAndColor(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	settings := NewOutputSettings()
	settings.ProgressOptions.Status = "starting"
	settings.ProgressOptions.Color = ProgressColorBlue
	pp := newPrettyProgress(settings)
	os.Stdout = orig
	_ = r.Close()
	_ = w.Close()

	if pp.tracker.Message != "starting" {
		t.Errorf("expected status 'starting', got %s", pp.tracker.Message)
	}
	if pp.options.Color != ProgressColorBlue {
		t.Errorf("expected color blue, got %v", pp.options.Color)
	}

	pp.SetStatus("running")
	if pp.tracker.Message != "running" {
		t.Errorf("expected message 'running', got %s", pp.tracker.Message)
	}

	pp.SetColor(ProgressColorYellow)
	if pp.options.Color != ProgressColorYellow {
		t.Errorf("expected color yellow, got %v", pp.options.Color)
	}
}

func TestPrettyProgressCompletionStates(t *testing.T) {
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	settings := NewOutputSettings()
	pp := newPrettyProgress(settings)
	os.Stdout = orig
	_ = r.Close()
	_ = w.Close()

	pp.SetTotal(1)
	pp.Complete()
	if pp.options.Color != ProgressColorGreen {
		t.Errorf("expected green color on complete, got %v", pp.options.Color)
	}
	if pp.IsActive() {
		t.Errorf("progress should be inactive after Complete")
	}

	pp = newPrettyProgress(settings)
	pp.SetTotal(1)
	pp.Fail(assertError("bad"))
	if pp.options.Color != ProgressColorRed {
		t.Errorf("expected red color on fail, got %v", pp.options.Color)
	}
	if pp.IsActive() {
		t.Errorf("progress should be inactive after Fail")
	}
}

func TestPrettyProgressConcurrent(t *testing.T) {
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	settings := NewOutputSettings()
	pp := newPrettyProgress(settings)
	os.Stdout = orig
	_ = r.Close()
	_ = w.Close()

	pp.SetTotal(100)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			pp.Increment(2)
			wg.Done()
		}()
	}
	wg.Wait()
	if v := pp.tracker.Value(); v != 100 {
		t.Errorf("expected value 100, got %d", v)
	}
}

func TestPrettyProgressTTYDetection(t *testing.T) {
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	settings := NewOutputSettings()
	pp := newPrettyProgress(settings)
	os.Stdout = orig
	_ = r.Close()
	_ = w.Close()

	if pp.writer != nil {
		t.Errorf("expected writer to be nil for non TTY stdout")
	}
}

func TestPrettyProgressIntegrationTableOutput(t *testing.T) {
	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	settings := NewOutputSettings()
	settings.OutputFormat = "table"
	pp := newPrettyProgress(settings)
	oa := OutputArray{
		Settings: settings,
		Contents: []OutputHolder{{Contents: map[string]interface{}{"col": "val"}}},
		Keys:     []string{"col"},
	}
	oa.Write()
	os.Stdout = orig
	_ = r.Close()
	_ = w.Close()

	if pp.IsActive() {
		t.Errorf("progress should stop before output is written")
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
