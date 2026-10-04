package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/interp"
)

// @spec SHELL-HIST-001
func TestHistoryFileSetting(t *testing.T) {
	cases := []struct {
		name string
		env  interp.MapEnvironment
		home string
		want string
	}{
		{"home directory", nil, "/home/me", filepath.Join("/home/me", ".c64sh_history")},
		{"environment variable", interp.MapEnvironment{"C64SH_HISTORY": "/tmp/h"}, "/home/me", "/tmp/h"},
		{"empty environment variable disables", interp.MapEnvironment{"C64SH_HISTORY": ""}, "/home/me", ""},
		{"no home directory", nil, "", ""},
	}
	for _, c := range cases {
		var env interp.Environment
		if c.env != nil {
			env = c.env
		}
		if got := readSettings(env, c.home).historyFile; got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// @spec SHELL-SET-002, SHELL-SET-003
func TestReadSettings(t *testing.T) {
	def := readSettings(nil, "")
	if def.historySize != 100 || def.input != "\x1b[36m" || def.ready != "\x1b[32m" || def.fail != "\x1b[31m" || def.noColor || len(def.invalid) != 0 {
		t.Errorf("defaults: %+v", def)
	}
	set := readSettings(interp.MapEnvironment{
		"C64SH_HISTSIZE":    "500",
		"C64SH_INPUT_COLOR": "38;2;108;94;181",
		"C64SH_READY_COLOR": "",
		"C64SH_ERROR_COLOR": "1;31",
		"NO_COLOR":          "1",
	}, "")
	if set.historySize != 500 || set.input != "\x1b[38;2;108;94;181m" || set.ready != "" || set.fail != "\x1b[1;31m" || !set.noColor || len(set.invalid) != 0 {
		t.Errorf("set: %+v", set)
	}
	if s := readSettings(interp.MapEnvironment{"C64SH_HISTSIZE": "0", "NO_COLOR": ""}, ""); s.historySize != 0 || s.noColor {
		t.Errorf("size 0, empty NO_COLOR: %+v", s)
	}
	if s := readSettings(interp.MapEnvironment{"C64SH_HISTSIZE": ""}, ""); s.historySize != 100 {
		t.Errorf("empty size: %d, want the default", s.historySize)
	}
}

// @spec SHELL-SET-004
func TestInvalidSettings(t *testing.T) {
	env := interp.MapEnvironment{"C64SH_HISTSIZE": "LOTS", "C64SH_INPUT_COLOR": "cyan", "C64SH_ERROR_COLOR": "31;", "C64SH_READY_COLOR": "-1"}
	s := readSettings(env, "")
	if s.historySize != 100 || s.input != "\x1b[36m" || s.fail != "\x1b[31m" || s.ready != "\x1b[32m" {
		t.Errorf("invalid values should fall back to defaults: %+v", s)
	}
	_, stderr := runStyled(t, Config{Interactive: true, Env: env}, "")
	for _, name := range []string{"C64SH_HISTSIZE", "C64SH_INPUT_COLOR", "C64SH_ERROR_COLOR", "C64SH_READY_COLOR"} {
		msg := fmt.Sprintf("c64sh: %s: invalid value %q\n", name, env[name])
		if strings.Count(stderr, msg) != 1 {
			t.Errorf("stderr %q: want %q exactly once", stderr, msg)
		}
	}
}

// rcFixture writes a run-commands file into a new home directory and
// returns the home directory.
func rcFixture(t *testing.T, content string) string {
	t.Helper()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".c64shrc"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return home
}

// @spec SHELL-SET-001, SHELL-SCREEN-001
func TestSettingsReadAfterRunCommands(t *testing.T) {
	home := rcFixture(t, "ENVIRON \"C64SH_READY_COLOR=33\"\nENVIRON \"NO_COLOR=\"\n")
	_, stderr := runStyled(t, Config{Interactive: true, StderrTerminal: true, Home: home, Env: interp.MapEnvironment{}}, "")
	if !strings.Contains(stderr, "\x1b[33m    **** C64SH BASIC V2 ****") {
		t.Errorf("stderr %q: want the banner in the color the file set", stderr)
	}

	home = rcFixture(t, "ENVIRON \"NO_COLOR=1\"\n")
	stdout, stderr := runStyled(t, Config{Interactive: true, StderrTerminal: true, Terminal: true, Home: home, Env: interp.MapEnvironment{}}, "PRINT CHR$(28);\"X\"\n")
	if strings.Contains(stdout, "\x1b[38;2") || strings.Contains(stderr, "\x1b[3") {
		t.Errorf("stdout %q, stderr %q: NO_COLOR set by the file should turn colors off", stdout, stderr)
	}
}

// @spec SHELL-RC-001
func TestRunCommandsFileLocation(t *testing.T) {
	home := rcFixture(t, "PRINT \"HOME FILE\"\n")
	other := filepath.Join(t.TempDir(), "other")
	os.WriteFile(other, []byte("PRINT \"OTHER FILE\"\n"), 0o644)
	cases := []struct {
		name string
		env  interp.MapEnvironment
		home string
		want string
	}{
		{"home directory", interp.MapEnvironment{}, home, "HOME FILE\n"},
		{"C64SH_RC", interp.MapEnvironment{"C64SH_RC": other}, home, "OTHER FILE\n"},
		{"empty C64SH_RC", interp.MapEnvironment{"C64SH_RC": ""}, home, ""},
		{"no home directory", interp.MapEnvironment{}, "", ""},
	}
	for _, c := range cases {
		stdout, stderr := runStyled(t, Config{Interactive: true, Env: c.env, Home: c.home}, "")
		if stdout != c.want {
			t.Errorf("%s: stdout %q, want %q", c.name, stdout, c.want)
		}
		if !strings.HasPrefix(stderr, "\n    **** C64SH BASIC V2 ****") {
			t.Errorf("%s: stderr %q: want the banner after the file's output", c.name, stderr)
		}
	}
}

// @spec SHELL-RC-002
func TestRunCommandsLinesAsTyped(t *testing.T) {
	home := rcFixture(t, "REM SETUP\r\n10 PRINT \"FROM THE PROGRAM\"\r\n\r\nG$=\"HI\"\r\n")
	stdout, stderr := runStyled(t, Config{Interactive: true, Home: home}, "PRINT G$\nRUN\n")
	if stdout != "HI\nFROM THE PROGRAM\n" {
		t.Errorf("stdout %q: want the variable and program line the file set", stdout)
	}
	if want := "\n    **** C64SH BASIC V2 ****\n\nREADY.\nREADY.\nREADY.\n\n"; stderr != want {
		t.Errorf("stderr %q, want %q (no READY. for the file's lines)", stderr, want)
	}
}

// @spec SHELL-RC-003
func TestRunCommandsErrorStopsFile(t *testing.T) {
	home := rcFixture(t, "PRINT \"ONE\"\nPRINT 1/0\nPRINT \"THREE\"\n")
	path := filepath.Join(home, ".c64shrc")
	stdout, stderr := runStyled(t, Config{Interactive: true, Home: home}, "PRINT \"TYPED\"\n")
	if stdout != "ONE\nTYPED\n" {
		t.Errorf("stdout %q: want the file to stop at its error, and the session to go on", stdout)
	}
	if want := "c64sh: " + path + ":2: ?DIVISION BY ZERO  ERROR\n\n    **** C64SH BASIC V2 ****"; !strings.HasPrefix(stderr, want) {
		t.Errorf("stderr %q; want it to start %q", stderr, want)
	}
	_, stderr = runStyled(t, Config{Interactive: true, Home: home, StderrTerminal: true}, "")
	if want := "\x1b[31mc64sh: " + path + ":2: ?DIVISION BY ZERO  ERROR\x1b[0m\n"; !strings.HasPrefix(stderr, want) {
		t.Errorf("styled stderr %q; want it to start %q", stderr, want)
	}

	t.Chdir(t.TempDir())
	os.WriteFile("P.bas", nil, 0o644)
	home = rcFixture(t, "10 END\nSAVE \"P\",8\nPRINT \"NOT RUN\"\n")
	stdout, stderr = runStyled(t, Config{Interactive: true, Home: home}, "")
	if want := "c64sh: " + filepath.Join(home, ".c64shrc") + ":2: P.bas: file exists (use SAVE \"@0:P\" to replace it)\n"; !strings.Contains(stderr, want) || stdout != "" {
		t.Errorf("storage failure: stdout %q, stderr %q; want it to start %q", stdout, stderr, want)
	}
}

// @spec SHELL-RC-004
func TestRunCommandsFileProblems(t *testing.T) {
	_, stderr := runStyled(t, Config{Interactive: true, Home: t.TempDir()}, "")
	if !strings.HasPrefix(stderr, "\n    **** C64SH BASIC V2 ****") {
		t.Errorf("missing file: stderr %q, want only the session", stderr)
	}
	home := t.TempDir()
	path := filepath.Join(home, ".c64shrc")
	if err := os.Mkdir(path, 0o755); err != nil { // a directory cannot be read as a file
		t.Fatal(err)
	}
	_, stderr = runStyled(t, Config{Interactive: true, Home: home}, "")
	if !strings.HasPrefix(stderr, "c64sh: "+path+": is a directory\n\n    **** C64SH BASIC V2 ****") {
		t.Errorf("unreadable file: stderr %q", stderr)
	}
}

// @spec SHELL-RC-005
func TestScriptsSkipRunCommands(t *testing.T) {
	home := rcFixture(t, "PRINT \"FROM THE FILE\"\n")
	stdout, _ := runStyled(t, Config{Home: home}, "PRINT \"SCRIPT\"\n")
	if stdout != "SCRIPT\n" {
		t.Errorf("stdout %q: a script must not run the run-commands file", stdout)
	}
}
