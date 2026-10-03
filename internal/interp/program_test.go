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

// @spec INTERP-068
func TestForRunsBody(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"count", lines(`10 FOR I=1 TO 3`, `20 PRINT I;`, `30 NEXT I`, "RUN"), " 1  2  3 ", nil},
		{"step", lines(`10 FOR I=1 TO 10 STEP 4:PRINT I;:NEXT`, "RUN"), " 1  5  9 ", nil},
		{"negative step", lines(`10 FOR I=3 TO 1 STEP -1:PRINT I;:NEXT`, "RUN"), " 3  2  1 ", nil},
		{"fraction step", lines(`10 FOR I=0 TO 1 STEP .5:PRINT I;:NEXT`, "RUN"), " 0  .5  1 ", nil},
		{"end evaluated once", lines(`10 N=3:FOR I=1 TO N:N=1:PRINT I;:NEXT`, "RUN"), " 1  2  3 ", nil},
		{"body runs once even past the end", lines(`10 FOR I=5 TO 1:PRINT I;:NEXT`, "RUN"), " 5 ", nil},
		{"variable after loop", lines(`10 FOR I=1 TO 3:NEXT:PRINT I`, "RUN"), " 4 \n", nil},
		{"nested", lines(`10 FOR I=1 TO 2:FOR J=1 TO 2:PRINT I;J;:NEXT J,I`, "RUN"), " 1  1  1  2  2  1  2  2 ", nil},
		{"across lines", lines(`10 FOR I=1 TO 2`, `20 FOR J=1 TO 2`, `30 PRINT I*10+J;`, `40 NEXT`, `50 NEXT`, "RUN"), " 11  12  21  22 ", nil},
	})
}

// @spec INTERP-069
func TestForTypeMismatch(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"string variable", lines(`10 FOR A$="X" TO "Y"`, "RUN", `PRINT A$`), "X\n", nil},
		{"string variable error", lines(`10 FOR A$="X" TO "Y"`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
		{"string end", lines(`10 FOR I=1 TO "X"`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
		{"string step", lines(`10 FOR I=1 TO 2 STEP "X"`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
		{"string start", lines(`10 FOR I="X" TO 2`, "RUN"), "", errIn(basicerr.TypeMismatch, 10)},
	})
}

// @spec INTERP-070
func TestForReusingVariable(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"restart discards old loop", lines(`10 FOR I=1 TO 2:FOR J=1 TO 9:FOR I=7 TO 8:PRINT I;:NEXT I`, `20 NEXT J`, "RUN"), " 7  8 ", errIn(basicerr.NextWithoutFor, 20)},
		{"same variable twice", lines(`10 FOR I=1 TO 2:FOR I=5 TO 6:PRINT I;:NEXT:NEXT`, "RUN"), " 5  6 ", errIn(basicerr.NextWithoutFor, 10)},
	})
}

// @spec INTERP-071
func TestNextEndTest(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"lands on end", lines(`10 FOR I=1 TO 3 STEP 2:PRINT I;:NEXT`, "RUN"), " 1  3 ", nil},
		{"zero step stops at end", lines(`10 FOR I=1 TO 1 STEP 0:PRINT I;:NEXT`, "RUN"), " 1 ", nil},
		{"zero step changing variable", lines(`10 FOR I=1 TO 3 STEP 0:PRINT I;:I=I+1:NEXT`, "RUN"), " 1  2 ", nil},
		{"body changes variable", lines(`10 FOR I=1 TO 5:PRINT I;:I=I+1:NEXT`, "RUN"), " 1  3  5 ", nil},
	})
}

// @spec INTERP-072
func TestNextSeveralVariables(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"NEXT J,I", lines(`10 FOR I=1 TO 2:FOR J=1 TO 2:NEXT J,I:PRINT I;J`, "RUN"), " 3  3 \n", nil},
		{"statement after NEXT", lines(`10 FOR I=1 TO 2:NEXT:PRINT "DONE"`, "RUN"), "DONE\n", nil},
	})
}

// @spec INTERP-073
func TestNextSearch(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"no loop", lines(`10 NEXT`, "RUN"), "", errIn(basicerr.NextWithoutFor, 10)},
		{"wrong variable", lines(`10 FOR I=1 TO 2:NEXT J`, "RUN"), "", errIn(basicerr.NextWithoutFor, 10)},
		{"bare NEXT is innermost", lines(`10 FOR I=1 TO 2:FOR J=1 TO 2:PRINT J;:NEXT:NEXT`, "RUN"), " 1  2  1  2 ", nil},
		{"outer NEXT drops inner loop", lines(`10 FOR I=1 TO 2:FOR J=1 TO 9:NEXT I:PRINT I;J`, "RUN", "NEXT J"), " 3  1 \n", &basicerr.Error{Kind: basicerr.NextWithoutFor}},
		{"direct", lines("NEXT"), "", &basicerr.Error{Kind: basicerr.NextWithoutFor}},
	})
}

// @spec INTERP-074
func TestLoopsResumeMidLine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"direct mode", lines(`FOR I=1 TO 3:PRINT I;:NEXT`), " 1  2  3 ", nil},
		{"after other statements", lines(`PRINT "X";:FOR I=1 TO 2:PRINT I;:NEXT:PRINT "Y"`), "X 1  2 Y\n", nil},
		{"program mid-line", lines(`10 PRINT "A";:FOR I=1 TO 2:PRINT I;`, `20 NEXT:PRINT "B"`, "RUN"), "A 1  2 B\n", nil},
		{"NEXT after program ends", lines(`10 FOR I=1 TO 3:PRINT I;:END`, "RUN", "NEXT"), " 1  2 ", nil},
	})
}

// @spec INTERP-075
func TestForStackLimit(t *testing.T) {
	ten := `10 FOR A=1 TO 1:FOR B=1 TO 1:FOR C=1 TO 1:FOR D=1 TO 1:FOR E=1 TO 1:FOR F=1 TO 1:FOR G=1 TO 1:FOR H=1 TO 1:FOR I=1 TO 1:FOR J=1 TO 1`
	runSessionCases(t, []sessionCase{
		{"ten nested", lines(ten+`:PRINT "OK"`, "RUN"), "OK\n", nil},
		{"eleven nested", lines(ten+`:FOR K=1 TO 1:PRINT "OK"`, "RUN"), "", errIn(basicerr.OutOfMemory, 10)},
		{"reused variable takes no room", lines(`10 FOR A=1 TO 1:FOR A=1 TO 1:FOR A=1 TO 1:FOR A=1 TO 1:FOR A=1 TO 1:FOR A=1 TO 1:FOR A=1 TO 1:FOR A=1 TO 1:FOR A=1 TO 1:FOR A=1 TO 1:FOR A=1 TO 1:PRINT "OK"`, "RUN"), "OK\n", nil},
	})
}

// @spec INTERP-076
func TestControlStackClearing(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"RUN clears", lines(`10 FOR I=1 TO 3:END`, `20 NEXT`, "RUN", "RUN 20"), "", errIn(basicerr.NextWithoutFor, 20)},
		{"storing clears", lines(`10 FOR I=1 TO 3:END`, "RUN", "20 REM", "NEXT"), "", &basicerr.Error{Kind: basicerr.NextWithoutFor}},
		{"error clears", lines(`10 FOR I=1 TO 3:PRINT 1/0`, "RUN", "NEXT"), "", &basicerr.Error{Kind: basicerr.NextWithoutFor}},
		{"NEW clears", lines(`10 FOR I=1 TO 3:END`, "RUN", "NEW", "NEXT"), "", &basicerr.Error{Kind: basicerr.NextWithoutFor}},
		{"direct loop removed after its line", lines(`FOR I=1 TO 3`, "NEXT"), "", &basicerr.Error{Kind: basicerr.NextWithoutFor}},
	})
}

// @spec INTERP-077
func TestGosub(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"call and return", lines(`10 GOSUB 100:PRINT "BACK"`, `20 END`, `100 PRINT "SUB"`, `110 RETURN`, "RUN"), "SUB\nBACK\n", nil},
		{"returns mid-line", lines(`10 PRINT "A";:GOSUB 100:PRINT "C"`, `20 END`, `100 PRINT "B";:RETURN`, "RUN"), "ABC\n", nil},
		{"nested", lines(`10 GOSUB 100:PRINT "3"`, `20 END`, `100 GOSUB 200:PRINT "2";:RETURN`, `200 PRINT "1";:RETURN`, "RUN"), "123\n", nil},
		{"direct mode", lines(`100 PRINT "SUB";:RETURN`, `GOSUB 100:PRINT "BACK"`), "SUBBACK\n", nil},
		{"keeps variables", lines(`100 PRINT A:RETURN`, "A=5", "GOSUB 100"), " 5 \n", nil},
		{"after THEN", lines(`10 IF 1 THEN GOSUB 100:PRINT "B"`, `20 END`, `100 PRINT "A";:RETURN`, "RUN"), "AB\n", nil},
		{"missing line", lines(`10 GOSUB 99`, "RUN"), "", errIn(basicerr.UndefdStatement, 10)},
	})
}

// @spec INTERP-078
func TestReturnDropsInnerLoops(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"loop left in subroutine", lines(`10 GOSUB 100:PRINT "BACK"`, `20 NEXT`, `100 FOR I=1 TO 9:RETURN`, "RUN"), "BACK\n", errIn(basicerr.NextWithoutFor, 20)},
		{"loop around the call survives", lines(`10 FOR I=1 TO 3:GOSUB 100:NEXT:END`, `100 PRINT I;:RETURN`, "RUN"), " 1  2  3 ", nil},
	})
}

// @spec INTERP-079
func TestReturnWithoutGosub(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"in a program", lines(`10 RETURN`, "RUN"), "", errIn(basicerr.ReturnWithoutGosub, 10)},
		{"only loops", lines(`10 FOR I=1 TO 2:RETURN`, "RUN"), "", errIn(basicerr.ReturnWithoutGosub, 10)},
		{"direct", lines("RETURN"), "", &basicerr.Error{Kind: basicerr.ReturnWithoutGosub}},
		{"falls into subroutine", lines(`10 PRINT "A"`, `20 RETURN`, "RUN"), "A\n", errIn(basicerr.ReturnWithoutGosub, 20)},
	})
}

// @spec INTERP-080
func TestNextCannotLeaveSubroutine(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"NEXT in subroutine", lines(`10 FOR I=1 TO 2:GOSUB 100`, `100 NEXT`, "RUN"), "", errIn(basicerr.NextWithoutFor, 100)},
		{"FOR reusing a variable outside", lines(`10 FOR I=1 TO 2:GOSUB 100:NEXT:END`, `100 FOR I=7 TO 8:PRINT I;:NEXT:RETURN`, "RUN"), " 7  8 ", nil},
	})
}

// @spec INTERP-075
func TestGosubStackLimit(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"26 nested", lines(`10 C=C+1:IF C<26 THEN GOSUB 10`, `20 RETURN`, `30 GOSUB 10:PRINT "OK";C:END`, "RUN 30"), "OK 26 \n", nil},
		{"27 nested", lines(`10 C=C+1:IF C<27 THEN GOSUB 10`, `20 RETURN`, `30 GOSUB 10:PRINT "OK";C:END`, "RUN 30"), "", errIn(basicerr.OutOfMemory, 10)},
		{"runaway recursion", lines(`10 GOSUB 10`, "RUN"), "", errIn(basicerr.OutOfMemory, 10)},
		{"nine loops leave room", lines(`10 FOR A=1 TO 1:FOR B=1 TO 1:FOR C=1 TO 1:FOR D=1 TO 1:FOR E=1 TO 1:FOR F=1 TO 1:FOR G=1 TO 1:FOR H=1 TO 1:FOR I=1 TO 1:GOSUB 100`, `100 PRINT "OK"`, "RUN"), "OK\n", nil},
		{"ten loops leave none", lines(`10 FOR A=1 TO 1:FOR B=1 TO 1:FOR C=1 TO 1:FOR D=1 TO 1:FOR E=1 TO 1:FOR F=1 TO 1:FOR G=1 TO 1:FOR H=1 TO 1:FOR I=1 TO 1:FOR J=1 TO 1:GOSUB 100`, `100 PRINT "OK"`, "RUN"), "", errIn(basicerr.OutOfMemory, 10)},
		{"loop after calls", lines(`10 GOSUB 20`, `20 C=C+1:IF C<25 THEN GOSUB 20`, `30 FOR I=1 TO 1:PRINT "OK"`, "RUN"), "", errIn(basicerr.OutOfMemory, 30)},
	})
}
