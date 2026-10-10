package shell

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// childSystem runs the programs SYS runs as child processes, with no
// shell between: the name is found on the PATH unless it holds a "/", and
// the arguments are passed as they are. The child shares c64sh's streams;
// a stdin that is not a file (as in tests) gives the child no input.
type childSystem struct {
	s              *session
	stdin          io.Reader
	stdout, stderr io.Writer
}

// Run runs the program at the terminal as it was before the line ran,
// returning its exit status, or 128 plus the signal's number for a
// program ended by a signal, as Unix shells report it.
//
// @spec SHELL-SYS-001, SHELL-SYS-002
func (c childSystem) Run(name string, args []string) (int, error) {
	cmd := exec.Command(name, args...)
	if cmd.Err != nil {
		return 0, cmd.Err
	}
	if f, ok := c.stdin.(*os.File); ok {
		cmd.Stdin = f
	}
	cmd.Stdout, cmd.Stderr = c.stdout, c.stderr
	if c.s.leaveProgramMode() {
		defer c.s.enterProgramMode()
	}
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 0, err // nil, or the program could not be started
	}
	if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal()), nil
	}
	return exitErr.ExitCode(), nil
}
