package format

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mattn/go-isatty"
)

// PrettyProgress wraps go-pretty's progress.Writer to implement the Progress
// interface. All methods are guarded by a mutex making the type safe for
// concurrent use by multiple goroutines.
type PrettyProgress struct {
	writer  progress.Writer
	tracker *progress.Tracker
	options ProgressOptions
	active  bool
	mutex   sync.Mutex
	ctx     context.Context
	signals chan os.Signal
}

// newPrettyProgress creates a new PrettyProgress instance.
func newPrettyProgress(settings *OutputSettings) *PrettyProgress {
	pp := &PrettyProgress{ctx: context.Background()}
	pp.tracker = &progress.Tracker{}
	if settings != nil {
		pp.options = settings.ProgressOptions
		pp.tracker.Message = settings.ProgressOptions.Status
	}
	if isatty.IsTerminal(os.Stdout.Fd()) {
		w := progress.NewWriter()
		w.SetAutoStop(false)
		length := pp.options.TrackerLength
		if length <= 0 {
			length = 40
		}
		w.SetTrackerLength(length)
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
	stopActiveProgress() // Ensure any previously active progress is stopped
	registerActiveProgress(pp)
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
	pp.signals = make(chan os.Signal, 1)
	signal.Notify(pp.signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGWINCH)
	go func() {
		for sig := range pp.signals {
			if sig == syscall.SIGWINCH {
				// trigger re-render on resize
				if pp.writer != nil {
					pp.writer.Render()
				}
				continue
			}
			pp.stop()
		}
	}()
	if pp.ctx != nil {
		go func() {
			<-pp.ctx.Done()
			pp.stop()
		}()
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// ensure progress stops on panic
				_ = r
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
	if pp.signals != nil {
		signal.Stop(pp.signals)
		close(pp.signals)
		pp.signals = nil
	}
	pp.active = false
}

// SetTotal sets the expected total value.
func (pp *PrettyProgress) SetTotal(total int) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	if pp.tracker != nil {
		pp.tracker.UpdateTotal(int64(total))
	}
}

// SetCurrent sets the current progress value.
func (pp *PrettyProgress) SetCurrent(current int) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	if pp.tracker != nil {
		pp.tracker.SetValue(int64(current))
	}
}

// Increment increases the progress by n.
func (pp *PrettyProgress) Increment(n int) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	if pp.tracker != nil {
		pp.tracker.Increment(int64(n))
	}
}

// SetStatus updates the status message.
func (pp *PrettyProgress) SetStatus(status string) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	if pp.tracker != nil {
		pp.tracker.UpdateMessage(status)
	}
}

// SetColor changes the progress bar color.
func (pp *PrettyProgress) SetColor(color ProgressColor) {
	pp.mutex.Lock()
	pp.options.Color = color
	if pp.writer == nil {
		pp.mutex.Unlock()
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
	pp.mutex.Unlock()
}

// Complete marks the progress as done successfully.
func (pp *PrettyProgress) Complete() {
	pp.SetColor(ProgressColorGreen)
	pp.mutex.Lock()
	if pp.tracker != nil {
		pp.tracker.MarkAsDone()
	}
	pp.mutex.Unlock()
	pp.stop()
}

// Fail marks the progress as failed with the error message.
func (pp *PrettyProgress) Fail(err error) {
	pp.SetColor(ProgressColorRed)
	pp.mutex.Lock()
	if pp.tracker != nil {
		pp.tracker.UpdateMessage(err.Error())
		pp.tracker.MarkAsErrored()
	}
	pp.mutex.Unlock()
	pp.stop()
}

// IsActive reports if progress output is running.
func (pp *PrettyProgress) IsActive() bool {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	return pp.active
}

// SetContext sets a context that, when cancelled, stops the progress.
func (pp *PrettyProgress) SetContext(ctx context.Context) {
	pp.mutex.Lock()
	pp.ctx = ctx
	pp.mutex.Unlock()
}
