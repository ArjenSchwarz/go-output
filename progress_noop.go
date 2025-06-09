// Copyright (c) 2021 Arjen Schwarz
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.
// Package format provides output rendering helpers. This file implements a no-op Progress.

package format

import "log"

// NoOpProgress implements Progress but performs no operations.
type NoOpProgress struct {
	total   int
	current int
	options ProgressOptions
	debug   bool
}

// newNoOpProgress creates a new NoOpProgress instance.
func newNoOpProgress(settings *OutputSettings) *NoOpProgress {
	nop := &NoOpProgress{}
	if settings != nil {
		nop.options = ProgressOptions{}
	}
	return nop
}

func (nop *NoOpProgress) debugf(format string, args ...interface{}) {
	if nop.debug {
		log.Printf(format, args...)
	}
}

// SetTotal sets the total steps expected.
func (nop *NoOpProgress) SetTotal(total int) {
	nop.total = total
	nop.debugf("SetTotal %d", total)
}

// SetCurrent sets the current progress value.
func (nop *NoOpProgress) SetCurrent(current int) {
	nop.current = current
	nop.debugf("SetCurrent %d", current)
}

// Increment increases the progress by n.
func (nop *NoOpProgress) Increment(n int) {
	nop.current += n
	nop.debugf("Increment %d -> %d", n, nop.current)
}

// SetStatus updates the status message. It is ignored for NoOpProgress.
func (nop *NoOpProgress) SetStatus(status string) {
	nop.debugf("SetStatus %s", status)
}

// SetColor changes the color setting. It does not produce any output.
func (nop *NoOpProgress) SetColor(color ProgressColor) {
	nop.options.Color = color
	nop.debugf("SetColor %v", color)
}

// Complete marks the progress as finished successfully.
func (nop *NoOpProgress) Complete() {
	nop.debugf("Complete")
}

// Fail marks the progress as failed. The error is only logged when debug is enabled.
func (nop *NoOpProgress) Fail(err error) {
	if err != nil {
		nop.debugf("Fail: %v", err)
	} else {
		nop.debugf("Fail")
	}
}

// IsActive always returns false for NoOpProgress.
func (nop *NoOpProgress) IsActive() bool { return false }
