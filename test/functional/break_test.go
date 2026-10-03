package functional_test

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
	"time"
)

// startBinary starts the real c64sh binary with args and returns the
// command, a writer to its stdin, and a reader of its stdout.
func startBinary(t *testing.T, stderr *bytes.Buffer, args ...string) (*exec.Cmd, io.WriteCloser, *bufio.Reader) {
	t.Helper()
	cmd := exec.Command(binary(t), args...)
	cmd.Stderr = stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	return cmd, stdin, bufio.NewReader(stdout)
}

// interruptUntilExit sends SIGINT to cmd until it exits, draining stdout,
// and returns its exit error.
func interruptUntilExit(t *testing.T, cmd *exec.Cmd, stdout io.Reader) error {
	t.Helper()
	go io.Copy(io.Discard, stdout)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		cmd.Process.Signal(os.Interrupt)
		select {
		case err := <-done:
			return err
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatal("c64sh did not exit after SIGINT")
	return nil
}

// @spec SHELL-BREAK-001, SHELL-SCRIPT-005, SHELL-ERR-004
func TestCtrlCStopsScriptProgram(t *testing.T) {
	script := filepath.Join(t.TempDir(), "loop.bas")
	if err := os.WriteFile(script, []byte("10 PRINT \"A\"\n20 GOTO 10\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd, _, stdout := startBinary(t, &stderr, script)
	if line, err := stdout.ReadString('\n'); err != nil || line != "A\n" {
		t.Fatalf("first output %q, %v; want \"A\\n\"", line, err)
	}
	err := interruptUntilExit(t, cmd, stdout)
	if code := exitCode(t, err); code != 130 {
		t.Errorf("exit status %d, want 130", code)
	}
	if !regexp.MustCompile(`^BREAK IN (10|20)\n$`).MatchString(stderr.String()) {
		t.Errorf("stderr %q, want BREAK IN 10 or 20", stderr.String())
	}
}

// @spec SHELL-BREAK-002
func TestCtrlCWhileWaitingForInputEndsC64sh(t *testing.T) {
	var stderr bytes.Buffer
	cmd, stdin, stdout := startBinary(t, &stderr)
	defer stdin.Close()
	io.WriteString(stdin, "PRINT \"A\"\n")
	if line, err := stdout.ReadString('\n'); err != nil || line != "A\n" {
		t.Fatalf("first output %q, %v; want \"A\\n\"", line, err)
	}
	err := interruptUntilExit(t, cmd, stdout)
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("c64sh exited with %v, want termination by SIGINT", err)
	}
	if ws, ok := ee.Sys().(syscall.WaitStatus); !ok || !ws.Signaled() || ws.Signal() != syscall.SIGINT {
		t.Errorf("c64sh ended with %v, want termination by SIGINT", err)
	}
	if stderr.String() != "" {
		t.Errorf("stderr %q, want nothing", stderr.String())
	}
}
