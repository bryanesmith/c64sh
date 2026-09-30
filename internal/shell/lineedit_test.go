package shell

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// Key sequences, as a terminal sends them.
const (
	keyUp        = "\x1b[A"
	keyDown      = "\x1b[B"
	keyLeft      = "\x1b[D"
	keyHome      = "\x1b[H"
	keyCtrlC     = "\x03"
	keyCtrlD     = "\x04"
	keyCtrlP     = "\x10"
	keyCtrlN     = "\x0e"
	keyBackspace = "\x7f"
	keyEnter     = "\r"
)

// fakeRaw counts raw-mode switches and restores.
type fakeRaw struct{ on, off int }

func (f *fakeRaw) raw() (func(), error) {
	f.on++
	return func() { f.off++ }, nil
}

func noSize() (int, int, bool) { return 0, 0, false }

// readAll reads lines from an editor fed keys until end of input.
func readAll(t *testing.T, keys string) []string {
	t.Helper()
	fr := &fakeRaw{}
	e := newEditorReader(strings.NewReader(keys), io.Discard, fr.raw, noSize, "", io.Discard)
	var lines []string
	for i := 0; i < 1000; i++ {
		line, err := e.ReadLine()
		if err == io.EOF {
			return lines
		}
		if err != nil {
			t.Fatalf("ReadLine error: %v", err)
		}
		lines = append(lines, line)
	}
	t.Fatal("editor did not reach end of input")
	return nil
}

func checkLines(t *testing.T, name, keys string, want ...string) {
	t.Helper()
	got := readAll(t, keys)
	if strings.Join(got, "|") != strings.Join(want, "|") || len(got) != len(want) {
		t.Errorf("%s: lines %q, want %q", name, got, want)
	}
}

// @spec SHELL-EDIT-001
func TestEditorEchoesToGivenWriter(t *testing.T) {
	var echo bytes.Buffer
	e := newEditorReader(strings.NewReader("HELLO"+keyEnter), &echo, (&fakeRaw{}).raw, noSize, "", io.Discard)
	if line, err := e.ReadLine(); err != nil || line != "HELLO" {
		t.Fatalf("ReadLine = %q, %v; want HELLO", line, err)
	}
	if !strings.Contains(echo.String(), "HELLO") {
		t.Errorf("echo %q does not contain the typed line", echo.String())
	}
}

// @spec SHELL-EDIT-001, SHELL-EDIT-002
func TestEditorWanted(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()
	cases := map[string]struct {
		cfg    Config
		stdin  io.Reader
		stderr io.Writer
	}{
		"script mode":              {Config{}, strings.NewReader(""), io.Discard},
		"interactive, test reader": {Config{Interactive: true}, strings.NewReader(""), io.Discard},
		"interactive, /dev/null":   {Config{Interactive: true}, devNull, devNull},
	}
	for name, c := range cases {
		if editorWanted(c.cfg, c.stdin, c.stderr) {
			t.Errorf("%s: editorWanted = true, want false", name)
		}
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no controlling terminal: %v", err)
	}
	defer tty.Close()
	if !editorWanted(Config{Interactive: true}, tty, tty) {
		t.Errorf("interactive with terminal stdin and stderr: editorWanted = false, want true")
	}
	if editorWanted(Config{Interactive: true}, tty, io.Discard) {
		t.Errorf("stderr not a terminal: editorWanted = true, want false")
	}
	if editorWanted(Config{Interactive: true, File: "x.bas"}, tty, tty) {
		t.Errorf("file input: editorWanted = true, want false")
	}
}

// @spec SHELL-EDIT-003
func TestHistoryNavigation(t *testing.T) {
	checkLines(t, "keyUp recalls the previous line", "A"+keyEnter+"B"+keyEnter+keyUp+keyEnter, "A", "B", "B")
	checkLines(t, "keyUp twice", "A"+keyEnter+"B"+keyEnter+keyUp+keyUp+keyEnter, "A", "B", "A")
	checkLines(t, "keyDown after keyUp", "A"+keyEnter+"B"+keyEnter+keyUp+keyUp+keyDown+keyEnter, "A", "B", "B")
	checkLines(t, "keyDown returns to the typed line", "A"+keyEnter+"XY"+keyUp+keyDown+keyEnter, "A", "XY")
	checkLines(t, "Ctrl-P and Ctrl-N", "A"+keyEnter+"B"+keyEnter+keyCtrlP+keyCtrlP+keyCtrlN+keyEnter, "A", "B", "B")
	checkLines(t, "keyUp past the oldest stays there", "A"+keyEnter+keyUp+keyUp+keyUp+keyEnter, "A", "A")
}

// @spec SHELL-EDIT-004
func TestHistoryContents(t *testing.T) {
	checkLines(t, "blank lines are not added", "A"+keyEnter+keyEnter+"  "+keyEnter+keyUp+keyEnter, "A", "", "  ", "A")
	h := &history{}
	for i := 0; i < 101; i++ {
		h.Add(strings.Repeat("x", i+1))
	}
	if h.Len() != 100 {
		t.Fatalf("history length %d after 101 adds, want 100", h.Len())
	}
	if got := h.At(0); got != strings.Repeat("x", 101) {
		t.Errorf("most recent entry has length %d, want 101", len(got))
	}
	if got := h.At(99); got != strings.Repeat("x", 2) {
		t.Errorf("oldest entry has length %d, want 2 (the first add dropped)", len(got))
	}
}

// @spec SHELL-EDIT-005
func TestCtrlCDiscardsLine(t *testing.T) {
	checkLines(t, "discarded line is not returned", "ABC"+keyCtrlC+"PRINT 2"+keyEnter, "PRINT 2")
	checkLines(t, "discarded line is not in history", "A"+keyEnter+"XYZ"+keyCtrlC+keyUp+keyEnter, "A", "A")
	checkLines(t, "Ctrl-C on an empty line", keyCtrlC+"B"+keyEnter, "B")
	checkLines(t, "Ctrl-C then end of input", "ABC"+keyCtrlC+keyCtrlD)
}

// @spec SHELL-EDIT-006
func TestCtrlDEndsInputOnEmptyLine(t *testing.T) {
	checkLines(t, "empty line", keyCtrlD)
	checkLines(t, "after a line", "A"+keyEnter+keyCtrlD, "A")
	checkLines(t, "not on a non-empty line", "AB"+keyLeft+keyCtrlD+keyEnter, "A")
}

// @spec SHELL-EDIT-007
func TestRawModeOnlyWhileReading(t *testing.T) {
	fr := &fakeRaw{}
	e := newEditorReader(strings.NewReader("A"+keyEnter+"B"+keyEnter), io.Discard, fr.raw, noSize, "", io.Discard)
	for i := 1; i <= 2; i++ {
		if _, err := e.ReadLine(); err != nil {
			t.Fatal(err)
		}
		if fr.on != i || fr.off != i {
			t.Errorf("after line %d: raw on %d, restored %d; want %d and %d", i, fr.on, fr.off, i, i)
		}
	}
	if _, err := e.ReadLine(); err != io.EOF {
		t.Fatalf("third ReadLine error = %v, want io.EOF", err)
	}
	if fr.on != fr.off {
		t.Errorf("after end of input: raw on %d, restored %d; want equal", fr.on, fr.off)
	}

	failing := func() (func(), error) { return nil, errors.New("not a terminal") }
	e = newEditorReader(strings.NewReader("A"+keyEnter), io.Discard, failing, noSize, "", io.Discard)
	if _, err := e.ReadLine(); err == nil {
		t.Errorf("ReadLine with a failing raw-mode switch returned no error")
	}
}

// @spec SHELL-EDIT-008
func TestPastedLines(t *testing.T) {
	checkLines(t, "two pasted lines", "\x1b[200~PRINT 1"+keyEnter+"PRINT 2"+keyEnter+"\x1b[201~", "PRINT 1", "PRINT 2")
}

// @spec SHELL-EDIT-009
func TestEditingKeys(t *testing.T) {
	checkLines(t, "keyLeft then insert", "AC"+keyLeft+"B"+keyEnter, "ABC")
	checkLines(t, "keyHome then insert", "BC"+keyHome+"A"+keyEnter, "ABC")
	checkLines(t, "keyBackspace", "ABX"+keyBackspace+"C"+keyEnter, "ABC")
}
