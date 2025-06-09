// Copyright (c) 2021 Arjen Schwarz
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package format contains helpers for rendering output in different
// formats. This file defines the Progress interface and related types
// used to display progress information.
package format

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
}

// Progress describes an abstract progress indicator.
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
}
