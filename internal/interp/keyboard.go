package interp

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// Console is where INPUT and GET read the keyboard. stop reports whether
// the interpreter has been interrupted; a method that is waiting returns
// ErrInterrupted when it becomes true.
type Console interface {
	// ReadLine reads one line, without its line terminator. echoed reports
	// whether the console has already shown what was typed, as a terminal
	// does; if not, the interpreter writes it. At the end of input it
	// returns io.EOF.
	ReadLine(stop func() bool) (line string, echoed bool, err error)
	// ReadKey returns the next key as a one-character string, or "" if no
	// key is waiting. At the end of input it returns io.EOF.
	ReadKey(stop func() bool) (key string, err error)
}

// ErrInterrupted is returned by a Console that stopped waiting because
// the interpreter was interrupted.
var ErrInterrupted = errors.New("interrupted")

// ErrEndOfInput is returned by Exec when INPUT or GET found the end of
// the console's input. It is not a BASIC error.
var ErrEndOfInput = errors.New("end of input")

// SetConsole sets where INPUT and GET read. Without one, they find the
// end of input.
func (in *Interp) SetConsole(c Console) {
	in.console = c
}

// noLine marks a BASIC error that is reported without a line number even
// in a running program.
type noLine struct{ err error }

func (e *noLine) Error() string { return e.err.Error() }
func (e *noLine) Unwrap() error { return e.err }

// execInput reads values for its variables as the ROM does ($ABBF,
// $AC0D): prompt and "? ", read a line, and take one value per variable,
// asking "?? " for more, starting again after "?REDO FROM START" when a
// value cannot be read, and saying "?EXTRA IGNORED" for any left over.
//
// @spec INTERP-081, INTERP-082, INTERP-083, INTERP-084, INTERP-085, INTERP-086, INTERP-087
func (in *Interp) execInput(s *ast.InputStmt) error {
	if s.File != nil {
		return in.execInputFile(s)
	}
	if in.cur.line == directLine {
		if err := in.write(s.Prompt); err != nil {
			return err
		}
		return &basicerr.Error{Kind: basicerr.IllegalDirect}
	}
	for {
		line, err := in.prompt(s.Prompt + "? ")
		if err != nil || line == "" {
			return err
		}
		redo, err := in.inputValues(s.Vars, line)
		if err != nil || !redo {
			return err
		}
		if err := in.write("?REDO FROM START\n"); err != nil {
			return err
		}
	}
}

// inputValues assigns the variables values read from line, reading more
// lines as needed. It reports whether a value could not be read.
func (in *Interp) inputValues(vars []*ast.VarRef, line string) (redo bool, err error) {
	i := 0
	for n, v := range vars {
		if n > 0 {
			i = skipSpaces(line, i)
			if i < len(line) && line[i] == ',' {
				i++
			} else {
				if line, err = in.prompt("?? "); err != nil {
					return false, err
				}
				i = 0
			}
		}
		var val value
		var ok bool
		if strings.HasSuffix(v.Name, "$") {
			val, i, ok = inputString(line, i)
		} else {
			var n float64
			n, i, ok = inputNumber(line, i)
			if ok {
				if val, err = inRange(n); err != nil {
					return false, err
				}
			}
		}
		if !ok {
			return true, nil
		}
		if err := in.assign(v, val); err != nil {
			return false, err
		}
	}
	if i = skipSpaces(line, i); i < len(line) {
		return false, in.write("?EXTRA IGNORED\n")
	}
	return false, nil
}

// prompt writes text, reads a line, and ends the output line, writing the
// line itself if the console did not echo it. Spaces at the end of the
// line are removed, as the C64's screen editor removes them.
func (in *Interp) prompt(text string) (string, error) {
	if err := in.write(text); err != nil {
		return "", err
	}
	if in.console == nil {
		return "", ErrEndOfInput
	}
	line, echoed, err := in.console.ReadLine(in.interrupted.Load)
	if err != nil {
		return "", in.consoleError(err)
	}
	line = strings.TrimRight(line, " ")
	if echoed {
		return line, in.write("\n")
	}
	return line, in.write(line + "\n")
}

// consoleError turns a console's error into the interpreter's.
//
// @spec INTERP-090
func (in *Interp) consoleError(err error) error {
	switch {
	case errors.Is(err, io.EOF):
		return ErrEndOfInput
	case errors.Is(err, ErrInterrupted):
		in.interrupted.Store(false)
		return &basicerr.Error{Kind: basicerr.Break}
	default:
		return err
	}
}

// inputString reads a string value at i: a quoted one, or the text up to
// the next "," or ":". It reports whether a valid terminator follows.
func inputString(line string, i int) (value, int, bool) {
	i = skipSpaces(line, i)
	var s string
	if i < len(line) && line[i] == '"' {
		end := strings.IndexByte(line[i+1:], '"')
		if end < 0 {
			return stringValue(line[i+1:]), len(line), true
		}
		s, i = line[i+1:i+1+end], i+end+2
	} else {
		end := strings.IndexAny(line[i:], ",:")
		if end < 0 {
			end = len(line) - i
		}
		s, i = line[i:i+end], i+end
	}
	return stringValue(s), i, terminated(line, i)
}

// inputNumber reads a number at i as the ROM does ($BCF3): an optional
// sign, digits with at most one ".", and an optional exponent, ignoring
// spaces; no digits is 0. It reports whether a valid terminator follows.
func inputNumber(line string, i int) (float64, int, bool) {
	var mant, exp strings.Builder
	neg, expNeg, point := false, false, false
	i = skipSpaces(line, i)
	if i < len(line) && (line[i] == '-' || line[i] == '+') {
		neg = line[i] == '-'
		i = skipSpaces(line, i+1)
	}
	for i < len(line) && (isDigit(line[i]) || line[i] == '.' && !point) {
		point = point || line[i] == '.'
		mant.WriteByte(line[i])
		i = skipSpaces(line, i+1)
	}
	if i < len(line) && line[i] == 'E' {
		i = skipSpaces(line, i+1)
		if i < len(line) && (line[i] == '-' || line[i] == '+') {
			expNeg = line[i] == '-'
			i = skipSpaces(line, i+1)
		}
		for i < len(line) && isDigit(line[i]) {
			exp.WriteByte(line[i])
			i = skipSpaces(line, i+1)
		}
	}
	text := "0" + mant.String() + "e"
	if expNeg {
		text += "-"
	}
	text += "0" + exp.String()
	n, _ := strconv.ParseFloat(text, 64) // too large gives +Inf, an OVERFLOW
	if neg {
		n = -n
	}
	return n, i, terminated(line, i)
}

// terminated reports whether, after spaces, line ends at i or continues
// with "," or ":".
func terminated(line string, i int) bool {
	i = skipSpaces(line, i)
	return i == len(line) || line[i] == ',' || line[i] == ':'
}

func skipSpaces(s string, i int) int {
	for i < len(s) && s[i] == ' ' {
		i++
	}
	return i
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

// execGet reads one key for each variable ($AB7B).
//
// @spec INTERP-081, INTERP-088, INTERP-089
func (in *Interp) execGet(s *ast.GetStmt) error {
	if in.cur.line == directLine {
		return &basicerr.Error{Kind: basicerr.IllegalDirect}
	}
	var f *ioFile
	if s.File != nil {
		var err error
		if f, err = in.inputFile(s.File); err != nil {
			return err
		}
		defer in.endCmd()
	}
	for _, v := range s.Vars {
		var key string
		var err error
		switch {
		case f != nil:
			key, err = in.readFileKey(f)
		case in.console == nil:
			err = ErrEndOfInput
		default:
			key, err = in.console.ReadKey(in.interrupted.Load)
			if err != nil {
				err = in.consoleError(err)
			}
		}
		if err != nil {
			return err
		}
		val := stringValue(key)
		if !strings.HasSuffix(v.Name, "$") {
			switch {
			case key == "" || strings.Contains(" .+-E", key):
				val = numberValue(0)
			case len(key) == 1 && isDigit(key[0]):
				val = numberValue(float64(key[0] - '0'))
			default:
				// The ROM reports this as if in direct mode ($AB53).
				return &noLine{&basicerr.Error{Kind: basicerr.Syntax}}
			}
		}
		if err := in.assign(v, val); err != nil {
			return err
		}
	}
	return nil
}
