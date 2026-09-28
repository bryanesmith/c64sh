package functional_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

// @spec SHELL-LINE-001
func TestLineTerminatorsAreRemoved(t *testing.T) {
	check(t, "CRLF", runMain(t, "PRINT \"A\"\r\nPRINT \"B\"\r\n"), result{"A\nB\n", "", 0})
	check(t, "CRLF after unclosed string", runMain(t, "PRINT \"HI\r\n"), result{"HI\n", "", 0})
}

// @spec SHELL-LINE-002
func TestFinalLineWithoutTerminator(t *testing.T) {
	check(t, "script", runMain(t, "PRINT \"A\"\nPRINT \"B\""), result{"A\nB\n", "", 0})
	check(t, "interactive", runInteractive(t, "PRINT \"A\""), result{"A\n", banner + "READY.\n\n", 0})
}

// @spec SHELL-LINE-003
func TestVeryLongLine(t *testing.T) {
	long := strings.Repeat("x", 1<<20)
	check(t, "1 MiB line", runMain(t, "PRINT \""+long+"\"\n"), result{long + "\n", "", 0})
}

// @spec SHELL-LINE-004
func TestBlankMeansOnlySpacesAndTabs(t *testing.T) {
	check(t, "spaces and tabs", runMain(t, " \t \n"), result{"", "", 0})
	check(t, "form feed is not blank", runMain(t, "\f\n"), result{"", "?SYNTAX  ERROR\n", 1})
}

// @spec SHELL-LINE-005
func TestSyntaxErrorAfterPartialExecution(t *testing.T) {
	check(t, "error in later statement", runMain(t, "PRINT \"A\":PRINT \"B\"@\n"),
		result{"A\nB\n", "?SYNTAX  ERROR\n", 1})
	check(t, "malformed item", runMain(t, "PRINT \"A\";\"B\"+@\n"),
		result{"A\n", "?SYNTAX  ERROR\n", 1})
	check(t, "error at statement start", runMain(t, "PRINT \"A\":@\n"),
		result{"A\n", "?SYNTAX  ERROR\n", 1})
}

// @spec SHELL-LINE-006
func TestRuntimeErrorWinsOverLaterSyntaxError(t *testing.T) {
	long := strings.Repeat("x", 200)
	input := "PRINT \"" + long + "\"+\"" + long + "\":@\n"
	check(t, "string too long then syntax", runMain(t, input),
		result{"", "?STRING TOO LONG  ERROR\n", 1})
}

// failWriter fails every write and counts calls.
type failWriter struct{ calls int }

func (w *failWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, errors.New("disk full")
}

// @spec SHELL-LINE-007
func TestOutputWriteFailure(t *testing.T) {
	out := &failWriter{}
	var stderr bytes.Buffer
	code := shellMain(strings.NewReader("PRINT \"A\"\nPRINT \"B\"\n"), out, &stderr)
	if code != 1 {
		t.Errorf("exit status %d, want 1", code)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr %q, want empty", stderr.String())
	}
	if out.calls != 1 {
		t.Errorf("stdout written %d times, want 1 (stop after the failure)", out.calls)
	}
}

// @spec SHELL-LINE-008
func TestClosedPipeEndsSilently(t *testing.T) {
	bin := binary(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	r.Close() // nobody will read c64sh's output

	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(strings.Repeat("PRINT \"A\"\n", 1000))
	cmd.Stdout = w
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	w.Close()

	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("c64sh exited normally (%v), want termination by SIGPIPE", err)
	}
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() || ws.Signal() != syscall.SIGPIPE {
		t.Errorf("c64sh ended with %v, want termination by SIGPIPE", ee)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr %q, want empty", stderr.String())
	}
}
