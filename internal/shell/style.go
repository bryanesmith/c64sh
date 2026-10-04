package shell

import "github.com/bryanesmith/c64sh/internal/interp"

// style is how the shell colors its own text in a terminal: input for
// typing, ready for the banner and READY., and fail for errors. Each is
// an escape code; the zero style colors nothing.
type style struct {
	input, ready, fail string
	end                func() string // ends a styled span
}

// newStyle returns the shell's styles from its settings. A span ends by
// resetting the terminal and putting back the color and reverse video the
// program has left set, so they stay in effect as on a C64.
//
// @spec SHELL-STYLE-004
func newStyle(in *interp.Interp, set settings) style {
	return style{
		input: set.input,
		ready: set.ready,
		fail:  set.fail,
		end:   func() string { return "\x1b[0m" + in.ScreenState() },
	}
}

// paint returns text as a span styled with sgr, or text alone if sgr is
// empty.
//
// @spec SHELL-STYLE-005
func (st style) paint(sgr, text string) string {
	if sgr == "" {
		return text
	}
	return sgr + text + st.end()
}
