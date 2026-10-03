package interp

import (
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
