package format

import "testing"

func TestNewProgressSelection(t *testing.T) {
	settings := NewOutputSettings()
	formats := map[string]bool{
		"json":     false,
		"yaml":     false,
		"csv":      false,
		"dot":      false,
		"table":    true,
		"markdown": true,
		"html":     true,
	}
	for format, pretty := range formats {
		settings.OutputFormat = format
		p := NewProgress(settings)
		if pretty {
			if _, ok := p.(*PrettyProgress); !ok {
				t.Errorf("expected PrettyProgress for %s format", format)
			}
		} else {
			if _, ok := p.(*NoOpProgress); !ok {
				t.Errorf("expected NoOpProgress for %s format", format)
			}
		}
	}

	settings.ProgressEnabled = false
	if _, ok := NewProgress(settings).(*NoOpProgress); !ok {
		t.Errorf("expected NoOpProgress when disabled")
	}
}

func TestNewProgressPropagatesOptions(t *testing.T) {
	settings := NewOutputSettings()
	settings.OutputFormat = "table"
	settings.ProgressOptions.Color = ProgressColorBlue
	settings.ProgressOptions.Status = "init"
	p := NewProgress(settings)
	pp, ok := p.(*PrettyProgress)
	if !ok {
		t.Fatalf("expected PrettyProgress")
	}
	if pp.options.Color != ProgressColorBlue {
		t.Errorf("expected color blue")
	}
	if pp.tracker.Message != "init" {
		t.Errorf("expected status propagated")
	}
}
