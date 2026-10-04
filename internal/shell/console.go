package shell

import (
	"bufio"
	"io"

	"time"
	"unicode/utf8"

	"github.com/bryanesmith/c64sh/internal/interp"
)

// C64 character codes for the keys a terminal sends differently.
const (
	c64Return = "\r"     // RETURN, CHR$(13)
	c64Delete = "\x14"   // DEL, CHR$(20)
	c64Up     = "\u0091" // cursor up, CHR$(145)
	c64Down   = "\u0011" // cursor down, CHR$(17)
	c64Right  = "\u001d" // cursor right, CHR$(29)
	c64Left   = "\u009d" // cursor left, CHR$(157)
)

// lineConsole reads INPUT and GET from input that is not a terminal: a
// pipe or a file, possibly the script itself.
//
// @spec SHELL-KEY-002
type lineConsole struct {
	r *bufio.Reader
}

func (c *lineConsole) ReadLine(stop func() bool) (string, bool, error) {
	line, err := (&plainReader{r: c.r}).ReadLine()
	return line, false, err
}

func (c *lineConsole) ReadKey(stop func() bool) (string, error) {
	r, _, err := c.r.ReadRune()
	if err != nil {
		return "", err
	}
	switch r {
	case '\r':
		if next, _, err := c.r.ReadRune(); err == nil && next != '\n' {
			c.r.UnreadRune()
		}
		return c64Return, nil
	case '\n':
		return c64Return, nil
	}
	return string(r), nil
}

// ttyConsole reads INPUT and GET from a terminal in program mode, where a
// read returns at once, with no bytes (or io.EOF, as os.File reports it)
// when no key is waiting.
type ttyConsole struct {
	in    io.Reader
	echo  io.Writer // where ReadLine echoes typing
	style style     // how echoed typing is styled
	buf   []byte    // bytes read but not yet returned as keys
}

// pollInterval is how long ReadLine waits between checks for a key.
const pollInterval = 10 * time.Millisecond

// ReadKey returns the next key, or "" if none is waiting.
//
// @spec SHELL-KEY-004
func (c *ttyConsole) ReadKey(stop func() bool) (string, error) {
	for {
		if len(c.buf) == 0 {
			var b [64]byte
			n, err := c.in.Read(b[:])
			if err != nil && err != io.EOF {
				return "", err
			}
			if n == 0 {
				return "", nil
			}
			c.buf = append(c.buf, b[:n]...)
		}
		if key, ok := c.key(); ok {
			return key, nil
		}
	}
}

// key removes the first key from buf and returns it as a C64 character;
// ok is false for an ignored escape sequence.
func (c *ttyConsole) key() (string, bool) {
	b := c.buf
	if b[0] == 0x1b {
		// An escape sequence: ESC [ or ESC O, parameters, a final byte.
		n := 1
		if len(b) > 1 && (b[1] == '[' || b[1] == 'O') {
			n = 2
			for n < len(b) && (b[n] < 0x40 || b[n] > 0x7e) {
				n++
			}
			n = min(n+1, len(b))
		}
		seq := string(b[:n])
		c.buf = b[n:]
		switch seq {
		case "\x1b[A", "\x1bOA":
			return c64Up, true
		case "\x1b[B", "\x1bOB":
			return c64Down, true
		case "\x1b[C", "\x1bOC":
			return c64Right, true
		case "\x1b[D", "\x1bOD":
			return c64Left, true
		}
		return "", false
	}
	r, size := utf8.DecodeRune(b)
	c.buf = b[size:]
	switch r {
	case '\r', '\n':
		return c64Return, true
	case 0x7f, '\b':
		return c64Delete, true
	}
	return string(r), true
}

// ReadLine reads keys until Return, echoing typing and handling
// Backspace, and returns interp.ErrInterrupted if stop becomes true while
// it waits.
//
// @spec SHELL-KEY-005, SHELL-STYLE-003
func (c *ttyConsole) ReadLine(stop func() bool) (string, bool, error) {
	var line []rune
	for {
		key, err := c.ReadKey(stop)
		if err != nil {
			return "", false, err
		}
		switch {
		case key == "":
			if stop() {
				return "", false, interp.ErrInterrupted
			}
			time.Sleep(pollInterval)
		case key == c64Return:
			return string(line), true, nil
		case key == c64Delete:
			if len(line) > 0 {
				line = line[:len(line)-1]
				io.WriteString(c.echo, "\b \b")
			}
		default:
			if r, _ := utf8.DecodeRuneInString(key); r >= ' ' && (r < 0x80 || r > 0x9f) {
				line = append(line, r)
				io.WriteString(c.echo, c.style.paint(c.style.input, key))
			}
		}
	}
}
