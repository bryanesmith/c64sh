package interp

import (
	"fmt"
	"testing"
	"time"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
	"github.com/bryanesmith/c64sh/internal/lexer"
	"github.com/bryanesmith/c64sh/internal/parser"
)

// sline parses one line of source.
func sline(s string) *ast.Line {
	tree, _ := parser.Parse(lexer.Lex(s))
	return tree
}

// @spec INTERP-122
func TestNumberFunctions(t *testing.T) {
	runPrintCases(t, []printCase{
		{"ABS", sline(`PRINT ABS(-3);ABS(2.5);ABS(0)`), " 3  2.5  0 \n"},
		{"INT", sline(`PRINT INT(2.7);INT(-2.5);INT(3)`), " 2 -3  3 \n"},
		{"SGN", sline(`PRINT SGN(-9);SGN(0);SGN(.1)`), "-1  0  1 \n"},
		{"SQR", sline(`PRINT SQR(16);SQR(2)`), " 4  1.41421356 \n"},
		{"LOG EXP", sline(`PRINT LOG(1);EXP(0);EXP(1);LOG(EXP(2))`), " 0  1  2.71828183  2 \n"},
		{"trig", sline(`PRINT SIN(0);COS(0);TAN(0);ATN(1)*4`), " 0  1  0  3.14159265 \n"},
		{"pi", sline(`PRINT π;SIN(π/2)`), " 3.14159265  1 \n"},
	})
	for _, c := range []struct {
		line string
		kind basicerr.Kind
	}{
		{`PRINT ABS("X")`, basicerr.TypeMismatch},
		{`PRINT EXP(100)`, basicerr.Overflow},
	} {
		if _, err := exec(sline(c.line)); !isKind(err, c.kind) {
			t.Errorf("%s: error %v, want kind %d", c.line, err, c.kind)
		}
	}
}

// @spec INTERP-123
func TestNumberFunctionErrors(t *testing.T) {
	for _, c := range []struct {
		line string
		kind basicerr.Kind
	}{
		{`PRINT SQR(-1)`, basicerr.IllegalQuantity},
		{`PRINT LOG(0)`, basicerr.IllegalQuantity},
		{`PRINT LOG(-1)`, basicerr.IllegalQuantity},
	} {
		if _, err := exec(sline(c.line)); !isKind(err, c.kind) {
			t.Errorf("%s: error %v, want kind %d", c.line, err, c.kind)
		}
	}
}

// rndOutput runs lines on a new interpreter with the given clock and
// returns the output.
func rndOutput(t *testing.T, clock time.Time, ls ...string) string {
	t.Helper()
	rec := &recorder{}
	in := New(rec)
	in.SetClock(func() time.Time { return clock })
	for _, l := range ls {
		if err := enter(in, l); err != nil {
			t.Fatal(err)
		}
	}
	return rec.String()
}

// @spec INTERP-124
func TestRnd(t *testing.T) {
	t0 := time.Unix(1000, 0)
	seq := rndOutput(t, t0, `FOR I=1 TO 5:PRINT RND(1);:NEXT`)
	again := rndOutput(t, t0, `FOR I=1 TO 5:PRINT RND(1);:NEXT`)
	if seq != again {
		t.Errorf("sequence differs between interpreters: %q, %q", seq, again)
	}
	a := rndOutput(t, t0, `PRINT RND(-7);RND(1);RND(1)`)
	b := rndOutput(t, t0, `X=RND(5):PRINT RND(-7);RND(1);RND(1)`)
	if a != b {
		t.Errorf("negative seed not repeatable: %q, %q", a, b)
	}
	if c := rndOutput(t, t0, `PRINT RND(-8);RND(1);RND(1)`); c == a {
		t.Errorf("seeds -7 and -8 give the same numbers: %q", c)
	}
	if rndOutput(t, t0, `PRINT RND(0)`) != rndOutput(t, t0, `PRINT RND(0)`) {
		t.Error("RND(0) differs for the same clock")
	}
	if rndOutput(t, t0, `PRINT RND(0)`) == rndOutput(t, time.Unix(2000, 5), `PRINT RND(0)`) {
		t.Error("RND(0) is the same for different clocks")
	}
	out := rndOutput(t, t0, `X=RND(-1):FOR I=1 TO 1000:R=RND(1):B=B-(R<0 OR R>=1):NEXT:PRINT B`, `D=0:FOR I=1 TO 1000:D=D+INT(RND(1)*6+1):NEXT:PRINT D>3000 AND D<4000`)
	if out != " 0 \n-1 \n" {
		t.Errorf("range or spread check printed %q", out)
	}
}

// @spec INTERP-125
func TestRndInitialSeed(t *testing.T) {
	a := rndOutput(t, time.Unix(1, 0), `PRINT RND(1)`)
	b := rndOutput(t, time.Unix(99999, 0), `PRINT RND(1)`)
	if a != b {
		t.Errorf("first RND(1) depends on the clock: %q, %q", a, b)
	}
	rec := &recorder{}
	if err := enter(New(rec), `PRINT RND(0)>=0`); err != nil || rec.String() != "-1 \n" {
		t.Errorf("RND(0) with the system clock: %q, %v", rec.String(), err)
	}
}

// @spec INTERP-157
func TestFre(t *testing.T) {
	fre := func(lines ...string) string {
		t.Helper()
		rec := &recorder{}
		in := New(rec)
		for _, l := range lines {
			if err := enter(in, l); err != nil {
				t.Fatalf("%s: %v", l, err)
			}
		}
		rec.writes = nil
		if err := enter(in, `PRINT FRE(0)-65536*(FRE(0)<0);FRE("X")-65536*(FRE("X")<0)`); err != nil {
			t.Fatal(err)
		}
		return rec.String()
	}
	cases := []struct {
		name  string
		lines []string
		free  int
	}{
		{"nothing in use", nil, 38909},
		{"a number variable", []string{"A=1"}, 38909 - 7},
		{"a string variable", []string{`A$="HELLO"`}, 38909 - 7 - 5},
		// DEF and FN are one byte each: 5 + len(`D FA(X)=X`), then 7 for the function.
		{"a function", []string{"10 DEF FNA(X)=X", "RUN"}, 38909 - 14 - 7},
		{"an array", []string{"DIM B(9)"}, 38909 - (5 + 2 + 5*10)},
		{"a string array element", []string{`B$(1)="AB"`}, 38909 - (5 + 2 + 3*11) - 2},
		// PRINT is one byte: 5 + len(`P"HI"`) = 5 + 5.
		{"a program line", []string{`10 PRINT"HI"`}, 38909 - 10},
		// Spaces are kept; REM is one byte: 5 + len(`R HELLO`).
		{"a comment line", []string{`20 REM HELLO`}, 38909 - 12},
	}
	for _, c := range cases {
		want := fmt.Sprintf(" %d  %d \n", c.free, c.free)
		if got := fre(c.lines...); got != want {
			t.Errorf("%s: %q, want %q", c.name, got, want)
		}
	}

	rec := &recorder{}
	in := New(rec)
	if err := enter(in, "PRINT FRE(0)"); err != nil || rec.String() != "-26627 \n" {
		t.Errorf("signed result: %q, %v; want -26627 as a C64 shows 38909", rec.String(), err)
	}
	rec = &recorder{}
	in = New(rec)
	enter(in, `A$="X":FOR I=1 TO 16:A$=A$+A$:NEXT`) // 65536 characters
	if err := enter(in, "PRINT FRE(0)"); err != nil || rec.String() != " 0 \n" {
		t.Errorf("more in use than a C64 holds: %q, %v; want 0", rec.String(), err)
	}
}
