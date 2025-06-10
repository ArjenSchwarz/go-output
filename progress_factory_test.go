package format

import "testing"

func TestNewProgressSelection(t *testing.T) {
	settings := NewOutputSettings()

	settings.OutputFormat = "json"
	if _, ok := NewProgress(settings).(*NoOpProgress); !ok {
		t.Errorf("expected NoOpProgress for json format")
	}

	settings.OutputFormat = "table"
	if _, ok := NewProgress(settings).(*PrettyProgress); !ok {
		t.Errorf("expected PrettyProgress for table format")
	}

	settings.ProgressEnabled = false
	if _, ok := NewProgress(settings).(*NoOpProgress); !ok {
		t.Errorf("expected NoOpProgress when disabled")
	}
}
