//go:build !windows

package output

import (
	"os"
	"os/signal"
	"syscall"
)

func (p *prettyProgress) setupSignals() {
	signal.Notify(p.signals, syscall.SIGWINCH)
}

func (p *prettyProgress) isSIGWINCH(sig os.Signal) bool {
	return sig == syscall.SIGWINCH
}
