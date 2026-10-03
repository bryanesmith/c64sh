//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd)

package shell

import (
	"errors"
	"os"
)

// terminalProgramMode is unavailable here; INPUT and GET then read the
// terminal with its usual line buffering.
func terminalProgramMode(f *os.File) func() (func(), error) {
	return func() (func(), error) { return nil, errors.ErrUnsupported }
}
