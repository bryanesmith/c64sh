// Package functional_test runs c64sh end to end: it gives the shell input
// and asserts on the captured stdout, stderr, and exit status.
package functional_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/bryanesmith/c64sh/internal/shell"
)

// banner is what interactive mode writes to stderr when it starts.
const banner = "\n    **** C64SH BASIC V2 ****\n\nREADY.\n"

type result struct {
	stdout, stderr string
	code           int
}

// runMain runs shell.Main with args and stdin, as the c64sh command does.
func runMain(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	return runMainReader(t, strings.NewReader(stdin), args...)
}

func runMainReader(t *testing.T, stdin io.Reader, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := shell.Main(args, stdin, &stdout, &stderr)
	return result{stdout.String(), stderr.String(), code}
}

// runInteractive runs the shell in interactive mode on stdin.
func runInteractive(t *testing.T, stdin string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := shell.Run(shell.Config{Interactive: true}, strings.NewReader(stdin), &stdout, &stderr)
	return result{stdout.String(), stderr.String(), code}
}

// check reports differences between got and want.
func check(t *testing.T, name string, got, want result) {
	t.Helper()
	if got.stdout != want.stdout {
		t.Errorf("%s: stdout %q, want %q", name, got.stdout, want.stdout)
	}
	if got.stderr != want.stderr {
		t.Errorf("%s: stderr %q, want %q", name, got.stderr, want.stderr)
	}
	if got.code != want.code {
		t.Errorf("%s: exit status %d, want %d", name, got.code, want.code)
	}
}

// writeFile writes content to a new file in a temporary directory and
// returns its path.
func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// repoRoot returns the repository root directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
	buildOut  []byte
)

// binary builds the c64sh command once per test run and returns its path.
func binary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "c64sh-functional-")
		if err != nil {
			buildErr = err
			return
		}
		binPath = filepath.Join(dir, "c64sh")
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/c64sh")
		cmd.Dir = repoRoot(t)
		buildOut, buildErr = cmd.CombinedOutput()
	})
	if buildErr != nil {
		t.Fatalf("building c64sh: %v\n%s", buildErr, buildOut)
	}
	return binPath
}

// exitCode returns the exit status of a finished command, or -1 if it was
// terminated by a signal.
func exitCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	t.Fatalf("running command: %v", err)
	return 0
}
