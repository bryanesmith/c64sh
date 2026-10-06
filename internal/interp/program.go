package interp

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/parser"
	"github.com/bryanesmith/c64sh/internal/token"
)

// progLine is one stored program line.
type progLine struct {
	number int
	text   string    // as typed after the line number, for LIST
	tree   *ast.Line // text parsed, syntax errors included where they occur
}

// errEnd is returned by END, LIST, and NEW: it ends the line, and a
// running program, without error.
var errEnd = errors.New("end")

// jump is returned by RUN and GOTO: it ends the line, and the program
// continues at line, or starts from its first line if !hasLine.
type jump struct {
	line    int
	hasLine bool
}

func (*jump) Error() string { return "jump" }

// Store stores text as program line n (0 to 63999), replacing any line n,
// or deletes line n if text is empty. Either way, it clears the variables,
// as the C64 ROM does whenever the program changes ($A4ED, $A52A).
//
// @spec INTERP-048, INTERP-049, INTERP-050, INTERP-051
func (in *Interp) Store(n int, text string) {
	in.clr()
	in.program = storeLine(in.program, n, text)
}

// storeLine stores text as line n of prog, replacing any line n, or
// deletes line n if text is empty, and returns the program.
func storeLine(prog []progLine, n int, text string) []progLine {
	i, found := findLine(prog, n)
	switch {
	case text == "" && found:
		return slices.Delete(prog, i, i+1)
	case text == "":
		return prog
	}
	l := newProgLine(n, text)
	if found {
		prog[i] = l
		return prog
	}
	return slices.Insert(prog, i, l)
}

// newProgLine returns a program line, its text parsed; syntax errors are
// in the tree.
func newProgLine(n int, text string) progLine {
	tree, _ := parser.Parse(lexer.Lex(text))
	return progLine{number: n, text: text, tree: tree}
}

// NeverRun reports whether the stored program holds lines and no RUN or
// GOTO has been executed since the Interp was created.
//
// @spec INTERP-062
func (in *Interp) NeverRun() bool {
	return len(in.program) > 0 && !in.ran
}

// find returns the index of line n in the program, or where it would go.
func (in *Interp) find(n int) (int, bool) {
	return findLine(in.program, n)
}

// findLine returns the index of line n in prog, or where it would go.
func findLine(prog []progLine, n int) (int, bool) {
	return slices.BinarySearchFunc(prog, n, func(l progLine, n int) int {
		return l.number - n
	})
}

// execRun clears the variables and returns the request to run the
// program, which execute carries out.
func (in *Interp) execRun(s *ast.RunStmt) error {
	in.ran = true
	in.clr()
	return &jump{line: s.Line, hasLine: s.HasLine}
}

// execGoto returns the request to continue the program at a line, which
// execute carries out, keeping the variables.
//
// @spec INTERP-063, INTERP-064
func (in *Interp) execGoto(s *ast.GotoStmt) error {
	in.ran = true
	return &jump{line: s.Line, hasLine: true}
}

// atLine returns err with the program line it occurred in, if it is a
// BASIC error and line is not -1 (direct mode).
func atLine(err error, line int) error {
	be, ok := errors.AsType[*basicerr.Error](err)
	if !ok || line < 0 {
		return err
	}
	e := *be
	e.Line, e.HasLine = line, true
	return &e
}

// execStop stops with BREAK, as the STOP key does, keeping the place to
// continue from in a running program ($A82C).
//
// @spec INTERP-159
func (in *Interp) execStop() error {
	if in.cur.line != directLine {
		in.cont = &pos{in.cur.line, in.cur.stmt + 1}
	}
	return &basicerr.Error{Kind: basicerr.Break, Stopped: true}
}

// execCont continues the program where it stopped ($A857). In a running
// program, the ROM's place to continue from is the CONT statement itself,
// so the program loops on it until interrupted.
//
// @spec INTERP-162, INTERP-163
func (in *Interp) execCont() error {
	if in.cur.line != directLine {
		return &resume{at: in.cur}
	}
	if in.cont == nil {
		return &basicerr.Error{Kind: basicerr.CantContinue}
	}
	if in.cont.line >= len(in.program) {
		return nil // the program had ended
	}
	return &resume{at: *in.cont}
}

// execList writes the program's lines in the statement's range as the
// C64 ROM lists them ($A6C9): before each line a newline, then the line number, a space, and the text, with
// "?" shown as the PRINT keyword it stands for; after the last line, the
// newline that begins a C64's READY. message.
//
// @spec INTERP-059
func (in *Interp) execList(s *ast.ListStmt) error {
	var buf strings.Builder
	for _, l := range in.program {
		if l.number >= s.From && l.number <= s.To {
			buf.WriteString("\n" + strconv.Itoa(l.number) + " " + listText(l.text))
		}
	}
	if buf.Len() == 0 {
		return errEnd
	}
	buf.WriteString("\n")
	if err := in.emit(in.cmdFile(), buf.String()); err != nil {
		return err
	}
	return errEnd
}

// listText returns text with each "?" that the lexer reads as PRINT
// written as PRINT, as a C64 stores both as the same keyword.
func listText(text string) string {
	var b strings.Builder
	last := 0
	for _, t := range lexer.Lex(text) {
		if t.Kind == token.Print && t.Value == "?" {
			b.WriteString(text[last:t.Pos] + "PRINT")
			last = t.Pos + 1
		}
	}
	b.WriteString(text[last:])
	return b.String()
}

// execNew erases the program and the variables.
//
// @spec INTERP-060
func (in *Interp) execNew() error {
	in.program = nil
	in.clr()
	return errEnd
}
