package shell

import (
	"io"

	"github.com/bryanesmith/c64sh/internal/interp"
)

// style is how the shell colors its own text in a terminal: input for
// typing, ready for the banner and READY., and fail for errors. Each is
// an escape code; the zero style colors nothing.
type style struct {
	input, ready, fail string
	end                func() string // ends a styled span
}

// newStyle returns the shell's styles, in the terminal's own theme
// colors. A span ends by resetting the terminal and putting back the
// color and reverse video the program has left set, so they stay in
// effect as on a C64.
//
// @spec SHELL-STYLE-004
func newStyle(in *interp.Interp) style {
	return style{
		input: "\x1b[36m", // cyan
		ready: "\x1b[32m", // green
		fail:  "\x1b[31m", // red
		end:   func() string { return "\x1b[0m" + in.ScreenState() },
	}
}

// paint returns text as a span styled with sgr, or text alone if sgr is
// empty.
func (st style) paint(sgr, text string) string {
	if sgr == "" {
		return text
	}
	return sgr + text + st.end()
}

// styled reports whether the shell styles its own text: when stderr, where
// it writes that text, is a terminal, and NO_COLOR is not set.
//
// @spec SHELL-STYLE-001
func styled(stderr io.Writer, noColor bool) bool {
	return isTerminal(stderr) && !noColor
}
