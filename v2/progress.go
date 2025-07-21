package output

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/mattn/go-isatty"
)

// ProgressColor defines the color used for a progress indicator (v1 compatibility)
type ProgressColor int

const (
	// ProgressColorDefault is the neutral color
	ProgressColorDefault ProgressColor = iota
	// ProgressColorGreen indicates a success state
	ProgressColorGreen
	// ProgressColorRed indicates an error state
	ProgressColorRed
	// ProgressColorYellow indicates a warning state
	ProgressColorYellow
	// ProgressColorBlue indicates an informational state
	ProgressColorBlue
)

// Progress provides progress indication for long operations with full v1 compatibility
type Progress interface {
	// Core progress methods
	SetTotal(total int)
	SetCurrent(current int)
	Increment(delta int)
	SetStatus(status string)
	Complete()
	Fail(err error)

	// v1 compatibility methods
	SetColor(color ProgressColor)
	IsActive() bool
	SetContext(ctx context.Context)

	// v2 enhancements
	Close() error
}

// ProgressConfig holds configuration for progress indicators (v1 compatible)
type ProgressConfig struct {
	// v1 compatibility options
	Color         ProgressColor
	Status        string
	TrackerLength int

	// Output writer for progress display (defaults to os.Stderr)
	Writer io.Writer

	// UpdateInterval controls how often the progress is refreshed (defaults to 100ms)
	UpdateInterval time.Duration

	// ShowPercentage controls whether to show percentage complete
	ShowPercentage bool

	// ShowETA controls whether to estimate time to completion
	ShowETA bool

	// ShowRate controls whether to show processing rate
	ShowRate bool

	// Template for progress display format
	Template string

	// Width of the progress bar (for bar-style progress)
	Width int

	// Prefix to show before the progress indicator
	Prefix string

	// Suffix to show after the progress indicator
	Suffix string
}

// ProgressOption configures a ProgressConfig
type ProgressOption func(*ProgressConfig)

// WithProgressWriter sets the output writer for progress display
func WithProgressWriter(w io.Writer) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.Writer = w
	}
}

// WithUpdateInterval sets the progress refresh interval
func WithUpdateInterval(interval time.Duration) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.UpdateInterval = interval
	}
}

// WithPercentage enables or disables percentage display
func WithPercentage(show bool) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.ShowPercentage = show
	}
}

// WithETA enables or disables estimated time to completion
func WithETA(show bool) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.ShowETA = show
	}
}

// WithRate enables or disables processing rate display
func WithRate(show bool) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.ShowRate = show
	}
}

// WithTemplate sets a custom progress display template
func WithTemplate(template string) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.Template = template
	}
}

// WithWidth sets the progress bar width
func WithWidth(width int) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.Width = width
	}
}

// WithPrefix sets a prefix for the progress display
func WithPrefix(prefix string) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.Prefix = prefix
	}
}

// WithSuffix sets a suffix for the progress display
func WithSuffix(suffix string) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.Suffix = suffix
	}
}

// WithProgressColor sets the progress color (v1 compatibility)
func WithProgressColor(color ProgressColor) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.Color = color
	}
}

// WithProgressStatus sets an initial status message (v1 compatibility)
func WithProgressStatus(status string) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.Status = status
	}
}

// WithTrackerLength sets the width of the progress bar (v1 compatibility)
func WithTrackerLength(length int) ProgressOption {
	return func(pc *ProgressConfig) {
		pc.TrackerLength = length
		pc.Width = length // Also set Width for consistency
	}
}

// defaultProgressConfig returns the default configuration
func defaultProgressConfig() *ProgressConfig {
	return &ProgressConfig{
		// v1 compatibility defaults
		Color:         ProgressColorDefault,
		Status:        "",
		TrackerLength: 40,

		// v2 defaults
		Writer:         os.Stderr,
		UpdateInterval: 100 * time.Millisecond,
		ShowPercentage: true,
		ShowETA:        true,
		ShowRate:       false,
		Width:          40,
		Template:       "", // Will use default template based on progress type
	}
}

// NewProgress creates a new text-based progress indicator
func NewProgress(opts ...ProgressOption) Progress {
	config := defaultProgressConfig()
	for _, opt := range opts {
		opt(config)
	}

	return &textProgress{
		config:    config,
		startTime: time.Now(),
	}
}

// NewNoOpProgress creates a progress indicator that does nothing
func NewNoOpProgress() Progress {
	return &noOpProgress{}
}

// NewProgressForFormat creates a progress indicator appropriate for the given format
func NewProgressForFormat(format Format, opts ...ProgressOption) Progress {
	return NewProgressForFormatName(format.Name, opts...)
}

// NewProgressForFormatName creates a progress indicator appropriate for the given format name
func NewProgressForFormatName(formatName string, opts ...ProgressOption) Progress {
	switch formatName {
	case FormatJSON, FormatCSV, FormatYAML, FormatDOT:
		// Non-visual formats should use no-op progress
		return NewNoOpProgress()
	case FormatTable, FormatHTML, FormatMarkdown, FormatMermaid, FormatDrawIO:
		// Visual formats should use pretty progress when TTY available
		if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
			return NewPrettyProgress(opts...)
		}
		// Fall back to text progress when no TTY
		return NewProgress(opts...)
	default:
		// Unknown format - use text progress as safe default
		return NewProgress(opts...)
	}
}

// NewProgressForFormats creates a progress indicator appropriate for multiple output formats
func NewProgressForFormats(formats []Format, opts ...ProgressOption) Progress {
	if len(formats) == 0 {
		return NewNoOpProgress()
	}

	// Check if any formats require visual progress
	hasVisualFormat := false
	hasNonVisualFormat := false

	for _, format := range formats {
		switch format.Name {
		case FormatJSON, FormatCSV, FormatYAML, FormatDOT:
			hasNonVisualFormat = true
		case FormatTable, FormatHTML, FormatMarkdown, FormatMermaid, FormatDrawIO:
			hasVisualFormat = true
		}
	}

	// Decision logic:
	// - If only non-visual formats: use NoOpProgress
	// - If only visual formats: use appropriate progress for TTY
	// - If mixed formats: use visual progress if TTY available, otherwise NoOp

	switch {
	case hasVisualFormat && !hasNonVisualFormat:
		// Only visual formats
		if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
			return NewPrettyProgress(opts...)
		}
		return NewProgress(opts...)
	case hasNonVisualFormat && !hasVisualFormat:
		// Only non-visual formats
		return NewNoOpProgress()
	case hasVisualFormat && hasNonVisualFormat:
		// Mixed formats - be conservative and use visual progress only if TTY
		if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
			return NewPrettyProgress(opts...)
		}
		return NewNoOpProgress()
	}

	// Default case
	return NewNoOpProgress()
}

// NewAutoProgress creates a progress indicator with automatic format detection
// This is a convenience function that detects format from writer or uses default
func NewAutoProgress(opts ...ProgressOption) Progress {
	// Parse options to detect format hints
	config := defaultProgressConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Try to detect format from writer type or other hints
	// For now, default to pretty progress if TTY available
	if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
		return NewPrettyProgress(opts...)
	}

	return NewProgress(opts...)
}
