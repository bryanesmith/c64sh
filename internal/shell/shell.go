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
	"time"

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

// banner is written to stderr, with a blank line before it, when an
// interactive session starts.
const banner = "    **** C64SH BASIC V2 ****\n\nREADY."

// Config selects how Run behaves.
type Config struct {
	Interactive    bool               // run-commands file, banner, READY., continue after errors
	File           string             // input file; empty means stdin
	Clock          func() time.Time   // the interpreter's clock; nil: the system clock
	Terminal       bool               // stdout is a terminal: screen codes become escape codes
	StderrTerminal bool               // stderr is a terminal: the shell may style its own text
	Env            interp.Environment // ENVIRON's environment and the shell's settings; nil: empty
	Home           string             // home directory, for ~/.c64shrc and ~/.c64sh_history; empty: unknown
}

// Main runs c64sh with the given command-line arguments and streams and
// returns the process exit status.
//
// @spec SHELL-CLI-002, SHELL-CLI-003, SHELL-CLI-004, SHELL-MODE-001, SHELL-MODE-002
// @spec SHELL-HIST-001, SHELL-HIST-005, SHELL-STYLE-001, SHELL-ENV-001
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
	cfg.Terminal = isTerminal(stdout)
	cfg.StderrTerminal = isTerminal(stderr)
	cfg.Env = osEnvironment{}
	cfg.Home, _ = os.UserHomeDir() // "" if unknown
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

// isTerminal reports whether r, a reader or writer, is a terminal.
func isTerminal(r any) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// Run is Main with the mode chosen by the caller instead of detected.
// Interactive selects the behavior; File, or stdin when File is empty,
// selects the input.
//
// @spec SHELL-MODE-003, SHELL-CLI-005, SHELL-CLI-006, SHELL-FILE-001, SHELL-FILE-002, SHELL-FILE-004, SHELL-CLOCK-001, SHELL-SCREEN-002, SHELL-ENV-001
// @spec SHELL-SET-001, SHELL-SET-004, SHELL-RC-005
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
	if cfg.Clock != nil {
		s.interp.SetClock(cfg.Clock)
	}
	if cfg.Env != nil {
		s.interp.SetEnvironment(cfg.Env)
	}
	s.interp.SetStorage(dirStorage{})
	if cfg.Interactive {
		s.interp.SetMessages(stderr)
	}

	// The settings are read again after the run-commands file, which can
	// set them; that reading decides them.
	set := readSettings(cfg.Env, cfg.Home)
	s.apply(cfg, set)
	status, stop := 0, false
	if cfg.Interactive {
		if rc := runCommandsFile(cfg.Env, cfg.Home); rc != "" {
			status, stop = s.runCommands(rc)
		}
		set = readSettings(cfg.Env, cfg.Home)
		s.apply(cfg, set)
	}
	for _, msg := range set.invalid {
		io.WriteString(stderr, msg+"\n")
	}
	if !stop {
		if editorWanted(cfg, stdin, stderr) {
			tty := stdin.(*os.File)
			editor := newEditorReader(tty, stderr, terminalRawMode(tty), terminalSize(tty), set.historyFile, set.historySize, stderr)
			editor.setStyle(s.style)
			lines = editor
		}
		status = s.run(lines, name)
	}
	if cfg.Terminal {
		// A program's color stays set, as on a C64; the user's own shell
		// should not inherit it.
		io.WriteString(stdout, "\x1b[0m")
	}
	// Data files a program left open are written now.
	if se, ok := errors.AsType[*interp.StorageError](s.interp.CloseFiles()); ok {
		s.reportStorage(se)
		if status == 0 {
			status = 1
		}
	}
	return status
}

// apply puts the shell's settings into effect: colors in program output,
// unless NO_COLOR, the string limit, and the shell's styles, when stderr
// is a terminal.
//
// @spec SHELL-SCREEN-001, SHELL-STYLE-001
func (s *session) apply(cfg Config, set settings) {
	s.interp.SetScreen(cfg.Terminal, !set.noColor)
	s.interp.SetStringLimit(set.stringLimit)
	s.style = style{}
	if cfg.StderrTerminal && !set.noColor {
		s.style = newStyle(s.interp, set)
	}
}

// runCommands runs the lines of the run-commands file as if typed, with
// no READY. after them, and stops at the first error, which it reports
// with the file and line in front. It returns the exit status and whether
// the session must stop, as execLine does.
//
// @spec SHELL-RC-002, SHELL-RC-003, SHELL-RC-004
func (s *session) runCommands(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, false
	}
	if err != nil {
		io.WriteString(s.stderr, s.style.paint(s.style.fail, fmt.Sprintf("c64sh: %s: %s", path, reason(err)))+"\n")
		return 0, false
	}
	defer func() { s.where = "" }()
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if isBlank(line) {
			continue
		}
		s.where = fmt.Sprintf("c64sh: %s:%d: ", path, i+1)
		_, err := s.enter(line)
		if status, stop := s.handle(err); stop {
			return status, true
		}
		if err != nil {
			break
		}
	}
	return 0, false
}

// reason returns why an operation on a file failed, without the operation
// and path a *fs.PathError adds.
func reason(err error) string {
	if pathErr, ok := errors.AsType[*fs.PathError](err); ok {
		return pathErr.Err.Error()
	}
	return err.Error()
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
	style       style  // how the shell's own text is styled
	where       string // while the run-commands file runs, "c64sh: FILE:N: " for messages
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
		s.interp.SetConsole(&ttyConsole{in: tty, echo: s.stderr, style: &s.style})
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
		io.WriteString(s.stderr, "\n"+s.style.paint(s.style.ready, banner)+"\n")
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
// in direct mode, and reports any error. It returns the exit status and
// whether the session must stop.
//
// @spec SHELL-INT-002, SHELL-INT-005, SHELL-SCRIPT-005, SHELL-KEY-006
func (s *session) execLine(line string) (int, bool) {
	stored, err := s.enter(line)
	if stored {
		return 0, false // a C64 prints no READY. after storing a line
	}
	if status, stop := s.handle(err); stop {
		return status, true
	}
	if s.interactive {
		s.ready()
	}
	return 0, false
}

// enter stores a numbered line in the program, reporting stored, or runs
// any other line in direct mode, returning its error.
//
// @spec SHELL-LINE-005, SHELL-LINE-006, SHELL-LINE-007, SHELL-MODE-004, SHELL-PROG-001, SHELL-PROG-002
func (s *session) enter(line string) (stored bool, err error) {
	n, rest, numbered, err := lexer.LineNumber(line)
	switch {
	case numbered && err == nil:
		s.interp.Store(n, rest)
		return true, nil
	case !numbered:
		// A syntax error is part of the tree, where parsing failed, so it
		// is reported only if execution reaches it: statements before it
		// run first, and one skipped by a false IF is never reported, as
		// on a C64.
		tree, _ := parser.Parse(lexer.Lex(line))
		return false, s.exec(tree)
	}
	return false, err
}

// handle reports an error from a line, if any, and returns the exit status
// and whether the session must stop.
func (s *session) handle(err error) (int, bool) {
	basicErr, isBasic := errors.AsType[*basicerr.Error](err)
	var storageErr *interp.StorageError
	switch {
	case errors.Is(err, interp.ErrEndOfInput):
		s.freshLine()
		io.WriteString(s.stderr, s.style.paint(s.style.fail, s.label("c64sh: ")+"stdin: end of input")+"\n")
		return 1, true
	case errors.As(err, &storageErr):
		s.reportStorage(storageErr)
		if !s.interactive {
			return 1, true
		}
	case err == nil:
	case isBasic:
		s.report(basicErr)
		if !s.interactive && basicErr.Kind == basicerr.Break && !basicErr.Stopped {
			return 130, true // 128 + SIGINT, as Unix shells report Ctrl-C
		}
		if !s.interactive {
			return 1, true
		}
	default:
		return 1, true // program output could not be written
	}
	return 0, false
}

// label returns what a message starts with: the run-commands file's place
// while it runs, otherwise def.
func (s *session) label(def string) string {
	if s.where != "" {
		return s.where
	}
	return def
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
	why := reason(err.Err)
	if errors.Is(err.Err, fs.ErrExist) {
		why = fmt.Sprintf("file exists (use %s to replace it)", err.Replace)
	}
	io.WriteString(s.stderr, s.style.paint(s.style.fail, fmt.Sprintf("%s%s: %s", s.label("c64sh: "), err.File, why))+"\n")
}

// report writes a BASIC error the way a C64 prints it, on a fresh line,
// naming the program line it occurred in, if any. BREAK is written
// without "?" and "ERROR", as a C64 writes it.
//
// @spec SHELL-ERR-001, SHELL-ERR-002, SHELL-ERR-004, SHELL-STYLE-002
func (s *session) report(err *basicerr.Error) {
	s.freshLine()
	msg := "?" + err.Error() + "  ERROR"
	if err.Kind == basicerr.Break {
		msg = "BREAK"
	}
	if err.HasLine {
		msg += fmt.Sprintf(" IN %d", err.Line)
	}
	io.WriteString(s.stderr, s.style.paint(s.style.fail, s.label("")+msg)+"\n")
}

// ready writes the READY. prompt on a fresh line.
//
// @spec SHELL-INT-003
func (s *session) ready() {
	s.freshLine()
	io.WriteString(s.stderr, s.style.paint(s.style.ready, "READY.")+"\n")
}

// freshLine ends stdout's current line if program output left it mid-line.
// A write error is ignored: the next program output hits the same failure
// and stops the session.
func (s *session) freshLine() {
	s.interp.FreshLine()
}
