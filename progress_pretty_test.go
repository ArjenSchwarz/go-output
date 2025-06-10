package format

import (
	"context"
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

type assertError string

func (e assertError) Error() string { return string(e) }
