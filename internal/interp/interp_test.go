package interp

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/ast"
	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// AST helpers.
func str(v string) *ast.StringLit                     { return &ast.StringLit{Value: v} }
func cat(l, r ast.Expr) *ast.BinaryExpr               { return &ast.BinaryExpr{Op: ast.Add, Left: l, Right: r} }
func num(v float64) *ast.NumberLit                    { return &ast.NumberLit{Value: v} }
func bin(op ast.Op, l, r ast.Expr) *ast.BinaryExpr    { return &ast.BinaryExpr{Op: op, Left: l, Right: r} }
func neg(x ast.Expr) *ast.NegExpr                     { return &ast.NegExpr{X: x} }
func item(e ast.Expr) *ast.ExprItem                   { return &ast.ExprItem{Expr: e} }
func printStmt(items ...ast.PrintItem) *ast.PrintStmt { return &ast.PrintStmt{Items: items} }
func line(stmts ...ast.Stmt) *ast.Line                { return &ast.Line{Statements: stmts} }

var (
	semi  = &ast.Semicolon{}
	comma = &ast.Comma{}
)

func spaces(n int) string { return strings.Repeat(" ", n) }

func syntaxErr() error { return &basicerr.Error{Kind: basicerr.Syntax} }

func isKind(err error, k basicerr.Kind) bool {
	var be *basicerr.Error
	return errors.As(err, &be) && be.Kind == k
}

// recorder records each call to Write.
type recorder struct{ writes []string }

func (r *recorder) Write(p []byte) (int, error) {
	r.writes = append(r.writes, string(p))
	return len(p), nil
}

func (r *recorder) String() string { return strings.Join(r.writes, "") }

// exec runs l and returns the output and error.
func exec(l *ast.Line) (*recorder, error) {
	rec := &recorder{}
	err := New(rec).Exec(l)
	return rec, err
}

type printCase struct {
	name string
	line *ast.Line
	want string
}

func runPrintCases(t *testing.T, cases []printCase) {
	t.Helper()
	for _, c := range cases {
		rec, err := exec(c.line)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
		}
		if got := rec.String(); got != c.want {
			t.Errorf("%s: output %q, want %q", c.name, got, c.want)
		}
	}
}

// @spec INTERP-001
func TestStatementsRunInOrderAndStopAtFailure(t *testing.T) {
	runPrintCases(t, []printCase{
		{"two statements", line(printStmt(item(str("A"))), printStmt(item(str("B")))), "A\nB\n"},
	})

	long := strings.Repeat("x", 200)
	rec, err := exec(line(
		printStmt(item(str("A"))),
		printStmt(item(cat(str(long), str(long)))),
		printStmt(item(str("B"))),
	))
	if !isKind(err, basicerr.StringTooLong) {
		t.Errorf("error = %v, want STRING TOO LONG", err)
	}
	if got := rec.String(); got != "A\n" {
		t.Errorf("output %q, want %q (statement after the failure must not run)", got, "A\n")
	}
}

// @spec INTERP-002
func TestOutputGoesToGivenWriter(t *testing.T) {
	runPrintCases(t, []printCase{
		{"hello", line(printStmt(item(str("HELLO")))), "HELLO\n"},
	})
}

// @spec INTERP-003
func TestUnhandledNodeTypesPanic(t *testing.T) {
	cases := map[string]*ast.Line{
		"statement":  line(nil),
		"print item": line(&ast.PrintStmt{Items: []ast.PrintItem{nil}}),
		"expression": line(printStmt(&ast.ExprItem{Expr: nil})),
	}
	for name, l := range cases {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("%s: nil node did not panic", name)
					return
				}
				if msg := fmt.Sprint(r); !strings.Contains(msg, "<nil>") {
					t.Errorf("%s: panic message %q does not name the node type", name, msg)
				}
			}()
			exec(l)
		}()
	}
}

// @spec INTERP-014
func TestRemDoesNothing(t *testing.T) {
	rem := &ast.RemStmt{Text: " HELLO"}
	runPrintCases(t, []printCase{
		{"comment alone", line(rem), ""},
		{"comment after a print", line(printStmt(item(str("A")), semi), rem), "A"},
		{"statement after a comment runs", line(rem, printStmt(item(str("B")))), "B\n"},
	})
	rec, _ := exec(line(rem))
	if len(rec.writes) != 0 {
		t.Errorf("REM made %d writes, want 0", len(rec.writes))
	}
}

// @spec INTERP-004
func TestPrintItems(t *testing.T) {
	runPrintCases(t, []printCase{
		{"semicolon joins", line(printStmt(item(str("A")), semi, item(str("B")))), "AB\n"},
		{"comma moves to the next zone", line(printStmt(item(str("A")), comma, item(str("B")))), "A" + spaces(9) + "B\n"},
		{"adjacent items", line(printStmt(item(str("A")), item(str("B")))), "AB\n"},
		{"leading comma", line(printStmt(comma, item(str("A")))), spaces(10) + "A\n"},
		{"mixed", line(printStmt(item(str("A")), semi, item(str("B")), comma, item(str("C")))), "AB" + spaces(8) + "C\n"},
	})
}

// @spec INTERP-005
func TestPrintEndsWithNewline(t *testing.T) {
	runPrintCases(t, []printCase{
		{"PRINT alone", line(printStmt()), "\n"},
		{"one item", line(printStmt(item(str("A")))), "A\n"},
		{"separator then item", line(printStmt(item(str("A")), semi, item(str("B")))), "AB\n"},
	})
}

// @spec INTERP-006
func TestTrailingSeparatorSuppressesNewline(t *testing.T) {
	runPrintCases(t, []printCase{
		{"trailing semicolon", line(printStmt(item(str("A")), semi)), "A"},
		{"trailing comma", line(printStmt(item(str("A")), comma)), "A" + spaces(9)},
		{"only a semicolon", line(printStmt(semi)), ""},
	})
}

// @spec INTERP-007
func TestPrintWritesOnce(t *testing.T) {
	rec, err := exec(line(printStmt(item(str("A")), semi, item(str("B")), comma, item(str("C")))))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if len(rec.writes) != 1 {
		t.Errorf("PRINT made %d writes %q, want 1", len(rec.writes), rec.writes)
	}
}

// @spec INTERP-008
func TestFailingItemWritesEarlierItems(t *testing.T) {
	long := strings.Repeat("x", 200)
	cases := []struct {
		name    string
		line    *ast.Line
		want    string
		wantErr basicerr.Kind
	}{
		{"bad item after items",
			line(printStmt(item(str("A")), semi, &ast.BadItem{Err: syntaxErr()})),
			"A", basicerr.Syntax},
		{"bad item first",
			line(printStmt(&ast.BadItem{Err: syntaxErr()})),
			"", basicerr.Syntax},
		{"failing expression after items",
			line(printStmt(item(str("A")), semi, item(cat(str(long), str(long))), item(str("B")))),
			"A", basicerr.StringTooLong},
	}
	for _, c := range cases {
		rec, err := exec(c.line)
		if !isKind(err, c.wantErr) {
			t.Errorf("%s: error = %v, want kind %d", c.name, err, c.wantErr)
		}
		if got := rec.String(); got != c.want {
			t.Errorf("%s: output %q, want %q", c.name, got, c.want)
		}
		if len(rec.writes) > 1 {
			t.Errorf("%s: made %d writes, want at most 1", c.name, len(rec.writes))
		}
	}
}

// @spec INTERP-009
func TestStringLiteralValueUnchanged(t *testing.T) {
	v := "a:\t;é\xff"
	runPrintCases(t, []printCase{
		{"special contents", line(printStmt(item(str(v)))), v + "\n"},
	})
}

// @spec INTERP-010
func TestConcat(t *testing.T) {
	runPrintCases(t, []printCase{
		{"two", line(printStmt(item(cat(str("FOO"), str("BAR"))))), "FOOBAR\n"},
		{"nested", line(printStmt(item(cat(cat(str("A"), str("B")), str("C"))))), "ABC\n"},
	})
}

// @spec INTERP-011
func TestConcatLongerThan255IsStringTooLong(t *testing.T) {
	cases := []struct {
		name        string
		left, right string
		wantErr     bool
	}{
		{"255 ASCII", strings.Repeat("x", 200), strings.Repeat("y", 55), false},
		{"256 ASCII", strings.Repeat("x", 200), strings.Repeat("y", 56), true},
		{"255 two-byte characters", strings.Repeat("é", 128), strings.Repeat("é", 127), false},
		{"256 two-byte characters", strings.Repeat("é", 128), strings.Repeat("é", 128), true},
		{"255 with invalid bytes", strings.Repeat("\xff", 200), strings.Repeat("y", 55), false},
		{"256 with invalid bytes", strings.Repeat("\xff", 200), strings.Repeat("y", 56), true},
	}
	for _, c := range cases {
		rec, err := exec(line(printStmt(item(cat(str(c.left), str(c.right))))))
		if c.wantErr {
			if !isKind(err, basicerr.StringTooLong) {
				t.Errorf("%s: error = %v, want STRING TOO LONG", c.name, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
		}
		if want := c.left + c.right + "\n"; rec.String() != want {
			t.Errorf("%s: output has %d bytes, want %d", c.name, len(rec.String()), len(want))
		}
	}
}

// @spec INTERP-012
func TestLongStringLiteralIsNotLimited(t *testing.T) {
	long := strings.Repeat("x", 1000)
	runPrintCases(t, []printCase{
		{"1000-character literal", line(printStmt(item(str(long)))), long + "\n"},
	})
}

// failWriter fails every write.
type failWriter struct{ calls int }

var errBoom = errors.New("boom")

func (w *failWriter) Write(p []byte) (int, error) {
	w.calls++
	return 0, errBoom
}

// @spec INTERP-013
func TestWriteErrorIsReturnedUnchanged(t *testing.T) {
	w := &failWriter{}
	err := New(w).Exec(line(printStmt(item(str("A"))), printStmt(item(str("B")))))
	if err != errBoom {
		t.Errorf("error = %v, want the writer's error unchanged", err)
	}
	if w.calls != 1 {
		t.Errorf("writer called %d times, want 1 (no statements after the failure)", w.calls)
	}
}

// @spec INTERP-015
func TestCommaMovesToNextPrintZone(t *testing.T) {
	runPrintCases(t, []printCase{
		{"from column 0, a full zone", line(printStmt(comma, item(str("X")))), spaces(10) + "X\n"},
		{"from column 1", line(printStmt(item(str("A")), comma, item(str("X")))), "A" + spaces(9) + "X\n"},
		{"from column 9", line(printStmt(item(str("123456789")), comma, item(str("X")))), "123456789 X\n"},
		{"from a zone start, a full zone", line(printStmt(item(str("0123456789")), comma, item(str("X")))), "0123456789" + spaces(10) + "X\n"},
		{"from column 11", line(printStmt(item(str("01234567890")), comma, item(str("X")))), "01234567890" + spaces(9) + "X\n"},
		{"columns 0, 10, 20", line(printStmt(item(str("A")), comma, item(str("B")), comma, item(str("C")))), "A" + spaces(9) + "B" + spaces(9) + "C\n"},
		{"two commas", line(printStmt(item(str("A")), comma, comma, item(str("C")))), "A" + spaces(19) + "C\n"},
		{"past column 40", line(printStmt(item(str(strings.Repeat("x", 45))), comma, item(str("X")))), strings.Repeat("x", 45) + spaces(5) + "X\n"},
		{"multi-byte characters count once", line(printStmt(item(str("éé")), comma, item(str("X")))), "éé" + spaces(8) + "X\n"},
	})
}

// @spec INTERP-016
func TestColumnCarriesAcrossStatementsAndLines(t *testing.T) {
	rec := &recorder{}
	in := New(rec)
	if got := in.Column(); got != 0 {
		t.Errorf("initial column %d, want 0", got)
	}
	steps := []struct {
		line *ast.Line
		want int
	}{
		{line(printStmt(item(str("AB")), semi)), 2},
		{line(printStmt(item(str("é\xff")), semi)), 4},
		{line(printStmt(item(str("C")))), 0},
		{line(printStmt(item(str("XY")), semi), printStmt(item(str("Z")), comma)), 10},
	}
	for i, s := range steps {
		if err := in.Exec(s.line); err != nil {
			t.Fatal(err)
		}
		if got := in.Column(); got != s.want {
			t.Errorf("after step %d: column %d, want %d", i+1, got, s.want)
		}
	}

	// A comma on a later line uses the column left by an earlier line.
	rec2 := &recorder{}
	in2 := New(rec2)
	in2.Exec(line(printStmt(item(str("AB")), semi)))
	in2.Exec(line(printStmt(comma, item(str("X")))))
	if got, want := rec2.String(), "AB"+spaces(8)+"X\n"; got != want {
		t.Errorf("output %q, want %q", got, want)
	}
}

// @spec INTERP-017
func TestFreshLine(t *testing.T) {
	rec := &recorder{}
	in := New(rec)
	if err := in.FreshLine(); err != nil || rec.String() != "" {
		t.Errorf("at column 0: wrote %q (err %v), want nothing", rec.String(), err)
	}
	in.Exec(line(printStmt(item(str("A")), semi)))
	if err := in.FreshLine(); err != nil {
		t.Fatal(err)
	}
	if got := rec.String(); got != "A\n" {
		t.Errorf("mid-line: output %q, want %q", got, "A\n")
	}
	if got := in.Column(); got != 0 {
		t.Errorf("column after FreshLine %d, want 0", got)
	}
	if err := New(&failWriter{}).FreshLine(); err != nil {
		t.Errorf("FreshLine at column 0 with a failing writer returned %v, want nil (nothing written)", err)
	}
	failing := New(&failWriter{})
	failing.column = 3
	if err := failing.FreshLine(); err != errBoom {
		t.Errorf("FreshLine write error = %v, want the writer's error", err)
	}
}

// shortWriter writes at most n bytes, then fails.
type shortWriter struct {
	n   int
	got string
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) <= w.n {
		w.n -= len(p)
		w.got += string(p)
		return len(p), nil
	}
	written := w.n
	w.got += string(p[:written])
	w.n = 0
	return written, errBoom
}

// @spec INTERP-018
func TestColumnAdvancesOnlyByWrittenCharacters(t *testing.T) {
	failing := New(&failWriter{})
	failing.Exec(line(printStmt(item(str("ABC")), semi)))
	if got := failing.Column(); got != 0 {
		t.Errorf("after a failed write: column %d, want 0", got)
	}
	w := &shortWriter{n: 2}
	short := New(w)
	short.Exec(line(printStmt(item(str("ABCD")), semi)))
	if got := short.Column(); got != 2 {
		t.Errorf("after a 2-byte partial write: column %d, want 2", got)
	}
}

// @spec INTERP-004, INTERP-019
func TestNumberFormat(t *testing.T) {
	cases := []struct {
		value float64
		want  string
	}{
		{0, " 0 "},
		{45, " 45 "},
		{100, " 100 "},
		{3.14, " 3.14 "},
		{0.5, " .5 "},
		{1.0 / 3, " .333333333 "},
		{2.0 / 3, " .666666667 "},
		{0.01, " .01 "},
		{0.0123, " .0123 "},
		{0.001, " 1E-03 "},
		{1.5e-10, " 1.5E-10 "},
		{123.456789123, " 123.456789 "},
		{1e8, " 100000000 "},
		{123456789, " 123456789 "},
		{999999999, " 999999999 "},
		{999999999.6, " 1E+09 "},
		{1e9, " 1E+09 "},
		{1234567890, " 1.23456789E+09 "},
		{1e38, " 1E+38 "},
		{1.70141183e38, " 1.70141183E+38 "},
		{2.93873588e-39, " 2.93873588E-39 "},
		{-2.5, "-2.5 "},
		{-0.001, "-1E-03 "},
		{-45, "-45 "},
	}
	for _, c := range cases {
		rec, err := exec(line(printStmt(item(num(c.value)), semi)))
		if err != nil {
			t.Errorf("%v: unexpected error %v", c.value, err)
			continue
		}
		if got := rec.String(); got != c.want {
			t.Errorf("PRINT %v; wrote %q, want %q", c.value, got, c.want)
		}
	}
}

// @spec INTERP-020
func TestNumberLiteralsPrint(t *testing.T) {
	runPrintCases(t, []printCase{
		{"number", line(printStmt(item(num(5)))), " 5 \n"},
		{"string then number", line(printStmt(item(str("5*9=")), semi, item(num(45)))), "5*9= 45 \n"},
		{"numbers side by side", line(printStmt(item(num(1)), semi, item(num(2)))), " 1  2 \n"},
		{"numbers in zones", line(printStmt(item(num(2)), comma, item(num(3)))), " 2 " + spaces(7) + " 3 \n"},
	})
}

// @spec INTERP-021
func TestAddNumbers(t *testing.T) {
	runPrintCases(t, []printCase{
		{"integers", line(printStmt(item(cat(num(1), num(2))))), " 3 \n"},
		{"decimals", line(printStmt(item(cat(num(0.1), num(0.2))))), " .3 \n"},
		{"chain", line(printStmt(item(cat(cat(num(1), num(2)), num(3))))), " 6 \n"},
	})
}

// @spec INTERP-022
func TestAddStringAndNumberIsTypeMismatch(t *testing.T) {
	cases := map[string]*ast.Line{
		"string + number": line(printStmt(item(str("A")), semi, item(cat(str("B"), num(1))))),
		"number + string": line(printStmt(item(str("A")), semi, item(cat(num(1), str("B"))))),
	}
	for name, l := range cases {
		rec, err := exec(l)
		if !isKind(err, basicerr.TypeMismatch) {
			t.Errorf("%s: error = %v, want TYPE MISMATCH", name, err)
		}
		if got := rec.String(); got != "A" {
			t.Errorf("%s: output %q, want %q (items before the failure)", name, got, "A")
		}
	}
}

// @spec INTERP-023
func TestOverflow(t *testing.T) {
	cases := map[string]ast.Expr{
		"literal above the maximum": num(1.8e38),
		"infinite literal":          num(math.Inf(1)),
		"sum above the maximum":     cat(num(1e38), num(1e38)),
	}
	for name, e := range cases {
		if _, err := exec(line(printStmt(item(e)))); !isKind(err, basicerr.Overflow) {
			t.Errorf("%s: error = %v, want OVERFLOW", name, err)
		}
	}
	runPrintCases(t, []printCase{
		{"maximum is fine", line(printStmt(item(num(1.70141183e38)))), " 1.70141183E+38 \n"},
	})
}

// @spec INTERP-024
func TestUnderflowBecomesZero(t *testing.T) {
	runPrintCases(t, []printCase{
		{"tiny literal", line(printStmt(item(num(1e-40)))), " 0 \n"},
		{"tiny sum", line(printStmt(item(cat(num(1e-40), num(1e-41))))), " 0 \n"},
		{"smallest is kept", line(printStmt(item(num(2.93873588e-39)))), " 2.93873588E-39 \n"},
	})
}

// @spec INTERP-025
func TestLeftOperandErrorWins(t *testing.T) {
	cases := map[string]ast.Expr{
		"overflow left of a string":  cat(num(1e39), str("A")),
		"overflow right of a string": cat(str("A"), num(1e39)),
	}
	for name, e := range cases {
		if _, err := exec(line(printStmt(item(e)))); !isKind(err, basicerr.Overflow) {
			t.Errorf("%s: error = %v, want OVERFLOW", name, err)
		}
	}
}

// @spec INTERP-016
func TestNumberOutputCountsColumns(t *testing.T) {
	rec := &recorder{}
	in := New(rec)
	in.Exec(line(printStmt(item(num(5)), semi)))
	if got := in.Column(); got != 3 {
		t.Errorf("after PRINT 5; column %d, want 3", got)
	}
}

// @spec INTERP-026
func TestSubtractMultiplyDivide(t *testing.T) {
	runPrintCases(t, []printCase{
		{"subtract", line(printStmt(item(bin(ast.Sub, num(8), num(2))))), " 6 \n"},
		{"negative result", line(printStmt(item(bin(ast.Sub, num(2), num(8))))), "-6 \n"},
		{"multiply", line(printStmt(item(bin(ast.Mul, num(5), num(9))))), " 45 \n"},
		{"divide", line(printStmt(item(bin(ast.Div, num(1), num(3))))), " .333333333 \n"},
		{"divide evenly", line(printStmt(item(bin(ast.Div, num(10), num(4))))), " 2.5 \n"},
		{"divide then multiply", line(printStmt(item(bin(ast.Mul, bin(ast.Div, num(1), num(3)), num(3))))), " 1 \n"},
	})
}

// @spec INTERP-027
func TestArithmeticOnStringsIsTypeMismatch(t *testing.T) {
	cases := map[string]ast.Expr{
		"string - number": bin(ast.Sub, str("A"), num(1)),
		"number * string": bin(ast.Mul, num(2), str("A")),
		"string / string": bin(ast.Div, str("A"), str("B")),
		"string / zero":   bin(ast.Div, str("A"), num(0)),
		"negated string":  neg(str("A")),
		"string - string": bin(ast.Sub, str("A"), str("B")),
	}
	for name, e := range cases {
		if _, err := exec(line(printStmt(item(e)))); !isKind(err, basicerr.TypeMismatch) {
			t.Errorf("%s: error = %v, want TYPE MISMATCH", name, err)
		}
	}
}

// @spec INTERP-028
func TestDivisionByZero(t *testing.T) {
	rec, err := exec(line(printStmt(item(str("A")), semi, item(bin(ast.Div, num(1), num(0))))))
	if !isKind(err, basicerr.DivisionByZero) {
		t.Errorf("error = %v, want DIVISION BY ZERO", err)
	}
	if got := rec.String(); got != "A" {
		t.Errorf("output %q, want %q (items before the failure)", got, "A")
	}
	if _, err := exec(line(printStmt(item(bin(ast.Div, num(0), num(0)))))); !isKind(err, basicerr.DivisionByZero) {
		t.Errorf("0/0: error = %v, want DIVISION BY ZERO", err)
	}
	runPrintCases(t, []printCase{
		{"zero divided", line(printStmt(item(bin(ast.Div, num(0), num(5))))), " 0 \n"},
	})
}

// @spec INTERP-029, INTERP-019
func TestNegate(t *testing.T) {
	runPrintCases(t, []printCase{
		{"negate", line(printStmt(item(neg(num(5))))), "-5 \n"},
		{"double negate", line(printStmt(item(neg(neg(num(5)))))), " 5 \n"},
		{"negate a decimal", line(printStmt(item(neg(num(0.5))))), "-.5 \n"},
		{"negative zero prints as zero", line(printStmt(item(neg(num(0))))), " 0 \n"},
		{"zero times negative", line(printStmt(item(bin(ast.Mul, num(0), neg(num(1)))))), " 0 \n"},
	})
}

// @spec INTERP-023, INTERP-024
func TestArithmeticRange(t *testing.T) {
	for name, e := range map[string]ast.Expr{
		"product":           bin(ast.Mul, num(1e38), num(10)),
		"difference":        bin(ast.Sub, neg(num(1e38)), num(1e38)),
		"quotient":          bin(ast.Div, num(1e38), num(0.1)),
		"intermediate only": bin(ast.Div, bin(ast.Mul, num(1e38), num(10)), num(100)),
	} {
		if _, err := exec(line(printStmt(item(e)))); !isKind(err, basicerr.Overflow) {
			t.Errorf("%s: error = %v, want OVERFLOW", name, err)
		}
	}
	runPrintCases(t, []printCase{
		{"tiny product", line(printStmt(item(bin(ast.Mul, num(1e-30), num(1e-30))))), " 0 \n"},
		{"tiny quotient", line(printStmt(item(bin(ast.Div, num(1e-30), num(1e30))))), " 0 \n"},
	})
}

// @spec INTERP-030
func TestPowerFunction(t *testing.T) {
	pow := func(l, r float64) ast.Expr { return bin(ast.Pow, num(l), num(r)) }
	runPrintCases(t, []printCase{
		{"power", line(printStmt(item(pow(2, 3)))), " 8 \n"},
		{"square root", line(printStmt(item(pow(9, 0.5)))), " 3 \n"},
		{"negative exponent", line(printStmt(item(pow(2, -1)))), " .5 \n"},
		{"anything to the 0 is 1", line(printStmt(item(pow(5, 0)))), " 1 \n"},
		{"0 to the 0 is 1", line(printStmt(item(pow(0, 0)))), " 1 \n"},
		{"0 to a power is 0", line(printStmt(item(pow(0, 5)))), " 0 \n"},
		{"0 to a negative power is 0", line(printStmt(item(pow(0, -1)))), " 0 \n"},
		{"negative base, even power", line(printStmt(item(bin(ast.Pow, neg(num(2)), num(2))))), " 4 \n"},
		{"negative base, odd power", line(printStmt(item(bin(ast.Pow, neg(num(2)), num(3))))), "-8 \n"},
		{"negative base, negative odd power", line(printStmt(item(bin(ast.Pow, neg(num(2)), neg(num(1)))))), "-.5 \n"},
		{"negative base, zero power", line(printStmt(item(bin(ast.Pow, neg(num(2)), num(0))))), " 1 \n"},
		{"fractional result", line(printStmt(item(pow(2, 0.5)))), " 1.41421356 \n"},
		{"tiny result becomes 0", line(printStmt(item(pow(10, -50)))), " 0 \n"},
	})
	if _, err := exec(line(printStmt(item(pow(10, 39))))); !isKind(err, basicerr.Overflow) {
		t.Errorf("10^39: error = %v, want OVERFLOW", err)
	}
	if _, err := exec(line(printStmt(item(bin(ast.Pow, str("A"), num(2)))))); !isKind(err, basicerr.TypeMismatch) {
		t.Errorf(`"A"^2: error = %v, want TYPE MISMATCH`, err)
	}
}

// @spec INTERP-031
func TestNegativeBaseFractionalPowerIsIllegalQuantity(t *testing.T) {
	rec, err := exec(line(printStmt(item(str("A")), semi, item(bin(ast.Pow, neg(num(8)), num(1.0/3))))))
	if !isKind(err, basicerr.IllegalQuantity) {
		t.Errorf("error = %v, want ILLEGAL QUANTITY", err)
	}
	if got := rec.String(); got != "A" {
		t.Errorf("output %q, want %q (items before the failure)", got, "A")
	}
}

func vr(name string) *ast.VarRef { return &ast.VarRef{Name: name, Text: name} }
func let(name string, e ast.Expr) *ast.LetStmt {
	return &ast.LetStmt{Var: vr(name), Value: e}
}

// @spec INTERP-032, INTERP-033
func TestVariablesStoreAndPersist(t *testing.T) {
	rec := &recorder{}
	in := New(rec)
	steps := []*ast.Line{
		line(let("A", num(5)), printStmt(item(vr("A")))),
		line(printStmt(item(bin(ast.Mul, vr("A"), num(2))))),
		line(let("A", bin(ast.Add, vr("A"), num(1))), printStmt(item(vr("A")))),
		line(let("N$", str("HI")), printStmt(item(vr("N$")), semi, item(str("!")))),
		line(printStmt(item(vr("A")), semi, item(vr("N$")))),
	}
	for _, l := range steps {
		if err := in.Exec(l); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := rec.String(), " 5 \n 10 \n 6 \nHI!\n 6 HI\n"; got != want {
		t.Errorf("output %q, want %q", got, want)
	}
}

// @spec INTERP-033
func TestUnsetVariables(t *testing.T) {
	runPrintCases(t, []printCase{
		{"number variable", line(printStmt(item(vr("X")))), " 0 \n"},
		{"string variable", line(printStmt(item(vr("X$")))), "\n"},
		{"number and string are separate", line(let("A", num(1)), printStmt(item(vr("A$")), semi, item(str("|")))), "|\n"},
		{"integer variable", line(printStmt(item(vr("X%")))), " 0 \n"},
		{"number and integer are separate", line(let("A", num(1)), printStmt(item(vr("A%")))), " 0 \n"},
	})
}

// @spec INTERP-034
func TestAssignWrongTypeIsTypeMismatch(t *testing.T) {
	for name, l := range map[string]*ast.LetStmt{
		"string to number variable":  let("A", str("HI")),
		"number to string variable":  let("A$", num(5)),
		"string to integer variable": let("A%", str("HI")),
	} {
		rec := &recorder{}
		in := New(rec)
		in.Exec(line(let("A", num(1)), let("A$", str("OLD"))))
		if err := in.Exec(line(l)); !isKind(err, basicerr.TypeMismatch) {
			t.Errorf("%s: error = %v, want TYPE MISMATCH", name, err)
		}
		in.Exec(line(printStmt(item(vr("A")), semi, item(vr("A$")))))
		if got := rec.String(); got != " 1 OLD\n" {
			t.Errorf("%s: variables after the error printed %q, want unchanged %q", name, got, " 1 OLD\n")
		}
	}
}

// @spec INTERP-035
func TestAssignLongStringIsStringTooLong(t *testing.T) {
	rec := &recorder{}
	in := New(rec)
	if err := in.Exec(line(let("A$", str(strings.Repeat("x", 255))))); err != nil {
		t.Fatalf("255 characters: unexpected error %v", err)
	}
	if err := in.Exec(line(let("A$", str(strings.Repeat("y", 256))))); !isKind(err, basicerr.StringTooLong) {
		t.Errorf("256 characters: error = %v, want STRING TOO LONG", err)
	}
	in.Exec(line(printStmt(item(vr("A$")))))
	if got := rec.String(); got != strings.Repeat("x", 255)+"\n" {
		t.Errorf("variable changed after the error")
	}
}

// @spec INTERP-036
func TestFailedAssignmentLeavesVariable(t *testing.T) {
	rec := &recorder{}
	in := New(rec)
	in.Exec(line(let("A", num(7))))
	if err := in.Exec(line(let("A", bin(ast.Div, num(1), num(0))))); !isKind(err, basicerr.DivisionByZero) {
		t.Errorf("error = %v, want DIVISION BY ZERO", err)
	}
	in.Exec(line(printStmt(item(vr("A")))))
	if got := rec.String(); got != " 7 \n" {
		t.Errorf("output %q, want %q", got, " 7 \n")
	}
}

// @spec INTERP-037
func TestIntegerVariablesRoundDown(t *testing.T) {
	cases := []struct {
		value float64
		want  string
	}{
		{3.7, " 3 \n"},
		{-3.7, "-4 \n"},
		{5, " 5 \n"},
		{-5, "-5 \n"},
		{0.5, " 0 \n"},
		{-0.5, "-1 \n"},
		{32767.9, " 32767 \n"},
		{-32767.5, "-32768 \n"},
		{-32768, "-32768 \n"},
	}
	for _, c := range cases {
		rec, err := exec(line(let("C%", num(c.value)), printStmt(item(vr("C%")))))
		if err != nil {
			t.Errorf("C%%=%v: unexpected error %v", c.value, err)
			continue
		}
		if got := rec.String(); got != c.want {
			t.Errorf("C%%=%v: printed %q, want %q", c.value, got, c.want)
		}
	}
	runPrintCases(t, []printCase{
		{"used in arithmetic", line(let("C%", num(7)), printStmt(item(bin(ast.Div, vr("C%"), num(2))))), " 3.5 \n"},
	})
}

// @spec INTERP-038
func TestIntegerVariableRange(t *testing.T) {
	for _, v := range []float64{32768, 40000, -32768.5, -32769, 1e10} {
		rec := &recorder{}
		in := New(rec)
		in.Exec(line(let("C%", num(1))))
		if err := in.Exec(line(let("C%", num(v)))); !isKind(err, basicerr.IllegalQuantity) {
			t.Errorf("C%%=%v: error = %v, want ILLEGAL QUANTITY", v, err)
		}
		in.Exec(line(printStmt(item(vr("C%")))))
		if got := rec.String(); got != " 1 \n" {
			t.Errorf("C%%=%v: variable changed to %q after the error", v, got)
		}
	}
}

func rel(rel ast.Relation, l, r ast.Expr) *ast.CompareExpr {
	return &ast.CompareExpr{Rel: rel, Left: l, Right: r}
}

// @spec INTERP-039
func TestCompareNumbers(t *testing.T) {
	ne := ast.RelLess | ast.RelGreater
	le := ast.RelLess | ast.RelEqual
	ge := ast.RelGreater | ast.RelEqual
	all := ast.RelLess | ast.RelEqual | ast.RelGreater
	cases := []struct {
		name string
		e    ast.Expr
		want string
	}{
		{"1<2", rel(ast.RelLess, num(1), num(2)), "-1 "},
		{"2<1", rel(ast.RelLess, num(2), num(1)), " 0 "},
		{"1=1", rel(ast.RelEqual, num(1), num(1)), "-1 "},
		{"1=2", rel(ast.RelEqual, num(1), num(2)), " 0 "},
		{"2>1", rel(ast.RelGreater, num(2), num(1)), "-1 "},
		{"1<>2", rel(ne, num(1), num(2)), "-1 "},
		{"1<>1", rel(ne, num(1), num(1)), " 0 "},
		{"1<=1", rel(le, num(1), num(1)), "-1 "},
		{"2<=1", rel(le, num(2), num(1)), " 0 "},
		{"1>=2", rel(ge, num(1), num(2)), " 0 "},
		{"<=> is always true", rel(all, num(5), num(-5)), "-1 "},
		{"negative numbers", rel(ast.RelLess, num(-3), num(-2)), "-1 "},
		{"chained", rel(ast.RelLess, rel(ast.RelLess, num(3), num(2)), num(1)), "-1 "},
	}
	for _, c := range cases {
		rec, err := exec(line(printStmt(item(c.e), semi)))
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
			continue
		}
		if got := rec.String(); got != c.want {
			t.Errorf("%s: printed %q, want %q", c.name, got, c.want)
		}
	}
}

// @spec INTERP-040
func TestCompareStrings(t *testing.T) {
	cases := []struct {
		name string
		e    ast.Expr
		want string
	}{
		{"alphabetical", rel(ast.RelLess, str("APPLE"), str("BANANA")), "-1 "},
		{"equal", rel(ast.RelEqual, str("HI"), str("HI")), "-1 "},
		{"not equal", rel(ast.RelEqual, str("HI"), str("HO")), " 0 "},
		{"prefix is less", rel(ast.RelLess, str("A"), str("AB")), "-1 "},
		{"empty is least", rel(ast.RelLess, str(""), str("A")), "-1 "},
		{"digits before letters", rel(ast.RelLess, str("9"), str("A")), "-1 "},
		{"case matters", rel(ast.RelEqual, str("A"), str("a")), " 0 "},
	}
	for _, c := range cases {
		rec, err := exec(line(printStmt(item(c.e), semi)))
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
			continue
		}
		if got := rec.String(); got != c.want {
			t.Errorf("%s: printed %q, want %q", c.name, got, c.want)
		}
	}
}

// @spec INTERP-041
func TestCompareStringWithNumberIsTypeMismatch(t *testing.T) {
	for name, e := range map[string]ast.Expr{
		"string = number": rel(ast.RelEqual, str("1"), num(1)),
		"number < string": rel(ast.RelLess, num(1), str("A")),
	} {
		if _, err := exec(line(printStmt(item(e)))); !isKind(err, basicerr.TypeMismatch) {
			t.Errorf("%s: error = %v, want TYPE MISMATCH", name, err)
		}
	}
}
