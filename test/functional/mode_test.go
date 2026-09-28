package functional_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/bryanesmith/c64sh/internal/shell"
)

// @spec SHELL-MODE-002
func TestScriptModeForFileOrNonTerminalStdin(t *testing.T) {
	path := writeFile(t, "hello.bas", "PRINT \"A\"\n")
	check(t, "file argument", runMain(t, "", path), result{"A\n", "", 0})
	check(t, "piped stdin", runMain(t, "PRINT \"A\"\n"), result{"A\n", "", 0})
}

// @spec SHELL-MODE-003
func TestRunUsesConfigWithoutDetection(t *testing.T) {
	path := writeFile(t, "hello.bas", "PRINT \"A\"\n")

	var stdout, stderr bytes.Buffer
	code := shell.Run(shell.Config{Interactive: true, File: path}, strings.NewReader("PRINT \"IGNORED\"\n"), &stdout, &stderr)
	check(t, "interactive with file", result{stdout.String(), stderr.String(), code},
		result{"A\n", banner + "READY.\n\n", 0})

	stdout.Reset()
	stderr.Reset()
	code = shell.Run(shell.Config{}, strings.NewReader("PRINT \"A\"\n@\nPRINT \"B\"\n"), &stdout, &stderr)
	check(t, "script from stdin", result{stdout.String(), stderr.String(), code},
		result{"A\n", "?SYNTAX  ERROR\n", 1})
}

// readWithin reads exactly n bytes from r, failing the test after a timeout.
func readWithin(t *testing.T, r io.Reader, n int) string {
	t.Helper()
	got := make(chan string, 1)
	go func() {
		buf := make([]byte, n)
		io.ReadFull(r, buf)
		got <- string(buf)
	}()
	select {
	case s := <-got:
		return s
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %d bytes of output", n)
		return ""
	}
}

// @spec SHELL-MODE-004
func TestEachLineRunsImmediately(t *testing.T) {
	run := map[string]func(stdin io.Reader, stdout io.Writer) int{
		"script": func(stdin io.Reader, stdout io.Writer) int {
			return shell.Main(nil, stdin, stdout, io.Discard)
		},
		"interactive": func(stdin io.Reader, stdout io.Writer) int {
			return shell.Run(shell.Config{Interactive: true}, stdin, stdout, io.Discard)
		},
	}
	for name, runShell := range run {
		inR, inW := io.Pipe()
		outR, outW := io.Pipe()
		done := make(chan int, 1)
		go func() {
			done <- runShell(inR, outW)
			outW.Close()
		}()

		// The second line is written only after the first line's output
		// has been read, so the first line must run before more input arrives.
		go io.WriteString(inW, "PRINT \"A\"\n")
		if got := readWithin(t, outR, 2); got != "A\n" {
			t.Errorf("%s: first output %q, want %q", name, got, "A\n")
		}
		go io.WriteString(inW, "PRINT \"B\"\n")
		if got := readWithin(t, outR, 2); got != "B\n" {
			t.Errorf("%s: second output %q, want %q", name, got, "B\n")
		}
		inW.Close()
		io.Copy(io.Discard, outR)
		if code := <-done; code != 0 {
			t.Errorf("%s: exit status %d, want 0", name, code)
		}
	}
}
