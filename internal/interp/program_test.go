package interp

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/parser"
)

// session runs lines on one interpreter as the shell does: a numbered line
// is stored, and any other line is parsed and executed. It returns the
// output and the error of the last line.
func session(t *testing.T, lines ...string) (*Interp, string, error) {
	t.Helper()
	rec := &recorder{}
	in := New(rec)
	var err error
	for _, l := range lines {
		err = enter(in, l)
	}
	return in, rec.String(), err
}

func enter(in *Interp, l string) error {
	if n, rest, ok, err := lexer.LineNumber(l); ok {
		if err == nil {
			in.Store(n, rest)
		}
		return err
	}
	tree, _ := parser.Parse(lexer.Lex(l))
	return in.Exec(tree)
}

type sessionCase struct {
	name    string
	lines   []string
	want    string // output
	wantErr error  // nil, or a *basicerr.Error to match exactly
}

func runSessionCases(t *testing.T, cases []sessionCase) {
	t.Helper()
	for _, c := range cases {
		_, got, err := session(t, c.lines...)
		if got != c.want {
			t.Errorf("%s: output %q, want %q", c.name, got, c.want)
		}
		if !sameErr(err, c.wantErr) {
			t.Errorf("%s: error %#v, want %#v", c.name, err, c.wantErr)
		}
	}
}

func sameErr(got, want error) bool {
	if want == nil {
		return got == nil
	}
	var g, w *basicerr.Error
	return errors.As(got, &g) && errors.As(want, &w) && *g == *w
}

func lines(l ...string) []string { return l }

func errIn(k basicerr.Kind, line int) error {
	return &basicerr.Error{Kind: k, Line: line, HasLine: true}
}

// @spec INTERP-048
func TestStoreKeepsLinesInOrder(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"out of order", lines(`20 PRINT "B"`, `10 PRINT "A"`, `30 PRINT "C"`, "RUN"), "A\nB\nC\n", nil},
		{"replace", lines(`10 PRINT "A"`, `10 PRINT "B"`, "RUN"), "B\n", nil},
		{"line 0 and 63999", lines(`63999 PRINT "B"`, `0 PRINT "A"`, "RUN"), "A\nB\n", nil},
	})
}

// @spec INTERP-049
func TestStoreEmptyTextDeletes(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"delete", lines(`10 PRINT "A"`, `20 PRINT "B"`, "10", "RUN"), "B\n", nil},
		{"delete missing line", lines(`10 PRINT "A"`, "15", "RUN"), "A\n", nil},
		{"delete only line", lines(`10 PRINT "A"`, "10", "RUN"), "", nil},
	})
}

// @spec INTERP-050
func TestStoreClearsVariables(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"store", lines("A=5", "10 REM", "PRINT A"), " 0 \n", nil},
		{"delete", lines("10 REM", "A$=\"X\"", "10", `PRINT A$;"."`), ".\n", nil},
		{"delete missing line", lines("A=5", "10", "PRINT A"), " 0 \n", nil},
	})
}

// @spec INTERP-051
func TestStoredSyntaxErrorReportedWhenReached(t *testing.T) {
	in, out, err := session(t, `10 PRINT "A"@`, `20 IF 0 THEN @`)
	if err != nil || out != "" {
		t.Errorf("storing: output %q, error %v; want nothing", out, err)
	}
	rec := &recorder{}
	in.out = rec
	err = enter(in, "RUN")
	if rec.String() != "A" || !sameErr(err, errIn(basicerr.Syntax, 10)) {
		t.Errorf("RUN: output %q, error %#v; want \"A\" and SYNTAX IN 10", rec.String(), err)
	}
	runSessionCases(t, []sessionCase{
		{"skipped by IF", lines(`10 IF 0 THEN @`, `20 PRINT "OK"`, "RUN"), "OK\n", nil},
	})
}

// @spec INTERP-052
func TestRunFromFirstLine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"runs every line", lines(`10 PRINT "A";`, `20 PRINT "B"`, "RUN"), "AB\n", nil},
		{"clears variables", lines(`10 PRINT A`, "A=5", "RUN"), " 0 \n", nil},
		{"variables carry between lines", lines("10 A=5", "20 PRINT A", "RUN", "PRINT A"), " 5 \n 5 \n", nil},
		{"empty program", lines("RUN"), "", nil},
		{"twice", lines(`10 PRINT "A"`, "RUN", "RUN"), "A\nA\n", nil},
	})
}

// @spec INTERP-053
func TestRunFromLine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"RUN n", lines(`10 PRINT "A"`, `20 PRINT "B"`, `30 PRINT "C"`, "RUN 20"), "B\nC\n", nil},
		{"last line", lines(`10 PRINT "A"`, `20 PRINT "B"`, "RUN 20"), "B\n", nil},
		{"clears variables", lines(`10 PRINT A`, "A=5", "RUN 10"), " 0 \n", nil},
	})
}

// @spec INTERP-054
func TestRunMissingLine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"direct", lines(`10 PRINT "A"`, "RUN 15"), "", &basicerr.Error{Kind: basicerr.UndefdStatement}},
		{"empty program", lines("RUN 0"), "", &basicerr.Error{Kind: basicerr.UndefdStatement}},
		{"variables cleared", lines(`10 PRINT "A"`, "A=5", "RUN 15", "PRINT A"), " 0 \n", nil},
	})
}

// @spec INTERP-055
func TestFalseIfEndsOnlyItsLine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"next line runs", lines(`10 IF 0 THEN PRINT "A"`, `20 PRINT "B"`, "RUN"), "B\n", nil},
		{"rest of line skipped", lines(`10 IF 0 THEN PRINT "A":PRINT "X"`, `20 PRINT "B"`, "RUN"), "B\n", nil},
	})
}

// @spec INTERP-056
func TestProgramErrorsCarryTheirLine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"error in line 20", lines(`10 PRINT "A"`, `20 PRINT 1/0`, `30 PRINT "C"`, "RUN"), "A\n", errIn(basicerr.DivisionByZero, 20)},
		{"line 0", lines(`0 @`, "RUN"), "", errIn(basicerr.Syntax, 0)},
		{"RUN n in a program", lines(`10 RUN 99`, "RUN"), "", errIn(basicerr.UndefdStatement, 10)},
		{"direct error", lines(`10 PRINT "A"`, "PRINT 1/0"), "", &basicerr.Error{Kind: basicerr.DivisionByZero}},
		{"direct after program error", lines(`10 @`, "RUN", "@"), "", &basicerr.Error{Kind: basicerr.Syntax}},
	})
}

// @spec INTERP-057
func TestProgramCommandsEndTheirLine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"RUN", lines(`10 PRINT "A"`, `RUN:PRINT "X"`), "A\n", nil},
		{"RUN n", lines(`10 PRINT "A"`, `RUN 10:PRINT "X"`), "A\n", nil},
		{"LIST", lines(`LIST:PRINT "X"`), "", nil},
		{"NEW", lines(`NEW:PRINT "X"`), "", nil},
		{"END", lines(`END:PRINT "X"`), "", nil},
		{"END after a statement", lines(`PRINT "A":END:PRINT "X"`), "A\n", nil},
		{"END in a program", lines(`10 PRINT "A":END:PRINT "X"`, "RUN"), "A\n", nil},
		{"GOTO", lines(`10 PRINT "A"`, `GOTO 10:PRINT "X"`), "A\n", nil},
		{"GOTO in a program", lines(`10 GOTO 30:PRINT "X"`, `20 PRINT "B"`, `30 PRINT "C"`, "RUN"), "C\n", nil},
	})
}

// @spec INTERP-058
func TestProgramCommandsEndTheProgram(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"END", lines(`10 PRINT "A"`, "20 END", `30 PRINT "B"`, "RUN"), "A\n", nil},
		{"LIST", lines(`10 LIST`, `20 PRINT "B"`, "RUN"), "\n10 LIST\n20 PRINT \"B\"\n", nil},
		{"NEW", lines(`10 NEW`, `20 PRINT "B"`, "RUN", "LIST", "RUN"), "", nil},
		{"END with IF", lines(`10 IF 1 THEN END`, `20 PRINT "B"`, "RUN"), "", nil},
		{"program continues after direct END", lines("END", `10 PRINT "A"`, "RUN"), "A\n", nil},
	})
}

// @spec INTERP-059
func TestList(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"empty", lines("LIST"), "", nil},
		{"lines in order", lines(`20 PRINT "B"`, `10 PRINT "A"`, "LIST"), "\n10 PRINT \"A\"\n20 PRINT \"B\"\n", nil},
		{"text as typed", lines(`10PRINT  "A" ;  B`, "LIST"), "\n10 PRINT  \"A\" ;  B\n", nil},
		{"question mark", lines(`10 ?"HI":? 1`, "LIST"), "\n10 PRINT\"HI\":PRINT 1\n", nil},
		{"question mark in string and REM", lines(`10 PRINT "?":REM ?`, "LIST"), "\n10 PRINT \"?\":REM ?\n", nil},
		{"syntax error kept", lines(`10 PRINT "A"@`, "LIST"), "\n10 PRINT \"A\"@\n", nil},
		{"after unfinished output", lines(`10 REM`, `PRINT "A";:LIST`), "A\n10 REM\n", nil},
		{"line 0", lines(`0 REM`, "LIST"), "\n0 REM\n", nil},
	})
}

// @spec INTERP-060
func TestNew(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"erases the program", lines(`10 PRINT "A"`, "NEW", "LIST", "RUN"), "", nil},
		{"clears variables", lines("A=5", "NEW", "PRINT A"), " 0 \n", nil},
		{"new lines after NEW", lines(`10 PRINT "A"`, "NEW", `20 PRINT "B"`, "RUN"), "B\n", nil},
	})
}

// @spec INTERP-061
func TestRunInAProgramRestarts(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"RUN n", lines(`10 PRINT "A":END`, `20 PRINT "B":RUN 10`, "RUN 20"), "B\nA\n", nil},
		{"clears variables", lines(`10 PRINT X:END`, `20 X=5:RUN 10`, "RUN 20"), " 0 \n", nil},
	})
}

// @spec INTERP-062
func TestNeverRun(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  bool
	}{
		{"new interpreter", nil, false},
		{"direct lines only", lines(`PRINT "A"`), false},
		{"stored line", lines(`10 PRINT "A"`), true},
		{"after RUN", lines(`10 PRINT "A"`, "RUN"), false},
		{"after failing RUN", lines(`10 PRINT "A"`, "RUN 99"), false},
		{"RUN before storing", lines("RUN", `10 PRINT "A"`), false},
		{"lines deleted", lines(`10 PRINT "A"`, "10"), false},
		{"after NEW", lines(`10 PRINT "A"`, "NEW"), false},
		{"error in program", lines(`10 @`, "RUN"), false},
		{"after GOTO", lines(`10 PRINT "A"`, "GOTO 10"), false},
		{"after failing GOTO", lines(`10 PRINT "A"`, "GOTO 99"), false},
	}
	for _, c := range cases {
		in, _, _ := session(t, c.lines...)
		if got := in.NeverRun(); got != c.want {
			t.Errorf("%s: NeverRun() = %v, want %v", c.name, got, c.want)
		}
	}
}

// @spec INTERP-063
func TestGotoContinuesAtLine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"forward", lines(`10 GOTO 30`, `20 PRINT "B"`, `30 PRINT "C"`, "RUN"), "C\n", nil},
		{"backward", lines(`10 A=A+1:IF A=3 THEN END`, `20 PRINT A;`, `30 GOTO 10`, "RUN"), " 1  2 ", nil},
		{"direct keeps variables", lines(`10 PRINT A`, "A=5", "GOTO 10"), " 5 \n", nil},
		{"direct to a middle line", lines(`10 PRINT "A"`, `20 PRINT "B"`, "GOTO 20"), "B\n", nil},
		{"IF THEN n", lines(`10 IF 1 THEN 30`, `20 PRINT "B"`, `30 PRINT "C"`, "RUN"), "C\n", nil},
		{"IF THEN n false", lines(`10 IF 0 THEN 30`, `20 PRINT "B"`, `30 PRINT "C"`, "RUN"), "B\nC\n", nil},
		{"IF GOTO n", lines(`10 IF 1 GOTO 30`, `20 PRINT "B"`, `30 PRINT "C"`, "RUN"), "C\n", nil},
		{"GO TO", lines(`10 GO TO 30`, `20 PRINT "B"`, `30 PRINT "C"`, "RUN"), "C\n", nil},
		{"direct IF THEN n", lines(`10 PRINT "A"`, "IF 1 THEN 10"), "A\n", nil},
	})
}

// @spec INTERP-064
func TestGotoMissingLine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"in a program", lines(`10 PRINT "A"`, `20 GOTO 99`, "RUN"), "A\n", errIn(basicerr.UndefdStatement, 20)},
		{"direct", lines(`10 PRINT "A"`, "GOTO 99"), "", &basicerr.Error{Kind: basicerr.UndefdStatement}},
		{"empty program", lines("GOTO"), "", &basicerr.Error{Kind: basicerr.UndefdStatement}},
		{"line 0 by default", lines(`0 PRINT "ZERO"`, "GOTO"), "ZERO\n", nil},
	})
}

// syncRecorder is a recorder safe for one goroutine writing while another
// reads.
type syncRecorder struct {
	mu  chan struct{}
	buf strings.Builder
}

func newSyncRecorder() *syncRecorder { return &syncRecorder{mu: make(chan struct{}, 1)} }

func (r *syncRecorder) Write(p []byte) (int, error) {
	r.mu <- struct{}{}
	defer func() { <-r.mu }()
	return r.buf.Write(p)
}

func (r *syncRecorder) String() string {
	r.mu <- struct{}{}
	defer func() { <-r.mu }()
	return r.buf.String()
}

// runUntilInterrupted runs program, a list of stored lines, with RUN, calls
// Interrupt from another goroutine once output appears, and returns the
// output and Exec's error.
func runUntilInterrupted(t *testing.T, program ...string) (string, error) {
	t.Helper()
	rec := newSyncRecorder()
	in := New(rec)
	for _, l := range program {
		if err := enter(in, l); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan error, 1)
	go func() {
		tree, _ := parser.Parse(lexer.Lex("RUN"))
		done <- in.Exec(tree)
	}()
	for rec.String() == "" {
		time.Sleep(time.Millisecond)
	}
	in.Interrupt()
	select {
	case err := <-done:
		return rec.String(), err
	case <-time.After(10 * time.Second):
		t.Fatal("Exec did not stop after Interrupt")
		return "", nil
	}
}

// @spec INTERP-065, INTERP-067
func TestInterruptStopsProgram(t *testing.T) {
	out, err := runUntilInterrupted(t, `10 PRINT "A";`, "20 GOTO 10")
	var be *basicerr.Error
	if !errors.As(err, &be) || be.Kind != basicerr.Break || !be.HasLine || (be.Line != 10 && be.Line != 20) {
		t.Errorf("error %#v, want BREAK in line 10 or 20", err)
	}
	if !regexp.MustCompile(`^A+$`).MatchString(out) {
		t.Errorf("output %q, want A repeated", out)
	}

	_, err = runUntilInterrupted(t, `10 PRINT "A";:GOTO 10`)
	if !sameErr(err, errIn(basicerr.Break, 10)) {
		t.Errorf("one-line loop: error %#v, want BREAK IN 10", err)
	}

	_, err = runUntilInterrupted(t, `10 PRINT "A";:IF 0 THEN @`, "20 GOTO 10")
	if !errors.As(err, &be) || be.Kind != basicerr.Break {
		t.Errorf("false IF: error %#v, want BREAK", err)
	}
}

// @spec INTERP-065
func TestInterruptInDirectMode(t *testing.T) {
	// The first PRINT's output interrupts, as Ctrl-C during it would.
	w := &interruptingWriter{}
	in := New(w)
	w.in = in
	tree, _ := parser.Parse(lexer.Lex(`PRINT "A":PRINT "B"`))
	err := in.Exec(tree)
	if !sameErr(err, &basicerr.Error{Kind: basicerr.Break}) {
		t.Errorf("error %#v, want BREAK with no line", err)
	}
	if w.buf.String() != "A\n" {
		t.Errorf("output %q, want %q", w.buf.String(), "A\n")
	}
}

// interruptingWriter calls Interrupt on its interpreter when written to.
type interruptingWriter struct {
	in  *Interp
	buf strings.Builder
}

func (w *interruptingWriter) Write(p []byte) (int, error) {
	w.in.Interrupt()
	return w.buf.Write(p)
}

// @spec INTERP-066
func TestExecDiscardsEarlierInterrupt(t *testing.T) {
	rec := &recorder{}
	in := New(rec)
	in.Interrupt()
	tree, _ := parser.Parse(lexer.Lex(`PRINT "A":PRINT "B"`))
	if err := in.Exec(tree); err != nil || rec.String() != "A\nB\n" {
		t.Errorf("output %q, error %v; want both lines and no error", rec.String(), err)
	}
}
