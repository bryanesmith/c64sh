package interp

import (
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

func stopIn(line int) error {
	return &basicerr.Error{Kind: basicerr.Break, Line: line, HasLine: true, Stopped: true}
}

var cantContinue = &basicerr.Error{Kind: basicerr.CantContinue}

// @spec INTERP-158
func TestClr(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"variables", lines(`A=5:A$="X":DIM B(3):B(1)=2`, "CLR", `PRINT A;A$;B(1)`), " 0  0 \n", nil},
		{"functions", lines(`10 DEF FNA(X)=X`, "RUN", "CLR", "PRINT FNA(1)"), "", &basicerr.Error{Kind: basicerr.UndefdFunction}},
		{"in a running program, the next statement runs", lines(`10 A=5:CLR:PRINT A`, "RUN"), " 0 \n", nil},
		{"the control stack", lines(`10 FOR I=1 TO 2:CLR:NEXT`, "RUN"), "", errIn(basicerr.NextWithoutFor, 10)},
		{"the data pointer", lines(`10 DATA 1,2`, `20 READ A:CLR:READ B:PRINT B`, "RUN"), " 1 \n", nil},
	})
}

// @spec INTERP-159
func TestStop(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"in a program", lines(`10 PRINT "A":STOP:PRINT "B"`, "RUN"), "A\n", stopIn(10)},
		{"on a later line", lines(`10 PRINT "A"`, "20 STOP", `30 PRINT "B"`, "RUN"), "A\n", stopIn(20)},
		{"in direct mode", lines(`PRINT "A":STOP:PRINT "B"`), "A\n", &basicerr.Error{Kind: basicerr.Break, Stopped: true}},
	})
}

// @spec INTERP-160, INTERP-162
func TestCont(t *testing.T) {
	runSessionCases(t, []sessionCase{
		{"after STOP", lines(`10 PRINT "A":STOP:PRINT "B"`, "RUN", "CONT"), "A\nB\n", nil},
		{"after END", lines(`10 PRINT "A":END:PRINT "B"`, "RUN", "CONT"), "A\nB\n", nil},
		{"after the last line, nothing more", lines(`10 PRINT "A"`, "RUN", "CONT"), "A\n", nil},
		{"variables kept", lines(`10 A=1:STOP:PRINT A`, "RUN", "A=7", "CONT"), " 7 \n", nil},
		{"loops kept", lines(`10 FOR I=1 TO 3:PRINT I;:STOP:NEXT`, "RUN", "CONT", "CONT", "CONT"), " 1  2  3 ", nil},
		{"subroutines kept", lines(`10 GOSUB 100:PRINT "BACK"`, "20 END", `100 STOP:RETURN`, "RUN", "CONT"), "BACK\n", nil},
		{"stopped again", lines(`10 STOP:STOP:PRINT "C"`, "RUN", "CONT"), "", stopIn(10)},
		{"direct STOP keeps the place", lines(`10 STOP:PRINT "B"`, "RUN", "STOP", "CONT"), "B\n", nil},
		{"never stopped", lines("CONT"), "", cantContinue},
		{"direct END sets nothing", lines("END", "CONT"), "", cantContinue},
	})
}

// interruptOnce calls Interrupt on its interpreter at its first write.
type interruptOnce struct {
	in   *Interp
	done bool
	buf  strings.Builder
}

func (w *interruptOnce) Write(p []byte) (int, error) {
	if !w.done {
		w.done = true
		w.in.Interrupt()
	}
	return w.buf.Write(p)
}

// interruptedConsole reports an interrupt for its first line, then
// answers with line.
type interruptedConsole struct {
	line  string
	reads int
}

func (c *interruptedConsole) ReadLine(stop func() bool) (string, bool, error) {
	c.reads++
	if c.reads == 1 {
		return "", false, ErrInterrupted
	}
	return c.line, false, nil
}

func (c *interruptedConsole) ReadKey(stop func() bool) (string, error) { return "", nil }

// @spec INTERP-160
func TestContAfterInterrupt(t *testing.T) {
	w := &interruptOnce{}
	in := New(w)
	w.in = in
	enter(in, `10 PRINT "A"`)
	enter(in, `20 PRINT "B"`)
	if err := enter(in, "RUN"); !sameErr(err, errIn(basicerr.Break, 10)) {
		t.Fatalf("RUN: error %#v, want BREAK IN 10", err)
	}
	if err := enter(in, "CONT"); err != nil || w.buf.String() != "A\nB\n" {
		t.Errorf("CONT after an interrupt: output %q, error %v; want the next statement to run", w.buf.String(), err)
	}

	rec := &recorder{}
	in = New(rec)
	in.SetConsole(&interruptedConsole{line: "5"})
	enter(in, `10 INPUT A:PRINT A*2`)
	if err := enter(in, "RUN"); !sameErr(err, errIn(basicerr.Break, 10)) {
		t.Fatalf("INPUT: error %#v, want BREAK IN 10", err)
	}
	if err := enter(in, "CONT"); err != nil || !strings.HasSuffix(rec.String(), " 10 \n") {
		t.Errorf("CONT after an interrupted INPUT: output %q, error %v; want INPUT to ask again", rec.String(), err)
	}
}

// @spec INTERP-161
func TestContCleared(t *testing.T) {
	stopped := []string{`10 STOP:PRINT "B"`, "RUN"}
	runSessionCases(t, []sessionCase{
		{"an error", append(lines(stopped...), "PRINT 1/0", "CONT"), "", cantContinue},
		{"an error in the program", lines(`10 PRINT 1/0`, "RUN", "CONT"), "", cantContinue},
		{"CLR", append(lines(stopped...), "CLR", "CONT"), "", cantContinue},
		{"storing a line", append(lines(stopped...), "20 REM", "CONT"), "", cantContinue},
		{"deleting a line", append(lines(stopped...), "20 REM", "RUN", "20", "CONT"), "", cantContinue},
		{"NEW", append(lines(stopped...), "NEW", "CONT"), "", cantContinue},
		{"a BREAK does not clear it", append(lines(stopped...), "STOP", "CONT"), "B\n", nil},
	})
}

// @spec INTERP-163
func TestContInProgramContinuesAtItself(t *testing.T) {
	out, err := runUntilInterrupted(t, `10 PRINT "A";:CONT:PRINT "B"`)
	if out != "A" || !sameErr(err, errIn(basicerr.Break, 10)) {
		t.Errorf("output %q, error %#v; want A once, then the loop on CONT until BREAK IN 10", out, err)
	}
}

// @spec INTERP-059
func TestListRange(t *testing.T) {
	prog := lines(`10 PRINT "A"`, `20 PRINT "B"`, `30 PRINT "C"`)
	runSessionCases(t, []sessionCase{
		{"one line", append(prog, "LIST 20"), "\n20 PRINT \"B\"\n", nil},
		{"a range", append(prog, "LIST 15-30"), "\n20 PRINT \"B\"\n30 PRINT \"C\"\n", nil},
		{"up to", append(prog, "LIST -20"), "\n10 PRINT \"A\"\n20 PRINT \"B\"\n", nil},
		{"from", append(prog, "LIST 20-"), "\n20 PRINT \"B\"\n30 PRINT \"C\"\n", nil},
		{"no line in range", append(prog, "LIST 40"), "", nil},
		{"in a program, the program ends", lines(`10 LIST 10:PRINT "X"`, "RUN"), "\n10 LIST 10:PRINT \"X\"\n", nil},
	})
}
