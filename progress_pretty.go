package format

import (
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mattn/go-isatty"
)

// PrettyProgress wraps go-pretty's progress.Writer to implement the Progress interface.
type PrettyProgress struct {
	writer  progress.Writer
	tracker *progress.Tracker
	options ProgressOptions
	active  bool
	mutex   sync.Mutex
}

// newPrettyProgress creates a new PrettyProgress instance.
func newPrettyProgress(settings *OutputSettings) *PrettyProgress {
	pp := &PrettyProgress{}
	pp.tracker = &progress.Tracker{}
	if settings != nil {
		pp.options = ProgressOptions{}
		pp.tracker.Message = ""
	}
	if isatty.IsTerminal(os.Stdout.Fd()) {
		w := progress.NewWriter()
		w.SetAutoStop(false)
		w.SetTrackerLength(40)
		w.SetUpdateFrequency(time.Millisecond * 100)
		w.SetTrackerPosition(progress.PositionRight)
		w.SetOutputWriter(os.Stdout)
		w.SetNumTrackersExpected(1)
		style := progress.StyleDefault
		style.Visibility.TrackerOverall = false
		w.SetStyle(style)
		w.AppendTracker(pp.tracker)
		pp.writer = w
		pp.start()
	}
	pp.SetColor(pp.options.Color)
	runtime.SetFinalizer(pp, func(p *PrettyProgress) { p.stop() })
	return pp
}

func (pp *PrettyProgress) start() {
	if pp.writer == nil {
		return
	}
	pp.mutex.Lock()
	if pp.active {
		pp.mutex.Unlock()
		return
	}
	pp.active = true
	pp.mutex.Unlock()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// ensure progress stops on panic
			}
			pp.stop()
		}()
		pp.writer.Render()
	}()
}

func (pp *PrettyProgress) stop() {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	if pp.writer != nil && pp.active {
		pp.writer.Stop()
	}
	pp.active = false
}

// SetTotal sets the expected total value.
func (pp *PrettyProgress) SetTotal(total int) {
	if pp.tracker != nil {
		pp.tracker.UpdateTotal(int64(total))
	}
}

// SetCurrent sets the current progress value.
func (pp *PrettyProgress) SetCurrent(current int) {
	if pp.tracker != nil {
		pp.tracker.SetValue(int64(current))
	}
}

// Increment increases the progress by n.
func (pp *PrettyProgress) Increment(n int) {
	if pp.tracker != nil {
		pp.tracker.Increment(int64(n))
	}
}

// SetStatus updates the status message.
func (pp *PrettyProgress) SetStatus(status string) {
	if pp.tracker != nil {
		pp.tracker.UpdateMessage(status)
	}
}

// SetColor changes the progress bar color.
func (pp *PrettyProgress) SetColor(color ProgressColor) {
	pp.options.Color = color
	if pp.writer == nil {
		return
	}
	style := *pp.writer.Style()
	var c text.Colors
	switch color {
	case ProgressColorGreen:
		c = text.Colors{text.FgGreen}
	case ProgressColorRed:
		c = text.Colors{text.FgRed}
	case ProgressColorYellow:
		c = text.Colors{text.FgYellow}
	case ProgressColorBlue:
		c = text.Colors{text.FgBlue}
	default:
		c = text.Colors{}
	}
	style.Colors.Message = c
	style.Colors.Tracker = c
	style.Colors.Value = c
	pp.writer.SetStyle(style)
}

// Complete marks the progress as done successfully.
func (pp *PrettyProgress) Complete() {
	pp.SetColor(ProgressColorGreen)
	if pp.tracker != nil {
		pp.tracker.MarkAsDone()
	}
	pp.stop()
}

// Fail marks the progress as failed with the error message.
func (pp *PrettyProgress) Fail(err error) {
	pp.SetColor(ProgressColorRed)
	if pp.tracker != nil {
		pp.tracker.UpdateMessage(err.Error())
		pp.tracker.MarkAsErrored()
	}
	pp.stop()
}

// IsActive reports if progress output is running.
func (pp *PrettyProgress) IsActive() bool {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	return pp.active
}
