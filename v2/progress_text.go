package output

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// textProgress implements Progress with text-based output
type textProgress struct {
	config    *ProgressConfig
	mu        sync.RWMutex
	total     int
	current   int
	status    string
	startTime time.Time
	lastDraw  time.Time
	completed bool
	failed    bool
	err       error
	ctx       context.Context
	cancel    context.CancelFunc
}

// SetTotal sets the total number of units to be processed
func (p *textProgress) SetTotal(total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = total
	p.draw()
}

// SetCurrent sets the current progress count
func (p *textProgress) SetCurrent(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = current
	p.draw()
}

// Increment advances the progress by the specified delta
func (p *textProgress) Increment(delta int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current += delta
	p.draw()
}

// SetStatus sets a descriptive status message
func (p *textProgress) SetStatus(status string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = status
	p.draw()
}

// Complete marks the operation as successfully completed
func (p *textProgress) Complete() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completed = true
	p.current = p.total
	p.drawFinal()
}

// Fail marks the operation as failed with the given error
func (p *textProgress) Fail(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failed = true
	p.err = err
	p.drawFinal()
}

// Close cleans up any resources used by the progress indicator
func (p *textProgress) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}

	if !p.completed && !p.failed {
		p.drawFinal()
	}

	return nil
}

// draw renders the current progress state (must be called with lock held)
func (p *textProgress) draw() {
	now := time.Now()
	if now.Sub(p.lastDraw) < p.config.UpdateInterval && !p.completed && !p.failed {
		return
	}
	p.lastDraw = now

	var line string
	if p.config.Template != "" {
		line = p.renderTemplate()
	} else {
		line = p.renderDefault()
	}

	// Clear line and write progress
	_, _ = fmt.Fprintf(p.config.Writer, "\r%s\r%s",
		fmt.Sprintf("%*s", len(line), ""), line)
}

// drawFinal renders the final progress state (must be called with lock held)
func (p *textProgress) drawFinal() {
	var line string
	if p.config.Template != "" {
		line = p.renderTemplate()
	} else {
		line = p.renderDefault()
	}

	// Write final state with newline
	_, _ = fmt.Fprintf(p.config.Writer, "\r%s\r%s\n",
		fmt.Sprintf("%*s", len(line), ""), line)
}

// renderDefault creates the default progress display
func (p *textProgress) renderDefault() string {
	var parts []string

	// Add prefix
	if p.config.Prefix != "" {
		parts = append(parts, p.config.Prefix)
	}

	// Add progress bar
	if p.total > 0 {
		progress := float64(p.current) / float64(p.total)
		barWidth := p.config.Width
		filled := int(progress * float64(barWidth))

		bar := fmt.Sprintf("[%s%s]",
			strings.Repeat("=", filled),
			strings.Repeat(" ", barWidth-filled))
		parts = append(parts, bar)

		// Add percentage
		if p.config.ShowPercentage {
			parts = append(parts, fmt.Sprintf("%.1f%%", progress*100))
		}

		// Add count
		parts = append(parts, fmt.Sprintf("(%d/%d)", p.current, p.total))
	}

	// Add ETA
	if p.config.ShowETA && p.total > 0 && p.current > 0 {
		elapsed := time.Since(p.startTime)
		rate := float64(p.current) / elapsed.Seconds()
		remaining := float64(p.total-p.current) / rate
		eta := time.Duration(remaining) * time.Second
		parts = append(parts, fmt.Sprintf("ETA: %v", eta.Round(time.Second)))
	}

	// Add rate
	if p.config.ShowRate && p.current > 0 {
		elapsed := time.Since(p.startTime)
		rate := float64(p.current) / elapsed.Seconds()
		parts = append(parts, fmt.Sprintf("%.1f/s", rate))
	}

	// Add status
	if p.status != "" {
		parts = append(parts, p.status)
	}

	// Add status indicators
	if p.completed {
		parts = append(parts, "✓ Complete")
	} else if p.failed {
		parts = append(parts, fmt.Sprintf("✗ Failed: %v", p.err))
	}

	// Add suffix
	if p.config.Suffix != "" {
		parts = append(parts, p.config.Suffix)
	}

	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}

	// Join all parts with spaces
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += " " + parts[i]
	}
	return result
}

// renderTemplate renders progress using a custom template
func (p *textProgress) renderTemplate() string {
	// This would implement template rendering - simplified for now
	return p.renderDefault()
}

// v1 compatibility methods for textProgress

// SetColor changes the progress color (v1 compatibility)
func (p *textProgress) SetColor(color ProgressColor) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config.Color = color
}

// IsActive returns true when the progress indicator is running (v1 compatibility)
func (p *textProgress) IsActive() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.completed && !p.failed
}

// SetContext sets a context that, when cancelled, stops the progress (v1 compatibility)
func (p *textProgress) SetContext(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ctx = ctx

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
			if !p.completed && !p.failed {
				p.failed = true
				p.err = p.ctx.Err()
				p.drawFinal()
			}
			p.mu.Unlock()
		}()
	}
}
