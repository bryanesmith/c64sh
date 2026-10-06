package shell

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// @spec SHELL-MODE-001
func TestNonTerminalsAreNotTerminals(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	path := filepath.Join(t.TempDir(), "script.bas")
	if err := os.WriteFile(path, []byte("PRINT \"A\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	cases := map[string]io.Reader{
		"/dev/null":      devNull,
		"pipe":           r,
		"regular file":   file,
		"strings.Reader": strings.NewReader(""),
	}
	for name, in := range cases {
		if isTerminal(in) {
			t.Errorf("isTerminal(%s) = true, want false", name)
		}
	}
}

// @spec SHELL-MODE-001
func TestTerminalIsTerminal(t *testing.T) {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		t.Skipf("no controlling terminal: %v", err)
	}
	defer tty.Close()
	if !isTerminal(tty) {
		t.Errorf("isTerminal(/dev/tty) = false, want true")
	}
}

// @spec SHELL-SCRIPT-005
func TestStopExitStatus(t *testing.T) {
	var stdout, stderr strings.Builder
	if status := Run(Config{}, strings.NewReader("10 PRINT \"A\":STOP\n"), &stdout, &stderr); status != 1 || stderr.String() != "BREAK IN 10\n" {
		t.Errorf("status %d, stderr %q; want 1 and BREAK IN 10 (STOP is not Ctrl-C's 130)", status, stderr.String())
	}
}
