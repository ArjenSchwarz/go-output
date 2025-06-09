package format

import "testing"

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
