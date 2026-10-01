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
	"strings"

	"golang.org/x/term"

	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/interp"
	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/parser"
)

const usage = `usage: c64sh [FILE]
       c64sh -h | --help

Runs Commodore 64 BASIC V2 in direct mode. With FILE, or when input is
piped, each line is run in turn and c64sh stops at the first error.
Otherwise c64sh starts an interactive session; end it with Ctrl-D.
`

// banner is written to stderr when an interactive session starts.
const banner = "\n    **** C64SH BASIC V2 ****\n\nREADY.\n"

// Config selects how Run behaves.
type Config struct {
	Interactive bool   // banner, READY., continue after errors
	File        string // input file; empty means stdin
}

// Main runs c64sh with the given command-line arguments and streams and
// returns the process exit status.
//
// @spec SHELL-CLI-002, SHELL-CLI-003, SHELL-CLI-004, SHELL-MODE-001, SHELL-MODE-002
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
// @spec SHELL-MODE-003, SHELL-CLI-005, SHELL-CLI-006
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
	var lines lineReader = &plainReader{r: bufio.NewReader(input)}
	if editorWanted(cfg, stdin, stderr) {
		tty := stdin.(*os.File)
		lines = newEditorReader(tty, stderr, terminalRawMode(tty), terminalSize(tty))
	}
	return s.run(lines, name)
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
}

// run reads and executes lines until the input ends or, in script mode, a
// BASIC error occurs, and returns the exit status.
//
// @spec SHELL-MODE-004, SHELL-INT-001, SHELL-INT-004, SHELL-INT-006
// @spec SHELL-SCRIPT-001, SHELL-SCRIPT-002, SHELL-SCRIPT-003, SHELL-SCRIPT-004
// @spec SHELL-SCRIPT-006, SHELL-SCRIPT-007, SHELL-SCRIPT-008
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
	}
	return 0
}

// isBlank reports whether line is empty or holds only spaces and tabs.
func isBlank(line string) bool {
	return strings.Trim(line, " \t") == ""
}

// execLine runs one line in direct mode. It returns the exit status and
// whether the session must stop.
//
// @spec SHELL-LINE-005, SHELL-LINE-006, SHELL-LINE-007
// @spec SHELL-INT-002, SHELL-INT-005, SHELL-SCRIPT-005
func (s *session) execLine(line string) (int, bool) {
	tree, parseErr := parser.Parse(lexer.Lex(line))
	// The statements before a syntax error run first, as on a C64. A
	// runtime error among them is reported instead of the syntax error.
	err := s.interp.Exec(tree)
	if err == nil {
		err = parseErr
	}

	basicErr, isBasic := errors.AsType[*basicerr.Error](err)
	switch {
	case err == nil:
	case isBasic:
		s.report(basicErr)
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

// report writes a BASIC error the way a C64 prints it, on a fresh line.
//
// @spec SHELL-ERR-001, SHELL-ERR-002
func (s *session) report(err *basicerr.Error) {
	s.freshLine()
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
