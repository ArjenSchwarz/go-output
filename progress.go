// Copyright (c) 2021 Arjen Schwarz
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package format contains helpers for rendering output in different
// formats.  Progress support is provided through the Progress
// interface with both a real terminal implementation and a no-op
// fallback.  All progress implementations are safe for concurrent use
// by multiple goroutines.
//
// Basic usage:
//
//	settings := format.NewOutputSettings()
//	settings.SetOutputFormat("table")
//	p := format.NewProgress(settings)
//	p.SetTotal(3)
//	for i := 0; i < 3; i++ {
//	    p.Increment(1)
//	}
//	p.Complete()
package format

import (
	"context"
	"sync"
)

// ProgressColor defines the color used for a progress indicator.
type ProgressColor int

const (
	// ProgressColorDefault is the neutral color.
	ProgressColorDefault ProgressColor = iota
	// ProgressColorGreen indicates a success state.
	ProgressColorGreen
	// ProgressColorRed indicates an error state.
	ProgressColorRed
	// ProgressColorYellow indicates a warning state.
	ProgressColorYellow
	// ProgressColorBlue indicates an informational state.
	ProgressColorBlue
)

// ProgressOptions configures the behaviour of a Progress implementation.
type ProgressOptions struct {
	// Color specifies the default color for the progress indicator.
	Color ProgressColor
	// Status sets an optional initial status message.
	Status string
	// TrackerLength sets the width of the progress bar. Values <= 0
	// use a sensible default.
	TrackerLength int
}

// Progress describes an abstract progress indicator. Implementations
// must be safe for concurrent use by multiple goroutines.
type Progress interface {
	// SetTotal sets the total number of steps for this progress.
	SetTotal(total int)
	// SetCurrent sets the current progress value.
	SetCurrent(current int)
	// Increment increases the current value by n.
	Increment(n int)
	// SetStatus updates the displayed status message.
	SetStatus(status string)
	// SetColor changes the color of the progress indicator.
	SetColor(color ProgressColor)
	// Complete marks the progress as finished successfully.
	Complete()
	// Fail marks the progress as failed with the provided error.
	Fail(err error)
	// IsActive returns true when the progress indicator is running.
	IsActive() bool
	// SetContext sets a context used to cancel the progress.
	SetContext(ctx context.Context)
}

var (
	activeProgress Progress
	activeMutex    sync.Mutex
)

// registerActiveProgress stores the provided Progress implementation so it can
// be cleaned up before other output is written. Only a single progress can be
// active at any given time.
func registerActiveProgress(p Progress) {
	stopActiveProgress()
	activeMutex.Lock()
	activeProgress = p
	activeMutex.Unlock()
}

// stopActiveProgress stops and clears the currently active progress if it is a
// PrettyProgress instance.
func stopActiveProgress() {
	activeMutex.Lock()
	if pp, ok := activeProgress.(*PrettyProgress); ok {
		pp.stop()
	}
	activeProgress = nil
	activeMutex.Unlock()
}
