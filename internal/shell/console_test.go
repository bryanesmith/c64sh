package shell

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/interp"
)

func never() bool { return false }

// @spec SHELL-KEY-002
func TestLineConsole(t *testing.T) {
	c := &lineConsole{r: bufio.NewReader(strings.NewReader("ALICE\r\nBOB\n"))}
	for _, want := range []string{"ALICE", "BOB"} {
		if got, echoed, err := c.ReadLine(never); got != want || echoed || err != nil {
			t.Errorf("ReadLine = %q, %v, %v; want %q, false, nil", got, echoed, err, want)
		}
	}
	if _, _, err := c.ReadLine(never); err != io.EOF {
		t.Errorf("ReadLine at end: %v, want io.EOF", err)
	}

	c = &lineConsole{r: bufio.NewReader(strings.NewReader("AB\nC\r\nπ"))}
	for _, want := range []string{"A", "B", "\r", "C", "\r", "π"} {
		if got, err := c.ReadKey(never); got != want || err != nil {
			t.Errorf("ReadKey = %q, %v; want %q, nil", got, err, want)
		}
	}
	if _, err := c.ReadKey(never); err != io.EOF {
		t.Errorf("ReadKey at end: %v, want io.EOF", err)
	}
}

// chunkReader returns one chunk per Read, and then (0, io.EOF) whenever
// it has none, as a terminal in program mode does when no key is waiting.
type chunkReader struct{ chunks []string }

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[0])
	r.chunks = r.chunks[1:]
	return n, nil
}

// @spec SHELL-KEY-004
func TestTerminalConsoleKeys(t *testing.T) {
	c := &ttyConsole{in: &chunkReader{chunks: []string{"A", "\r", "\n", "\x7f", "\b", "\x1b[A", "\x1b[B\x1b[C", "\x1b[D", "\x1b[5~X", "é"}}}
	want := []string{"A", "\r", "\r", "\x14", "\x14", "\u0091", "\u0011", "\u001d", "\u009d", "X", "é", ""}
	for _, w := range want {
		if got, err := c.ReadKey(never); got != w || err != nil {
			t.Errorf("ReadKey = %q, %v; want %q, nil", got, err, w)
		}
	}
}

// @spec SHELL-KEY-005
func TestTerminalConsoleLine(t *testing.T) {
	var echo strings.Builder
	c := &ttyConsole{in: &chunkReader{chunks: []string{"AB", "\x7f", "C\x1b[A", "\r", "NEXT"}}, echo: &echo}
	line, echoed, err := c.ReadLine(never)
	if line != "AC" || !echoed || err != nil {
		t.Errorf("ReadLine = %q, %v, %v; want \"AC\", true, nil", line, echoed, err)
	}
	if echo.String() != "AB\b \bC" {
		t.Errorf("echo %q, want %q", echo.String(), "AB\b \bC")
	}
	if k, _ := c.ReadKey(never); k != "N" {
		t.Errorf("key after the line %q, want \"N\"", k)
	}

	calls := 0
	stop := func() bool { calls++; return calls > 3 }
	c = &ttyConsole{in: &chunkReader{}, echo: io.Discard}
	if _, _, err := c.ReadLine(stop); !errors.Is(err, interp.ErrInterrupted) {
		t.Errorf("ReadLine while interrupted: %v, want ErrInterrupted", err)
	}
	c = &ttyConsole{in: &chunkReader{chunks: []string{"\x7f\x7fX\r"}}, echo: io.Discard}
	if line, _, _ := c.ReadLine(never); line != "X" {
		t.Errorf("Backspace on an empty line: %q, want \"X\"", line)
	}
}

// @spec SHELL-KEY-003
func TestProgramModeAroundEachLine(t *testing.T) {
	var events []string
	s := &session{
		stderr: io.Discard,
		interp: interp.New(io.Discard),
		programMode: func() (func(), error) {
			events = append(events, "enter")
			return func() { events = append(events, "restore") }, nil
		},
	}
	s.run(&plainReader{r: bufio.NewReader(strings.NewReader("PRINT 1\n10 REM\nPRINT 2\n"))}, "stdin")
	// Storing line 10 executes nothing; the final RUN executes.
	if got := strings.Join(events, " "); got != "enter restore enter restore enter restore" {
		t.Errorf("program mode %q, want it entered and restored around each of 3 executed lines", got)
	}
}
