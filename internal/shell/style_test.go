package shell

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	cyan  = "\x1b[36m"
	green = "\x1b[32m"
	red   = "\x1b[31m"
	reset = "\x1b[0m"
)

// runStyled runs stdin with cfg and returns stdout and stderr.
func runStyled(t *testing.T, cfg Config, stdin string) (string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	Run(cfg, strings.NewReader(stdin), &stdout, &stderr)
	return stdout.String(), stderr.String()
}

// @spec SHELL-STYLE-002, SHELL-STYLE-004
func TestStyledSession(t *testing.T) {
	stdout, stderr := runStyled(t, Config{Interactive: true, Styled: true}, "PRINT \"HI\"\nX\n10 GOTO 10\n")
	want := "\n" + green + "    **** C64SH BASIC V2 ****\n\nREADY." + reset + "\n" +
		green + "READY." + reset + "\n" +
		red + "?SYNTAX  ERROR" + reset + "\n" +
		green + "READY." + reset + "\n" +
		"\n"
	if stderr != want {
		t.Errorf("stderr:\n%q\nwant\n%q", stderr, want)
	}
	if stdout != "HI\n" {
		t.Errorf("stdout %q: program output is not styled", stdout)
	}
	_, stderr = runStyled(t, Config{Styled: true}, "10 PRINT 1/0\n")
	if want := red + "?DIVISION BY ZERO  ERROR IN 10" + reset + "\n"; stderr != want {
		t.Errorf("error in a line: %q; want %q", stderr, want)
	}
}

// @spec SHELL-STYLE-002
func TestStyledFailures(t *testing.T) {
	_, stderr := runStyled(t, Config{Styled: true}, "10 INPUT A\n")
	if want := red + "c64sh: stdin: end of input" + reset + "\n"; stderr != want {
		t.Errorf("end of input: %q; want %q", stderr, want)
	}

	t.Chdir(t.TempDir())
	if err := os.WriteFile(filepath.Join(".", "P.bas"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr = runStyled(t, Config{Styled: true}, "10 END\nSAVE \"P\",8\n")
	if want := red + "c64sh: P.bas: file exists (use SAVE \"@0:P\" to replace it)" + reset + "\n"; stderr != want {
		t.Errorf("storage failure: %q; want %q", stderr, want)
	}
}

// @spec SHELL-STYLE-004
func TestStyleKeepsProgramScreenState(t *testing.T) {
	_, stderr := runStyled(t, Config{Interactive: true, Styled: true, Terminal: true}, "PRINT CHR$(28);\n")
	want := green + "READY." + reset + "\x1b[38;2;104;55;43m\n"
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr %q; want it to contain %q", stderr, want)
	}
}

// @spec SHELL-STYLE-005
func TestUnstyledWritesNoEscapes(t *testing.T) {
	_, stderr := runStyled(t, Config{Interactive: true, Terminal: true}, "PRINT CHR$(28);\nX\nINPUT A\n")
	if strings.Contains(stderr, "\x1b") {
		t.Errorf("stderr %q has escape codes", stderr)
	}
}

// @spec SHELL-STYLE-003
func TestStyledTyping(t *testing.T) {
	st := style{input: cyan, end: func() string { return "<END>" }}

	var echo bytes.Buffer
	e := newEditorReader(strings.NewReader("HI"+keyEnter), &echo, (&fakeRaw{}).raw, noSize, "", io.Discard)
	e.setStyle(st)
	if line, err := e.ReadLine(); err != nil || line != "HI" {
		t.Fatalf("ReadLine = %q, %v", line, err)
	}
	if got := echo.String(); !strings.HasPrefix(got, cyan) || !strings.Contains(got, "HI") || !strings.HasSuffix(got, "<END>") {
		t.Errorf("editor echo %q; want it to start with %q, show HI, and end with <END>", got, cyan)
	}

	echo.Reset()
	c := &ttyConsole{in: &chunkReader{chunks: []string{"AB\x7f\r"}}, echo: &echo, style: st}
	if line, _, err := c.ReadLine(never); err != nil || line != "A" {
		t.Fatalf("console ReadLine = %q, %v", line, err)
	}
	if want := cyan + "A<END>" + cyan + "B<END>\b \b"; echo.String() != want {
		t.Errorf("INPUT echo %q; want %q", echo.String(), want)
	}
}

// @spec SHELL-STYLE-001
func TestStyledCondition(t *testing.T) {
	if styled(io.Discard, false) {
		t.Error("styled(io.Discard, false) = true")
	}
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		t.Skipf("no controlling terminal: %v", err)
	}
	defer tty.Close()
	if !styled(tty, false) {
		t.Error("styled(tty, false) = false")
	}
	if styled(tty, true) {
		t.Error("styled(tty, true) = true: NO_COLOR turns styling off")
	}
}
