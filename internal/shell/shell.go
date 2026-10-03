// Package shell runs c64sh: it reads lines interactively or from a script,
// sends each through the lexer, parser, and interpreter, and reports errors.
package shell

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"strings"

	"golang.org/x/term"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/interp"
	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/parser"
)

const usage = `usage: c64sh [FILE]
       c64sh -h | --help

Runs Commodore 64 BASIC V2. With FILE, or when input is piped, each line
is handled as if typed, and c64sh stops at the first error: numbered
lines are stored, other lines run, and a stored program that was never
run with RUN or GOTO runs at the end. Otherwise c64sh starts an
interactive session; end it with Ctrl-D. Ctrl-C stops a running program.
`

// banner is written to stderr when an interactive session starts.
const banner = "\n    **** C64SH BASIC V2 ****\n\nREADY.\n"

// Config selects how Run behaves.
type Config struct {
	Interactive bool   // banner, READY., continue after errors
	File        string // input file; empty means stdin
	HistoryFile string // line-editor history file; empty: none
}

// Main runs c64sh with the given command-line arguments and streams and
// returns the process exit status.
//
// @spec SHELL-CLI-002, SHELL-CLI-003, SHELL-CLI-004, SHELL-MODE-001, SHELL-MODE-002
// @spec SHELL-HIST-001, SHELL-HIST-005
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var files []string
	for _, arg := range args {
		switch {
		case arg == "-h" || arg == "--help":
			io.WriteString(stdout, usage)
			return 0
		case strings.HasPrefix(arg, "-"):
			return usageError(stderr, "unknown option "+arg)
		default:
			files = append(files, arg)
		}
	}
	if len(files) > 1 {
		return usageError(stderr, "too many arguments")
	}

	var cfg Config
	cfg.HistoryFile = defaultHistoryFile(os.LookupEnv, os.UserHomeDir)
	if len(files) == 1 {
		cfg.File = files[0]
	} else {
		cfg.Interactive = isTerminal(stdin)
	}
	return Run(cfg, stdin, stdout, stderr)
}

func usageError(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "c64sh: %s\n%s", msg, usage)
	return 2
}

// isTerminal reports whether r is a terminal.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// Run is Main with the mode chosen by the caller instead of detected.
// Interactive selects the behavior; File, or stdin when File is empty,
// selects the input.
//
// @spec SHELL-MODE-003, SHELL-CLI-005, SHELL-CLI-006, SHELL-FILE-001, SHELL-FILE-002, SHELL-FILE-004
func Run(cfg Config, stdin io.Reader, stdout, stderr io.Writer) int {
	input, name := stdin, "stdin"
	if cfg.File != "" {
		f, err := os.Open(cfg.File)
		if err != nil {
			return inputError(stderr, cfg.File, err)
		}
		defer f.Close()
		input, name = f, cfg.File
	}

	s := &session{
		interactive: cfg.Interactive,
		stderr:      stderr,
		interp:      interp.New(stdout),
	}
	plain := bufio.NewReader(input)
	var lines lineReader = &plainReader{r: plain}
	s.setConsole(cfg, stdin, plain)
	s.interp.SetStorage(dirStorage{})
	if cfg.Interactive {
		s.interp.SetMessages(stderr)
	}
	if editorWanted(cfg, stdin, stderr) {
		tty := stdin.(*os.File)
		lines = newEditorReader(tty, stderr, terminalRawMode(tty), terminalSize(tty), cfg.HistoryFile, stderr)
	}
	status := s.run(lines, name)
	// Data files a program left open are written now.
	if se, ok := errors.AsType[*interp.StorageError](s.interp.CloseFiles()); ok {
		s.reportStorage(se)
		if status == 0 {
			status = 1
		}
	}
	return status
}

// inputError reports that the named input could not be read.
func inputError(stderr io.Writer, name string, err error) int {
	if pathErr, ok := errors.AsType[*fs.PathError](err); ok {
		err = pathErr.Err
	}
	fmt.Fprintf(stderr, "c64sh: %s: %v\n", name, err)
	return 2
}

// session is one run of the shell over one input.
type session struct {
	interactive bool
	stderr      io.Writer
	interp      *interp.Interp
	programMode func() (restore func(), err error) // for INPUT and GET at a terminal; nil otherwise
}

// setConsole gives the interpreter a console reading stdin for INPUT and
// GET: the terminal in program mode, or else stdin as plain text, sharing
// the reader of the script's lines when the script is stdin.
//
// @spec SHELL-KEY-001
func (s *session) setConsole(cfg Config, stdin io.Reader, lines *bufio.Reader) {
	switch {
	case isTerminal(stdin):
		tty := stdin.(*os.File)
		s.interp.SetConsole(&ttyConsole{in: tty, echo: s.stderr})
		s.programMode = terminalProgramMode(tty)
	case cfg.File == "":
		s.interp.SetConsole(&lineConsole{r: lines})
	default:
		s.interp.SetConsole(&lineConsole{r: bufio.NewReader(stdin)})
	}
}

// run reads and executes lines until the input ends or, in script mode, a
// BASIC error occurs, and returns the exit status.
//
// @spec SHELL-MODE-004, SHELL-INT-001, SHELL-INT-004, SHELL-INT-006
// @spec SHELL-SCRIPT-001, SHELL-SCRIPT-002, SHELL-SCRIPT-003, SHELL-SCRIPT-004
// @spec SHELL-SCRIPT-006, SHELL-SCRIPT-007, SHELL-SCRIPT-008, SHELL-SCRIPT-009
// @spec SHELL-LINE-001, SHELL-LINE-002, SHELL-LINE-003, SHELL-LINE-004
func (s *session) run(lines lineReader, name string) int {
	if s.interactive {
		io.WriteString(s.stderr, banner)
	}
	for first := true; ; first = false {
		line, err := lines.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return inputError(s.stderr, name, err)
		}

		skip := isBlank(line) || (first && !s.interactive && strings.HasPrefix(line, "#!"))
		if !skip {
			if status, stop := s.execLine(line); stop {
				return status
			}
		}
	}
	if s.interactive {
		io.WriteString(s.stderr, "\n")
		return 0
	}
	// A script that stored a program but never ran it runs it now.
	if s.interp.NeverRun() {
		if status, stop := s.execLine("RUN"); stop {
			return status
		}
	}
	return 0
}

// isBlank reports whether line is empty or holds only spaces and tabs.
func isBlank(line string) bool {
	return strings.Trim(line, " \t") == ""
}

// execLine stores a numbered line in the program, or runs any other line
// in direct mode. It returns the exit status and whether the session must
// stop.
//
// @spec SHELL-LINE-005, SHELL-LINE-006, SHELL-LINE-007, SHELL-MODE-004
// @spec SHELL-INT-002, SHELL-INT-005, SHELL-SCRIPT-005, SHELL-PROG-001, SHELL-PROG-002, SHELL-KEY-006
func (s *session) execLine(line string) (int, bool) {
	n, rest, numbered, err := lexer.LineNumber(line)
	switch {
	case numbered && err == nil:
		s.interp.Store(n, rest)
		return 0, false // a C64 prints no READY. after storing a line
	case !numbered:
		// A syntax error is part of the tree, where parsing failed, so it
		// is reported only if execution reaches it: statements before it
		// run first, and one skipped by a false IF is never reported, as
		// on a C64.
		tree, _ := parser.Parse(lexer.Lex(line))
		err = s.exec(tree)
	}

	basicErr, isBasic := errors.AsType[*basicerr.Error](err)
	var storageErr *interp.StorageError
	switch {
	case errors.Is(err, interp.ErrEndOfInput):
		s.freshLine()
		io.WriteString(s.stderr, "c64sh: stdin: end of input\n")
		return 1, true
	case errors.As(err, &storageErr):
		s.reportStorage(storageErr)
		if !s.interactive {
			return 1, true
		}
	case err == nil:
	case isBasic:
		s.report(basicErr)
		if !s.interactive && basicErr.Kind == basicerr.Break {
			return 130, true // 128 + SIGINT, as Unix shells report Ctrl-C
		}
		if !s.interactive {
			return 1, true
		}
	default:
		return 1, true // program output could not be written
	}

	if s.interactive {
		s.ready()
	}
	return 0, false
}

// notifyInterrupt and stopInterrupt start and stop delivering SIGINT to
// a channel; tests replace them.
var (
	notifyInterrupt = func(c chan<- os.Signal) { signal.Notify(c, os.Interrupt) }
	stopInterrupt   = signal.Stop
)

// exec executes a line, turning SIGINT (Ctrl-C) during it into a call to
// the interpreter's Interrupt. Outside exec, SIGINT keeps the handling it
// had when c64sh started.
//
// @spec SHELL-BREAK-001, SHELL-BREAK-002
func (s *session) exec(tree *ast.Line) error {
	if s.programMode != nil {
		if restore, err := s.programMode(); err == nil {
			defer restore()
		}
	}
	sigs := make(chan os.Signal, 1)
	notifyInterrupt(sigs)
	defer stopInterrupt(sigs)
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-sigs:
			s.interp.Interrupt()
		case <-done:
		}
	}()
	return s.interp.Exec(tree)
}

// reportStorage writes a storage failure the way c64sh reports problems
// outside BASIC. A disk file that may not be replaced is the 1541's
// FILE EXISTS, which a C64 shows only with its drive light.
//
// @spec SHELL-FILE-003
func (s *session) reportStorage(err *interp.StorageError) {
	s.freshLine()
	reason := err.Err.Error()
	if pathErr, ok := errors.AsType[*fs.PathError](err.Err); ok {
		reason = pathErr.Err.Error()
	}
	if errors.Is(err.Err, fs.ErrExist) {
		reason = fmt.Sprintf("file exists (use %s to replace it)", err.Replace)
	}
	fmt.Fprintf(s.stderr, "c64sh: %s: %s\n", err.File, reason)
}

// report writes a BASIC error the way a C64 prints it, on a fresh line,
// naming the program line it occurred in, if any. BREAK is written
// without "?" and "ERROR", as a C64 writes it.
//
// @spec SHELL-ERR-001, SHELL-ERR-002, SHELL-ERR-004
func (s *session) report(err *basicerr.Error) {
	s.freshLine()
	if err.Kind == basicerr.Break {
		io.WriteString(s.stderr, "BREAK")
		if err.HasLine {
			fmt.Fprintf(s.stderr, " IN %d", err.Line)
		}
		io.WriteString(s.stderr, "\n")
		return
	}
	if err.HasLine {
		fmt.Fprintf(s.stderr, "?%s  ERROR IN %d\n", err.Error(), err.Line)
		return
	}
	fmt.Fprintf(s.stderr, "?%s  ERROR\n", err.Error())
}

// ready writes the READY. prompt on a fresh line.
//
// @spec SHELL-INT-003
func (s *session) ready() {
	s.freshLine()
	io.WriteString(s.stderr, "READY.\n")
}

// freshLine ends stdout's current line if program output left it mid-line.
// A write error is ignored: the next program output hits the same failure
// and stops the session.
func (s *session) freshLine() {
	s.interp.FreshLine()
}
