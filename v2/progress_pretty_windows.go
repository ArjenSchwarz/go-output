//go:build windows

package output

import (
	"os"
)

func (p *prettyProgress) setupSignals() {
	// Windows doesn't support SIGWINCH, so no signal setup needed
}

func (p *prettyProgress) isSIGWINCH(os.Signal) bool {
	return false
}
