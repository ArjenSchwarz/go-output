package format

import (
	"bytes"
	"context"
	"log"
	"testing"
	"time"
)

func TestNoOpProgressBasics(t *testing.T) {
	settings := NewOutputSettings()
	np := newNoOpProgress(settings)

	np.SetTotal(10)
	np.Increment(3)
	if np.current != 3 {
		t.Errorf("expected value 3, got %d", np.current)
	}

	np.SetCurrent(5)
	if np.current != 5 {
		t.Errorf("expected value 5, got %d", np.current)
	}

	np.Complete()
	if np.IsActive() {
		t.Errorf("progress should not be active")
	}
}

func TestNoOpProgressFail(t *testing.T) {
	settings := NewOutputSettings()
	np := newNoOpProgress(settings)
	np.SetTotal(2)
	np.Fail(assertError("boom"))
	if np.IsActive() {
		t.Errorf("progress should remain inactive")
	}
}

func TestNoOpProgressContextCancel(t *testing.T) {
	settings := NewOutputSettings()
	np := newNoOpProgress(settings)
	ctx, cancel := context.WithCancel(context.Background())
	np.SetContext(ctx)
	cancel()
	time.Sleep(10 * time.Millisecond)
	if np.IsActive() {
		t.Errorf("progress should remain inactive")
	}
}

func TestNoOpProgressColorAndStatus(t *testing.T) {
	settings := NewOutputSettings()
	settings.ProgressOptions.Color = ProgressColorGreen
	settings.ProgressOptions.Status = "go"
	np := newNoOpProgress(settings)
	if np.options.Color != ProgressColorGreen {
		t.Errorf("expected color to be propagated")
	}

	np.SetColor(ProgressColorRed)
	if np.options.Color != ProgressColorRed {
		t.Errorf("expected color red after SetColor")
	}

	np.SetStatus("running")
	// only ensures no panic; NoOpProgress doesn't store status
}

func TestNoOpProgressNoOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	settings := NewOutputSettings()
	np := newNoOpProgress(settings)
	np.SetTotal(1)
	np.SetCurrent(1)
	np.Increment(1)
	np.Complete()
	if buf.Len() != 0 {
		t.Errorf("expected no log output, got %s", buf.String())
	}
}
