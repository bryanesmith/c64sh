package shell

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// signalingWriter sends SIGINT on a channel the first time it is written
// to, as Ctrl-C pressed while a program prints would.
type signalingWriter struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	sigs chan<- os.Signal
	sent bool
}

func (w *signalingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.sent && w.sigs != nil {
		w.sigs <- os.Interrupt
		w.sent = true
	}
	return w.buf.Write(p)
}

// fakeSignals replaces the shell's SIGINT hooks for the duration of a test,
// handing each channel registered for SIGINT to w.
func fakeSignals(t *testing.T, w *signalingWriter) (notified *int) {
	t.Helper()
	oldNotify, oldStop := notifyInterrupt, stopInterrupt
	t.Cleanup(func() { notifyInterrupt, stopInterrupt = oldNotify, oldStop })
	n := 0
	notifyInterrupt = func(c chan<- os.Signal) {
		w.mu.Lock()
		defer w.mu.Unlock()
		n++
		w.sigs = c
	}
	stopInterrupt = func(c chan<- os.Signal) {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.sigs = nil
	}
	return &n
}

// @spec SHELL-BREAK-001, SHELL-ERR-004, SHELL-INT-005
func TestCtrlCBreaksInteractiveProgram(t *testing.T) {
	w := &signalingWriter{}
	fakeSignals(t, w)
	var stderr bytes.Buffer
	in := "10 PRINT \"A\";\n20 GOTO 10\nRUN\nPRINT \"OK\"\n"
	code := Run(Config{Interactive: true}, strings.NewReader(in), w, &stderr)
	if code != 0 {
		t.Errorf("exit status %d, want 0", code)
	}
	if !regexp.MustCompile(`^A+\nOK\n$`).MatchString(w.buf.String()) {
		t.Errorf("stdout %q, want A repeated, a newline, and OK", w.buf.String())
	}
	want := regexp.MustCompile(`^\n    \*\*\*\* C64SH BASIC V2 \*\*\*\*\n\nREADY\.\nBREAK IN (10|20)\nREADY\.\nREADY\.\n\n$`)
	if !want.MatchString(stderr.String()) {
		t.Errorf("stderr %q, want BREAK IN 10 or 20 then READY.", stderr.String())
	}
}

// @spec SHELL-BREAK-001, SHELL-SCRIPT-005, SHELL-ERR-004
func TestCtrlCBreaksScript(t *testing.T) {
	w := &signalingWriter{}
	fakeSignals(t, w)
	var stderr bytes.Buffer
	in := "10 PRINT \"A\";\n20 GOTO 10\nRUN\nPRINT \"NEVER\"\n"
	code := Run(Config{}, strings.NewReader(in), w, &stderr)
	if code != 130 {
		t.Errorf("exit status %d, want 130", code)
	}
	if !regexp.MustCompile(`^A+\n$`).MatchString(w.buf.String()) {
		t.Errorf("stdout %q, want A repeated and a newline", w.buf.String())
	}
	if !regexp.MustCompile(`^BREAK IN (10|20)\n$`).MatchString(stderr.String()) {
		t.Errorf("stderr %q, want BREAK IN 10 or 20", stderr.String())
	}
}

// @spec SHELL-BREAK-002
func TestSignalsCaughtOnlyWhileExecuting(t *testing.T) {
	w := &signalingWriter{}
	notified := fakeSignals(t, w)
	var stderr bytes.Buffer
	in := "10 PRINT \"A\"\n\nREM\n"
	Run(Config{}, strings.NewReader(in), w, &stderr)
	// One REM line and the final RUN execute; storing a line and a blank
	// line do not.
	if *notified != 2 {
		t.Errorf("SIGINT caught for %d lines, want 2", *notified)
	}
	if w.sigs != nil {
		t.Errorf("SIGINT still caught after the session")
	}
}
