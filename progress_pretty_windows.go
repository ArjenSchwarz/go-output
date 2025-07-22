//go:build windows

package format

import (
	"os"
	"os/signal"
	"syscall"
)

func (pp *PrettyProgress) setupSignals() {
	signal.Notify(pp.signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
}

func (pp *PrettyProgress) isSIGWINCH(os.Signal) bool {
	return false
}
