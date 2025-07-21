package output

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/mattn/go-isatty"
)

// prettyProgress implements Progress using go-pretty for professional progress bars
type prettyProgress struct {
	config    *ProgressConfig
	mu        sync.RWMutex
	writer    progress.Writer
	tracker   *progress.Tracker
	total     int
	current   int
	status    string
	startTime time.Time
	active    bool
	completed bool
	failed    bool
	err       error
	ctx       context.Context
	cancel    context.CancelFunc
	signals   chan os.Signal
}

// NewPrettyProgress creates a new professional progress indicator using go-pretty
func NewPrettyProgress(opts ...ProgressOption) Progress {
	config := defaultProgressConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Only create pretty progress if we have a TTY
	if !isatty.IsTerminal(os.Stderr.Fd()) && !isatty.IsCygwinTerminal(os.Stderr.Fd()) {
		// Fall back to text progress if not in terminal
		return NewProgress(opts...)
	}

	p := &prettyProgress{
		config:    config,
		startTime: time.Now(),
		active:    true,
		signals:   make(chan os.Signal, 1),
	}

	// Setup go-pretty progress writer
	p.writer = progress.NewWriter()
	p.writer.SetOutputWriter(config.Writer)
	p.writer.SetUpdateFrequency(config.UpdateInterval)
	p.writer.SetStyle(progress.StyleDefault)
	p.writer.SetNumTrackersExpected(1)

	// Enable colors if supported
	if isatty.IsTerminal(os.Stderr.Fd()) {
		p.writer.Style().Colors = progress.StyleColorsExample
	}

	// Setup signal handling for terminal resize
	signal.Notify(p.signals, syscall.SIGWINCH)
	go p.handleSignals()

	// Start the progress writer
	go p.writer.Render()

	return p
}

// handleSignals processes terminal resize and shutdown signals
func (p *prettyProgress) handleSignals() {
	for {
		select {
		case sig := <-p.signals:
			if sig == syscall.SIGWINCH {
				// Terminal was resized - go-pretty handles this automatically
				continue
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// SetTotal sets the total number of units to be processed
func (p *prettyProgress) SetTotal(total int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.total = total

	if p.tracker == nil {
		// Create tracker with initial configuration
		p.tracker = &progress.Tracker{
			Message: p.getStatusMessage(),
			Total:   int64(total),
			Units:   progress.UnitsDefault,
		}
		p.writer.AppendTracker(p.tracker)
	} else {
		// Update existing tracker
		p.tracker.Total = int64(total)
		p.tracker.UpdateMessage(p.getStatusMessage())
	}
}

// SetCurrent sets the current progress count
func (p *prettyProgress) SetCurrent(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active || p.tracker == nil {
		return
	}

	p.current = current
	p.tracker.SetValue(int64(current))
	p.tracker.UpdateMessage(p.getStatusMessage())
}

// Increment advances the progress by the specified delta
func (p *prettyProgress) Increment(delta int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active || p.tracker == nil {
		return
	}

	p.current += delta
	p.tracker.Increment(int64(delta))
	p.tracker.UpdateMessage(p.getStatusMessage())
}

// SetStatus sets a descriptive status message
func (p *prettyProgress) SetStatus(status string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.status = status
	if p.tracker != nil {
		p.tracker.UpdateMessage(p.getStatusMessage())
	}
}

// Complete marks the operation as successfully completed
func (p *prettyProgress) Complete() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active || p.tracker == nil {
		return
	}

	p.completed = true
	p.active = false
	p.current = p.total

	// Mark tracker as done
	p.tracker.MarkAsDone()
	p.tracker.UpdateMessage(p.getStatusMessage())
}

// Fail marks the operation as failed with the given error
func (p *prettyProgress) Fail(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active || p.tracker == nil {
		return
	}

	p.failed = true
	p.active = false
	p.err = err

	// Mark tracker as errored
	p.tracker.MarkAsErrored()
	p.tracker.UpdateMessage(p.getStatusMessage())
}

// Close cleans up resources and stops the progress display
func (p *prettyProgress) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}

	if p.active && p.tracker != nil {
		p.tracker.MarkAsDone()
	}

	p.active = false

	// Stop signal handling
	signal.Stop(p.signals)
	close(p.signals)

	// Stop the writer
	p.writer.Stop()

	return nil
}

// getStatusMessage creates the display message (must be called with lock held)
func (p *prettyProgress) getStatusMessage() string {
	var parts []string

	// Add prefix if configured
	if p.config.Prefix != "" {
		parts = append(parts, p.config.Prefix)
	}

	// Add current status
	if p.status != "" {
		parts = append(parts, p.status)
	} else if p.config.Status != "" {
		parts = append(parts, p.config.Status)
	}

	// Add completion/failure status
	if p.completed {
		parts = append(parts, "✓ Complete")
	} else if p.failed {
		if p.err != nil {
			parts = append(parts, fmt.Sprintf("✗ Failed: %v", p.err))
		} else {
			parts = append(parts, "✗ Failed")
		}
	}

	// Add suffix if configured
	if p.config.Suffix != "" {
		parts = append(parts, p.config.Suffix)
	}

	if len(parts) == 0 {
		return "Processing..."
	}

	return strings.Join(parts, " ")
}

// v1 compatibility methods for prettyProgress

// SetColor changes the progress color (v1 compatibility)
func (p *prettyProgress) SetColor(color ProgressColor) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config.Color = color

	// Apply color to go-pretty progress bar
	if p.tracker != nil {
		switch color {
		case ProgressColorGreen:
			// Set success color theme
			p.writer.Style().Colors.Tracker = progress.StyleColorsExample.Tracker
		case ProgressColorRed:
			// Set error color theme
			p.writer.Style().Colors.Error = progress.StyleColorsExample.Error
		case ProgressColorYellow:
			// Set warning color theme
			p.writer.Style().Colors.Stats = progress.StyleColorsExample.Stats
		case ProgressColorBlue:
			// Set info color theme
			p.writer.Style().Colors.Message = progress.StyleColorsExample.Message
		default:
			// Use default colors
			p.writer.Style().Colors = progress.StyleColorsExample
		}
	}
}

// IsActive returns true when the progress indicator is running (v1 compatibility)
func (p *prettyProgress) IsActive() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.active && !p.completed && !p.failed
}

// SetContext sets a context that, when cancelled, stops the progress (v1 compatibility)
func (p *prettyProgress) SetContext(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Cancel any existing context
	if p.cancel != nil {
		p.cancel()
	}

	// Create new context with cancellation
	if ctx != nil {
		p.ctx, p.cancel = context.WithCancel(ctx)

		// Start goroutine to watch for cancellation
		go func() {
			<-p.ctx.Done()
			p.mu.Lock()
			if p.active && p.tracker != nil {
				p.failed = true
				p.active = false
				p.err = p.ctx.Err()
				p.tracker.MarkAsErrored()
				p.tracker.UpdateMessage(p.getStatusMessage())
			}
			p.mu.Unlock()
		}()
	}
}
