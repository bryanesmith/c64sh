//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package shell

import (
	"os"

	"golang.org/x/sys/unix"
)

// terminalProgramMode returns a function that switches the terminal open
// as f to program mode, for INPUT and GET: no line buffering and no echo,
// with reads returning at once, while signal keys (Ctrl-C) and output
// processing stay on. It returns a function restoring the previous mode.
//
// @spec SHELL-KEY-003
func terminalProgramMode(f *os.File) func() (func(), error) {
	fd := int(f.Fd())
	return func() (func(), error) {
		t, err := unix.IoctlGetTermios(fd, ioctlGetTermios)
		if err != nil {
			return nil, err
		}
		old := *t
		t.Lflag &^= unix.ICANON | unix.ECHO
		t.Cc[unix.VMIN] = 0
		t.Cc[unix.VTIME] = 0
		if err := unix.IoctlSetTermios(fd, ioctlSetTermios, t); err != nil {
			return nil, err
		}
		return func() { unix.IoctlSetTermios(fd, ioctlSetTermios, &old) }, nil
	}
}
