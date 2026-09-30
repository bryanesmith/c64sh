package shell

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// editorSession runs a line editor on keys with the given history file and
// returns the lines read and the warnings written.
func editorSession(t *testing.T, file, keys string) ([]string, string) {
	t.Helper()
	var warn bytes.Buffer
	e := newEditorReader(strings.NewReader(keys), io.Discard, (&fakeRaw{}).raw, noSize, file, &warn)
	var lines []string
	for {
		line, err := e.ReadLine()
		if err == io.EOF {
			return lines, warn.String()
		}
		if err != nil {
			t.Fatalf("ReadLine error: %v", err)
		}
		lines = append(lines, line)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// @spec SHELL-HIST-001
func TestDefaultHistoryFile(t *testing.T) {
	env := func(vars map[string]string) func(string) (string, bool) {
		return func(k string) (string, bool) { v, ok := vars[k]; return v, ok }
	}
	home := func() (string, error) { return "/home/me", nil }
	noHome := func() (string, error) { return "", errors.New("no home") }
	cases := []struct {
		name string
		env  map[string]string
		home func() (string, error)
		want string
	}{
		{"home directory", nil, home, filepath.Join("/home/me", ".c64sh_history")},
		{"environment variable", map[string]string{"C64SH_HISTORY": "/tmp/h"}, home, "/tmp/h"},
		{"empty environment variable disables", map[string]string{"C64SH_HISTORY": ""}, home, ""},
		{"no home directory", nil, noHome, ""},
	}
	for _, c := range cases {
		if got := defaultHistoryFile(env(c.env), c.home); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// @spec SHELL-HIST-002
func TestHistoryLoadedFromFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "history")
	var content strings.Builder
	for i := 1; i <= 150; i++ {
		fmt.Fprintf(&content, "PRINT %d\r\n\n", i) // CRLF, and a blank line after each
	}
	if err := os.WriteFile(file, []byte(content.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	lines, warn := editorSession(t, file, keyUp+keyEnter)
	if len(lines) != 1 || lines[0] != "PRINT 150" {
		t.Errorf("up recalled %q, want %q", lines, "PRINT 150")
	}
	if warn != "" {
		t.Errorf("unexpected warning %q", warn)
	}
	// The session above ran PRINT 150 again, saving it; start over.
	if err := os.WriteFile(file, []byte(content.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	lines, _ = editorSession(t, file, strings.Repeat(keyUp, 200)+keyEnter)
	if len(lines) != 1 || lines[0] != "PRINT 51" {
		t.Errorf("oldest recalled %q, want %q (only the most recent 100 are loaded)", lines, "PRINT 51")
	}

	missing := filepath.Join(t.TempDir(), "none")
	lines, warn = editorSession(t, missing, keyUp+"X"+keyEnter)
	if len(lines) != 1 || lines[0] != "X" || warn != "" {
		t.Errorf("missing file: lines %q, warning %q; want [X] and no warning", lines, warn)
	}
}

// @spec SHELL-HIST-003
func TestHistorySavedToFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "history")
	editorSession(t, file, "A"+keyEnter+keyEnter+"B"+keyEnter+"XYZ"+keyCtrlC)
	if got := readFile(t, file); got != "A\nB\n" {
		t.Errorf("file %q, want %q (blank and discarded lines not saved)", got, "A\nB\n")
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode %v, want 0600", perm)
	}
	lines, _ := editorSession(t, file, keyUp+keyEnter+"C"+keyEnter)
	if strings.Join(lines, "|") != "B|C" {
		t.Errorf("next session read %q, want [B C]", lines)
	}
	if got := readFile(t, file); got != "A\nB\nB\nC\n" {
		t.Errorf("file after second session %q, want %q", got, "A\nB\nB\nC\n")
	}

	var many strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&many, "L%d\n", i)
	}
	os.WriteFile(file, []byte(many.String()), 0o600)
	editorSession(t, file, "NEW"+keyEnter)
	got := strings.Split(strings.TrimSuffix(readFile(t, file), "\n"), "\n")
	if len(got) != 100 || got[0] != "L2" || got[99] != "NEW" {
		t.Errorf("capped file has %d lines, first %q, last %q; want 100, L2, NEW", len(got), got[0], got[len(got)-1])
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("directory holds %d entries, want only the history file (no temporary files left)", len(entries))
	}
}

// @spec SHELL-HIST-004
func TestHistoryFileErrorsWarnOnce(t *testing.T) {
	dir := t.TempDir()
	lines, warn := editorSession(t, dir, "A"+keyEnter+"B"+keyEnter+keyUp+keyEnter)
	if strings.Join(lines, "|") != "A|B|B" {
		t.Errorf("lines %q, want history to work in memory", lines)
	}
	if strings.Count(warn, "c64sh: history: ") != 1 || !strings.HasSuffix(warn, "\n") {
		t.Errorf("warning %q, want exactly one c64sh: history: line", warn)
	}

	if os.Geteuid() != 0 {
		readOnly := t.TempDir()
		if err := os.Chmod(readOnly, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(readOnly, 0o700) })
		_, warn = editorSession(t, filepath.Join(readOnly, "history"), "A"+keyEnter+"B"+keyEnter)
		if strings.Count(warn, "c64sh: history: ") != 1 {
			t.Errorf("unwritable directory: warning %q, want exactly one", warn)
		}
	}
}

// @spec SHELL-HIST-005
func TestNoHistoryFileWithoutEditor(t *testing.T) {
	file := filepath.Join(t.TempDir(), "history")
	var stdout, stderr bytes.Buffer
	cfg := Config{Interactive: true, HistoryFile: file}
	Run(cfg, strings.NewReader("PRINT 1\n"), &stdout, &stderr)
	cfg.Interactive = false
	Run(cfg, strings.NewReader("PRINT 2\n"), &stdout, &stderr)
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("history file exists after sessions without the line editor (err %v)", err)
	}
}
