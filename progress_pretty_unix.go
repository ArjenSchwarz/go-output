//go:build !windows

package format

import (
	"os"
	"os/signal"
	"syscall"
)

func (pp *PrettyProgress) setupSignals() {
	signal.Notify(pp.signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGWINCH)
}

func (pp *PrettyProgress) isSIGWINCH(sig os.Signal) bool {
	return sig == syscall.SIGWINCH
}
