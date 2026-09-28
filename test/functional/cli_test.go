package functional_test

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// @spec SHELL-CLI-001
func TestBinaryExitsWithShellStatus(t *testing.T) {
	bin := binary(t)
	cases := []struct {
		name  string
		args  []string
		stdin string
		want  int
	}{
		{"success", nil, "PRINT \"A\"\n", 0},
		{"help", []string{"-h"}, "", 0},
		{"BASIC error", nil, "@\n", 1},
		{"usage error", []string{"--bogus"}, "", 2},
	}
	for _, c := range cases {
		cmd := exec.Command(bin, c.args...)
		cmd.Stdin = strings.NewReader(c.stdin)
		if got := exitCode(t, cmd.Run()); got != c.want {
			t.Errorf("%s: exit status %d, want %d", c.name, got, c.want)
		}
	}
}

// @spec SHELL-CLI-002
func TestHelp(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		r := runMain(t, "", flag)
		if !strings.Contains(strings.ToLower(r.stdout), "usage:") || !strings.Contains(r.stdout, "c64sh") {
			t.Errorf("%s: stdout %q does not contain usage text", flag, r.stdout)
		}
		if r.stderr != "" {
			t.Errorf("%s: stderr %q, want empty", flag, r.stderr)
		}
		if r.code != 0 {
			t.Errorf("%s: exit status %d, want 0", flag, r.code)
		}
	}
}

// checkUsageError checks a usage error's stderr and status.
func checkUsageError(t *testing.T, name string, r result, mention string) {
	t.Helper()
	if !strings.HasPrefix(r.stderr, "c64sh: ") {
		t.Errorf("%s: stderr %q does not start with %q", name, r.stderr, "c64sh: ")
	}
	if !strings.Contains(r.stderr, mention) {
		t.Errorf("%s: stderr %q does not mention %q", name, r.stderr, mention)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "usage:") {
		t.Errorf("%s: stderr %q does not contain usage text", name, r.stderr)
	}
	if r.stdout != "" {
		t.Errorf("%s: stdout %q, want empty", name, r.stdout)
	}
	if r.code != 2 {
		t.Errorf("%s: exit status %d, want 2", name, r.code)
	}
}

// @spec SHELL-CLI-003
func TestUnknownOption(t *testing.T) {
	checkUsageError(t, "-x", runMain(t, "", "-x"), "-x")
	checkUsageError(t, "--bogus", runMain(t, "", "--bogus"), "--bogus")
}

// @spec SHELL-CLI-004
func TestTooManyArguments(t *testing.T) {
	a := writeFile(t, "a.bas", "PRINT \"A\"\n")
	b := writeFile(t, "b.bas", "PRINT \"B\"\n")
	checkUsageError(t, "two files", runMain(t, "", a, b), "")
}

// @spec SHELL-CLI-005
func TestUnreadableFile(t *testing.T) {
	missing := t.TempDir() + "/missing.bas"
	check(t, "missing file", runMain(t, "", missing),
		result{"", "c64sh: " + missing + ": no such file or directory\n", 2})

	dir := t.TempDir()
	r := runMain(t, "", dir)
	if !strings.HasPrefix(r.stderr, "c64sh: "+dir+": ") || !strings.Contains(r.stderr, "directory") {
		t.Errorf("directory: stderr %q, want %q followed by a reason", r.stderr, "c64sh: "+dir+": ")
	}
	if r.code != 2 {
		t.Errorf("directory: exit status %d, want 2", r.code)
	}

	if os.Geteuid() != 0 {
		unreadable := writeFile(t, "secret.bas", "PRINT \"A\"\n")
		if err := os.Chmod(unreadable, 0o000); err != nil {
			t.Fatal(err)
		}
		check(t, "unreadable file", runMain(t, "", unreadable),
			result{"", "c64sh: " + unreadable + ": permission denied\n", 2})
	}
}

// errReader fails every read with err.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

// @spec SHELL-CLI-006
func TestStdinReadError(t *testing.T) {
	stdin := io.MultiReader(strings.NewReader("PRINT \"A\"\n"), errReader{errors.New("boom")})
	check(t, "stdin read error", runMainReader(t, stdin),
		result{"A\n", "c64sh: stdin: boom\n", 2})
}
